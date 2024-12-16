package lib

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	common "github.com/claes/mqtt-bridges/common"

	"github.com/ConnorsApps/snapcast-go/snapcast"
	"github.com/ConnorsApps/snapcast-go/snapclient"
)

type SnapcastMQTTBridge struct {
	MqttClient       mqtt.Client
	SnapClient       *snapclient.Client
	TopicPrefix      string
	SnapClientConfig SnapClientConfig
	ServerStatus     SnapcastServer

	sendMutex sync.Mutex
}

type SnapClientConfig struct {
	SnapServerAddress string
}

type MQTTClientConfig struct {
	MQTTBroker string
}

func CreateSnapclient(config SnapClientConfig) (*snapclient.Client, error) {
	var client = snapclient.New(&snapclient.Options{
		Host:             config.SnapServerAddress,
		SecureConnection: false,
	})
	return client, nil
}

func NewSnapcastMQTTBridge(snapClientConfig SnapClientConfig, mqttClient mqtt.Client, topicPrefix string) (*SnapcastMQTTBridge, error) {

	snapClient, err := CreateSnapclient(snapClientConfig)
	if err != nil {
		slog.Error("Error creating Snapcast client", "error", err, "address", snapClientConfig.SnapServerAddress)
		return nil, err
	}

	bridge := &SnapcastMQTTBridge{
		MqttClient:       mqttClient,
		SnapClient:       snapClient,
		TopicPrefix:      topicPrefix,
		SnapClientConfig: snapClientConfig,
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"snapcast/group/+/stream/set":  bridge.onGroupStreamSet,
		"snapcast/client/+/stream/set": bridge.onClientStreamSet,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(common.Prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}

	return bridge, nil
}

func (bridge *SnapcastMQTTBridge) onGroupStreamSet(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	re := regexp.MustCompile(`^snapcast/group/([^/]+)/stream/set$`)
	matches := re.FindStringSubmatch(message.Topic())
	if matches != nil {
		groupId := matches[1]

		streamId := string(message.Payload())
		if streamId != "" {

			bridge.PublishMQTT("snapcast/group/"+groupId+"/stream/set", "", false)

			res, err := bridge.SnapClient.Send(context.Background(), snapcast.MethodGroupSetStream,
				&snapcast.GroupSetStreamRequest{ID: groupId, StreamID: streamId})
			if err != nil {
				slog.Error("Error when setting group stream id ", "error", err, "streamid", streamId, "groupid", groupId)
			}
			if res.Error != nil {
				slog.Error("Error in response to group set stream id", "error", res.Error, "streamid", streamId, "groupid", groupId)
			}

			_, err = snapcast.ParseResult[snapcast.GroupSetStreamResponse](res.Result)
			if err != nil {
				slog.Error("Error when parsing group set stream id response", "error", res.Error, "streamid", streamId, "groupid", groupId)
			}
		}
	}
}

func (bridge *SnapcastMQTTBridge) onClientStreamSet(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	re := regexp.MustCompile(`^snapcast/client/([^/]+)/stream/set$`)
	matches := re.FindStringSubmatch(message.Topic())
	if matches != nil {
		streamId := string(message.Payload())
		if streamId != "" {
			clientId := matches[1]
			client, exists := bridge.ServerStatus.Clients[clientId]
			if !exists {
				slog.Error("Client not found", "clientId", clientId)
				return
			}
			groupId := client.GroupID

			bridge.PublishMQTT("snapcast/client/"+clientId+"/stream/set", "", false)

			res, err := bridge.SnapClient.Send(context.Background(), snapcast.MethodGroupSetStream,
				&snapcast.GroupSetStreamRequest{ID: groupId, StreamID: streamId})
			if err != nil {
				slog.Error("Error when setting group stream id ", "error", err, "streamid", streamId, "groupid", groupId)
			}
			if res.Error != nil {
				slog.Error("Error in response to group set stream id", "error", res.Error, "streamid", streamId, "groupid", groupId)
			}

			_, err = snapcast.ParseResult[snapcast.GroupSetStreamResponse](res.Result)
			if err != nil {
				slog.Error("Error when parsing group set stream id response", "error", res.Error, "streamid", streamId, "groupid", groupId)
			}
		}
	}
}

func (bridge *SnapcastMQTTBridge) PublishMQTT(subtopic string, message string, retained bool) {
	token := bridge.MqttClient.Publish(common.Prefixify(bridge.TopicPrefix, subtopic), 0, retained, message)
	token.Wait()
}

func (bridge *SnapcastMQTTBridge) publishServerStatus(serverStatus SnapcastServer, publishGroup, publishClient, publishStream bool) {

	// TODO: Delete what no longer exist?

	if publishGroup {
		for _, group := range serverStatus.Groups {
			bridge.publishGroupStatus(group)
		}
	}
	if publishClient {
		for _, client := range serverStatus.Clients {
			bridge.publishClientStatus(client)
		}
	}
	if publishStream {
		for _, stream := range serverStatus.Streams {
			bridge.publishStreamStatus(stream)
		}
	}
}

func (bridge *SnapcastMQTTBridge) publishStreamStatus(streamStatus SnapcastStream) {
	publish := false
	if bridge.ServerStatus.Clients != nil {
		currentStreamStatus, exists := bridge.ServerStatus.Streams[streamStatus.StreamID]
		publish = !(exists && currentStreamStatus == streamStatus)
	} else {
		publish = true
	}

	if publish {
		jsonData, err := json.MarshalIndent(streamStatus, "", "    ")
		if err != nil {
			slog.Error("Failed to create json for stream", "error", err, "stream", streamStatus)
			return
		}
		bridge.PublishMQTT("snapcast/stream/"+streamStatus.StreamID, string(jsonData), true)
	}
}

func (bridge *SnapcastMQTTBridge) publishClientStatus(clientStatus SnapcastClient) {
	publish := false
	if bridge.ServerStatus.Clients != nil {
		currentClientStatus, exists := bridge.ServerStatus.Clients[clientStatus.ClientID]
		publish = !(exists && currentClientStatus == clientStatus)
	} else {
		publish = true
	}

	if publish {
		jsonData, err := json.MarshalIndent(clientStatus, "", "    ")
		if err != nil {
			slog.Error("Failed to create json for client", "error", err, "client", clientStatus.ClientID)
			return
		}
		bridge.PublishMQTT("snapcast/client/"+clientStatus.ClientID, string(jsonData), true)
	}
}

func (bridge *SnapcastMQTTBridge) publishGroupStatus(groupStatus SnapcastGroup) {
	publish := false
	if bridge.ServerStatus.Groups != nil {
		currentGroupStatus, exists := bridge.ServerStatus.Groups[groupStatus.GroupID]
		publish = !(exists && snapcastGroupsEqual(currentGroupStatus, groupStatus))
	} else {
		publish = true
	}

	if publish {
		jsonData, err := json.MarshalIndent(groupStatus, "", "    ")
		if err != nil {
			slog.Error("Failed to create json for group", "error", err, "group", groupStatus.GroupID)
			return
		}
		bridge.PublishMQTT("snapcast/group/"+groupStatus.GroupID, string(jsonData), true)
	}
}

func (bridge *SnapcastMQTTBridge) processServerStatus(ctx context.Context, publishGroup, publishClient, publishStream bool) {

	res, err := bridge.SnapClient.Send(ctx, snapcast.MethodServerGetStatus, struct{}{})
	if err != nil {
		slog.Error("Error when requesting server status", "error", err)
	}
	if res.Error != nil {
		slog.Error("Error in response to server get status", "error", res.Error)
	}

	serverStatusRes, err := snapcast.ParseResult[snapcast.ServerGetStatusResponse](res.Result)
	if err != nil {
		slog.Error("Error when parsing server status response", "error", res.Error)
	}

	serverStatus, err := parseServerStatus(serverStatusRes)
	if err != nil {
		slog.Error("Error when parsing server status ", "error", err)
	}

	bridge.publishServerStatus(*serverStatus, publishGroup, publishClient, publishStream)
	bridge.ServerStatus = *serverStatus
}

func (bridge *SnapcastMQTTBridge) processGroupStatus(ctx context.Context, groupID string) {

	res, err := bridge.SnapClient.Send(ctx, snapcast.MethodGroupGetStatus, &snapcast.GroupGetStatusRequest{ID: groupID})
	if err != nil {
		slog.Error("Error when requesting group status", "error", err)
	}
	if res.Error != nil {
		slog.Error("Error in response to group get status", "error", res.Error)
	}

	groupStatusRes, err := snapcast.ParseResult[snapcast.GroupGetStatusResponse](res.Result)
	if err != nil {
		slog.Error("Error when parsing group status response", "error", res.Error)
	}

	groupStatus, err := parseGroupStatus(&groupStatusRes.Group)
	if err != nil {
		slog.Error("Error when parsing group status ", "error", err)
	}

	bridge.publishGroupStatus(*groupStatus)
	bridge.ServerStatus.Groups[groupStatus.GroupID] = *groupStatus
}

func (bridge *SnapcastMQTTBridge) EventLoop(ctx context.Context) {

	var notify = &snapclient.Notifications{
		MsgReaderErr:          make(chan error),
		StreamOnUpdate:        make(chan *snapcast.StreamOnUpdate),
		ServerOnUpdate:        make(chan *snapcast.ServerOnUpdate),
		GroupOnMute:           make(chan *snapcast.GroupOnMute),
		GroupOnNameChanged:    make(chan *snapcast.GroupOnNameChanged),
		GroupOnStreamChanged:  make(chan *snapcast.GroupOnStreamChanged),
		ClientOnVolumeChanged: make(chan *snapcast.ClientOnVolumeChanged),
		ClientOnNameChanged:   make(chan *snapcast.ClientOnNameChanged),
		ClientOnConnect:       make(chan *snapcast.ClientOnConnect),
		ClientOnDisconnect:    make(chan *snapcast.ClientOnDisconnect),
	}

	wsClose, err := bridge.SnapClient.Listen(notify)
	if err != nil {
		slog.Error("Error listening for notifications on snapclient", "error", err)
	}
	defer close(wsClose)

	bridge.processServerStatus(ctx, true, true, true)

	for {
		select {

		case <-notify.StreamOnUpdate:
			bridge.processServerStatus(ctx, false, false, true)

		case <-notify.ClientOnConnect:
			bridge.processServerStatus(ctx, false, true, false)
		case <-notify.ClientOnDisconnect:
			bridge.processServerStatus(ctx, false, true, false)
		case <-notify.ClientOnNameChanged:
			bridge.processServerStatus(ctx, false, true, false)
		case <-notify.ClientOnVolumeChanged:
			bridge.processServerStatus(ctx, false, true, false) //todo update value directly?

		case m := <-notify.GroupOnMute:
			bridge.processGroupStatus(ctx, m.ID) //todo update value directly?
		case <-notify.GroupOnNameChanged:
			bridge.processServerStatus(ctx, true, true, false)
		case <-notify.GroupOnStreamChanged:
			bridge.processServerStatus(ctx, true, true, false)

		case <-notify.ServerOnUpdate:
			bridge.processServerStatus(ctx, true, true, true)
		case m := <-notify.MsgReaderErr:
			slog.Debug("Message reader error", "error", m.Error())
			continue
		}
	}
}

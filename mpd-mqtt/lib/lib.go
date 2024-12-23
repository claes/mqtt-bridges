package lib

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	common "github.com/claes/mqtt-bridges/common"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fhs/gompd/v2/mpd"
)

type MpdMQTTBridge struct {
	common.BaseMQTTBridge
	MPDClient       mpd.Client
	PlaylistWatcher mpd.Watcher
	sendMutex       sync.Mutex
}

type MpdClientConfig struct {
	MpdServer, MpdPassword string
}

func CreateMPDClient(config MpdClientConfig) (*mpd.Client, *mpd.Watcher, error) {
	mpdClient, err := mpd.DialAuthenticated("tcp", config.MpdServer, config.MpdPassword)
	if err != nil {
		slog.Error("Could not connect to MPD server", "mpdServer", config.MpdServer, "error", err)
		return mpdClient, nil, err
	} else {
		slog.Info("Connected to MPD server", "mpdServer", config.MpdServer)
	}

	watcher, err := mpd.NewWatcher("tcp", config.MpdServer, config.MpdPassword, "player", "output")
	if err != nil {
		slog.Error("Could not create MPD watcher", "mpdServer", config.MpdServer, "error", err)
		return mpdClient, watcher, err
	} else {
		slog.Info("Created MPD watcher", "mpdServer", config.MpdServer)
	}
	return mpdClient, watcher, nil
}

func NewMpdMQTTBridge(config MpdClientConfig, mqttClient mqtt.Client, topicPrefix string) (*MpdMQTTBridge, error) {

	mpdClient, watcher, err := CreateMPDClient(config)
	if err != nil {
		slog.Error("Could not create MPD client", "error", err)
		return nil, err
	}

	bridge := &MpdMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		MPDClient:       *mpdClient,
		PlaylistWatcher: *watcher,
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"mpd/output/+/set": bridge.onMpdOutputSet,
		"mpd/pause/set":    bridge.onMpdPauseSet,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}
	time.Sleep(2 * time.Second)
	bridge.initialize()

	return bridge, nil
}

func prefixify(topicPrefix, subtopic string) string {
	if len(strings.TrimSpace(topicPrefix)) > 0 {
		return topicPrefix + "/" + subtopic
	} else {
		return subtopic
	}
}

func (bridge *MpdMQTTBridge) onMpdOutputSet(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	re := regexp.MustCompile(`^mpd/output/([^/]+)/set$`)
	matches := re.FindStringSubmatch(message.Topic())
	if matches != nil {
		outputStr := matches[1]
		output, err := strconv.ParseInt(outputStr, 10, 32)
		if err != nil {
			slog.Error("Could not parse output as int", "output", outputStr, "error", err)
			return
		}
		p := string(message.Payload())
		if p != "" {
			enable, err := strconv.ParseBool(p)
			if err != nil {
				slog.Error("Could not parse bool", "payload", p, "error", err)
				return
			}
			bridge.PublishStringMQTT("mpd/output/"+outputStr+"/set", "", false)
			if enable {
				bridge.MPDClient.EnableOutput(int(output))
			} else {
				bridge.MPDClient.DisableOutput(int(output))
			}
		}
	}
}

func (bridge *MpdMQTTBridge) onMpdPauseSet(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	pause, err := strconv.ParseBool(string(message.Payload()))
	if err != nil {
		slog.Error("Could not parse bool", "payload", message.Payload(), "error", err)
		return
	}
	bridge.PublishStringMQTT("mpd/pause/set", "", false)
	bridge.MPDClient.Pause(pause)
}

func (bridge *MpdMQTTBridge) initialize() {
	bridge.publishStatus()
	bridge.publishOutputs()
}

func (bridge *MpdMQTTBridge) publishStatus() {
	status, err := bridge.MPDClient.Status()
	if err != nil {
		slog.Error("Error retrieving MPD status", "error", err)
	} else {
		bridge.PublishJSONMQTT("mpd/status", status, false)
	}
}

func (bridge *MpdMQTTBridge) publishOutputs() {
	outputs, err := bridge.MPDClient.ListOutputs()
	if err != nil {
		slog.Error("Error retrieving MPD outputs", "error", err)
	} else {
		bridge.PublishJSONMQTT("mpd/outputs", outputs, false)
	}
}

func (bridge *MpdMQTTBridge) EventLoop(ctx context.Context) {
	for subsystem := range bridge.PlaylistWatcher.Event {
		slog.Debug("Event received", "subsystem", subsystem)
		if subsystem == "player" {
			bridge.publishStatus()
		} else if subsystem == "output" {
			bridge.publishOutputs()
		}
	}
}

func (bridge *MpdMQTTBridge) DetectReconnectMPDClient(config MpdClientConfig) {
	for {
		time.Sleep(10 * time.Second)
		err := bridge.MPDClient.Ping()
		if err != nil {
			slog.Error("Ping error, reconnecting", "error", err)
			mpdClient, watcher, err := CreateMPDClient(config)
			if err == nil {
				bridge.MPDClient = *mpdClient
				bridge.PlaylistWatcher = *watcher
				slog.Error("Reconnected")
			} else {
				slog.Error("Ping when reconnecting", "error", err)
			}
		}
	}
}

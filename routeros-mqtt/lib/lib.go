package lib

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"time"

	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	routeros "github.com/go-routeros/routeros/v3"
)

type WifiClient struct {
	MacAddress    string `json:"mac_address"`
	Interface     string `json:"interface"`
	Uptime        string `json:"uptime"`
	LastActivity  string `json:"last_activity"`
	SignalToNoise string `json:"signal_to_noise"`
}

type RouterOSMQTTBridge struct {
	common.BaseMQTTBridge
	RouterOSClient       *routeros.Client
	RouterOSClientConfig RouterOSClientConfig
}

type RouterOSClientConfig struct {
	RouterAddress, Username, Password string
}

func CreateRouterOSClient(config RouterOSClientConfig) (*routeros.Client, error) {
	client, err := routeros.DialTLS(config.RouterAddress, config.Username, config.Password, &tls.Config{
		InsecureSkipVerify: true,
	})
	return client, err
}

func NewRouterOSMQTTBridge(routerOSConfig RouterOSClientConfig, mqttClient mqtt.Client, topicPrefix string) (*RouterOSMQTTBridge, error) {

	routerOSClient, err := CreateRouterOSClient(routerOSConfig)
	if err != nil {
		slog.Error("Error creating RouterOS client", "error", err, "address", routerOSConfig.RouterAddress)
		return nil, err
	}

	bridge := &RouterOSMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		RouterOSClient:       routerOSClient,
		RouterOSClientConfig: routerOSConfig,
	}

	routerOSClient.Listen()
	return bridge, nil
}

func (bridge *RouterOSMQTTBridge) EventLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	err := bridge.retrieveRegistrationTable()
	if err != nil {
		bridge.reconnect()
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Closing down RouterOSMQTTBridge event loop")
			return
		case <-ticker.C:
			err := bridge.retrieveRegistrationTable()
			if err != nil {
				bridge.reconnect()
			}
		}
	}
}

func (bridge *RouterOSMQTTBridge) retrieveRegistrationTable() error {
	reply, err := bridge.RouterOSClient.Run("/interface/wireless/registration-table/print")
	if err != nil {
		slog.Error("Could not retrieve registration table", "error", err)
		return err
	} else {
		var clients []WifiClient
		for _, re := range reply.Re {
			client := WifiClient{
				MacAddress:    re.Map["mac-address"],
				Interface:     re.Map["interface"],
				Uptime:        re.Map["uptime"],
				LastActivity:  re.Map["last-activity"],
				SignalToNoise: re.Map["signal-to-noise"],
			}
			clients = append(clients, client)
		}
		jsonData, err := json.MarshalIndent(clients, "", "    ")
		if err != nil {
			slog.Error("Failed to create json", "error", err)
			return err
		}
		bridge.PublishMQTT("routeros/wificlients", string(jsonData), false)
		bridge.MQTTClient.IsConnected() //Why? Forgot..
	}
	return nil
}

func (bridge *RouterOSMQTTBridge) reconnect() {
	slog.Error("Reconnecting RouterOS client")
	err := bridge.RouterOSClient.Close()
	if err != nil {
		slog.Error("Error when closing RouterOS client", "error", err)
		return
	}
	client, err := CreateRouterOSClient(bridge.RouterOSClientConfig)
	if err != nil {
		slog.Error("Error when recreating RouterOS client", "error", err)
		return
	}
	bridge.RouterOSClient = client
}

package lib

import (
	"log/slog"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateMQTTClient(mqttBroker string) (mqtt.Client, error) {
	slog.Info("Creating MQTT client", "broker", mqttBroker)
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker).SetAutoReconnect(true)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Could not connect to broker", "mqttBroker", mqttBroker, "error", token.Error())
		return nil, token.Error()
	}
	slog.Info("Connected to MQTT broker", "mqttBroker", mqttBroker)

	return client, nil
}

func Prefixify(topicPrefix, subtopic string) string {
	if len(strings.TrimSpace(topicPrefix)) > 0 {
		return topicPrefix + "/" + subtopic
	} else {
		return subtopic
	}
}

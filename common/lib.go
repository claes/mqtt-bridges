package lib

import (
	"encoding/json"
	"log/slog"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BaseMQTTBridge struct {
	MQTTClient  mqtt.Client
	TopicPrefix string
}

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

func (bridge *BaseMQTTBridge) PublishStringMQTT(subtopic string, message string, retained bool) {
	token := bridge.MQTTClient.Publish(Prefixify(bridge.TopicPrefix, subtopic), 0, retained, message)
	token.Wait()
}

func (bridge *BaseMQTTBridge) PublishBytesMQTT(subtopic string, message []byte, retained bool) {
	token := bridge.MQTTClient.Publish(Prefixify(bridge.TopicPrefix, subtopic), 0, retained, message)
	token.Wait()
}

func (bridge *BaseMQTTBridge) PublishJSONMQTT(subtopic string, obj any, retained bool) {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		slog.Error("Error marshalling object to publish", "error", err, "obj", obj)
		return
	}
	token := bridge.MQTTClient.Publish(Prefixify(bridge.TopicPrefix, subtopic), 0, retained, string(jsonData))
	token.Wait()
}

func Prefixify(topicPrefix, subtopic string) string {
	if len(strings.TrimSpace(topicPrefix)) > 0 {
		return topicPrefix + "/" + subtopic
	} else {
		return subtopic
	}
}

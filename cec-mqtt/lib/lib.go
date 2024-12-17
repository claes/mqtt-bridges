package lib

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"sync"

	common "github.com/claes/mqtt-bridges/common"

	cec "github.com/claes/cec"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var debug *bool

type CECMQTTBridge struct {
	MQTTClient    mqtt.Client
	CECConnection *cec.Connection
	TopicPrefix   string
}

type CECClientConfig struct {
	CECName, CECDeviceName string
}

func CreateCECConnection(config CECClientConfig) (*cec.Connection, error) {
	slog.Info("Initializing CEC connection", "cecName", config.CECName, "cecDeviceName", config.CECDeviceName)

	cecConnection, err := cec.Open(config.CECName, config.CECDeviceName)
	if err != nil {
		slog.Error("Could not connect to CEC device",
			"cecName", config.CECName, "cecDeviceName", config.CECDeviceName, "error", err)
		return nil, err
	}

	slog.Info("CEC connection opened")
	return cecConnection, nil
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

func NewCECMQTTBridge(config CECClientConfig, mqttClient mqtt.Client, topicPrefix string) (*CECMQTTBridge, error) {

	cecConnection, err := CreateCECConnection(config)
	if err != nil {
		slog.Error("Could not create CEC connection", "error", err)
		return nil, err
	}

	slog.Info("Creating CEC MQTT bridge")
	bridge := &CECMQTTBridge{
		MQTTClient:    mqttClient,
		CECConnection: cecConnection,
		TopicPrefix:   topicPrefix,
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"cec/key/send":   bridge.onKeySend,
		"cec/command/tx": bridge.onCommandSend,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(common.Prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}

	bridge.initialize()
	slog.Info("CEC MQTT bridge initialized")
	return bridge, nil
}

func (bridge *CECMQTTBridge) initialize() {
	cecDevices := bridge.CECConnection.List()
	for key, value := range cecDevices {
		slog.Info("Connected device",
			"key", key,
			"activeSource", value.ActiveSource,
			"logicalAddress", value.LogicalAddress,
			"osdName", value.OSDName,
			"physicalAddress", value.PhysicalAddress,
			"powerStatus", value.PowerStatus,
			"vendor", value.Vendor)
		bridge.PublishMQTT("cec/source/"+strconv.Itoa(value.LogicalAddress)+"/active",
			strconv.FormatBool(value.ActiveSource), true)
		bridge.PublishMQTT("cec/source/"+strconv.Itoa(value.LogicalAddress)+"/name",
			value.OSDName, true)
		bridge.PublishMQTT("cec/source/"+strconv.Itoa(value.LogicalAddress)+"/power",
			value.PowerStatus, true)
	}
}

func (bridge *CECMQTTBridge) PublishMQTT(subtopic string, message string, retained bool) {
	token := bridge.MQTTClient.Publish(common.Prefixify(bridge.TopicPrefix, subtopic), 0, retained, message)
	token.Wait()
}

func (bridge *CECMQTTBridge) PublishCommands(ctx context.Context) {
	bridge.CECConnection.Commands = make(chan *cec.Command, 10) // Buffered channel
	for {
		select {
		case <-ctx.Done():
			slog.Info("PublishCommands function is being cancelled")
			return
		case command := <-bridge.CECConnection.Commands:
			slog.Debug("Create command", "command", command.CommandString)
			bridge.PublishMQTT("cec/command/rx", command.CommandString, false)
		}
	}
}

func (bridge *CECMQTTBridge) PublishKeyPresses(ctx context.Context) {
	bridge.CECConnection.KeyPresses = make(chan *cec.KeyPress, 10) // Buffered channel

	for {
		select {
		case <-ctx.Done():
			slog.Info("PublishKeyPresses function is being cancelled")
			return
		case keyPress := <-bridge.CECConnection.KeyPresses:
			slog.Debug("Key press", "keyCode", keyPress.KeyCode, "duration", keyPress.Duration)
			if keyPress.Duration == 0 {
				bridge.PublishMQTT("cec/key", strconv.Itoa(keyPress.KeyCode), false)
			}
		}
	}
}

func (bridge *CECMQTTBridge) PublishSourceActivations(ctx context.Context) {
	bridge.CECConnection.SourceActivations = make(chan *cec.SourceActivation, 10) // Buffered channel

	for {
		select {
		case <-ctx.Done():
			slog.Info("PublishCommands function is being cancelled")
			return
		case sourceActivation := <-bridge.CECConnection.SourceActivations:
			slog.Debug("Source activation",
				"logicalAddress", sourceActivation.LogicalAddress,
				"state", sourceActivation.State)
			bridge.PublishMQTT("cec/source/"+strconv.Itoa(sourceActivation.LogicalAddress)+"/active",
				strconv.FormatBool(sourceActivation.State), true)
		}
	}
}

func (bridge *CECMQTTBridge) PublishMessages(ctx context.Context, logOnly bool) {

	pattern := `^(>>|<<)\s([0-9A-Fa-f]{2}(?::[0-9A-Fa-f]{2})*)`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		slog.Info("Error compiling regex", "error", err)
		return
	}

	bridge.CECConnection.Messages = make(chan string, 10) // Buffered channel
	for {
		select {
		case <-ctx.Done():
			slog.Info("PublishMessages function is being cancelled")
			return
		case message := <-bridge.CECConnection.Messages:
			slog.Debug("Message", "message", message)
			if !logOnly {
				bridge.PublishMQTT("cec/message", message, false)
			}
			matches := regex.FindStringSubmatch(message)
			if matches != nil {
				prefix := matches[1]
				hexPart := matches[2]
				slog.Debug("CEC Message payload match", "prefix", prefix, "hex", hexPart)
				if prefix == "<<" {
					bridge.PublishMQTT("cec/message/hex/rx", hexPart, true)
				} else if prefix == ">>" {
					bridge.PublishMQTT("cec/message/hex/tx", hexPart, true)
				}
			}
		}
	}
}

var sendMutex sync.Mutex

func (bridge *CECMQTTBridge) onCommandSend(client mqtt.Client, message mqtt.Message) {
	sendMutex.Lock()
	defer sendMutex.Unlock()

	if "" == string(message.Payload()) {
		return
	}
	command := string(message.Payload())
	if command != "" {
		bridge.PublishMQTT("cec/command/tx", "", false)
		slog.Debug("Sending command", "command", command)
		bridge.CECConnection.Transmit(command)
	}
}

func (bridge *CECMQTTBridge) onKeySend(client mqtt.Client, message mqtt.Message) {
	sendMutex.Lock()
	defer sendMutex.Unlock()

	if "" == string(message.Payload()) {
		return
	}
	var payload map[string]interface{}
	err := json.Unmarshal(message.Payload(), &payload)
	if err != nil {
		slog.Error("Could not parse payload", "payload", string(message.Payload()))
	}
	address := payload["address"].(float64)
	key := payload["key"].(string)
	if key != "" {
		bridge.PublishMQTT("cec/key/send", "", false)
		slog.Debug("Sending key", "address", address, "key", key)
		bridge.CECConnection.Key(int(address), key)
	}
}

// Create conditions to ping cec connection
// and to reconnect

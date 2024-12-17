package lib

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/tarm/serial"
)

type RotelState struct {
	Balance  string `json:"balance"`
	Bass     string `json:"bass"`
	Display  string `json:"display"`
	Display1 string `json:"display1"`
	Display2 string `json:"display2"`
	Freq     string `json:"freq"`
	Mute     string `json:"mute"`
	Source   string `json:"source"`
	State    string `json:"state"`
	Tone     string `json:"tone"`
	Treble   string `json:"treble"`
	Volume   string `json:"volume"`
}

type RotelMQTTBridge struct {
	common.BaseMQTTBridge
	SerialPort       *serial.Port
	RotelDataParser  RotelDataParser
	State            *RotelState
	sendMutex        sync.Mutex
	serialWriteMutex sync.Mutex
}

type RotelClientConfig struct {
	SerialDevice string
}

func CreateSerialPort(config RotelClientConfig) (*serial.Port, error) {
	serialConfig := &serial.Config{Name: config.SerialDevice, Baud: 115200}
	serialPort, err := serial.OpenPort(serialConfig)
	if err != nil {
		slog.Error("Could not open port", "serialDevice", config.SerialDevice, "error", err)
		return nil, err
	} else {
		slog.Info("Connected to serial device", "serialDevice", config.SerialDevice)
	}
	return serialPort, nil
}

func NewRotelMQTTBridge(config RotelClientConfig, mqttClient mqtt.Client, topicPrefix string) (*RotelMQTTBridge, error) {

	serialPort, err := CreateSerialPort(config)
	if err != nil {
		slog.Error("Could not open serial port ", "config", config, "error", err)
		return nil, err
	}

	bridge := &RotelMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		SerialPort:      serialPort,
		RotelDataParser: *NewRotelDataParser(),
		State:           &RotelState{},
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"rotel/command/send":       bridge.onCommandSend,
		"rotel/command/initialize": bridge.onInitialize,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(common.Prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}
	time.Sleep(2 * time.Second)
	bridge.initialize(true)
	return bridge, nil
}

func (bridge *RotelMQTTBridge) initialize(askPower bool) {
	if askPower {
		// to avoid recursion when initializing after power on
		bridge.SendSerialRequest("get_current_power!")
	}
	bridge.SendSerialRequest("display_update_auto!")
	bridge.SendSerialRequest("get_display!")
	bridge.SendSerialRequest("get_display1!")
	bridge.SendSerialRequest("get_display2!")
	bridge.SendSerialRequest("get_volume!")
	bridge.SendSerialRequest("get_current_source!")
	bridge.SendSerialRequest("get_current_freq!")
	bridge.SendSerialRequest("get_tone!")
	bridge.SendSerialRequest("get_bass!")
	bridge.SendSerialRequest("get_treble!")
	bridge.SendSerialRequest("get_balance!")
}

func (bridge *RotelMQTTBridge) onCommandSend(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	// Sends command to the Rotel without intermediate parsing
	// Rotel commands are documented here:
	// https://www.rotel.com/sites/default/files/product/rs232/RA12%20Protocol.pdf
	command := string(message.Payload())
	if command != "" {
		bridge.PublishMQTT("rotel/command/send", "", false)
		bridge.SendSerialRequest(command)
	}
}

func (bridge *RotelMQTTBridge) onInitialize(client mqtt.Client, message mqtt.Message) {
	command := string(message.Payload())
	if command != "" {
		bridge.PublishMQTT("rotel/command/initialize", "", false)
		bridge.initialize(true)
	}
}

func (bridge *RotelMQTTBridge) EventLoop(ctx context.Context) {
	defer bridge.SerialPort.Close()

	buf := make([]byte, 128)
	for {

		select {
		case <-ctx.Done():
			slog.Info("Closing down RotelMQTTBridge event loop")
			return
		default:

			n, err := bridge.SerialPort.Read(buf)
			if err != nil {
				slog.Error("Error reading bytes from serial port", "buf", buf)
				continue
			}
			bridge.ProcessRotelData(string(buf[:n]))

			jsonState, err := json.Marshal(bridge.State)
			if err != nil {
				slog.Error("Error marshalling state", "state", bridge.State)
				continue
			}
			bridge.PublishMQTT("rotel/state", string(jsonState), true)
		}
	}
}

func (bridge *RotelMQTTBridge) SendSerialRequest(message string) {
	bridge.serialWriteMutex.Lock()
	defer bridge.serialWriteMutex.Unlock()

	_, err := bridge.SerialPort.Write([]byte(message))
	if err != nil {
		slog.Error("Error writing message", "message", message)
	}
}

func (bridge *RotelMQTTBridge) ProcessRotelData(data string) {

	bridge.RotelDataParser.HandleParsedData(data)

	for cmd := bridge.RotelDataParser.GetNextRotelData(); cmd != nil; cmd = bridge.RotelDataParser.GetNextRotelData() {
		slog.Debug("Processed Rotel data", "data", data, "command", cmd)

		switch action := cmd[0]; action {
		case "volume":
			bridge.State.Volume = cmd[1]
		case "source":
			bridge.State.Source = cmd[1]
		case "freq":
			bridge.State.Freq = cmd[1]
		case "display":
			bridge.State.Display = cmd[1]
		case "display1":
			bridge.State.Display1 = cmd[1]
		case "display2":
			bridge.State.Display2 = cmd[1]
		case "treble":
			bridge.State.Treble = cmd[1]
		case "bass":
			bridge.State.Bass = cmd[1]
		case "tone":
			bridge.State.Tone = cmd[1]
		case "balance":
			bridge.State.Balance = cmd[1]
		case "mute":
			if cmd[1] == "on/off" {
				bridge.SendSerialRequest("get_volume!")
			} else {
				bridge.State.Mute = cmd[1]
			}
		case "power":
			if cmd[1] == "on" {
				bridge.State.State = cmd[1]
				bridge.initialize(false)
			} else if cmd[1] == "standby" {
				bridge.State.State = cmd[1]
				bridge.initialize(false)
			}
		case "power_off":
			bridge.State.State = "standby"
			bridge.State.Volume = ""
			bridge.State.Source = ""
			bridge.State.Freq = ""
			bridge.State.Display = ""
			bridge.State.Display1 = ""
			bridge.State.Display2 = ""
			bridge.State.Treble = ""
			bridge.State.Bass = ""
			bridge.State.Tone = ""
			bridge.State.Balance = ""
		}
	}
	slog.Debug("Rotel data processed", "state", bridge.State)
}

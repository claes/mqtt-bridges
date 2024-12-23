package lib

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/godbus/dbus/v5"
)

type BluezMediaPlayerConfig struct {
	BluetoothMACAddress string
}

type BluezMediaPlayerMQTTBridge struct {
	common.BaseMQTTBridge
	BluezMediaControl      dbus.BusObject
	BluezMediaPlayer       dbus.BusObject
	BluezMediaPlayerConfig BluezMediaPlayerConfig
	DevicePath             string
	sendMutex              sync.Mutex
}

func CreateBluezMediaPlayer(config BluezMediaPlayerConfig) (dbus.BusObject, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		slog.Error("Error connecting to system dbus", "error", err)
		return nil, err
	}
	devicePath, err := bluetoothMACToBluezDevicePath(config.BluetoothMACAddress)
	if err != nil {
		slog.Error("Error converting Bluetooth MAC address to device path", "error", err, "mac", config.BluetoothMACAddress)
		return nil, err
	}
	obj := conn.Object("org.bluez", dbus.ObjectPath(devicePath+"/player0"))
	return obj, nil
}

func CreateBluezMediaControl(config BluezMediaPlayerConfig) (dbus.BusObject, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		slog.Error("Error connecting to system dbus", "error", err)
		return nil, err
	}
	devicePath, err := bluetoothMACToBluezDevicePath(config.BluetoothMACAddress)
	if err != nil {
		slog.Error("Error converting Bluetooth MAC address to device path", "error", err, "mac", config.BluetoothMACAddress)
		return nil, err
	}
	obj := conn.Object("org.bluez", dbus.ObjectPath(devicePath))
	return obj, nil
}

func NewBluezMediaPlayerMQTTBridge(config BluezMediaPlayerConfig, mqttClient mqtt.Client, topicPrefix string) (*BluezMediaPlayerMQTTBridge, error) {

	bluezMediaPlayer, err := CreateBluezMediaPlayer(config)
	if err != nil {
		slog.Error("Error creating Bluez MediaPlayer client", "error", err)
		return nil, err
	}

	bluezMediaControl, err := CreateBluezMediaControl(config)
	if err != nil {
		slog.Error("Error creating Bluez MediaPlayer client", "error", err)
		return nil, err
	}

	devicePath, err := bluetoothMACToBluezDevicePath(config.BluetoothMACAddress)
	if err != nil {
		slog.Error("Error converting Bluetooth MAC address to device path", "error", err, "mac", config.BluetoothMACAddress)
		return nil, err
	}

	bridge := &BluezMediaPlayerMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		BluezMediaPlayerConfig: config,
		DevicePath:             devicePath,
		BluezMediaPlayer:       bluezMediaPlayer,
		BluezMediaControl:      bluezMediaControl,
	}
	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"bluez/" + bridge.BluezMediaPlayerConfig.BluetoothMACAddress + "/mediacontrol/command/send": bridge.onMediaControlCommandSend,
		"bluez/" + bridge.BluezMediaPlayerConfig.BluetoothMACAddress + "/mediaplayer/command/send":  bridge.onMediaPlayerCommandSend,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(common.Prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}

	return bridge, nil
}

func (bridge *BluezMediaPlayerMQTTBridge) onMediaControlCommandSend(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	command := string(message.Payload())
	if command != "" {
		bridge.PublishStringMQTT("bluez/"+bridge.BluezMediaPlayerConfig.BluetoothMACAddress+"/mediacontrol/command/send", "", false)
		method := fmt.Sprintf("org.bluez.MediaControl1.%s", command)
		call := bridge.BluezMediaControl.Call(method, 0)
		if call.Err != nil {
			slog.Error("Error sending bluez MediaControl command", "error", call.Err, "devicePath", bridge.DevicePath, "method", method)
		}
	}
}

func (bridge *BluezMediaPlayerMQTTBridge) onMediaPlayerCommandSend(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	command := string(message.Payload())
	if command != "" {
		bridge.PublishStringMQTT("bluez/"+bridge.BluezMediaPlayerConfig.BluetoothMACAddress+"/mediaplayer/command/send", "", false)
		method := fmt.Sprintf("org.bluez.MediaPlayer1.%s", command)
		call := bridge.BluezMediaPlayer.Call(method, 0)
		if call.Err != nil {
			slog.Error("Error sending bluez MediaPlayer command", "error", call.Err, "devicePath", bridge.DevicePath, "method", method)
		}
	}
}

func (bridge *BluezMediaPlayerMQTTBridge) EventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

// Converts a Bluetooth MAC address
// (e.g., "C0:4B:24:D6:38:C9") to a BlueZ DBus device path
// (e.g., "/org/bluez/hci0/dev_C0_4B_24_D6_38_C9").
func bluetoothMACToBluezDevicePath(macAddress string) (string, error) {
	if !isValidBluetoothAddress(macAddress) {
		return "", fmt.Errorf("Invalid Bluetooth MAC address: %s", macAddress)
	}
	sanitizedAddress := strings.ReplaceAll(macAddress, ":", "_")
	devicePath := fmt.Sprintf("/org/bluez/hci0/dev_%s", sanitizedAddress)
	return devicePath, nil
}

func isValidBluetoothAddress(macAddress string) bool {
	if len(macAddress) != 17 {
		return false
	}

	parts := strings.Split(macAddress, ":")
	if len(parts) != 6 {
		return false
	}

	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
	}

	return true
}

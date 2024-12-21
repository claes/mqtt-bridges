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
	obj := conn.Object("org.bluez", dbus.ObjectPath(devicePath))
	return obj, nil
}

func NewBluezMediaPlayerMQTTBridge(config BluezMediaPlayerConfig, mqttClient mqtt.Client, topicPrefix string) (*BluezMediaPlayerMQTTBridge, error) {

	bluezMediaPlayer, err := CreateBluezMediaPlayer(config)
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
	}
	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"bluez/" + bridge.BluezMediaPlayerConfig.BluetoothMACAddress + "/mediacontrol/command/send": bridge.onMediaControlCommandSend,
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

	//mediacontrol seems deprecated but it is what I have
	command := string(message.Payload())
	if command != "" {
		bridge.PublishMQTT("bluez/"+bridge.BluezMediaPlayerConfig.BluetoothMACAddress+"/mediacontrol/command/send", "", false)
		method := fmt.Sprintf("org.bluez.MediaControl1.%s", command)
		call := bridge.BluezMediaPlayer.Call(method, 0)
		if call.Err != nil {
			slog.Error("Error sending bluez mediaplayer command", "error", call.Err, "devicePath", bridge.DevicePath, "method", method)
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
	// conn, err := dbus.SessionBus()

	// if err != nil {
	// 	slog.Error("Failed to connect to dbus system bus", "error", err)
	// 	return
	// }

	// // // Match PropertiesChanged signals
	// // // Specifically for BlueZ MediaPlayer1 interface
	// // call := bridge.BluezMediaPlayer.Call(
	// // 	"org.freedesktop.DBus.AddMatch", 0,
	// // 	"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'",
	// // 	//"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='"+bridge.DevicePath+"/player0'",
	// // )

	// // if call.Err != nil {
	// // 	log.Fatalf("Failed to add match: %v", call.Err)
	// // }

	// signalChan := make(chan *dbus.Signal, 10)
	// conn.Signal(signalChan)

	// for signal := range signalChan {
	// 	fmt.Println("hej")
	// 	if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
	// 		fmt.Println("PropertiesChanged signal received:", signal.Body)
	// 	}

	// 	if len(signal.Body) < 3 {
	// 		continue
	// 	}
	// 	interfaceName := signal.Body[0].(string)
	// 	changedProperties := signal.Body[1].(map[string]dbus.Variant)

	// 	if interfaceName == "org.bluez.MediaPlayer1" {
	// 		slog.Info("Signal received for interface", "interface", interfaceName)
	// 		for key, value := range changedProperties {
	// 			slog.Info("Property changed", "property", key, "value", value.Value())
	// 		}
	// 	}
	// }
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

package lib

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/karalabe/hid"

	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type HIDMQTTBridge struct {
	common.BaseMQTTBridge
	HIDDevice hid.Device
	HIDConfig HIDBridgeConfig
	sendMutex sync.Mutex
}

type MQTTClientConfig struct {
	MQTTBroker string
}

type HIDBridgeConfig struct {
	VendorID, ProductID                          uint16
	PublishBytes, PublishNative, PublishReadable bool
}

func CreateHIDClient(hidConfig HIDBridgeConfig) (hid.Device, error) {
	deviceInfos, err := hid.Enumerate(hidConfig.VendorID, hidConfig.ProductID)
	if err != nil {
		slog.Error("Could not enumerate HID devices", "hidConfig", hidConfig)
		return nil, err
	}
	if len(deviceInfos) == 0 {
		slog.Error("No HID devices found", "hidConfig", hidConfig)
		return nil, errors.New("No HID devices found")
	}
	for i, deviceInfo := range deviceInfos {
		slog.Info("HID Details",
			slog.Int("HID Number", i),
			slog.String("OS Path", deviceInfo.Path),
			slog.String("Vendor ID", fmt.Sprintf("%#04x", deviceInfo.VendorID)),
			slog.String("Product ID", fmt.Sprintf("%#04x", deviceInfo.ProductID)),
			slog.String("Serial", deviceInfo.Serial),
			slog.String("Manufacturer", deviceInfo.Manufacturer),
			slog.String("Product", deviceInfo.Product),
			slog.Int("Interface", deviceInfo.Interface),
		)
	}
	var device hid.Device
	for i, deviceInfo := range deviceInfos {
		device, err = deviceInfo.Open()
		if err != nil {
			slog.Error("Could not open hid device", "error", err, "hidConfig", hidConfig, "device", device, "HID Number", i)
		} else {
			slog.Info("Opened device", "device", device, "HID Number", i)
			if device != nil {
				break
			}
		}
	}
	if device == nil {
		slog.Error("No hid device could be opened")
		return nil, errors.New("No hid device could be opened")
	} else {
		return device, nil
	}
}

func NewHIDMQTTBridge(hidConfig HIDBridgeConfig, mqttClient mqtt.Client, topicPrefix string) (*HIDMQTTBridge, error) {

	hidDevice, err := CreateHIDClient(hidConfig)
	if err != nil {
		return nil, err
	}

	bridge := &HIDMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		HIDDevice: hidDevice,
		HIDConfig: hidConfig,
	}

	// funcs := map[string]func(client mqtt.Client, message mqtt.Message){
	// 	"snapcast/group/+/stream/set":  bridge.onGroupStreamSet,
	// 	"snapcast/client/+/stream/set": bridge.onClientStreamSet,
	// }
	// for key, function := range funcs {
	// 	token := mqttClient.Subscribe(prefixify(topicPrefix, key), 0, function)
	// 	token.Wait()
	// }

	return bridge, nil
}

func (bridge *HIDMQTTBridge) EventLoop(ctx context.Context) {

	buf := make([]byte, 64)
	for {
		n, err := bridge.HIDDevice.Read(buf)
		if err != nil {
			slog.Error("Error reading from HID device", "error", err)
			continue
		}

		bytes := buf[:n]

		if bridge.HIDConfig.PublishBytes {
			bridge.PublishBytesMQTT("hid/device/bytes", bytes, false)
		}
		if bridge.HIDConfig.PublishNative {
			nativeReport, _ := CreateNativeHIDReport(bytes)
			bridge.PublishJSONMQTT("hid/device/native", nativeReport, false)
		}
		if bridge.HIDConfig.PublishReadable {
			readableReport, _ := CreateReadableHIDReport(bytes)
			bridge.PublishJSONMQTT("hid/device/readable", readableReport, false)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

package lib

import (
	"context"
	"encoding/json"
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
	sendMutex sync.Mutex
}

type MQTTClientConfig struct {
	MQTTBroker string
}

type HIDConfig struct {
	VendorID, ProductID uint16
}

func CreateHIDClient(hidConfig HIDConfig) (hid.Device, error) {
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
	device, err := deviceInfos[0].Open()
	if err != nil {
		slog.Error("Could not open hid device", "error", err, "hidConfig", hidConfig, "device", device)
		return nil, err
	}
	slog.Info("Opened device", "device", device)
	return device, nil
}

func NewHIDMQTTBridge(hidConfig HIDConfig, mqttClient mqtt.Client, topicPrefix string) (*HIDMQTTBridge, error) {

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
		slog.Info("Waiting for read")
		n, err := bridge.HIDDevice.Read(buf)
		if err != nil {
			slog.Error("Error reading from HID device", "error", err)
			continue
		}
		slog.Info("Read data")

		data := buf[:n]
		report, err := ParseHIDReport(data)
		// if err != nil {
		// 	slog.Error("Error parsing HID report", "error", err)
		// 	continue
		// }
		// json, err := report.ToJSON()
		json, err := HIDReportToJSON(data)
		if err != nil {
			slog.Error("Error generating JSON for HID report", "error", err, "hidReport", report)
		}
		slog.Info("HID report", "hidReport", json)
		bridge.PublishMQTT("hid/device/data", string(data), false)
		time.Sleep(100 * time.Millisecond)
	}

}

// Modifier represents modifier keys (Shift, Ctrl, etc.)
type Modifier uint8

// Enum values for Modifier keys
const (
	LeftCtrl Modifier = 1 << iota
	LeftShift
	LeftAlt
	LeftGUI
	RightCtrl
	RightShift
	RightAlt
	RightGUI
)

// Keycode represents standard keyboard keys
type Keycode uint8

// Enum values for Keycodes (based on HID Usage Table 0x07)
const (
	KeyA     Keycode = 0x04
	KeyB     Keycode = 0x05
	KeyC     Keycode = 0x06
	KeyD     Keycode = 0x07
	KeyE     Keycode = 0x08
	KeyF     Keycode = 0x09
	KeyG     Keycode = 0x0A
	KeyH     Keycode = 0x0B
	KeyI     Keycode = 0x0C
	KeyJ     Keycode = 0x0D
	KeyK     Keycode = 0x0E
	KeyL     Keycode = 0x0F
	KeyM     Keycode = 0x10
	KeyN     Keycode = 0x11
	KeyO     Keycode = 0x12
	KeyP     Keycode = 0x13
	KeyQ     Keycode = 0x14
	KeyR     Keycode = 0x15
	KeyS     Keycode = 0x16
	KeyT     Keycode = 0x17
	KeyU     Keycode = 0x18
	KeyV     Keycode = 0x19
	KeyW     Keycode = 0x1A
	KeyX     Keycode = 0x1B
	KeyY     Keycode = 0x1C
	KeyZ     Keycode = 0x1D
	Key1     Keycode = 0x1E
	Key2     Keycode = 0x1F
	KeyEnter Keycode = 0x28
	KeySpace Keycode = 0x2C
)

// HIDReport represents a keyboard HID input report
type HIDReport struct {
	Modifiers Modifier   // Modifier byte
	Keys      [6]Keycode // Up to 6 simultaneously pressed keys
}

func (h *HIDReport) ToJSON() (string, error) {
	// Convert the struct to JSON
	jsonData, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("failed to marshal HIDReport to JSON: %w", err)
	}

	// Return as a string
	return string(jsonData), nil
}

// ParseHIDReport parses the raw HID report byte array into an HIDReport struct
func ParseHIDReport(data []byte) (*HIDReport, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid report length: expected 8 bytes, got %d", len(data))
	}

	report := &HIDReport{
		Modifiers: Modifier(data[0]), // Modifier byte
		Keys: [6]Keycode{
			Keycode(data[2]),
			Keycode(data[3]),
			Keycode(data[4]),
			Keycode(data[5]),
			Keycode(data[6]),
			Keycode(data[7]),
		},
	}

	return report, nil
}

// ModifierToString converts Modifier enums to a readable list of names
func ModifierToString(modifier Modifier) []string {
	var keys []string
	if modifier&LeftCtrl != 0 {
		keys = append(keys, "LeftCtrl")
	}
	if modifier&LeftShift != 0 {
		keys = append(keys, "LeftShift")
	}
	if modifier&LeftAlt != 0 {
		keys = append(keys, "LeftAlt")
	}
	if modifier&LeftGUI != 0 {
		keys = append(keys, "LeftGUI")
	}
	if modifier&RightCtrl != 0 {
		keys = append(keys, "RightCtrl")
	}
	if modifier&RightShift != 0 {
		keys = append(keys, "RightShift")
	}
	if modifier&RightAlt != 0 {
		keys = append(keys, "RightAlt")
	}
	if modifier&RightGUI != 0 {
		keys = append(keys, "RightGUI")
	}
	return keys
}

// KeycodesToString converts Keycode enums to readable names
func KeycodesToString(keys [6]Keycode) []string {
	var pressedKeys []string
	for _, key := range keys {
		switch key {
		case KeyA:
			pressedKeys = append(pressedKeys, "A")
		case KeyB:
			pressedKeys = append(pressedKeys, "B")
		case KeyC:
			pressedKeys = append(pressedKeys, "C")
		case KeySpace:
			pressedKeys = append(pressedKeys, "Space")
		case KeyEnter:
			pressedKeys = append(pressedKeys, "Enter")
		default:
			if key != 0 {
				pressedKeys = append(pressedKeys, fmt.Sprintf("Unknown(0x%x)", key))
			}
		}
	}
	return pressedKeys
}

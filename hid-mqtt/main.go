package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	common "github.com/claes/mqtt-bridges/common"

	hidmqtt "github.com/claes/mqtt-bridges/hid-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: hid-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	topicPrefix := flag.String("topicPrefix", "", "MQTT topic prefix to use")
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	vendorIDStr := flag.String("vendorId", "", "Vendor ID")
	productIDStr := flag.String("productId", "", "Product ID")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	vendorID, err := strconv.ParseUint(*vendorIDStr, 16, 16)
	if err != nil {
		fmt.Printf("Error converting Vendor ID: %v\n", err)
		return
	}

	productID, err := strconv.ParseUint(*productIDStr, 16, 16)
	if err != nil {
		fmt.Printf("Error converting Product ID: %v\n", err)
		return
	}

	hidConfig := hidmqtt.HIDBridgeConfig{
		VendorID:        uint16(vendorID),
		ProductID:       uint16(productID),
		PublishBytes:    false,
		PublishNative:   true,
		PublishReadable: true,
	}

	mqttClient, err := common.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating MQTT client", "error", err)
		os.Exit(1)
	}

	bridge, err := hidmqtt.NewHIDMQTTBridge(hidConfig, mqttClient, *topicPrefix)

	if err != nil {
		slog.Error("Error creating HID-MQTT bridge", "error", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")

	ctx := context.TODO()
	go bridge.EventLoop(ctx)
	<-c
	bridge.HIDDevice.Close()
	fmt.Printf("Shut down\n")

	os.Exit(0)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	common "github.com/claes/mqtt-bridges/common"

	lib "github.com/claes/mqtt-bridges/cec-mqtt/lib"
)

var debug *bool

func MainLoop(ctx context.Context, bridge lib.CECMQTTBridge) {
	for {
		time.Sleep(10 * time.Second)
		bridge.CECConnection.Transmit("10:8F")
		time.Sleep(10 * time.Second)
	}
}

func printHelp() {
	fmt.Println("Usage: cec-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	cecName := flag.String("cecName", "/dev/ttyACM0", "CEC name")
	cecDeviceName := flag.String("cecDeviceName", "CEC-MQTT", "CEC device name")
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	topicPrefix := flag.String("topicPrefix", "", "MQTT topic prefix")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *debug {
		var programLevel = new(slog.LevelVar)
		programLevel.Set(slog.LevelDebug)
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
		slog.SetDefault(slog.New(handler))
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	mqttClient, err := common.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating mqtt client", "error", err, "broker", *mqttBroker)
		os.Exit(1)
	}

	cecClientConfig := lib.CECClientConfig{CECName: *cecName, CECDeviceName: *cecDeviceName}
	bridge, err := lib.NewCECMQTTBridge(cecClientConfig, mqttClient, *topicPrefix)
	if err != nil {
		slog.Error("Error creating CECMQTTBridge", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	go bridge.PublishCommands(ctx)
	go bridge.PublishKeyPresses(ctx)
	go bridge.PublishSourceActivations(ctx)
	go bridge.PublishMessages(ctx, true)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	slog.Info("Started")
	go MainLoop(ctx, *bridge)
	<-c
	// bridge.Controller.Close()

	slog.Info("Shut down")
	bridge.CECConnection.Destroy()
	slog.Info("Exit")

	os.Exit(0)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	common "github.com/claes/mqtt-bridges/common"

	"github.com/claes/mqtt-bridges/audio-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: audio-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	topicPrefix := flag.String("topicPrefix", "", "MQTT topic prefix")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	mqttClient, err := common.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating MQTT client", "error", err)
		os.Exit(1)
	}

	bridge, err := lib.NewAudioMQTTBridge(lib.AudioConfig{}, mqttClient, *topicPrefix)
	if err != nil {
		slog.Error("Error creating AudioMQTTBridge bridge", "error", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx := context.TODO()

	fmt.Printf("Started\n")
	go bridge.EventLoop(ctx)
	<-c
	fmt.Printf("Shut down\n")

	os.Exit(0)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/claes/snapcast-mqtt/lib"

	common "github.com/claes/mqtt-bridges/common"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: snapcast-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	snapServerAddress := flag.String("address", "", "Snapcast server address:port")
	topicPrefix := flag.String("topicPrefix", "", "MQTT topic prefix to use")
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	snapClientConfig := lib.SnapClientConfig{SnapServerAddress: *snapServerAddress}

	mqttClient, err := common.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating MQTT client", "error", err)
		os.Exit(1)
	}

	bridge, err := lib.NewSnapcastMQTTBridge(snapClientConfig, mqttClient, *topicPrefix)

	if err != nil {
		slog.Error("Error creating Snapcast-MQTT bridge", "error", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")

	ctx := context.TODO()
	go bridge.EventLoop(ctx)
	<-c
	bridge.SnapClient.Close()
	fmt.Printf("Shut down\n")

	os.Exit(0)
}

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	common "github.com/claes/mqtt-bridges/common"

	"github.com/claes/samsungtv-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: samsungtv-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	tvIPAddress := flag.String("tv", "", "TV IP address")
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

	samsungTvClientConfig := lib.SamsungTVClientConfig{TVIPAddress: *tvIPAddress}

	bridge := lib.NewSamsungRemoteMQTTBridge(samsungTvClientConfig, mqttClient, *topicPrefix)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.MainLoop()
	<-c
	bridge.Controller.Close()
	fmt.Printf("Shut down\n")

	os.Exit(0)
}

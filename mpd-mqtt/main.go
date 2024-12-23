package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	common "github.com/claes/mqtt-bridges/common"

	"github.com/claes/mqtt-bridges/mpd-mqtt/lib"
)

var (
	mpdServer   *string
	mpdPassword *string
	mqttBroker  *string
	topicPrefix *string
	help        *bool
	debug       *bool
)

func init() {
	mpdServer = flag.String("mpd-address", "localhost:6600", "MPD Server address and port")
	mpdPassword = flag.String("mpd-password", "", "MPD password (optional)")
	mqttBroker = flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	topicPrefix = flag.String("topicPrefix", "", "MQTT topic prefix")

	help = flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
}

func printHelp() {
	fmt.Println("Usage: mpd-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	mpdClientConfig := lib.MpdClientConfig{MpdServer: *mpdServer, MpdPassword: *mpdPassword}

	mqttClient, err := common.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating MQTT client", "error", err)
		os.Exit(1)
	}

	bridge, err := lib.NewMpdMQTTBridge(mpdClientConfig, mqttClient, *topicPrefix)
	if err != nil {
		slog.Error("Error creating MPD-MQTT bridge", "error", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")

	go func() {
		bridge.DetectReconnectMPDClient(mpdClientConfig)
	}()

	ctx := context.TODO()
	go bridge.EventLoop(ctx)
	<-c
	bridge.PlaylistWatcher.Close()
	fmt.Printf("Shut down\n")

	os.Exit(0)
}

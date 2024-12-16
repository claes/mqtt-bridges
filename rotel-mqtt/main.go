package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	common "github.com/claes/mqtt-bridges/common"

	"github.com/claes/rotel-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: rotel-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	serialDevice := flag.String("serial", "/dev/ttyUSB0", "Serial device path")
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	topicPrefix := flag.String("topicPrefix", "", "MQTT topic prefix to use")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	rotelClientConfig := lib.RotelClientConfig{SerialDevice: *serialDevice}

	serialPort, err := lib.CreateSerialPort(rotelClientConfig)
	if err != nil {
		slog.Error("Error creating serial device", "error", err, "serialDevice", *serialDevice)
	}
	mqttClient, err := common.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating mqtt client", "error", err, "broker", *mqttBroker)
	}
	bridge := lib.NewRotelMQTTBridge(serialPort, mqttClient, *topicPrefix)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.SerialLoop()
	<-c
	fmt.Printf("Shut down\n")
	os.Exit(0)
}

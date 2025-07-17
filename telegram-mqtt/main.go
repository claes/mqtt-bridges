package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"

	common "github.com/claes/mqtt-bridges/common"

	"github.com/claes/mqtt-bridges/telegram-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: telegram-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	topicPrefix := flag.String("topicPrefix", "", "MQTT topic prefix")
	telegramBotToken := flag.String("telegramToken", "", "Telegram bot token")

	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")

	var chatArgs multiFlag
	flag.Var(&chatArgs, "chat", "Specify chat name to id mapping as name:id (can be used multiple times)")
	flag.Parse()

	chatNameToIds := make(map[string]int64)
	for _, arg := range chatArgs {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) == 2 {
			id, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				chatNameToIds[parts[0]] = id
			}
		}
	}

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

	config := lib.TelegramConfig{
		BotToken:       *telegramBotToken,
		ChatNamesToIds: chatNameToIds,
	}
	bridge, err := lib.NewTelegramMQTTBridge(config, mqttClient, *topicPrefix)
	if err != nil {
		slog.Error("Error creating TelegramMQTTBridge bridge", "error", err)
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

package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type TelegramMQTTBridge struct {
	common.BaseMQTTBridge
	sendMutex   sync.Mutex
	telegramBot *TelegramBot
}

type TelegramConfig struct {
	BotToken       string
	ChatNamesToIds map[string]int64
}

type TelegramBot struct {
	telegramConfig TelegramConfig
}

type TelegramToMQTTPayload struct {
	ChatID   int64  `json:"chat_id"`
	Username string `json:"username"`
	Text     string `json:"text"`
}

type TelegramUpdate struct {
	UpdateID int             `json:"update_id"`
	Message  TelegramMessage `json:"message"`
}

type TelegramGetUpdatesResponse struct {
	Ok     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

type TelegramMessage struct {
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text string `json:"text"`
	From struct {
		Username string `json:"username"`
	} `json:"from"`
}

func NewTelegramMQTTBridge(telegramConfig TelegramConfig, mqttClient mqtt.Client, topicPrefix string) (*TelegramMQTTBridge, error) {

	bot := &TelegramBot{
		telegramConfig: telegramConfig,
	}

	bridge := &TelegramMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		telegramBot: bot,
	}

	for chatName, chatId := range telegramConfig.ChatNamesToIds {
		topic := common.Prefixify(topicPrefix, "telegram/"+chatName+"/send")
		token := mqttClient.Subscribe(topic, 0, bridge.onTelegramMessageSend)
		if token.Error() != nil {
			slog.Error("Error subscribing to topic", "topic", topic, "error", token.Error())
			continue
		}
		slog.Info("Subscribed to topic", "topic", topic, "chatName", chatName, "chatId", chatId)
		token.Wait()
	}
	return bridge, nil
}

func (bridge *TelegramMQTTBridge) findChatNameByID(chatID int64) (string, bool) {
	for key, value := range bridge.telegramBot.telegramConfig.ChatNamesToIds {
		if value == chatID {
			return key, true
		}
	}
	return "", false
}

func (bridge *TelegramMQTTBridge) EventLoop(ctx context.Context) {
	pollInterval := 1 * time.Second
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			updates, err := bridge.getUpdates(offset)
			if err != nil {
				slog.Error("Error fetching Telegram updates", "err", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for _, update := range updates {
				msg := update.Message

				chatName, found := bridge.findChatNameByID(msg.Chat.ID)
				if !found {
					slog.Warn("Chat ID not found in configuration", "chatID", msg.Chat.ID)
					continue
				}
				if msg.Text != "" {
					telegramToMqttPayload := TelegramToMQTTPayload{
						ChatID:   msg.Chat.ID,
						Username: msg.From.Username,
						Text:     msg.Text,
					}
					payload, err := json.Marshal(telegramToMqttPayload)
					if err != nil {
						slog.Error("Error marshalling payload for publish to MQTT", "payload", telegramToMqttPayload)
					}

					token := bridge.MQTTClient.Publish("telegram/"+chatName+"/receive", 0, false, payload)
					token.Wait()
					slog.Debug("Published message to MQTT", "payload", string(payload))
				}
				if update.UpdateID >= offset {
					offset = update.UpdateID + 1
				}
			}
			time.Sleep(pollInterval)
		}
	}
}

func (bridge *TelegramMQTTBridge) getUpdates(offset int) ([]TelegramUpdate, error) {
	telegramPollTimeoutSec := 10

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=%d&offset=%d",
		url.QueryEscape(bridge.telegramBot.telegramConfig.BotToken), telegramPollTimeoutSec, offset)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updatesResp TelegramGetUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&updatesResp); err != nil {
		return nil, err
	}
	bodyBytes, err := json.Marshal(updatesResp)
	if err != nil {
		slog.Error("Error marshalling response", "err", err, "response", updatesResp)
		return nil, err
	}

	slog.Debug("Telegram getUpdates response", "body", string(bodyBytes))
	return updatesResp.Result, nil
}

var chatNameRegex = regexp.MustCompile(`telegram/([^/]+)/send`)

func getChatName(topic string) (string, bool) {
	matches := chatNameRegex.FindStringSubmatch(topic)
	if len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

func (bridge *TelegramMQTTBridge) onTelegramMessageSend(client mqtt.Client, message mqtt.Message) {

	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	chatName, found := getChatName(message.Topic())
	if !found {
		slog.Error("Chat name not found in topic", "topic", message.Topic())
		return
	}
	chatID, exists := bridge.telegramBot.telegramConfig.ChatNamesToIds[chatName]
	if !exists {
		slog.Error("Chat ID not found for chat name", "topic", message.Topic(), "chatName", chatName)
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage",
		url.QueryEscape(bridge.telegramBot.telegramConfig.BotToken))

	text := string(message.Payload())

	slog.Info("Sending message to Telegram", "chatName", chatName, "chatID", chatID, "text", text, "x", strconv.FormatInt(chatID, 10))
	postData := url.Values{}
	postData.Set("chat_id", strconv.FormatInt(chatID, 10))
	postData.Set("text", text)

	resp, err := http.PostForm(apiURL, postData)
	if err != nil {
		slog.Error("Error while posting data to Telegram.", "err", err, "postData", postData)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body struct {
			Description string `json:"description"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		if err != nil {
			slog.Error("Error decoding Telegram API response", "err", err)
			return
		}
		slog.Error("Telegram API error", "description", body.Description)
	}
}

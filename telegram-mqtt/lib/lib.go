package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
	BotToken string
}

type TelegramBot struct {
	BotToken   string
	LastChatID int64
}

type TelegramToMqttPayload struct {
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
		BotToken: telegramConfig.BotToken,
	}

	bridge := &TelegramMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
		telegramBot: bot,
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"telegram/send": bridge.onSendTelegramMessage,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(common.Prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}

	return bridge, nil
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
				if msg.Text != "" {
					bridge.telegramBot.LastChatID = msg.Chat.ID
					telegramToMqttPayload := TelegramToMqttPayload{
						ChatID:   msg.Chat.ID,
						Username: msg.From.Username,
						Text:     msg.Text,
					}
					payload, err := json.Marshal(telegramToMqttPayload)
					if err != nil {
						slog.Error("Error marshalling payload for publish to MQTT", "payload", telegramToMqttPayload)
					}

					token := bridge.MQTTClient.Publish("telegram/receive", 0, false, payload)
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
		url.QueryEscape(bridge.telegramBot.BotToken), telegramPollTimeoutSec, offset)
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

	slog.Info("Telegram getUpdates response body", "body", string(bodyBytes))
	return updatesResp.Result, nil
}

func (bridge *TelegramMQTTBridge) onSendTelegramMessage(client mqtt.Client, message mqtt.Message) {
	if bridge.telegramBot.LastChatID == 0 {
		slog.Error("No known chat ID yet; can't forward MQTT message to Telegram.")
		return
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", url.QueryEscape(bridge.telegramBot.BotToken))

	text := string(message.Payload())

	postData := url.Values{}
	postData.Set("chat_id", strconv.FormatInt(bridge.telegramBot.LastChatID, 10))
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

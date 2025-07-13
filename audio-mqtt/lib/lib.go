package lib

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/flac"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

type AudioMQTTBridge struct {
	common.BaseMQTTBridge
	embeddedFiles embed.FS
	sendMutex     sync.Mutex
}

type readCloser struct {
	io.Reader
	closer func()
}

func (rc *readCloser) Close() error {
	rc.closer()
	return nil
}

func NewAudioMQTTBridge(mqttClient mqtt.Client, topicPrefix string) (*AudioMQTTBridge, error) {
	bridge := &AudioMQTTBridge{
		BaseMQTTBridge: common.BaseMQTTBridge{
			MQTTClient:  mqttClient,
			TopicPrefix: topicPrefix,
		},
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"audio/play": bridge.onPlayURL,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(common.Prefixify(topicPrefix, key), 0, function)
		token.Wait()
	}

	return bridge, nil
}

func (bridge *AudioMQTTBridge) onPlayURL(client mqtt.Client, message mqtt.Message) {
	bridge.sendMutex.Lock()
	defer bridge.sendMutex.Unlock()

	url := string(message.Payload())
	if url != "" {
		slog.Debug("Playing URL", "URL", url)

		err := bridge.playAudio(url)
		if err != nil {
			slog.Error("Error playing URL", "error", err, "URL", url)
		}
	}
}

func (bridge *AudioMQTTBridge) playAudio(audioSource string) error {
	reader, closer, err := bridge.openAudioSource(audioSource)
	defer closer()

	if err != nil {
		return fmt.Errorf("Cannot play audio for %s, %v", audioSource, err)
	}

	buffered := bufio.NewReader(reader)
	rc := &readCloser{
		Reader: buffered,
		closer: closer,
	}
	header, err := buffered.Peek(512)

	formatType := detectAudioFormat(header)

	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
	)

	switch formatType {
	case "mp3":
		streamer, format, err = mp3.Decode(rc)
	case "wav":
		streamer, format, err = wav.Decode(buffered)
	case "flac":
		streamer, format, err = flac.Decode(buffered)
	default:
		return fmt.Errorf("%s not a supported audio format", formatType)
	}
	if err != nil {
		return err
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))
	<-done
	return nil
}

func (bridge *AudioMQTTBridge) openAudioSource(src string) (io.Reader, func(), error) {
	if strings.HasPrefix(src, "embed://") {
		path := strings.TrimPrefix(src, "embed://")
		file, err := bridge.embeddedFiles.Open(path)
		if err != nil {
			return nil, func() {}, fmt.Errorf("Could not open embedded path %s, %v", path, err)
		}
		return file, func() { file.Close() }, nil
	} else if strings.HasPrefix(src, "file://") {
		u, err := url.Parse(src)
		if err != nil {
			return nil, func() {}, fmt.Errorf("Invalid file:// path %s, %v", src, err)
		}
		path := u.Path
		f, err := os.Open(path)
		if err != nil {
			return nil, func() {}, fmt.Errorf("Could not open %s, %v", src, err)
		}
		return f, func() { f.Close() }, nil
	} else if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		resp, err := http.Get(src)
		if err != nil {
			return nil, func() {}, fmt.Errorf("Could not load from %s, %v", src, err)
		}
		return resp.Body, func() { resp.Body.Close() }, nil
	}
	return nil, func() {}, fmt.Errorf("%s not a supported protocol", src)
}

func detectAudioFormat(header []byte) string {
	// Check for WAV: "RIFF" at the start and "WAVE" at bytes 8-11
	if len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WAVE")) {
		return "wav"
	}

	// Check for FLAC: "fLaC" at the start
	if len(header) >= 4 && bytes.Equal(header[:4], []byte("fLaC")) {
		return "flac"
	}

	// Check for MP3: "ID3" at the start or frame sync (FFEx) anywhere in the header
	if len(header) >= 3 && bytes.Equal(header[:3], []byte("ID3")) {
		return "mp3"
	}
	for i := 0; i < len(header)-1; i++ {
		if header[i] == 0xFF && (header[i+1]&0xE0) == 0xE0 {
			return "mp3"
		}
	}

	// Unknown format
	return ""
}

func (bridge *AudioMQTTBridge) EventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Closing down AudioMQTTBridge event loop")
			return
		}
	}
}

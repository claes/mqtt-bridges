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

	"github.com/Defacto2/magicnumber"
	common "github.com/claes/mqtt-bridges/common"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/flac"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/vorbis"
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

	detectorReader, detectorCloser, err := bridge.openAudioSource(audioSource)
	defer detectorCloser()
	if err != nil {
		return fmt.Errorf("Cannot open stream for audio source %s, %v", audioSource, err)
	}

	signature, err := detectAudioTypeFromURL(detectorReader)
	if err != nil {
		return fmt.Errorf("Cannot detect audio type for %s, %v", audioSource, err)
	}

	reader, closer, err := bridge.openAudioSource(audioSource)
	defer closer()

	if err != nil {
		return fmt.Errorf("Cannot play audio for %s, %v", audioSource, err)
	}

	buffered := bufio.NewReader(reader)

	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
	)
	rc := &readCloser{
		Reader: buffered,
		closer: closer,
	}
	defer rc.Close()
	fmt.Println("si" + signature.String())
	switch signature.String() {
	case "MP3 audio":
		streamer, format, err = mp3.Decode(rc)
	case "Wave audio":
		streamer, format, err = wav.Decode(buffered)
	case "Ogg audio":
		streamer, format, err = vorbis.Decode(rc)
	case "FLAC audio":
		streamer, format, err = flac.Decode(buffered)
	default:
		return fmt.Errorf("%s not a supported audio format", signature.String())
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

func detectAudioTypeFromURL(reader io.Reader) (magicnumber.Signature, error) {
	const sniffSize = 1024
	buf := make([]byte, sniffSize)
	n, err := io.ReadFull(reader, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return magicnumber.Unknown, err
	}
	return magicnumber.Find(bytes.NewReader(buf[:n])), nil
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

package dashscope

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

const (
	wsEndpoint       = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	ModelFlashRealtime = "qwen3-tts-flash-realtime"
	ModelVCRealtime    = "qwen3-tts-vc-realtime-2026-01-15"
	ModelEnrollment    = "qwen-voice-enrollment"
)

// RealtimeClient handles WebSocket streaming TTS
type RealtimeClient struct {
	apiKey string
}

func NewRealtimeClient(apiKey string) *RealtimeClient {
	return &RealtimeClient{apiKey: apiKey}
}

// wsMessage represents a generic WebSocket protocol message
type wsMessage struct {
	EventID string `json:"event_id,omitempty"`
	Type    string `json:"type"`
}

type sessionUpdate struct {
	EventID string        `json:"event_id,omitempty"`
	Type    string        `json:"type"`
	Session sessionParams `json:"session"`
}

type sessionParams struct {
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	SampleRate     int     `json:"sample_rate,omitempty"`
	Mode           string  `json:"mode,omitempty"`
	LanguageType   string  `json:"language_type,omitempty"`
	Volume         int     `json:"volume,omitempty"`
	SpeechRate     float64 `json:"speech_rate,omitempty"`
	PitchRate      float64 `json:"pitch_rate,omitempty"`
}

type textAppend struct {
	EventID string `json:"event_id,omitempty"`
	Type    string `json:"type"`
	Text    string `json:"text"`
}

type serverMessage struct {
	Type    string          `json:"type"`
	Delta   string          `json:"delta,omitempty"`
	Session json.RawMessage `json:"session,omitempty"`
}

// StreamTTS opens a WebSocket, sends text, and streams PCM audio chunks via callback.
// The callback receives raw PCM bytes (24000Hz 16-bit mono).
func (rc *RealtimeClient) StreamTTS(ctx context.Context, model, voice, text, lang string, onAudio func([]byte)) error {
	url := fmt.Sprintf("%s?model=%s", wsEndpoint, model)

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{"Bearer " + rc.apiKey},
		},
	})
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(1 << 20) // 1MB — audio chunks can be large

	// Read session.created
	if err := rc.expectMessage(ctx, conn, "session.created"); err != nil {
		return err
	}

	// Send session.update
	langType := "auto"
	if lang != "" {
		langType = lang
	}
	update := sessionUpdate{
		Type: "session.update",
		Session: sessionParams{
			Voice:          voice,
			ResponseFormat: "pcm",
			SampleRate:     24000,
			Mode:           "server_commit",
			LanguageType:   langType,
			Volume:         50,
			SpeechRate:     1.0,
			PitchRate:      1.0,
		},
	}
	if err := rc.writeJSON(ctx, conn, update); err != nil {
		return fmt.Errorf("session.update: %w", err)
	}

	// Send text
	append := textAppend{
		Type: "input_text_buffer.append",
		Text: text,
	}
	if err := rc.writeJSON(ctx, conn, append); err != nil {
		return fmt.Errorf("text append: %w", err)
	}

	// Send finish
	finish := wsMessage{Type: "session.finish"}
	if err := rc.writeJSON(ctx, conn, finish); err != nil {
		return fmt.Errorf("session.finish: %w", err)
	}

	// Read audio chunks until session.finished
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var msg serverMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "response.audio.delta":
			pcm, err := base64.StdEncoding.DecodeString(msg.Delta)
			if err != nil {
				return fmt.Errorf("decode audio: %w", err)
			}
			onAudio(pcm)

		case "response.done":
			// Single response done, keep reading for session.finished
			continue

		case "session.finished":
			conn.Close(websocket.StatusNormalClosure, "done")
			return nil

		case "error":
			return fmt.Errorf("server error: %s", string(data))
		}
	}
}

func (rc *RealtimeClient) expectMessage(ctx context.Context, conn *websocket.Conn, expectedType string) error {
	_, data, err := conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("waiting for %s: %w", expectedType, err)
	}
	var msg serverMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("parse %s: %w", expectedType, err)
	}
	if msg.Type != expectedType {
		return fmt.Errorf("expected %s, got %s", expectedType, msg.Type)
	}
	return nil
}

func (rc *RealtimeClient) writeJSON(ctx context.Context, conn *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

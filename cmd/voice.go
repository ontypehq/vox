package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ontypehq/vox/internal/audio"
	"github.com/ontypehq/vox/internal/config"
	"github.com/ontypehq/vox/internal/dashscope"
	"github.com/ontypehq/vox/internal/ui"
)

type VoiceCmd struct {
	List   VoiceListCmd   `cmd:"" help:"List available voices"`
	Record VoiceRecordCmd `cmd:"" help:"Record and enroll a voice clone"`
	Delete VoiceDeleteCmd `cmd:"" help:"Delete a cloned voice"`
}

// --- voice list ---

type VoiceListCmd struct{}

func (c *VoiceListCmd) Run(cfg *config.AppConfig) error {
	// System voices (always available)
	ui.Info("\n%s", ui.Key("System Voices"))
	ui.Info("%s", ui.Dim("  (use with: vox say --voice <name> \"text\")"))
	for _, v := range dashscope.SystemVoices {
		ui.Info("  %-12s %s  %s", ui.Key(v.ID), ui.Dim(v.Gender), ui.Dim(v.Language))
	}

	// Cloned voices (from API)
	apiKey, err := cfg.RequireAPIKey()
	if err != nil {
		ui.Info("\n%s", ui.Dim("  (login to see cloned voices)"))
		return nil
	}

	client := dashscope.NewClient(apiKey)
	voices, err := client.ListVoices(0, 50)
	if err != nil {
		ui.Warn("Failed to fetch cloned voices: %v", err)
		return nil
	}

	if len(voices) == 0 {
		ui.Info("\n%s", ui.Dim("  No cloned voices. Use: vox voice record --lang zh"))
		return nil
	}

	ui.Info("\n%s", ui.Key("Cloned Voices"))
	for _, v := range voices {
		voiceID, _ := v["voice"].(string)
		lang, _ := v["language"].(string)
		model, _ := v["target_model"].(string)
		// Extract name from voice ID: "qwen-tts-vc-<name>-voice-..."
		name := extractNameFromVoiceID(voiceID)
		ui.Info("  %-12s %s  %s  %s", ui.Key(name), ui.Dim(voiceID), ui.Dim(lang), ui.Dim(model))
	}

	return nil
}

// --- voice record ---

var sampleTexts = map[string]string{
	"zh":      "今天天气真不错，适合出去走走。技术正在以前所未有的速度发展，改变着我们的生活方式。",
	"en":      "The quick brown fox jumps over the lazy dog. Technology is evolving faster than ever before.",
	"ja":      "今日はとても良い天気ですね。テクノロジーはかつてないスピードで進化しています。",
	"Chinese": "今天天气真不错，适合出去走走。技术正在以前所未有的速度发展，改变着我们的生活方式。",
	"English": "The quick brown fox jumps over the lazy dog. Technology is evolving faster than ever before.",
	"Japanese": "今日はとても良い天気ですね。テクノロジーはかつてないスピードで進化しています。",
}

type VoiceRecordCmd struct {
	Lang     string `short:"l" default:"zh" help:"Language for sample text (zh, en, ja)"`
	Name     string `short:"n" help:"Name for the cloned voice"`
	Duration int    `short:"d" default:"10" help:"Recording duration in seconds"`
	File     string `short:"f" help:"Use existing audio file instead of recording"`
}

func (c *VoiceRecordCmd) Run(cfg *config.AppConfig) error {
	apiKey, err := cfg.RequireAPIKey()
	if err != nil {
		return err
	}

	var audioData []byte

	if c.File != "" {
		// Use existing file
		audioData, err = os.ReadFile(c.File)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		ui.Info("Using audio file: %s", ui.Key(c.File))
	} else {
		// Record from microphone
		sample, ok := sampleTexts[c.Lang]
		if !ok {
			sample = sampleTexts["en"]
		}

		ui.Info("\n%s", ui.Key("Read this aloud:"))
		ui.Info("  %s\n", sample)
		ui.Info("Recording for %ds... %s", c.Duration, ui.Dim("(speak now)"))

		recorder, err := audio.NewRecorder(audio.SampleRate, 1)
		if err != nil {
			return fmt.Errorf("init recorder: %w", err)
		}

		recorder.Start()
		time.Sleep(time.Duration(c.Duration) * time.Second)
		audioData = recorder.Stop()

		ui.Success("Recorded %d bytes", len(audioData))

		// Save locally
		wavPath := filepath.Join(cfg.Dir, "voices", fmt.Sprintf("recording-%d.wav", time.Now().Unix()))
		if err := writeWAV(wavPath, audioData); err != nil {
			ui.Warn("Failed to save local copy: %v", err)
		}
	}

	// Determine name
	name := c.Name
	if name == "" {
		name = fmt.Sprintf("vox-%d", time.Now().Unix())
	}

	// Wrap raw PCM in WAV if needed (enrollment expects audio file format)
	var wavData []byte
	if c.File != "" {
		wavData = audioData
	} else {
		wavData = wrapPCMAsWAV(audioData)
	}

	// Enroll voice
	ui.Info("Enrolling voice %s...", ui.Key(name))
	client := dashscope.NewClient(apiKey)
	voiceID, err := client.EnrollVoice(name, base64.StdEncoding.EncodeToString(wavData))
	if err != nil {
		return fmt.Errorf("enroll: %w", err)
	}

	ui.Success("Voice enrolled!")
	ui.KV("Voice ID", voiceID)
	ui.KV("Name", name)
	ui.Info("\n  Use it: %s", ui.Key(fmt.Sprintf("vox say --voice %s \"Hello!\"", voiceID)))

	// Save as last voice
	cfg.State.LastVoice = voiceID
	cfg.SaveState()

	return nil
}

func writeWAV(path string, pcm []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	wav := wrapPCMAsWAV(pcm)
	_, err = f.Write(wav)
	return err
}

func wrapPCMAsWAV(pcm []byte) []byte {
	dataLen := uint32(len(pcm))
	fileLen := dataLen + 36
	sr := uint32(audio.SampleRate)
	br := sr * 2

	header := []byte{
		'R', 'I', 'F', 'F',
		byte(fileLen), byte(fileLen >> 8), byte(fileLen >> 16), byte(fileLen >> 24),
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0,
		1, 0, // PCM
		1, 0, // mono
		byte(sr), byte(sr >> 8), byte(sr >> 16), byte(sr >> 24),
		byte(br), byte(br >> 8), byte(br >> 16), byte(br >> 24),
		2, 0,  // block align
		16, 0, // bits per sample
		'd', 'a', 't', 'a',
		byte(dataLen), byte(dataLen >> 8), byte(dataLen >> 16), byte(dataLen >> 24),
	}

	return append(header, pcm...)
}

// --- voice delete ---

type VoiceDeleteCmd struct {
	VoiceID string `arg:"" help:"Voice ID to delete"`
}

func (c *VoiceDeleteCmd) Run(cfg *config.AppConfig) error {
	apiKey, err := cfg.RequireAPIKey()
	if err != nil {
		return err
	}

	if dashscope.IsSystemVoice(c.VoiceID) {
		return fmt.Errorf("cannot delete system voice: %s", c.VoiceID)
	}

	client := dashscope.NewClient(apiKey)
	if err := client.DeleteVoice(c.VoiceID); err != nil {
		return err
	}

	ui.Success("Deleted voice: %s", c.VoiceID)

	// Clear last voice if it was this one
	if cfg.State.LastVoice == c.VoiceID {
		cfg.State.LastVoice = ""
		cfg.SaveState()
	}

	return nil
}

// extractNameFromVoiceID extracts the user-chosen name from voice ID
// e.g. "qwen-tts-vc-dio-voice-20260220..." → "dio"
func extractNameFromVoiceID(id string) string {
	// Pattern: qwen-tts-vc-<name>-voice-<timestamp>-<hash>
	const prefix = "qwen-tts-vc-"
	const marker = "-voice-"
	if !strings.HasPrefix(id, prefix) {
		return id
	}
	rest := id[len(prefix):]
	idx := strings.Index(rest, marker)
	if idx < 0 {
		return id
	}
	return rest[:idx]
}

// normalizeLang converts shorthand to DashScope language names
func normalizeLang(lang string) string {
	switch strings.ToLower(lang) {
	case "zh", "chinese":
		return "Chinese"
	case "en", "english":
		return "English"
	case "ja", "japanese":
		return "Japanese"
	default:
		return lang
	}
}

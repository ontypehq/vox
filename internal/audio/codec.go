package audio

import (
	"bytes"
	"fmt"
	"os/exec"
)

// EncodePCMToOpus encodes raw PCM (24kHz 16-bit mono) to Opus via ffmpeg.
func EncodePCMToOpus(pcm []byte) ([]byte, error) {
	cmd := exec.Command("ffmpeg",
		"-f", "s16le",
		"-ar", fmt.Sprintf("%d", SampleRate),
		"-ac", fmt.Sprintf("%d", ChannelCount),
		"-i", "pipe:0",
		"-c:a", "libopus",
		"-b:a", "24k",
		"-f", "opus",
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg encode: %w", err)
	}
	return out.Bytes(), nil
}

// DecodeOpusToPCM decodes Opus back to raw PCM (24kHz 16-bit mono) via ffmpeg.
func DecodeOpusToPCM(opus []byte) ([]byte, error) {
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-f", "s16le",
		"-ar", fmt.Sprintf("%d", SampleRate),
		"-ac", fmt.Sprintf("%d", ChannelCount),
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(opus)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg decode: %w", err)
	}
	return out.Bytes(), nil
}

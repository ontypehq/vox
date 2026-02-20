package audio

import (
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	SampleRate   = 24000
	ChannelCount = 1
)

// StreamPlayer plays PCM audio chunks as they arrive
type StreamPlayer struct {
	ctx    *oto.Context
	player *oto.Player
	pw     *io.PipeWriter
	pr     *io.PipeReader
	done   chan struct{}
	once   sync.Once
}

var (
	otoCtx     *oto.Context
	otoCtxOnce sync.Once
)

func getOtoContext() *oto.Context {
	otoCtxOnce.Do(func() {
		op := &oto.NewContextOptions{
			SampleRate:   SampleRate,
			ChannelCount: ChannelCount,
			Format:       oto.FormatSignedInt16LE,
		}
		var ready chan struct{}
		var err error
		otoCtx, ready, err = oto.NewContext(op)
		if err != nil {
			panic("oto init: " + err.Error())
		}
		<-ready
	})
	return otoCtx
}

// NewStreamPlayer creates a player that accepts PCM chunks via Write
func NewStreamPlayer() *StreamPlayer {
	pr, pw := io.Pipe()
	ctx := getOtoContext()
	player := ctx.NewPlayer(pr)

	sp := &StreamPlayer{
		ctx:    ctx,
		player: player,
		pw:     pw,
		pr:     pr,
		done:   make(chan struct{}),
	}

	player.Play()

	return sp
}

// Write sends PCM data to the player. Safe to call from any goroutine.
func (sp *StreamPlayer) Write(pcm []byte) {
	sp.pw.Write(pcm)
}

// Close signals end of audio data and waits for playback to fully drain
func (sp *StreamPlayer) Close() {
	sp.pw.Close()

	// Wait for oto player to finish — poll IsPlaying with a safety timeout.
	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			return
		default:
			if !sp.player.IsPlaying() {
				// oto may report not-playing while audio hardware still has
				// samples in flight. Wait based on oto's internal buffer size.
				// Default buffer is 8192 bytes. At 24kHz 16-bit mono = 48000 bytes/s,
				// that's ~170ms. Add generous margin for OS audio pipeline.
				time.Sleep(500 * time.Millisecond)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// AllPCM collects all written PCM bytes (for caching). Must be used via WriteTee.
type PCMCollector struct {
	buf []byte
}

func (pc *PCMCollector) Write(p []byte) {
	pc.buf = append(pc.buf, p...)
}

func (pc *PCMCollector) Bytes() []byte {
	return pc.buf
}

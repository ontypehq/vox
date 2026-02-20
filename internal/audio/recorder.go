package audio

import (
	"sync"

	"github.com/gen2brain/malgo"
)

// Recorder captures audio from the default input device
type Recorder struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	sampleRate uint32
	channels   uint32
	mu         sync.Mutex
	buf        []byte
}

func NewRecorder(sampleRate, channels int) (*Recorder, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	return &Recorder{
		ctx:        ctx,
		sampleRate: uint32(sampleRate),
		channels:   uint32(channels),
	}, nil
}

func (r *Recorder) Start() error {
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = r.channels
	deviceConfig.SampleRate = r.sampleRate

	onData := func(outputSamples, inputSamples []byte, frameCount uint32) {
		r.mu.Lock()
		r.buf = append(r.buf, inputSamples...)
		r.mu.Unlock()
	}

	callbacks := malgo.DeviceCallbacks{
		Data: onData,
	}

	device, err := malgo.InitDevice(r.ctx.Context, deviceConfig, callbacks)
	if err != nil {
		return err
	}

	r.device = device
	return device.Start()
}

// Stop ends recording and returns the captured PCM data
func (r *Recorder) Stop() []byte {
	if r.device != nil {
		r.device.Stop()
		r.device.Uninit()
	}
	if r.ctx != nil {
		r.ctx.Free()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf
}

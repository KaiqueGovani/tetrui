package main

import (
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

type SoundEvent int

const (
	SoundLock SoundEvent = iota
	SoundLine1
	SoundLine2
	SoundLine3
	SoundLine4
	SoundRotate
	SoundMove
	SoundDrop
	SoundMenuMove
	SoundMenuSelect
)

type SoundEngine struct {
	enabled    bool
	sampleRate int
	ctx        *oto.Context
	mu         sync.RWMutex
}

func NewSoundEngine(enabled bool) *SoundEngine {
	engine := &SoundEngine{
		enabled:    enabled,
		sampleRate: 44100,
	}
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   engine.sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		engine.enabled = false
		return engine
	}
	<-ready
	engine.ctx = ctx
	return engine
}

func (s *SoundEngine) SetEnabled(enabled bool) {
	s.mu.Lock()
	s.enabled = enabled
	s.mu.Unlock()
}

func (s *SoundEngine) Play(event SoundEvent) {
	s.mu.RLock()
	ctx := s.ctx
	enabled := s.enabled
	s.mu.RUnlock()
	if !enabled || ctx == nil {
		return
	}
	sequence := tonesForEvent(event)
	if len(sequence) == 0 {
		return
	}
	go func() {
		buffer := renderToneSequence(sequence, s.sampleRate)
		player := ctx.NewPlayer()
		_, _ = player.Write(buffer)
		_ = player.Close()
	}()
}

type toneSpec struct {
	frequency float64
	duration  time.Duration
}

func tonesForEvent(event SoundEvent) []toneSpec {
	switch event {
	case SoundLock:
		return []toneSpec{{frequency: 220, duration: 70 * time.Millisecond}}
	case SoundLine1:
		return []toneSpec{{frequency: 440, duration: 90 * time.Millisecond}}
	case SoundLine2:
		return []toneSpec{{frequency: 440, duration: 70 * time.Millisecond}, {frequency: 660, duration: 90 * time.Millisecond}}
	case SoundLine3:
		return []toneSpec{{frequency: 440, duration: 70 * time.Millisecond}, {frequency: 660, duration: 70 * time.Millisecond}, {frequency: 880, duration: 90 * time.Millisecond}}
	case SoundLine4:
		return []toneSpec{{frequency: 660, duration: 80 * time.Millisecond}, {frequency: 880, duration: 80 * time.Millisecond}, {frequency: 990, duration: 120 * time.Millisecond}}
	case SoundRotate:
		return []toneSpec{{frequency: 520, duration: 40 * time.Millisecond}}
	case SoundMove:
		return []toneSpec{{frequency: 360, duration: 30 * time.Millisecond}}
	case SoundDrop:
		return []toneSpec{{frequency: 180, duration: 80 * time.Millisecond}}
	case SoundMenuMove:
		return []toneSpec{{frequency: 300, duration: 30 * time.Millisecond}}
	case SoundMenuSelect:
		return []toneSpec{{frequency: 700, duration: 80 * time.Millisecond}}
	default:
		return nil
	}
}

func renderToneSequence(sequence []toneSpec, sampleRate int) []byte {
	volume := 0.3
	gap := 10 * time.Millisecond
	gapSamples := int(float64(sampleRate) * gap.Seconds())
	bytesPerSample := 4
	totalSamples := 0
	for i, spec := range sequence {
		samples := int(float64(sampleRate) * spec.duration.Seconds())
		totalSamples += samples
		if i < len(sequence)-1 {
			totalSamples += gapSamples
		}
	}
	buffer := make([]byte, totalSamples*bytesPerSample)
	index := 0
	for i, spec := range sequence {
		renderTone(buffer, index, spec, sampleRate, volume)
		samples := int(float64(sampleRate) * spec.duration.Seconds())
		index += samples * bytesPerSample
		if i < len(sequence)-1 {
			index += gapSamples * bytesPerSample
		}
	}
	return buffer
}

func renderTone(buffer []byte, start int, spec toneSpec, sampleRate int, volume float64) {
	const maxInt16 = 1<<15 - 1
	samples := int(float64(sampleRate) * spec.duration.Seconds())
	for i := 0; i < samples; i++ {
		sample := math.Sin(2 * math.Pi * spec.frequency * float64(i) / float64(sampleRate))
		value := int16(sample * volume * maxInt16)
		buffer[start+i*4] = byte(value)
		buffer[start+i*4+1] = byte(value >> 8)
		buffer[start+i*4+2] = byte(value)
		buffer[start+i*4+3] = byte(value >> 8)
	}
}

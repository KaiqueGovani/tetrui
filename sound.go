package main

import (
	"bytes"
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
	SoundGameOver
)

type SoundEngine struct {
	enabled    bool
	sampleRate int
	ctx        *oto.Context
	volume     float64
	mu         sync.RWMutex
}

func NewSoundEngine(enabled bool) *SoundEngine {
	engine := &SoundEngine{
		enabled:    enabled,
		sampleRate: 44100,
		volume:     0.7,
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

func (s *SoundEngine) SetVolume(volume float64) {
	s.mu.Lock()
	s.volume = clampVolume(volume)
	s.mu.Unlock()
}

func (s *SoundEngine) Context() *oto.Context {
	s.mu.RLock()
	ctx := s.ctx
	s.mu.RUnlock()
	return ctx
}

func (s *SoundEngine) Play(event SoundEvent) {
	s.mu.RLock()
	ctx := s.ctx
	enabled := s.enabled
	volume := s.volume
	s.mu.RUnlock()
	if !enabled || ctx == nil {
		return
	}
	sequence := tonesForEvent(event)
	if len(sequence) == 0 {
		return
	}
	go func() {
		buffer := renderToneSequence(sequence, s.sampleRate, volume)
		reader := bytes.NewReader(buffer)
		player := ctx.NewPlayer(reader)
		player.Play()
		for player.IsPlaying() {
			time.Sleep(5 * time.Millisecond)
		}
		_ = player.Close()
	}()
}

type toneSpec struct {
	frequency float64
	duration  time.Duration
	volume    float64
}

func tonesForEvent(event SoundEvent) []toneSpec {
	switch event {
	case SoundLock:
		return []toneSpec{{frequency: 220, duration: 70 * time.Millisecond, volume: 0.3}}
	case SoundLine1:
		return []toneSpec{{frequency: 440, duration: 90 * time.Millisecond, volume: 0.3}}
	case SoundLine2:
		return []toneSpec{
			{frequency: 440, duration: 70 * time.Millisecond, volume: 0.3},
			{frequency: 660, duration: 90 * time.Millisecond, volume: 0.3},
		}
	case SoundLine3:
		return []toneSpec{
			{frequency: 440, duration: 70 * time.Millisecond, volume: 0.3},
			{frequency: 660, duration: 70 * time.Millisecond, volume: 0.3},
			{frequency: 880, duration: 90 * time.Millisecond, volume: 0.3},
		}
	case SoundLine4:
		return []toneSpec{
			{frequency: 660, duration: 80 * time.Millisecond, volume: 0.3},
			{frequency: 880, duration: 80 * time.Millisecond, volume: 0.3},
			{frequency: 990, duration: 120 * time.Millisecond, volume: 0.3},
		}
	case SoundRotate:
		return []toneSpec{{frequency: 520, duration: 40 * time.Millisecond, volume: 0.25}}
	case SoundMove:
		return []toneSpec{{frequency: 380, duration: 25 * time.Millisecond, volume: 0.18}}
	case SoundDrop:
		return []toneSpec{{frequency: 240, duration: 55 * time.Millisecond, volume: 0.22}}
	case SoundMenuMove:
		return []toneSpec{{frequency: 260, duration: 24 * time.Millisecond, volume: 0.16}}
	case SoundMenuSelect:
		return []toneSpec{{frequency: 520, duration: 70 * time.Millisecond, volume: 0.2}}
	case SoundGameOver:
		return []toneSpec{{frequency: 180, duration: 160 * time.Millisecond, volume: 0.28}}
	default:
		return nil
	}
}

func renderToneSequence(sequence []toneSpec, sampleRate int, masterVolume float64) []byte {
	baseVolume := 0.3
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
		volume := baseVolume
		if spec.volume > 0 {
			volume = spec.volume
		}
		volume *= clampVolume(masterVolume)
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
	fadeSamples := int(float64(sampleRate) * 0.003)
	for i := 0; i < samples; i++ {
		env := 1.0
		if fadeSamples > 0 {
			if i < fadeSamples {
				env = float64(i) / float64(fadeSamples)
			} else if i > samples-fadeSamples {
				env = float64(samples-i) / float64(fadeSamples)
			}
			if env < 0 {
				env = 0
			}
		}
		sample := math.Sin(2 * math.Pi * spec.frequency * float64(i) / float64(sampleRate))
		value := int16(sample * volume * env * maxInt16)
		buffer[start+i*4] = byte(value)
		buffer[start+i*4+1] = byte(value >> 8)
		buffer[start+i*4+2] = byte(value)
		buffer[start+i*4+3] = byte(value >> 8)
	}
}

func clampVolume(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

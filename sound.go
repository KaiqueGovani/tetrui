package main

import (
	"fmt"
	"sync"
	"time"
)

type SoundEvent int

const (
	SoundLock SoundEvent = iota
	SoundLine1
	SoundLine2
	SoundLine3
	SoundLine4
)

type SoundEngine struct {
	enabled bool
	mu      sync.RWMutex
}

func NewSoundEngine(enabled bool) *SoundEngine {
	return &SoundEngine{enabled: enabled}
}

func (s *SoundEngine) SetEnabled(enabled bool) {
	s.mu.Lock()
	s.enabled = enabled
	s.mu.Unlock()
}

func (s *SoundEngine) Play(event SoundEvent) {
	s.mu.RLock()
	enabled := s.enabled
	s.mu.RUnlock()
	if !enabled {
		return
	}
	sequence := bellSequenceForEvent(event)
	if len(sequence) == 0 {
		return
	}
	go func() {
		for _, tone := range sequence {
			setBellTone(tone.frequency, tone.duration)
			fmt.Print("\a")
			time.Sleep(tone.duration + 10*time.Millisecond)
		}
	}()
}

type bellSpec struct {
	frequency int
	duration  time.Duration
}

func bellSequenceForEvent(event SoundEvent) []bellSpec {
	switch event {
	case SoundLock:
		return []bellSpec{{frequency: 360, duration: 60 * time.Millisecond}}
	case SoundLine1:
		return []bellSpec{{frequency: 660, duration: 80 * time.Millisecond}}
	case SoundLine2:
		return []bellSpec{
			{frequency: 660, duration: 60 * time.Millisecond},
			{frequency: 820, duration: 90 * time.Millisecond},
		}
	case SoundLine3:
		return []bellSpec{
			{frequency: 660, duration: 60 * time.Millisecond},
			{frequency: 820, duration: 60 * time.Millisecond},
			{frequency: 1040, duration: 90 * time.Millisecond},
		}
	case SoundLine4:
		return []bellSpec{
			{frequency: 740, duration: 70 * time.Millisecond},
			{frequency: 980, duration: 70 * time.Millisecond},
			{frequency: 1280, duration: 120 * time.Millisecond},
		}
	default:
		return nil
	}
}

func setBellTone(frequency int, duration time.Duration) {
	if frequency > 0 {
		fmt.Printf("\033[10;%d]", frequency)
	}
	ms := int(duration / time.Millisecond)
	if ms > 0 {
		fmt.Printf("\033[11;%d]", ms)
	}
}

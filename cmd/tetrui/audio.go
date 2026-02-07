package main

import (
	"sync"

	"github.com/ebitengine/oto/v3"
)

var (
	audioOnce       sync.Once
	audioCtx        *oto.Context
	audioSampleRate int
	audioErr        error
)

func initAudioContext() (*oto.Context, int, error) {
	audioOnce.Do(func() {
		sampleRate := 44100
		if dec, err := newSafeDecoder(); err == nil {
			sampleRate = dec.SampleRate()
		} else {
			DebugLogf("audio sample rate fallback: %v", err)
		}
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
		})
		if err != nil {
			audioErr = err
			return
		}
		<-ready
		audioCtx = ctx
		audioSampleRate = sampleRate
	})
	return audioCtx, audioSampleRate, audioErr
}

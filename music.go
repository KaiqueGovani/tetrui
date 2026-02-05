package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"io"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ebitengine/oto/v3"
	"github.com/llehouerou/go-mp3"
)

//go:embed tetris-theme-korobeiniki.mp3
var musicFS embed.FS

const musicFile = "tetris-theme-korobeiniki.mp3"

type MusicMode int

const (
	musicOff MusicMode = iota
	musicMenu
	musicGame
)

type MusicPlayer struct {
	ctx    *oto.Context
	mu     sync.Mutex
	mode   MusicMode
	player *oto.Player
	dec    *safeDecoder
	stop   chan struct{}
	volume float64
}

func NewMusicPlayer(ctx *oto.Context, volume float64, enabled bool) *MusicPlayer {
	if ctx == nil {
		return nil
	}
	player := &MusicPlayer{
		ctx:    ctx,
		mode:   musicOff,
		volume: clampVolume(volume),
	}
	if !enabled {
		player.mode = musicOff
	}
	return player
}

func (m *MusicPlayer) SetVolume(volume float64) {
	m.mu.Lock()
	m.volume = clampVolume(volume)
	m.mu.Unlock()
}

func (m *MusicPlayer) StartMenuCmd() tea.Cmd {
	return func() tea.Msg {
		m.StartMenu()
		return nil
	}
}

func (m *MusicPlayer) StartGameCmd() tea.Cmd {
	return func() tea.Msg {
		m.StartGame()
		return nil
	}
}

func (m *MusicPlayer) StartMenu() {
	m.start(musicMenu, time.Second, 38*time.Second)
}

func (m *MusicPlayer) StartGame() {
	m.start(musicGame, 0, 0)
}

func (m *MusicPlayer) Stop() {
	m.mu.Lock()
	m.stopLocked()
	m.mode = musicOff
	m.mu.Unlock()
}

func (m *MusicPlayer) start(mode MusicMode, loopStart, loopEnd time.Duration) {
	m.mu.Lock()
	if m.mode == mode && m.player != nil {
		m.mu.Unlock()
		return
	}
	m.stopLocked()
	dec, err := newSafeDecoder()
	if err != nil {
		m.mu.Unlock()
		return
	}
	if loopEnd <= 0 {
		loopEnd = dec.Duration()
		if loopEnd <= 0 {
			loopEnd = 0
		}
	}
	_ = dec.SeekToTime(loopStart)
	vr := &volumeReader{
		reader:    dec,
		getVolume: m.volumeValue,
	}
	player := m.ctx.NewPlayer(vr)
	player.Play()
	m.player = player
	m.dec = dec
	m.stop = make(chan struct{})
	m.mode = mode
	stop := m.stop
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if loopEnd > 0 {
					if dec.Position() >= loopEnd {
						_ = dec.SeekToTime(loopStart)
						player.Play()
					}
				} else if !player.IsPlaying() {
					_ = dec.SeekToTime(loopStart)
					player.Play()
				}
			}
		}
	}()
}

func (m *MusicPlayer) stopLocked() {
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	if m.player != nil {
		_ = m.player.Close()
		m.player = nil
	}
	m.dec = nil
}

func (m *MusicPlayer) volumeValue() float64 {
	m.mu.Lock()
	volume := m.volume
	m.mu.Unlock()
	return volume
}

type safeDecoder struct {
	mu  sync.Mutex
	dec *mp3.Decoder
}

func newSafeDecoder() (*safeDecoder, error) {
	data, err := musicFS.ReadFile(musicFile)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(data)
	dec, err := mp3.NewDecoder(reader)
	if err != nil {
		return nil, err
	}
	return &safeDecoder{dec: dec}, nil
}

func (s *safeDecoder) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dec.Read(p)
}

func (s *safeDecoder) SeekToTime(t time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dec.SeekToTime(t)
}

func (s *safeDecoder) Position() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dec.Position()
}

func (s *safeDecoder) Duration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dec.Duration()
}

type volumeReader struct {
	reader    io.Reader
	getVolume func() float64
}

func (v *volumeReader) Read(p []byte) (int, error) {
	n, err := v.reader.Read(p)
	volume := clampVolume(v.getVolume())
	if volume >= 0.999 {
		return n, err
	}
	for i := 0; i+1 < n; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(p[i:]))
		scaled := int16(float64(sample) * volume)
		binary.LittleEndian.PutUint16(p[i:], uint16(scaled))
	}
	return n, err
}

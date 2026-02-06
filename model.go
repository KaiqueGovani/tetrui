package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Screen int

const (
	screenMenu Screen = iota
	screenGame
	screenThemes
	screenScores
	screenConfig
	screenNameEntry
)

type tickMsg struct{}
type soundMsg struct{}
type scoresLoadedMsg struct {
	scores []ScoreEntry
	err    error
}

type scoreUploadedMsg struct {
	err error
}

type syncTickMsg struct{}

type Model struct {
	screen       Screen
	width        int
	height       int
	menuIndex    int
	configIndex  int
	themeIndex   int
	scoresOffset int
	config       Config
	scores       []ScoreEntry
	game         Game
	nameInput    string
	sound        *SoundEngine
	sync         *ScoreSync
	syncWarning  string
	syncLoading  bool
	syncDots     int
	music        *MusicPlayer
}

func NewModel() Model {
	config, _ := loadConfig()
	scores, _ := loadScores()
	index := themeIndexByName(config.Theme)
	if index < 0 {
		index = 0
		config.Theme = themes[index].Name
	}
	ctx, sampleRate, err := initAudioContext()
	if err != nil {
		DebugLogf("audio context init error: %v", err)
	}
	sound := NewSoundEngine(ctx, sampleRate, config.Sound)
	sound.SetVolume(volumeFromPercent(config.Volume))
	return Model{
		screen:     screenMenu,
		config:     config,
		scores:     scores,
		themeIndex: index,
		game:       NewGame(),
		sound:      sound,
		sync:       NewScoreSyncFromEnv(config.Sync),
		music:      NewMusicPlayer(ctx, sampleRate, volumeFromPercent(config.Volume), config.Music),
	}
}

func (m Model) Init() tea.Cmd {
	return m.syncMusicForScreen()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		if m.screen == screenGame && !m.game.Paused && !m.game.Over {
			locked, cleared := m.game.Step()
			if m.game.Over {
				cmd := m.setScreen(screenNameEntry)
				m.nameInput = ""
				return m, cmd
			}
			cmds := []tea.Cmd{tickCmd(m.game.FallInterval())}
			if event, ok := soundEventForAction(locked, cleared); ok && m.config.Sound {
				cmds = append(cmds, playSound(m.sound, event))
			}
			return m, tea.Batch(cmds...)
		}
		if m.screen == screenGame {
			return m, tickCmd(m.game.FallInterval())
		}
		return m, nil
	case soundMsg:
		return m, nil
	case syncTickMsg:
		if m.syncLoading {
			m.syncDots = (m.syncDots + 1) % 4
			return m, syncTickCmd()
		}
		return m, nil
	case scoresLoadedMsg:
		if msg.err != nil {
			DebugLogf("scores fetch error: %v", msg.err)
			m.syncWarning = "Offline: scores not synced."
			m.syncLoading = false
			return m, nil
		}
		if m.sync == nil || !m.sync.Enabled() {
			m.syncWarning = "Score sync is disabled."
		} else {
			m.syncWarning = ""
		}
		if len(msg.scores) > 0 {
			m.scores = mergeScores(m.scores, msg.scores)
			_ = saveScores(m.scores)
		}
		m.syncLoading = false
		return m, nil
	case scoreUploadedMsg:
		if msg.err != nil {
			DebugLogf("score upload error: %v", msg.err)
			m.syncWarning = "Offline: scores not synced."
			m.syncLoading = false
			return m, nil
		}
		m.syncWarning = ""
		m.syncLoading = false
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+=", "ctrl++":
			m.adjustScale(1)
			return m, nil
		case "ctrl+-", "ctrl+_":
			m.adjustScale(-1)
			return m, nil
		}
		switch m.screen {
		case screenMenu:
			return m, m.updateMenu(msg)
		case screenGame:
			return m, m.updateGame(msg)
		case screenThemes:
			return m, m.updateThemes(msg)
		case screenScores:
			return m, m.updateScores(msg)
		case screenConfig:
			return m, m.updateConfig(msg)
		case screenNameEntry:
			return m, m.updateNameEntry(msg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenMenu:
		return viewMenu(m)
	case screenGame:
		return viewGame(m)
	case screenThemes:
		return viewThemes(m)
	case screenScores:
		return viewScores(m)
	case screenConfig:
		return viewConfig(m)
	case screenNameEntry:
		return viewNameEntry(m)
	default:
		return ""
	}
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return tickMsg{} })
}

func syncTickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return syncTickMsg{} })
}

func playSound(engine *SoundEngine, event SoundEvent) tea.Cmd {
	return func() tea.Msg {
		if engine != nil {
			engine.Play(event)
		}
		return soundMsg{}
	}
}

func soundEventForAction(locked bool, cleared int) (SoundEvent, bool) {
	if cleared > 0 {
		switch cleared {
		case 1:
			return SoundLine1, true
		case 2:
			return SoundLine2, true
		case 3:
			return SoundLine3, true
		default:
			return SoundLine4, true
		}
	}
	if locked {
		return SoundLock, true
	}
	return SoundLock, false
}

func (m *Model) adjustScale(delta int) {
	minScale := 1
	maxScale := 3
	newScale := m.config.Scale + delta
	if newScale < minScale {
		newScale = minScale
	}
	if newScale > maxScale {
		newScale = maxScale
	}
	if newScale != m.config.Scale {
		m.config.Scale = newScale
		_ = saveConfig(m.config)
	}
}

func (m *Model) adjustVolume(delta int) {
	newVolume := m.config.Volume + delta
	if newVolume < 0 {
		newVolume = 0
	}
	if newVolume > 100 {
		newVolume = 100
	}
	if newVolume == m.config.Volume {
		return
	}
	m.config.Volume = newVolume
	if m.sound != nil {
		m.sound.SetVolume(volumeFromPercent(newVolume))
	}
	if m.music != nil {
		m.music.SetVolume(volumeFromPercent(newVolume))
	}
	_ = saveConfig(m.config)
}

func volumeFromPercent(value int) float64 {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return float64(value) / 100
}

func (m *Model) setScreen(screen Screen) tea.Cmd {
	m.screen = screen
	return m.syncMusicForScreen()
}

func (m *Model) syncMusicForScreen() tea.Cmd {
	if m.music == nil {
		DebugLogf("music sync skipped: player nil")
		return nil
	}
	if !m.config.Music {
		DebugLogf("music sync stopped: disabled")
		m.music.Stop()
		return nil
	}
	if m.screen == screenGame {
		DebugLogf("music sync: start game")
		m.music.StartGame()
		return nil
	}
	DebugLogf("music sync: stop (non-game)")
	m.music.Stop()
	return nil
}

func (m *Model) updateMenu(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch msg.String() {
	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
			if m.config.Sound {
				cmd = playSound(m.sound, SoundMenuMove)
			}
		}
	case "down", "j":
		if m.menuIndex < len(menuItems)-1 {
			m.menuIndex++
			if m.config.Sound {
				cmd = playSound(m.sound, SoundMenuMove)
			}
		}
	case "enter":
		if m.config.Sound {
			cmd = playSound(m.sound, SoundMenuSelect)
		}
		switch m.menuIndex {
		case 0:
			m.game = NewGame()
			return tea.Batch(cmd, m.setScreen(screenGame), tickCmd(m.game.FallInterval()))
		case 1:
			return tea.Batch(cmd, m.setScreen(screenThemes))
		case 2:
			m.scoresOffset = 0
			if m.sync != nil && m.sync.Enabled() {
				m.syncLoading = true
				m.syncDots = 0
				return tea.Batch(cmd, m.setScreen(screenScores), m.sync.FetchScoresCmd(), syncTickCmd())
			}
			m.syncWarning = "Score sync is disabled."
			return tea.Batch(cmd, m.setScreen(screenScores))
		case 3:
			return tea.Batch(cmd, m.setScreen(screenConfig))
		case 4:
			return tea.Quit
		}
	case "q", "esc":
		return tea.Quit
	}
	return cmd
}

func (m *Model) updateGame(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "left", "h":
		if m.game.Move(-1) && m.config.Sound {
			return playSound(m.sound, SoundMove)
		}
	case "right", "l":
		if m.game.Move(1) && m.config.Sound {
			return playSound(m.sound, SoundMove)
		}
	case "down", "j":
		m.game.SoftDrop()
	case " ":
		locked, cleared := m.game.HardDrop()
		if m.game.Over {
			cmd := m.setScreen(screenNameEntry)
			m.nameInput = ""
			return cmd
		}
		if m.config.Sound {
			if cleared == 0 {
				return playSound(m.sound, SoundDrop)
			}
			if event, ok := soundEventForAction(locked, cleared); ok {
				return playSound(m.sound, event)
			}
		}
	case "up", "x":
		m.game.Rotate(1)
		if m.config.Sound {
			return playSound(m.sound, SoundRotate)
		}
	case "z":
		m.game.Rotate(-1)
		if m.config.Sound {
			return playSound(m.sound, SoundRotate)
		}
	case "c":
		m.game.Hold()
	case "p":
		m.game.Paused = !m.game.Paused
	case "q", "esc":
		return m.setScreen(screenMenu)
	}
	return nil
}

func (m *Model) updateThemes(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.themeIndex > 0 {
			m.themeIndex--
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
	case "down", "j":
		if m.themeIndex < len(themes)-1 {
			m.themeIndex++
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
	case "enter":
		m.config.Theme = themes[m.themeIndex].Name
		_ = saveConfig(m.config)
		cmd := m.setScreen(screenMenu)
		if m.config.Sound {
			return tea.Batch(cmd, playSound(m.sound, SoundMenuSelect))
		}
		return cmd
	case "q", "esc":
		return m.setScreen(screenMenu)
	}
	return nil
}

func (m *Model) updateScores(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "esc", "enter":
		cmd := m.setScreen(screenMenu)
		if m.config.Sound {
			return tea.Batch(cmd, playSound(m.sound, SoundMenuSelect))
		}
		return cmd
	case "up", "k":
		if m.scoresOffset > 0 {
			m.scoresOffset--
		}
	case "down", "j":
		max := len(m.scores) - scoresPageSize
		if max < 0 {
			max = 0
		}
		if m.scoresOffset < max {
			m.scoresOffset++
		}
	}
	return nil
}

func (m *Model) updateConfig(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.configIndex > 0 {
			m.configIndex--
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
	case "down", "j":
		if m.configIndex < len(configItems)-1 {
			m.configIndex++
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
	case "enter":
		switch m.configIndex {
		case 0:
			m.config.Sound = !m.config.Sound
			if m.sound != nil {
				m.sound.SetEnabled(m.config.Sound)
			}
			_ = saveConfig(m.config)
		case 1:
			m.config.Music = !m.config.Music
			_ = saveConfig(m.config)
			if m.config.Sound {
				return tea.Batch(m.syncMusicForScreen(), playSound(m.sound, SoundMenuSelect))
			}
			return m.syncMusicForScreen()
		case 2:
			m.adjustVolume(5)
		case 3:
			m.config.Shadow = !m.config.Shadow
			_ = saveConfig(m.config)
		case 4:
			m.adjustScale(1)
		case 5:
			m.config.Sync = !m.config.Sync
			if m.sync != nil {
				m.sync.SetEnabled(m.config.Sync)
			}
			_ = saveConfig(m.config)
		}
		if m.config.Sound {
			return playSound(m.sound, SoundMenuSelect)
		}
	case "left", "h":
		if m.configIndex == 2 {
			m.adjustVolume(-5)
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
		if m.configIndex == 4 {
			m.adjustScale(-1)
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
	case "right", "l":
		if m.configIndex == 2 {
			m.adjustVolume(5)
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
		if m.configIndex == 4 {
			m.adjustScale(1)
			if m.config.Sound {
				return playSound(m.sound, SoundMenuMove)
			}
		}
	case "q", "esc":
		return m.setScreen(screenMenu)
	}
	return nil
}

func (m *Model) updateNameEntry(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.nameInput)
		if name == "" {
			name = "AAA"
		}
		entry := ScoreEntry{
			Name:  name,
			Score: m.game.Score,
			Lines: m.game.Lines,
			Level: m.game.Level,
			When:  time.Now().Format("2006-01-02 15:04"),
		}
		m.scores = insertScore(m.scores, entry)
		_ = saveScores(m.scores)
		m.scoresOffset = 0
		cmd := m.setScreen(screenScores)
		var cmds []tea.Cmd
		if m.sync != nil && m.sync.Enabled() {
			m.syncLoading = true
			m.syncDots = 0
			cmds = append(cmds, m.sync.UploadScoreCmd(entry))
			cmds = append(cmds, m.sync.FetchScoresCmd())
			cmds = append(cmds, syncTickCmd())
		}
		if len(cmds) == 0 {
			return cmd
		}
		cmds = append(cmds, cmd)
		return tea.Batch(cmds...)
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		}
	case tea.KeyRunes:
		if len(m.nameInput) < 12 {
			m.nameInput += string(msg.Runes)
		}
	case tea.KeyEsc:
		return m.setScreen(screenMenu)
	}
	return nil
}

var menuItems = []string{
	"Start Game",
	"Themes",
	"Scores",
	"Config",
	"Quit",
}

var configItems = []string{
	"Sound Effects",
	"Music",
	"Volume",
	"Shadow",
	"Game Scale",
	"Score Sync",
}

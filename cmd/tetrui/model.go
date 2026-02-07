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
type lineClearTickMsg struct{}
type countdownTickMsg struct{}
type topOutTickMsg struct{}
type hardDropTraceTickMsg struct{}

const (
	lineClearFlashDuration    = 140 * time.Millisecond
	lineClearBigFlashDuration = 160 * time.Millisecond
	hardDropTraceDuration     = 100 * time.Millisecond
)

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
	flashRows    []int
	flashStart   time.Time
	flashUntil   time.Time
	lastDelta    int
	lastEvent    string
	lastEventTil time.Time
	lastMoveDir  int
	lastMoveAt   time.Time
	lineClearTil time.Time
	startCount   int
	topOutTil    time.Time
	hardDropPath []Point
	hardDropDest []Point
	hardDropFrom time.Time
	hardDropTil  time.Time
}

func NewModel() Model {
	config, _ := loadConfig()
	index := themeIndexByName(config.Theme)
	if index < 0 {
		index = 0
		config.Theme = themes[index].Name
	}
	sync := NewScoreSyncFromEnv(config.Sync)
	scores := []ScoreEntry{}
	if sync == nil || !sync.Enabled() {
		scores, _ = loadScores()
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
		sync:       sync,
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
			if m.startCount > 0 {
				return m, nil
			}
			m.updateFlash()
			if m.isLineClearAnimating() {
				return m, tickCmd(m.game.FallInterval())
			}
			result := m.game.Step()
			if m.game.Over {
				return m, m.startTopOutEffect()
			}
			cmds := []tea.Cmd{tickCmd(m.game.FallInterval())}
			if result.Locked {
				if cmd := m.applyScoreEvent(result); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if comboCmd := m.comboSoundCmd(result); comboCmd != nil {
					cmds = append(cmds, comboCmd)
				}
			}
			if event, ok := soundEventForAction(result); ok && m.config.Sound {
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
	case lineClearTickMsg:
		if m.screen != screenGame || m.game.Over {
			return m, nil
		}
		m.updateFlash()
		if m.isLineClearAnimating() {
			return m, lineClearTickCmd()
		}
		m.game.ResolveLineClear()
		if m.game.Over {
			return m, m.startTopOutEffect()
		}
		return m, nil
	case countdownTickMsg:
		if m.screen != screenGame || m.game.Paused || m.game.Over {
			return m, nil
		}
		if m.startCount <= 0 {
			return m, tickCmd(m.game.FallInterval())
		}
		m.startCount--
		if m.startCount > 0 {
			return m, countdownTickCmd()
		}
		if m.config.Sound {
			return m, tea.Batch(playSound(m.sound, SoundMenuSelect), tickCmd(m.game.FallInterval()))
		}
		return m, tickCmd(m.game.FallInterval())
	case topOutTickMsg:
		if m.screen != screenGame || m.topOutTil.IsZero() {
			return m, nil
		}
		m.updateFlash()
		if m.isTopOutAnimating() {
			return m, topOutTickCmd()
		}
		m.topOutTil = time.Time{}
		cmd := m.setScreen(screenNameEntry)
		m.nameInput = ""
		return m, cmd
	case hardDropTraceTickMsg:
		if m.screen != screenGame || m.hardDropTil.IsZero() {
			return m, nil
		}
		m.updateFlash()
		if m.isHardDropTraceAnimating() {
			return m, hardDropTraceTickCmd()
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
		m.scores = msg.scores
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

func lineClearTickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg { return lineClearTickMsg{} })
}

func countdownTickCmd() tea.Cmd {
	return tea.Tick(380*time.Millisecond, func(time.Time) tea.Msg { return countdownTickMsg{} })
}

func topOutTickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg { return topOutTickMsg{} })
}

func hardDropTraceTickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg { return hardDropTraceTickMsg{} })
}

func playSound(engine *SoundEngine, event SoundEvent) tea.Cmd {
	return func() tea.Msg {
		if engine != nil {
			engine.Play(event)
		}
		return soundMsg{}
	}
}

func playComboSound(engine *SoundEngine, combo, backToBack int) tea.Cmd {
	return func() tea.Msg {
		if engine != nil {
			engine.PlayCombo(combo, backToBack)
		}
		return soundMsg{}
	}
}

func soundEventForAction(result LockResult) (SoundEvent, bool) {
	if result.TSpin {
		return SoundTSpin, true
	}
	if result.Cleared > 0 {
		switch result.Cleared {
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
	if result.Locked {
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
			m.startCount = 2
			return tea.Batch(cmd, m.setScreen(screenGame), countdownTickCmd())
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
	if m.startCount > 0 {
		switch msg.String() {
		case "q", "esc":
			return m.setScreen(screenMenu)
		}
		return nil
	}

	if m.isLineClearAnimating() {
		switch msg.String() {
		case "q", "esc":
			return m.setScreen(screenMenu)
		}
		return nil
	}

	switch msg.String() {
	case "left", "h":
		m.lastMoveDir = -1
		m.lastMoveAt = time.Now()
		if m.game.Move(-1) {
			if m.config.Sound {
				return playSound(m.sound, SoundMove)
			}
		}
	case "right", "l":
		m.lastMoveDir = 1
		m.lastMoveAt = time.Now()
		if m.game.Move(1) {
			if m.config.Sound {
				return playSound(m.sound, SoundMove)
			}
		}
	case "down", "j":
		m.game.SoftDrop()
	case " ":
		traceCmd := m.startHardDropTrace()
		result := m.game.HardDrop()
		if m.game.Over {
			topOutCmd := m.startTopOutEffect()
			if traceCmd != nil {
				return tea.Batch(traceCmd, topOutCmd)
			}
			return topOutCmd
		}
		animCmd := m.applyScoreEvent(result)
		cmds := []tea.Cmd{}
		if traceCmd != nil {
			cmds = append(cmds, traceCmd)
		}
		if animCmd != nil {
			cmds = append(cmds, animCmd)
		}
		if comboCmd := m.comboSoundCmd(result); comboCmd != nil {
			cmds = append(cmds, comboCmd)
		}
		if m.config.Sound {
			if result.Cleared == 0 && !result.TSpin {
				soundCmd := playSound(m.sound, SoundDrop)
				cmds = append(cmds, soundCmd)
				if len(cmds) == 0 {
					return nil
				}
				return tea.Batch(cmds...)
			}
			if event, ok := soundEventForAction(result); ok {
				soundCmd := playSound(m.sound, event)
				cmds = append(cmds, soundCmd)
				if len(cmds) == 0 {
					return nil
				}
				return tea.Batch(cmds...)
			}
		}
		if len(cmds) > 0 {
			return tea.Batch(cmds...)
		}
	case "up", "x":
		m.game.Rotate(1)
		m.applyMoveBuffer()
		if m.config.Sound {
			return playSound(m.sound, SoundRotate)
		}
	case "z":
		m.game.Rotate(-1)
		m.applyMoveBuffer()
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
			m.config.Animations = !m.config.Animations
			if !m.config.Animations {
				m.flashRows = nil
				m.flashStart = time.Time{}
				m.flashUntil = time.Time{}
				m.lineClearTil = time.Time{}
				m.game.ResolveLineClear()
			}
			_ = saveConfig(m.config)
		case 5:
			m.config.HardDropTrace = !m.config.HardDropTrace
			if !m.config.HardDropTrace {
				m.hardDropPath = nil
				m.hardDropDest = nil
				m.hardDropFrom = time.Time{}
				m.hardDropTil = time.Time{}
			}
			_ = saveConfig(m.config)
		case 6:
			m.adjustScale(1)
		case 7:
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
		if m.configIndex == 6 {
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
		if m.configIndex == 6 {
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
		if m.sync == nil || !m.sync.Enabled() {
			m.scores = insertScore(m.scores, entry)
			_ = saveScores(m.scores)
		}
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
	"Line Clear Animation",
	"Hard Drop Trace",
	"Game Scale",
	"Score Sync",
}

func (m *Model) applyScoreEvent(result LockResult) tea.Cmd {
	var animCmd tea.Cmd
	if len(result.ClearedRows) > 0 {
		if m.config.Animations {
			m.flashRows = append([]int{}, result.ClearedRows...)
			flash := lineClearFlashDuration
			if result.TSpin || result.Cleared >= 4 {
				flash = lineClearBigFlashDuration
			}
			m.flashStart = time.Now()
			m.flashUntil = m.flashStart.Add(flash)
			m.lineClearTil = m.flashUntil
			animCmd = lineClearTickCmd()
		} else {
			m.flashRows = nil
			m.flashStart = time.Time{}
			m.flashUntil = time.Time{}
			m.lineClearTil = time.Time{}
			m.game.ResolveLineClear()
		}
	}
	if result.ScoreDelta > 0 {
		m.lastDelta = result.ScoreDelta
		if result.TSpin {
			m.lastEvent = "T-SPIN"
		} else {
			m.lastEvent = "LINE CLEAR"
		}
		duration := 900 * time.Millisecond
		if result.TSpin || result.Cleared >= 4 {
			duration = 1400 * time.Millisecond
		}
		m.lastEventTil = time.Now().Add(duration)
	}
	return animCmd
}

func (m *Model) updateFlash() {
	if !m.flashUntil.IsZero() && time.Now().After(m.flashUntil) {
		m.flashRows = nil
		m.flashStart = time.Time{}
		m.flashUntil = time.Time{}
	}
	if !m.lineClearTil.IsZero() && time.Now().After(m.lineClearTil) {
		m.lineClearTil = time.Time{}
	}
	if !m.lastEventTil.IsZero() && time.Now().After(m.lastEventTil) {
		m.lastEvent = ""
		m.lastDelta = 0
		m.lastEventTil = time.Time{}
	}
	if !m.topOutTil.IsZero() && time.Now().After(m.topOutTil) {
		m.topOutTil = time.Time{}
	}
	if !m.hardDropTil.IsZero() && time.Now().After(m.hardDropTil) {
		m.hardDropPath = nil
		m.hardDropDest = nil
		m.hardDropFrom = time.Time{}
		m.hardDropTil = time.Time{}
	}
}

func (m *Model) isLineClearAnimating() bool {
	return !m.lineClearTil.IsZero() && time.Now().Before(m.lineClearTil)
}

func (m *Model) isTopOutAnimating() bool {
	return !m.topOutTil.IsZero() && time.Now().Before(m.topOutTil)
}

func (m *Model) isHardDropTraceAnimating() bool {
	return !m.hardDropTil.IsZero() && time.Now().Before(m.hardDropTil)
}

func (m *Model) startTopOutEffect() tea.Cmd {
	m.flashRows = make([]int, boardHeight)
	for i := 0; i < boardHeight; i++ {
		m.flashRows[i] = i
	}
	m.flashStart = time.Now()
	m.flashUntil = m.flashStart.Add(240 * time.Millisecond)
	m.topOutTil = m.flashUntil
	cmds := []tea.Cmd{topOutTickCmd()}
	if m.config.Sound {
		cmds = append(cmds, playSound(m.sound, SoundGameOver))
	}
	return tea.Batch(cmds...)
}

func (m *Model) comboSoundCmd(result LockResult) tea.Cmd {
	if !m.config.Sound || result.Combo <= 1 {
		return nil
	}
	return playComboSound(m.sound, result.Combo, result.BackToBack)
}

func (m *Model) startHardDropTrace() tea.Cmd {
	if !m.config.HardDropTrace {
		return nil
	}
	ghostY := m.game.GhostY()
	if ghostY <= m.game.Y {
		return nil
	}
	pathMap := make(map[Point]struct{})
	destMap := make(map[Point]struct{})
	for _, block := range pieceRotations[m.game.Current][m.game.Rotation] {
		dx := m.game.X + block.X
		startY := m.game.Y + block.Y
		destY := ghostY + block.Y
		if dx < 0 || dx >= boardWidth {
			continue
		}
		if destY >= 0 && destY < boardHeight {
			destMap[Point{X: dx, Y: destY}] = struct{}{}
		}
		for y := startY; y < destY; y++ {
			if y < 0 || y >= boardHeight {
				continue
			}
			pathMap[Point{X: dx, Y: y}] = struct{}{}
		}
	}
	if len(pathMap) == 0 && len(destMap) == 0 {
		return nil
	}
	m.hardDropPath = make([]Point, 0, len(pathMap))
	for p := range pathMap {
		m.hardDropPath = append(m.hardDropPath, p)
	}
	m.hardDropDest = make([]Point, 0, len(destMap))
	for p := range destMap {
		m.hardDropDest = append(m.hardDropDest, p)
	}
	now := time.Now()
	m.hardDropFrom = now
	m.hardDropTil = now.Add(hardDropTraceDuration)
	return hardDropTraceTickCmd()
}

func (m *Model) applyMoveBuffer() {
	if m.lastMoveDir == 0 {
		return
	}
	if time.Since(m.lastMoveAt) > 140*time.Millisecond {
		return
	}
	if m.game.Move(m.lastMoveDir) && m.config.Sound {
		_ = playSound(m.sound, SoundMove)
	}
}

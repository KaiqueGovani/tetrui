package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name        string
	BorderColor lipgloss.Color
	TextColor   lipgloss.Color
	AccentColor lipgloss.Color
	PieceColors []lipgloss.Color
}

var themes = []Theme{
	{
		Name:        "Classic Tetris",
		BorderColor: lipgloss.Color("15"),
		TextColor:   lipgloss.Color("250"),
		AccentColor: lipgloss.Color("226"),
		PieceColors: []lipgloss.Color{"51", "226", "93", "46", "196", "21", "208"},
	},
	{
		Name:        "Amber Terminal",
		BorderColor: lipgloss.Color("214"),
		TextColor:   lipgloss.Color("223"),
		AccentColor: lipgloss.Color("208"),
		PieceColors: []lipgloss.Color{"220", "214", "222", "208", "215", "216", "223"},
	},
	{
		Name:        "Ocean Neon",
		BorderColor: lipgloss.Color("33"),
		TextColor:   lipgloss.Color("159"),
		AccentColor: lipgloss.Color("39"),
		PieceColors: []lipgloss.Color{"45", "39", "51", "44", "50", "75", "81"},
	},
	{
		Name:        "Forest CRT",
		BorderColor: lipgloss.Color("22"),
		TextColor:   lipgloss.Color("120"),
		AccentColor: lipgloss.Color("34"),
		PieceColors: []lipgloss.Color{"47", "64", "77", "48", "160", "35", "106"},
	},
}

func themeIndexByName(name string) int {
	for i, theme := range themes {
		if theme.Name == name {
			return i
		}
	}
	return -1
}

func viewMenu(m Model) string {
	theme := themes[m.themeIndex]
	content := renderMenu("TETRUI", menuItems, m.menuIndex, "Enter to select, Q to quit", theme)
	return center(m.width, m.height, content)
}

func viewThemes(m Model) string {
	theme := themes[m.themeIndex]
	items := make([]string, 0, len(themes))
	for _, t := range themes {
		items = append(items, t.Name)
	}
	content := renderMenu("Themes", items, m.themeIndex, "Enter to apply, Esc to back", theme)
	return center(m.width, m.height, content)
}

func viewScores(m Model) string {
	theme := themes[m.themeIndex]
	var b strings.Builder
	b.WriteString(titleStyle(theme).Render("Scores"))
	b.WriteString("\n\n")
	if len(m.scores) == 0 {
		b.WriteString("No scores yet.\n")
	} else {
		start := m.scoresOffset
		end := start + scoresPageSize
		if end > len(m.scores) {
			end = len(m.scores)
		}
		for i, score := range m.scores[start:end] {
			line := fmt.Sprintf("%2d. %-12s %7d  L%2d  %s", start+i+1, score.Name, score.Score, score.Level, score.When)
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(m.scores) > scoresPageSize {
			b.WriteString("\n")
			b.WriteString(helpStyle(theme).Render("Use Up/Down to scroll"))
			b.WriteString("\n")
		}
	}
	if m.syncWarning != "" {
		b.WriteString("\n")
		b.WriteString(warningStyle(theme).Render(m.syncWarning))
		b.WriteString("\n")
	}
	if m.syncLoading {
		b.WriteString("\n")
		b.WriteString(helpStyle(theme).Render(renderSyncLoader(m.syncDots)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle(theme).Render("Enter to back"))
	return center(m.width, m.height, b.String())
}

const scoresPageSize = 20

func viewConfig(m Model) string {
	theme := themes[m.themeIndex]
	items := make([]string, 0, len(configItems))
	for i, item := range configItems {
		state := "OFF"
		switch i {
		case 0:
			if m.config.Sound {
				state = "ON"
			}
			items = append(items, fmt.Sprintf("%s: %s", item, state))
		case 1:
			if m.config.Music {
				state = "ON"
			}
			items = append(items, fmt.Sprintf("%s: %s", item, state))
		case 2:
			items = append(items, fmt.Sprintf("%s: %d%%", item, clampVolumePercent(m.config.Volume)))
		case 3:
			if m.config.Shadow {
				state = "ON"
			}
			items = append(items, fmt.Sprintf("%s: %s", item, state))
		case 4:
			items = append(items, fmt.Sprintf("%s: %dx", item, clampScale(m.config.Scale)))
		case 5:
			if m.config.Sync {
				state = "ON"
			}
			items = append(items, fmt.Sprintf("%s: %s", item, state))
		}
	}
	content := renderMenu("Config", items, m.configIndex, "Enter to toggle, Left/Right to adjust, Esc to back", theme)
	return center(m.width, m.height, content)
}

func viewNameEntry(m Model) string {
	theme := themes[m.themeIndex]
	var b strings.Builder
	b.WriteString(titleStyle(theme).Render("Game Over"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Score: %d  Lines: %d  Level: %d\n\n", m.game.Score, m.game.Lines, m.game.Level))
	b.WriteString("Enter your name: ")
	b.WriteString(highlightStyle(theme).Render(m.nameInput))
	b.WriteString("\n\n")
	b.WriteString(helpStyle(theme).Render("Enter to save, Esc to skip"))
	return center(m.width, m.height, b.String())
}

func viewGame(m Model) string {
	theme := themes[m.themeIndex]
	scale := clampScale(m.config.Scale)
	minWidth, minHeight := minGameSize(scale)
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		message := fmt.Sprintf("Terminal too small. Need at least %dx%d. Current %dx%d.", minWidth, minHeight, m.width, m.height)
		return center(m.width, m.height, message)
	}
	board := renderBoard(m.game, theme, scale, m.config.Shadow, m.flashRows, m.flashUntil)
	info := renderInfo(m.game, theme, scale, m.lastEvent, m.lastDelta)
	if m.width >= minWidth+24 {
		return center(m.width, m.height, lipgloss.JoinHorizontal(lipgloss.Top, board, info))
	}
	return center(m.width, m.height, lipgloss.JoinVertical(lipgloss.Left, board, info))
}

func renderBoard(g Game, theme Theme, scale int, showShadow bool, flashRows []int, flashUntil time.Time) string {
	border := lipgloss.NewStyle().Foreground(theme.BorderColor)
	cellEmpty := lipgloss.NewStyle()
	cellText := strings.Repeat(" ", cellWidth(scale))
	board := make([][]int, boardHeight)
	for y := range board {
		board[y] = make([]int, boardWidth)
		copy(board[y], g.Board[y])
	}
	ghost := make([][]bool, boardHeight)
	for y := range ghost {
		ghost[y] = make([]bool, boardWidth)
	}
	ghostY := g.GhostY()
	if showShadow && ghostY != g.Y {
		for _, p := range pieceRotations[g.Current][g.Rotation] {
			bx := g.X + p.X
			by := ghostY + p.Y
			if by >= 0 && by < boardHeight && bx >= 0 && bx < boardWidth {
				if board[by][bx] == 0 {
					ghost[by][bx] = true
				}
			}
		}
	}
	for _, p := range pieceRotations[g.Current][g.Rotation] {
		bx := g.X + p.X
		by := g.Y + p.Y
		if by >= 0 && by < boardHeight && bx >= 0 && bx < boardWidth {
			board[by][bx] = g.Current + 1
		}
	}
	flashActive := !flashUntil.IsZero() && time.Now().Before(flashUntil)
	flashMap := map[int]struct{}{}
	if flashActive {
		for _, row := range flashRows {
			flashMap[row] = struct{}{}
		}
	}
	var b strings.Builder
	b.WriteString(border.Render("+" + strings.Repeat("-", boardWidth*cellWidth(scale)) + "+"))
	b.WriteString("\n")
	for y := 0; y < boardHeight; y++ {
		for repeat := 0; repeat < scale; repeat++ {
			b.WriteString(border.Render("|"))
			for x := 0; x < boardWidth; x++ {
				val := board[y][x]
				_, flashRow := flashMap[y]
				if val == 0 {
					if ghost[y][x] {
						color := theme.PieceColors[g.Current%len(theme.PieceColors)]
						ghostText := strings.Repeat(".", cellWidth(scale))
						b.WriteString(lipgloss.NewStyle().Foreground(color).Faint(true).Render(ghostText))
					} else if flashRow {
						b.WriteString(lipgloss.NewStyle().Foreground(theme.AccentColor).Render(cellText))
					} else {
						b.WriteString(cellEmpty.Render(cellText))
					}
					continue
				}
				color := theme.PieceColors[(val-1)%len(theme.PieceColors)]
				style := lipgloss.NewStyle().Background(color)
				if flashRow {
					style = style.Foreground(theme.AccentColor)
				}
				b.WriteString(style.Render(cellText))
			}
			b.WriteString(border.Render("|"))
			b.WriteString("\n")
		}
	}
	b.WriteString(border.Render("+" + strings.Repeat("-", boardWidth*cellWidth(scale)) + "+"))
	return b.String()
}

func renderInfo(g Game, theme Theme, scale int, lastEvent string, lastDelta int) string {
	var b strings.Builder
	pad := lipgloss.NewStyle().PaddingLeft(2)
	b.WriteString(pad.Render(titleStyle(theme).Render("Next")))
	b.WriteString("\n")
	b.WriteString(pad.Render(renderMiniPiece(g.Next, theme, scale)))
	b.WriteString("\n\n")
	b.WriteString(pad.Render(titleStyle(theme).Render("Hold")))
	b.WriteString("\n")
	if g.HasHold {
		b.WriteString(pad.Render(renderMiniPiece(g.HoldKind, theme, scale)))
	} else {
		b.WriteString(pad.Render("(empty)"))
	}
	b.WriteString("\n\n")
	b.WriteString(pad.Render(fmt.Sprintf("Score: %d", g.Score)))
	b.WriteString("\n")
	b.WriteString(pad.Render(fmt.Sprintf("Lines: %d", g.Lines)))
	b.WriteString("\n")
	b.WriteString(pad.Render(fmt.Sprintf("Level: %d", g.Level)))
	b.WriteString("\n\n")
	if lastEvent != "" || lastDelta > 0 {
		label := lastEvent
		if label == "" {
			label = "POINTS"
		}
		b.WriteString(pad.Render(highlightStyle(theme).Render(label)))
		b.WriteString("\n")
		b.WriteString(pad.Render(highlightStyle(theme).Render(fmt.Sprintf("+%d", lastDelta))))
		b.WriteString("\n\n")
	}
	keys := []string{
		"Arrows/HJKL: move",
		"Z/X or Up: rotate",
		"Space: hard drop",
		"C: hold",
		"P: pause",
		"Q: menu",
	}
	for _, line := range keys {
		b.WriteString(pad.Render(helpStyle(theme).Render(line)))
		b.WriteString("\n")
	}
	if g.Paused {
		b.WriteString("\n")
		b.WriteString(pad.Render(highlightStyle(theme).Render("Paused")))
	}
	return b.String()
}

func renderMiniPiece(kind int, theme Theme, scale int) string {
	grid := make([][]int, 4)
	for y := range grid {
		grid[y] = make([]int, 4)
	}
	for _, p := range pieceRotations[kind][0] {
		grid[p.Y][p.X] = 1
	}
	cellEmpty := lipgloss.NewStyle()
	cellText := strings.Repeat(" ", cellWidth(scale))
	var b strings.Builder
	for y := 0; y < 4; y++ {
		for repeat := 0; repeat < scale; repeat++ {
			for x := 0; x < 4; x++ {
				if grid[y][x] == 0 {
					b.WriteString(cellEmpty.Render(cellText))
					continue
				}
				color := theme.PieceColors[kind%len(theme.PieceColors)]
				b.WriteString(lipgloss.NewStyle().Background(color).Render(cellText))
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func minGameSize(scale int) (int, int) {
	width := boardWidth*cellWidth(scale) + 4
	height := boardHeight*scale + 4
	return width, height
}

func titleStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.AccentColor).Bold(true)
}

func highlightStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.AccentColor).Bold(true)
}

func helpStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.TextColor)
}

func warningStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
}

func center(width, height int, content string) string {
	if width == 0 || height == 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func renderSyncLoader(dots int) string {
	if dots < 0 {
		dots = 0
	}
	if dots > 3 {
		dots = dots % 4
	}
	return "Syncing" + strings.Repeat(".", dots)
}

func clampScale(value int) int {
	if value < 1 {
		return 1
	}
	if value > 3 {
		return 3
	}
	return value
}

func clampVolumePercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func cellWidth(scale int) int {
	if scale < 1 {
		scale = 1
	}
	return 2 * scale
}

func renderMenu(title string, items []string, selected int, footer string, theme Theme) string {
	maxWidth := lipgloss.Width(title)
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item)
		if width := lipgloss.Width(item); width > maxWidth {
			maxWidth = width
		}
	}
	if width := lipgloss.Width(footer); width > maxWidth {
		maxWidth = width
	}
	lineStyle := lipgloss.NewStyle().Width(maxWidth).Align(lipgloss.Center)
	var b strings.Builder
	b.WriteString(lineStyle.Render(titleStyle(theme).Render(title)))
	b.WriteString("\n\n")
	for i, line := range lines {
		if i == selected {
			b.WriteString(lineStyle.Render(highlightStyle(theme).Render(line)))
			b.WriteString("\n")
			continue
		}
		b.WriteString(lineStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(lineStyle.Render(helpStyle(theme).Render(footer)))
	return b.String()
}

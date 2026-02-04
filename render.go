package main

import (
	"fmt"
	"strings"

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
	var b strings.Builder
	b.WriteString(titleStyle(theme).Render("TETRUI"))
	b.WriteString("\n\n")
	for i, item := range menuItems {
		cursor := "  "
		if i == m.menuIndex {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s", cursor, item)
		if i == m.menuIndex {
			line = highlightStyle(theme).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle(theme).Render("Enter to select, Q to quit"))
	return center(m.width, m.height, b.String())
}

func viewThemes(m Model) string {
	theme := themes[m.themeIndex]
	var b strings.Builder
	b.WriteString(titleStyle(theme).Render("Themes"))
	b.WriteString("\n\n")
	for i, t := range themes {
		cursor := "  "
		if i == m.themeIndex {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s", cursor, t.Name)
		if i == m.themeIndex {
			line = highlightStyle(theme).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle(theme).Render("Enter to apply, Esc to back"))
	return center(m.width, m.height, b.String())
}

func viewScores(m Model) string {
	theme := themes[m.themeIndex]
	var b strings.Builder
	b.WriteString(titleStyle(theme).Render("Scores"))
	b.WriteString("\n\n")
	if len(m.scores) == 0 {
		b.WriteString("No scores yet.\n")
	} else {
		for i, score := range m.scores {
			line := fmt.Sprintf("%2d. %-12s %7d  L%2d  %s", i+1, score.Name, score.Score, score.Level, score.When)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle(theme).Render("Enter to back"))
	return center(m.width, m.height, b.String())
}

func viewConfig(m Model) string {
	theme := themes[m.themeIndex]
	var b strings.Builder
	b.WriteString(titleStyle(theme).Render("Config"))
	b.WriteString("\n\n")
	for i, item := range configItems {
		cursor := "  "
		if i == m.configIndex {
			cursor = "> "
		}
		state := "OFF"
		if i == 0 && m.config.Sound {
			state = "ON"
		}
		line := fmt.Sprintf("%s%s: %s", cursor, item, state)
		if i == m.configIndex {
			line = highlightStyle(theme).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle(theme).Render("Enter to toggle, Esc to back"))
	return center(m.width, m.height, b.String())
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
	minWidth, minHeight := minGameSize()
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		message := fmt.Sprintf("Terminal too small. Need at least %dx%d. Current %dx%d.", minWidth, minHeight, m.width, m.height)
		return center(m.width, m.height, message)
	}
	board := renderBoard(m.game, theme)
	info := renderInfo(m.game, theme)
	if m.width >= minWidth+24 {
		return center(m.width, m.height, lipgloss.JoinHorizontal(lipgloss.Top, board, info))
	}
	return center(m.width, m.height, lipgloss.JoinVertical(lipgloss.Left, board, info))
}

func renderBoard(g Game, theme Theme) string {
	border := lipgloss.NewStyle().Foreground(theme.BorderColor)
	cellEmpty := lipgloss.NewStyle()
	cellText := "  "
	board := make([][]int, boardHeight)
	for y := range board {
		board[y] = make([]int, boardWidth)
		copy(board[y], g.Board[y])
	}
	for _, p := range pieceRotations[g.Current][g.Rotation] {
		bx := g.X + p.X
		by := g.Y + p.Y
		if by >= 0 && by < boardHeight && bx >= 0 && bx < boardWidth {
			board[by][bx] = g.Current + 1
		}
	}
	var b strings.Builder
	b.WriteString(border.Render("+" + strings.Repeat("-", boardWidth*2) + "+"))
	b.WriteString("\n")
	for y := 0; y < boardHeight; y++ {
		b.WriteString(border.Render("|"))
		for x := 0; x < boardWidth; x++ {
			val := board[y][x]
			if val == 0 {
				b.WriteString(cellEmpty.Render(cellText))
				continue
			}
			color := theme.PieceColors[(val-1)%len(theme.PieceColors)]
			b.WriteString(lipgloss.NewStyle().Background(color).Render(cellText))
		}
		b.WriteString(border.Render("|"))
		b.WriteString("\n")
	}
	b.WriteString(border.Render("+" + strings.Repeat("-", boardWidth*2) + "+"))
	return b.String()
}

func renderInfo(g Game, theme Theme) string {
	var b strings.Builder
	pad := lipgloss.NewStyle().PaddingLeft(2)
	b.WriteString(pad.Render(titleStyle(theme).Render("Next")))
	b.WriteString("\n")
	b.WriteString(pad.Render(renderMiniPiece(g.Next, theme)))
	b.WriteString("\n\n")
	b.WriteString(pad.Render(titleStyle(theme).Render("Hold")))
	b.WriteString("\n")
	if g.HasHold {
		b.WriteString(pad.Render(renderMiniPiece(g.HoldKind, theme)))
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

func renderMiniPiece(kind int, theme Theme) string {
	grid := make([][]int, 4)
	for y := range grid {
		grid[y] = make([]int, 4)
	}
	for _, p := range pieceRotations[kind][0] {
		grid[p.Y][p.X] = 1
	}
	cellEmpty := lipgloss.NewStyle()
	var b strings.Builder
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if grid[y][x] == 0 {
				b.WriteString(cellEmpty.Render("  "))
				continue
			}
			color := theme.PieceColors[kind%len(theme.PieceColors)]
			b.WriteString(lipgloss.NewStyle().Background(color).Render("  "))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func minGameSize() (int, int) {
	return boardWidth*2 + 4, boardHeight + 4
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

func center(width, height int, content string) string {
	if width == 0 || height == 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

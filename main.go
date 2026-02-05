package main

import (
	"flag"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()
	EnableDebugLogging(*debug)
	DebugLogf("tetrui start debug=%v", *debug)
	program := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		DebugLogf("program error: %v", err)
		os.Exit(1)
	}
}

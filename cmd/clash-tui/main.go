package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"clash-tui/internal/config"
	"clash-tui/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	m := ui.NewModel(cfg)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("run tui failed: %v", err)
	}
}

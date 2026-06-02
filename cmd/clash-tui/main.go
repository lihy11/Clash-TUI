package main

import (
	"context"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clash-tui/internal/config"
	"clash-tui/internal/runtime"
	"clash-tui/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	rt, err := runtime.New()
	if err != nil {
		log.Fatalf("init runtime failed: %v", err)
	}
	// Core download can be very slow on some networks; keep a generous startup timeout.
	bootCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	if err := rt.Boot(bootCtx, &cfg); err != nil {
		log.Fatalf("boot runtime failed: %v", err)
	}

	m := ui.NewModel(cfg, rt)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("run tui failed: %v", err)
	}
}

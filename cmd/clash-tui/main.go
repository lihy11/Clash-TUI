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
	log.Printf("preparing runtime (may download mihomo core on first run)...")
	bootCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	if err := rt.Boot(bootCtx, &cfg); err != nil {
		log.Printf("boot runtime warning: %v", err)
		log.Printf("continue launching TUI; you can reconfigure in Settings/Profiles")
	} else {
		log.Printf("runtime ready")
	}
	defer rt.Close()

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

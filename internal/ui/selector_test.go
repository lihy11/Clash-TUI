package ui

import (
	"testing"

	"clash-tui/internal/config"
)

func TestOpenSelectorThemeSetsState(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.themeIndex = 1

	m.openSelector("theme")

	if !m.selectorOpen {
		t.Fatalf("expected selector open")
	}
	if m.selectorKind != "theme" {
		t.Fatalf("expected selector kind theme, got %q", m.selectorKind)
	}
	if got := m.selectorLV.Index(); got != 1 {
		t.Fatalf("expected selected index 1, got %d", got)
	}
}

func TestApplySelectorChoiceThemeClosesAndUpdatesTheme(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.themeIndex = 1
	m.openSelector("theme")
	m.selectorLV.Select(0)

	cmd := m.applySelectorChoice()

	if cmd != nil {
		t.Fatalf("expected nil cmd for theme selection")
	}
	if m.selectorOpen {
		t.Fatalf("expected selector closed after apply")
	}
	if m.themeIndex != 0 {
		t.Fatalf("expected theme index 0, got %d", m.themeIndex)
	}
}

func TestHandleSelectorMouseOutsideCloses(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.width = 120
	m.height = 40
	m.openSelector("theme")
	_ = m.renderSelectorOverlay(40, 12)

	handled, cmd := m.handleSelectorMouse(0, 0)

	if !handled {
		t.Fatalf("expected outside click handled")
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd for outside click")
	}
	if m.selectorOpen {
		t.Fatalf("expected selector closed after outside click")
	}
}

func TestHandleSelectorMouseSelectsItem(t *testing.T) {
	m := NewModel(config.Default(), nil)
	m.width = 120
	m.height = 40
	m.themeIndex = 1
	m.openSelector("theme")
	_ = m.renderSelectorOverlay(40, 12)
	if len(m.selectorItemTargets) == 0 {
		t.Fatalf("expected selector item targets")
	}
	target := m.selectorItemTargets[0]
	x := target.x1
	y := target.y1

	handled, cmd := m.handleSelectorMouse(x, y)

	if !handled {
		t.Fatalf("expected item click handled")
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd for theme selection by mouse")
	}
	if m.selectorOpen {
		t.Fatalf("expected selector closed after item click")
	}
	if m.themeIndex != 0 {
		t.Fatalf("expected theme index 0, got %d", m.themeIndex)
	}
}

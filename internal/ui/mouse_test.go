package ui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"clash-tui/internal/config"
)

func newTestModel(t *testing.T) *model {
	t.Helper()
	// Isolate config writes from toggleLanguage().
	tmp := t.TempDir()
	prev := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			_ = os.Setenv("XDG_CONFIG_HOME", prev)
		}
	})
	return NewModel(config.Default(), nil)
}

func TestProxyWheelScrollDoesNotChangeSelection(t *testing.T) {
	m := newTestModel(t)
	m.tab = 1
	m.networkTab = 0
	m.nodes = []string{"n0", "n1", "n2", "n3"}
	m.nodeCursor = 2
	m.nodeOffset = 1
	m.nodePageSize = 2

	_ = m.handleMouseMsg(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	if m.nodeCursor != 2 {
		t.Fatalf("wheel changed selection cursor: got=%d want=%d", m.nodeCursor, 2)
	}
	if m.nodeOffset != 2 {
		t.Fatalf("wheel should scroll offset: got=%d want=%d", m.nodeOffset, 2)
	}

	_ = m.handleMouseMsg(tea.MouseMsg{
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	if m.nodeCursor != 2 {
		t.Fatalf("wheel up changed selection cursor: got=%d want=%d", m.nodeCursor, 2)
	}
	if m.nodeOffset != 1 {
		t.Fatalf("wheel up should scroll offset back: got=%d want=%d", m.nodeOffset, 1)
	}
}

func TestProxyNodeClickSelects(t *testing.T) {
	m := newTestModel(t)
	m.tab = 1
	m.networkTab = 0
	m.groups = []string{"g"}
	m.groupCursor = 0
	m.nodes = []string{"n0", "n1", "n2"}
	m.nodeCursor = 0
	m.proxyNodeTargets = []clickTarget{
		{x1: 0, y1: 0, x2: 10, y2: 1, idx: 1},
	}

	cmd := m.handleMouseMsg(tea.MouseMsg{
		X:      1,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.nodeCursor != 1 {
		t.Fatalf("click should select target row: got=%d want=%d", m.nodeCursor, 1)
	}
	if cmd == nil {
		t.Fatalf("clicking node should return switch cmd")
	}
}

func TestNonLeftPressDoesNotTriggerNodeSelection(t *testing.T) {
	m := newTestModel(t)
	m.tab = 1
	m.networkTab = 0
	m.groups = []string{"g"}
	m.groupCursor = 0
	m.nodes = []string{"n0", "n1", "n2"}
	m.nodeCursor = 0
	m.proxyNodeTargets = []clickTarget{
		{x1: 0, y1: 0, x2: 10, y2: 1, idx: 2},
	}

	_ = m.handleMouseMsg(tea.MouseMsg{
		X:      1,
		Y:      0,
		Button: tea.MouseButtonNone,
		Action: tea.MouseActionPress,
	})
	if m.nodeCursor != 0 {
		t.Fatalf("non-left press should not select node: got=%d want=%d", m.nodeCursor, 0)
	}
}

func TestLanguageChipClickTogglesLanguage(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Language = "en"
	m.langChipTarget = clickTarget{x1: 0, y1: 0, x2: 20, y2: 1}

	_ = m.handleMouseMsg(tea.MouseMsg{
		X:      1,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.cfg.Language != "zh-CN" {
		t.Fatalf("language should toggle to zh-CN on chip click, got=%s", m.cfg.Language)
	}
}

package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseRouterDispatchesHighestPriorityAction(t *testing.T) {
	r := mouseRouter{}
	r.register(mouseAction{ID: "page.tab", Box: hitBox{x1: 0, y1: 0, x2: 10, y2: 1}})
	r.register(mouseAction{ID: "overlay.close", Box: hitBox{x1: 2, y1: 0, x2: 6, y2: 1}})

	got, ok := r.dispatch(tea.MouseMsg{X: 3, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if !ok {
		t.Fatalf("expected dispatch hit")
	}
	if got.ID != "overlay.close" {
		t.Fatalf("expected overlay.close, got %q", got.ID)
	}
}

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestComposeHeaderLineWidthBounded(t *testing.T) {
	cases := []struct {
		name  string
		left  string
		right string
		w     int
	}{
		{
			name:  "ascii",
			left:  "Clash-TUI",
			right: "status connected",
			w:     40,
		},
		{
			name:  "mixed_symbols",
			left:  "Clash-TUI",
			right: "up 0.0 B/s down 0.0 B/s | core 127.0.0.1:9090 | theme Gruvbox",
			w:     72,
		},
		{
			name:  "cjk_text",
			left:  "状态",
			right: "连接正常 | 核心 127.0.0.1:9090",
			w:     36,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := composeHeaderLine(tc.left, tc.right, tc.w)
			if width := lipgloss.Width(got); width > tc.w {
				t.Fatalf("line width overflow: got=%d want<=%d line=%q", width, tc.w, got)
			}
		})
	}
}

func TestComposeHeaderLinePreservesBothSidesWhenSpaceAllows(t *testing.T) {
	left := "left"
	right := "right"
	got := composeHeaderLine(left, right, 24)
	if !strings.Contains(got, left) {
		t.Fatalf("missing left content: %q", got)
	}
	if !strings.Contains(got, right) {
		t.Fatalf("missing right content: %q", got)
	}
}

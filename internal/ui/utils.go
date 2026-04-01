package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func formatRate(v float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	u := 0
	for v >= 1024 && u < len(units)-1 {
		v /= 1024
		u++
	}
	if u == 0 {
		return fmt.Sprintf("%.0f %s", v, units[u])
	}
	return fmt.Sprintf("%.1f %s", v, units[u])
}

func formatRateFixed(v float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	u := 0
	for v >= 1024 && u < len(units)-1 {
		v /= 1024
		u++
	}
	if v > 9999 {
		v = 9999
	}
	return fmt.Sprintf("%6.1f %-4s", v, units[u])
}

func truncateByWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	ellipsis := "…"
	maxW := w - lipgloss.Width(ellipsis)
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if cur+rw > maxW {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	return b.String() + ellipsis
}

func fitTextWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	out := truncateByWidth(s, w)
	cur := lipgloss.Width(out)
	if cur < w {
		out += strings.Repeat(" ", w-cur)
	}
	return out
}

func composeHeaderLine(left, right string, w int) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if w <= 0 {
		return ""
	}
	gap := 2
	leftW := lipgloss.Width(left)
	if left == "" {
		gap = 0
	}
	rightW := max(0, w-leftW-gap)
	if leftW >= w {
		return truncateByWidth(left, w)
	}
	right = truncateByWidth(right, rightW)
	if left == "" {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(right)
	}
	return left + strings.Repeat(" ", gap) + lipgloss.NewStyle().Width(rightW).Align(lipgloss.Right).Render(right)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

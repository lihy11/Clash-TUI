package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	app         lipgloss.Style
	header      lipgloss.Style
	subtle      lipgloss.Style
	errorText   lipgloss.Style
	successText lipgloss.Style
	tab         lipgloss.Style
	tabActive   lipgloss.Style
	panel       lipgloss.Style
	panelTitle  lipgloss.Style
	cursor      lipgloss.Style
	selected    lipgloss.Style
	footer      lipgloss.Style
}

func defaultStyles() styles {
	border := lipgloss.NormalBorder()
	return styles{
		app: lipgloss.NewStyle(),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		subtle: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#606060", Dark: "#909090"}),
		errorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")),
		successText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("84")),
		tab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.AdaptiveColor{Light: "#444", Dark: "#bbb"}),
		tabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("31")),
		panel: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#bbbbbb", Dark: "#555555"}).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81")),
		cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")),
		selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("84")).
			Bold(true),
		footer: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666", Dark: "#999"}),
	}
}

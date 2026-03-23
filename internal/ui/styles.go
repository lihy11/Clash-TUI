package ui

import "github.com/charmbracelet/lipgloss"

type theme struct {
	Name       string
	Primary    string
	Secondary  string
	Accent     string
	Muted      string
	Surface    string
	Panel      string
	Error      string
	Success    string
	Warning    string
	PillText   string
	CursorText string
}

var themes = []theme{
	{
		Name:       "Dracula",
		Primary:    "#bd93f9", // Purple
		Secondary:  "#8be9fd", // Cyan
		Accent:     "#ff79c6", // Pink
		Muted:      "#6272a4", // Comment
		Surface:    "#282a36", // Background
		Panel:      "#44475a", // Current Line
		Error:      "#ff5555", // Red
		Success:    "#50fa7b", // Green
		Warning:    "#f1fa8c", // Yellow
		PillText:   "#f8f8f2", // Foreground
		CursorText: "#282a36", // Background for contrast
	},
	{
		Name:       "Gruvbox",
		Primary:    "#fabd2f", // Yellow
		Secondary:  "#8ec07c", // Aqua
		Accent:     "#fe8019", // Orange
		Muted:      "#928374", // Gray
		Surface:    "#282828", // Bg0
		Panel:      "#504945", // Bg2
		Error:      "#fb4934", // Red
		Success:    "#b8bb26", // Green
		Warning:    "#fabd2f", // Yellow
		PillText:   "#ebdbb2", // Fg
		CursorText: "#282828", // Bg0
	},
	{
		Name:       "One Dark",
		Primary:    "#61afef", // Blue
		Secondary:  "#56b6c2", // Cyan
		Accent:     "#c678dd", // Purple
		Muted:      "#5c6370", // Comment
		Surface:    "#282c34", // Background
		Panel:      "#3e4451", // Line Highlight
		Error:      "#e06c75", // Red
		Success:    "#98c379", // Green
		Warning:    "#e5c07b", // Yellow
		PillText:   "#abb2bf", // Foreground
		CursorText: "#282c34", // Background
	},
	{
		Name:       "Geist UI",
		Primary:    "#ededed", // White
		Secondary:  "#a0a0a0", // Gray
		Accent:     "#0070f3", // Blue
		Muted:      "#888888", // Muted
		Surface:    "#000000", // Black
		Panel:      "#333333", // Dark Gray
		Error:      "#ff0000", // Red
		Success:    "#17c964", // Green
		Warning:    "#f5a623", // Orange
		PillText:   "#ffffff", // White
		CursorText: "#ffffff", // White on Accent
	},
	{
		Name:       "Catppuccin Mocha",
		Primary:    "#cba6f7", // Mauve
		Secondary:  "#89b4fa", // Blue
		Accent:     "#f38ba8", // Red as an alternative accent
		Muted:      "#6c7086", // Overlay0
		Surface:    "#1e1e2e", // Base
		Panel:      "#313244", // Surface0
		Error:      "#f38ba8", // Red
		Success:    "#a6e3a1", // Green
		Warning:    "#f9e2af", // Yellow
		PillText:   "#cdd6f4", // Text
		CursorText: "#11111b", // Crust
	},
}

type styles struct {
	app          lipgloss.Style
	header       lipgloss.Style
	statusBar    lipgloss.Style
	mainTabBar   lipgloss.Style
	subTabBar    lipgloss.Style
	sectionLabel lipgloss.Style
	subtle       lipgloss.Style
	errorText    lipgloss.Style
	successText  lipgloss.Style
	tab          lipgloss.Style
	tabActive    lipgloss.Style
	panel        lipgloss.Style
	panelTitle   lipgloss.Style
	cursor       lipgloss.Style
	selected     lipgloss.Style
	footer       lipgloss.Style
	pill         lipgloss.Style
	pillActive   lipgloss.Style
	subTab       lipgloss.Style
	subTabActive lipgloss.Style
	action       lipgloss.Style
	overlay      lipgloss.Style
	notifTitle   lipgloss.Style
}

func defaultStyles(themeIdx int) styles {
	if themeIdx < 0 || themeIdx >= len(themes) {
		themeIdx = 0
	}
	t := themes[themeIdx]
	return styles{
		app: lipgloss.NewStyle(),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),
		statusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Surface)).
			Padding(0, 1).
			Bold(true),
		mainTabBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Panel)),
		subTabBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)).
			Background(lipgloss.Color(t.Surface)),
		sectionLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),
		subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)),
		errorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)),
		successText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Success)),
		tab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(t.PillText)),
		tabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Underline(true).
			Foreground(lipgloss.Color(t.Secondary)),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Panel)).
			Background(lipgloss.Color(t.Surface)).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),
		cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.CursorText)).
			Background(lipgloss.Color(t.Accent)),
		selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Success)).
			Bold(true),
		footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)),
		pill: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Panel)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Panel)),
		pillActive: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Accent)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Accent)).
			Bold(true),
		subTab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(t.PillText)),
		subTabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Primary)),
		action: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.CursorText)).
			Background(lipgloss.Color(t.Primary)).
			Bold(true).
			Padding(0, 1),
		overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Accent)).
			Background(lipgloss.Color("235")).
			Padding(1, 2),
		notifTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),
	}
}

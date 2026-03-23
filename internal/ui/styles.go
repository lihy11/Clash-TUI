package ui

import "github.com/charmbracelet/lipgloss"

type theme struct {
	Name       string
	Primary    string
	Accent     string
	Muted      string
	Panel      string
	Error      string
	Success    string
	PillText   string
	CursorText string
}

var themes = []theme{
	{Name: "Classic", Primary: "86", Accent: "31", Muted: "244", Panel: "60", Error: "203", Success: "84", PillText: "230", CursorText: "229"},
	{Name: "Nord", Primary: "110", Accent: "67", Muted: "245", Panel: "59", Error: "203", Success: "114", PillText: "230", CursorText: "231"},
	{Name: "Solarized", Primary: "136", Accent: "33", Muted: "244", Panel: "240", Error: "160", Success: "64", PillText: "230", CursorText: "231"},
	{Name: "Mono", Primary: "252", Accent: "238", Muted: "245", Panel: "241", Error: "160", Success: "70", PillText: "255", CursorText: "231"},
}

type styles struct {
	app          lipgloss.Style
	header       lipgloss.Style
	statusBar    lipgloss.Style
	mainTabBar   lipgloss.Style
	subTabBar    lipgloss.Style
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
	border := lipgloss.NormalBorder()
	return styles{
		app: lipgloss.NewStyle(),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),
		statusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Accent)).
			Bold(true),
		mainTabBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Panel)),
		subTabBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)).
			Background(lipgloss.Color("236")),
		subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)),
		errorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)),
		successText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Success)),
		tab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("252")),
		tabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Accent)),
		panel: lipgloss.NewStyle().
			Border(border).
			BorderForeground(lipgloss.Color(t.Panel)).
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
			Background(lipgloss.Color(t.Panel)),
		pillActive: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Accent)).
			Bold(true),
		subTab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("250")),
		subTabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color(t.Accent)).
			Underline(true),
		action: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PillText)).
			Background(lipgloss.Color(t.Accent)).
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

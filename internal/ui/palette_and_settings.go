package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clash-tui/internal/config"
	"clash-tui/internal/mihomo"
)

func (m *model) applySettings() tea.Cmd {
	pollSec, err := strconv.Atoi(strings.TrimSpace(m.settingsInputs[2].Value()))
	if err != nil || pollSec < 1 {
		m.lastErr = fmt.Errorf("poll interval must be integer >=1")
		return nil
	}

	m.cfg.Endpoint = strings.TrimSpace(m.settingsInputs[0].Value())
	m.cfg.Secret = strings.TrimSpace(m.settingsInputs[1].Value())
	m.cfg.PollInterval = time.Duration(pollSec) * time.Second
	m.cfg.LogLevel = strings.TrimSpace(m.settingsInputs[3].Value())
	if m.cfg.LogLevel == "" {
		m.cfg.LogLevel = "info"
	}

	if err := config.Save(m.cfg); err != nil {
		m.lastErr = err
		m.setStatus("save config failed")
		return nil
	}

	if m.client != nil {
		m.client.Close()
	}
	c, err := mihomo.NewClient(m.cfg.Endpoint, m.cfg.Secret)
	if err != nil {
		m.lastErr = err
		m.setStatus("new client failed")
		return nil
	}
	m.client = c
	m.setStatus("settings saved")

	return tea.Batch(
		fetchVersionCmd(m.client),
		fetchConfigCmd(m.client),
		fetchProxiesCmd(m.client),
		fetchRulesCmd(m.client),
		fetchConnectionsCmd(m.client),
		fetchProvidersCmd(m.client),
		loadSubsCmd(m.rt),
		reloadCoreCmd(m.rt, m.cfg),
		listenLogCmd(m.client, m.cfg.LogLevel),
	)
}

func (m *model) cycleTheme() {
	m.themeIndex = (m.themeIndex + 1) % len(themes)
	m.styles = defaultStyles(m.themeIndex)
	m.pushNote("info", "theme switched to "+themes[m.themeIndex].Name)
}

func (m *model) paletteActions() []paletteAction {
	return []paletteAction{
		{ID: "goto_dashboard", Title: "Go: Dashboard"},
		{ID: "goto_network", Title: "Go: Network"},
		{ID: "goto_profiles", Title: "Go: Profiles"},
		{ID: "goto_system", Title: "Go: System"},
		{ID: "network_proxies", Title: "Network: Proxies"},
		{ID: "network_rules", Title: "Network: Rules"},
		{ID: "network_connections", Title: "Network: Connections"},
		{ID: "system_logs", Title: "System: Logs"},
		{ID: "system_settings", Title: "System: Settings"},
		{ID: "mode_rule", Title: "Set Mode: Rule"},
		{ID: "mode_global", Title: "Set Mode: Global"},
		{ID: "mode_direct", Title: "Set Mode: Direct"},
		{ID: "toggle_system_proxy", Title: "Toggle: System Proxy"},
		{ID: "toggle_tun", Title: "Toggle: TUN Mode"},
		{ID: "test_all", Title: "Proxies: Test All Nodes"},
		{ID: "update_all_subs", Title: "Profiles: Update All Subscriptions"},
		{ID: "theme_cycle", Title: "Theme: Cycle"},
	}
}

func (m *model) filteredPaletteActions() []paletteAction {
	all := m.paletteActions()
	q := strings.ToLower(strings.TrimSpace(m.paletteInput.Value()))
	if q == "" {
		return all
	}
	out := make([]paletteAction, 0, len(all))
	for _, a := range all {
		if strings.Contains(strings.ToLower(a.Title), q) {
			out = append(out, a)
		}
	}
	return out
}

func (m *model) runPaletteAction(id string) tea.Cmd {
	switch id {
	case "goto_dashboard":
		m.tab = 0
	case "goto_network":
		m.tab = 1
	case "goto_profiles":
		m.tab = 2
	case "goto_system":
		m.tab = 3
	case "network_proxies":
		m.tab = 1
		m.networkTab = 0
	case "network_rules":
		m.tab = 1
		m.networkTab = 1
	case "network_connections":
		m.tab = 1
		m.networkTab = 2
	case "system_logs":
		m.tab = 3
		m.systemTab = 0
	case "system_settings":
		m.tab = 3
		m.systemTab = 1
	case "mode_rule":
		return tea.Batch(setModeCmd(m.client, "rule"), fetchConfigCmd(m.client))
	case "mode_global":
		return tea.Batch(setModeCmd(m.client, "global"), fetchConfigCmd(m.client))
	case "mode_direct":
		return tea.Batch(setModeCmd(m.client, "direct"), fetchConfigCmd(m.client))
	case "toggle_system_proxy":
		return setSystemProxyCmd(m.client, !m.systemProxyEnabled())
	case "toggle_tun":
		return setTunCmd(m.client, !m.baseCfg.Tun.Enable)
	case "test_all":
		if len(m.nodes) > 0 {
			return testAllNodesCmd(m.client, m.nodes)
		}
	case "update_all_subs":
		if len(m.subscriptions) > 0 {
			cmds := make([]tea.Cmd, 0, len(m.subscriptions))
			for _, it := range m.subscriptions {
				cmds = append(cmds, updateProviderCmd2(m.client, it.ProviderName))
			}
			return tea.Batch(cmds...)
		}
	case "theme_cycle":
		m.cycleTheme()
	}
	return nil
}

func (m *model) renderPalette(w, h int) string {
	actions := m.filteredPaletteActions()
	lines := []string{
		m.styles.panelTitle.Render("Command Palette"),
		m.paletteInput.View(),
		"",
	}
	maxRows := max(1, h-5)
	for i, a := range actions {
		if i >= maxRows {
			break
		}
		line := a.Title
		if i == m.paletteCursor {
			line = m.styles.cursor.Render(line)
		}
		lines = append(lines, line)
	}
	if len(actions) == 0 {
		lines = append(lines, m.styles.subtle.Render("No matched actions"))
	}
	return m.styles.overlay.Width(w).Render(strings.Join(lines, "\n"))
}

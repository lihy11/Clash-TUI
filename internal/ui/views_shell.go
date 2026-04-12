package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderHeader(w int) string {
	title := "Clash-TUI"
	mode := strings.ToUpper(strings.TrimSpace(m.baseCfg.Mode))
	if mode == "" {
		mode = "RULE"
	}
	controller := strings.TrimPrefix(strings.TrimPrefix(m.cfg.Endpoint, "http://"), "https://")
	if controller == "" {
		controller = "127.0.0.1:9090"
	}
	conn := "CONNECTED"
	if m.lastErr != nil {
		conn = "DEGRADED"
	}
	connIcon := "●"
	if m.lastErr != nil {
		connIcon = "○"
	}
	up := formatRateFixed(m.txRate)
	down := formatRateFixed(m.rxRate)
	chip := m.renderLanguageChip()
	right1Prefix := fmt.Sprintf("%s %s | %s %s | ", connIcon, conn, m.t("mode", "模式"), mode)
	right1 := right1Prefix + chip
	right2 := fmt.Sprintf("%s %s %s %s | %s %s | %s %s",
		m.t("up", "上行"), up, m.t("down", "下行"), down,
		m.t("core", "内核"), controller,
		m.t("theme", "主题"), themes[m.themeIndex].Name)
	totalW := max(1, w)
	innerW := max(1, totalW-m.styles.statusBar.GetHorizontalFrameSize())
	lineW := max(1, innerW-2)
	line1 := composeHeaderLine(title, right1, lineW)
	line2 := composeHeaderLine("", right2, lineW)
	top := m.styles.statusBar.Width(totalW).MaxWidth(totalW).Render(line1)
	bottom := m.styles.statusBar.Width(totalW).MaxWidth(totalW).Render(line2)

	leftW := lipgloss.Width(strings.TrimSpace(title))
	gap := 2
	rightW := max(0, lineW-leftW-gap)
	if lw := lipgloss.Width(right1); lw <= rightW {
		statusPadLeft := 2
		chipX := statusPadLeft + leftW + gap + (rightW - lw) + lipgloss.Width(right1Prefix)
		m.mouse.register(mouseAction{
			ID:  "header.language.toggle",
			Box: hitBox{x1: chipX, y1: 0, x2: chipX + lipgloss.Width(chip), y2: 1},
		})
	}
	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m *model) renderTabs(w, y int) string {
	out := make([]string, 0, len(tabs))
	cursorX := 0
	for i := range tabs {
		icon := ""
		if i < len(tabIcons) {
			icon = tabIcons[i] + " "
		}
		label := fmt.Sprintf("%s%s", icon, m.mainTabLabel(i))
		var token string
		if i == m.tab {
			token = m.styles.tabActive.Render(" " + label + " ")
		} else {
			token = m.styles.tab.Render(" " + label + " ")
		}
		wToken := lipgloss.Width(token)
		m.mouse.register(mouseAction{
			ID:    "tab.main.select",
			Index: i,
			Box:   hitBox{x1: cursorX, y1: y, x2: cursorX + wToken, y2: y + 1},
		})
		cursorX += wToken
		out = append(out, token)
	}
	line := strings.Join(out, "")
	return m.styles.mainTabBar.Width(w).MaxWidth(w).Render(line)
}

func (m *model) renderNetwork(w, h int) string {
	items := make([]string, 0, len(networkTabs))
	for i := range networkTabs {
		items = append(items, m.networkTabLabel(i))
	}
	sub := m.renderSubTabs(m.t("NETWORK", "网络"), items, m.networkTab, m.bodyY)
	subW := max(1, w)
	sub = m.styles.subTabBar.Width(subW).MaxWidth(subW).Render(sub)
	bodyH := max(4, h-lipgloss.Height(sub))
	var body string
	switch m.networkTab {
	case 0:
		body = m.renderProxies(w, bodyH)
	case 1:
		body = m.renderRules(w, bodyH)
	default:
		body = m.renderConnections(w, bodyH)
	}
	return lipgloss.JoinVertical(lipgloss.Left, sub, body)
}

func (m *model) renderSystem(w, h int) string {
	items := make([]string, 0, len(systemTabs))
	for i := range systemTabs {
		items = append(items, m.systemTabLabel(i))
	}
	sub := m.renderSubTabs(m.t("SYSTEM", "系统"), items, m.systemTab, m.bodyY)
	subW := max(1, w)
	sub = m.styles.subTabBar.Width(subW).MaxWidth(subW).Render(sub)
	bodyH := max(4, h-lipgloss.Height(sub))
	if m.systemTab == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, sub, m.renderLogs(w, bodyH))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sub, m.renderSettings(w, bodyH))
}

func (m *model) renderSubTabs(section string, items []string, active, y int) string {
	out := make([]string, 0, len(items)+2)
	sectionToken := m.styles.sectionLabel.Render(section)
	out = append(out, sectionToken, m.styles.subtle.Render("│"))
	cursorX := m.styles.subTabBar.GetHorizontalFrameSize()/2 + lipgloss.Width(sectionToken) + 1
	for i, t := range items {
		var token string
		if i == active {
			token = m.styles.subTabActive.Render(t)
		} else {
			token = m.styles.subTab.Render(t)
		}
		wToken := lipgloss.Width(token)
		actionID := "tab.network.select"
		if section == m.t("SYSTEM", "系统") {
			actionID = "tab.system.select"
		}
		m.mouse.register(mouseAction{
			ID:    actionID,
			Index: i,
			Box:   hitBox{x1: cursorX, y1: y, x2: cursorX + wToken, y2: y + 1},
		})
		cursorX += wToken
		out = append(out, token)
	}
	return strings.Join(out, "")
}

func (m *model) renderBody(bodyW, bodyH int) string {
	if m.loading {
		return m.renderPanel(bodyW, bodyH, m.t("Loading Mihomo data...", "正在加载 Mihomo 数据..."))
	}
	switch m.tab {
	case 0:
		return m.renderOverview(bodyW, bodyH)
	case 1:
		return m.renderNetwork(bodyW, bodyH)
	case 2:
		return m.renderProfiles(bodyW, bodyH)
	case 3:
		return m.renderSystem(bodyW, bodyH)
	default:
		return ""
	}
}

func (m *model) renderFooter(w int) string {
	binds := m.footerBindings()
	msg := strings.TrimSpace(m.status)
	if msg == "" {
		msg = "ready"
	}
	totalW := max(1, w)
	innerW := max(1, totalW-m.styles.footer.GetHorizontalFrameSize())
	lineW := max(1, innerW-2)
	m.help.Width = lineW
	helpLine := m.help.View(footerKeys{items: binds})
	line := composeHeaderLine(helpLine, m.t("status ", "状态 ")+msg, lineW)
	line = m.styles.footer.Width(totalW).MaxWidth(totalW).Render(line)
	return lipgloss.NewStyle().Width(totalW).MaxWidth(totalW).Render(line)
}

func (m *model) footerBindings() []key.Binding {
	switch m.tab {
	case 0:
		return []key.Binding{
			key.NewBinding(key.WithKeys("r", "g", "d"), key.WithHelp("r/g/d", m.t("mode", "模式"))),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", m.t("sys-proxy", "系统代理"))),
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "TUN")),
			key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", m.t("palette", "面板"))),
		}
	case 1:
		if m.networkTab == 2 {
			return []key.Binding{
				key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/j k", m.t("move", "移动"))),
				key.NewBinding(key.WithKeys("x"), key.WithHelp("x", m.t("close", "关闭"))),
				key.NewBinding(key.WithKeys("X"), key.WithHelp("X", m.t("close all", "全部关闭"))),
			}
		}
		if m.networkTab == 0 {
			return []key.Binding{
				key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/j k", m.t("move", "移动"))),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", m.t("select", "选择"))),
				key.NewBinding(key.WithKeys("t", "T"), key.WithHelp("t/T", m.t("delay test", "延迟测试"))),
				key.NewBinding(key.WithKeys("r", "g", "d"), key.WithHelp("r/g/d", m.t("mode", "模式"))),
			}
		}
	case 2:
		return []key.Binding{
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", m.t("import", "导入"))),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", m.t("update", "更新"))),
			key.NewBinding(key.WithKeys("U"), key.WithHelp("U", m.t("update all", "更新全部"))),
			key.NewBinding(key.WithKeys("x"), key.WithHelp("x", m.t("delete", "删除"))),
		}
	case 3:
		if m.systemTab == 1 {
			return []key.Binding{
				key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/shift+tab", m.t("focus", "焦点"))),
				key.NewBinding(key.WithKeys("s"), key.WithHelp("s", m.t("save", "保存"))),
			}
		}
		return []key.Binding{
			key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/j k", m.t("scroll", "滚动"))),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", m.t("clear", "清空"))),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", m.t("palette", "面板"))),
		key.NewBinding(key.WithKeys("1", "2", "3", "4"), key.WithHelp("1-4", m.t("tabs", "标签"))),
		key.NewBinding(key.WithKeys("L"), key.WithHelp("L", m.t("language", "语言"))),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", m.t("quit", "退出"))),
	}
}

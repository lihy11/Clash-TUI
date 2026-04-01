package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) proxyPanels(w, h int) (leftX, leftY, leftW, leftH, rightX, rightY, rightW, rightH int) {
	leftX, leftY = 0, m.bodyY+1
	if w < 90 {
		topH := h / 2
		bottomH := h - topH - 1
		if bottomH < 4 {
			bottomH = 4
		}
		leftW = w
		leftH = topH
		rightX = 0
		rightY = leftY + topH + 1
		rightW = w
		rightH = bottomH
		return
	}

	leftW = max(26, w/4)
	leftH = h
	rightX = leftW + 1
	rightY = leftY
	rightW = w - leftW - 1
	rightH = h
	return
}

func (m *model) renderProxyModeLine(contentX, y int) string {
	current := strings.ToLower(m.baseCfg.Mode)
	tokens := []struct {
		label string
		mode  string
	}{
		{label: m.t("Rule", "规则"), mode: "rule"},
		{label: m.t("Global", "全局"), mode: "global"},
		{label: m.t("Direct", "直连"), mode: "direct"},
	}

	prefix := m.t("Mode: ", "模式: ")
	line := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	for i, t := range tokens {
		token := "[" + t.label + "]"
		if t.mode == current {
			token = "[✓ " + strings.ToUpper(t.label) + "]"
		} else {
			token = "[○ " + t.label + "]"
		}
		startX := cursorX
		endX := startX + lipgloss.Width(token)
		m.proxyModeTargets = append(m.proxyModeTargets, clickTarget{x1: startX, y1: y, x2: endX, y2: y + 1, text: t.mode})
		line += token
		cursorX = endX
		if i < len(tokens)-1 {
			line += " "
			cursorX++
		}
	}
	return line
}

func (m *model) renderOverviewModeLines(contentX, y, maxW int) []string {
	current := strings.ToLower(m.baseCfg.Mode)
	tokens := []struct {
		label string
		mode  string
	}{
		{label: m.t("Rule", "规则"), mode: "rule"},
		{label: m.t("Global", "全局"), mode: "global"},
		{label: m.t("Direct", "直连"), mode: "direct"},
	}

	prefix := m.t("Switch: ", "切换: ")
	sep := " "
	currentY := y
	currentLine := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	lines := make([]string, 0, 2)
	for i, t := range tokens {
		token := "[○ " + t.label + "]"
		if t.mode == current {
			token = "[✓ " + strings.ToUpper(t.label) + "]"
		}
		addSep := ""
		if i > 0 {
			addSep = sep
		}
		projectedW := lipgloss.Width(currentLine + addSep + token)
		if i > 0 && projectedW > maxW {
			lines = append(lines, currentLine)
			currentY++
			currentLine = strings.Repeat(" ", lipgloss.Width(prefix))
			cursorX = contentX + lipgloss.Width(prefix)
			addSep = ""
		}
		if addSep != "" {
			currentLine += addSep
			cursorX += lipgloss.Width(addSep)
		}
		startX := cursorX
		endX := startX + lipgloss.Width(token)
		m.overviewModeTargets = append(m.overviewModeTargets, clickTarget{x1: startX, y1: currentY, x2: endX, y2: currentY + 1, text: t.mode})
		currentLine += token
		cursorX = endX
	}
	lines = append(lines, currentLine)
	return lines
}

func (m *model) renderOverviewToggleLines(contentX, y, maxW int) []string {
	systemOn := m.systemProxyEnabled()
	tunOn := m.baseCfg.Tun.Enable

	tokens := []struct {
		id    string
		label string
		on    bool
	}{
		{id: "system-proxy", label: m.t("System Proxy", "系统代理"), on: systemOn},
		{id: "tun", label: m.t("TUN Mode", "TUN 模式"), on: tunOn},
	}

	prefix := m.t("Toggles: ", "开关: ")
	sep := "   "
	currentY := y
	currentLine := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	lines := make([]string, 0, 2)
	for i, t := range tokens {
		label := m.styles.subTab.Render(t.label)
		sw := m.renderWebToggle(t.on)
		token := label + " " + sw
		addSep := ""
		if i > 0 {
			addSep = sep
		}
		projectedW := lipgloss.Width(currentLine + addSep + token)
		if i > 0 && projectedW > maxW {
			lines = append(lines, currentLine)
			currentY++
			currentLine = strings.Repeat(" ", lipgloss.Width(prefix))
			cursorX = contentX + lipgloss.Width(prefix)
			addSep = ""
		}
		if addSep != "" {
			currentLine += addSep
			cursorX += lipgloss.Width(addSep)
		}
		currentLine += token

		wToken := lipgloss.Width(token)
		m.overviewToggleTargets = append(m.overviewToggleTargets, clickTarget{
			x1: cursorX, y1: currentY, x2: cursorX + wToken, y2: currentY + 1,
			text: fmt.Sprintf("%s:%s", t.id, map[bool]string{true: "off", false: "on"}[t.on]),
		})
		cursorX += wToken
	}
	lines = append(lines, currentLine)
	return lines
}

func (m *model) renderWebToggle(on bool) string {
	t := themes[m.themeIndex]
	if on {
		return lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color(t.CursorText)).
			Background(lipgloss.Color(t.Success)).
			Padding(0, 1).Render("ON  ●")
	}
	return lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color(t.PillText)).
		Background(lipgloss.Color(t.Panel)).
		Padding(0, 1).Render("○  OFF")
}

func (m *model) renderProxyActionLine(contentX, y int) string {
	prefix := m.t("Action: ", "操作: ")
	button := m.styles.action.Render(m.t(" ⚡ Test All (T) ", " ⚡ 全部测试 (T) "))

	startX := contentX + lipgloss.Width(prefix)
	endX := startX + lipgloss.Width(button)
	m.proxyActionTarget = append(m.proxyActionTarget, clickTarget{x1: startX, y1: y, x2: endX, y2: y + 1, text: "test_all"})
	return prefix + button
}

func (m *model) panelContentHeight(outerHeight int) int {
	return max(1, outerHeight-m.styles.panel.GetVerticalFrameSize())
}

func (m *model) panelContentOrigin(panelX, panelY int) (x, y int) {
	return panelX + 3, panelY + 2
}

func (m *model) renderPanel(outerW, outerH int, content string) string {
	w := max(1, outerW)
	h := max(1, outerH)
	return m.styles.panel.Width(w).Height(h).MaxWidth(w).MaxHeight(h).Render(content)
}

func (m *model) renderPanelFocused(outerW, outerH int, content string, focused bool) string {
	w := max(1, outerW)
	h := max(1, outerH)
	st := m.styles.panel
	if focused {
		st = st.BorderForeground(lipgloss.Color(themes[m.themeIndex].Primary))
	} else {
		st = st.BorderForeground(lipgloss.Color(themes[m.themeIndex].Panel))
	}
	return st.Width(w).Height(h).MaxWidth(w).MaxHeight(h).Render(content)
}

func (m *model) renderDelayCell(node string, cellW int) string {
	delay, ok := m.delayMap[node]
	if !ok {
		return m.styles.subtle.Render(fitTextWidth("○", cellW))
	}
	if delay < 0 {
		return m.styles.subtle.Render(fitTextWidth("○ Timeout", cellW))
	}
	if delay == 0 {
		return m.styles.subtle.Render(fitTextWidth("○", cellW))
	}
	var color, dot string
	var bars int
	switch {
	case delay < 100:
		color, dot, bars = themes[m.themeIndex].Success, "●", 2
	case delay <= 300:
		color, dot, bars = themes[m.themeIndex].Warning, "●", 4
	default:
		color, dot, bars = themes[m.themeIndex].Error, "●", 6
	}
	bar := strings.Repeat("▮", bars)
	cell := fmt.Sprintf("%s %-6s %4dms", dot, bar, delay)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(fitTextWidth(cell, cellW))
}

func (m *model) renderDelayCellPlain(node string, cellW int) string {
	delay, ok := m.delayMap[node]
	if !ok || delay == 0 {
		return fitTextWidth("○", cellW)
	}
	if delay < 0 {
		return fitTextWidth("○ Timeout", cellW)
	}
	var bars int
	switch {
	case delay < 100:
		bars = 2
	case delay <= 300:
		bars = 4
	default:
		bars = 6
	}
	bar := strings.Repeat("▮", bars)
	return fitTextWidth(fmt.Sprintf("● %-6s %4dms", bar, delay), cellW)
}

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderOverview(w, h int) string {
	leftX := 0
	leftY := m.bodyY
	leftW := w
	if w >= 90 {
		leftW = w / 2
	}
	leftContentW := max(1, leftW-m.styles.panel.GetHorizontalFrameSize())
	leftContentX, leftContentY := m.panelContentOrigin(leftX, leftY)
	modeBlock := make([]string, 0, 28)
	row := 0
	addLine := func(s string) {
		modeBlock = append(modeBlock, s)
		row += max(1, lipgloss.Height(s))
	}

	addLine(m.styles.panelTitle.Render(m.t("Dashboard", "总览")))
	addLine("")
	addLine(fmt.Sprintf("%s: %s", m.t("Mode", "模式"), strings.ToUpper(m.baseCfg.Mode)))
	for _, line := range m.renderOverviewModeLines(leftContentX, leftContentY+row, leftContentW) {
		addLine(line)
	}
	addLine(m.styles.subtle.Render(m.t("Keyboard: r/g/d", "快捷键: r/g/d")))
	addLine("")
	addLine(m.styles.panelTitle.Render(m.t("Runtime", "运行时")))
	addLine(fmt.Sprintf("%s: %s", m.t("Endpoint", "端点"), m.cfg.Endpoint))
	addLine(fmt.Sprintf("%s: %s", m.t("Version", "版本"), m.version))
	addLine(fmt.Sprintf("%s: %s", m.t("Theme", "主题"), themes[m.themeIndex].Name))
	for _, line := range m.renderOverviewToggleLines(leftContentX, leftContentY+row, leftContentW) {
		addLine(line)
	}
	addLine(m.styles.subtle.Render(m.t("s: toggle system-proxy   n: toggle tun", "s: 切换系统代理   n: 切换 TUN")))
	if !m.baseCfg.HasSystemProxy {
		addLine(m.styles.subtle.Render(m.t("system-proxy state is tracked locally (core /configs does not expose it)", "system-proxy 状态由本地维护（内核 /configs 未暴露该字段）")))
	}
	addLine("")
	addLine(fmt.Sprintf("%s: %d", m.t("Groups", "代理组"), len(m.groups)))
	addLine(fmt.Sprintf("%s: %d", m.t("Connections", "连接"), len(m.conns)))
	addLine(fmt.Sprintf("%s: %d", m.t("Rules", "规则"), len(m.rules)))
	addLine("")
	addLine(m.styles.panelTitle.Render(m.t("Mini Trend", "迷你趋势")))
	addLine(m.styles.subtle.Render(m.sparkline()))
	addLine("")
	addLine(m.styles.panelTitle.Render(m.t("Quick Actions", "快捷操作")))
	addLine(m.styles.action.Render(m.t(" : Command Palette ", " : 命令面板 ")))
	addLine(m.t("F2: Cycle Theme", "F2: 切换主题"))
	if m.lastErr != nil {
		modeBlock = append(modeBlock, "", m.styles.errorText.Render(m.t("Error: ", "错误: ")+m.lastErr.Error()))
	}
	if w < 90 {
		topH := h / 2
		bottomH := h - topH - 1
		if bottomH < 4 {
			bottomH = 4
		}
		top := m.renderPanel(w, topH, strings.Join(modeBlock, "\n"))
		bottom := m.renderPanel(w, bottomH, m.renderProviderLines())
		return lipgloss.JoinVertical(lipgloss.Left, top, " ", bottom)
	}
	leftW = w / 2
	rightW := w - leftW - 1
	left := m.renderPanel(leftW, h, strings.Join(modeBlock, "\n"))
	right := m.renderPanel(rightW, h, m.renderNotificationCenter(rightW, h))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m *model) renderProviderLines() string {
	providerLines := []string{m.styles.panelTitle.Render(m.t("Providers", "提供者"))}
	if len(m.providers) == 0 {
		providerLines = append(providerLines, "", m.t("No providers", "暂无提供者"))
	} else {
		providerLines = append(providerLines, m.styles.subtle.Render(m.t("u: update provider", "u: 更新 provider")))
		providerLines = append(providerLines, "")
		for i, p := range m.providers {
			item := m.providersState[p]
			alive := 0
			for _, node := range item.Proxies {
				if node.Alive {
					alive++
				}
			}
			line := fmt.Sprintf("• %s (%d/%d alive)", p, alive, len(item.Proxies))
			if i == m.providerCursor {
				line = m.styles.cursor.Render(line)
			}
			providerLines = append(providerLines, line)
		}
	}
	return strings.Join(providerLines, "\n")
}

func (m *model) pillForMode() string {
	mode := strings.ToLower(m.baseCfg.Mode)
	if mode == "" {
		mode = "rule"
	}
	return m.styles.pillActive.Render("mode: " + strings.ToUpper(mode))
}

func (m *model) sparkline() string {
	if len(m.statusMini) == 0 {
		return "▁▁▁▁▁▁▁▁"
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	minV, maxV := m.statusMini[0], m.statusMini[0]
	for _, v := range m.statusMini {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV == minV {
		return strings.Repeat("▅", min(24, len(m.statusMini)))
	}
	start := max(0, len(m.statusMini)-24)
	var b strings.Builder
	for _, v := range m.statusMini[start:] {
		idx := (v - minV) * (len(blocks) - 1) / (maxV - minV)
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

func (m *model) renderNotificationCenter(w, h int) string {
	lines := []string{m.styles.notifTitle.Render(m.t("Notifications", "通知"))}
	if len(m.notifications) == 0 {
		lines = append(lines, "", m.t("No recent events", "暂无事件"))
		return strings.Join(lines, "\n")
	}
	contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
	contentH := max(3, m.panelContentHeight(h)-2)
	stream := make([]string, 0, len(m.notifications))
	for _, n := range m.notifications {
		ts := n.At.Format("15:04:05")
		line := fmt.Sprintf("[%s] %s", ts, n.Text)
		if n.Level == "error" {
			line = m.styles.errorText.Render(line)
		}
		stream = append(stream, line)
	}
	m.notifView.Width = contentW
	m.notifView.Height = contentH
	m.notifView.SetContent(strings.Join(stream, "\n"))
	lines = append(lines, m.notifView.View())
	return strings.Join(lines, "\n")
}

package ui

import (
	"strings"
	"time"
)

func (m *model) renderLogs(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render(m.t("Logs", "日志")),
		m.styles.subtle.Render(m.t("Streaming /logs (c to clear)", "实时 /logs（按 c 清空）")),
	}
	contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
	contentH := max(3, m.panelContentHeight(h)-2)
	stickBottom := m.logsView.AtBottom()
	m.logsView.Width = contentW
	m.logsView.Height = contentH
	m.logsView.SetContent(strings.Join(m.logs, "\n"))
	if stickBottom {
		m.logsView.GotoBottom()
	}
	lines = append(lines, m.logsView.View())
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderProfiles(w, h int) string {
	m.profileImportTargets = nil

	lines := []string{
		m.styles.panelTitle.Render(m.t("Profiles / Subscriptions", "配置 / 订阅")),
		m.styles.subtle.Render(m.t("i: import   x: delete selected   u: update selected   U: update all", "i: 导入   x: 删除当前   u: 更新当前   U: 全部更新")),
		"",
	}
	if len(m.subscriptions) == 0 {
		lines = append(lines, m.t("No subscriptions. Press i to import.", "暂无订阅，按 i 导入。"))
	} else {
		contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
		contentH := max(3, m.panelContentHeight(h)-5)
		m.profileLV.SetSize(contentW, contentH)
		m.profileLV.Select(clamp(m.profileCursor, 0, max(0, len(m.subscriptions)-1)))
		lines = append(lines, m.profileLV.View())
		if p := m.selectedProfileProvider(); p != "" {
			for _, s := range m.subscriptions {
				if s.ProviderName != p {
					continue
				}
				if !s.LastUpdated.IsZero() {
					lines = append(lines, m.styles.subtle.Render(m.t("updated", "更新时间")+": "+s.LastUpdated.Format(time.RFC3339)))
				}
				if s.LastError != "" {
					lines = append(lines, m.styles.errorText.Render(m.t("error", "错误")+": "+s.LastError))
				}
				break
			}
		}
	}

	if m.importing {
		contentX, contentY := m.panelContentOrigin(0, m.bodyY)
		contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
		base := len(lines)
		lines = append(lines, "")
		lines = append(lines, m.styles.panelTitle.Render(m.t("Import Subscription", "导入订阅")))
		lines = append(lines, m.t("Name (optional)", "名称（可选）"))
		lines = append(lines, m.importName.View())
		lines = append(lines, "URL")
		lines = append(lines, m.importURL.View())
		lines = append(lines, m.styles.subtle.Render(m.t("Enter: confirm  Esc: cancel", "Enter: 确认  Esc: 取消")))
		m.profileImportTargets = append(m.profileImportTargets,
			clickTarget{x1: contentX, y1: contentY + base + 3, x2: contentX + contentW, y2: contentY + base + 4, idx: 0},
			clickTarget{x1: contentX, y1: contentY + base + 5, x2: contentX + contentW, y2: contentY + base + 6, idx: 1},
		)
	}

	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderSettings(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render(m.t("Settings", "设置")),
		m.styles.subtle.Render(m.t("tab/shift+tab move focus  s save and reconnect", "tab/shift+tab 移动焦点  s 保存并重连")),
		"",
		m.t("Controller Endpoint", "控制器地址"),
		m.settingsInputs[0].View(),
		"",
		m.t("Secret", "密钥"),
		m.settingsInputs[1].View(),
		"",
		m.t("Poll Interval (seconds)", "轮询间隔（秒）"),
		m.settingsInputs[2].View(),
		"",
		m.t("Log Level (debug/info/warning/error)", "日志级别（debug/info/warning/error）"),
		m.settingsInputs[3].View(),
		"",
	}
	lines = append(lines, m.styles.successText.Render(m.t("Config file: ~/.config/clash-tui/config.yaml", "配置文件: ~/.config/clash-tui/config.yaml")))
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

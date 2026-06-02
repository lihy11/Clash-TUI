package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderProxies(w, h int) string {
	leftX, leftY, leftW, leftH, rightX, rightY, rightW, rightH := m.proxyPanels(w, h)
	leftContentX, leftContentY := m.panelContentOrigin(leftX, leftY)
	rightContentX, rightContentY := m.panelContentOrigin(rightX, rightY)

	leftLines := []string{m.styles.panelTitle.Render(m.t("Proxy Groups", "代理组"))}
	leftRow := lipgloss.Height(leftLines[0])
	for i, g := range m.groups {
		typ := strings.ToUpper(strings.TrimSpace(m.proxies[g].Type))
		if typ == "" {
			typ = "-"
		}
		line := fmt.Sprintf("%s  ·  %s", g, typ)
		if i == m.groupCursor {
			line = m.styles.cursor.Render("┃ " + line)
		}
		leftLines = append(leftLines, line)
		row := leftRow
		m.mouse.register(mouseAction{
			ID:    "proxy.group.select",
			Index: i,
			Box: hitBox{
				x1: leftContentX,
				y1: leftContentY + row,
				x2: leftContentX + max(1, leftW-m.styles.panel.GetHorizontalFrameSize()),
				y2: leftContentY + row + 1,
			},
		})
		leftRow += max(1, lipgloss.Height(line))
	}
	if len(m.groups) == 0 {
		leftLines = append(leftLines, "No proxy groups")
	}
	if m.proxyPane == 0 {
		leftLines = append(leftLines, "", m.styles.subtle.Render(m.t("Focus: groups (h/l to switch)", "焦点: 组列表（h/l 切换）")))
	}

	rightLines := []string{m.styles.panelTitle.Render(m.t("Nodes", "节点"))}
	rightRow := lipgloss.Height(rightLines[0])
	modeLine := m.renderProxyModeLine(rightContentX, rightContentY+rightRow)
	rightLines = append(rightLines, modeLine)
	rightRow += max(1, lipgloss.Height(modeLine))
	actionLine := m.renderProxyActionLine(rightContentX, rightContentY+rightRow)
	rightLines = append(rightLines, actionLine)
	rightRow += max(1, lipgloss.Height(actionLine))
	group := ""
	now := ""
	if len(m.groups) > 0 {
		group = m.groups[m.groupCursor]
		now = m.proxies[group].Now
		groupLine := m.styles.subtle.Render(m.t("Group", "组") + ": " + group)
		rightLines = append(rightLines, groupLine)
		rightRow += max(1, lipgloss.Height(groupLine))
	} else {
		groupLine := m.styles.subtle.Render(m.t("Group", "组") + ": -")
		rightLines = append(rightLines, groupLine)
		rightRow += max(1, lipgloss.Height(groupLine))
	}
	countLine := m.styles.subtle.Render(fmt.Sprintf("%s: %d", m.t("Nodes", "节点数"), len(m.nodes)))
	rightLines = append(rightLines, countLine)
	rightRow += max(1, lipgloss.Height(countLine))
	nodeStartRow := rightRow
	maxNodeRows := max(1, m.panelContentHeight(rightH)-nodeStartRow)
	if m.proxyPane == 1 {
		maxNodeRows = max(1, maxNodeRows-2)
	}
	m.nodePageSize = maxNodeRows
	m.ensureNodeVisible()
	start := m.nodeOffset
	end := min(len(m.nodes), start+maxNodeRows)
	for i := start; i < end; i++ {
		n := m.nodes[i]
		innerW := max(1, rightW-m.styles.panel.GetHorizontalFrameSize())
		delayW := 18
		nameW := max(10, innerW-delayW-1)
		delayCell := m.renderDelayCell(n, delayW)
		line := fitTextWidth(n, nameW) + " " + delayCell
		linePlain := fitTextWidth(n, nameW) + " " + m.renderDelayCellPlain(n, delayW)
		if n == now {
			line = m.styles.selected.Render("★ " + line)
			linePlain = "★ " + linePlain
		}
		if i == m.nodeCursor {
			line = m.styles.cursor.Render(linePlain)
		}
		rightLines = append(rightLines, line)
		row := nodeStartRow + (i - start)
		m.mouse.register(mouseAction{
			ID:    "proxy.node.select",
			Index: i,
			Box: hitBox{
				x1: rightContentX,
				y1: rightContentY + row,
				x2: rightContentX + max(1, rightW-m.styles.panel.GetHorizontalFrameSize()),
				y2: rightContentY + row + 1,
			},
		})
	}
	if len(m.nodes) == 0 {
		rightLines = append(rightLines, m.t("No nodes", "暂无节点"))
	}
	if m.proxyPane == 1 {
		rightLines = append(rightLines, "", m.styles.subtle.Render(m.t("Enter: switch node   t: test one   T: test all", "Enter: 切换节点   t: 测试当前   T: 全部测试")))
	}
	left := m.renderPanelFocused(leftW, leftH, strings.Join(leftLines, "\n"), m.proxyPane == 0)
	right := m.renderPanelFocused(rightW, rightH, strings.Join(rightLines, "\n"), m.proxyPane == 1)
	if w < 90 {
		return lipgloss.JoinVertical(lipgloss.Left, left, " ", right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m *model) renderConnections(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render(m.t("Connections", "连接")),
		m.styles.subtle.Render(m.t("x: close selected   X: close all", "x: 关闭当前   X: 全部关闭")),
	}
	if len(m.conns) == 0 {
		lines = append(lines, "", m.t("No active connections", "没有活动连接"))
		return m.renderPanel(w, h, strings.Join(lines, "\n"))
	}
	contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
	contentH := max(3, m.panelContentHeight(h)-2)
	hostW := max(16, contentW/5)
	procW := max(12, contentW/6)
	chainW := max(20, contentW-hostW-procW-6)
	rows := make([]table.Row, 0, len(m.conns))
	for _, c := range m.conns {
		host := c.Metadata.Host
		if host == "" {
			host = c.Metadata.DestinationIP
		}
		process := c.Metadata.Process
		if process == "" {
			process = "-"
		}
		chain := strings.Join(c.Chains, " -> ")
		if chain == "" {
			chain = "-"
		}
		rows = append(rows, table.Row{fitTextWidth(host, hostW), fitTextWidth(process, procW), fitTextWidth(chain, chainW)})
	}
	m.connsTable.Focus()
	m.connsTable.SetColumns([]table.Column{{Title: m.t("Host", "主机"), Width: hostW}, {Title: m.t("Process", "进程"), Width: procW}, {Title: m.t("Chain", "链路"), Width: chainW}})
	m.connsTable.SetRows(rows)
	m.connsTable.SetWidth(contentW)
	m.connsTable.SetHeight(contentH)
	m.connsTable.SetCursor(clamp(m.connCursor, 0, max(0, len(rows)-1)))
	m.connCursor = m.connsTable.Cursor()
	lines = append(lines, m.connsTable.View())
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderRules(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render(m.t("Rules", "规则")),
		m.styles.subtle.Render(m.t("j/k scroll", "j/k 滚动")),
	}
	if len(m.rules) == 0 {
		lines = append(lines, "", m.t("No rules", "暂无规则"))
		return m.renderPanel(w, h, strings.Join(lines, "\n"))
	}
	contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
	contentH := max(3, m.panelContentHeight(h)-2)
	typeW := max(12, contentW/6)
	proxyW := max(14, contentW/5)
	payloadW := max(16, contentW-typeW-proxyW-6)
	rows := make([]table.Row, 0, len(m.rules))
	for _, r := range m.rules {
		rows = append(rows, table.Row{fitTextWidth(r.Type, typeW), fitTextWidth(r.Payload, payloadW), fitTextWidth("-> "+r.Proxy, proxyW)})
	}
	m.rulesTable.Focus()
	m.rulesTable.SetColumns([]table.Column{{Title: m.t("Type", "类型"), Width: typeW}, {Title: m.t("Payload", "匹配"), Width: payloadW}, {Title: m.t("Proxy", "策略"), Width: proxyW}})
	m.rulesTable.SetRows(rows)
	m.rulesTable.SetWidth(contentW)
	m.rulesTable.SetHeight(contentH)
	m.rulesTable.SetCursor(clamp(m.rulesOffset, 0, max(0, len(rows)-1)))
	m.rulesOffset = m.rulesTable.Cursor()
	lines = append(lines, m.rulesTable.View())
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

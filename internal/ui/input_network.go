package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleProxyKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "h":
		m.proxyPane = 0
	case "l":
		m.proxyPane = 1
	case "up", "k":
		if m.proxyPane == 0 {
			m.groupCursor = clamp(m.groupCursor-1, 0, max(0, len(m.groups)-1))
			m.syncNodeCursorByGroup()
		} else {
			m.nodeCursor = clamp(m.nodeCursor-1, 0, max(0, len(m.nodes)-1))
			m.ensureNodeVisible()
		}
	case "down", "j":
		if m.proxyPane == 0 {
			m.groupCursor = clamp(m.groupCursor+1, 0, max(0, len(m.groups)-1))
			m.syncNodeCursorByGroup()
		} else {
			m.nodeCursor = clamp(m.nodeCursor+1, 0, max(0, len(m.nodes)-1))
			m.ensureNodeVisible()
		}
	case "enter":
		if m.proxyPane == 1 && len(m.groups) > 0 && len(m.nodes) > 0 {
			group := m.groups[m.groupCursor]
			node := m.nodes[m.nodeCursor]
			return setProxyCmd(m.client, group, node)
		}
	case "t":
		if m.proxyPane == 1 && len(m.nodes) > 0 {
			return testDelayCmd(m.client, m.nodes[m.nodeCursor])
		}
	case "T":
		if len(m.nodes) > 0 {
			m.setStatus(fmt.Sprintf("testing %d nodes...", len(m.nodes)))
			return testAllNodesCmd(m.client, m.nodes)
		}
	}
	return nil
}

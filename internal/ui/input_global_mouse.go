package ui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	if m.selectorOpen {
		switch msg.String() {
		case "esc":
			m.selectorOpen = false
			return nil
		case "enter":
			return m.applySelectorChoice()
		}
		var cmd tea.Cmd
		m.selectorLV, cmd = m.selectorLV.Update(msg)
		return cmd
	}
	switch msg.String() {
	case "ctrl+c", "q":
		if m.client != nil {
			m.client.Close()
		}
		return tea.Quit
	case ":", "ctrl+k":
		m.paletteOpen = true
		m.paletteCursor = 0
		m.paletteInput.SetValue("")
		m.paletteInput.Focus()
		return nil
	case "L":
		m.toggleLanguage()
		return nil
	case "tab", "right":
		m.tab = (m.tab + 1) % len(tabs)
		return nil
	case "shift+tab", "left":
		m.tab = (m.tab - 1 + len(tabs)) % len(tabs)
		return nil
	case "1", "2", "3", "4":
		i, _ := strconv.Atoi(msg.String())
		m.tab = i - 1
		return nil
	case "]":
		if m.tab == 1 {
			m.networkTab = (m.networkTab + 1) % len(networkTabs)
			return nil
		}
		if m.tab == 3 {
			m.systemTab = (m.systemTab + 1) % len(systemTabs)
			return nil
		}
	case "[":
		if m.tab == 1 {
			m.networkTab = (m.networkTab - 1 + len(networkTabs)) % len(networkTabs)
			return nil
		}
		if m.tab == 3 {
			m.systemTab = (m.systemTab - 1 + len(systemTabs)) % len(systemTabs)
			return nil
		}
	case "F2", "f2":
		m.openSelector("theme")
		return nil
	case "m":
		m.openSelector("mode")
		return nil
	case "r":
		if m.tab == 0 || (m.tab == 1 && m.networkTab == 0) {
			return tea.Batch(
				setModeCmd(m.client, "rule"),
				fetchConfigCmd(m.client),
			)
		}
	case "g":
		if m.tab == 0 || (m.tab == 1 && m.networkTab == 0) {
			return tea.Batch(
				setModeCmd(m.client, "global"),
				fetchConfigCmd(m.client),
			)
		}
	case "d":
		if m.tab == 0 || (m.tab == 1 && m.networkTab == 0) {
			return tea.Batch(
				setModeCmd(m.client, "direct"),
				fetchConfigCmd(m.client),
			)
		}
	}
	return nil
}

func (m *model) handleMouseMsg(msg tea.MouseMsg) tea.Cmd {
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		if action, ok := m.mouse.dispatch(msg); ok {
			return m.handleMouseAction(action)
		}
	}

	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		if m.selectorOpen {
			if handled, cmd := m.handleSelectorMouse(msg.X, msg.Y); handled {
				return cmd
			}
		}
	}

	if m.tab == 1 && m.networkTab == 0 {
		if msg.Button == tea.MouseButtonWheelUp {
			m.scrollProxyNodesByWheel(-1)
			return nil
		}
		if msg.Button == tea.MouseButtonWheelDown {
			m.scrollProxyNodesByWheel(1)
			return nil
		}
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	if m.tab == 2 && m.importing {
		return m.handleProfilesMouse(msg.X, msg.Y)
	}
	return nil
}

func (m *model) scrollProxyNodesByWheel(delta int) {
	if len(m.nodes) == 0 {
		return
	}
	m.proxyPane = 1
	page := max(1, m.nodePageSize)
	maxOffset := max(0, len(m.nodes)-page)
	m.nodeOffset = clamp(m.nodeOffset+delta, 0, maxOffset)
}

func (m *model) handleMouseAction(action mouseAction) tea.Cmd {
	switch action.ID {
	case "header.language.toggle":
		m.toggleLanguage()
		return nil
	case "tab.main.select":
		m.tab = clamp(action.Index, 0, len(tabs)-1)
		return nil
	case "tab.network.select":
		m.networkTab = clamp(action.Index, 0, len(networkTabs)-1)
		return nil
	case "tab.system.select":
		m.systemTab = clamp(action.Index, 0, len(systemTabs)-1)
		return nil
	case "overview.mode.set", "proxy.mode.set":
		return tea.Batch(setModeCmd(m.client, action.Text), fetchConfigCmd(m.client))
	case "overview.toggle":
		switch action.Text {
		case "system-proxy:on":
			return setSystemProxyCmd(m.client, true)
		case "system-proxy:off":
			return setSystemProxyCmd(m.client, false)
		case "tun:on":
			return setTunCmd(m.client, true)
		case "tun:off":
			return setTunCmd(m.client, false)
		}
		return nil
	case "proxy.action":
		if action.Text == "test_all" && len(m.nodes) > 0 {
			m.setStatus(fmt.Sprintf("testing %d nodes...", len(m.nodes)))
			return testAllNodesCmd(m.client, m.nodes)
		}
		return nil
	case "proxy.group.select":
		m.groupCursor = clamp(action.Index, 0, max(0, len(m.groups)-1))
		m.syncNodeCursorByGroup()
		return nil
	case "selector.dismiss":
		m.selectorOpen = false
		return nil
	case "selector.choose":
		return m.applySelectorChoiceByID(action.Text)
	case "proxy.node.select":
		m.nodeCursor = clamp(action.Index, 0, max(0, len(m.nodes)-1))
		m.ensureNodeVisible()
		if len(m.groups) > 0 && len(m.nodes) > 0 {
			group := m.groups[m.groupCursor]
			node := m.nodes[m.nodeCursor]
			return setProxyCmd(m.client, group, node)
		}
		return nil
	default:
		return nil
	}
}

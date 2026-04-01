package ui

import (
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
		if m.selectorOpen {
			if handled, cmd := m.handleSelectorMouse(msg.X, msg.Y); handled {
				return cmd
			}
		}
		if m.langChipTarget.hit(msg.X, msg.Y) {
			m.toggleLanguage()
			return nil
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
	for _, t := range m.mainTabTargets {
		if t.hit(msg.X, msg.Y) {
			m.tab = t.idx
			return nil
		}
	}
	if m.tab == 1 {
		for _, t := range m.networkTabTargets {
			if t.hit(msg.X, msg.Y) {
				m.networkTab = t.idx
				return nil
			}
		}
	}
	if m.tab == 3 {
		for _, t := range m.systemTabTargets {
			if t.hit(msg.X, msg.Y) {
				m.systemTab = t.idx
				return nil
			}
		}
	}
	if m.tab == 0 {
		return m.handleOverviewMouse(msg.X, msg.Y)
	}
	if m.tab == 1 && m.networkTab == 0 {
		return m.handleProxyMouse(msg.X, msg.Y)
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

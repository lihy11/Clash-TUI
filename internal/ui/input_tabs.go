package ui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleOverviewMouse(x, y int) tea.Cmd {
	for _, t := range m.overviewModeTargets {
		if t.hit(x, y) {
			return tea.Batch(setModeCmd(m.client, t.text), fetchConfigCmd(m.client))
		}
	}
	for _, t := range m.overviewToggleTargets {
		if t.hit(x, y) {
			switch t.text {
			case "system-proxy:on":
				return setSystemProxyCmd(m.client, true)
			case "system-proxy:off":
				return setSystemProxyCmd(m.client, false)
			case "tun:on":
				return setTunCmd(m.client, true)
			case "tun:off":
				return setTunCmd(m.client, false)
			}
		}
	}
	return nil
}

func (m *model) handleProfilesMouse(x, y int) tea.Cmd {
	for _, t := range m.profileImportTargets {
		if t.hit(x, y) {
			m.importFocus = clamp(t.idx, 0, 1)
			return nil
		}
	}
	return nil
}

func (m *model) handleOverviewKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd
		m.notifView, cmd = m.notifView.Update(msg)
		return cmd
	}
	switch msg.String() {
	case "up", "k":
		m.providerCursor = clamp(m.providerCursor-1, 0, max(0, len(m.providers)-1))
	case "down", "j":
		m.providerCursor = clamp(m.providerCursor+1, 0, max(0, len(m.providers)-1))
	case "u":
		if len(m.providers) > 0 {
			return updateProviderCmd(m.client, m.providers[m.providerCursor])
		}
	case "s":
		return setSystemProxyCmd(m.client, !m.systemProxyEnabled())
	case "n":
		return setTunCmd(m.client, !m.baseCfg.Tun.Enable)
	}
	return nil
}

func (m *model) handleConnectionKeys(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.connsTable, cmd = m.connsTable.Update(msg)
	m.connCursor = m.connsTable.Cursor()
	switch msg.String() {
	case "x":
		if len(m.conns) > 0 {
			i := clamp(m.connsTable.Cursor(), 0, max(0, len(m.conns)-1))
			return closeConnCmd(m.client, m.conns[i].ID)
		}
	case "X":
		return closeAllConnCmd(m.client)
	}
	return cmd
}

func (m *model) handleRuleKeys(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.rulesTable, cmd = m.rulesTable.Update(msg)
	m.rulesOffset = m.rulesTable.Cursor()
	return cmd
}

func (m *model) handleLogKeys(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.logsView, cmd = m.logsView.Update(msg)
	switch msg.String() {
	case "c":
		m.logs = m.logs[:0]
		m.logsView.SetContent("")
	}
	return cmd
}

func (m *model) handleSettingsKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "shift+tab":
		m.settingsFocus = (m.settingsFocus - 1 + len(m.settingsInputs)) % len(m.settingsInputs)
	case "down", "tab":
		m.settingsFocus = (m.settingsFocus + 1) % len(m.settingsInputs)
	case "s":
		return m.applySettings()
	}
	return nil
}

func (m *model) handleProfilesKeys(msg tea.KeyMsg) tea.Cmd {
	if m.importing {
		switch msg.String() {
		case "esc":
			m.importing = false
			return nil
		case "up", "k":
			m.importFocus = (m.importFocus - 1 + 2) % 2
			return nil
		case "down", "j":
			m.importFocus = (m.importFocus + 1) % 2
			return nil
		case "tab":
			m.importFocus = (m.importFocus + 1) % 2
			return nil
		case "shift+tab":
			m.importFocus = (m.importFocus - 1 + 2) % 2
			return nil
		case "enter":
			return importSubCmd(m.rt, m.importName.Value(), m.importURL.Value())
		}
		return nil
	}

	switch msg.String() {
	case "up", "k", "down", "j":
		var cmd tea.Cmd
		m.profileLV, cmd = m.profileLV.Update(msg)
		m.profileCursor = m.profileLV.Index()
		return cmd
	case "i":
		m.importing = true
		m.importFocus = 1
		m.importName.SetValue("")
		m.importURL.SetValue("")
	case "u":
		if p := m.selectedProfileProvider(); p != "" {
			return updateProviderCmd2(m.client, p)
		}
	case "x":
		if p := m.selectedProfileProvider(); p != "" {
			return deleteSubCmd(m.rt, p)
		}
	case "U":
		if len(m.subscriptions) > 0 {
			cmds := make([]tea.Cmd, 0, len(m.subscriptions))
			for _, it := range m.subscriptions {
				cmds = append(cmds, updateProviderCmd2(m.client, it.ProviderName))
			}
			return tea.Batch(cmds...)
		}
	}
	var cmd tea.Cmd
	m.profileLV, cmd = m.profileLV.Update(msg)
	m.profileCursor = m.profileLV.Index()
	return cmd
}

func (m *model) handlePaletteKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.paletteOpen = false
		m.paletteInput.Blur()
		return nil
	case "enter":
		actions := m.filteredPaletteActions()
		if len(actions) == 0 {
			m.paletteOpen = false
			return nil
		}
		m.paletteOpen = false
		m.paletteInput.Blur()
		return m.runPaletteAction(actions[m.paletteCursor].ID)
	case "up", "k":
		actions := m.filteredPaletteActions()
		if len(actions) > 0 {
			m.paletteCursor = clamp(m.paletteCursor-1, 0, len(actions)-1)
		}
		return nil
	case "down", "j":
		actions := m.filteredPaletteActions()
		if len(actions) > 0 {
			m.paletteCursor = clamp(m.paletteCursor+1, 0, len(actions)-1)
		}
		return nil
	}
	var cmd tea.Cmd
	m.paletteInput, cmd = m.paletteInput.Update(msg)
	actions := m.filteredPaletteActions()
	m.paletteCursor = clamp(m.paletteCursor, 0, max(0, len(actions)-1))
	return cmd
}

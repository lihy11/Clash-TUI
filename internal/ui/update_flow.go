package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleKeyUpdate(msg tea.KeyMsg) tea.Cmd {
	var deferredCmd tea.Cmd
	if m.paletteOpen {
		return m.handlePaletteKeys(msg)
	}
	if m.selectorOpen {
		return m.handleGlobalKeys(msg)
	}
	if m.tab == 2 && m.importing {
		deferredCmd = m.handleProfilesKeys(msg)
	} else {
		if cmd := m.handleGlobalKeys(msg); cmd != nil {
			return cmd
		}
	}

	switch m.tab {
	case 0:
		return m.handleOverviewKeys(msg)
	case 1:
		switch m.networkTab {
		case 0:
			return m.handleProxyKeys(msg)
		case 1:
			return m.handleRuleKeys(msg)
		case 2:
			return m.handleConnectionKeys(msg)
		}
	case 2:
		if !m.importing {
			return m.handleProfilesKeys(msg)
		}
	case 3:
		if m.systemTab == 0 {
			return m.handleLogKeys(msg)
		}
		deferredCmd = m.handleSettingsKeys(msg)
	}
	return m.postUpdateComponents(msg, deferredCmd)
}

func (m *model) handleAppMessage(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case versionMsg:
		m.version = msg.v.Version
		m.loading = false
		m.setStatus("connected")
		m.lastErr = nil
	case cfgMsg:
		m.baseCfg = msg.c
		if msg.c.HasSystemProxy {
			m.systemProxyOn = msg.c.SystemProxy
		}
		m.lastErr = nil
	case proxiesMsg:
		m.proxies = msg.p.Proxies
		m.rebuildGroupsAndNodes()
		m.lastErr = nil
	case rulesMsg:
		m.rules = msg.r.Rules
		m.rulesOffset = clamp(m.rulesOffset, 0, max(0, len(m.rules)-1))
		m.lastErr = nil
	case connectionsMsg:
		m.conns = msg.c.Connections
		m.updateThroughput(msg.c.Connections)
		m.connCursor = clamp(m.connCursor, 0, max(0, len(m.conns)-1))
		m.lastErr = nil
	case providersMsg:
		m.providersState = msg.p.Providers
		m.providers = m.providers[:0]
		for k := range m.providersState {
			m.providers = append(m.providers, k)
		}
		sort.Strings(m.providers)
		m.providerCursor = clamp(m.providerCursor, 0, max(0, len(m.providers)-1))
		m.lastErr = nil
	case subsMsg:
		m.subscriptions = msg.items
		m.profileCursor = clamp(m.profileCursor, 0, max(0, len(m.subscriptions)-1))
		m.syncProfileListItems()
		m.lastErr = nil
	case importSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("import subscription failed")
		} else {
			m.setStatus("subscription imported")
			m.importing = false
			m.lastErr = nil
		}
		cmds := []tea.Cmd{loadSubsCmd(m.rt)}
		if m.rt != nil {
			cmds = append(cmds,
				reloadCoreCmd(m.rt, m.cfg),
				fetchVersionCmd(m.client),
				fetchConfigCmd(m.client),
				fetchProxiesCmd(m.client),
				fetchProvidersCmd(m.client),
				listenLogCmd(m.client, m.cfg.LogLevel),
			)
		}
		return tea.Batch(cmds...), true
	case updateSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("update subscription failed")
		} else {
			m.setStatus("subscription updated")
			m.lastErr = nil
		}
		return loadSubsCmd(m.rt), true
	case deleteSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("delete subscription failed")
			return nil, true
		}
		m.setStatus("subscription deleted")
		m.lastErr = nil
		cmds := []tea.Cmd{loadSubsCmd(m.rt)}
		if m.rt != nil {
			cmds = append(cmds,
				reloadCoreCmd(m.rt, m.cfg),
				fetchVersionCmd(m.client),
				fetchConfigCmd(m.client),
				fetchProxiesCmd(m.client),
				fetchProvidersCmd(m.client),
				listenLogCmd(m.client, m.cfg.LogLevel),
			)
		}
		return tea.Batch(cmds...), true
	case modeSetMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("set mode failed")
		} else {
			m.baseCfg.Mode = msg.mode
			m.setStatus("mode updated")
			m.lastErr = nil
		}
	case featureSetMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus(fmt.Sprintf("set %s failed", msg.name))
		} else {
			switch msg.name {
			case "system-proxy":
				m.baseCfg.SystemProxy = msg.enabled
				m.systemProxyOn = msg.enabled
			case "tun":
				m.baseCfg.Tun.Enable = msg.enabled
			}
			state := "off"
			if msg.enabled {
				state = "on"
			}
			m.setStatus(fmt.Sprintf("%s %s", msg.name, state))
			m.lastErr = nil
		}
		return fetchConfigCmd(m.client), true
	case proxySetMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("switch node failed")
		} else {
			m.setStatus(fmt.Sprintf("group %s -> %s", msg.group, msg.node))
			if p, ok := m.proxies[msg.group]; ok {
				p.Now = msg.node
				m.proxies[msg.group] = p
			}
			m.lastErr = nil
		}
		return fetchProxiesCmd(m.client), true
	case delayMsg:
		if msg.err != nil {
			m.delayMap[msg.name] = -1
			m.lastErr = msg.err
			m.setStatus("delay test failed")
		} else {
			m.delayMap[msg.name] = msg.delay
			m.setStatus(fmt.Sprintf("delay %s: %dms", msg.name, msg.delay))
			m.lastErr = nil
		}
	case connClosedMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("close connection failed")
		} else {
			if msg.id == "" {
				m.setStatus("closed all connections")
			} else {
				m.setStatus("connection closed")
			}
			m.lastErr = nil
		}
		return fetchConnectionsCmd(m.client), true
	case providersUpdatedMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("provider update failed")
		} else {
			m.setStatus("provider updated")
			m.lastErr = nil
		}
		return fetchProvidersCmd(m.client), true
	case logMsg:
		line := fmt.Sprintf("[%s] %s", strings.ToUpper(msg.log.Type), strings.TrimSpace(msg.log.Payload))
		if line != "[INFO] " {
			m.logs = append(m.logs, line)
		}
		if len(m.logs) > 300 {
			m.logs = m.logs[len(m.logs)-300:]
		}
		return listenLogCmd(m.client, m.cfg.LogLevel), true
	case logErrMsg:
		m.setStatus("log stream reconnecting...")
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return retryLogMsg{} }), true
	case retryLogMsg:
		return listenLogCmd(m.client, m.cfg.LogLevel), true
	case tickMsg:
		m.pollCounter++
		cmds := []tea.Cmd{
			fetchConfigCmd(m.client),
			fetchProxiesCmd(m.client),
			fetchConnectionsCmd(m.client),
			pollCmd(m.cfg.PollInterval),
		}
		if m.pollCounter%5 == 0 {
			cmds = append(cmds, fetchProvidersCmd(m.client))
		}
		if m.pollCounter%15 == 0 {
			cmds = append(cmds, fetchRulesCmd(m.client))
		}
		return tea.Batch(cmds...), true
	case errMsg:
		m.lastErr = msg.err
		m.loading = false
	}
	return nil, false
}

func (m *model) postUpdateComponents(msg tea.Msg, deferredCmd tea.Cmd) tea.Cmd {
	cmds := make([]tea.Cmd, 0, 6)
	if deferredCmd != nil {
		cmds = append(cmds, deferredCmd)
	}
	if m.tab == 3 && m.systemTab == 1 {
		for i := range m.settingsInputs {
			if i == m.settingsFocus {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
			var c tea.Cmd
			m.settingsInputs[i], c = m.settingsInputs[i].Update(msg)
			if c != nil {
				cmds = append(cmds, c)
			}
		}
	}
	if m.tab == 2 && m.importing {
		if m.importFocus == 0 {
			m.importName.Focus()
			m.importURL.Blur()
		} else {
			m.importURL.Focus()
			m.importName.Blur()
		}
		var c1, c2 tea.Cmd
		m.importName, c1 = m.importName.Update(msg)
		m.importURL, c2 = m.importURL.Update(msg)
		if c1 != nil {
			cmds = append(cmds, c1)
		}
		if c2 != nil {
			cmds = append(cmds, c2)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

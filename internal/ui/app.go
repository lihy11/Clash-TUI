package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"clash-tui/internal/config"
	"clash-tui/internal/mihomo"
	"clash-tui/internal/runtime"
	"clash-tui/internal/subscription"
)

var tabs = []string{"Overview", "Proxies", "Connections", "Rules", "Logs", "Profiles", "Settings"}

type errMsg struct{ err error }
type versionMsg struct{ v mihomo.VersionResponse }
type cfgMsg struct{ c mihomo.ConfigResponse }
type proxiesMsg struct{ p mihomo.ProxiesResponse }
type rulesMsg struct{ r mihomo.RulesResponse }
type connectionsMsg struct{ c mihomo.ConnectionsResponse }
type providersMsg struct{ p mihomo.ProvidersResponse }
type modeSetMsg struct {
	mode string
	err  error
}
type proxySetMsg struct {
	group string
	node  string
	err   error
}
type delayMsg struct {
	name  string
	delay int
	err   error
}
type connClosedMsg struct {
	id  string
	err error
}
type providersUpdatedMsg struct {
	name string
	err  error
}
type logMsg struct{ log mihomo.LogEvent }
type logErrMsg struct{ err error }
type retryLogMsg struct{}
type tickMsg time.Time
type subsMsg struct{ items []subscription.Item }
type importSubMsg struct {
	item subscription.Item
	err  error
}
type updateSubMsg struct {
	name string
	err  error
}

type clickTarget struct {
	x1   int
	y1   int
	x2   int
	y2   int
	idx  int
	text string
}

func (t clickTarget) hit(x, y int) bool {
	return x >= t.x1 && x < t.x2 && y >= t.y1 && y < t.y2
}

type model struct {
	styles styles

	width  int
	height int

	tab int

	cfg    config.Settings
	client *mihomo.Client
	rt     *runtime.Manager

	version string
	baseCfg mihomo.ConfigResponse
	proxies map[string]mihomo.Proxy
	groups  []string
	nodes   []string
	rules   []mihomo.Rule
	conns   []mihomo.Connection

	providers      []string
	providersState map[string]mihomo.Provider

	delayMap map[string]int
	logs     []string

	status      string
	lastErr     error
	loading     bool
	pollCounter int

	proxyPane      int
	groupCursor    int
	nodeCursor     int
	connCursor     int
	rulesOffset    int
	providerCursor int

	settingsInputs []textinput.Model
	settingsFocus  int
	subscriptions  []subscription.Item
	profileCursor  int
	importing      bool
	importName     textinput.Model
	importURL      textinput.Model
	importFocus    int

	bodyY int
	bodyW int
	bodyH int

	overviewModeTargets []clickTarget
	proxyModeTargets    []clickTarget
	proxyActionTarget   []clickTarget
	proxyGroupTargets   []clickTarget
	proxyNodeTargets    []clickTarget
}

func NewModel(cfg config.Settings, rt *runtime.Manager) *model {
	client, err := mihomo.NewClient(cfg.Endpoint, cfg.Secret)
	m := &model{
		styles:         defaultStyles(),
		cfg:            cfg,
		client:         client,
		rt:             rt,
		tab:            0,
		proxies:        map[string]mihomo.Proxy{},
		delayMap:       map[string]int{},
		logs:           make([]string, 0, 200),
		loading:        true,
		status:         "connecting mihomo controller...",
		providersState: map[string]mihomo.Provider{},
	}
	m.initSettingsInputs()
	m.initImportInputs()
	if err != nil {
		m.lastErr = err
		m.status = "init client failed"
	}
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		fetchVersionCmd(m.client),
		fetchConfigCmd(m.client),
		fetchProxiesCmd(m.client),
		fetchRulesCmd(m.client),
		fetchConnectionsCmd(m.client),
		fetchProvidersCmd(m.client),
		loadSubsCmd(m.rt),
		pollCmd(m.cfg.PollInterval),
		listenLogCmd(m.client, m.cfg.LogLevel),
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		return m, nil
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			if msg.Y == 1 {
				if idx := m.tabHit(msg.X); idx >= 0 {
					m.tab = idx
				}
			} else if m.tab == 0 {
				if cmd := m.handleOverviewMouse(msg.X, msg.Y); cmd != nil {
					return m, cmd
				}
			} else if m.tab == 1 {
				if cmd := m.handleProxyMouse(msg.X, msg.Y); cmd != nil {
					return m, cmd
				}
			}
		}
	case tea.KeyMsg:
		if cmd := m.handleGlobalKeys(msg); cmd != nil {
			return m, cmd
		}
		switch m.tab {
		case 0:
			return m, m.handleOverviewKeys(msg)
		case 1:
			return m, m.handleProxyKeys(msg)
		case 2:
			return m, m.handleConnectionKeys(msg)
		case 3:
			return m, m.handleRuleKeys(msg)
		case 4:
			return m, m.handleLogKeys(msg)
		case 5:
			return m, m.handleProfilesKeys(msg)
		case 6:
			return m, m.handleSettingsKeys(msg)
		}
	case versionMsg:
		m.version = msg.v.Version
		m.loading = false
		m.status = "connected"
		m.lastErr = nil
	case cfgMsg:
		m.baseCfg = msg.c
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
		m.lastErr = nil
	case importSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "import subscription failed"
		} else {
			m.status = "subscription imported"
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
		return m, tea.Batch(cmds...)
	case updateSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "update subscription failed"
		} else {
			m.status = "subscription updated"
			m.lastErr = nil
		}
		return m, loadSubsCmd(m.rt)
	case modeSetMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "set mode failed"
		} else {
			m.baseCfg.Mode = msg.mode
			m.status = "mode updated"
			m.lastErr = nil
		}
	case proxySetMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "switch node failed"
		} else {
			m.status = fmt.Sprintf("group %s -> %s", msg.group, msg.node)
			if p, ok := m.proxies[msg.group]; ok {
				p.Now = msg.node
				m.proxies[msg.group] = p
			}
			m.lastErr = nil
		}
		return m, fetchProxiesCmd(m.client)
	case delayMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "delay test failed"
		} else {
			m.delayMap[msg.name] = msg.delay
			m.status = fmt.Sprintf("delay %s: %dms", msg.name, msg.delay)
			m.lastErr = nil
		}
	case connClosedMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "close connection failed"
		} else {
			if msg.id == "" {
				m.status = "closed all connections"
			} else {
				m.status = "connection closed"
			}
			m.lastErr = nil
		}
		return m, fetchConnectionsCmd(m.client)
	case providersUpdatedMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.status = "provider update failed"
		} else {
			m.status = "provider updated"
			m.lastErr = nil
		}
		return m, fetchProvidersCmd(m.client)
	case logMsg:
		line := fmt.Sprintf("[%s] %s", strings.ToUpper(msg.log.Type), strings.TrimSpace(msg.log.Payload))
		if line != "[INFO] " {
			m.logs = append(m.logs, line)
		}
		if len(m.logs) > 300 {
			m.logs = m.logs[len(m.logs)-300:]
		}
		return m, listenLogCmd(m.client, m.cfg.LogLevel)
	case logErrMsg:
		m.status = "log stream reconnecting..."
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return retryLogMsg{}
		})
	case retryLogMsg:
		return m, listenLogCmd(m.client, m.cfg.LogLevel)
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
		return m, tea.Batch(cmds...)
	case errMsg:
		m.lastErr = msg.err
		m.loading = false
	}

	var cmd tea.Cmd
	if m.tab == 6 {
		for i := range m.settingsInputs {
			if i == m.settingsFocus {
				m.settingsInputs[i].Focus()
			} else {
				m.settingsInputs[i].Blur()
			}
			m.settingsInputs[i], cmd = m.settingsInputs[i].Update(msg)
		}
	}
	if m.tab == 5 && m.importing {
		for i := 0; i < 2; i++ {
			if i == m.importFocus {
				if i == 0 {
					m.importName.Focus()
					m.importURL.Blur()
				} else {
					m.importURL.Focus()
					m.importName.Blur()
				}
			}
		}
		m.importName, _ = m.importName.Update(msg)
		m.importURL, _ = m.importURL.Update(msg)
	}
	return m, cmd
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	if m.width < 48 || m.height < 12 {
		msg := "Terminal too small. Resize to at least 48x12."
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.styles.subtle.Render(msg))
	}

	head := m.renderHeader(m.width)
	tabline := m.renderTabs(m.width)
	foot := m.renderFooter(m.width)
	headerH := lipgloss.Height(head)
	tabH := lipgloss.Height(tabline)
	footH := lipgloss.Height(foot)
	bodyHeight := m.height - headerH - tabH - footH
	bodyHeight = max(5, bodyHeight)
	m.bodyY = headerH + tabH
	m.bodyW = m.width
	m.bodyH = bodyHeight
	body := m.renderBody(m.width, bodyHeight)

	ui := lipgloss.JoinVertical(lipgloss.Left, head, tabline, body, foot)
	ui = m.styles.app.Render(ui)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, ui)
}

func (m *model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.client != nil {
			m.client.Close()
		}
		return tea.Quit
	case "tab", "right":
		m.tab = (m.tab + 1) % len(tabs)
		return nil
	case "shift+tab", "left":
		m.tab = (m.tab - 1 + len(tabs)) % len(tabs)
		return nil
	case "1", "2", "3", "4", "5", "6":
		i, _ := strconv.Atoi(msg.String())
		m.tab = i - 1
		return nil
	case "r":
		if m.tab == 0 || m.tab == 1 {
			return tea.Batch(
				setModeCmd(m.client, "rule"),
				fetchConfigCmd(m.client),
			)
		}
	case "g":
		if m.tab == 0 || m.tab == 1 {
			return tea.Batch(
				setModeCmd(m.client, "global"),
				fetchConfigCmd(m.client),
			)
		}
	case "d":
		if m.tab == 0 || m.tab == 1 {
			return tea.Batch(
				setModeCmd(m.client, "direct"),
				fetchConfigCmd(m.client),
			)
		}
	}
	return nil
}

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
		}
	case "down", "j":
		if m.proxyPane == 0 {
			m.groupCursor = clamp(m.groupCursor+1, 0, max(0, len(m.groups)-1))
			m.syncNodeCursorByGroup()
		} else {
			m.nodeCursor = clamp(m.nodeCursor+1, 0, max(0, len(m.nodes)-1))
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
			m.status = fmt.Sprintf("testing %d nodes...", len(m.nodes))
			return testAllNodesCmd(m.client, m.nodes)
		}
	}
	return nil
}

func (m *model) handleOverviewMouse(x, y int) tea.Cmd {
	for _, t := range m.overviewModeTargets {
		if t.hit(x, y) {
			return tea.Batch(setModeCmd(m.client, t.text), fetchConfigCmd(m.client))
		}
	}
	return nil
}

func (m *model) handleProxyMouse(x, y int) tea.Cmd {
	for _, t := range m.proxyModeTargets {
		if t.hit(x, y) {
			return tea.Batch(setModeCmd(m.client, t.text), fetchConfigCmd(m.client))
		}
	}
	for _, t := range m.proxyActionTarget {
		if t.hit(x, y) && t.text == "test_all" {
			if len(m.nodes) > 0 {
				m.status = fmt.Sprintf("testing %d nodes...", len(m.nodes))
				return testAllNodesCmd(m.client, m.nodes)
			}
			return nil
		}
	}
	for _, t := range m.proxyGroupTargets {
		if t.hit(x, y) {
			m.groupCursor = clamp(t.idx, 0, max(0, len(m.groups)-1))
			m.syncNodeCursorByGroup()
			return nil
		}
	}
	for _, t := range m.proxyNodeTargets {
		if t.hit(x, y) {
			m.nodeCursor = clamp(t.idx, 0, max(0, len(m.nodes)-1))
			if len(m.groups) > 0 && len(m.nodes) > 0 {
				group := m.groups[m.groupCursor]
				node := m.nodes[m.nodeCursor]
				return setProxyCmd(m.client, group, node)
			}
			return nil
		}
	}
	return nil
}

func (m *model) handleOverviewKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		m.providerCursor = clamp(m.providerCursor-1, 0, max(0, len(m.providers)-1))
	case "down", "j":
		m.providerCursor = clamp(m.providerCursor+1, 0, max(0, len(m.providers)-1))
	case "u":
		if len(m.providers) > 0 {
			return updateProviderCmd(m.client, m.providers[m.providerCursor])
		}
	}
	return nil
}

func (m *model) handleConnectionKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		m.connCursor = clamp(m.connCursor-1, 0, max(0, len(m.conns)-1))
	case "down", "j":
		m.connCursor = clamp(m.connCursor+1, 0, max(0, len(m.conns)-1))
	case "x":
		if len(m.conns) > 0 {
			return closeConnCmd(m.client, m.conns[m.connCursor].ID)
		}
	case "X":
		return closeAllConnCmd(m.client)
	}
	return nil
}

func (m *model) handleRuleKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		m.rulesOffset = clamp(m.rulesOffset-1, 0, max(0, len(m.rules)-1))
	case "down", "j":
		m.rulesOffset = clamp(m.rulesOffset+1, 0, max(0, len(m.rules)-1))
	}
	return nil
}

func (m *model) handleLogKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "c":
		m.logs = m.logs[:0]
	}
	return nil
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
		case "tab":
			m.importFocus = (m.importFocus + 1) % 2
			return nil
		case "shift+tab":
			m.importFocus = (m.importFocus + 1) % 2
			return nil
		case "enter":
			return importSubCmd(m.rt, m.importName.Value(), m.importURL.Value())
		}
		return nil
	}

	switch msg.String() {
	case "up", "k":
		m.profileCursor = clamp(m.profileCursor-1, 0, max(0, len(m.subscriptions)-1))
	case "down", "j":
		m.profileCursor = clamp(m.profileCursor+1, 0, max(0, len(m.subscriptions)-1))
	case "i":
		m.importing = true
		m.importFocus = 1
		m.importName.SetValue("")
		m.importURL.SetValue("")
	case "u":
		if len(m.subscriptions) > 0 {
			it := m.subscriptions[m.profileCursor]
			return updateProviderCmd2(m.client, it.ProviderName)
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
	return nil
}

func (m *model) applySettings() tea.Cmd {
	pollSec, err := strconv.Atoi(strings.TrimSpace(m.settingsInputs[2].Value()))
	if err != nil || pollSec < 1 {
		m.lastErr = fmt.Errorf("poll interval must be integer >=1")
		return nil
	}

	m.cfg.Endpoint = strings.TrimSpace(m.settingsInputs[0].Value())
	m.cfg.Secret = strings.TrimSpace(m.settingsInputs[1].Value())
	m.cfg.PollInterval = time.Duration(pollSec) * time.Second
	m.cfg.LogLevel = strings.TrimSpace(m.settingsInputs[3].Value())
	if m.cfg.LogLevel == "" {
		m.cfg.LogLevel = "info"
	}

	if err := config.Save(m.cfg); err != nil {
		m.lastErr = err
		m.status = "save config failed"
		return nil
	}

	if m.client != nil {
		m.client.Close()
	}
	c, err := mihomo.NewClient(m.cfg.Endpoint, m.cfg.Secret)
	if err != nil {
		m.lastErr = err
		m.status = "new client failed"
		return nil
	}
	m.client = c
	m.status = "settings saved"

	return tea.Batch(
		fetchVersionCmd(m.client),
		fetchConfigCmd(m.client),
		fetchProxiesCmd(m.client),
		fetchRulesCmd(m.client),
		fetchConnectionsCmd(m.client),
		fetchProvidersCmd(m.client),
		loadSubsCmd(m.rt),
		reloadCoreCmd(m.rt, m.cfg),
		listenLogCmd(m.client, m.cfg.LogLevel),
	)
}

func (m *model) renderHeader(w int) string {
	line := m.styles.header.Render("Clash TUI · Mihomo")
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(line)
}

func (m *model) renderTabs(w int) string {
	out := make([]string, 0, len(tabs))
	for i, t := range tabs {
		label := fmt.Sprintf("%d.%s", i+1, t)
		if i == m.tab {
			out = append(out, m.styles.tabActive.Render(label))
		} else {
			out = append(out, m.styles.tab.Render(label))
		}
	}
	line := strings.Join(out, " ")
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(line)
}

func (m *model) renderBody(bodyW, bodyH int) string {
	if m.loading {
		return m.renderPanel(bodyW, bodyH, "Loading Mihomo data...")
	}
	switch m.tab {
	case 0:
		return m.renderOverview(bodyW, bodyH)
	case 1:
		return m.renderProxies(bodyW, bodyH)
	case 2:
		return m.renderConnections(bodyW, bodyH)
	case 3:
		return m.renderRules(bodyW, bodyH)
	case 4:
		return m.renderLogs(bodyW, bodyH)
	case 5:
		return m.renderProfiles(bodyW, bodyH)
	case 6:
		return m.renderSettings(bodyW, bodyH)
	default:
		return ""
	}
}

func (m *model) renderOverview(w, h int) string {
	m.overviewModeTargets = nil

	leftX := 0
	leftY := m.bodyY
	leftContentX, leftContentY := m.panelContentOrigin(leftX, leftY)

	modeBlock := []string{
		m.styles.panelTitle.Render("Modes"),
		"",
		fmt.Sprintf("Current: %s", strings.ToUpper(m.baseCfg.Mode)),
		m.renderOverviewModeLine(leftContentX, leftContentY+3),
		m.styles.subtle.Render("Keyboard: r/g/d"),
		"",
		m.styles.panelTitle.Render("Controller"),
		fmt.Sprintf("Endpoint: %s", m.cfg.Endpoint),
		fmt.Sprintf("Version: %s", m.version),
		fmt.Sprintf("Status: %s", m.status),
		"",
		m.styles.panelTitle.Render("Runtime"),
		fmt.Sprintf("Groups: %d", len(m.groups)),
		fmt.Sprintf("Connections: %d", len(m.conns)),
		fmt.Sprintf("Rules: %d", len(m.rules)),
	}
	if m.lastErr != nil {
		modeBlock = append(modeBlock, "", m.styles.errorText.Render("Error: "+m.lastErr.Error()))
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
	leftW := w / 2
	rightW := w - leftW - 1
	left := m.renderPanel(leftW, h, strings.Join(modeBlock, "\n"))
	right := m.renderPanel(rightW, h, m.renderProviderLines())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m *model) renderProviderLines() string {
	providerLines := []string{m.styles.panelTitle.Render("Providers")}
	if len(m.providers) == 0 {
		providerLines = append(providerLines, "", "No providers")
	} else {
		providerLines = append(providerLines, m.styles.subtle.Render("u: update provider"))
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

func (m *model) renderProxies(w, h int) string {
	m.proxyModeTargets = nil
	m.proxyActionTarget = nil
	m.proxyGroupTargets = nil
	m.proxyNodeTargets = nil

	leftX, leftY, leftW, leftH, rightX, rightY, rightW, rightH := m.proxyPanels(w, h)
	leftContentX, leftContentY := m.panelContentOrigin(leftX, leftY)
	rightContentX, rightContentY := m.panelContentOrigin(rightX, rightY)

	leftLines := []string{m.styles.panelTitle.Render("Proxy Groups")}
	for i, g := range m.groups {
		line := g
		if i == m.groupCursor {
			line = m.styles.cursor.Render(line)
		}
		leftLines = append(leftLines, line)
		row := 1 + i
		m.proxyGroupTargets = append(m.proxyGroupTargets, clickTarget{
			x1:  leftContentX,
			y1:  leftContentY + row,
			x2:  leftContentX + max(1, leftW-m.styles.panel.GetHorizontalFrameSize()),
			y2:  leftContentY + row + 1,
			idx: i,
		})
	}
	if len(m.groups) == 0 {
		leftLines = append(leftLines, "No proxy groups")
	}
	if m.proxyPane == 0 {
		leftLines = append(leftLines, "", m.styles.subtle.Render("Focus: groups (h/l to switch)"))
	}

	rightLines := []string{m.styles.panelTitle.Render("Nodes")}
	modeLine := m.renderProxyModeLine(rightContentX, rightContentY+1)
	rightLines = append(rightLines, modeLine)
	actionLine := m.renderProxyActionLine(rightContentX, rightContentY+2)
	rightLines = append(rightLines, actionLine)
	group := ""
	now := ""
	if len(m.groups) > 0 {
		group = m.groups[m.groupCursor]
		now = m.proxies[group].Now
		rightLines = append(rightLines, m.styles.subtle.Render("Group: "+group))
	} else {
		rightLines = append(rightLines, m.styles.subtle.Render("Group: -"))
	}
	nodeStartRow := 4
	for i, n := range m.nodes {
		parts := []string{n}
		if d, ok := m.delayMap[n]; ok {
			parts = append(parts, fmt.Sprintf("%dms", d))
		}
		line := strings.Join(parts, "  ")
		if n == now {
			line = m.styles.selected.Render("★ " + line)
		}
		if i == m.nodeCursor {
			line = m.styles.cursor.Render(line)
		}
		rightLines = append(rightLines, line)
		row := nodeStartRow + i
		m.proxyNodeTargets = append(m.proxyNodeTargets, clickTarget{
			x1:  rightContentX,
			y1:  rightContentY + row,
			x2:  rightContentX + max(1, rightW-m.styles.panel.GetHorizontalFrameSize()),
			y2:  rightContentY + row + 1,
			idx: i,
		})
	}
	if len(m.nodes) == 0 {
		rightLines = append(rightLines, "No nodes")
	}
	if m.proxyPane == 1 {
		rightLines = append(rightLines, "", m.styles.subtle.Render("Enter: switch node   t: test one   T: test all"))
	}
	left := m.renderPanel(leftW, leftH, strings.Join(leftLines, "\n"))
	right := m.renderPanel(rightW, rightH, strings.Join(rightLines, "\n"))
	if w < 90 {
		return lipgloss.JoinVertical(lipgloss.Left, left, " ", right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m *model) renderConnections(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render("Connections"),
		m.styles.subtle.Render("x: close selected   X: close all"),
	}
	if len(m.conns) == 0 {
		lines = append(lines, "", "No active connections")
		return m.renderPanel(w, h, strings.Join(lines, "\n"))
	}
	maxRows := max(1, m.panelContentHeight(h)-2)
	start := clamp(m.connCursor-maxRows/2, 0, max(0, len(m.conns)-maxRows))
	end := min(len(m.conns), start+maxRows)
	for i := start; i < end; i++ {
		c := m.conns[i]
		host := c.Metadata.Host
		if host == "" {
			host = c.Metadata.DestinationIP
		}
		label := fmt.Sprintf("%s  %s  %s", host, c.Metadata.Process, strings.Join(c.Chains, "->"))
		if i == m.connCursor {
			label = m.styles.cursor.Render(label)
		}
		lines = append(lines, label)
	}
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderRules(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render("Rules"),
		m.styles.subtle.Render("j/k scroll"),
	}
	if len(m.rules) == 0 {
		lines = append(lines, "", "No rules")
		return m.renderPanel(w, h, strings.Join(lines, "\n"))
	}
	maxRows := max(1, m.panelContentHeight(h)-2)
	start := clamp(m.rulesOffset, 0, max(0, len(m.rules)-maxRows))
	end := min(len(m.rules), start+maxRows)
	for i := start; i < end; i++ {
		r := m.rules[i]
		lines = append(lines, fmt.Sprintf("%s  %s  -> %s", r.Type, r.Payload, r.Proxy))
	}
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderLogs(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render("Logs"),
		m.styles.subtle.Render("Streaming /logs (c to clear)"),
	}
	maxRows := max(1, m.panelContentHeight(h)-2)
	start := max(0, len(m.logs)-maxRows)
	lines = append(lines, m.logs[start:]...)
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderProfiles(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render("Profiles / Subscriptions"),
		m.styles.subtle.Render("i: import   u: update selected   U: update all"),
		"",
	}
	if len(m.subscriptions) == 0 {
		lines = append(lines, "No subscriptions. Press i to import.")
	} else {
		for i, s := range m.subscriptions {
			state := "enabled"
			if !s.Enabled {
				state = "disabled"
			}
			line := fmt.Sprintf("%s  [%s]  provider=%s", s.Name, state, s.ProviderName)
			if i == m.profileCursor {
				line = m.styles.cursor.Render(line)
			}
			lines = append(lines, line)
			if !s.LastUpdated.IsZero() {
				lines = append(lines, m.styles.subtle.Render("  updated: "+s.LastUpdated.Format(time.RFC3339)))
			}
			if s.LastError != "" {
				lines = append(lines, m.styles.errorText.Render("  error: "+s.LastError))
			}
		}
	}

	if m.importing {
		lines = append(lines, "")
		lines = append(lines, m.styles.panelTitle.Render("Import Subscription"))
		lines = append(lines, "Name (optional)")
		lines = append(lines, m.importName.View())
		lines = append(lines, "URL")
		lines = append(lines, m.importURL.View())
		lines = append(lines, m.styles.subtle.Render("Enter: confirm  Esc: cancel"))
	}

	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderSettings(w, h int) string {
	lines := []string{
		m.styles.panelTitle.Render("Settings"),
		m.styles.subtle.Render("tab/shift+tab move focus  s save and reconnect"),
		"",
		"Controller Endpoint",
		m.settingsInputs[0].View(),
		"",
		"Secret",
		m.settingsInputs[1].View(),
		"",
		"Poll Interval (seconds)",
		m.settingsInputs[2].View(),
		"",
		"Log Level (debug/info/warning/error)",
		m.settingsInputs[3].View(),
		"",
	}
	lines = append(lines, m.styles.successText.Render("Config file: ~/.config/clash-tui/config.yaml"))
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

func (m *model) renderFooter(w int) string {
	msg := m.status
	if msg == "" {
		msg = "Ready"
	}
	help := "q quit | tab switch page | 1-6 jump | mouse click tabs"
	if len(tabs) > 6 {
		help = "q quit | tab switch page | 1-7 jump | mouse click tabs"
	}
	line := m.styles.footer.Render(msg + "  |  " + help)
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(line)
}

func (m *model) rebuildGroupsAndNodes() {
	groups := make([]string, 0)
	for name, p := range m.proxies {
		if len(p.All) == 0 {
			continue
		}
		switch strings.ToLower(p.Type) {
		case "selector", "urltest", "fallback", "loadbalance":
			groups = append(groups, name)
		}
	}
	sort.Strings(groups)
	m.groups = groups
	m.groupCursor = clamp(m.groupCursor, 0, max(0, len(m.groups)-1))
	m.syncNodeCursorByGroup()
}

func (m *model) syncNodeCursorByGroup() {
	if len(m.groups) == 0 {
		m.nodes = nil
		m.nodeCursor = 0
		return
	}
	group := m.groups[m.groupCursor]
	m.nodes = append([]string{}, m.proxies[group].All...)
	m.nodeCursor = clamp(m.nodeCursor, 0, max(0, len(m.nodes)-1))
	now := m.proxies[group].Now
	for i, n := range m.nodes {
		if n == now {
			m.nodeCursor = i
			break
		}
	}
}

func (m *model) initSettingsInputs() {
	ins := make([]textinput.Model, 4)
	for i := range ins {
		ins[i] = textinput.New()
		ins[i].Prompt = "> "
		ins[i].Width = 60
	}
	ins[0].SetValue(m.cfg.Endpoint)
	ins[1].SetValue(m.cfg.Secret)
	ins[2].SetValue(fmt.Sprintf("%d", int(m.cfg.PollInterval/time.Second)))
	ins[3].SetValue(m.cfg.LogLevel)
	m.settingsInputs = ins
	m.settingsFocus = 0
	m.settingsInputs[0].Focus()
}

func (m *model) initImportInputs() {
	m.importName = textinput.New()
	m.importName.Prompt = "> "
	m.importName.Placeholder = "optional name"
	m.importName.Width = 60

	m.importURL = textinput.New()
	m.importURL.Prompt = "> "
	m.importURL.Placeholder = "https://example.com/subscription"
	m.importURL.Width = 60
}

func (m *model) resizeInputs() {
	w := max(24, m.width-10)
	for i := range m.settingsInputs {
		m.settingsInputs[i].Width = w
	}
	m.importName.Width = w
	m.importURL.Width = w
}

func (m *model) proxyPanels(w, h int) (leftX, leftY, leftW, leftH, rightX, rightY, rightW, rightH int) {
	leftX, leftY = 0, m.bodyY
	if w < 90 {
		topH := h / 2
		bottomH := h - topH - 1
		if bottomH < 4 {
			bottomH = 4
		}
		leftW = w
		leftH = topH
		rightX = 0
		rightY = m.bodyY + topH + 1
		rightW = w
		rightH = bottomH
		return
	}

	leftW = max(22, w/3)
	leftH = h
	rightX = leftW + 1
	rightY = m.bodyY
	rightW = w - leftW - 1
	rightH = h
	return
}

func (m *model) renderProxyModeLine(contentX, y int) string {
	current := strings.ToLower(m.baseCfg.Mode)
	type modeToken struct {
		label string
		mode  string
	}
	tokens := []modeToken{
		{label: "Rule", mode: "rule"},
		{label: "Global", mode: "global"},
		{label: "Direct", mode: "direct"},
	}

	prefix := "Mode: "
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
		m.proxyModeTargets = append(m.proxyModeTargets, clickTarget{
			x1:   startX,
			y1:   y,
			x2:   endX,
			y2:   y + 1,
			text: t.mode,
		})
		line += token
		cursorX = endX
		if i < len(tokens)-1 {
			line += " "
			cursorX++
		}
	}
	return line
}

func (m *model) renderOverviewModeLine(contentX, y int) string {
	current := strings.ToLower(m.baseCfg.Mode)
	type modeToken struct {
		label string
		mode  string
	}
	tokens := []modeToken{
		{label: "Rule", mode: "rule"},
		{label: "Global", mode: "global"},
		{label: "Direct", mode: "direct"},
	}

	prefix := "Switch: "
	line := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	for i, t := range tokens {
		token := "[○ " + t.label + "]"
		if t.mode == current {
			token = "[✓ " + strings.ToUpper(t.label) + "]"
		}
		startX := cursorX
		endX := startX + lipgloss.Width(token)
		m.overviewModeTargets = append(m.overviewModeTargets, clickTarget{
			x1:   startX,
			y1:   y,
			x2:   endX,
			y2:   y + 1,
			text: t.mode,
		})
		line += token
		cursorX = endX
		if i < len(tokens)-1 {
			line += " "
			cursorX++
		}
	}
	return line
}

func (m *model) renderProxyActionLine(contentX, y int) string {
	prefix := "Action: "
	button := "[⚡ Test All]"
	startX := contentX + lipgloss.Width(prefix)
	endX := startX + lipgloss.Width(button)
	m.proxyActionTarget = append(m.proxyActionTarget, clickTarget{
		x1:   startX,
		y1:   y,
		x2:   endX,
		y2:   y + 1,
		text: "test_all",
	})
	return prefix + button
}

func (m *model) panelContentHeight(outerHeight int) int {
	return max(1, outerHeight-m.styles.panel.GetVerticalFrameSize())
}

func (m *model) panelContentOrigin(panelX, panelY int) (x, y int) {
	return panelX + 2, panelY + 1
}

func (m *model) renderPanel(outerW, outerH int, content string) string {
	cw := max(1, outerW-m.styles.panel.GetHorizontalFrameSize())
	ch := max(1, outerH-m.styles.panel.GetVerticalFrameSize())
	return m.styles.panel.
		Width(cw).
		Height(ch).
		MaxWidth(cw).
		MaxHeight(ch).
		Render(content)
}

func (m *model) tabHit(x int) int {
	pos := 0
	for i, t := range tabs {
		label := fmt.Sprintf("%d.%s", i+1, t)
		w := lipgloss.Width(label) + 2
		if x >= pos && x < pos+w {
			return i
		}
		pos += w + 1
	}
	return -1
}

func pollCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchVersionCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return errMsg{fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		v, err := c.GetVersion(ctx)
		if err != nil {
			return errMsg{err}
		}
		return versionMsg{v}
	}
}

func fetchConfigCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return errMsg{fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		v, err := c.GetConfig(ctx)
		if err != nil {
			return errMsg{err}
		}
		return cfgMsg{v}
	}
}

func fetchProxiesCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return errMsg{fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		v, err := c.GetProxies(ctx)
		if err != nil {
			return errMsg{err}
		}
		return proxiesMsg{v}
	}
}

func fetchRulesCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return errMsg{fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		v, err := c.GetRules(ctx)
		if err != nil {
			return errMsg{err}
		}
		return rulesMsg{v}
	}
}

func fetchConnectionsCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return errMsg{fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		v, err := c.GetConnections(ctx)
		if err != nil {
			return errMsg{err}
		}
		return connectionsMsg{v}
	}
}

func fetchProvidersCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return errMsg{fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		v, err := c.GetProxyProviders(ctx)
		if err != nil {
			return errMsg{err}
		}
		return providersMsg{v}
	}
}

func setModeCmd(c *mihomo.Client, mode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.SetMode(ctx, mode)
		return modeSetMsg{mode: mode, err: err}
	}
}

func setProxyCmd(c *mihomo.Client, group, node string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		err := c.SelectProxy(ctx, group, node)
		return proxySetMsg{group: group, node: node, err: err}
	}
}

func testDelayCmd(c *mihomo.Client, node string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		resp, err := c.TestProxyDelay(ctx, node, 5000)
		if err != nil {
			return delayMsg{name: node, err: err}
		}
		return delayMsg{name: node, delay: resp.Delay}
	}
}

func testAllNodesCmd(c *mihomo.Client, nodes []string) tea.Cmd {
	if len(nodes) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(nodes))
	for _, n := range nodes {
		cmds = append(cmds, testDelayCmd(c, n))
	}
	return tea.Batch(cmds...)
}

func closeConnCmd(c *mihomo.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.CloseConnection(ctx, id)
		return connClosedMsg{id: id, err: err}
	}
}

func closeAllConnCmd(c *mihomo.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		err := c.CloseAllConnections(ctx)
		return connClosedMsg{id: "", err: err}
	}
}

func updateProviderCmd(c *mihomo.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := c.UpdateProxyProvider(ctx, name)
		return providersUpdatedMsg{name: name, err: err}
	}
}

func updateProviderCmd2(c *mihomo.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return updateSubMsg{name: name, err: fmt.Errorf("mihomo client unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		err := c.UpdateProxyProvider(ctx, name)
		return updateSubMsg{name: name, err: err}
	}
}

func loadSubsCmd(rt *runtime.Manager) tea.Cmd {
	return func() tea.Msg {
		if rt == nil {
			return subsMsg{items: []subscription.Item{}}
		}
		items, err := rt.Subscriptions().Load()
		if err != nil {
			return errMsg{err}
		}
		return subsMsg{items: items}
	}
}

func importSubCmd(rt *runtime.Manager, name, rawURL string) tea.Cmd {
	return func() tea.Msg {
		if rt == nil {
			return importSubMsg{err: fmt.Errorf("runtime unavailable")}
		}
		item, err := rt.Subscriptions().Import(name, rawURL)
		return importSubMsg{item: item, err: err}
	}
}

func reloadCoreCmd(rt *runtime.Manager, cfg config.Settings) tea.Cmd {
	return func() tea.Msg {
		if rt == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := rt.ReloadCore(ctx, cfg); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func listenLogCmd(c *mihomo.Client, level string) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return logErrMsg{err: fmt.Errorf("client nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ev, err := c.NextLog(ctx, level)
		if err != nil {
			return logErrMsg{err: err}
		}
		return logMsg{log: ev}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

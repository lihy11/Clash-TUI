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

var tabs = []string{"Dashboard", "Network", "Profiles", "System"}
var tabIcons = []string{"󰕮", "󰤨", "󰈯", "󰒓"}
var networkTabs = []string{"Proxies", "Rules", "Connections"}
var systemTabs = []string{"Logs", "Settings"}

const defaultThemeIndex = 1 // Gruvbox

type notifyItem struct {
	At    time.Time
	Level string
	Text  string
}

type paletteAction struct {
	ID    string
	Title string
}

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
type featureSetMsg struct {
	name    string
	enabled bool
	err     error
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
type deleteSubMsg struct {
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

	tab        int
	networkTab int
	systemTab  int
	themeIndex int

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
	txRate  float64
	rxRate  float64
	lastUp  int64
	lastDn  int64
	lastIO  time.Time

	providers      []string
	providersState map[string]mihomo.Provider

	delayMap map[string]int
	logs     []string

	status        string
	lastErr       error
	loading       bool
	pollCounter   int
	notifications []notifyItem
	statusMini    []int
	systemProxyOn bool

	proxyPane      int
	groupCursor    int
	nodeCursor     int
	nodeOffset     int
	nodePageSize   int
	currentGroup   string
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
	paletteOpen    bool
	paletteInput   textinput.Model
	paletteCursor  int

	bodyY int
	bodyW int
	bodyH int

	mainTabTargets        []clickTarget
	networkTabTargets     []clickTarget
	systemTabTargets      []clickTarget
	profileImportTargets  []clickTarget
	overviewModeTargets   []clickTarget
	overviewToggleTargets []clickTarget
	proxyModeTargets      []clickTarget
	proxyActionTarget     []clickTarget
	proxyGroupTargets     []clickTarget
	proxyNodeTargets      []clickTarget
}

func NewModel(cfg config.Settings, rt *runtime.Manager) *model {
	client, err := mihomo.NewClient(cfg.Endpoint, cfg.Secret)
	m := &model{
		styles:         defaultStyles(defaultThemeIndex),
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
		networkTab:     0,
		systemTab:      0,
		themeIndex:     defaultThemeIndex,
		notifications:  make([]notifyItem, 0, 30),
		statusMini:     make([]int, 0, 60),
		nodePageSize:   12,
	}
	m.initSettingsInputs()
	m.initImportInputs()
	m.initPaletteInput()
	m.pushNote("info", m.status)
	if err != nil {
		m.lastErr = err
		m.setStatus("init client failed")
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
	var deferredCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		return m, nil
	case tea.MouseMsg:
		if m.tab == 1 && m.networkTab == 0 {
			if msg.Button == tea.MouseButtonWheelUp {
				if len(m.nodes) > 0 {
					m.proxyPane = 1
					m.nodeCursor = clamp(m.nodeCursor-1, 0, max(0, len(m.nodes)-1))
					m.ensureNodeVisible()
				}
				return m, nil
			}
			if msg.Button == tea.MouseButtonWheelDown {
				if len(m.nodes) > 0 {
					m.proxyPane = 1
					m.nodeCursor = clamp(m.nodeCursor+1, 0, max(0, len(m.nodes)-1))
					m.ensureNodeVisible()
				}
				return m, nil
			}
		}
		if msg.Action == tea.MouseActionPress {
			for _, t := range m.mainTabTargets {
				if t.hit(msg.X, msg.Y) {
					m.tab = t.idx
					return m, nil
				}
			}
			if m.tab == 1 {
				for _, t := range m.networkTabTargets {
					if t.hit(msg.X, msg.Y) {
						m.networkTab = t.idx
						return m, nil
					}
				}
			}
			if m.tab == 3 {
				for _, t := range m.systemTabTargets {
					if t.hit(msg.X, msg.Y) {
						m.systemTab = t.idx
						return m, nil
					}
				}
			}
			if m.tab == 0 {
				if cmd := m.handleOverviewMouse(msg.X, msg.Y); cmd != nil {
					return m, cmd
				}
			} else if m.tab == 1 && m.networkTab == 0 {
				if cmd := m.handleProxyMouse(msg.X, msg.Y); cmd != nil {
					return m, cmd
				}
			} else if m.tab == 2 && m.importing {
				if cmd := m.handleProfilesMouse(msg.X, msg.Y); cmd != nil {
					return m, cmd
				}
			}
		}
	case tea.KeyMsg:
		if m.paletteOpen {
			return m, m.handlePaletteKeys(msg)
		}
		if m.tab == 2 && m.importing {
			deferredCmd = m.handleProfilesKeys(msg)
		} else {
			if cmd := m.handleGlobalKeys(msg); cmd != nil {
				return m, cmd
			}
		}
		switch m.tab {
		case 0:
			return m, m.handleOverviewKeys(msg)
		case 1:
			switch m.networkTab {
			case 0:
				return m, m.handleProxyKeys(msg)
			case 1:
				return m, m.handleRuleKeys(msg)
			case 2:
				return m, m.handleConnectionKeys(msg)
			}
		case 2:
			if !m.importing {
				return m, m.handleProfilesKeys(msg)
			}
		case 3:
			if m.systemTab == 0 {
				return m, m.handleLogKeys(msg)
			}
			deferredCmd = m.handleSettingsKeys(msg)
		}
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
		return m, tea.Batch(cmds...)
	case updateSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("update subscription failed")
		} else {
			m.setStatus("subscription updated")
			m.lastErr = nil
		}
		return m, loadSubsCmd(m.rt)
	case deleteSubMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("delete subscription failed")
			return m, nil
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
		return m, tea.Batch(cmds...)
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
		return m, fetchConfigCmd(m.client)
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
		return m, fetchProxiesCmd(m.client)
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
		return m, fetchConnectionsCmd(m.client)
	case providersUpdatedMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.setStatus("provider update failed")
		} else {
			m.setStatus("provider updated")
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
		m.setStatus("log stream reconnecting...")
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
		return m, nil
	}
	return m, tea.Batch(cmds...)
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
	headerH := lipgloss.Height(head)
	tabline := m.renderTabs(m.width, headerH)
	foot := m.renderFooter(m.width)
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
	base := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, ui)
	if m.paletteOpen {
		overlay := m.renderPalette(max(54, m.width/2), max(10, m.height/2))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}
	return base
}

func (m *model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
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
		m.cycleTheme()
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

func (m *model) handleProxyMouse(x, y int) tea.Cmd {
	for _, t := range m.proxyModeTargets {
		if t.hit(x, y) {
			return tea.Batch(setModeCmd(m.client, t.text), fetchConfigCmd(m.client))
		}
	}
	for _, t := range m.proxyActionTarget {
		if t.hit(x, y) && t.text == "test_all" {
			if len(m.nodes) > 0 {
				m.setStatus(fmt.Sprintf("testing %d nodes...", len(m.nodes)))
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
			m.ensureNodeVisible()
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
	case "s":
		return setSystemProxyCmd(m.client, !m.systemProxyEnabled())
	case "n":
		return setTunCmd(m.client, !m.baseCfg.Tun.Enable)
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
	case "x":
		if len(m.subscriptions) > 0 {
			it := m.subscriptions[m.profileCursor]
			return deleteSubCmd(m.rt, it.ProviderName)
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
		m.setStatus("save config failed")
		return nil
	}

	if m.client != nil {
		m.client.Close()
	}
	c, err := mihomo.NewClient(m.cfg.Endpoint, m.cfg.Secret)
	if err != nil {
		m.lastErr = err
		m.setStatus("new client failed")
		return nil
	}
	m.client = c
	m.setStatus("settings saved")

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
	title := "Clash-TUI"
	mode := strings.ToUpper(strings.TrimSpace(m.baseCfg.Mode))
	if mode == "" {
		mode = "RULE"
	}
	controller := strings.TrimPrefix(strings.TrimPrefix(m.cfg.Endpoint, "http://"), "https://")
	if controller == "" {
		controller = "127.0.0.1:9090"
	}
	conn := "CONNECTED"
	if m.lastErr != nil {
		conn = "DEGRADED"
	}
	connIcon := "●"
	if m.lastErr != nil {
		connIcon = "○"
	}
	up := formatRateFixed(m.txRate)
	down := formatRateFixed(m.rxRate)
	right1 := fmt.Sprintf("%s %s | mode %s | v%s", connIcon, conn, mode, m.version)
	right2 := fmt.Sprintf("↑ %s ↓ %s | core %s | theme %s", up, down, controller, themes[m.themeIndex].Name)
	innerW := max(1, w-m.styles.statusBar.GetHorizontalFrameSize())
	line1 := composeHeaderLine(title, right1, innerW)
	line2 := composeHeaderLine("", right2, innerW)
	top := m.styles.statusBar.Width(innerW).MaxWidth(innerW).Render(line1)
	bottom := m.styles.statusBar.Width(innerW).MaxWidth(innerW).Render(line2)
	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m *model) renderTabs(w, y int) string {
	m.mainTabTargets = nil
	out := make([]string, 0, len(tabs))
	cursorX := 0
	for i, t := range tabs {
		icon := ""
		if i < len(tabIcons) {
			icon = tabIcons[i] + " "
		}
		label := fmt.Sprintf("%s%s", icon, t)
		var token string
		if i == m.tab {
			token = m.styles.tabActive.Render(" " + label + " ")
		} else {
			token = m.styles.tab.Render(" " + label + " ")
		}
		wToken := lipgloss.Width(token)
		m.mainTabTargets = append(m.mainTabTargets, clickTarget{
			x1:  cursorX,
			y1:  y,
			x2:  cursorX + wToken,
			y2:  y + 1,
			idx: i,
		})
		cursorX += wToken
		out = append(out, token)
	}
	line := strings.Join(out, "")
	return m.styles.mainTabBar.Width(w).MaxWidth(w).Render(line)
}

func (m *model) renderNetwork(w, h int) string {
	sub := m.renderSubTabs("NETWORK", networkTabs, m.networkTab, m.bodyY, &m.networkTabTargets)
	subW := max(1, w-m.styles.subTabBar.GetHorizontalFrameSize())
	sub = m.styles.subTabBar.Width(subW).MaxWidth(subW).Render(sub)
	bodyH := max(4, h-lipgloss.Height(sub)-1)
	var body string
	switch m.networkTab {
	case 0:
		body = m.renderProxies(w, bodyH)
	case 1:
		body = m.renderRules(w, bodyH)
	default:
		body = m.renderConnections(w, bodyH)
	}
	return lipgloss.JoinVertical(lipgloss.Left, sub, body)
}

func (m *model) renderSystem(w, h int) string {
	sub := m.renderSubTabs("SYSTEM", systemTabs, m.systemTab, m.bodyY, &m.systemTabTargets)
	subW := max(1, w-m.styles.subTabBar.GetHorizontalFrameSize())
	sub = m.styles.subTabBar.Width(subW).MaxWidth(subW).Render(sub)
	bodyH := max(4, h-lipgloss.Height(sub)-1)
	if m.systemTab == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, sub, m.renderLogs(w, bodyH))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sub, m.renderSettings(w, bodyH))
}

func (m *model) renderSubTabs(section string, items []string, active, y int, targets *[]clickTarget) string {
	*targets = nil
	out := make([]string, 0, len(items)+2)
	sectionToken := m.styles.sectionLabel.Render(section)
	out = append(out, sectionToken, m.styles.subtle.Render("│"))
	cursorX := m.styles.subTabBar.GetHorizontalFrameSize()/2 + lipgloss.Width(sectionToken) + 1
	for i, t := range items {
		var token string
		if i == active {
			token = m.styles.subTabActive.Render(t)
		} else {
			token = m.styles.subTab.Render(t)
		}
		wToken := lipgloss.Width(token)
		*targets = append(*targets, clickTarget{
			x1:  cursorX,
			y1:  y,
			x2:  cursorX + wToken,
			y2:  y + 1,
			idx: i,
		})
		cursorX += wToken
		out = append(out, token)
	}
	return strings.Join(out, "")
}

func (m *model) renderBody(bodyW, bodyH int) string {
	if m.loading {
		return m.renderPanel(bodyW, bodyH, "Loading Mihomo data...")
	}
	switch m.tab {
	case 0:
		return m.renderOverview(bodyW, bodyH)
	case 1:
		return m.renderNetwork(bodyW, bodyH)
	case 2:
		return m.renderProfiles(bodyW, bodyH)
	case 3:
		return m.renderSystem(bodyW, bodyH)
	default:
		return ""
	}
}

func (m *model) renderOverview(w, h int) string {
	m.overviewModeTargets = nil
	m.overviewToggleTargets = nil

	leftX := 0
	leftY := m.bodyY
	leftContentX, leftContentY := m.panelContentOrigin(leftX, leftY)
	modeBlock := make([]string, 0, 28)
	row := 0
	addLine := func(s string) {
		modeBlock = append(modeBlock, s)
		row += max(1, lipgloss.Height(s))
	}

	addLine(m.styles.panelTitle.Render("Dashboard"))
	addLine("")
	addLine(fmt.Sprintf("Mode: %s", strings.ToUpper(m.baseCfg.Mode)))
	addLine(m.renderOverviewModeLine(leftContentX, leftContentY+row))
	addLine(m.styles.subtle.Render("Keyboard: r/g/d"))
	addLine("")
	addLine(m.styles.panelTitle.Render("Runtime"))
	addLine(fmt.Sprintf("Endpoint: %s", m.cfg.Endpoint))
	addLine(fmt.Sprintf("Version: %s", m.version))
	addLine(fmt.Sprintf("Theme: %s", themes[m.themeIndex].Name))
	addLine(m.renderOverviewToggleLine(leftContentX, leftContentY+row))
	addLine(m.styles.subtle.Render("s: toggle system-proxy   n: toggle tun"))
	if !m.baseCfg.HasSystemProxy {
		addLine(m.styles.subtle.Render("system-proxy state is tracked locally (core /configs does not expose it)"))
	}
	addLine("")
	addLine(fmt.Sprintf("Groups: %d", len(m.groups)))
	addLine(fmt.Sprintf("Connections: %d", len(m.conns)))
	addLine(fmt.Sprintf("Rules: %d", len(m.rules)))
	addLine("")
	addLine(m.styles.panelTitle.Render("Mini Trend"))
	addLine(m.styles.subtle.Render(m.sparkline()))
	addLine("")
	addLine(m.styles.panelTitle.Render("Quick Actions"))
	addLine(m.styles.action.Render(" : Command Palette "))
	addLine("F2: Cycle Theme")
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
	right := m.renderPanel(rightW, h, m.renderNotificationCenter())
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
		typ := strings.ToUpper(strings.TrimSpace(m.proxies[g].Type))
		if typ == "" {
			typ = "-"
		}
		line := fmt.Sprintf("%s  ·  %s", g, typ)
		if i == m.groupCursor {
			line = m.styles.cursor.Render("┃ " + line)
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
	left := m.renderPanelFocused(leftW, leftH, strings.Join(leftLines, "\n"), m.proxyPane == 0)
	right := m.renderPanelFocused(rightW, rightH, strings.Join(rightLines, "\n"), m.proxyPane == 1)
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
	contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
	hostW := max(16, contentW/4)
	procW := max(12, contentW/6)
	chainW := max(20, contentW-hostW-procW-4)
	for i := start; i < end; i++ {
		c := m.conns[i]
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
		label := fmt.Sprintf("%s  %s  %s",
			fitTextWidth(host, hostW),
			fitTextWidth(process, procW),
			fitTextWidth(chain, chainW),
		)
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
	contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
	typeW := max(12, contentW/6)
	proxyW := max(14, contentW/5)
	payloadW := max(16, contentW-typeW-proxyW-4)
	for i := start; i < end; i++ {
		r := m.rules[i]
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			fitTextWidth(r.Type, typeW),
			fitTextWidth(r.Payload, payloadW),
			fitTextWidth("-> "+r.Proxy, proxyW),
		))
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
	m.profileImportTargets = nil

	lines := []string{
		m.styles.panelTitle.Render("Profiles / Subscriptions"),
		m.styles.subtle.Render("i: import   x: delete selected   u: update selected   U: update all"),
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
		contentX, contentY := m.panelContentOrigin(0, m.bodyY)
		contentW := max(1, w-m.styles.panel.GetHorizontalFrameSize())
		base := len(lines)
		lines = append(lines, "")
		lines = append(lines, m.styles.panelTitle.Render("Import Subscription"))
		lines = append(lines, "Name (optional)")
		lines = append(lines, m.importName.View())
		lines = append(lines, "URL")
		lines = append(lines, m.importURL.View())
		lines = append(lines, m.styles.subtle.Render("Enter: confirm  Esc: cancel"))
		m.profileImportTargets = append(m.profileImportTargets,
			clickTarget{
				x1:  contentX,
				y1:  contentY + base + 3,
				x2:  contentX + contentW,
				y2:  contentY + base + 4,
				idx: 0,
			},
			clickTarget{
				x1:  contentX,
				y1:  contentY + base + 5,
				x2:  contentX + contentW,
				y2:  contentY + base + 6,
				idx: 1,
			},
		)
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
	left := "[Ctrl+K] Command Palette  [1-4] Switch Tab  [q] Quit"
	switch m.tab {
	case 0:
		left = "[r g d] Mode  [s] Toggle System Proxy  [n] Toggle TUN  [Ctrl+K] Palette"
	case 1:
		if m.networkTab == 0 {
			left = "[↑↓/j k] Move  [Enter] Select  [t/T] Test Delay  [r g d] Mode"
		} else if m.networkTab == 2 {
			left = "[↑↓/j k] Move  [x] Close  [X] Close All"
		}
	case 2:
		left = "[i] Import  [u] Update Selected  [U] Update All"
	case 3:
		if m.systemTab == 1 {
			left = "[Tab] Next Field  [Shift+Tab] Prev Field  [s] Save"
		} else {
			left = "[c] Clear Logs  [Tab] Switch"
		}
	}
	msg := strings.TrimSpace(m.status)
	if msg == "" {
		msg = "ready"
	}
	innerW := max(1, w-m.styles.footer.GetHorizontalFrameSize())
	line := composeHeaderLine(left, "📝 "+msg, innerW)
	line = m.styles.footer.Width(innerW).MaxWidth(innerW).Render(line)
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(line)
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

func (m *model) renderNotificationCenter() string {
	lines := []string{m.styles.notifTitle.Render("Notifications")}
	if len(m.notifications) == 0 {
		lines = append(lines, "", "No recent events")
		return strings.Join(lines, "\n")
	}
	start := max(0, len(m.notifications)-12)
	for _, n := range m.notifications[start:] {
		ts := n.At.Format("15:04:05")
		line := fmt.Sprintf("[%s] %s", ts, n.Text)
		if n.Level == "error" {
			line = m.styles.errorText.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *model) cycleTheme() {
	m.themeIndex = (m.themeIndex + 1) % len(themes)
	m.styles = defaultStyles(m.themeIndex)
	m.pushNote("info", "theme switched to "+themes[m.themeIndex].Name)
}

func (m *model) paletteActions() []paletteAction {
	return []paletteAction{
		{ID: "goto_dashboard", Title: "Go: Dashboard"},
		{ID: "goto_network", Title: "Go: Network"},
		{ID: "goto_profiles", Title: "Go: Profiles"},
		{ID: "goto_system", Title: "Go: System"},
		{ID: "network_proxies", Title: "Network: Proxies"},
		{ID: "network_rules", Title: "Network: Rules"},
		{ID: "network_connections", Title: "Network: Connections"},
		{ID: "system_logs", Title: "System: Logs"},
		{ID: "system_settings", Title: "System: Settings"},
		{ID: "mode_rule", Title: "Set Mode: Rule"},
		{ID: "mode_global", Title: "Set Mode: Global"},
		{ID: "mode_direct", Title: "Set Mode: Direct"},
		{ID: "toggle_system_proxy", Title: "Toggle: System Proxy"},
		{ID: "toggle_tun", Title: "Toggle: TUN Mode"},
		{ID: "test_all", Title: "Proxies: Test All Nodes"},
		{ID: "update_all_subs", Title: "Profiles: Update All Subscriptions"},
		{ID: "theme_cycle", Title: "Theme: Cycle"},
	}
}

func (m *model) filteredPaletteActions() []paletteAction {
	all := m.paletteActions()
	q := strings.ToLower(strings.TrimSpace(m.paletteInput.Value()))
	if q == "" {
		return all
	}
	out := make([]paletteAction, 0, len(all))
	for _, a := range all {
		if strings.Contains(strings.ToLower(a.Title), q) {
			out = append(out, a)
		}
	}
	return out
}

func (m *model) runPaletteAction(id string) tea.Cmd {
	switch id {
	case "goto_dashboard":
		m.tab = 0
	case "goto_network":
		m.tab = 1
	case "goto_profiles":
		m.tab = 2
	case "goto_system":
		m.tab = 3
	case "network_proxies":
		m.tab = 1
		m.networkTab = 0
	case "network_rules":
		m.tab = 1
		m.networkTab = 1
	case "network_connections":
		m.tab = 1
		m.networkTab = 2
	case "system_logs":
		m.tab = 3
		m.systemTab = 0
	case "system_settings":
		m.tab = 3
		m.systemTab = 1
	case "mode_rule":
		return tea.Batch(setModeCmd(m.client, "rule"), fetchConfigCmd(m.client))
	case "mode_global":
		return tea.Batch(setModeCmd(m.client, "global"), fetchConfigCmd(m.client))
	case "mode_direct":
		return tea.Batch(setModeCmd(m.client, "direct"), fetchConfigCmd(m.client))
	case "toggle_system_proxy":
		return setSystemProxyCmd(m.client, !m.systemProxyEnabled())
	case "toggle_tun":
		return setTunCmd(m.client, !m.baseCfg.Tun.Enable)
	case "test_all":
		if len(m.nodes) > 0 {
			return testAllNodesCmd(m.client, m.nodes)
		}
	case "update_all_subs":
		if len(m.subscriptions) > 0 {
			cmds := make([]tea.Cmd, 0, len(m.subscriptions))
			for _, it := range m.subscriptions {
				cmds = append(cmds, updateProviderCmd2(m.client, it.ProviderName))
			}
			return tea.Batch(cmds...)
		}
	case "theme_cycle":
		m.cycleTheme()
	}
	return nil
}

func (m *model) renderPalette(w, h int) string {
	actions := m.filteredPaletteActions()
	lines := []string{
		m.styles.panelTitle.Render("Command Palette"),
		m.paletteInput.View(),
		"",
	}
	maxRows := max(1, h-5)
	for i, a := range actions {
		if i >= maxRows {
			break
		}
		line := a.Title
		if i == m.paletteCursor {
			line = m.styles.cursor.Render(line)
		}
		lines = append(lines, line)
	}
	if len(actions) == 0 {
		lines = append(lines, m.styles.subtle.Render("No matched actions"))
	}
	return m.styles.overlay.Width(w).Render(strings.Join(lines, "\n"))
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
		m.nodeOffset = 0
		m.currentGroup = ""
		return
	}
	group := m.groups[m.groupCursor]
	prevGroup := m.currentGroup
	prevNode := ""
	if prevGroup == group && m.nodeCursor >= 0 && m.nodeCursor < len(m.nodes) {
		prevNode = m.nodes[m.nodeCursor]
	}
	m.nodes = append([]string{}, m.proxies[group].All...)
	if prevGroup != group {
		now := m.proxies[group].Now
		m.nodeCursor = 0
		for i, n := range m.nodes {
			if n == now {
				m.nodeCursor = i
				break
			}
		}
	} else if prevNode != "" {
		found := -1
		for i, n := range m.nodes {
			if n == prevNode {
				found = i
				break
			}
		}
		if found >= 0 {
			m.nodeCursor = found
		} else {
			m.nodeCursor = clamp(m.nodeCursor, 0, max(0, len(m.nodes)-1))
		}
	} else {
		m.nodeCursor = clamp(m.nodeCursor, 0, max(0, len(m.nodes)-1))
	}
	m.currentGroup = group
	m.ensureNodeVisible()
}

func (m *model) ensureNodeVisible() {
	page := max(1, m.nodePageSize)
	m.nodeCursor = clamp(m.nodeCursor, 0, max(0, len(m.nodes)-1))
	maxOffset := max(0, len(m.nodes)-page)
	m.nodeOffset = clamp(m.nodeOffset, 0, maxOffset)
	if len(m.nodes) == 0 {
		m.nodeOffset = 0
		return
	}
	if m.nodeCursor < m.nodeOffset {
		m.nodeOffset = m.nodeCursor
	}
	if m.nodeCursor >= m.nodeOffset+page {
		m.nodeOffset = m.nodeCursor - page + 1
	}
	m.nodeOffset = clamp(m.nodeOffset, 0, maxOffset)
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

func (m *model) initPaletteInput() {
	m.paletteInput = textinput.New()
	m.paletteInput.Prompt = "> "
	m.paletteInput.Placeholder = "search action..."
	m.paletteInput.Width = 48
}

func (m *model) setStatus(s string) {
	m.status = s
	level := "info"
	if strings.Contains(strings.ToLower(s), "failed") || strings.Contains(strings.ToLower(s), "error") {
		level = "error"
	}
	m.pushNote(level, s)
}

func (m *model) pushNote(level, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	m.notifications = append(m.notifications, notifyItem{
		At:    time.Now(),
		Level: level,
		Text:  text,
	})
	if len(m.notifications) > 40 {
		m.notifications = m.notifications[len(m.notifications)-40:]
	}
	val := len(m.conns)
	m.statusMini = append(m.statusMini, val)
	if len(m.statusMini) > 48 {
		m.statusMini = m.statusMini[len(m.statusMini)-48:]
	}
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
	// Network page has a one-line sub-tab row before proxy panels.
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

func (m *model) renderOverviewToggleLine(contentX, y int) string {
	systemOn := m.systemProxyEnabled()
	tunOn := m.baseCfg.Tun.Enable

	type toggleToken struct {
		id    string
		label string
		on    bool
	}
	tokens := []toggleToken{
		{id: "system-proxy", label: "System Proxy", on: systemOn},
		{id: "tun", label: "TUN Mode", on: tunOn},
	}

	prefix := "Toggles: "
	line := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	for i, t := range tokens {
		label := m.styles.subTab.Render(t.label)
		sw := m.renderWebToggle(t.on)
		token := label + " " + sw
		line += token

		wToken := lipgloss.Width(token)
		m.overviewToggleTargets = append(m.overviewToggleTargets, clickTarget{
			x1:   cursorX,
			y1:   y,
			x2:   cursorX + wToken,
			y2:   y + 1,
			text: fmt.Sprintf("%s:%s", t.id, map[bool]string{true: "off", false: "on"}[t.on]),
		})
		cursorX += wToken
		if i < len(tokens)-1 {
			sep := "   "
			line += sep
			cursorX += lipgloss.Width(sep)
		}
	}
	return line
}

func (m *model) renderWebToggle(on bool) string {
	t := themes[m.themeIndex]
	if on {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.CursorText)).
			Background(lipgloss.Color(t.Success)).
			Padding(0, 1).
			Render("ON  ●")
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.PillText)).
		Background(lipgloss.Color(t.Panel)).
		Padding(0, 1).
		Render("○  OFF")
}

func (m *model) systemProxyEnabled() bool {
	if m.baseCfg.HasSystemProxy {
		return m.baseCfg.SystemProxy
	}
	return m.systemProxyOn
}

func (m *model) renderProxyActionLine(contentX, y int) string {
	prefix := "Action: "
	// 使用 styles.action 渲染按钮，让它看起来像一个真实的按钮块
	button := m.styles.action.Render(" ⚡ Test All (T) ")

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
	return panelX + 3, panelY + 2
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

func (m *model) renderPanelFocused(outerW, outerH int, content string, focused bool) string {
	cw := max(1, outerW-m.styles.panel.GetHorizontalFrameSize())
	ch := max(1, outerH-m.styles.panel.GetVerticalFrameSize())
	st := m.styles.panel
	if focused {
		st = st.BorderForeground(lipgloss.Color(themes[m.themeIndex].Primary))
	} else {
		st = st.BorderForeground(lipgloss.Color(themes[m.themeIndex].Panel))
	}
	return st.
		Width(cw).
		Height(ch).
		MaxWidth(cw).
		MaxHeight(ch).
		Render(content)
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
	var dot string
	var bars int
	switch {
	case delay < 100:
		dot, bars = "●", 2
	case delay <= 300:
		dot, bars = "●", 4
	default:
		dot, bars = "●", 6
	}
	bar := strings.Repeat("▮", bars)
	return fitTextWidth(fmt.Sprintf("%s %-6s %4dms", dot, bar, delay), cellW)
}

func (m *model) updateThroughput(conns []mihomo.Connection) {
	var up, dn int64
	for _, c := range conns {
		up += c.Upload
		dn += c.Download
	}
	now := time.Now()
	if m.lastIO.IsZero() {
		m.lastUp = up
		m.lastDn = dn
		m.lastIO = now
		return
	}
	dt := now.Sub(m.lastIO).Seconds()
	if dt <= 0 {
		return
	}
	du := up - m.lastUp
	dd := dn - m.lastDn
	if du < 0 {
		du = 0
	}
	if dd < 0 {
		dd = 0
	}
	m.txRate = float64(du) / dt
	m.rxRate = float64(dd) / dt
	m.lastUp = up
	m.lastDn = dn
	m.lastIO = now
}

func formatRate(v float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	u := 0
	for v >= 1024 && u < len(units)-1 {
		v /= 1024
		u++
	}
	if u == 0 {
		return fmt.Sprintf("%.0f %s", v, units[u])
	}
	return fmt.Sprintf("%.1f %s", v, units[u])
}

func formatRateFixed(v float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	u := 0
	for v >= 1024 && u < len(units)-1 {
		v /= 1024
		u++
	}
	if v > 9999 {
		v = 9999
	}
	// Fixed width: value 6 chars + unit 4 chars.
	return fmt.Sprintf("%6.1f %-4s", v, units[u])
}

func truncateByWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	ellipsis := "…"
	maxW := w - lipgloss.Width(ellipsis)
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if cur+rw > maxW {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	return b.String() + ellipsis
}

func fitTextWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	out := truncateByWidth(s, w)
	cur := lipgloss.Width(out)
	if cur < w {
		out += strings.Repeat(" ", w-cur)
	}
	return out
}

func composeHeaderLine(left, right string, w int) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if w <= 0 {
		return ""
	}
	gap := 2
	leftW := lipgloss.Width(left)
	if left == "" {
		gap = 0
	}
	rightW := max(0, w-leftW-gap)
	if leftW >= w {
		return truncateByWidth(left, w)
	}
	right = truncateByWidth(right, rightW)
	if left == "" {
		return lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(right)
	}
	return left + strings.Repeat(" ", gap) + lipgloss.NewStyle().Width(rightW).Align(lipgloss.Right).Render(right)
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

func setSystemProxyCmd(c *mihomo.Client, enable bool) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return featureSetMsg{name: "system-proxy", enabled: enable, err: fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.SetSystemProxy(ctx, enable)
		return featureSetMsg{name: "system-proxy", enabled: enable, err: err}
	}
}

func setTunCmd(c *mihomo.Client, enable bool) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return featureSetMsg{name: "tun", enabled: enable, err: fmt.Errorf("client is nil")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := c.SetTun(ctx, enable)
		return featureSetMsg{name: "tun", enabled: enable, err: err}
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

func deleteSubCmd(rt *runtime.Manager, provider string) tea.Cmd {
	return func() tea.Msg {
		if rt == nil {
			return deleteSubMsg{err: fmt.Errorf("runtime unavailable")}
		}
		err := rt.Subscriptions().DeleteByProvider(provider)
		return deleteSubMsg{name: provider, err: err}
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

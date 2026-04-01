package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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

type profileListItem struct {
	title    string
	desc     string
	filter   string
	provider string
}

func (i profileListItem) Title() string       { return i.title }
func (i profileListItem) Description() string { return i.desc }
func (i profileListItem) FilterValue() string { return i.filter }

type selectorItem struct {
	id    string
	title string
	desc  string
}

func (i selectorItem) Title() string       { return i.title }
func (i selectorItem) Description() string { return i.desc }
func (i selectorItem) FilterValue() string { return i.title + " " + i.desc }

type footerKeys struct {
	items []key.Binding
}

func (k footerKeys) ShortHelp() []key.Binding { return k.items }
func (k footerKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.items}
}

func (t clickTarget) hit(x, y int) bool {
	return x >= t.x1 && x < t.x2 && y >= t.y1 && y < t.y2
}

type model struct {
	styles styles
	help   help.Model

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
	langOpen       bool
	langCursor     int
	selectorKind   string
	selectorTitle  string
	selectorLV     list.Model

	rulesTable table.Model
	connsTable table.Model
	logsView   viewport.Model
	notifView  viewport.Model
	profileLV  list.Model

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
	langChipTarget        clickTarget
	selectorItemTargets   []clickTarget
}

func NewModel(cfg config.Settings, rt *runtime.Manager) *model {
	client, err := mihomo.NewClient(cfg.Endpoint, cfg.Secret)
	m := &model{
		styles:         defaultStyles(defaultThemeIndex),
		help:           help.New(),
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
		rulesTable:     table.New(),
		connsTable:     table.New(),
		logsView:       viewport.New(0, 0),
		notifView:      viewport.New(0, 0),
	}
	d := list.NewDefaultDelegate()
	m.profileLV = list.New([]list.Item{}, d, 0, 0)
	m.profileLV.SetShowHelp(false)
	m.profileLV.SetShowTitle(false)
	m.profileLV.SetShowFilter(false)
	m.profileLV.SetShowPagination(false)
	m.profileLV.SetShowStatusBar(false)
	m.selectorLV = list.New([]list.Item{}, d, 0, 0)
	m.selectorLV.SetShowHelp(false)
	m.selectorLV.SetShowFilter(false)
	m.selectorLV.SetShowPagination(false)
	m.selectorLV.SetShowStatusBar(false)
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

func (m *model) syncProfileListItems() {
	items := make([]list.Item, 0, len(m.subscriptions))
	for _, s := range m.subscriptions {
		state := m.t("enabled", "启用")
		if !s.Enabled {
			state = m.t("disabled", "禁用")
		}
		title := s.Name
		if title == "" {
			title = s.ProviderName
		}
		desc := fmt.Sprintf("[%s] %s=%s", state, m.t("provider", "提供者"), s.ProviderName)
		items = append(items, profileListItem{
			title:    title,
			desc:     desc,
			filter:   title + " " + s.ProviderName,
			provider: s.ProviderName,
		})
	}
	_ = m.profileLV.SetItems(items)
	if len(items) == 0 {
		return
	}
	idx := clamp(m.profileCursor, 0, len(items)-1)
	m.profileLV.Select(idx)
}

func (m *model) selectedProfileProvider() string {
	it := m.profileLV.SelectedItem()
	if it == nil {
		return ""
	}
	p, ok := it.(profileListItem)
	if !ok {
		return ""
	}
	return p.provider
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
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.langOpen {
				if handled, cmd := m.handleSelectorMouse(msg.X, msg.Y); handled {
					return m, cmd
				}
			}
			if m.langChipTarget.hit(msg.X, msg.Y) {
				m.toggleLanguage()
				return m, nil
			}
		}
		if m.tab == 1 && m.networkTab == 0 {
			if msg.Button == tea.MouseButtonWheelUp {
				if len(m.nodes) > 0 {
					m.proxyPane = 1
					page := max(1, m.nodePageSize)
					maxOffset := max(0, len(m.nodes)-page)
					m.nodeOffset = clamp(m.nodeOffset-1, 0, maxOffset)
				}
				return m, nil
			}
			if msg.Button == tea.MouseButtonWheelDown {
				if len(m.nodes) > 0 {
					m.proxyPane = 1
					page := max(1, m.nodePageSize)
					maxOffset := max(0, len(m.nodes)-page)
					m.nodeOffset = clamp(m.nodeOffset+1, 0, maxOffset)
				}
				return m, nil
			}
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
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
		if m.langOpen {
			return m, m.handleGlobalKeys(msg)
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
	base = m.styles.app.Width(m.width).Height(m.height).Render(base)
	if m.paletteOpen {
		overlay := m.renderPalette(max(54, m.width/2), max(10, m.height/2))
		placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
		return m.styles.app.Width(m.width).Height(m.height).Render(placed)
	}
	if m.langOpen {
		overlay := m.renderSelectorOverlay(max(36, m.width/3), max(10, m.height/3))
		placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
		return m.styles.app.Width(m.width).Height(m.height).Render(placed)
	}
	return base
}

func (m *model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	if m.langOpen {
		switch msg.String() {
		case "esc":
			m.langOpen = false
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
	m.langChipTarget = clickTarget{}
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
	chip := m.renderLanguageChip()
	right1Prefix := fmt.Sprintf("%s %s | %s %s | ", connIcon, conn, m.t("mode", "模式"), mode)
	right1 := right1Prefix + chip
	right2 := fmt.Sprintf("%s %s %s %s | %s %s | %s %s",
		m.t("up", "上行"), up, m.t("down", "下行"), down,
		m.t("core", "内核"), controller,
		m.t("theme", "主题"), themes[m.themeIndex].Name)
	totalW := max(1, w)
	innerW := max(1, totalW-m.styles.statusBar.GetHorizontalFrameSize())
	// Keep a small safety margin for terminals whose glyph width differs
	// from runewidth/lipgloss assumptions, preventing visual auto-wrap.
	lineW := max(1, innerW-2)
	line1 := composeHeaderLine(title, right1, lineW)
	line2 := composeHeaderLine("", right2, lineW)
	top := m.styles.statusBar.Width(totalW).MaxWidth(totalW).Render(line1)
	bottom := m.styles.statusBar.Width(totalW).MaxWidth(totalW).Render(line2)
	// Approximate chip hit box on header first line for mouse toggle.
	leftW := lipgloss.Width(strings.TrimSpace(title))
	gap := 2
	rightW := max(0, lineW-leftW-gap)
	if lw := lipgloss.Width(right1); lw <= rightW {
		statusPadLeft := 2 // statusBar uses Padding(0, 2)
		chipX := statusPadLeft + leftW + gap + (rightW - lw) + lipgloss.Width(right1Prefix)
		m.langChipTarget = clickTarget{
			x1: chipX,
			y1: 0,
			x2: chipX + lipgloss.Width(chip),
			y2: 1,
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m *model) renderTabs(w, y int) string {
	m.mainTabTargets = nil
	out := make([]string, 0, len(tabs))
	cursorX := 0
	for i := range tabs {
		icon := ""
		if i < len(tabIcons) {
			icon = tabIcons[i] + " "
		}
		label := fmt.Sprintf("%s%s", icon, m.mainTabLabel(i))
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
	items := make([]string, 0, len(networkTabs))
	for i := range networkTabs {
		items = append(items, m.networkTabLabel(i))
	}
	sub := m.renderSubTabs(m.t("NETWORK", "网络"), items, m.networkTab, m.bodyY, &m.networkTabTargets)
	subW := max(1, w)
	sub = m.styles.subTabBar.Width(subW).MaxWidth(subW).Render(sub)
	bodyH := max(4, h-lipgloss.Height(sub))
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
	items := make([]string, 0, len(systemTabs))
	for i := range systemTabs {
		items = append(items, m.systemTabLabel(i))
	}
	sub := m.renderSubTabs(m.t("SYSTEM", "系统"), items, m.systemTab, m.bodyY, &m.systemTabTargets)
	subW := max(1, w)
	sub = m.styles.subTabBar.Width(subW).MaxWidth(subW).Render(sub)
	bodyH := max(4, h-lipgloss.Height(sub))
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
		return m.renderPanel(bodyW, bodyH, m.t("Loading Mihomo data...", "正在加载 Mihomo 数据..."))
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

func (m *model) renderProxies(w, h int) string {
	m.proxyModeTargets = nil
	m.proxyActionTarget = nil
	m.proxyGroupTargets = nil
	m.proxyNodeTargets = nil

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
		m.proxyGroupTargets = append(m.proxyGroupTargets, clickTarget{
			x1:  leftContentX,
			y1:  leftContentY + row,
			x2:  leftContentX + max(1, leftW-m.styles.panel.GetHorizontalFrameSize()),
			y2:  leftContentY + row + 1,
			idx: i,
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
		m.proxyNodeTargets = append(m.proxyNodeTargets, clickTarget{
			x1:  rightContentX,
			y1:  rightContentY + row,
			x2:  rightContentX + max(1, rightW-m.styles.panel.GetHorizontalFrameSize()),
			y2:  rightContentY + row + 1,
			idx: i,
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
		rows = append(rows, table.Row{
			fitTextWidth(host, hostW),
			fitTextWidth(process, procW),
			fitTextWidth(chain, chainW),
		})
	}
	m.connsTable.Focus()
	m.connsTable.SetColumns([]table.Column{
		{Title: m.t("Host", "主机"), Width: hostW},
		{Title: m.t("Process", "进程"), Width: procW},
		{Title: m.t("Chain", "链路"), Width: chainW},
	})
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
		rows = append(rows, table.Row{
			fitTextWidth(r.Type, typeW),
			fitTextWidth(r.Payload, payloadW),
			fitTextWidth("-> "+r.Proxy, proxyW),
		})
	}
	m.rulesTable.Focus()
	m.rulesTable.SetColumns([]table.Column{
		{Title: m.t("Type", "类型"), Width: typeW},
		{Title: m.t("Payload", "匹配"), Width: payloadW},
		{Title: m.t("Proxy", "策略"), Width: proxyW},
	})
	m.rulesTable.SetRows(rows)
	m.rulesTable.SetWidth(contentW)
	m.rulesTable.SetHeight(contentH)
	m.rulesTable.SetCursor(clamp(m.rulesOffset, 0, max(0, len(rows)-1)))
	m.rulesOffset = m.rulesTable.Cursor()
	lines = append(lines, m.rulesTable.View())
	return m.renderPanel(w, h, strings.Join(lines, "\n"))
}

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

func (m *model) renderFooter(w int) string {
	binds := m.footerBindings()
	msg := strings.TrimSpace(m.status)
	if msg == "" {
		msg = "ready"
	}
	totalW := max(1, w)
	innerW := max(1, totalW-m.styles.footer.GetHorizontalFrameSize())
	lineW := max(1, innerW-2)
	m.help.Width = lineW
	helpLine := m.help.View(footerKeys{items: binds})
	line := composeHeaderLine(helpLine, m.t("status ", "状态 ")+msg, lineW)
	line = m.styles.footer.Width(totalW).MaxWidth(totalW).Render(line)
	return lipgloss.NewStyle().Width(totalW).MaxWidth(totalW).Render(line)
}

func (m *model) footerBindings() []key.Binding {
	switch m.tab {
	case 0:
		return []key.Binding{
			key.NewBinding(key.WithKeys("r", "g", "d"), key.WithHelp("r/g/d", m.t("mode", "模式"))),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", m.t("sys-proxy", "系统代理"))),
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "TUN")),
			key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", m.t("palette", "面板"))),
		}
	case 1:
		if m.networkTab == 2 {
			return []key.Binding{
				key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/j k", m.t("move", "移动"))),
				key.NewBinding(key.WithKeys("x"), key.WithHelp("x", m.t("close", "关闭"))),
				key.NewBinding(key.WithKeys("X"), key.WithHelp("X", m.t("close all", "全部关闭"))),
			}
		}
		if m.networkTab == 0 {
			return []key.Binding{
				key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/j k", m.t("move", "移动"))),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", m.t("select", "选择"))),
				key.NewBinding(key.WithKeys("t", "T"), key.WithHelp("t/T", m.t("delay test", "延迟测试"))),
				key.NewBinding(key.WithKeys("r", "g", "d"), key.WithHelp("r/g/d", m.t("mode", "模式"))),
			}
		}
	case 2:
		return []key.Binding{
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", m.t("import", "导入"))),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", m.t("update", "更新"))),
			key.NewBinding(key.WithKeys("U"), key.WithHelp("U", m.t("update all", "更新全部"))),
			key.NewBinding(key.WithKeys("x"), key.WithHelp("x", m.t("delete", "删除"))),
		}
	case 3:
		if m.systemTab == 1 {
			return []key.Binding{
				key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/shift+tab", m.t("focus", "焦点"))),
				key.NewBinding(key.WithKeys("s"), key.WithHelp("s", m.t("save", "保存"))),
			}
		}
		return []key.Binding{
			key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/j k", m.t("scroll", "滚动"))),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", m.t("clear", "清空"))),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", m.t("palette", "面板"))),
		key.NewBinding(key.WithKeys("1", "2", "3", "4"), key.WithHelp("1-4", m.t("tabs", "标签"))),
		key.NewBinding(key.WithKeys("L"), key.WithHelp("L", m.t("language", "语言"))),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", m.t("quit", "退出"))),
	}
}

func (m *model) renderLanguageChip() string {
	// Keep this chip strictly single-line/plain text; bordered styles (pill)
	// are multi-line blocks and will break header line layout.
	if m.lang() == "en" {
		return m.t("Lang", "语言") + ": [EN●] [中文○]"
	}
	return m.t("Lang", "语言") + ": [EN○] [中文●]"
}

func (m *model) toggleLanguage() {
	code := "en"
	if m.lang() == "en" {
		code = "zh-CN"
	}
	m.cfg.Language = code
	m.syncProfileListItems()
	if err := config.Save(m.cfg); err != nil {
		m.setStatus(m.tf("save language failed: %v", "保存语言失败: %v", err))
		return
	}
	m.setStatus(m.tf("language switched to %s", "语言已切换为 %s", m.langLabel(code)))
}

func (m *model) openSelector(kind string) {
	m.langOpen = true
	m.selectorKind = kind
	m.selectorTitle = kind
	items := make([]list.Item, 0, 8)
	selectedIdx := 0
	switch kind {
	case "language":
		m.selectorTitle = m.t("Language", "语言")
		items = []list.Item{
			selectorItem{id: "en", title: "English", desc: "English UI"},
			selectorItem{id: "zh-CN", title: "中文", desc: "中文界面"},
		}
		if m.lang() == "zh-CN" {
			selectedIdx = 1
		}
	case "theme":
		m.selectorTitle = m.t("Theme", "主题")
		for i, th := range themes {
			items = append(items, selectorItem{id: strconv.Itoa(i), title: th.Name, desc: ""})
			if i == m.themeIndex {
				selectedIdx = i
			}
		}
	case "mode":
		m.selectorTitle = m.t("Mode", "模式")
		items = []list.Item{
			selectorItem{id: "rule", title: m.t("Rule", "规则"), desc: ""},
			selectorItem{id: "global", title: m.t("Global", "全局"), desc: ""},
			selectorItem{id: "direct", title: m.t("Direct", "直连"), desc: ""},
		}
		switch strings.ToLower(m.baseCfg.Mode) {
		case "global":
			selectedIdx = 1
		case "direct":
			selectedIdx = 2
		}
	}
	_ = m.selectorLV.SetItems(items)
	m.selectorLV.Title = m.selectorTitle
	m.selectorLV.SetShowTitle(true)
	m.selectorLV.Select(selectedIdx)
}

func (m *model) applySelectorChoice() tea.Cmd {
	it := m.selectorLV.SelectedItem()
	if it == nil {
		m.langOpen = false
		return nil
	}
	opt, ok := it.(selectorItem)
	if !ok {
		m.langOpen = false
		return nil
	}
	m.langOpen = false
	switch m.selectorKind {
	case "language":
		m.cfg.Language = opt.id
		m.syncProfileListItems()
		if err := config.Save(m.cfg); err != nil {
			m.setStatus(m.tf("save language failed: %v", "保存语言失败: %v", err))
			return nil
		}
		m.setStatus(m.tf("language switched to %s", "语言已切换为 %s", m.langLabel(opt.id)))
		return nil
	case "theme":
		idx, err := strconv.Atoi(opt.id)
		if err == nil && idx >= 0 && idx < len(themes) {
			m.themeIndex = idx
			m.styles = defaultStyles(m.themeIndex)
			m.setStatus(m.t("theme updated", "主题已更新"))
		}
		return nil
	case "mode":
		return tea.Batch(setModeCmd(m.client, opt.id), fetchConfigCmd(m.client))
	default:
		return nil
	}
}

func (m *model) renderSelectorOverlay(w, h int) string {
	m.selectorItemTargets = nil
	cw := max(1, w-m.styles.overlay.GetHorizontalFrameSize())
	ch := max(4, h-m.styles.overlay.GetVerticalFrameSize())
	m.selectorLV.SetSize(cw, ch)
	ox := max(0, (m.width-w)/2)
	oy := max(0, (m.height-h)/2)
	// overlay border+padding => content origin
	contentX := ox + 3
	contentY := oy + 2
	// list title occupies first line when enabled
	rowY := contentY + 1
	for i, it := range m.selectorLV.VisibleItems() {
		si, ok := it.(selectorItem)
		if !ok {
			continue
		}
		m.selectorItemTargets = append(m.selectorItemTargets, clickTarget{
			x1:   contentX,
			y1:   rowY + i,
			x2:   contentX + cw,
			y2:   rowY + i + 1,
			idx:  i,
			text: si.id,
		})
	}
	body := m.selectorLV.View() + "\n" + m.styles.subtle.Render(m.t("Enter confirm   Esc close", "Enter 确认   Esc 关闭"))
	return m.styles.overlay.Width(w).Height(h).MaxWidth(w).MaxHeight(h).Render(body)
}

func (m *model) handleSelectorMouse(x, y int) (bool, tea.Cmd) {
	for _, t := range m.selectorItemTargets {
		if t.hit(x, y) {
			m.selectorLV.Select(t.idx)
			return true, m.applySelectorChoice()
		}
	}
	// click outside closes selector
	ow := max(36, m.width/3)
	oh := max(10, m.height/3)
	ox := max(0, (m.width-ow)/2)
	oy := max(0, (m.height-oh)/2)
	if x < ox || x >= ox+ow || y < oy || y >= oy+oh {
		m.langOpen = false
		return true, nil
	}
	return false, nil
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
		{label: m.t("Rule", "规则"), mode: "rule"},
		{label: m.t("Global", "全局"), mode: "global"},
		{label: m.t("Direct", "直连"), mode: "direct"},
	}

	prefix := m.t("Mode: ", "模式: ")
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

func (m *model) renderOverviewModeLines(contentX, y, maxW int) []string {
	current := strings.ToLower(m.baseCfg.Mode)
	type modeToken struct {
		label string
		mode  string
	}
	tokens := []modeToken{
		{label: m.t("Rule", "规则"), mode: "rule"},
		{label: m.t("Global", "全局"), mode: "global"},
		{label: m.t("Direct", "直连"), mode: "direct"},
	}

	prefix := m.t("Switch: ", "切换: ")
	sep := " "
	currentY := y
	currentLine := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	lines := make([]string, 0, 2)
	for i, t := range tokens {
		token := "[○ " + t.label + "]"
		if t.mode == current {
			token = "[✓ " + strings.ToUpper(t.label) + "]"
		}
		addSep := ""
		if i > 0 {
			addSep = sep
		}
		projectedW := lipgloss.Width(currentLine + addSep + token)
		if i > 0 && projectedW > maxW {
			lines = append(lines, currentLine)
			currentY++
			currentLine = strings.Repeat(" ", lipgloss.Width(prefix))
			cursorX = contentX + lipgloss.Width(prefix)
			addSep = ""
		}
		if addSep != "" {
			currentLine += addSep
			cursorX += lipgloss.Width(addSep)
		}
		startX := cursorX
		endX := startX + lipgloss.Width(token)
		m.overviewModeTargets = append(m.overviewModeTargets, clickTarget{
			x1:   startX,
			y1:   currentY,
			x2:   endX,
			y2:   currentY + 1,
			text: t.mode,
		})
		currentLine += token
		cursorX = endX
	}
	lines = append(lines, currentLine)
	return lines
}

func (m *model) renderOverviewToggleLines(contentX, y, maxW int) []string {
	systemOn := m.systemProxyEnabled()
	tunOn := m.baseCfg.Tun.Enable

	type toggleToken struct {
		id    string
		label string
		on    bool
	}
	tokens := []toggleToken{
		{id: "system-proxy", label: m.t("System Proxy", "系统代理"), on: systemOn},
		{id: "tun", label: m.t("TUN Mode", "TUN 模式"), on: tunOn},
	}

	prefix := m.t("Toggles: ", "开关: ")
	sep := "   "
	currentY := y
	currentLine := prefix
	cursorX := contentX + lipgloss.Width(prefix)
	lines := make([]string, 0, 2)
	for i, t := range tokens {
		label := m.styles.subTab.Render(t.label)
		sw := m.renderWebToggle(t.on)
		token := label + " " + sw
		addSep := ""
		if i > 0 {
			addSep = sep
		}
		projectedW := lipgloss.Width(currentLine + addSep + token)
		if i > 0 && projectedW > maxW {
			lines = append(lines, currentLine)
			currentY++
			currentLine = strings.Repeat(" ", lipgloss.Width(prefix))
			cursorX = contentX + lipgloss.Width(prefix)
			addSep = ""
		}
		if addSep != "" {
			currentLine += addSep
			cursorX += lipgloss.Width(addSep)
		}
		currentLine += token

		wToken := lipgloss.Width(token)
		m.overviewToggleTargets = append(m.overviewToggleTargets, clickTarget{
			x1:   cursorX,
			y1:   currentY,
			x2:   cursorX + wToken,
			y2:   currentY + 1,
			text: fmt.Sprintf("%s:%s", t.id, map[bool]string{true: "off", false: "on"}[t.on]),
		})
		cursorX += wToken
	}
	lines = append(lines, currentLine)
	return lines
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
	prefix := m.t("Action: ", "操作: ")
	// 使用 styles.action 渲染按钮，让它看起来像一个真实的按钮块
	button := m.styles.action.Render(m.t(" ⚡ Test All (T) ", " ⚡ 全部测试 (T) "))

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
	w := max(1, outerW)
	h := max(1, outerH)
	return m.styles.panel.
		Width(w).
		Height(h).
		MaxWidth(w).
		MaxHeight(h).
		Render(content)
}

func (m *model) renderPanelFocused(outerW, outerH int, content string, focused bool) string {
	w := max(1, outerW)
	h := max(1, outerH)
	st := m.styles.panel
	if focused {
		st = st.BorderForeground(lipgloss.Color(themes[m.themeIndex].Primary))
	} else {
		st = st.BorderForeground(lipgloss.Color(themes[m.themeIndex].Panel))
	}
	return st.
		Width(w).
		Height(h).
		MaxWidth(w).
		MaxHeight(h).
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

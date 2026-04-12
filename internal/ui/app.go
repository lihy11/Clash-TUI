package ui

import (
	"fmt"
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
	selectorOpen   bool
	selectorKind   string
	selectorTitle  string
	selectorLV     list.Model

	rulesTable table.Model
	connsTable table.Model
	logsView   viewport.Model
	notifView  viewport.Model
	profileLV  list.Model
	mouse      mouseRouter

	bodyY int
	bodyW int
	bodyH int

	profileImportTargets []clickTarget
	selectorItemTargets  []clickTarget
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		return m, nil
	case tea.MouseMsg:
		return m, m.handleMouseMsg(msg)
	case tea.KeyMsg:
		return m, m.handleKeyUpdate(msg)
	default:
		if cmd, immediate := m.handleAppMessage(msg); immediate {
			return m, cmd
		}
		return m, m.postUpdateComponents(msg, nil)
	}
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	if m.width < 48 || m.height < 12 {
		msg := "Terminal too small. Resize to at least 48x12."
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.styles.subtle.Render(msg))
	}

	m.mouse.reset()
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
	if m.selectorOpen {
		overlay := m.renderSelectorOverlay(max(36, m.width/3), max(10, m.height/3))
		placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
		return m.styles.app.Width(m.width).Height(m.height).Render(placed)
	}
	return base
}

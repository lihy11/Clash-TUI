package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"clash-tui/internal/config"
)

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
	m.selectorOpen = true
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
		m.selectorOpen = false
		return nil
	}
	opt, ok := it.(selectorItem)
	if !ok {
		m.selectorOpen = false
		return nil
	}
	m.selectorOpen = false
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

func (m *model) applySelectorChoiceByID(id string) tea.Cmd {
	for i, it := range m.selectorLV.Items() {
		opt, ok := it.(selectorItem)
		if !ok || opt.id != id {
			continue
		}
		m.selectorLV.Select(i)
		return m.applySelectorChoice()
	}
	m.selectorOpen = false
	return nil
}

func (m *model) renderSelectorOverlay(w, h int) string {
	m.selectorItemTargets = nil
	cw := max(1, w-m.styles.overlay.GetHorizontalFrameSize())
	ch := max(4, h-m.styles.overlay.GetVerticalFrameSize())
	m.selectorLV.SetSize(cw, ch)
	ox := max(0, (m.width-w)/2)
	oy := max(0, (m.height-h)/2)
	m.mouse.register(mouseAction{
		ID:  "selector.dismiss",
		Box: hitBox{x1: 0, y1: 0, x2: m.width, y2: m.height},
	})
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
		m.mouse.register(mouseAction{
			ID:   "selector.choose",
			Text: si.id,
			Box:  hitBox{x1: contentX, y1: rowY + i, x2: contentX + cw, y2: rowY + i + 1},
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
		m.selectorOpen = false
		return true, nil
	}
	return false, nil
}

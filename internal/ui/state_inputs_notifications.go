package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

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

func (m *model) systemProxyEnabled() bool {
	if m.baseCfg.HasSystemProxy {
		return m.baseCfg.SystemProxy
	}
	return m.systemProxyOn
}

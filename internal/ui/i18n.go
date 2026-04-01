package ui

import "fmt"

func (m *model) lang() string {
	if m.cfg.Language == "zh-CN" {
		return "zh-CN"
	}
	return "en"
}

func (m *model) isZH() bool {
	return m.lang() == "zh-CN"
}

func (m *model) t(en, zh string) string {
	if m.isZH() {
		return zh
	}
	return en
}

func (m *model) tf(en, zh string, args ...any) string {
	if m.isZH() {
		return fmt.Sprintf(zh, args...)
	}
	return fmt.Sprintf(en, args...)
}

func (m *model) langLabel(code string) string {
	if code == "zh-CN" {
		return "中文"
	}
	return "English"
}

func (m *model) mainTabLabel(i int) string {
	en := []string{"Dashboard", "Network", "Profiles", "System"}
	zh := []string{"总览", "网络", "配置", "系统"}
	if i < 0 || i >= len(en) {
		return ""
	}
	return m.t(en[i], zh[i])
}

func (m *model) networkTabLabel(i int) string {
	en := []string{"Proxies", "Rules", "Connections"}
	zh := []string{"代理", "规则", "连接"}
	if i < 0 || i >= len(en) {
		return ""
	}
	return m.t(en[i], zh[i])
}

func (m *model) systemTabLabel(i int) string {
	en := []string{"Logs", "Settings"}
	zh := []string{"日志", "设置"}
	if i < 0 || i >= len(en) {
		return ""
	}
	return m.t(en[i], zh[i])
}

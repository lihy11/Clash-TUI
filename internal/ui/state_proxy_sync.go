package ui

import (
	"sort"
	"strings"
	"time"

	"clash-tui/internal/mihomo"
)

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

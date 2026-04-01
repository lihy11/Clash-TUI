package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clash-tui/internal/config"
	"clash-tui/internal/mihomo"
	"clash-tui/internal/runtime"
	"clash-tui/internal/subscription"
)

func pollCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
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

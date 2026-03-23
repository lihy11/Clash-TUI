package runtime

import (
	"context"
	"net/http"
	"time"

	"clash-tui/internal/config"
	"clash-tui/internal/core"
	"clash-tui/internal/subscription"
)

type Manager struct {
	core *core.Manager
	subs *subscription.Manager
}

func New() (*Manager, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	coreMgr, err := core.NewManager(dataDir)
	if err != nil {
		return nil, err
	}
	subsMgr, err := subscription.NewManager()
	if err != nil {
		return nil, err
	}
	return &Manager{
		core: coreMgr,
		subs: subsMgr,
	}, nil
}

func (m *Manager) Subscriptions() *subscription.Manager {
	return m.subs
}

func (m *Manager) Boot(ctx context.Context, cfg *config.Settings) error {
	cfg.Secret = subscription.EnsureSecret(cfg.Secret)
	if err := config.Save(*cfg); err != nil {
		return err
	}
	if !cfg.ManageCore {
		return nil
	}
	if err := m.core.EnsureBinary(ctx); err != nil {
		return err
	}
	items, err := m.subs.Load()
	if err != nil {
		return err
	}
	content, err := subscription.BuildMihomoConfig(*cfg, items)
	if err != nil {
		return err
	}
	if err := m.core.WriteConfig(content); err != nil {
		return err
	}
	if err := m.core.Start(ctx); err != nil {
		return err
	}
	return waitController(ctx, cfg.Endpoint)
}

func (m *Manager) ReloadCore(ctx context.Context, cfg config.Settings) error {
	if !cfg.ManageCore {
		return nil
	}
	items, err := m.subs.Load()
	if err != nil {
		return err
	}
	content, err := subscription.BuildMihomoConfig(cfg, items)
	if err != nil {
		return err
	}
	if err := m.core.WriteConfig(content); err != nil {
		return err
	}
	if err := m.core.Restart(ctx); err != nil {
		return err
	}
	return waitController(ctx, cfg.Endpoint)
}

func (m *Manager) Close() {
	m.core.Stop()
}

func waitController(ctx context.Context, endpoint string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(20 * time.Second)
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/version", nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			return context.DeadlineExceeded
		}
		time.Sleep(600 * time.Millisecond)
	}
}

package runtime

import (
	"context"
	"log"

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
	return &Manager{core: coreMgr, subs: subsMgr}, nil
}

func (m *Manager) Subscriptions() *subscription.Manager {
	return m.subs
}

func (m *Manager) Boot(ctx context.Context, cfg *config.Settings) error {
	if ok, noSecret := canUseController(ctx, cfg.Endpoint, cfg.Secret); ok {
		if noSecret && cfg.Secret != "" {
			cfg.Secret = ""
			_ = config.Save(*cfg)
		}
		m.autoImportSubscriptionsFromLocalConfig(cfg.Endpoint)
		log.Printf("connected to existing core: %s", cfg.Endpoint)
		return nil
	}

	if endpoint, secret, ok := discoverRunningController(ctx, *cfg); ok {
		cfg.Endpoint = endpoint
		cfg.Secret = secret
		_ = config.Save(*cfg)
		m.autoImportSubscriptionsFromLocalConfig(endpoint)
		log.Printf("detected local running core: %s", endpoint)
		return nil
	}

	if !cfg.ManageCore {
		return nil
	}

	cfg.Secret = subscription.EnsureSecret(cfg.Secret)
	if err := config.Save(*cfg); err != nil {
		return err
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

package subscription

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"clash-tui/internal/config"
)

type Item struct {
	Name         string    `yaml:"name"`
	URL          string    `yaml:"url"`
	ProviderName string    `yaml:"provider_name"`
	Enabled      bool      `yaml:"enabled"`
	LastUpdated  time.Time `yaml:"last_updated"`
	LastError    string    `yaml:"last_error"`
}

type store struct {
	Items []Item `yaml:"items"`
}

type Manager struct {
	file string
}

func NewManager() (*Manager, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Manager{file: filepath.Join(dir, "subscriptions.yaml")}, nil
}

func (m *Manager) Load() ([]Item, error) {
	b, err := os.ReadFile(m.file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, err
	}
	var s store
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return s.Items, nil
}

func (m *Manager) Save(items []Item) error {
	b, err := yaml.Marshal(store{Items: items})
	if err != nil {
		return err
	}
	return os.WriteFile(m.file, b, 0o644)
}

func (m *Manager) Import(name, rawURL string) (Item, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return Item{}, fmt.Errorf("subscription url is empty")
	}
	if _, err := url.Parse(rawURL); err != nil {
		return Item{}, fmt.Errorf("invalid subscription url: %w", err)
	}
	if strings.TrimSpace(name) == "" {
		name = guessNameFromURL(rawURL)
	}
	name = strings.TrimSpace(name)
	items, err := m.Load()
	if err != nil {
		return Item{}, err
	}
	provider := uniqueProviderName(items, slugify(name))
	item := Item{
		Name:         name,
		URL:          rawURL,
		ProviderName: provider,
		Enabled:      true,
		LastUpdated:  time.Time{},
		LastError:    "",
	}
	// Keep a single active subscription by default to avoid conflicting
	// provider sets after importing a new source.
	for i := range items {
		items[i].Enabled = false
	}
	items = append(items, item)
	if err := m.Save(items); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (m *Manager) DeleteByProvider(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return fmt.Errorf("provider name is empty")
	}
	items, err := m.Load()
	if err != nil {
		return err
	}
	out := make([]Item, 0, len(items))
	removed := false
	for _, it := range items {
		if it.ProviderName == provider {
			removed = true
			continue
		}
		out = append(out, it)
	}
	if !removed {
		return fmt.Errorf("subscription not found: %s", provider)
	}
	// Ensure at least one enabled item if there are remaining entries.
	enabled := false
	for _, it := range out {
		if it.Enabled {
			enabled = true
			break
		}
	}
	if !enabled && len(out) > 0 {
		out[0].Enabled = true
	}
	return m.Save(out)
}

func EnsureSecret(in string) string {
	if strings.TrimSpace(in) != "" {
		return in
	}
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func BuildMihomoConfig(cfg config.Settings, subs []Item) (string, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return "", err
	}
	controller := u.Host
	if controller == "" {
		controller = "127.0.0.1:9090"
	}
	root := map[string]any{
		"mixed-port":          cfg.MixedPort,
		"allow-lan":           false,
		"mode":                "rule",
		"log-level":           cfg.LogLevel,
		"ipv6":                false,
		"secret":              cfg.Secret,
		"external-controller": controller,
		"dns": map[string]any{
			"enable": true,
			"ipv6":   false,
		},
	}

	enabledSubs := normalizeEnabledSubs(subs)
	providers := map[string]any{}
	for _, s := range enabledSubs {
		if !s.Enabled {
			continue
		}
		providers[s.ProviderName] = map[string]any{
			"type":     "http",
			"url":      s.URL,
			"path":     fmt.Sprintf("./proxy_providers/%s.yaml", s.ProviderName),
			"interval": 86400,
			"health-check": map[string]any{
				"enable":   true,
				"url":      "https://www.gstatic.com/generate_204",
				"interval": 600,
			},
		}
	}
	root["proxy-providers"] = providers

	if len(providers) == 0 {
		root["proxies"] = []any{
			map[string]any{
				"name": "DIRECT",
				"type": "direct",
			},
		}
		root["proxy-groups"] = []any{
			map[string]any{
				"name":    "PROXY",
				"type":    "select",
				"proxies": []string{"DIRECT"},
			},
		}
		root["rules"] = []string{"MATCH,DIRECT"}
	} else {
		use := make([]string, 0, len(providers))
		for _, s := range enabledSubs {
			if s.Enabled {
				use = append(use, s.ProviderName)
			}
		}
		root["proxy-groups"] = []any{
			map[string]any{
				"name": "PROXY",
				"type": "select",
				"use":  use,
			},
			map[string]any{
				"name":     "AUTO",
				"type":     "url-test",
				"use":      use,
				"url":      "https://www.gstatic.com/generate_204",
				"interval": 300,
			},
		}
		root["rules"] = []string{"MATCH,PROXY"}
	}

	b, err := yaml.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeEnabledSubs(subs []Item) []Item {
	out := append([]Item{}, subs...)
	firstEnabled := -1
	for i := range out {
		if out[i].Enabled {
			firstEnabled = i
			break
		}
	}
	if firstEnabled == -1 {
		return out
	}
	for i := range out {
		out[i].Enabled = i == firstEnabled
	}
	return out
}

func guessNameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		return u.Host
	}
	return "subscription"
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "sub"
	}
	return s
}

func uniqueProviderName(items []Item, base string) string {
	name := base
	used := map[string]struct{}{}
	for _, it := range items {
		used[it.ProviderName] = struct{}{}
	}
	if _, ok := used[name]; !ok {
		return name
	}
	for i := 2; ; i++ {
		name = fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[name]; !ok {
			return name
		}
	}
}

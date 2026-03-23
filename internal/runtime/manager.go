package runtime

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"clash-tui/internal/config"
	"clash-tui/internal/core"
	"clash-tui/internal/subscription"
	"gopkg.in/yaml.v3"
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

func canUseController(ctx context.Context, endpoint, secret string) (ok bool, noSecret bool) {
	if endpoint == "" {
		return false, false
	}
	if probeController(ctx, endpoint, secret) {
		return true, false
	}
	if secret != "" && probeController(ctx, endpoint, "") {
		return true, true
	}
	return false, false
}

func probeController(ctx context.Context, endpoint, secret string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/version", nil)
	if err != nil {
		return false
	}
	if strings.TrimSpace(secret) != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

type endpointCandidate struct {
	Endpoint string
	Secret   string
}

func discoverRunningController(ctx context.Context, cfg config.Settings) (string, string, bool) {
	candidates := []endpointCandidate{
		{Endpoint: "http://127.0.0.1:9090", Secret: cfg.Secret},
		{Endpoint: "http://127.0.0.1:9090", Secret: ""},
		{Endpoint: "http://127.0.0.1:17650", Secret: ""},
	}
	candidates = append(candidates, discoverCandidatesFromFiles()...)
	for _, c := range candidates {
		if c.Endpoint == "" {
			continue
		}
		if probeController(ctx, c.Endpoint, c.Secret) {
			return c.Endpoint, c.Secret, true
		}
	}
	return "", "", false
}

func discoverCandidatesFromFiles() []endpointCandidate {
	paths := knownConfigPaths()
	out := make([]endpointCandidate, 0, len(paths))
	for _, p := range paths {
		ep, sec := parseControllerFromFile(p)
		if ep != "" {
			out = append(out, endpointCandidate{Endpoint: ep, Secret: sec})
		}
	}
	return out
}

func parseControllerFromFile(path string) (endpoint, secret string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "external-controller:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "external-controller:"))
			v = strings.Trim(v, `"'`)
			if v != "" {
				if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
					endpoint = v
				} else {
					endpoint = "http://" + v
				}
			}
		}
		if strings.HasPrefix(line, "secret:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "secret:"))
			secret = strings.Trim(v, `"'`)
		}
	}
	return endpoint, secret
}

func knownConfigPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".config", "clash", "config.yaml"),
		filepath.Join(home, ".config", "clash.meta", "config.yaml"),
		filepath.Join(home, "Library", "Application Support", "Clash Nyanpasu", "config", "clash-config.yaml"),
		filepath.Join(home, "Library", "Application Support", "Clash Nyanpasu", "config", "clash-guard-overrides.yaml"),
	}
}

func (m *Manager) autoImportSubscriptionsFromLocalConfig(endpoint string) {
	endpoint = normalizeEndpoint(endpoint)
	if endpoint == "" || m == nil || m.subs == nil {
		return
	}
	loaded, err := m.subs.Load()
	if err != nil {
		log.Printf("load subscriptions for auto-import failed: %v", err)
		return
	}

	byURL := make(map[string]struct{}, len(loaded))
	byProvider := make(map[string]struct{}, len(loaded))
	for _, it := range loaded {
		u := strings.TrimSpace(it.URL)
		if u != "" {
			byURL[u] = struct{}{}
		}
		p := strings.TrimSpace(it.ProviderName)
		if p != "" {
			byProvider[p] = struct{}{}
		}
	}

	candidates := readProviderSubscriptionsByEndpoint(endpoint)
	if len(candidates) == 0 {
		return
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].ProviderName < candidates[j].ProviderName
	})

	added := 0
	items := append([]subscription.Item{}, loaded...)
	for _, c := range candidates {
		if _, ok := byURL[c.URL]; ok {
			continue
		}
		base := slugify(c.ProviderName)
		if base == "" {
			base = "provider"
		}
		providerName := uniqueName(base, byProvider)
		name := strings.TrimSpace(c.Name)
		if name == "" {
			name = c.ProviderName
		}
		if strings.TrimSpace(name) == "" {
			name = "Imported Subscription"
		}
		items = append(items, subscription.Item{
			Name:         name,
			URL:          c.URL,
			ProviderName: providerName,
			Enabled:      true,
		})
		byURL[c.URL] = struct{}{}
		byProvider[providerName] = struct{}{}
		added++
	}
	if added == 0 {
		return
	}
	if err := m.subs.Save(items); err != nil {
		log.Printf("save auto-import subscriptions failed: %v", err)
		return
	}
	log.Printf("auto-imported %d subscriptions from local config", added)
}

type providerSubCandidate struct {
	Name         string
	ProviderName string
	URL          string
}

func readProviderSubscriptionsByEndpoint(targetEndpoint string) []providerSubCandidate {
	out := make([]providerSubCandidate, 0, 8)
	for _, path := range knownConfigPaths() {
		c, err := readProviderSubscriptions(path, targetEndpoint)
		if err != nil {
			continue
		}
		out = append(out, c...)
	}
	return out
}

func readProviderSubscriptions(path, targetEndpoint string) ([]providerSubCandidate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	controller := normalizeEndpoint(getString(root["external-controller"]))
	if controller != targetEndpoint {
		return nil, nil
	}
	pp, ok := root["proxy-providers"].(map[string]any)
	if !ok || len(pp) == 0 {
		return nil, nil
	}
	out := make([]providerSubCandidate, 0, len(pp))
	for providerName, raw := range pp {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(getString(m["type"])))
		if typ != "" && typ != "http" {
			continue
		}
		u := strings.TrimSpace(getString(m["url"]))
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			continue
		}
		out = append(out, providerSubCandidate{
			Name:         providerName,
			ProviderName: providerName,
			URL:          u,
		})
	}
	return out, nil
}

func getString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", v)
	}
}

func normalizeEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return ""
	}
	if strings.HasPrefix(ep, "http://") || strings.HasPrefix(ep, "https://") {
		return strings.TrimRight(ep, "/")
	}
	return "http://" + strings.TrimRight(ep, "/")
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func uniqueName(base string, used map[string]struct{}) string {
	name := base
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

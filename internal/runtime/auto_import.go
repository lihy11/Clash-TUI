package runtime

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"clash-tui/internal/subscription"
	"gopkg.in/yaml.v3"
)

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
			Enabled:      false,
		})
		byURL[c.URL] = struct{}{}
		byProvider[providerName] = struct{}{}
		added++
	}
	if added == 0 {
		return
	}
	enabled := -1
	for i := range items {
		if items[i].Enabled {
			enabled = i
			break
		}
	}
	if enabled == -1 && len(items) > 0 {
		items[0].Enabled = true
	} else if enabled >= 0 {
		for i := range items {
			items[i].Enabled = i == enabled
		}
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
		out = append(out, providerSubCandidate{Name: providerName, ProviderName: providerName, URL: u})
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

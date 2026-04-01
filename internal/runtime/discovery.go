package runtime

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clash-tui/internal/config"
)

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

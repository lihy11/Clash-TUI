package core

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const latestReleasePage = "https://github.com/MetaCubeX/mihomo/releases/latest"
const latestReleaseAPI = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"

var defaultGitHubMirrors = []string{
	"https://ghproxy.cn",
	"https://ghproxy.com",
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

func fetchLatestRelease(ctx context.Context) (release, error) {
	if rel, err := fetchLatestReleaseFromAPI(ctx); err == nil && len(rel.Assets) > 0 {
		return rel, nil
	}
	candidates := githubURLCandidates(latestReleasePage)
	var lastErr error
	for i, candidate := range candidates {
		if i > 0 {
			log.Printf("switching release-page source (%d/%d): %s", i+1, len(candidates), candidate)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "clash-tui/1.x")
		resp, err := coreHTTPClient().Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("fetch release page failed: %s", resp.Status)
			continue
		}
		body := string(b)
		tag := ""
		if resp.Request != nil && resp.Request.URL != nil {
			u := resp.Request.URL.String()
			if idx := strings.Index(u, "/releases/tag/"); idx >= 0 {
				tag = u[idx+len("/releases/tag/"):]
				if q := strings.IndexAny(tag, "?#"); q >= 0 {
					tag = tag[:q]
				}
			}
		}
		if tag == "" {
			if t := parseTagFromHTML(body); t != "" {
				tag = t
			}
		}
		out := release{TagName: tag, Assets: parseReleaseAssetsFromHTML(body)}
		if len(out.Assets) == 0 && out.TagName != "" {
			out.Assets = synthesizeAssetsByTag(out.TagName)
		}
		if len(out.Assets) == 0 {
			lastErr = fmt.Errorf("no downloadable assets found on release page (tag=%q)", out.TagName)
			continue
		}
		return out, nil
	}
	if lastErr != nil {
		return release{}, lastErr
	}
	return release{}, fmt.Errorf("fetch release page failed from all sources")
}

func fetchLatestReleaseFromAPI(ctx context.Context) (release, error) {
	candidates := githubURLCandidates(latestReleaseAPI)
	var lastErr error
	for i, candidate := range candidates {
		if i > 0 {
			log.Printf("switching release-api source (%d/%d): %s", i+1, len(candidates), candidate)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "clash-tui/1.x")
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := coreHTTPClient().Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("fetch latest release api failed: %s", resp.Status)
			_ = resp.Body.Close()
			continue
		}
		var out release
		decodeErr := json.NewDecoder(resp.Body).Decode(&out)
		_ = resp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}
		if out.TagName == "" || len(out.Assets) == 0 {
			lastErr = fmt.Errorf("latest release api response missing tag/assets")
			continue
		}
		return out, nil
	}
	if lastErr != nil {
		return release{}, lastErr
	}
	return release{}, fmt.Errorf("fetch latest release api failed from all sources")
}

func parseTagFromHTML(html string) string {
	re := regexp.MustCompile(`/MetaCubeX/mihomo/releases/tag/([A-Za-z0-9._\-]+)`)
	m := re.FindStringSubmatch(html)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func parseReleaseAssetsFromHTML(html string) []releaseAsset {
	re1 := regexp.MustCompile(`href="(/MetaCubeX/mihomo/releases/download/[^"]+)"`)
	matches := re1.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		re2 := regexp.MustCompile(`(/MetaCubeX/mihomo/releases/download/[^\s"'<>]+)`)
		raw := re2.FindAllStringSubmatch(html, -1)
		for _, m := range raw {
			if len(m) >= 2 {
				matches = append(matches, []string{"", m[1]})
			}
		}
	}
	seen := map[string]struct{}{}
	out := make([]releaseAsset, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		path := strings.TrimSpace(m[1])
		if path == "" {
			continue
		}
		full := "https://github.com" + path
		name := filepath.Base(path)
		if _, ok := seen[full]; ok {
			continue
		}
		seen[full] = struct{}{}
		out = append(out, releaseAsset{Name: name, URL: full})
	}
	return out
}

func synthesizeAssetsByTag(tag string) []releaseAsset {
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH
	exts := []string{".gz", ".zip"}
	patterns := []string{
		fmt.Sprintf("mihomo-%s-%s-compatible-%s", targetOS, targetArch, tag),
		fmt.Sprintf("mihomo-%s-%s-%s", targetOS, targetArch, tag),
	}
	out := make([]releaseAsset, 0, len(patterns)*len(exts))
	for _, p := range patterns {
		for _, ext := range exts {
			name := p + ext
			url := fmt.Sprintf("https://github.com/MetaCubeX/mihomo/releases/download/%s/%s", tag, name)
			out = append(out, releaseAsset{Name: name, URL: url})
		}
	}
	return out
}

func pickAssets(assets []releaseAsset) ([]releaseAsset, error) {
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH
	var candidates []releaseAsset
	for _, a := range assets {
		n := strings.ToLower(a.Name)
		if strings.Contains(n, "sha256") {
			continue
		}
		if !strings.HasSuffix(n, ".gz") && !strings.HasSuffix(n, ".zip") {
			continue
		}
		if strings.Contains(n, targetOS) && strings.Contains(n, targetArch) {
			candidates = append(candidates, a)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no release asset for %s/%s", targetOS, targetArch)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return assetPriority(candidates[i].Name) < assetPriority(candidates[j].Name)
	})
	if len(candidates) > 4 {
		candidates = candidates[:4]
	}
	return candidates, nil
}

func assetPriority(name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "compatible"):
		return 0
	case strings.Contains(n, "-v1-") || strings.Contains(n, "-v1."):
		return 1
	case strings.Contains(n, "-v2-") || strings.Contains(n, "-v2."):
		return 2
	case strings.Contains(n, "-v3-") || strings.Contains(n, "-v3."):
		return 3
	default:
		return 4
	}
}

func downloadFile(ctx context.Context, src, dst string) error {
	candidates := githubURLCandidates(src)
	var lastErr error
	for i, candidate := range candidates {
		if i > 0 {
			log.Printf("switching download source (%d/%d): %s", i+1, len(candidates), candidate)
		}
		if err := downloadFileOnce(ctx, candidate, dst); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("download failed from all sources")
}

func downloadFileOnce(ctx context.Context, src, dst string) error {
	startAt := int64(0)
	if st, err := os.Stat(dst); err == nil && st.Size() > 0 {
		startAt = st.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return err
	}
	if startAt > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startAt))
	}
	resp, err := coreHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("download asset failed: %s", resp.Status)
	}

	appendMode := resp.StatusCode == http.StatusPartialContent && startAt > 0
	if resp.StatusCode == http.StatusOK && startAt > 0 {
		startAt = 0
	}

	var f *os.File
	if appendMode {
		f, err = os.OpenFile(dst, os.O_WRONLY|os.O_APPEND, 0o644)
	} else {
		f, err = os.Create(dst)
	}
	if err != nil {
		return err
	}
	defer f.Close()

	total := contentTotal(resp, startAt)
	buf := make([]byte, 256*1024)
	downloaded := startAt
	start := time.Now()
	lastRender := time.Time{}

	renderDownloadProgress(downloaded, total, 0)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			downloaded += int64(n)
			now := time.Now()
			if lastRender.IsZero() || now.Sub(lastRender) >= 300*time.Millisecond {
				renderDownloadProgress(downloaded, total, now.Sub(start))
				lastRender = now
			}
		}
		if readErr == io.EOF {
			renderDownloadProgress(downloaded, total, time.Since(start))
			fmt.Fprintln(os.Stderr)
			log.Printf("download complete")
			return nil
		}
		if readErr != nil {
			fmt.Fprintln(os.Stderr)
			return readErr
		}
	}
}

func extractBinary(src, dst string) error {
	l := strings.ToLower(src)
	if strings.HasSuffix(l, ".zip") {
		return extractFromZip(src, dst)
	}
	return extractFromGzip(src, dst)
}

func extractFromZip(src, dst string) error {
	z, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer z.Close()
	for _, f := range z.File {
		n := strings.ToLower(filepath.Base(f.Name))
		if n == "mihomo" || n == "mihomo.exe" {
			r, err := f.Open()
			if err != nil {
				return err
			}
			defer r.Close()
			out, err := os.Create(dst)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, r); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("mihomo binary not found in zip")
}

func extractFromGzip(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, gz)
	return err
}

func renderDownloadProgress(done, total int64, elapsed time.Duration) {
	sec := elapsed.Seconds()
	if sec <= 0 {
		sec = 0.001
	}
	speed := float64(done) / sec
	if total > 0 {
		pct := float64(done) * 100 / float64(total)
		bar := progressBar(done, total, 24)
		fmt.Fprintf(os.Stderr, "\rdownloading core %s %5.1f%%  %s/%s  %s/s", bar, pct, formatBytes(done), formatBytes(total), formatBytes(int64(speed)))
		return
	}
	fmt.Fprintf(os.Stderr, "\rdownloading core %s  %s/s", formatBytes(done), formatBytes(int64(speed)))
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func contentTotal(resp *http.Response, startAt int64) int64 {
	if resp.StatusCode == http.StatusPartialContent {
		cr := resp.Header.Get("Content-Range")
		if cr != "" {
			parts := strings.Split(cr, "/")
			if len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					return total
				}
			}
		}
		if resp.ContentLength > 0 {
			return startAt + resp.ContentLength
		}
	}
	return resp.ContentLength
}

func progressBar(done, total int64, width int) string {
	if width <= 0 {
		width = 20
	}
	if total <= 0 {
		return "[" + strings.Repeat("=", width/3) + ">" + strings.Repeat(" ", width-width/3-1) + "]"
	}
	ratio := float64(done) / float64(total)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(math.Round(ratio * float64(width)))
	if filled > width {
		filled = width
	}
	if filled == width {
		return "[" + strings.Repeat("=", width) + "]"
	}
	return "[" + strings.Repeat("=", filled) + ">" + strings.Repeat(" ", width-filled-1) + "]"
}

func coreHTTPClient() *http.Client {
	if os.Getenv("CLASH_TUI_INSECURE_TLS") == "1" {
		log.Printf("warning: CLASH_TUI_INSECURE_TLS=1 enabled, TLS cert verification is disabled for core download")
		return &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
		}
	}
	return &http.Client{}
}

func githubURLCandidates(raw string) []string {
	if !strings.HasPrefix(raw, "https://github.com/") && !strings.HasPrefix(raw, "https://api.github.com/") {
		return []string{raw}
	}
	out := []string{raw}
	if custom := strings.TrimSpace(os.Getenv("CLASH_TUI_GITHUB_PROXY")); custom != "" {
		out = append(out, strings.TrimRight(custom, "/")+"/"+raw)
	}
	for _, mirror := range defaultGitHubMirrors {
		out = append(out, strings.TrimRight(mirror, "/")+"/"+raw)
	}
	seen := make(map[string]struct{}, len(out))
	uniq := make([]string, 0, len(out))
	for _, u := range out {
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		uniq = append(uniq, u)
	}
	return uniq
}

package core

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const latestReleasePage = "https://github.com/MetaCubeX/mihomo/releases/latest"

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type Manager struct {
	dataDir    string
	execPath   string
	configPath string
	logPath    string
	pidPath    string

	mu           sync.Mutex
	cmd          *exec.Cmd
	killExpected map[*exec.Cmd]bool
}

func NewManager(dataDir string) (*Manager, error) {
	coreDir := filepath.Join(dataDir, "core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		return nil, err
	}
	name := "mihomo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	m := &Manager{
		dataDir:      dataDir,
		execPath:     filepath.Join(coreDir, name),
		configPath:   filepath.Join(dataDir, "mihomo-config.yaml"),
		logPath:      filepath.Join(dataDir, "mihomo.log"),
		pidPath:      filepath.Join(dataDir, "mihomo.pid"),
		killExpected: map[*exec.Cmd]bool{},
	}
	m.tryMigrateLegacyBinary(name)
	return m, nil
}

func (m *Manager) EnsureBinary(ctx context.Context) error {
	if st, err := os.Stat(m.execPath); err == nil && st.Size() > 0 {
		return nil
	}
	log.Printf("mihomo core not found, downloading...")
	rel, err := fetchLatestRelease(ctx)
	if err != nil {
		return err
	}
	assets, err := pickAssets(rel.Assets)
	if err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(m.execPath), "mihomo.download.part")
	var lastErr error
	for i, asset := range assets {
		log.Printf("downloading core asset (%d/%d): %s", i+1, len(assets), asset.Name)
		if err := downloadFile(ctx, asset.URL, tmp); err != nil {
			lastErr = err
			log.Printf("download failed for %s: %v", asset.Name, err)
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return lastErr
	}
	log.Printf("extracting core binary...")
	if err := extractBinary(tmp, m.execPath); err != nil {
		return err
	}
	_ = os.Remove(tmp)
	log.Printf("mihomo core ready: %s", m.execPath)
	return os.Chmod(m.execPath, 0o755)
}

func (m *Manager) tryMigrateLegacyBinary(binName string) {
	if st, err := os.Stat(m.execPath); err == nil && st.Size() > 0 {
		return
	}
	cacheBase, err := os.UserCacheDir()
	if err != nil {
		return
	}
	oldPath := filepath.Join(cacheBase, "clash-tui", "core", binName)
	st, err := os.Stat(oldPath)
	if err != nil || st.Size() <= 0 {
		return
	}
	in, err := os.Open(oldPath)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(m.execPath)
	if err != nil {
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return
	}
	_ = os.Chmod(m.execPath, 0o755)
	log.Printf("migrated existing core binary from cache: %s", oldPath)
}

func (m *Manager) WriteConfig(content string) error {
	if err := os.MkdirAll(filepath.Join(m.dataDir, "proxy_providers"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.configPath, []byte(content), 0o644)
}

func (m *Manager) Start(ctx context.Context) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		return nil
	}
	if pid, ok := m.readPIDLocked(); ok && processExists(pid) {
		log.Printf("reuse existing mihomo process pid=%d", pid)
		return nil
	}
	_ = os.Remove(m.pidPath)
	// Do not bind mihomo process lifecycle to request-scoped context.
	// It should only be stopped by explicit Stop/Restart, otherwise short
	// timeout contexts (e.g. UI reload) will kill the core unexpectedly.
	cmd := exec.Command(m.execPath, "-f", m.configPath)
	// Keep mihomo working directory stable so relative paths in config
	// (e.g. ./proxy_providers/*.yaml) are resolved correctly.
	cmd.Dir = m.dataDir
	logf, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("open mihomo log file failed, fallback to discard: %v", err)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	if err := cmd.Start(); err != nil {
		if logf != nil {
			_ = logf.Close()
		}
		return err
	}
	if err := os.WriteFile(m.pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		log.Printf("write mihomo pid file failed: %v", err)
	}
	m.cmd = cmd
	go func() {
		defer func() {
			if logf != nil {
				_ = logf.Close()
			}
		}()
		err := cmd.Wait()
		m.mu.Lock()
		expected := m.killExpected[cmd]
		delete(m.killExpected, cmd)
		if m.cmd == cmd {
			m.cmd = nil
		}
		m.cleanupPIDLocked(cmd.Process.Pid)
		m.mu.Unlock()
		_ = err
		_ = expected
	}()
	return nil
}

func (m *Manager) Restart(ctx context.Context) error {
	m.Stop()
	time.Sleep(200 * time.Millisecond)
	return m.Start(ctx)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		m.killExpected[m.cmd] = true
		_ = m.cmd.Process.Kill()
		m.cleanupPIDLocked(m.cmd.Process.Pid)
		m.cmd = nil
		return
	}
	if pid, ok := m.readPIDLocked(); ok && processExists(pid) {
		p, err := os.FindProcess(pid)
		if err == nil {
			_ = p.Kill()
		}
		m.cleanupPIDLocked(pid)
	}
	m.cmd = nil
}

func (m *Manager) readPIDLocked() (int, bool) {
	b, err := os.ReadFile(m.pidPath)
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, false
	}
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func (m *Manager) cleanupPIDLocked(pid int) {
	storedPID, ok := m.readPIDLocked()
	if !ok || storedPID != pid {
		return
	}
	_ = os.Remove(m.pidPath)
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func fetchLatestRelease(ctx context.Context) (release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleasePage, nil)
	if err != nil {
		return release{}, err
	}
	req.Header.Set("User-Agent", "clash-tui/1.x")
	resp, err := coreHTTPClient().Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return release{}, fmt.Errorf("fetch release page failed: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return release{}, err
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
	out := release{
		TagName: tag,
		Assets:  parseReleaseAssetsFromHTML(body),
	}
	if len(out.Assets) == 0 && out.TagName != "" {
		out.Assets = synthesizeAssetsByTag(out.TagName)
	}
	if len(out.Assets) == 0 {
		return release{}, fmt.Errorf("no downloadable assets found on release page (tag=%q)", out.TagName)
	}
	return out, nil
}

func parseTagFromHTML(html string) string {
	// e.g. /MetaCubeX/mihomo/releases/tag/v1.19.21
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
	// fallback for slightly different/escaped templates
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
		if strings.Contains(n, targetOS) && strings.Contains(n, targetArch) {
			candidates = append(candidates, a)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no release asset for %s/%s", targetOS, targetArch)
	}
	return candidates, nil
}

func downloadFile(ctx context.Context, src, dst string) error {
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
	// most mihomo assets are single gz-compressed executable
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
			// bytes start-end/total
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
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
	}
	return &http.Client{Timeout: 60 * time.Second}
}

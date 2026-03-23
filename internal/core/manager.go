package core

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const latestReleaseAPI = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"

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

	mu  sync.Mutex
	cmd *exec.Cmd
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
	return &Manager{
		dataDir:    dataDir,
		execPath:   filepath.Join(coreDir, name),
		configPath: filepath.Join(dataDir, "mihomo-config.yaml"),
	}, nil
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
	asset, err := pickAsset(rel.Assets)
	if err != nil {
		return err
	}
	log.Printf("downloading core asset: %s", asset.Name)
	tmp := filepath.Join(filepath.Dir(m.execPath), "mihomo.download")
	if err := downloadFile(ctx, asset.URL, tmp); err != nil {
		return err
	}
	log.Printf("extracting core binary...")
	if err := extractBinary(tmp, m.execPath); err != nil {
		return err
	}
	_ = os.Remove(tmp)
	log.Printf("mihomo core ready: %s", m.execPath)
	return os.Chmod(m.execPath, 0o755)
}

func (m *Manager) WriteConfig(content string) error {
	return os.WriteFile(m.configPath, []byte(content), 0o644)
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, m.execPath, "-f", m.configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.cmd == cmd {
			m.cmd = nil
		}
		m.mu.Unlock()
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
		_ = m.cmd.Process.Kill()
	}
	m.cmd = nil
}

func fetchLatestRelease(ctx context.Context) (release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		return release{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return release{}, fmt.Errorf("fetch release failed: %s", resp.Status)
	}
	var out release
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return release{}, err
	}
	return out, nil
}

func pickAsset(assets []releaseAsset) (releaseAsset, error) {
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
		return releaseAsset{}, fmt.Errorf("no release asset for %s/%s", targetOS, targetArch)
	}
	return candidates[0], nil
}

func downloadFile(ctx context.Context, src, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("download asset failed: %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	buf := make([]byte, 256*1024)
	var downloaded int64
	start := time.Now()
	lastLog := time.Time{}

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			downloaded += int64(n)

			now := time.Now()
			if lastLog.IsZero() || now.Sub(lastLog) >= 500*time.Millisecond {
				logDownloadProgress(downloaded, total, now.Sub(start))
				lastLog = now
			}
		}

		if readErr == io.EOF {
			logDownloadProgress(downloaded, total, time.Since(start))
			log.Printf("download complete")
			return nil
		}
		if readErr != nil {
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

func logDownloadProgress(done, total int64, elapsed time.Duration) {
	sec := elapsed.Seconds()
	if sec <= 0 {
		sec = 0.001
	}
	speed := float64(done) / sec
	if total > 0 {
		pct := float64(done) * 100 / float64(total)
		log.Printf("download progress: %5.1f%% (%s/%s) %s/s", pct, formatBytes(done), formatBytes(total), formatBytes(int64(speed)))
		return
	}
	log.Printf("download progress: %s %s/s", formatBytes(done), formatBytes(int64(speed)))
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

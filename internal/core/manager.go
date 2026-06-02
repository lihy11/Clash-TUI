package core

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"os/exec"
)

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
		if err := ctx.Err(); err != nil {
			return err
		}
		if i > 0 {
			_ = os.Remove(tmp)
		}
		for attempt := 1; attempt <= 3; attempt++ {
			log.Printf("downloading core asset (%d/%d, attempt %d/3): %s", i+1, len(assets), attempt, asset.Name)
			if err := downloadFile(ctx, asset.URL, tmp); err != nil {
				lastErr = err
				log.Printf("download failed for %s (attempt %d/3): %v", asset.Name, attempt, err)
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				if attempt < 3 {
					time.Sleep(time.Second)
					continue
				}
				break
			}
			lastErr = nil
			break
		}
		if lastErr == nil {
			break
		}
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

func (m *Manager) WriteConfig(content string) error {
	if err := os.MkdirAll(filepath.Join(m.dataDir, "proxy_providers"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.configPath, []byte(content), 0o644)
}

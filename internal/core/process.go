package core

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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

	cmd := exec.Command(m.execPath, "-f", m.configPath)
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

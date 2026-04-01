package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultEndpoint     = "http://127.0.0.1:9090"
	defaultPollInterval = 2 * time.Second
)

type Settings struct {
	Endpoint     string        `yaml:"endpoint"`
	Secret       string        `yaml:"secret"`
	PollInterval time.Duration `yaml:"poll_interval"`
	LogLevel     string        `yaml:"log_level"`
	ManageCore   bool          `yaml:"manage_core"`
	CoreVersion  string        `yaml:"core_version"`
	MixedPort    int           `yaml:"mixed_port"`
	Language     string        `yaml:"language"`
}

func Default() Settings {
	return Settings{
		Endpoint:     defaultEndpoint,
		Secret:       "",
		PollInterval: defaultPollInterval,
		LogLevel:     "info",
		ManageCore:   true,
		CoreVersion:  "latest",
		MixedPort:    7890,
		Language:     "en",
	}
}

func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "clash-tui", "config.yaml"), nil
}

func Load() (Settings, error) {
	cfg := Default()
	if env := os.Getenv("MIHOMO_CONTROLLER"); env != "" {
		cfg.Endpoint = env
	}
	if env := os.Getenv("MIHOMO_SECRET"); env != "" {
		cfg.Secret = env
	}

	p, err := configPath()
	if err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, Save(cfg)
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	if cfg.PollInterval < time.Second {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.MixedPort == 0 {
		cfg.MixedPort = 7890
	}
	switch cfg.Language {
	case "en", "zh-CN":
	default:
		cfg.Language = "en"
	}
	return cfg, nil
}

func Save(cfg Settings) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "clash-tui"), nil
}

func DataDir() (string, error) {
	if v := os.Getenv("CLASH_TUI_DATA_DIR"); v != "" {
		return v, nil
	}
	// Use config directory for runtime assets to avoid cache cleanups causing re-downloads.
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "clash-tui", "runtime"), nil
}

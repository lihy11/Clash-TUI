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
}

func Default() Settings {
	return Settings{
		Endpoint:     defaultEndpoint,
		Secret:       "",
		PollInterval: defaultPollInterval,
		LogLevel:     "info",
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

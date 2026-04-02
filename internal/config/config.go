package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logging LoggingConfig `yaml:"logging"`
}

type LoggingConfig struct {
	Level     string `yaml:"level"`
	Dir       string `yaml:"dir"`
	MaxSizeMB int    `yaml:"max_size_mb"`
	MaxFiles  int    `yaml:"max_files"`
}

func DefaultConfig() *Config {
	return &Config{
		Logging: LoggingConfig{
			Level:     "info",
			Dir:       defaultLogDir(),
			MaxSizeMB: 50,
			MaxFiles:  5,
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	path := configPath()
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("SYNAPSE_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("SYNAPSE_LOG_DIR"); v != "" {
		cfg.Logging.Dir = v
	}

	if cfg.Logging.Dir == "" {
		cfg.Logging.Dir = defaultLogDir()
	}

	return cfg, nil
}

func (c *LoggingConfig) SlogLevel() slog.Level {
	switch c.Level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "synapse", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "synapse", "config.yaml")
}

func defaultLogDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "synapse", "logs")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "synapse", "logs")
}

package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Logging.Level != "info" {
		t.Errorf("Level = %q, want %q", cfg.Logging.Level, "info")
	}
	if cfg.Logging.MaxSizeMB != 50 {
		t.Errorf("MaxSizeMB = %d, want 50", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxFiles != 5 {
		t.Errorf("MaxFiles = %d, want 5", cfg.Logging.MaxFiles)
	}
	if cfg.Logging.Dir == "" {
		t.Error("Dir should not be empty")
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "synapse")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yaml := []byte("logging:\n  level: debug\n  max_size_mb: 10\n  max_files: 3\n")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Logging.MaxSizeMB != 10 {
		t.Errorf("MaxSizeMB = %d, want 10", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxFiles != 3 {
		t.Errorf("MaxFiles = %d, want 3", cfg.Logging.MaxFiles)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYNAPSE_LOG_LEVEL", "error")
	t.Setenv("SYNAPSE_LOG_DIR", "/tmp/test-logs")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("Level = %q, want %q", cfg.Logging.Level, "error")
	}
	if cfg.Logging.Dir != "/tmp/test-logs" {
		t.Errorf("Dir = %q, want %q", cfg.Logging.Dir, "/tmp/test-logs")
	}
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := &LoggingConfig{Level: tt.level}
			if got := cfg.SlogLevel(); got != tt.want {
				t.Errorf("SlogLevel(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestLoadMissingConfigFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Level = %q, want %q", cfg.Logging.Level, "info")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "synapse")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(":{bad yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", dir)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadEmptyDirFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "synapse")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// YAML with empty dir field
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("logging:\n  dir: \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.Dir == "" {
		t.Error("Dir should fall back to default, not be empty")
	}
}

func TestConfigPathWithoutXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	path := configPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "synapse", "config.yaml")
	if path != want {
		t.Errorf("configPath() = %q, want %q", path, want)
	}
}

func TestDefaultLogDirWithoutXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")

	dir := defaultLogDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".local", "share", "synapse", "logs")
	if dir != want {
		t.Errorf("defaultLogDir() = %q, want %q", dir, want)
	}
}

func TestDefaultLogDirWithXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")

	dir := defaultLogDir()
	want := filepath.Join("/custom/data", "synapse", "logs")
	if dir != want {
		t.Errorf("defaultLogDir() = %q, want %q", dir, want)
	}
}

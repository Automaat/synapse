package logging

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Automaat/synapse/internal/config"
)

func New(cfg config.LoggingConfig) (*slog.Logger, func(), error) {
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, nil, err
	}

	path := filepath.Join(cfg.Dir, "synapse.log")
	maxBytes := int64(cfg.MaxSizeMB) * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = 50 * 1024 * 1024
	}
	maxFiles := cfg.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 5
	}

	w, err := NewRotatingWriter(path, maxBytes, maxFiles)
	if err != nil {
		return nil, nil, err
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: cfg.SlogLevel(),
	})
	logger := slog.New(handler)

	cleanup := func() {
		_ = w.Close()
	}

	return logger, cleanup, nil
}

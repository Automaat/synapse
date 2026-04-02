package logging

import (
	"os"
	"path/filepath"
	"time"
)

func NewAgentOutputFile(logDir, agentID string) (*os.File, error) {
	dir := filepath.Join(logDir, "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	ts := time.Now().UTC().Format("2006-01-02T15-04-05")
	name := agentID + "-" + ts + ".ndjson"
	return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

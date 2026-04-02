package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type claudeSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

var nonAlphanumDash = regexp.MustCompile(`[^a-zA-Z0-9-]`)

func (m *Manager) DiscoverAgents() []*Agent {
	sessions := readClaudeSessions()

	m.mu.RLock()
	trackedPIDs := make(map[int]bool)
	for _, a := range m.agents {
		if a.cmd != nil && a.cmd.Process != nil {
			trackedPIDs[a.cmd.Process.Pid] = true
		}
		if a.PID != 0 {
			trackedPIDs[a.PID] = true
		}
	}
	m.mu.RUnlock()

	// Refresh state of already-tracked external agents
	m.mu.Lock()
	for _, a := range m.agents {
		if !a.External {
			continue
		}
		if !processAlive(a.PID) {
			a.State = StateStopped
		} else {
			a.State = inferState(a.sessionCWD, a.SessionID)
		}
	}
	m.mu.Unlock()

	var discovered []*Agent
	for _, s := range sessions {
		if trackedPIDs[s.PID] {
			continue
		}

		if !processAlive(s.PID) {
			continue
		}

		a := &Agent{
			ID:         fmt.Sprintf("ext-%d", s.PID),
			Mode:       sessionKind(s.Kind),
			State:      inferState(s.CWD, s.SessionID),
			External:   true,
			PID:        s.PID,
			SessionID:  s.SessionID,
			StartedAt:  time.UnixMilli(s.StartedAt).UTC(),
			Name:       s.Name,
			Project:    projectName(s.CWD),
			sessionCWD: s.CWD,
		}

		m.mu.Lock()
		if _, exists := m.agents[a.ID]; !exists {
			m.agents[a.ID] = a
		}
		m.mu.Unlock()

		discovered = append(discovered, a)
	}
	return discovered
}

func inferState(cwd, sessionID string) State {
	lastType := lastMessageType(cwd, sessionID)
	switch lastType {
	case "system", "":
		return StateIdle
	default:
		return StateRunning
	}
}

func lastMessageType(cwd, sessionID string) string {
	if cwd == "" || sessionID == "" {
		return ""
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	projectKey := nonAlphanumDash.ReplaceAllString(cwd, "-")
	jsonlPath := filepath.Join(home, ".claude", "projects", projectKey, sessionID+".jsonl")

	return readLastJSONLType(jsonlPath)
}

func readLastJSONLType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	// Seek near end of file and scan for last line
	info, err := f.Stat()
	if err != nil {
		return ""
	}

	// Read last 8KB — enough for the last JSONL entry
	offset := max(info.Size()-8192, 0)
	if _, err := f.Seek(offset, 0); err != nil {
		return ""
	}

	var lastLine string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lastLine = line
		}
	}

	if lastLine == "" {
		return ""
	}

	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(lastLine), &msg); err != nil {
		return ""
	}
	return msg.Type
}

func readClaudeSessions() []claudeSession {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	dir := filepath.Join(home, ".claude", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var sessions []claudeSession
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		var s claudeSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if s.PID == 0 {
			continue
		}

		sessions = append(sessions, s)
	}
	return sessions
}

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func projectName(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Base(cwd)
}

func sessionKind(kind string) string {
	if kind == "headless" {
		return "headless"
	}
	return "interactive"
}

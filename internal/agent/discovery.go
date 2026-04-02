package agent

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type discoveredProcess struct {
	PID     int
	Command string
	Started time.Time
}

func (m *Manager) DiscoverAgents() []*Agent {
	var discovered []*Agent

	discovered = append(discovered, m.discoverTmuxAgents()...)
	discovered = append(discovered, m.discoverHeadlessAgents()...)

	return discovered
}

func (m *Manager) discoverTmuxAgents() []*Agent {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return nil
	}

	m.mu.RLock()
	tracked := make(map[string]bool)
	for _, a := range m.agents {
		if a.TmuxSession != "" {
			tracked[a.TmuxSession] = true
		}
	}
	m.mu.RUnlock()

	var agents []*Agent
	for _, s := range sessions {
		if tracked[s.Name] {
			continue
		}

		paneCmd, err := m.tmux.PaneCommand(s.Name)
		if err != nil {
			continue
		}
		if !isClaude(paneCmd) {
			continue
		}

		a := &Agent{
			ID:          fmt.Sprintf("ext-%s", sanitizeID(s.Name)),
			Mode:        "interactive",
			State:       StateRunning,
			TmuxSession: s.Name,
			External:    true,
			StartedAt:   time.Now().UTC(),
			Command:     paneCmd,
		}

		m.mu.Lock()
		if _, exists := m.agents[a.ID]; !exists {
			m.agents[a.ID] = a
		}
		m.mu.Unlock()

		agents = append(agents, a)
	}
	return agents
}

func (m *Manager) discoverHeadlessAgents() []*Agent {
	procs := findClaudeProcesses()

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

	var agents []*Agent
	for _, p := range procs {
		if trackedPIDs[p.PID] {
			continue
		}

		a := &Agent{
			ID:        fmt.Sprintf("ext-%d", p.PID),
			Mode:      "headless",
			State:     StateRunning,
			External:  true,
			PID:       p.PID,
			StartedAt: p.Started,
			Command:   p.Command,
		}

		m.mu.Lock()
		if _, exists := m.agents[a.ID]; !exists {
			m.agents[a.ID] = a
		}
		m.mu.Unlock()

		agents = append(agents, a)
	}
	return agents
}

func findClaudeProcesses() []discoveredProcess {
	out, err := exec.Command("ps", "-eo", "pid,lstart,command").CombinedOutput()
	if err != nil {
		return nil
	}

	var procs []discoveredProcess
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "claude") || strings.Contains(line, "grep") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		// lstart format: "Day Mon DD HH:MM:SS YYYY" (5 fields)
		timeStr := strings.Join(fields[1:6], " ")
		started, _ := time.Parse("Mon Jan 2 15:04:05 2006", timeStr)

		cmd := strings.Join(fields[6:], " ")

		// Only match actual claude CLI processes, not editors or this app
		if !isClaude(cmd) {
			continue
		}

		procs = append(procs, discoveredProcess{
			PID:     pid,
			Command: cmd,
			Started: started,
		})
	}
	return procs
}

func isClaude(cmd string) bool {
	lower := strings.ToLower(cmd)
	return strings.Contains(lower, "claude") &&
		!strings.Contains(lower, "synapse") &&
		!strings.Contains(lower, "claude-code-guide")
}

func sanitizeID(name string) string {
	r := strings.NewReplacer("/", "-", " ", "-", ".", "-")
	id := r.Replace(name)
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

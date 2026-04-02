package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Automaat/synapse/internal/logging"
)

func (m *Manager) runHeadless(ctx context.Context, a *Agent, prompt string, allowedTools []string) {
	args := []string{"-p", prompt, "--output-format", "stream-json"}

	if a.SessionID != "" {
		args = append(args, "--resume", a.SessionID)
	}

	if len(allowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(allowedTools, ","))
	} else {
		args = append(args, "--dangerously-skip-permissions")
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	a.cmd = cmd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.handleError(a, fmt.Errorf("stdout pipe: %w", err))
		return
	}

	if err := cmd.Start(); err != nil {
		m.handleError(a, fmt.Errorf("start claude: %w", err))
		return
	}

	m.logger.Info("agent.headless.start", "id", a.ID, "pid", cmd.Process.Pid)

	outFile, fileErr := logging.NewAgentOutputFile(m.logDir, a.ID)
	if fileErr != nil {
		m.logger.Error("agent.output.file", "id", a.ID, "err", fileErr)
	}
	if outFile != nil {
		defer func() { _ = outFile.Close() }()
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		if outFile != nil {
			_, _ = outFile.Write(line)
			_, _ = outFile.WriteString("\n")
		}

		var event StreamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		a.outputBuffer = append(a.outputBuffer, event)
		m.emit("agent:output:"+a.ID, event)

		if event.Type == "result" {
			a.SessionID = event.SessionID
			a.CostUSD += event.CostUSD
			m.logger.Info("agent.headless.result", "id", a.ID, "session_id", event.SessionID, "cost", a.CostUSD)
		}
	}

	_ = cmd.Wait()

	a.State = StateStopped
	m.logger.Info("agent.headless.done", "id", a.ID, "cost", a.CostUSD)
	m.emit("agent:state:"+a.ID, a)
}

func (m *Manager) handleError(a *Agent, err error) {
	a.State = StateStopped
	m.logger.Error("agent.error", "id", a.ID, "err", err)
	m.emit("agent:error:"+a.ID, err.Error())
}

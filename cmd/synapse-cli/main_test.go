package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Automaat/synapse/internal/task"
)

func mustUnmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func setupStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SYNAPSE_TASKS_DIR", dir)
	return dir
}

func runCLI(t *testing.T, args ...string) (exitCode int, output string) {
	t.Helper()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code := run(args)

	_ = w.Close()
	os.Stdout = old

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	return code, string(buf[:n])
}

func TestListEmpty(t *testing.T) {
	setupStore(t)
	code, out := runCLI(t, "--json", "list")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var tasks []task.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestCreateAndGet(t *testing.T) {
	setupStore(t)

	code, out := runCLI(t, "--json", "create", "--title", "test task", "--body", "body text", "--tags", "a,b")
	if code != 0 {
		t.Fatalf("create exit %d: %s", code, out)
	}

	var created task.Task
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if created.Title != "test task" {
		t.Errorf("title = %q", created.Title)
	}
	if created.Body != "body text" {
		t.Errorf("body = %q", created.Body)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "a" || created.Tags[1] != "b" {
		t.Errorf("tags = %v", created.Tags)
	}

	code, out = runCLI(t, "--json", "get", created.ID)
	if code != 0 {
		t.Fatalf("get exit %d: %s", code, out)
	}
	var got task.Task
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("get returned id %q, want %q", got.ID, created.ID)
	}
}

func TestUpdateStatus(t *testing.T) {
	setupStore(t)

	code, out := runCLI(t, "--json", "create", "--title", "update me")
	if code != 0 {
		t.Fatalf("create exit %d", code)
	}
	var created task.Task
	mustUnmarshal(t, out, &created)

	code, out = runCLI(t, "--json", "update", created.ID, "--status", "in-progress")
	if code != 0 {
		t.Fatalf("update exit %d: %s", code, out)
	}
	var updated task.Task
	mustUnmarshal(t, out, &updated)
	if updated.Status != "in-progress" {
		t.Errorf("status = %q", updated.Status)
	}
}

func TestDelete(t *testing.T) {
	setupStore(t)

	code, out := runCLI(t, "--json", "create", "--title", "delete me")
	if code != 0 {
		t.Fatalf("create exit %d", code)
	}
	var created task.Task
	mustUnmarshal(t, out, &created)

	code, _ = runCLI(t, "--json", "delete", created.ID)
	if code != 0 {
		t.Fatalf("delete exit %d", code)
	}

	code, out = runCLI(t, "--json", "list")
	if code != 0 {
		t.Fatalf("list exit %d", code)
	}
	var tasks []task.Task
	mustUnmarshal(t, out, &tasks)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after delete, got %d", len(tasks))
	}
}

func TestListFilterStatus(t *testing.T) {
	setupStore(t)

	runCLI(t, "--json", "create", "--title", "task1")
	_, out := runCLI(t, "--json", "create", "--title", "task2")
	var t2 task.Task
	mustUnmarshal(t, out, &t2)
	runCLI(t, "--json", "update", t2.ID, "--status", "in-progress")

	code, out := runCLI(t, "--json", "list", "--status", "new")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var tasks []task.Task
	mustUnmarshal(t, out, &tasks)
	if len(tasks) != 1 {
		t.Errorf("expected 1 new task, got %d", len(tasks))
	}
}

func TestListFilterTag(t *testing.T) {
	setupStore(t)

	runCLI(t, "--json", "create", "--title", "tagged", "--tags", "api,backend")
	runCLI(t, "--json", "create", "--title", "untagged")

	code, out := runCLI(t, "--json", "list", "--tag", "api")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var tasks []task.Task
	mustUnmarshal(t, out, &tasks)
	if len(tasks) != 1 {
		t.Errorf("expected 1 tagged task, got %d", len(tasks))
	}
}

func TestUnknownCommand(t *testing.T) {
	setupStore(t)
	code, _ := runCLI(t, "bogus")
	if code == 0 {
		t.Error("expected non-zero exit for unknown command")
	}
}

func TestNoArgs(t *testing.T) {
	code, _ := runCLI(t)
	if code == 0 {
		t.Error("expected non-zero exit for no args")
	}
}

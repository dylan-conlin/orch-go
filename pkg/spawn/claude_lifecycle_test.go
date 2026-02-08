package spawn

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/internal/testutil"
)

func TestSpawnClaude_Lifecycle_NoLeakedTmuxWindowsAcrossInstances(t *testing.T) {
	restore := stubTmuxLifecycle(t)
	defer restore()

	activeWindows := make(map[string]bool)
	createTmuxWindow = func(sessionName, windowName, projectDir string) (string, string, error) {
		target := fmt.Sprintf("%s:%s", sessionName, windowName)
		activeWindows[target] = true
		return target, fmt.Sprintf("id-%s", windowName), nil
	}
	killTmuxWindow = func(windowTarget string) error {
		delete(activeWindows, windowTarget)
		return nil
	}

	const instances = 4
	results := make([]*tmuxSpawnResult, 0, instances)
	for i := 0; i < instances; i++ {
		cfg := &Config{
			Project:       "orch-go",
			ProjectDir:    "/tmp/orch-go",
			WorkspaceName: fmt.Sprintf("ws-%d", i),
			SkillName:     "feature-impl",
			BeadsID:       fmt.Sprintf("orch-go-%d", i),
		}

		res, err := SpawnClaude(cfg)
		if err != nil {
			t.Fatalf("SpawnClaude instance %d failed: %v", i, err)
		}
		results = append(results, &tmuxSpawnResult{windowTarget: res.Window})
	}

	if len(activeWindows) != instances {
		t.Fatalf("expected %d active windows, got %d", instances, len(activeWindows))
	}

	for _, res := range results {
		if err := AbandonClaude(res.windowTarget); err != nil {
			t.Fatalf("AbandonClaude failed for %s: %v", res.windowTarget, err)
		}
	}

	if len(activeWindows) != 0 {
		t.Fatalf("expected 0 active windows after abandon, got %d", len(activeWindows))
	}
}

func TestMonitorAndSendClaude_UsesTmuxAdapters(t *testing.T) {
	restore := stubTmuxLifecycle(t)
	defer restore()

	var (
		receivedTarget string
		receivedKeys   string
		enterCount     int
	)

	getTmuxPaneContent = func(windowTarget string) (string, error) {
		receivedTarget = windowTarget
		return "pane-output", nil
	}
	sendTmuxKeysLiteral = func(windowTarget, keys string) error {
		receivedTarget = windowTarget
		receivedKeys = keys
		return nil
	}
	sendTmuxEnter = func(windowTarget string) error {
		receivedTarget = windowTarget
		enterCount++
		return nil
	}

	content, err := MonitorClaude("win-1")
	if err != nil {
		t.Fatalf("MonitorClaude failed: %v", err)
	}
	if content != "pane-output" {
		t.Fatalf("expected pane-output, got %q", content)
	}

	if err := SendClaude("win-2", "hello"); err != nil {
		t.Fatalf("SendClaude failed: %v", err)
	}

	if receivedTarget != "win-2" {
		t.Fatalf("expected target win-2, got %s", receivedTarget)
	}
	if receivedKeys != "hello" {
		t.Fatalf("expected keys hello, got %s", receivedKeys)
	}
	if enterCount != 1 {
		t.Fatalf("expected one enter, got %d", enterCount)
	}
}

func TestAbandonClaude_PropagatesTmuxFailure(t *testing.T) {
	restore := stubTmuxLifecycle(t)
	defer restore()

	killTmuxWindow = func(windowTarget string) error {
		return errors.New("tmux kill failed")
	}

	err := AbandonClaude("win-3")
	if err == nil || !strings.Contains(err.Error(), "failed to kill tmux window") {
		t.Fatalf("expected wrapped kill error, got %v", err)
	}
}

func TestSpawnClaudeInline_Lifecycle_NoLeakedProcessAcrossInstances(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	pidLog := filepath.Join(tmpDir, "claude-pids.log")
	claudeScript := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claudeScript, []byte("#!/bin/sh\necho \"$$\" >> \"$ORCH_TEST_PID_LOG\"\ncat >/dev/null\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake claude script: %v", err)
	}

	t.Setenv("ORCH_TEST_PID_LOG", pidLog)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	projectDir := filepath.Join(tmpDir, "project")
	startGoroutines := runtime.NumGoroutine()

	const instances = 5
	for i := 0; i < instances; i++ {
		cfg := &Config{
			ProjectDir:     projectDir,
			WorkspaceName:  fmt.Sprintf("ws-%d", i),
			IsOrchestrator: false,
		}

		workspacePath := cfg.WorkspacePath()
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			t.Fatalf("failed to create workspace %d: %v", i, err)
		}
		if err := os.WriteFile(cfg.ContextFilePath(), []byte("context\n"), 0644); err != nil {
			t.Fatalf("failed to write context file %d: %v", i, err)
		}

		if err := SpawnClaudeInline(cfg); err != nil {
			t.Fatalf("SpawnClaudeInline instance %d failed: %v", i, err)
		}
	}

	pids := readPIDsFromLog(t, pidLog)
	if len(pids) != instances {
		t.Fatalf("expected %d pids, got %d", instances, len(pids))
	}

	for _, pid := range pids {
		waitForProcessExit(t, pid, "spawned claude helper")
	}

	testutil.WaitForWithTimeout(t, func() bool {
		return runtime.NumGoroutine() <= startGoroutines+2
	}, 2*time.Second, "goroutine count to return to baseline")
}

func TestSpawnClaudeInline_WriteFailureKillsChildProcess(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	pidLog := filepath.Join(tmpDir, "claude-failure-pids.log")
	claudeScript := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claudeScript, []byte("#!/bin/sh\necho \"$$\" >> \"$ORCH_TEST_PID_LOG\"\nexec 0<&-\nsleep 3\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake claude script: %v", err)
	}

	t.Setenv("ORCH_TEST_PID_LOG", pidLog)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	projectDir := filepath.Join(tmpDir, "project")
	cfg := &Config{
		ProjectDir:     projectDir,
		WorkspaceName:  "ws-failure",
		IsOrchestrator: false,
	}

	if err := os.MkdirAll(cfg.WorkspacePath(), 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	if err := os.WriteFile(cfg.ContextFilePath(), []byte(strings.Repeat("x", 1<<20)), 0644); err != nil {
		t.Fatalf("failed to write context file: %v", err)
	}

	err := SpawnClaudeInline(cfg)
	if err == nil {
		t.Fatal("expected SpawnClaudeInline to fail when child closes stdin")
	}

	pids := readPIDsFromLog(t, pidLog)
	if len(pids) != 1 {
		t.Fatalf("expected 1 pid, got %d", len(pids))
	}

	waitForProcessExit(t, pids[0], "failed claude helper")
}

func readPIDsFromLog(t *testing.T, path string) []int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read pid log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	pids := make([]int, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			t.Fatalf("failed to parse pid %q: %v", line, err)
		}
		pids = append(pids, pid)
	}
	return pids
}

func waitForProcessExit(t *testing.T, pid int, description string) {
	t.Helper()
	testutil.WaitForWithTimeout(t, func() bool {
		return !processExists(pid)
	}, 2*time.Second, description+" to exit")
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false
	}
	return true
}

type tmuxSpawnResult struct {
	windowTarget string
}

func stubTmuxLifecycle(t *testing.T) func() {
	t.Helper()

	oldEnsureOrchestratorSession := ensureOrchestratorSession
	oldEnsureWorkersSession := ensureWorkersSession
	oldBuildTmuxWindowName := buildTmuxWindowName
	oldCreateTmuxWindow := createTmuxWindow
	oldSendTmuxKeys := sendTmuxKeys
	oldSendTmuxKeysLiteral := sendTmuxKeysLiteral
	oldSendTmuxEnter := sendTmuxEnter
	oldKillTmuxWindow := killTmuxWindow
	oldGetTmuxPaneContent := getTmuxPaneContent

	ensureOrchestratorSession = func() (string, error) { return "orchestrator", nil }
	ensureWorkersSession = func(project, projectDir string) (string, error) { return "workers-" + project, nil }
	buildTmuxWindowName = func(workspaceName, skillName, beadsID string) string { return workspaceName }
	createTmuxWindow = func(sessionName, windowName, projectDir string) (string, string, error) {
		return sessionName + ":" + windowName, "win-id", nil
	}
	sendTmuxKeys = func(windowTarget, keys string) error { return nil }
	sendTmuxKeysLiteral = func(windowTarget, keys string) error { return nil }
	sendTmuxEnter = func(windowTarget string) error { return nil }
	killTmuxWindow = func(windowTarget string) error { return nil }
	getTmuxPaneContent = func(windowTarget string) (string, error) { return "", nil }

	return func() {
		ensureOrchestratorSession = oldEnsureOrchestratorSession
		ensureWorkersSession = oldEnsureWorkersSession
		buildTmuxWindowName = oldBuildTmuxWindowName
		createTmuxWindow = oldCreateTmuxWindow
		sendTmuxKeys = oldSendTmuxKeys
		sendTmuxKeysLiteral = oldSendTmuxKeysLiteral
		sendTmuxEnter = oldSendTmuxEnter
		killTmuxWindow = oldKillTmuxWindow
		getTmuxPaneContent = oldGetTmuxPaneContent
	}
}

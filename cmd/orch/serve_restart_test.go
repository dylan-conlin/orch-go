package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestartManagedOrchServe_Overmind(t *testing.T) {
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "calls.log")
	projectDir := t.TempDir()

	writeRestartScript(t, filepath.Join(binDir, "overmind"), `
echo "overmind:$*" >> "$ORCH_TEST_CALL_LOG"
if [ "$1" = "status" ]; then
  exit 0
fi
if [ "$1" = "restart" ] && [ "$2" = "api" ]; then
  exit 0
fi
exit 1
`)
	writeRestartScript(t, filepath.Join(binDir, "launchctl"), "exit 1")
	writeRestartScript(t, filepath.Join(binDir, "pgrep"), "exit 1")

	t.Setenv("ORCH_TEST_CALL_LOG", logPath)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	result, err := restartManagedOrchServe(projectDir)
	if err != nil {
		t.Fatalf("restartManagedOrchServe returned error: %v", err)
	}
	if !result.Restarted {
		t.Fatal("expected restart to occur")
	}
	if result.Method != "overmind" {
		t.Fatalf("expected method overmind, got %q", result.Method)
	}

	calls := readRestartLog(t, logPath)
	if !strings.Contains(calls, "overmind:status") {
		t.Fatalf("expected overmind status call, got %q", calls)
	}
	if !strings.Contains(calls, "overmind:restart api") {
		t.Fatalf("expected overmind restart call, got %q", calls)
	}
}

func TestRestartManagedOrchServe_LaunchdFallback(t *testing.T) {
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "calls.log")
	projectDir := t.TempDir()

	writeRestartScript(t, filepath.Join(binDir, "overmind"), `
echo "overmind:$*" >> "$ORCH_TEST_CALL_LOG"
exit 1
`)
	writeRestartScript(t, filepath.Join(binDir, "launchctl"), `
echo "launchctl:$*" >> "$ORCH_TEST_CALL_LOG"
if [ "$1" = "print" ]; then
  case "$2" in
    *com.overmind.orch-go) exit 0 ;;
  esac
fi
if [ "$1" = "kickstart" ]; then
  case "$3" in
    *com.overmind.orch-go) exit 0 ;;
  esac
fi
exit 1
`)
	writeRestartScript(t, filepath.Join(binDir, "pgrep"), "exit 1")

	t.Setenv("ORCH_TEST_CALL_LOG", logPath)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	result, err := restartManagedOrchServe(projectDir)
	if err != nil {
		t.Fatalf("restartManagedOrchServe returned error: %v", err)
	}
	if !result.Restarted {
		t.Fatal("expected restart to occur")
	}
	if result.Method != "launchctl:com.overmind.orch-go" {
		t.Fatalf("expected launchctl method, got %q", result.Method)
	}

	target := fmt.Sprintf("gui/%d/com.overmind.orch-go", os.Getuid())
	calls := readRestartLog(t, logPath)
	if !strings.Contains(calls, "launchctl:print "+target) {
		t.Fatalf("expected launchctl print call, got %q", calls)
	}
	if !strings.Contains(calls, "launchctl:kickstart -k "+target) {
		t.Fatalf("expected launchctl kickstart call, got %q", calls)
	}
}

func TestRestartManagedOrchServe_UnmanagedProcess(t *testing.T) {
	binDir := t.TempDir()
	projectDir := t.TempDir()

	writeRestartScript(t, filepath.Join(binDir, "overmind"), "exit 1")
	writeRestartScript(t, filepath.Join(binDir, "launchctl"), "exit 1")
	writeRestartScript(t, filepath.Join(binDir, "pgrep"), `
echo "123"
echo "456"
exit 0
`)

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	_, err := restartManagedOrchServe(projectDir)
	if err == nil {
		t.Fatal("expected unmanaged restart error")
	}
	if !strings.Contains(err.Error(), "unmanaged orch serve process") {
		t.Fatalf("expected unmanaged process message, got %v", err)
	}
}

func TestRestartManagedOrchServe_NoManagerNoProcess(t *testing.T) {
	binDir := t.TempDir()
	projectDir := t.TempDir()

	writeRestartScript(t, filepath.Join(binDir, "overmind"), "exit 1")
	writeRestartScript(t, filepath.Join(binDir, "launchctl"), "exit 1")
	writeRestartScript(t, filepath.Join(binDir, "pgrep"), "exit 1")

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	result, err := restartManagedOrchServe(projectDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Restarted {
		t.Fatal("expected no restart")
	}
}

func writeRestartScript(t *testing.T, path, body string) {
	t.Helper()
	content := "#!/bin/sh\n" + strings.TrimSpace(body) + "\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write script %s: %v", path, err)
	}
}

func readRestartLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read call log: %v", err)
	}
	return string(data)
}

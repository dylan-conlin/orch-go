package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/internal/testutil"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

func TestRegeneratePlistAndKickDaemon_Lifecycle_NoLeakedProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("failed to create home dir: %v", err)
	}
	t.Setenv("HOME", homeDir)

	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	pidLog := filepath.Join(tmpDir, "launchctl-pids.log")
	launchctlScript := filepath.Join(binDir, "launchctl")
	if err := os.WriteFile(launchctlScript, []byte("#!/bin/sh\necho \"$$\" >> \"$ORCH_TEST_PID_LOG\"\nsleep 0.05\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake launchctl script: %v", err)
	}

	t.Setenv("ORCH_TEST_PID_LOG", pidLog)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	// Seed config for deterministic plist generation in temp HOME.
	cfg := userconfig.DefaultConfig()
	if err := userconfig.Save(cfg); err != nil {
		t.Fatalf("failed to save user config: %v", err)
	}

	const instances = 4
	for i := 0; i < instances; i++ {
		if err := regeneratePlistAndKickDaemon(); err != nil {
			t.Fatalf("regeneratePlistAndKickDaemon instance %d failed: %v", i, err)
		}
	}

	plistPath := getPlistPath()
	if _, err := os.Stat(plistPath); err != nil {
		t.Fatalf("expected plist at %s: %v", plistPath, err)
	}

	pids := readLifecyclePIDs(t, pidLog)
	if len(pids) != instances {
		t.Fatalf("expected %d launchctl invocations, got %d", instances, len(pids))
	}

	for _, pid := range pids {
		waitForLifecycleProcessExit(t, pid, "launchctl helper")
	}
}

func readLifecyclePIDs(t *testing.T, path string) []int {
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

func waitForLifecycleProcessExit(t *testing.T, pid int, description string) {
	t.Helper()
	testutil.WaitForWithTimeout(t, func() bool {
		return !lifecycleProcessExists(pid)
	}, 2*time.Second, description+" to exit")
}

func lifecycleProcessExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false
	}
	return true
}

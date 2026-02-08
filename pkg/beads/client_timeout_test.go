package beads

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunBDCommand_TimeoutBudgetUnaffectedByAcquireWait(t *testing.T) {
	oldSem := bdSubprocessSem
	bdSubprocessSem = make(chan struct{}, 1)
	t.Cleanup(func() {
		bdSubprocessSem = oldSem
	})

	// Occupy the only slot so runBDCommand must wait before it can execute.
	bdSubprocessSem <- struct{}{}
	go func() {
		time.Sleep(9 * time.Second)
		<-bdSubprocessSem
	}()

	scriptPath := filepath.Join(t.TempDir(), "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"sleep 2",
		"printf 'ok'",
	}, "\n") + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	start := time.Now()
	output, err := runBDCommand("", scriptPath, nil, false, "stats")
	if err != nil {
		t.Fatalf("runBDCommand returned error: %v", err)
	}
	if string(output) != "ok" {
		t.Fatalf("runBDCommand output = %q, want %q", string(output), "ok")
	}

	elapsed := time.Since(start)
	if elapsed < 10*time.Second {
		t.Fatalf("runBDCommand elapsed = %v, want >= 10s to prove wait + full execution budget", elapsed)
	}
}

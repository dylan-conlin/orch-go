package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestRunBdCommandOutput_PrependsSandbox(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	scriptPath := filepath.Join(t.TempDir(), "fake-bd.sh")

	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s' \"$*\" > \"$ARGS_FILE\"",
		"printf '[]'",
	}, "\n") + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	oldBdPath := beads.BdPath
	beads.BdPath = scriptPath
	t.Cleanup(func() {
		beads.BdPath = oldBdPath
	})

	t.Setenv("ARGS_FILE", argsFile)

	if _, err := runBdCommandOutput("", "list", "--json"); err != nil {
		t.Fatalf("runBdCommandOutput returned error: %v", err)
	}

	argsBytes, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	args := strings.TrimSpace(string(argsBytes))
	if !strings.HasPrefix(args, "--sandbox ") {
		t.Fatalf("expected args to start with --sandbox, got %q", args)
	}
}

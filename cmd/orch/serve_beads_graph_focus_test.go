package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestBuildFocusGraphAvoidsPerIssueShellouts(t *testing.T) {
	tmpDir := t.TempDir()
	cmdLogPath := filepath.Join(tmpDir, "bd-calls.log")
	scriptPath := filepath.Join(tmpDir, "fake-bd.sh")

	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\n' \"$*\" >> \"$BD_CALL_LOG\"",
		"cmd=\"$2\"",
		"if [ \"$cmd\" = \"list\" ]; then",
		"  case \" $* \" in",
		"    *\" --status open \"*)",
		"      printf '[{\"id\":\"orch-go-1\",\"title\":\"Open p0\",\"status\":\"open\",\"priority\":0,\"issue_type\":\"task\",\"dependency_count\":0,\"dependent_count\":1},{\"id\":\"orch-go-3\",\"title\":\"Open p3\",\"status\":\"open\",\"priority\":3,\"issue_type\":\"task\",\"dependency_count\":1,\"dependent_count\":0}]'",
		"      ;;",
		"    *\" --status in_progress \"*)",
		"      printf '[{\"id\":\"orch-go-2\",\"title\":\"In progress\",\"status\":\"in_progress\",\"priority\":2,\"issue_type\":\"task\",\"dependency_count\":1,\"dependent_count\":0}]'",
		"      ;;",
		"    *\" --all \"*)",
		"      printf '[]'",
		"      ;;",
		"    *)",
		"      printf '[]'",
		"      ;;",
		"  esac",
		"  exit 0",
		"fi",
		"if [ \"$cmd\" = \"graph\" ]; then",
		"  printf '{\"edges\":[{\"from\":\"orch-go-2\",\"to\":\"orch-go-1\",\"type\":\"blocks\"},{\"from\":\"orch-go-2\",\"to\":\"orch-go-3\",\"type\":\"relates_to\"}]}'",
		"  exit 0",
		"fi",
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

	t.Setenv("BD_CALL_LOG", cmdLogPath)

	srv := newTestServer()
	srv.BeadsStatsCache = nil

	nodes, edges, err := srv.buildFocusGraph(tmpDir, "")
	if err != nil {
		t.Fatalf("buildFocusGraph returned error: %v", err)
	}

	if len(nodes) != 3 {
		t.Fatalf("expected 3 focus nodes, got %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 focus edges, got %d", len(edges))
	}

	logData, err := os.ReadFile(cmdLogPath)
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	logText := string(logData)

	if strings.Contains(logText, " dep ") {
		t.Fatalf("expected no per-issue dep shell-outs, got log:\n%s", logText)
	}
	if strings.Contains(logText, " show ") {
		t.Fatalf("expected no per-issue show shell-outs, got log:\n%s", logText)
	}
	if strings.Contains(logText, " list --json --limit 0 --all") {
		t.Fatalf("expected no all-issues fallback list for this dataset, got log:\n%s", logText)
	}
}

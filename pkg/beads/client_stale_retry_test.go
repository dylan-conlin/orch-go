package beads

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRunBDCommand_RetriesAllowStaleWhenImportRecent(t *testing.T) {
	workDir := t.TempDir()
	writeLastImportTime(t, workDir, time.Now().Add(-5*time.Second))

	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\n' \"$*\" >> \"" + invocations + "\"",
		"case \" $* \" in",
		"  *\" --allow-stale \"*)",
		"    printf 'ok'",
		"    exit 0",
		"    ;;",
		"esac",
		"echo \"Database out of sync with JSONL. Run 'bd sync --import-only' to fix.\" >&2",
		"exit 1",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	output, err := runBDCommand(workDir, scriptPath, nil, true, "list", "--json")
	if err != nil {
		t.Fatalf("runBDCommand returned error: %v", err)
	}
	if got := strings.TrimSpace(string(output)); got != "ok" {
		t.Fatalf("runBDCommand output = %q, want %q", got, "ok")
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 2 {
		t.Fatalf("invocation count = %d, want 2", len(calls))
	}
	if strings.Contains(calls[0], "--allow-stale") {
		t.Fatalf("first call unexpectedly had --allow-stale: %q", calls[0])
	}
	if !strings.Contains(calls[1], "--allow-stale") {
		t.Fatalf("second call missing --allow-stale: %q", calls[1])
	}
}

func TestRunBDCommand_DoesNotRetryAllowStaleWhenImportNotRecent(t *testing.T) {
	workDir := t.TempDir()
	writeLastImportTime(t, workDir, time.Now().Add(-2*time.Minute))

	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\n' \"$*\" >> \"" + invocations + "\"",
		"echo \"Database out of sync with JSONL. Run 'bd sync --import-only' to fix.\" >&2",
		"exit 1",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	_, err := runBDCommand(workDir, scriptPath, nil, true, "list", "--json")
	if err == nil {
		t.Fatal("runBDCommand returned nil error, want failure")
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 1 {
		t.Fatalf("invocation count = %d, want 1", len(calls))
	}
	if strings.Contains(calls[0], "--allow-stale") {
		t.Fatalf("call unexpectedly had --allow-stale: %q", calls[0])
	}
}

func TestRunBDCommand_RetriesAllowStaleWhenOutOfSyncJSONErrorPayload(t *testing.T) {
	workDir := t.TempDir()
	writeLastImportTime(t, workDir, time.Now().Add(-5*time.Second))

	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$*\" >> \"" + invocations + "\"",
		"case \" $* \" in",
		"  *\" --allow-stale \"*)",
		"    printf '[{\"id\":\"orch-go-1\"}]'",
		"    exit 0",
		"    ;;",
		"esac",
		"printf '{\"error\":\"Database out of sync with JSONL.\"}'",
		"exit 0",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	output, err := runBDCommand(workDir, scriptPath, nil, false, "show", "orch-go-1", "--json")
	if err != nil {
		t.Fatalf("runBDCommand returned error: %v", err)
	}
	if got := strings.TrimSpace(string(output)); got != `[{"id":"orch-go-1"}]` {
		t.Fatalf("runBDCommand output = %q, want %q", got, `[{"id":"orch-go-1"}]`)
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 2 {
		t.Fatalf("invocation count = %d, want 2", len(calls))
	}
	if strings.Contains(calls[0], "--allow-stale") {
		t.Fatalf("first call unexpectedly had --allow-stale: %q", calls[0])
	}
	if !strings.Contains(calls[1], "--allow-stale") {
		t.Fatalf("second call missing --allow-stale: %q", calls[1])
	}
}

func TestRunBDCommand_AddsQuietByDefault(t *testing.T) {
	workDir := t.TempDir()
	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$*\" >> \"" + invocations + "\"",
		"case \" $* \" in",
		"  *\" --quiet \"*)",
		"    printf 'ok'",
		"    exit 0",
		"    ;;",
		"esac",
		"echo \"WARNING: JSONL file hash mismatch detected. Clearing export_hashes to force full re-export.\" >&2",
		"printf 'ok'",
		"exit 0",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	t.Setenv("ORCH_DEBUG", "")
	output, err := runBDCommand(workDir, scriptPath, nil, true, "update", "orch-go-1", "--status", "in_progress")
	if err != nil {
		t.Fatalf("runBDCommand returned error: %v", err)
	}
	if got := strings.TrimSpace(string(output)); got != "ok" {
		t.Fatalf("runBDCommand output = %q, want %q", got, "ok")
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 1 {
		t.Fatalf("invocation count = %d, want 1", len(calls))
	}
	if !strings.Contains(calls[0], "--quiet") {
		t.Fatalf("expected --quiet in command invocation, got %q", calls[0])
	}
}

func TestRunBDCommand_DebugModeSkipsQuiet(t *testing.T) {
	workDir := t.TempDir()
	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$*\" >> \"" + invocations + "\"",
		"echo \"WARNING: JSONL file hash mismatch detected. Clearing export_hashes to force full re-export.\" >&2",
		"printf 'ok'",
		"exit 0",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	t.Setenv("ORCH_DEBUG", "1")
	output, err := runBDCommand(workDir, scriptPath, nil, true, "update", "orch-go-1", "--status", "in_progress")
	if err != nil {
		t.Fatalf("runBDCommand returned error: %v", err)
	}
	if !strings.Contains(string(output), "WARNING: JSONL file hash mismatch detected") {
		t.Fatalf("expected warning output in debug mode, got %q", string(output))
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 1 {
		t.Fatalf("invocation count = %d, want 1", len(calls))
	}
	if strings.Contains(calls[0], "--quiet") {
		t.Fatalf("did not expect --quiet in debug mode invocation, got %q", calls[0])
	}
}

func TestRunBDCommand_RetriesAllowStaleWhenJSONLRecentlyUpdated(t *testing.T) {
	workDir := t.TempDir()
	writeIssuesJSONL(t, workDir, time.Now().Add(-3*time.Second))

	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\n' \"$*\" >> \"" + invocations + "\"",
		"case \" $* \" in",
		"  *\" --allow-stale \"*)",
		"    printf 'ok'",
		"    exit 0",
		"    ;;",
		"esac",
		"echo \"Database out of sync with JSONL. Run 'bd sync --import-only' to fix.\" >&2",
		"exit 1",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	output, err := runBDCommand(workDir, scriptPath, nil, true, "list", "--json")
	if err != nil {
		t.Fatalf("runBDCommand returned error: %v", err)
	}
	if got := strings.TrimSpace(string(output)); got != "ok" {
		t.Fatalf("runBDCommand output = %q, want %q", got, "ok")
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 2 {
		t.Fatalf("invocation count = %d, want 2", len(calls))
	}
	if strings.Contains(calls[0], "--allow-stale") {
		t.Fatalf("first call unexpectedly had --allow-stale: %q", calls[0])
	}
	if !strings.Contains(calls[1], "--allow-stale") {
		t.Fatalf("second call missing --allow-stale: %q", calls[1])
	}
}

func TestRunBDCommand_DoesNotRetryAllowStaleWhenJSONLNotRecent(t *testing.T) {
	workDir := t.TempDir()
	writeIssuesJSONL(t, workDir, time.Now().Add(-2*time.Minute))

	invocations := filepath.Join(workDir, "invocations.log")
	scriptPath := filepath.Join(workDir, "fake-bd.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\n' \"$*\" >> \"" + invocations + "\"",
		"echo \"Database out of sync with JSONL. Run 'bd sync --import-only' to fix.\" >&2",
		"exit 1",
	}, "\n") + "\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bd script: %v", err)
	}

	_, err := runBDCommand(workDir, scriptPath, nil, true, "list", "--json")
	if err == nil {
		t.Fatal("runBDCommand returned nil error, want failure")
	}

	calls := readInvocationLines(t, invocations)
	if len(calls) != 1 {
		t.Fatalf("invocation count = %d, want 1", len(calls))
	}
	if strings.Contains(calls[0], "--allow-stale") {
		t.Fatalf("call unexpectedly had --allow-stale: %q", calls[0])
	}
}

func writeLastImportTime(t *testing.T, workDir string, ts time.Time) {
	t.Helper()

	beadsDir := filepath.Join(workDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir .beads: %v", err)
	}

	dbPath := filepath.Join(beadsDir, "beads.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatalf("create metadata table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO metadata(key, value) VALUES('last_import_time', ?)`, ts.Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert last_import_time: %v", err)
	}
}

func writeIssuesJSONL(t *testing.T, workDir string, ts time.Time) {
	t.Helper()

	beadsDir := filepath.Join(workDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir .beads: %v", err)
	}

	jsonlPath := filepath.Join(beadsDir, "issues.jsonl")
	if err := os.WriteFile(jsonlPath, []byte("\n"), 0o644); err != nil {
		t.Fatalf("write issues.jsonl: %v", err)
	}
	if err := os.Chtimes(jsonlPath, ts, ts); err != nil {
		t.Fatalf("chtimes issues.jsonl: %v", err)
	}
}

func readInvocationLines(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read invocations log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

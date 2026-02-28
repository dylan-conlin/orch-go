package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTrace_FileNotFound(t *testing.T) {
	_, err := ReadTrace("/nonexistent/trace.jsonl", TraceOptions{})
	if err == nil {
		t.Error("expected error for missing trace file")
	}
	if !containsStr(err.Error(), "HOOK_TRACE=1") {
		t.Errorf("expected hint about enabling tracing, got: %v", err)
	}
}

func TestReadTrace_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadTrace(path, TraceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadTrace_ParseEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	content := `{"ts": 1709157600, "hook": "gate-bd-close.py", "event": "PreToolUse", "tool": "Bash", "decision": "ALLOW", "duration_ms": 12.5, "context": "worker", "session": "abc123"}
{"ts": 1709157601, "hook": "task-gate.py", "event": "PreToolUse", "tool": "Task", "decision": "DENY", "duration_ms": 5.2, "context": "orchestrator", "session": "def456"}
{"ts": 1709157602, "hook": "session-start.sh", "event": "SessionStart", "tool": "", "decision": "ALLOW", "duration_ms": 100.0, "context": "", "session": "abc123"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadTrace(path, TraceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Hook != "gate-bd-close.py" {
		t.Errorf("expected first hook 'gate-bd-close.py', got '%s'", entries[0].Hook)
	}
}

func TestReadTrace_FilterBySession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	content := `{"ts": 1, "hook": "h1", "event": "PreToolUse", "session": "abc"}
{"ts": 2, "hook": "h2", "event": "PreToolUse", "session": "def"}
{"ts": 3, "hook": "h3", "event": "SessionStart", "session": "abc"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadTrace(path, TraceOptions{SessionFilter: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for session 'abc', got %d", len(entries))
	}
}

func TestReadTrace_FilterByHook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	content := `{"ts": 1, "hook": "gate-bd-close.py", "event": "PreToolUse", "session": "abc"}
{"ts": 2, "hook": "task-gate.py", "event": "PreToolUse", "session": "def"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadTrace(path, TraceOptions{HookFilter: "gate-bd"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry matching 'gate-bd', got %d", len(entries))
	}
}

func TestReadTrace_Limit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")
	content := `{"ts": 1, "hook": "h1", "event": "E1", "session": "a"}
{"ts": 2, "hook": "h2", "event": "E2", "session": "a"}
{"ts": 3, "hook": "h3", "event": "E3", "session": "a"}
{"ts": 4, "hook": "h4", "event": "E4", "session": "a"}
{"ts": 5, "hook": "h5", "event": "E5", "session": "a"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadTrace(path, TraceOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(entries))
	}
	// Should return the LAST 2 entries
	if entries[0].Hook != "h4" {
		t.Errorf("expected h4, got %s", entries[0].Hook)
	}
	if entries[1].Hook != "h5" {
		t.Errorf("expected h5, got %s", entries[1].Hook)
	}
}

func TestFormatTraceEntries_Empty(t *testing.T) {
	result := FormatTraceEntries(nil)
	if result != "No trace entries found" {
		t.Errorf("expected 'No trace entries found', got '%s'", result)
	}
}

func TestFormatTraceEntries_WithEntries(t *testing.T) {
	entries := []TraceEntry{
		{Timestamp: 1709157600, Hook: "gate.py", Event: "PreToolUse", Tool: "Bash", Decision: "ALLOW", DurationMs: 12.5},
	}
	result := FormatTraceEntries(entries)
	if !containsStr(result, "gate.py") {
		t.Errorf("expected formatted output with hook name, got: %s", result)
	}
	if !containsStr(result, "ALLOW") {
		t.Errorf("expected ALLOW in output, got: %s", result)
	}
}

package hook

import (
	"fmt"
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

func TestWriteTrace_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "trace.jsonl")

	entry := TraceEntry{
		Timestamp:  1709157600,
		Hook:       "test-hook.py",
		Event:      "PreToolUse",
		Tool:       "Bash",
		Decision:   "ALLOW",
		DurationMs: 12.5,
		Session:    "test-session",
	}

	err := WriteTrace(path, entry)
	if err != nil {
		t.Fatalf("WriteTrace failed: %v", err)
	}

	// Read it back
	entries, err := ReadTrace(path, TraceOptions{})
	if err != nil {
		t.Fatalf("ReadTrace failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Hook != "test-hook.py" {
		t.Errorf("expected hook 'test-hook.py', got '%s'", entries[0].Hook)
	}
	if entries[0].DurationMs != 12.5 {
		t.Errorf("expected duration 12.5ms, got %f", entries[0].DurationMs)
	}
}

func TestWriteTrace_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")

	// Write two entries
	err := WriteTrace(path, TraceEntry{Hook: "first", Event: "PreToolUse"})
	if err != nil {
		t.Fatal(err)
	}
	err = WriteTrace(path, TraceEntry{Hook: "second", Event: "PostToolUse"})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := ReadTrace(path, TraceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Hook != "first" {
		t.Errorf("expected 'first', got '%s'", entries[0].Hook)
	}
	if entries[1].Hook != "second" {
		t.Errorf("expected 'second', got '%s'", entries[1].Hook)
	}
}

func TestTraceEntryFromResult(t *testing.T) {
	result := &RunResult{
		Hook: ResolvedHook{
			Event:   "PreToolUse",
			Command: "$HOME/.claude/hooks/gate-git-add-all.py",
			Matcher: "Bash",
		},
		Duration: 45300000, // 45.3ms in nanoseconds
		Validation: &ValidationResult{
			Decision: DecisionAllow,
		},
	}

	entry := TraceEntryFromResult(result, "test-session")
	if entry.Hook != "gate-git-add-all.py" {
		t.Errorf("expected hook basename 'gate-git-add-all.py', got '%s'", entry.Hook)
	}
	if entry.Event != "PreToolUse" {
		t.Errorf("expected event 'PreToolUse', got '%s'", entry.Event)
	}
	if entry.Decision != "ALLOW" {
		t.Errorf("expected decision 'ALLOW', got '%s'", entry.Decision)
	}
	if entry.Session != "test-session" {
		t.Errorf("expected session 'test-session', got '%s'", entry.Session)
	}
	if entry.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
}

func TestTraceEntryFromResult_WithError(t *testing.T) {
	result := &RunResult{
		Hook: ResolvedHook{
			Event:   "PreToolUse",
			Command: "test-hook.sh",
		},
		Duration: 100000000,
		Error:    fmt.Errorf("hook timed out after 5s"),
	}

	entry := TraceEntryFromResult(result, "sess")
	if entry.Decision != "ERROR" {
		t.Errorf("expected decision 'ERROR', got '%s'", entry.Decision)
	}
	if entry.OutputPreview != "hook timed out after 5s" {
		t.Errorf("expected error in preview, got '%s'", entry.OutputPreview)
	}
}

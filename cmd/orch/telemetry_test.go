package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestDetectCallerContext_Human(t *testing.T) {
	// Save and clear env
	origCtx := os.Getenv("CLAUDE_CONTEXT")
	origSpawned := os.Getenv("ORCH_SPAWNED")
	os.Unsetenv("CLAUDE_CONTEXT")
	os.Unsetenv("ORCH_SPAWNED")
	defer func() {
		os.Setenv("CLAUDE_CONTEXT", origCtx)
		os.Setenv("ORCH_SPAWNED", origSpawned)
	}()

	got := detectCallerContext()
	if got != "human" {
		t.Errorf("detectCallerContext() = %q, want %q", got, "human")
	}
}

func TestDetectCallerContext_Worker(t *testing.T) {
	origCtx := os.Getenv("CLAUDE_CONTEXT")
	origSpawned := os.Getenv("ORCH_SPAWNED")
	defer func() {
		os.Setenv("CLAUDE_CONTEXT", origCtx)
		os.Setenv("ORCH_SPAWNED", origSpawned)
	}()

	os.Setenv("CLAUDE_CONTEXT", "worker")
	os.Unsetenv("ORCH_SPAWNED")

	got := detectCallerContext()
	if got != "worker" {
		t.Errorf("detectCallerContext() = %q, want %q", got, "worker")
	}
}

func TestDetectCallerContext_WorkerFromSpawned(t *testing.T) {
	origCtx := os.Getenv("CLAUDE_CONTEXT")
	origSpawned := os.Getenv("ORCH_SPAWNED")
	defer func() {
		os.Setenv("CLAUDE_CONTEXT", origCtx)
		os.Setenv("ORCH_SPAWNED", origSpawned)
	}()

	os.Unsetenv("CLAUDE_CONTEXT")
	os.Setenv("ORCH_SPAWNED", "1")

	got := detectCallerContext()
	if got != "worker" {
		t.Errorf("detectCallerContext() = %q, want %q", got, "worker")
	}
}

func TestDetectCallerContext_Orchestrator(t *testing.T) {
	origCtx := os.Getenv("CLAUDE_CONTEXT")
	origSpawned := os.Getenv("ORCH_SPAWNED")
	defer func() {
		os.Setenv("CLAUDE_CONTEXT", origCtx)
		os.Setenv("ORCH_SPAWNED", origSpawned)
	}()

	os.Setenv("CLAUDE_CONTEXT", "orchestrator")
	os.Unsetenv("ORCH_SPAWNED")

	got := detectCallerContext()
	if got != "orchestrator" {
		t.Errorf("detectCallerContext() = %q, want %q", got, "orchestrator")
	}
}

func TestDetectCallerContext_MetaOrchestrator(t *testing.T) {
	origCtx := os.Getenv("CLAUDE_CONTEXT")
	origSpawned := os.Getenv("ORCH_SPAWNED")
	defer func() {
		os.Setenv("CLAUDE_CONTEXT", origCtx)
		os.Setenv("ORCH_SPAWNED", origSpawned)
	}()

	os.Setenv("CLAUDE_CONTEXT", "meta-orchestrator")
	os.Unsetenv("ORCH_SPAWNED")

	got := detectCallerContext()
	if got != "orchestrator" {
		t.Errorf("detectCallerContext() = %q, want %q", got, "orchestrator")
	}
}

func TestEmitCommandInvoked_WritesEvent(t *testing.T) {
	// Set up temp events file
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	origPath := os.Getenv("ORCH_EVENTS_PATH")
	os.Setenv("ORCH_EVENTS_PATH", eventsPath)
	defer os.Setenv("ORCH_EVENTS_PATH", origPath)

	// Clear caller context env
	origCtx := os.Getenv("CLAUDE_CONTEXT")
	origSpawned := os.Getenv("ORCH_SPAWNED")
	os.Unsetenv("CLAUDE_CONTEXT")
	os.Unsetenv("ORCH_SPAWNED")
	defer func() {
		os.Setenv("CLAUDE_CONTEXT", origCtx)
		os.Setenv("ORCH_SPAWNED", origSpawned)
	}()

	emitCommandInvoked("harness audit", "--days=7", "--json=true")

	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("Failed to read events file: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, "command.invoked") {
		t.Error("Expected event type 'command.invoked'")
	}
	if !strings.Contains(raw, "harness audit") {
		t.Error("Expected command name 'harness audit'")
	}
	if !strings.Contains(raw, "human") {
		t.Error("Expected caller 'human'")
	}
	if !strings.Contains(raw, "--days=7") {
		t.Error("Expected flags to include '--days=7'")
	}

	// Parse and verify structure
	var event events.Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &event); err != nil {
		t.Fatalf("Failed to parse event JSON: %v", err)
	}
	if event.Type != events.EventTypeCommandInvoked {
		t.Errorf("event.Type = %q, want %q", event.Type, events.EventTypeCommandInvoked)
	}
	if event.Data["command"] != "harness audit" {
		t.Errorf("data.command = %v, want %q", event.Data["command"], "harness audit")
	}
	if event.Data["caller"] != "human" {
		t.Errorf("data.caller = %v, want %q", event.Data["caller"], "human")
	}
}

func TestEmitCommandInvoked_NoFlags(t *testing.T) {
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	origPath := os.Getenv("ORCH_EVENTS_PATH")
	os.Setenv("ORCH_EVENTS_PATH", eventsPath)
	defer os.Setenv("ORCH_EVENTS_PATH", origPath)

	origCtx := os.Getenv("CLAUDE_CONTEXT")
	os.Setenv("CLAUDE_CONTEXT", "worker")
	defer os.Setenv("CLAUDE_CONTEXT", origCtx)

	emitCommandInvoked("stats")

	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("Failed to read events file: %v", err)
	}

	var event events.Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &event); err != nil {
		t.Fatalf("Failed to parse event JSON: %v", err)
	}

	if event.Data["caller"] != "worker" {
		t.Errorf("data.caller = %v, want %q", event.Data["caller"], "worker")
	}
	// flags should be omitted when empty (no flags passed)
	if _, ok := event.Data["flags"]; ok {
		t.Error("Expected flags to be omitted when empty")
	}
}

package process

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLedgerRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "process-ledger.jsonl")
	ledger := NewLedger(path)

	entry := LedgerEntry{
		Workspace: "og-feat-auth-08feb",
		BeadsID:   "orch-go-abc1",
		SessionID: "sess-123",
		SpawnPID:  1000,
		ChildPID:  2000,
		PGID:      2000,
		StartedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := ledger.Record(entry); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify JSONL written
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read ledger: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var decoded LedgerEntry
	if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.Workspace != "og-feat-auth-08feb" {
		t.Errorf("workspace = %q, want %q", decoded.Workspace, "og-feat-auth-08feb")
	}
	if decoded.ChildPID != 2000 {
		t.Errorf("child_pid = %d, want 2000", decoded.ChildPID)
	}
}

func TestLedgerRemoveByWorkspace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "process-ledger.jsonl")
	ledger := NewLedger(path)

	now := time.Now()
	entries := []LedgerEntry{
		{Workspace: "ws-a", BeadsID: "id-a", ChildPID: 100, StartedAt: now, LastSeen: now},
		{Workspace: "ws-b", BeadsID: "id-b", ChildPID: 200, StartedAt: now, LastSeen: now},
		{Workspace: "ws-c", BeadsID: "id-c", ChildPID: 300, StartedAt: now, LastSeen: now},
	}
	for _, e := range entries {
		if err := ledger.Record(e); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	if err := ledger.RemoveByWorkspace("ws-b"); err != nil {
		t.Fatalf("RemoveByWorkspace failed: %v", err)
	}

	remaining, err := ledger.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(remaining))
	}
	for _, e := range remaining {
		if e.Workspace == "ws-b" {
			t.Error("ws-b should have been removed")
		}
	}
}

func TestLedgerRemoveByBeadsID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "process-ledger.jsonl")
	ledger := NewLedger(path)

	now := time.Now()
	entries := []LedgerEntry{
		{Workspace: "ws-a", BeadsID: "id-a", ChildPID: 100, StartedAt: now, LastSeen: now},
		{Workspace: "ws-b", BeadsID: "id-b", ChildPID: 200, StartedAt: now, LastSeen: now},
	}
	for _, e := range entries {
		if err := ledger.Record(e); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	if err := ledger.RemoveByBeadsID("id-a"); err != nil {
		t.Fatalf("RemoveByBeadsID failed: %v", err)
	}

	remaining, err := ledger.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(remaining))
	}
	if remaining[0].BeadsID != "id-b" {
		t.Errorf("expected id-b, got %s", remaining[0].BeadsID)
	}
}

func TestReconcile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "process-ledger.jsonl")
	ledger := NewLedger(path)

	now := time.Now()
	// Use PID 1 (init/launchd) which always exists, and PID 999999999 which doesn't
	entries := []LedgerEntry{
		{Workspace: "ws-alive", BeadsID: "id-alive", ChildPID: 1, StartedAt: now, LastSeen: now},
		{Workspace: "ws-dead", BeadsID: "id-dead", ChildPID: 999999999, StartedAt: now, LastSeen: now},
	}
	for _, e := range entries {
		if err := ledger.Record(e); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	stale, err := ledger.Reconcile()
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// PID 999999999 should be stale (doesn't exist)
	found := false
	for _, s := range stale {
		if s.Workspace == "ws-dead" {
			found = true
		}
	}
	if !found {
		t.Error("expected ws-dead to be stale (PID 999999999 should not exist)")
	}
}

func TestLedgerReadAllMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.jsonl")
	ledger := NewLedger(path)

	entries, err := ledger.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll on missing file should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestSweepRemovesDeadEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "process-ledger.jsonl")
	ledger := NewLedger(path)

	now := time.Now()
	// PID 1 (init/launchd) always exists; PID 999999999 does not
	entries := []LedgerEntry{
		{Workspace: "ws-alive", BeadsID: "id-alive", ChildPID: 1, StartedAt: now, LastSeen: now},
		{Workspace: "ws-dead", BeadsID: "id-dead", ChildPID: 999999999, StartedAt: now, LastSeen: now},
	}
	for _, e := range entries {
		if err := ledger.Record(e); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	result := ledger.Sweep()
	if result.TotalEntries != 2 {
		t.Errorf("TotalEntries = %d, want 2", result.TotalEntries)
	}
	if result.StaleRemoved != 1 {
		t.Errorf("StaleRemoved = %d, want 1", result.StaleRemoved)
	}

	// Verify dead entry was removed from the ledger
	remaining, err := ledger.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining entry, got %d", len(remaining))
	}
	if remaining[0].BeadsID != "id-alive" {
		t.Errorf("expected id-alive to survive, got %s", remaining[0].BeadsID)
	}
}

func TestSweepEmptyLedger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "process-ledger.jsonl")
	ledger := NewLedger(path)

	result := ledger.Sweep()
	if result.TotalEntries != 0 {
		t.Errorf("TotalEntries = %d, want 0", result.TotalEntries)
	}
	if result.StaleRemoved != 0 {
		t.Errorf("StaleRemoved = %d, want 0", result.StaleRemoved)
	}
}

func TestSweepMissingLedgerFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.jsonl")
	ledger := NewLedger(path)

	result := ledger.Sweep()
	if result.Error != nil {
		t.Errorf("Sweep on missing file should not error: %v", result.Error)
	}
	if result.TotalEntries != 0 {
		t.Errorf("TotalEntries = %d, want 0", result.TotalEntries)
	}
}

func TestLedgerRecordCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "process-ledger.jsonl")
	ledger := NewLedger(path)

	entry := LedgerEntry{
		Workspace: "ws-test",
		ChildPID:  42,
		StartedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := ledger.Record(entry); err != nil {
		t.Fatalf("Record should create parent directory: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("ledger file should exist after Record")
	}
}

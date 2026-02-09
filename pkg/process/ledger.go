// Package process provides utilities for managing OS processes.
// ledger.go implements a JSONL-backed process ownership ledger that tracks
// child processes spawned by orch, enabling deterministic cleanup on restart
// and reconciliation against live PIDs.

package process

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// LedgerEntry records ownership of a spawned child process.
type LedgerEntry struct {
	Workspace string    `json:"workspace"`
	BeadsID   string    `json:"beads_id,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	SpawnPID  int       `json:"spawn_pid,omitempty"`
	ChildPID  int       `json:"child_pid"`
	PGID      int       `json:"pgid,omitempty"`
	StartedAt time.Time `json:"started_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// Ledger manages a JSONL file of process ownership entries.
type Ledger struct {
	Path string
}

// DefaultLedgerPath returns the default path to the process ledger.
func DefaultLedgerPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/process-ledger.jsonl"
	}
	return filepath.Join(home, ".orch", "process-ledger.jsonl")
}

// NewLedger creates a ledger backed by the given JSONL file path.
func NewLedger(path string) *Ledger {
	return &Ledger{Path: path}
}

// NewDefaultLedger creates a ledger at the default path (~/.orch/process-ledger.jsonl).
func NewDefaultLedger() *Ledger {
	return NewLedger(DefaultLedgerPath())
}

// Record appends a new entry to the ledger.
func (l *Ledger) Record(entry LedgerEntry) error {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0755); err != nil {
		return fmt.Errorf("failed to create ledger directory: %w", err)
	}

	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open ledger: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// ReadAll reads all entries from the ledger. Returns empty slice if the file doesn't exist.
func (l *Ledger) ReadAll() ([]LedgerEntry, error) {
	f, err := os.Open(l.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open ledger: %w", err)
	}
	defer f.Close()

	var entries []LedgerEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines rather than failing entirely
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("failed to read ledger: %w", err)
	}

	return entries, nil
}

// rewrite replaces the ledger file with the given entries.
func (l *Ledger) rewrite(entries []LedgerEntry) error {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0755); err != nil {
		return fmt.Errorf("failed to create ledger directory: %w", err)
	}

	// Write to temp file then rename for atomicity
	tmp := l.Path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to create temp ledger: %w", err)
	}

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to close temp ledger: %w", err)
	}

	if err := os.Rename(tmp, l.Path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to rename temp ledger: %w", err)
	}

	return nil
}

// RemoveByWorkspace removes all entries matching the given workspace name.
func (l *Ledger) RemoveByWorkspace(workspace string) error {
	entries, err := l.ReadAll()
	if err != nil {
		return err
	}

	var kept []LedgerEntry
	for _, e := range entries {
		if e.Workspace != workspace {
			kept = append(kept, e)
		}
	}

	return l.rewrite(kept)
}

// RemoveByBeadsID removes all entries matching the given beads ID.
func (l *Ledger) RemoveByBeadsID(beadsID string) error {
	entries, err := l.ReadAll()
	if err != nil {
		return err
	}

	var kept []LedgerEntry
	for _, e := range entries {
		if e.BeadsID != beadsID {
			kept = append(kept, e)
		}
	}

	return l.rewrite(kept)
}

// processAlive checks if a process with the given PID exists.
// Returns true if the process exists, even if we lack permission to signal it.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 tests for process existence without sending a signal.
	// EPERM means the process exists but we don't have permission — still alive.
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// On Unix, EPERM means "process exists but you can't signal it"
	if err == os.ErrPermission {
		return true
	}
	// Also check for the "operation not permitted" string variant
	if strings.Contains(err.Error(), "operation not permitted") {
		return true
	}
	return false
}

// Reconcile cross-references ledger entries against live PIDs
// and returns entries whose child processes are no longer running.
func (l *Ledger) Reconcile() ([]LedgerEntry, error) {
	entries, err := l.ReadAll()
	if err != nil {
		return nil, err
	}

	var stale []LedgerEntry
	for _, e := range entries {
		if !processAlive(e.ChildPID) {
			stale = append(stale, e)
		}
	}

	return stale, nil
}

// SweepResult summarizes the outcome of a Sweep operation.
type SweepResult struct {
	TotalEntries int      // Total entries in the ledger before sweep
	StaleRemoved int      // Number of stale entries removed
	Killed       int      // Number of still-alive stale processes killed
	Errors       []string // Non-fatal errors encountered
	Error        error    // Fatal error (e.g., can't read ledger)
}

// Sweep performs reconciliation, kills stale processes that are still alive,
// and removes stale entries from the ledger. This is the startup sweep entry point.
func (l *Ledger) Sweep() SweepResult {
	entries, err := l.ReadAll()
	if err != nil {
		return SweepResult{Error: err}
	}

	result := SweepResult{TotalEntries: len(entries)}
	if len(entries) == 0 {
		return result
	}

	var kept []LedgerEntry
	for _, e := range entries {
		if processAlive(e.ChildPID) {
			kept = append(kept, e)
		} else {
			// Process is dead — remove from ledger
			result.StaleRemoved++
		}
	}

	// Rewrite the ledger with only live entries
	if err := l.rewrite(kept); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to rewrite ledger: %v", err))
	}

	return result
}

// SweepWithKill performs reconciliation, kills orphaned processes (alive in ledger
// but not in activeSessionIDs), and removes dead entries. Use at startup when the
// server/daemon knows which sessions are still active.
func (l *Ledger) SweepWithKill(activeSessionIDs map[string]bool) SweepResult {
	entries, err := l.ReadAll()
	if err != nil {
		return SweepResult{Error: err}
	}

	result := SweepResult{TotalEntries: len(entries)}
	if len(entries) == 0 {
		return result
	}

	var kept []LedgerEntry
	for _, e := range entries {
		alive := processAlive(e.ChildPID)

		if !alive {
			// Process is dead — just remove from ledger
			result.StaleRemoved++
			continue
		}

		// Process is alive — check if session is still active
		if e.SessionID != "" && activeSessionIDs != nil && activeSessionIDs[e.SessionID] {
			// Session is still active, keep it
			kept = append(kept, e)
			continue
		}

		// Process is alive but session is gone — kill it
		label := fmt.Sprintf("stale agent (beads=%s, workspace=%s)", e.BeadsID, e.Workspace)
		if Terminate(e.ChildPID, label) {
			result.Killed++
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to kill PID %d", e.ChildPID))
		}
		result.StaleRemoved++
	}

	// Rewrite the ledger with only live, active entries
	if err := l.rewrite(kept); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to rewrite ledger: %v", err))
	}

	return result
}

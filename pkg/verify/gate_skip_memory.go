// Package verify provides gate skip memory for session-level gate bypasses.
//
// Generalizes the build-specific build-skip.json into gate-skips.json that
// supports session-level skip reasons for any gate. Entries have a 2-hour TTL
// and auto-expire on read.
//
// See .kb/decisions/2026-02-06-completion-pipeline-parallel-redesign.md sections 3 and 5.
package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GateSkip represents a persisted gate skip decision for a single gate.
type GateSkip struct {
	Gate      string    `json:"gate"`       // Gate constant name (e.g., "build", "dashboard_health")
	Reason    string    `json:"reason"`     // Why the gate was skipped
	SetAt     time.Time `json:"set_at"`     // When it was set
	SetBy     string    `json:"set_by"`     // Who set it (beads ID or "orchestrator")
	ExpiresAt time.Time `json:"expires_at"` // When this skip expires (auto-cleanup)
}

// GateSkipFile represents the on-disk format for gate-skips.json.
type GateSkipFile struct {
	Skips []GateSkip `json:"skips"`
}

const (
	// GateSkipFilename is the name of the gate skip memory file.
	GateSkipFilename = "gate-skips.json"

	// GateSkipDuration is how long a gate skip decision lasts.
	// After this duration, the skip expires and the gate resumes.
	GateSkipDuration = 2 * time.Hour
)

// gateSkipPath returns the path to the gate skip memory file.
func gateSkipPath(projectDir string) string {
	return filepath.Join(projectDir, ".orch", GateSkipFilename)
}

// readGateSkipFile reads and returns the gate skip file, pruning expired entries.
// Returns an empty file struct if the file doesn't exist or is invalid.
func readGateSkipFile(projectDir string) GateSkipFile {
	path := gateSkipPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return GateSkipFile{}
	}

	var file GateSkipFile
	if err := json.Unmarshal(data, &file); err != nil {
		return GateSkipFile{}
	}

	// Prune expired entries
	now := time.Now()
	original := len(file.Skips)
	var active []GateSkip
	for _, skip := range file.Skips {
		if now.Before(skip.ExpiresAt) {
			active = append(active, skip)
		}
	}
	file.Skips = active

	// Write back pruned file if entries were removed
	if len(active) != original {
		writeGateSkipFile(projectDir, file)
	}

	return file
}

// writeGateSkipFile writes the gate skip file to disk.
func writeGateSkipFile(projectDir string, file GateSkipFile) error {
	if file.Skips == nil {
		file.Skips = []GateSkip{}
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gate skips: %w", err)
	}

	path := gateSkipPath(projectDir)

	// Ensure .orch directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// ReadGateSkipMemory reads the skip for a specific gate.
// Returns nil if no active skip exists for that gate.
func ReadGateSkipMemory(projectDir, gate string) *GateSkip {
	file := readGateSkipFile(projectDir)
	for _, skip := range file.Skips {
		if skip.Gate == gate {
			return &skip
		}
	}
	return nil
}

// WriteGateSkipMemory persists a gate skip decision to disk.
// If a skip already exists for the gate, it is replaced.
func WriteGateSkipMemory(projectDir, gate, reason, setBy string) error {
	file := readGateSkipFile(projectDir)

	// Remove existing entry for this gate
	var filtered []GateSkip
	for _, skip := range file.Skips {
		if skip.Gate != gate {
			filtered = append(filtered, skip)
		}
	}

	// Add new entry
	now := time.Now()
	filtered = append(filtered, GateSkip{
		Gate:      gate,
		Reason:    reason,
		SetAt:     now,
		SetBy:     setBy,
		ExpiresAt: now.Add(GateSkipDuration),
	})

	file.Skips = filtered
	return writeGateSkipFile(projectDir, file)
}

// ClearGateSkipMemory removes the skip for a specific gate.
func ClearGateSkipMemory(projectDir, gate string) error {
	file := readGateSkipFile(projectDir)

	var filtered []GateSkip
	for _, skip := range file.Skips {
		if skip.Gate != gate {
			filtered = append(filtered, skip)
		}
	}

	file.Skips = filtered
	return writeGateSkipFile(projectDir, file)
}

// ClearAllGateSkipMemory removes all gate skip entries.
func ClearAllGateSkipMemory(projectDir string) error {
	path := gateSkipPath(projectDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListGateSkipMemory returns all active (non-expired) gate skips.
func ListGateSkipMemory(projectDir string) []GateSkip {
	file := readGateSkipFile(projectDir)
	return file.Skips
}

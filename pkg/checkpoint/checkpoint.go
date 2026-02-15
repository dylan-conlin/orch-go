// Package checkpoint provides infrastructure for tracking verification checkpoints.
// This is Phase 1 of verifiability-first mechanical enforcement.
//
// The checkpoint file (~/.orch/verification-checkpoints.jsonl) tracks which
// deliverables have been human-verified through comprehension and behavioral gates.
package checkpoint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint represents a verification checkpoint entry.
// Each line in the JSONL file is one checkpoint.
type Checkpoint struct {
	BeadsID       string    `json:"beads_id"`
	Deliverable   string    `json:"deliverable"`
	Gate1Complete bool      `json:"gate1_complete"` // Comprehension gate
	Gate2Complete bool      `json:"gate2_complete"` // Behavioral gate
	Timestamp     time.Time `json:"timestamp"`
	ExplainText   string    `json:"explain_text"`
}

// DefaultCheckpointPath returns the default path to the verification checkpoints file.
func DefaultCheckpointPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir can't be determined
		return ".orch/verification-checkpoints.jsonl"
	}
	return filepath.Join(home, ".orch", "verification-checkpoints.jsonl")
}

// WriteCheckpoint appends a checkpoint entry to the JSONL file.
// Creates the file and parent directory if they don't exist.
func WriteCheckpoint(cp Checkpoint) error {
	path := DefaultCheckpointPath()

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Open file for append (create if doesn't exist)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open checkpoint file: %w", err)
	}
	defer file.Close()

	// Encode checkpoint as JSON
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to encode checkpoint: %w", err)
	}

	// Write JSON line
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	return nil
}

// ReadCheckpoints reads all checkpoint entries from the JSONL file.
// Returns an empty slice if the file doesn't exist.
func ReadCheckpoints() ([]Checkpoint, error) {
	path := DefaultCheckpointPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Checkpoint{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open checkpoint file: %w", err)
	}
	defer file.Close()

	var checkpoints []Checkpoint
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		var cp Checkpoint
		if err := json.Unmarshal([]byte(line), &cp); err != nil {
			return nil, fmt.Errorf("failed to parse checkpoint at line %d: %w", lineNum, err)
		}
		checkpoints = append(checkpoints, cp)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	return checkpoints, nil
}

// HasCheckpoint checks if a checkpoint exists for the given beads ID.
// Returns the checkpoint if found, nil otherwise.
func HasCheckpoint(beadsID string) (*Checkpoint, error) {
	checkpoints, err := ReadCheckpoints()
	if err != nil {
		return nil, err
	}

	// Find most recent checkpoint for this beads ID
	// (there may be multiple if checkpoint was updated)
	var found *Checkpoint
	for i := range checkpoints {
		if checkpoints[i].BeadsID == beadsID {
			found = &checkpoints[i]
		}
	}

	return found, nil
}

// HasGate1Checkpoint checks if gate1 (comprehension) checkpoint exists for the given beads ID.
func HasGate1Checkpoint(beadsID string) (bool, error) {
	cp, err := HasCheckpoint(beadsID)
	if err != nil {
		return false, err
	}
	if cp == nil {
		return false, nil
	}
	return cp.Gate1Complete, nil
}

// HasGate2Checkpoint checks if gate2 (behavioral) checkpoint exists for the given beads ID.
func HasGate2Checkpoint(beadsID string) (bool, error) {
	cp, err := HasCheckpoint(beadsID)
	if err != nil {
		return false, err
	}
	if cp == nil {
		return false, nil
	}
	return cp.Gate2Complete, nil
}

// IsTier1Work determines if the given issue type requires full two-gate verification.
// Tier 1 work = features/bugs/decisions (requires both comprehension and behavioral gates)
// Tier 2 work = investigations/probes (comprehension only)
// Tier 3 work = trivial fixes (acknowledge only)
func IsTier1Work(issueType string) bool {
	switch issueType {
	case "feature", "bug", "decision":
		return true
	default:
		return false
	}
}

// RequiresCheckpoint determines if work of the given issue type requires a verification checkpoint.
// Currently, Tier 1 work (features/bugs/decisions) requires checkpoints.
// Tier 2+ work (investigations, tasks, etc.) does not require checkpoints yet.
func RequiresCheckpoint(issueType string) bool {
	return IsTier1Work(issueType)
}

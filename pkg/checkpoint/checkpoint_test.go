package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteCheckpoint(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override default path for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cp := Checkpoint{
		BeadsID:       "orch-go-test",
		Deliverable:   "completion",
		Gate1Complete: true,
		Gate2Complete: false,
		Timestamp:     time.Now(),
		ExplainText:   "Test explanation",
	}

	if err := WriteCheckpoint(cp); err != nil {
		t.Fatalf("WriteCheckpoint failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, ".orch", "verification-checkpoints.jsonl")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Checkpoint file was not created at %s", expectedPath)
	}
}

func TestReadCheckpoints(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override default path for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Write two checkpoints
	cp1 := Checkpoint{
		BeadsID:       "orch-go-abc",
		Deliverable:   "completion",
		Gate1Complete: true,
		Gate2Complete: false,
		Timestamp:     time.Now(),
		ExplainText:   "First checkpoint",
	}

	cp2 := Checkpoint{
		BeadsID:       "orch-go-xyz",
		Deliverable:   "completion",
		Gate1Complete: true,
		Gate2Complete: true,
		Timestamp:     time.Now(),
		ExplainText:   "Second checkpoint",
	}

	if err := WriteCheckpoint(cp1); err != nil {
		t.Fatalf("WriteCheckpoint cp1 failed: %v", err)
	}

	if err := WriteCheckpoint(cp2); err != nil {
		t.Fatalf("WriteCheckpoint cp2 failed: %v", err)
	}

	// Read checkpoints
	checkpoints, err := ReadCheckpoints()
	if err != nil {
		t.Fatalf("ReadCheckpoints failed: %v", err)
	}

	if len(checkpoints) != 2 {
		t.Fatalf("Expected 2 checkpoints, got %d", len(checkpoints))
	}

	// Verify first checkpoint
	if checkpoints[0].BeadsID != "orch-go-abc" {
		t.Errorf("Expected BeadsID 'orch-go-abc', got '%s'", checkpoints[0].BeadsID)
	}

	if !checkpoints[0].Gate1Complete {
		t.Error("Expected Gate1Complete to be true")
	}

	if checkpoints[0].Gate2Complete {
		t.Error("Expected Gate2Complete to be false")
	}
}

func TestHasCheckpoint(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override default path for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Write checkpoint
	cp := Checkpoint{
		BeadsID:       "orch-go-test",
		Deliverable:   "completion",
		Gate1Complete: true,
		Gate2Complete: false,
		Timestamp:     time.Now(),
		ExplainText:   "Test checkpoint",
	}

	if err := WriteCheckpoint(cp); err != nil {
		t.Fatalf("WriteCheckpoint failed: %v", err)
	}

	// Check if checkpoint exists
	found, err := HasCheckpoint("orch-go-test")
	if err != nil {
		t.Fatalf("HasCheckpoint failed: %v", err)
	}

	if found == nil {
		t.Fatal("Expected checkpoint to be found")
	}

	if found.BeadsID != "orch-go-test" {
		t.Errorf("Expected BeadsID 'orch-go-test', got '%s'", found.BeadsID)
	}

	// Check non-existent checkpoint
	notFound, err := HasCheckpoint("orch-go-nonexistent")
	if err != nil {
		t.Fatalf("HasCheckpoint failed for non-existent: %v", err)
	}

	if notFound != nil {
		t.Error("Expected no checkpoint for non-existent beads ID")
	}
}

func TestHasGate1Checkpoint(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override default path for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Write checkpoint with Gate1 complete
	cp := Checkpoint{
		BeadsID:       "orch-go-test",
		Deliverable:   "completion",
		Gate1Complete: true,
		Gate2Complete: false,
		Timestamp:     time.Now(),
		ExplainText:   "Test checkpoint",
	}

	if err := WriteCheckpoint(cp); err != nil {
		t.Fatalf("WriteCheckpoint failed: %v", err)
	}

	// Check Gate1
	hasGate1, err := HasGate1Checkpoint("orch-go-test")
	if err != nil {
		t.Fatalf("HasGate1Checkpoint failed: %v", err)
	}

	if !hasGate1 {
		t.Error("Expected Gate1 checkpoint to exist")
	}

	// Check non-existent
	hasGate1Missing, err := HasGate1Checkpoint("orch-go-missing")
	if err != nil {
		t.Fatalf("HasGate1Checkpoint failed for missing: %v", err)
	}

	if hasGate1Missing {
		t.Error("Expected no Gate1 checkpoint for missing beads ID")
	}
}

func TestIsTier1Work(t *testing.T) {
	tests := []struct {
		issueType string
		expected  bool
	}{
		{"feature", true},
		{"bug", true},
		{"decision", true},
		{"investigation", false},
		{"task", false},
		{"probe", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			result := IsTier1Work(tt.issueType)
			if result != tt.expected {
				t.Errorf("IsTier1Work(%q) = %v, want %v", tt.issueType, result, tt.expected)
			}
		})
	}
}

func TestRequiresCheckpoint(t *testing.T) {
	tests := []struct {
		issueType string
		expected  bool
	}{
		{"feature", true},
		{"bug", true},
		{"decision", true},
		{"investigation", true}, // Tier 2: requires gate1 (comprehension)
		{"probe", true},         // Tier 2: requires gate1 (comprehension)
		{"task", false},         // Tier 3: no checkpoint
		{"question", false},     // Tier 3: no checkpoint
		{"", false},             // Tier 3: no checkpoint
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			result := RequiresCheckpoint(tt.issueType)
			if result != tt.expected {
				t.Errorf("RequiresCheckpoint(%q) = %v, want %v", tt.issueType, result, tt.expected)
			}
		})
	}
}

func TestTierForIssueType(t *testing.T) {
	tests := []struct {
		issueType string
		expected  int
	}{
		{"feature", 1},
		{"bug", 1},
		{"decision", 1},
		{"investigation", 2},
		{"probe", 2},
		{"task", 3},
		{"question", 3},
		{"", 3},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			result := TierForIssueType(tt.issueType)
			if result != tt.expected {
				t.Errorf("TierForIssueType(%q) = %v, want %v", tt.issueType, result, tt.expected)
			}
		})
	}
}

func TestIsTier2Work(t *testing.T) {
	tests := []struct {
		issueType string
		expected  bool
	}{
		{"investigation", true},
		{"probe", true},
		{"feature", false},
		{"task", false},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			result := IsTier2Work(tt.issueType)
			if result != tt.expected {
				t.Errorf("IsTier2Work(%q) = %v, want %v", tt.issueType, result, tt.expected)
			}
		})
	}
}

func TestRequiresGate2(t *testing.T) {
	tests := []struct {
		issueType string
		expected  bool
	}{
		{"feature", true},
		{"bug", true},
		{"decision", true},
		{"investigation", false},
		{"probe", false},
		{"task", false},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			result := RequiresGate2(tt.issueType)
			if result != tt.expected {
				t.Errorf("RequiresGate2(%q) = %v, want %v", tt.issueType, result, tt.expected)
			}
		})
	}
}

func TestReadCheckpoints_EmptyFile(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override default path for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Read checkpoints when file doesn't exist
	checkpoints, err := ReadCheckpoints()
	if err != nil {
		t.Fatalf("ReadCheckpoints failed: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints, got %d", len(checkpoints))
	}
}

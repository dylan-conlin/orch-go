// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"strings"
	"testing"
)

func TestInferTargetFilesFromIssue(t *testing.T) {
	tests := []struct {
		name     string
		issue    *Issue
		expected []string
	}{
		{
			name: "explicit file path in title",
			issue: &Issue{
				Title:       "Fix bug in pkg/daemon/daemon.go",
				Description: "",
			},
			expected: []string{"pkg/daemon/daemon.go"},
		},
		{
			name: "multiple file paths",
			issue: &Issue{
				Title:       "Refactor spawn logic",
				Description: "Need to extract code from cmd/orch/spawn_cmd.go and pkg/spawn/spawn.go",
			},
			expected: []string{"cmd/orch/spawn_cmd.go", "pkg/spawn/spawn.go"},
		},
		{
			name: "file mention without full path",
			issue: &Issue{
				Title:       "Update spawn_cmd.go to handle new feature",
				Description: "",
			},
			expected: []string{"spawn_cmd.go"},
		},
		{
			name: "no file mentions",
			issue: &Issue{
				Title:       "Add new feature",
				Description: "Implement new functionality",
			},
			expected: nil,
		},
		{
			name:     "nil issue",
			issue:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferTargetFilesFromIssue(tt.issue)

			if len(result) != len(tt.expected) {
				t.Errorf("InferTargetFilesFromIssue() returned %d files, expected %d\nGot: %v\nExpected: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			// Convert to maps for order-independent comparison
			resultMap := make(map[string]bool)
			for _, f := range result {
				resultMap[f] = true
			}

			for _, exp := range tt.expected {
				if !resultMap[exp] {
					t.Errorf("InferTargetFilesFromIssue() missing expected file: %s\nGot: %v", exp, result)
				}
			}
		})
	}
}

func TestFindCriticalHotspot(t *testing.T) {
	tests := []struct {
		name           string
		inferredFiles  []string
		hotspots       []HotspotWarning
		expectCritical bool
	}{
		{
			name:          "critical bloat hotspot matches",
			inferredFiles: []string{"cmd/orch/spawn_cmd.go"},
			hotspots: []HotspotWarning{
				{
					Path:  "cmd/orch/spawn_cmd.go",
					Type:  "bloat-size",
					Score: 2000, // >1500 = CRITICAL
				},
			},
			expectCritical: true,
		},
		{
			name:          "bloat hotspot not critical",
			inferredFiles: []string{"pkg/daemon/daemon.go"},
			hotspots: []HotspotWarning{
				{
					Path:  "pkg/daemon/daemon.go",
					Type:  "bloat-size",
					Score: 1200, // <=1500 = not critical
				},
			},
			expectCritical: false,
		},
		{
			name:          "fix-density hotspot ignored",
			inferredFiles: []string{"pkg/spawn/spawn.go"},
			hotspots: []HotspotWarning{
				{
					Path:  "pkg/spawn/spawn.go",
					Type:  "fix-density", // Not bloat-size
					Score: 2000,
				},
			},
			expectCritical: false,
		},
		{
			name:          "partial filename match",
			inferredFiles: []string{"spawn_cmd.go"}, // Without full path
			hotspots: []HotspotWarning{
				{
					Path:  "cmd/orch/spawn_cmd.go",
					Type:  "bloat-size",
					Score: 1600, // CRITICAL
				},
			},
			expectCritical: true,
		},
		{
			name:           "no matches",
			inferredFiles:  []string{"pkg/other/file.go"},
			hotspots:       []HotspotWarning{},
			expectCritical: false,
		},
		{
			name:           "nil inputs",
			inferredFiles:  nil,
			hotspots:       nil,
			expectCritical: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindCriticalHotspot(tt.inferredFiles, tt.hotspots)

			if tt.expectCritical && result == nil {
				t.Errorf("FindCriticalHotspot() expected critical hotspot, got nil")
			}

			if !tt.expectCritical && result != nil {
				t.Errorf("FindCriticalHotspot() expected nil, got %+v", result)
			}

			if result != nil && result.Score <= 1500 {
				t.Errorf("FindCriticalHotspot() returned hotspot with non-critical score: %d", result.Score)
			}
		})
	}
}

func TestMatchesFilePath(t *testing.T) {
	tests := []struct {
		name         string
		inferredFile string
		hotspotPath  string
		expected     bool
	}{
		{
			name:         "exact match",
			inferredFile: "pkg/daemon/daemon.go",
			hotspotPath:  "pkg/daemon/daemon.go",
			expected:     true,
		},
		{
			name:         "filename matches full path",
			inferredFile: "spawn_cmd.go",
			hotspotPath:  "cmd/orch/spawn_cmd.go",
			expected:     true,
		},
		{
			name:         "no match",
			inferredFile: "other.go",
			hotspotPath:  "pkg/daemon/daemon.go",
			expected:     false,
		},
		{
			name:         "case insensitive match",
			inferredFile: "Daemon.Go",
			hotspotPath:  "pkg/daemon/daemon.go",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFilePath(tt.inferredFile, tt.hotspotPath)
			if result != tt.expected {
				t.Errorf("matchesFilePath(%q, %q) = %v, expected %v",
					tt.inferredFile, tt.hotspotPath, result, tt.expected)
			}
		})
	}
}

func TestGenerateExtractionTask(t *testing.T) {
	tests := []struct {
		name         string
		issue        *Issue
		criticalFile string
		contains     []string // Check that these substrings are present
	}{
		{
			name: "extraction from spawn_cmd",
			issue: &Issue{
				Title:       "Add hotspot detection to spawn",
				Description: "",
			},
			criticalFile: "cmd/orch/spawn_cmd.go",
			contains:     []string{"Extract", "cmd/orch/spawn_cmd.go", "Pure structural extraction", "no behavior changes"},
		},
		{
			name: "extraction from daemon",
			issue: &Issue{
				Title:       "Fix daemon spawn logic",
				Description: "",
			},
			criticalFile: "pkg/daemon/daemon.go",
			contains:     []string{"Extract", "pkg/daemon/daemon.go", "Pure structural extraction"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateExtractionTask(tt.issue, tt.criticalFile)

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("GenerateExtractionTask() result doesn't contain %q\nGot: %s", substr, result)
				}
			}
		})
	}
}

func TestInferConcernFromIssue(t *testing.T) {
	tests := []struct {
		name     string
		issue    *Issue
		expected string
	}{
		{
			name: "add verb",
			issue: &Issue{
				Title: "Add hotspot detection",
			},
			expected: "hotspot detection",
		},
		{
			name: "fix verb",
			issue: &Issue{
				Title: "Fix daemon spawn",
			},
			expected: "daemon spawn",
		},
		{
			name: "no action verb",
			issue: &Issue{
				Title: "Daemon spawn improvements",
			},
			expected: "daemon spawn improvements",
		},
		{
			name: "long title truncated",
			issue: &Issue{
				Title: "This is a very long title with many words that should be truncated",
			},
			expected: "this is a very long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferConcernFromIssue(tt.issue)
			if result != tt.expected {
				t.Errorf("inferConcernFromIssue() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestInferTargetPackage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "cmd file to pkg",
			filePath: "cmd/orch/spawn_cmd.go",
			expected: "pkg/orch/",
		},
		{
			name:     "pkg file to extracted",
			filePath: "pkg/daemon/daemon.go",
			expected: "pkg/daemon/extracted/",
		},
		{
			name:     "no directory",
			filePath: "file.go",
			expected: "pkg/appropriate/package/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferTargetPackage(tt.filePath)
			if result != tt.expected {
				t.Errorf("inferTargetPackage(%q) = %q, expected %q", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestCheckExtractionNeeded(t *testing.T) {
	tests := []struct {
		name     string
		issue    *Issue
		hotspots []HotspotWarning
		expected bool
	}{
		{
			name: "critical hotspot triggers extraction",
			issue: &Issue{
				Title:       "Add feature to cmd/orch/spawn_cmd.go",
				Description: "Implement new spawn logic",
			},
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
			expected: true,
		},
		{
			name: "non-critical hotspot skips extraction",
			issue: &Issue{
				Title:       "Fix bug in pkg/daemon/daemon.go",
				Description: "",
			},
			hotspots: []HotspotWarning{
				{Path: "pkg/daemon/daemon.go", Type: "bloat-size", Score: 1200},
			},
			expected: false,
		},
		{
			name: "no file mentions skips extraction",
			issue: &Issue{
				Title:       "Add new feature",
				Description: "General improvement",
			},
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
			expected: false,
		},
		{
			name:  "nil issue returns nil",
			issue: nil,
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
			expected: false,
		},
		{
			name: "nil checker returns nil",
			issue: &Issue{
				Title: "Fix cmd/orch/spawn_cmd.go",
			},
			hotspots: nil, // nil checker
			expected: false,
		},
		{
			name: "fix-density type ignored",
			issue: &Issue{
				Title: "Fix cmd/orch/spawn_cmd.go",
			},
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "fix-density", Score: 2000},
			},
			expected: false,
		},
		{
			name: "extraction issues skipped to prevent recursion",
			issue: &Issue{
				Title:       "Extract spawn flags phase 1: --mode from cmd/orch/spawn_cmd.go into pkg/orch/. Pure structural extraction — no behavior changes.",
				Description: "Auto-generated extraction task",
			},
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2200},
			},
			expected: false, // Should NOT trigger extraction even though file is >1500 lines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var checker HotspotChecker
			if tt.hotspots != nil {
				checker = &mockHotspotChecker{hotspots: tt.hotspots}
			}

			result := CheckExtractionNeeded(tt.issue, checker)

			if tt.expected && (result == nil || !result.Needed) {
				t.Errorf("CheckExtractionNeeded() expected extraction needed, got %+v", result)
			}
			if !tt.expected && result != nil && result.Needed {
				t.Errorf("CheckExtractionNeeded() expected no extraction, got %+v", result)
			}

			// Verify result fields when extraction is needed
			if result != nil && result.Needed {
				if result.CriticalFile == "" {
					t.Error("ExtractionResult.CriticalFile should not be empty")
				}
				if result.ExtractionTask == "" {
					t.Error("ExtractionResult.ExtractionTask should not be empty")
				}
				if result.Hotspot == nil {
					t.Error("ExtractionResult.Hotspot should not be nil")
				}
				if !strings.Contains(result.ExtractionTask, "Extract") {
					t.Errorf("ExtractionTask should contain 'Extract', got: %s", result.ExtractionTask)
				}
			}
		})
	}
}

// mockHotspotChecker implements HotspotChecker for testing.
type mockHotspotChecker struct {
	hotspots []HotspotWarning
}

func (m *mockHotspotChecker) CheckHotspots(projectDir string) ([]HotspotWarning, error) {
	return m.hotspots, nil
}

func TestParseBeadsIDFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "standard beads ID",
			output:   "Created orch-go-b8c\n",
			expected: "orch-go-b8c",
		},
		{
			name:     "beads ID only",
			output:   "orch-go-a1b2",
			expected: "orch-go-a1b2",
		},
		{
			name:     "with extra whitespace",
			output:   "  orch-go-def3  \n",
			expected: "orch-go-def3",
		},
		{
			name:     "longer project name",
			output:   "my-cool-project-abc1",
			expected: "my-cool-project-abc1",
		},
		{
			name:     "no beads ID",
			output:   "error: something failed",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBeadsIDFromOutput(tt.output)
			if result != tt.expected {
				t.Errorf("parseBeadsIDFromOutput(%q) = %q, expected %q", tt.output, result, tt.expected)
			}
		})
	}
}

// mockDaemonHotspotChecker implements HotspotChecker for daemon-level auto-extraction tests.
type mockDaemonHotspotChecker struct {
	hotspots []HotspotWarning
}

func (m *mockDaemonHotspotChecker) CheckHotspots(projectDir string) ([]HotspotWarning, error) {
	return m.hotspots, nil
}

func TestOnceExcluding_AutoExtraction_SpawnsExtractionWhenCriticalHotspot(t *testing.T) {
	// When a triage:ready issue targets a CRITICAL hotspot file (>1500 lines),
	// the daemon should create an extraction issue and spawn it instead.
	spawnedID := ""
	d := &Daemon{
		Config: Config{Verbose: true},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			CreateExtractionIssueFunc: func(task, parentID string) (string, error) {
				// Verify the extraction task was generated correctly
				if parentID != "proj-1" {
					t.Errorf("CreateExtractionIssue parentID = %q, want 'proj-1'", parentID)
				}
				if !strings.Contains(task, "Extract") {
					t.Errorf("CreateExtractionIssue task should contain 'Extract', got: %s", task)
				}
				return "proj-ext1", nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockDaemonHotspotChecker{
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil // Mock: always succeed
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the extraction issue, not the original
	if spawnedID != "proj-ext1" {
		t.Errorf("Spawner called with %q, want 'proj-ext1' (extraction issue)", spawnedID)
	}
	if !result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be true")
	}
	if result.OriginalIssueID != "proj-1" {
		t.Errorf("OnceResult.OriginalIssueID = %q, want 'proj-1'", result.OriginalIssueID)
	}
	if result.Issue.ID != "proj-ext1" {
		t.Errorf("OnceResult.Issue.ID = %q, want 'proj-ext1'", result.Issue.ID)
	}
}

func TestOnceExcluding_AutoExtraction_SkipsWhenNoCriticalHotspot(t *testing.T) {
	// When hotspot check finds no CRITICAL files, spawn normally.
	spawnedID := ""
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to pkg/daemon/daemon.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockDaemonHotspotChecker{
			hotspots: []HotspotWarning{
				// Below critical threshold
				{Path: "pkg/daemon/daemon.go", Type: "bloat-size", Score: 1200},
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil // Mock: always succeed
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned the original issue normally
	if spawnedID != "proj-1" {
		t.Errorf("Spawner called with %q, want 'proj-1' (original issue)", spawnedID)
	}
	if result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be false")
	}
}

func TestOnceExcluding_AutoExtraction_FailsFastOnExtractionFailure(t *testing.T) {
	// When extraction issue creation fails, skip the issue (fail-fast).
	// Extraction gate is non-negotiable - do not proceed with normal spawn.
	spawnedID := ""
	d := &Daemon{
		Config: Config{Verbose: true},
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
			CreateExtractionIssueFunc: func(task, parentID string) (string, error) {
				return "", fmt.Errorf("bd create failed: command not found")
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnedID = beadsID
				return nil
			},
		},
		HotspotChecker: &mockDaemonHotspotChecker{
			hotspots: []HotspotWarning{
				{Path: "cmd/orch/spawn_cmd.go", Type: "bloat-size", Score: 2000},
			},
		},
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("OnceExcluding() expected non-nil result")
	}

	// Should NOT have processed/spawned - extraction gate is non-negotiable
	if result.Processed {
		t.Error("OnceExcluding() should NOT process when extraction setup fails (fail-fast)")
	}

	// Should not have spawned the original issue
	if spawnedID != "" {
		t.Errorf("Spawner should not be called when extraction fails, but was called with %q", spawnedID)
	}

	// Should have a message explaining the skip
	if result.Message == "" {
		t.Error("OnceResult.Message should explain why issue was skipped")
	}
}

func TestOnceExcluding_AutoExtraction_SkipsWhenNoHotspotChecker(t *testing.T) {
	// When HotspotChecker is nil, no extraction check happens.
	spawnedID := ""
	d := &Daemon{
		Issues: &mockIssueQuerier{
			ListReadyIssuesFunc: func() ([]Issue, error) {
				return []Issue{
					{
						ID:        "proj-1",
						Title:     "Add feature to cmd/orch/spawn_cmd.go",
						Priority:  2,
						IssueType: "feature",
						Status:    "open",
					},
				}, nil
			},
		},
		Spawner: &mockSpawner{
			SpawnWorkFunc: func(beadsID, model, workdir string) error {
				spawnedID = beadsID
				return nil
			},
		},
		StatusUpdater: &mockIssueUpdater{
			UpdateStatusFunc: func(beadsID string, status string) error {
				return nil // Mock: always succeed
			},
		},
		// HotspotChecker is nil
	}

	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result == nil || !result.Processed {
		t.Fatal("OnceExcluding() expected processed result")
	}

	// Should have spawned normally without extraction check
	if spawnedID != "proj-1" {
		t.Errorf("Spawner called with %q, want 'proj-1'", spawnedID)
	}
	if result.ExtractionSpawned {
		t.Error("OnceResult.ExtractionSpawned should be false when no checker")
	}
}

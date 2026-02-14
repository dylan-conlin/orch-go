// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
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

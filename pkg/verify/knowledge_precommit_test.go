package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckStagedKnowledge(t *testing.T) {
	tests := []struct {
		name         string
		stagedFiles  map[string]string // path -> content
		expectPassed bool
		expectCount  int // number of orphan files
	}{
		{
			name: "investigation with Model field passes",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-inv-test.md": "# Investigation\n\n**Model:** knowledge-physics\n\n**Question:** test\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "investigation without Model field blocks",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-inv-test.md": "# Investigation\n\n**Question:** test\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "investigation with Orphan acknowledged passes",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-inv-test.md": "# Investigation\n\n**Orphan:** acknowledged\n\n**Question:** test\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "non-investigation kb file ignored",
			stagedFiles: map[string]string{
				".kb/guides/test-guide.md": "# Guide\n\nSome guide content\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "modified investigation not checked (only new)",
			stagedFiles: map[string]string{}, // handled by setup below
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "investigation with probe file also staged passes",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-inv-test.md":                              "# Investigation\n\n**Question:** test\n",
				".kb/models/knowledge-physics/probes/2026-03-09-probe-test.md": "# Probe\n\nContent\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "multiple investigations one orphaned",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-inv-coupled.md":  "# Investigation\n\n**Model:** test-model\n",
				".kb/investigations/2026-03-09-inv-orphaned.md": "# Investigation\n\n**Question:** test\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "design type investigation same rules",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-design-test.md": "# Design\n\n**Question:** test\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "probe file without investigation passes",
			stagedFiles: map[string]string{
				".kb/models/test-model/probes/2026-03-09-probe-test.md": "# Probe\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "empty Model field blocks",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-09-inv-test.md": "# Investigation\n\n**Model:** \n\n**Question:** test\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupGitRepoForStaged(t)

			if tt.name == "modified investigation not checked (only new)" {
				// Create and commit an investigation WITHOUT Model field
				createInvestigationFile(t, tmpDir, ".kb/investigations/2026-03-09-inv-existing.md",
					"# Investigation\n\n**Question:** test\n")
				commitFiles(t, tmpDir, "Initial investigation")
				// Now modify it (should not trigger the check)
				createInvestigationFile(t, tmpDir, ".kb/investigations/2026-03-09-inv-existing.md",
					"# Investigation\n\n**Question:** updated test\n")
				stageAllFiles(t, tmpDir)
			} else {
				// Stage all test files
				for path, content := range tt.stagedFiles {
					createInvestigationFile(t, tmpDir, path, content)
				}
				if len(tt.stagedFiles) > 0 {
					stageAllFiles(t, tmpDir)
				}
			}

			result := CheckStagedKnowledge(tmpDir)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Passed != tt.expectPassed {
				t.Errorf("Passed = %v, want %v; orphans: %v", result.Passed, tt.expectPassed, result.OrphanFiles)
			}

			if len(result.OrphanFiles) != tt.expectCount {
				t.Errorf("expected %d orphan files, got %d: %v", tt.expectCount, len(result.OrphanFiles), result.OrphanFiles)
			}
		})
	}
}

func TestCheckStagedKnowledge_EmptyDir(t *testing.T) {
	result := CheckStagedKnowledge("")
	if result != nil {
		t.Error("expected nil for empty dir")
	}
}

func TestFormatStagedKnowledgeError(t *testing.T) {
	result := &StagedKnowledgeResult{
		Passed: false,
		OrphanFiles: []string{
			".kb/investigations/2026-03-09-inv-test.md",
		},
	}

	msg := FormatStagedKnowledgeError(result)
	if msg == "" {
		t.Error("expected non-empty error message")
	}

	// Should mention the file
	if !containsStr(msg, "inv-test.md") {
		t.Error("error should mention the orphan file")
	}

	// Should mention override
	if !containsStr(msg, "FORCE_ORPHAN") {
		t.Error("error should mention FORCE_ORPHAN override")
	}

	// Should mention Model field
	if !containsStr(msg, "Model:") {
		t.Error("error should mention Model field")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// createInvestigationFile creates a file at the given path with content, creating dirs as needed.
func createInvestigationFile(t *testing.T, repoDir, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(repoDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", relPath, err)
	}
}

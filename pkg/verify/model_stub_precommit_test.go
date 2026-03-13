package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckStagedModelStubs(t *testing.T) {
	tests := []struct {
		name         string
		stagedFiles  map[string]string // path -> content
		expectPassed bool
		expectCount  int // number of stub files
	}{
		{
			name: "filled model passes",
			stagedFiles: map[string]string{
				".kb/models/test-model/model.md": "# Model: Test Model\n\n**Created:** 2026-03-11\n**Status:** Active\n\n## What This Is\n\nThis model describes real behavior.\n\n## Core Claims (Testable)\n\n### Claim 1: Real claim\n\nReal explanation.\n\n**Test:** Run the test suite\n\n**Status:** Confirmed\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "stub model blocks",
			stagedFiles: map[string]string{
				".kb/models/test-model/model.md": "# Model: Test Model\n\n**Created:** 2026-03-11\n**Status:** Active\n\n## What This Is\n\n[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]\n\n## Core Claims (Testable)\n\n### Claim 1: [Concise claim statement]\n\n[Explanation of the claim.]\n\n**Test:** [How to test this claim]\n\n**Status:** Hypothesis\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "non-model file ignored",
			stagedFiles: map[string]string{
				".kb/investigations/2026-03-11-inv-test.md": "# Investigation\n\n[Concise claim statement]\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "probe file not checked",
			stagedFiles: map[string]string{
				".kb/models/test-model/probes/2026-03-11-probe-test.md": "# Probe\n\n[How to test this claim]\n",
			},
			expectPassed: true,
			expectCount:  0,
		},
		{
			name: "global model stub blocks",
			stagedFiles: map[string]string{
				".kb/global/models/test-global/model.md": "# Model: Test\n\n## What This Is\n\n[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "partially filled model with remaining placeholder blocks",
			stagedFiles: map[string]string{
				".kb/models/partial/model.md": "# Model: Partial\n\n## What This Is\n\nReal content here.\n\n## Boundaries\n\n**What this model covers:**\n- [Scope item 1]\n\n**What this model does NOT cover:**\n- [Exclusion 1]\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "multiple models one stub",
			stagedFiles: map[string]string{
				".kb/models/good-model/model.md": "# Model: Good\n\n## What This Is\n\nReal content.\n\n## Core Claims\n\n### Claim 1: Real claim\n",
				".kb/models/bad-model/model.md":  "# Model: Bad\n\n## What This Is\n\n[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]\n",
			},
			expectPassed: false,
			expectCount:  1,
		},
		{
			name: "empty dir returns nil",
			stagedFiles: nil,
			expectPassed: true, // will test separately
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "empty dir returns nil" {
				result := CheckStagedModelStubs("")
				if result != nil {
					t.Error("expected nil for empty dir")
				}
				return
			}

			tmpDir := setupGitRepoForStaged(t)

			for path, content := range tt.stagedFiles {
				createInvestigationFile(t, tmpDir, path, content)
			}
			if len(tt.stagedFiles) > 0 {
				stageAllFiles(t, tmpDir)
			}

			result := CheckStagedModelStubs(tmpDir)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Passed != tt.expectPassed {
				t.Errorf("Passed = %v, want %v; stubs: %v", result.Passed, tt.expectPassed, result.StubFiles)
			}

			if len(result.StubFiles) != tt.expectCount {
				t.Errorf("expected %d stub files, got %d: %v", tt.expectCount, len(result.StubFiles), result.StubFiles)
			}
		})
	}
}

func TestFindPlaceholders(t *testing.T) {
	tests := []struct {
		name    string
		content string
		count   int
	}{
		{
			name:    "no placeholders",
			content: "# Model: Real\n\nReal content everywhere.\n",
			count:   0,
		},
		{
			name:    "single placeholder",
			content: "## Boundaries\n\n- [Scope item 1]\n",
			count:   1,
		},
		{
			name:    "multiple placeholders",
			content: "[What phenomenon or pattern does this model describe? foo]\n[Concise claim statement]\n[How to test this claim]\n",
			count:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := findPlaceholders(tt.content)
			if len(found) != tt.count {
				t.Errorf("expected %d placeholders, got %d: %v", tt.count, len(found), found)
			}
		})
	}
}

func TestIsModelStubCandidate(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{".kb/models/test/model.md", true},
		{".kb/global/models/test/model.md", true},
		{".kb/models/test/probes/probe.md", false},
		{".kb/investigations/inv.md", false},
		{".kb/models/test/README.md", false},
		{"model.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isModelStubCandidate(tt.path); got != tt.expect {
				t.Errorf("isModelStubCandidate(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestFormatStagedModelStubError(t *testing.T) {
	result := &StagedModelStubResult{
		Passed: false,
		StubFiles: []ModelStubInfo{
			{
				Path:         ".kb/models/test/model.md",
				Placeholders: []string{"[Concise claim statement]"},
			},
		},
	}

	msg := FormatStagedModelStubError(result)
	if msg == "" {
		t.Error("expected non-empty error message")
	}

	if !containsStr(msg, "test/model.md") {
		t.Error("error should mention the stub file")
	}

	if !containsStr(msg, "FORCE_MODEL_STUB") {
		t.Error("error should mention override env var")
	}

	if !containsStr(msg, "Concise claim") {
		t.Error("error should mention the placeholder found")
	}

	// nil result returns empty
	if FormatStagedModelStubError(nil) != "" {
		t.Error("nil result should return empty string")
	}

	// passed result returns empty
	if FormatStagedModelStubError(&StagedModelStubResult{Passed: true}) != "" {
		t.Error("passed result should return empty string")
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

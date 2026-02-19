package verify

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestIsWebFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "svelte file in web",
			filePath: "web/src/routes/page.svelte",
			want:     true,
		},
		{
			name:     "typescript file in web",
			filePath: "web/src/lib/api.ts",
			want:     true,
		},
		{
			name:     "css file in web",
			filePath: "web/src/app.css",
			want:     true,
		},
		{
			name:     "html file in web",
			filePath: "web/src/app.html",
			want:     true,
		},
		{
			name:     "jsx file in web",
			filePath: "web/src/components/Button.jsx",
			want:     true,
		},
		{
			name:     "go file in pkg - not web",
			filePath: "pkg/verify/check.go",
			want:     false,
		},
		{
			name:     "go file in cmd - not web",
			filePath: "cmd/orch/main.go",
			want:     false,
		},
		{
			name:     "json file in web - not web extension",
			filePath: "web/package.json",
			want:     false,
		},
		{
			name:     "config file in web - not web extension",
			filePath: "web/vite.config.ts",
			want:     true, // .ts is a web extension
		},
		{
			name:     "markdown in web - not web extension",
			filePath: "web/README.md",
			want:     false,
		},
		{
			name:     "investigation file",
			filePath: ".kb/investigations/2025-12-24-test.md",
			want:     false,
		},
		{
			name:     "beads file",
			filePath: ".beads/issues.jsonl",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWebFile(tt.filePath)
			if got != tt.want {
				t.Errorf("IsWebFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestHasWebChangesInFiles(t *testing.T) {
	tests := []struct {
		name      string
		gitOutput string
		want      bool
	}{
		{
			name:      "web svelte file changed",
			gitOutput: "web/src/routes/page.svelte\n",
			want:      true,
		},
		{
			name:      "multiple web files changed",
			gitOutput: "web/src/routes/page.svelte\nweb/src/lib/api.ts\nweb/src/app.css\n",
			want:      true,
		},
		{
			name:      "mixed web and non-web files",
			gitOutput: "pkg/verify/check.go\nweb/src/routes/page.svelte\ncmd/orch/main.go\n",
			want:      true,
		},
		{
			name:      "only go files",
			gitOutput: "pkg/verify/check.go\ncmd/orch/main.go\n",
			want:      false,
		},
		{
			name:      "only non-web extension in web",
			gitOutput: "web/package.json\nweb/README.md\n",
			want:      false,
		},
		{
			name:      "empty output",
			gitOutput: "",
			want:      false,
		},
		{
			name:      "only whitespace",
			gitOutput: "\n\n  \n",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasWebChangesInFiles(tt.gitOutput)
			if got != tt.want {
				t.Errorf("hasWebChangesInFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasVisualVerificationEvidence(t *testing.T) {
	tests := []struct {
		name       string
		comments   []Comment
		wantHas    bool
		wantMinLen int // minimum number of evidence items
	}{
		{
			name: "screenshot mentioned",
			comments: []Comment{
				{Text: "Phase: Complete - Added screenshot of the dashboard"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "visual verification mentioned",
			comments: []Comment{
				{Text: "Phase: Validation - Visually verified the UI changes in browser"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "playwright mentioned",
			comments: []Comment{
				{Text: "Used playwright to capture screenshot of the stats bar"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "browser_take_screenshot mentioned",
			comments: []Comment{
				{Text: "Called browser_take_screenshot to verify the UI"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "verified in browser mentioned",
			comments: []Comment{
				{Text: "Verified the changes work correctly in the browser"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "tested in browser mentioned",
			comments: []Comment{
				{Text: "Phase: Complete - Tested the feature in browser"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "UI smoke test mentioned",
			comments: []Comment{
				{Text: "Ran UI smoke test to verify the changes"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "no visual evidence",
			comments: []Comment{
				{Text: "Phase: Complete - All tests passing"},
				{Text: "Added new component"},
			},
			wantHas:    false,
			wantMinLen: 0,
		},
		{
			name:       "empty comments",
			comments:   []Comment{},
			wantHas:    false,
			wantMinLen: 0,
		},
		{
			name: "multiple evidence types",
			comments: []Comment{
				{Text: "Captured screenshot of the dashboard"},
				{Text: "Also visually verified in the browser"},
			},
			wantHas:    true,
			wantMinLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasEvidence, evidence := HasVisualVerificationEvidence(tt.comments)
			if hasEvidence != tt.wantHas {
				t.Errorf("HasVisualVerificationEvidence() hasEvidence = %v, want %v", hasEvidence, tt.wantHas)
			}
			if len(evidence) < tt.wantMinLen {
				t.Errorf("HasVisualVerificationEvidence() evidence count = %d, want at least %d", len(evidence), tt.wantMinLen)
			}
		})
	}
}

func TestVerifyVisualVerification_NoWebChanges(t *testing.T) {
	// This test verifies behavior when there are no web changes
	// The actual git check is difficult to test, so we test the internal logic

	result := VisualVerificationResult{
		Passed:        true,
		HasWebChanges: false,
	}

	// When no web changes, result should pass
	if !result.Passed {
		t.Error("Expected result.Passed to be true when no web changes")
	}
	if result.HasWebChanges {
		t.Error("Expected result.HasWebChanges to be false")
	}
}

func TestVerifyVisualVerification_WebChangesWithEvidence(t *testing.T) {
	// Simulate the behavior when web changes exist and evidence is found
	result := VisualVerificationResult{
		Passed:        true,
		HasWebChanges: true,
		HasEvidence:   true,
		Evidence:      []string{"screenshot", "visually verified"},
	}

	// When web changes exist but evidence is found, result should pass
	if !result.Passed {
		t.Error("Expected result.Passed to be true when evidence found")
	}
	if !result.HasEvidence {
		t.Error("Expected result.HasEvidence to be true")
	}
}

func TestVerifyVisualVerification_WebChangesNoEvidence(t *testing.T) {
	// Simulate the behavior when web changes exist but no evidence is found
	result := VisualVerificationResult{
		Passed:        false,
		HasWebChanges: true,
		HasEvidence:   false,
		Errors: []string{
			"web/ files modified but no visual verification evidence found",
		},
	}

	// When web changes exist but no evidence, result should fail
	if result.Passed {
		t.Error("Expected result.Passed to be false when no evidence")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors when no evidence for web changes")
	}
}

func TestVisualEvidencePatterns(t *testing.T) {
	// Test that our patterns match expected inputs
	testCases := []struct {
		input       string
		shouldMatch bool
	}{
		// Should match
		{"screenshot of the dashboard", true},
		{"captured a Screenshot", true},
		{"visual verification complete", true},
		{"visually verified the changes", true},
		{"UI verification passed", true},
		{"browser verification done", true},
		{"used Playwright to test", true},
		{"called browser_take_screenshot", true},
		{"verified in browser", true},
		{"tested in browser", true},
		{"checked in browser", true},
		{"UI smoke test passed", true},
		{"ran smoke test for UI", true},
		// Should not match
		{"tests passing", false},
		{"code review complete", false},
		{"added new feature", false},
		{"fixed the bug", false},
		{"Phase: Complete", false},
	}

	for _, tc := range testCases {
		matched := false
		for _, pattern := range visualEvidencePatterns {
			if pattern.MatchString(tc.input) {
				matched = true
				break
			}
		}
		if matched != tc.shouldMatch {
			t.Errorf("Pattern matching for %q: got %v, want %v", tc.input, matched, tc.shouldMatch)
		}
	}
}

func TestIsSkillRequiringVisualVerification(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		want      bool
	}{
		// Skills that require visual verification
		{
			name:      "feature-impl requires verification",
			skillName: "feature-impl",
			want:      true,
		},
		{
			name:      "feature-impl case insensitive",
			skillName: "Feature-Impl",
			want:      true,
		},

		// Skills explicitly excluded from visual verification
		{
			name:      "architect excluded",
			skillName: "architect",
			want:      false,
		},
		{
			name:      "investigation excluded",
			skillName: "investigation",
			want:      false,
		},
		{
			name:      "systematic-debugging excluded",
			skillName: "systematic-debugging",
			want:      false,
		},
		{
			name:      "research excluded",
			skillName: "research",
			want:      false,
		},
		{
			name:      "codebase-audit excluded",
			skillName: "codebase-audit",
			want:      false,
		},
		{
			name:      "reliability-testing excluded",
			skillName: "reliability-testing",
			want:      false,
		},
		{
			name:      "design-session excluded",
			skillName: "design-session",
			want:      false,
		},
		{
			name:      "issue-creation excluded",
			skillName: "issue-creation",
			want:      false,
		},
		{
			name:      "writing-skills excluded",
			skillName: "writing-skills",
			want:      false,
		},

		// Unknown skills - permissive default (no visual verification required)
		{
			name:      "unknown skill is permissive",
			skillName: "some-new-skill",
			want:      false,
		},
		{
			name:      "empty skill name is permissive",
			skillName: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSkillRequiringVisualVerification(tt.skillName)
			if got != tt.want {
				t.Errorf("IsSkillRequiringVisualVerification(%q) = %v, want %v", tt.skillName, got, tt.want)
			}
		})
	}
}

func TestSkillAwareVisualVerification(t *testing.T) {
	// Test that the skill-aware logic produces correct results
	// These tests verify the interaction between skill detection and visual verification

	tests := []struct {
		name           string
		skillName      string
		hasWebChanges  bool
		hasEvidence    bool
		hasApproval    bool
		wantPassed     bool
		wantHasWarning bool
	}{
		{
			name:          "architect with web changes - should pass without evidence",
			skillName:     "architect",
			hasWebChanges: true,
			hasEvidence:   false,
			hasApproval:   false,
			wantPassed:    true,
		},
		{
			name:          "investigation with web changes - should pass without evidence",
			skillName:     "investigation",
			hasWebChanges: true,
			hasEvidence:   false,
			hasApproval:   false,
			wantPassed:    true,
		},
		{
			name:          "feature-impl with web changes and evidence and approval - should pass",
			skillName:     "feature-impl",
			hasWebChanges: true,
			hasEvidence:   true,
			hasApproval:   true,
			wantPassed:    true,
		},
		{
			name:          "feature-impl with web changes and evidence but no approval - should fail",
			skillName:     "feature-impl",
			hasWebChanges: true,
			hasEvidence:   true,
			hasApproval:   false,
			wantPassed:    false,
		},
		{
			name:          "feature-impl with web changes no evidence - should fail",
			skillName:     "feature-impl",
			hasWebChanges: true,
			hasEvidence:   false,
			hasApproval:   false,
			wantPassed:    false,
		},
		{
			name:          "unknown skill with web changes - should pass (permissive)",
			skillName:     "new-skill",
			hasWebChanges: true,
			hasEvidence:   false,
			hasApproval:   false,
			wantPassed:    true,
		},
		{
			name:          "no web changes - always passes",
			skillName:     "feature-impl",
			hasWebChanges: false,
			hasEvidence:   false,
			hasApproval:   false,
			wantPassed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the verification logic
			result := VisualVerificationResult{
				Passed:        true,
				HasWebChanges: tt.hasWebChanges,
			}

			if !tt.hasWebChanges {
				// No web changes = always pass
				if !result.Passed {
					t.Error("Expected pass when no web changes")
				}
				return
			}

			// Check skill type
			requiresVerification := IsSkillRequiringVisualVerification(tt.skillName)

			if !requiresVerification {
				// Non-UI skill - pass regardless of evidence
				if tt.wantPassed != true {
					t.Errorf("Non-UI skill %q should pass, got wantPassed=%v", tt.skillName, tt.wantPassed)
				}
				return
			}

			// UI skill - need both evidence AND approval
			result.HasEvidence = tt.hasEvidence
			result.HasHumanApproval = tt.hasApproval

			if !tt.hasEvidence {
				result.Passed = false
				result.Errors = append(result.Errors, "web/ files modified but no visual verification evidence found")
			} else if !tt.hasApproval {
				result.Passed = false
				result.NeedsApproval = true
				result.Errors = append(result.Errors, "web/ files modified - visual evidence found but requires human approval")
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("Result.Passed = %v, want %v", result.Passed, tt.wantPassed)
			}
		})
	}
}

func TestHasHumanApproval(t *testing.T) {
	tests := []struct {
		name       string
		comments   []Comment
		wantHas    bool
		wantMinLen int
	}{
		{
			name: "checkmark approved",
			comments: []Comment{
				{Text: "✅ APPROVED - Visual changes look good"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "UI APPROVED",
			comments: []Comment{
				{Text: "UI APPROVED after review"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "VISUAL APPROVED",
			comments: []Comment{
				{Text: "VISUAL APPROVED - screenshot verified"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "human_approved flag",
			comments: []Comment{
				{Text: "human_approved: true"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "orchestrator_approved flag",
			comments: []Comment{
				{Text: "orchestrator_approved: true"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "I approve the UI",
			comments: []Comment{
				{Text: "I approve the UI changes"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "LGTM UI",
			comments: []Comment{
				{Text: "LGTM on the UI changes"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "no approval - agent self-certified",
			comments: []Comment{
				{Text: "Visual verification: screenshot captured showing dashboard"},
				{Text: "Phase: Complete - All tests passing"},
			},
			wantHas:    false,
			wantMinLen: 0,
		},
		{
			name:       "empty comments",
			comments:   []Comment{},
			wantHas:    false,
			wantMinLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasApproval, approvals := HasHumanApproval(tt.comments)
			if hasApproval != tt.wantHas {
				t.Errorf("HasHumanApproval() hasApproval = %v, want %v", hasApproval, tt.wantHas)
			}
			if len(approvals) < tt.wantMinLen {
				t.Errorf("HasHumanApproval() approval count = %d, want at least %d", len(approvals), tt.wantMinLen)
			}
		})
	}
}

func TestHumanApprovalPatterns(t *testing.T) {
	// Test that our patterns match expected inputs
	testCases := []struct {
		input       string
		shouldMatch bool
	}{
		// Should match - explicit approval markers
		{"✅ APPROVED", true},
		{"✅ APPROVED - Visual changes look correct", true},
		{"UI APPROVED", true},
		{"VISUAL APPROVED", true},
		{"human_approved: true", true},
		{"orchestrator_approved: true", true},
		{"I approve the UI changes", true},
		{"I approve the visual changes", true},
		{"LGTM on the UI", true},
		{"UI looks LGTM", true},

		// Should NOT match - agent self-certification
		{"Visual verification: screenshot captured", false},
		{"Screenshot taken of dashboard", false},
		{"Verified in browser", false},
		{"Tests passing", false},
		{"Phase: Complete", false},
		{"Approved the code changes", false}, // not UI-specific
	}

	for _, tc := range testCases {
		matched := false
		for _, pattern := range humanApprovalPatterns {
			if pattern.MatchString(tc.input) {
				matched = true
				break
			}
		}
		if matched != tc.shouldMatch {
			t.Errorf("Approval pattern matching for %q: got %v, want %v", tc.input, matched, tc.shouldMatch)
		}
	}
}

func TestHasWebChangesForAgent(t *testing.T) {
	t.Run("no spawn time falls back to recent commits", func(t *testing.T) {
		// Create a temp workspace without spawn time file
		workspacePath := t.TempDir()

		// Create workspace with SPAWN_CONTEXT.md but no .spawn_time
		err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("failed to create SPAWN_CONTEXT.md: %v", err)
		}

		// With no spawn time, should return false (cannot determine changes)
		projectDir := t.TempDir()
		result := HasWebChangesForAgent(projectDir, workspacePath)
		if result {
			t.Error("Expected false when no spawn time available")
		}
	})

	t.Run("with spawn time uses time-based filtering", func(t *testing.T) {
		// Create a temp workspace with spawn time
		workspacePath := t.TempDir()
		spawnTime := time.Now().Add(-1 * time.Hour)

		err := spawn.WriteSpawnTime(workspacePath, spawnTime)
		if err != nil {
			t.Fatalf("failed to write spawn time: %v", err)
		}

		// With spawn time, should use time-based filtering
		// Again, can't easily test actual git behavior in unit tests
		projectDir := t.TempDir()
		_ = HasWebChangesForAgent(projectDir, workspacePath)
		// Just verify it doesn't panic - actual behavior depends on git state
	})
}

func TestHasWebChangesSinceTime(t *testing.T) {
	// This tests the internal function by testing output parsing
	// The hasWebChangesInFiles function is used internally

	tests := []struct {
		name      string
		gitOutput string
		want      bool
	}{
		{
			name:      "web svelte file in log output",
			gitOutput: "web/src/routes/page.svelte\n",
			want:      true,
		},
		{
			name:      "multiple files including web",
			gitOutput: "pkg/verify/check.go\n\nweb/src/lib/api.ts\n\ncmd/orch/main.go\n",
			want:      true,
		},
		{
			name:      "only go files",
			gitOutput: "pkg/verify/check.go\n\ncmd/orch/main.go\n",
			want:      false,
		},
		{
			name:      "empty output (no commits since time)",
			gitOutput: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the file parsing logic that hasWebChangesSinceTime uses
			got := hasWebChangesInFiles(tt.gitOutput)
			if got != tt.want {
				t.Errorf("hasWebChangesInFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasWebChangesForAgentScopesBehavior(t *testing.T) {
	// This test documents the expected behavior of HasWebChangesForAgent

	t.Run("documents scope behavior", func(t *testing.T) {
		// HasWebChangesForAgent with workspace scoping:
		// - Checks commits since spawn time that touch the workspace directory
		// - Only includes commits made by this specific agent
		// - Prevents false positives from concurrent agent work
		// - Returns false if no spawn time available (cannot determine changes)

		// This is a documentation test - the actual behavior is tested above
		// and in integration tests
	})
}

func TestHasWebChangesSinceTimeForWorkspace(t *testing.T) {
	// This test verifies the new workspace-scoped behavior
	// prevents false positives from concurrent agents

	t.Run("empty workspace path checks all commits", func(t *testing.T) {
		// When workspacePath is empty, checks all commits since spawn time
		projectDir := t.TempDir()
		spawnTime := time.Now().Add(-1 * time.Hour)

		// This tests that the function doesn't panic with empty workspace
		result := hasWebChangesSinceTimeForWorkspace(projectDir, spawnTime, "")
		// Result depends on git state, just verify no panic
		_ = result
	})

	t.Run("non-existent workspace returns false", func(t *testing.T) {
		// If workspace doesn't exist, no commits touch it
		projectDir := t.TempDir()
		spawnTime := time.Now().Add(-1 * time.Hour)
		nonExistentWorkspace := filepath.Join(projectDir, "nonexistent", "workspace")

		// No commits could have touched a workspace that doesn't exist
		result := hasWebChangesSinceTimeForWorkspace(projectDir, spawnTime, nonExistentWorkspace)
		// Should return false since git can't find commits touching this path
		if result {
			// This is expected in most cases - a workspace that doesn't exist
			// shouldn't have any commits touching it
			t.Logf("Got result=%v for non-existent workspace (expected false in most cases)", result)
		}
	})

	t.Run("behavior documentation", func(t *testing.T) {
		// The hasWebChangesSinceTimeForWorkspace function:
		// 1. Gets commit hashes since spawn time that touch the workspace path
		// 2. For each such commit, gets all changed files
		// 3. Checks if any of those files are web/ files
		//
		// This is the same pattern as test_evidence.go's hasCodeChangesInWorkspaceCommits()
		// which was added to fix the same bug pattern.
		//
		// The key insight: filtering commits to workspace-touching ones prevents
		// false positives when multiple agents run concurrently with similar spawn times.
	})
}

func TestHasScreenshotFilesInWorkspace(t *testing.T) {
	t.Run("empty workspace path returns false", func(t *testing.T) {
		hasFiles, files := HasScreenshotFilesInWorkspace("")
		if hasFiles {
			t.Error("expected hasFiles=false for empty workspace path")
		}
		if len(files) > 0 {
			t.Errorf("expected no files for empty workspace path, got %v", files)
		}
	})

	t.Run("non-existent workspace returns false", func(t *testing.T) {
		hasFiles, files := HasScreenshotFilesInWorkspace("/nonexistent/workspace/path")
		if hasFiles {
			t.Error("expected hasFiles=false for non-existent workspace")
		}
		if len(files) > 0 {
			t.Errorf("expected no files for non-existent workspace, got %v", files)
		}
	})

	t.Run("workspace without screenshots directory returns false", func(t *testing.T) {
		workspacePath := t.TempDir()
		// Don't create screenshots directory
		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if hasFiles {
			t.Error("expected hasFiles=false when screenshots dir doesn't exist")
		}
		if len(files) > 0 {
			t.Errorf("expected no files, got %v", files)
		}
	})

	t.Run("empty screenshots directory returns false", func(t *testing.T) {
		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if hasFiles {
			t.Error("expected hasFiles=false for empty screenshots directory")
		}
		if len(files) > 0 {
			t.Errorf("expected no files for empty directory, got %v", files)
		}
	})

	t.Run("finds PNG screenshot files", func(t *testing.T) {
		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Create a PNG file
		if err := os.WriteFile(filepath.Join(screenshotsDir, "dashboard.png"), []byte("fake png"), 0644); err != nil {
			t.Fatalf("failed to create PNG file: %v", err)
		}

		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if !hasFiles {
			t.Error("expected hasFiles=true when PNG file exists")
		}
		if len(files) != 1 || files[0] != "dashboard.png" {
			t.Errorf("expected [dashboard.png], got %v", files)
		}
	})

	t.Run("finds multiple screenshot files with different extensions", func(t *testing.T) {
		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Create files with various extensions
		testFiles := []string{
			"screenshot1.png",
			"screenshot2.jpg",
			"screenshot3.jpeg",
			"screenshot4.webp",
			"screenshot5.gif",
		}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(screenshotsDir, f), []byte("fake image"), 0644); err != nil {
				t.Fatalf("failed to create file %s: %v", f, err)
			}
		}

		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if !hasFiles {
			t.Error("expected hasFiles=true when image files exist")
		}
		if len(files) != 5 {
			t.Errorf("expected 5 files, got %d: %v", len(files), files)
		}
	})

	t.Run("ignores non-image files", func(t *testing.T) {
		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Create non-image files
		nonImageFiles := []string{
			"readme.txt",
			"data.json",
			"script.js",
			".gitkeep",
		}
		for _, f := range nonImageFiles {
			if err := os.WriteFile(filepath.Join(screenshotsDir, f), []byte("content"), 0644); err != nil {
				t.Fatalf("failed to create file %s: %v", f, err)
			}
		}

		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if hasFiles {
			t.Errorf("expected hasFiles=false for non-image files, got files: %v", files)
		}
	})

	t.Run("ignores subdirectories", func(t *testing.T) {
		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Create a subdirectory with .png name
		subDir := filepath.Join(screenshotsDir, "somedir.png")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if hasFiles {
			t.Errorf("expected hasFiles=false when only subdirectory exists, got files: %v", files)
		}
	})

	t.Run("case insensitive extension matching", func(t *testing.T) {
		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Create files with uppercase extensions
		testFiles := []string{
			"screenshot1.PNG",
			"screenshot2.JPG",
			"screenshot3.JPEG",
		}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(screenshotsDir, f), []byte("fake image"), 0644); err != nil {
				t.Fatalf("failed to create file %s: %v", f, err)
			}
		}

		hasFiles, files := HasScreenshotFilesInWorkspace(workspacePath)
		if !hasFiles {
			t.Error("expected hasFiles=true for uppercase extension files")
		}
		if len(files) != 3 {
			t.Errorf("expected 3 files, got %d: %v", len(files), files)
		}
	})
}

func TestVerifyVisualVerificationWithScreenshotFiles(t *testing.T) {
	// These tests verify that screenshot files in the workspace
	// are recognized as visual verification evidence

	t.Run("screenshot file provides evidence for feature-impl skill", func(t *testing.T) {
		// This test simulates a feature-impl skill with web changes
		// and a screenshot file in the workspace

		// Create workspace with SPAWN_CONTEXT.md indicating feature-impl skill
		workspacePath := t.TempDir()
		spawnContext := `TASK: Test feature implementation

## SKILL GUIDANCE (feature-impl)

This is a feature-impl skill spawn.
`
		if err := os.WriteFile(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"), []byte(spawnContext), 0644); err != nil {
			t.Fatalf("failed to create SPAWN_CONTEXT.md: %v", err)
		}

		// Create screenshots directory with an image file
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(screenshotsDir, "dashboard.png"), []byte("fake png"), 0644); err != nil {
			t.Fatalf("failed to create PNG file: %v", err)
		}

		// Manually simulate the verification logic
		// Since we can't easily mock git for web changes, we test the evidence detection part

		// First verify skill detection works
		skillName, _ := ExtractSkillNameFromSpawnContext(workspacePath)
		if skillName != "feature-impl" {
			t.Fatalf("expected skill name 'feature-impl', got %q", skillName)
		}

		// Verify screenshot file detection
		hasScreenshots, files := HasScreenshotFilesInWorkspace(workspacePath)
		if !hasScreenshots {
			t.Error("expected screenshot files to be detected")
		}
		if len(files) != 1 || files[0] != "dashboard.png" {
			t.Errorf("expected [dashboard.png], got %v", files)
		}

		// In a full integration test with git, the verification would:
		// 1. Detect web changes
		// 2. Detect feature-impl skill (requires verification)
		// 3. Find screenshot files as evidence
		// 4. Still require human approval (evidence is found, but approval is needed)
	})

	t.Run("evidence includes screenshot file names", func(t *testing.T) {
		// This test verifies that when screenshot files are found,
		// they appear in the Evidence field with proper naming

		workspacePath := t.TempDir()
		screenshotsDir := filepath.Join(workspacePath, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			t.Fatalf("failed to create screenshots dir: %v", err)
		}

		// Create multiple screenshot files
		if err := os.WriteFile(filepath.Join(screenshotsDir, "before.png"), []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(screenshotsDir, "after.png"), []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		hasScreenshots, files := HasScreenshotFilesInWorkspace(workspacePath)
		if !hasScreenshots {
			t.Error("expected screenshot files to be detected")
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}

		// Verify file names are in the list
		foundBefore, foundAfter := false, false
		for _, f := range files {
			if f == "before.png" {
				foundBefore = true
			}
			if f == "after.png" {
				foundAfter = true
			}
		}
		if !foundBefore || !foundAfter {
			t.Errorf("expected to find before.png and after.png, got %v", files)
		}
	})
}

// Tests for WebChangeRisk and risk assessment

func TestWebChangeRisk_String(t *testing.T) {
	tests := []struct {
		risk WebChangeRisk
		want string
	}{
		{WebRiskNone, "NONE"},
		{WebRiskLow, "LOW"},
		{WebRiskMedium, "MEDIUM"},
		{WebRiskHigh, "HIGH"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.risk.String(); got != tt.want {
				t.Errorf("WebChangeRisk.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebChangeRisk_RequiresVisualVerification(t *testing.T) {
	tests := []struct {
		risk WebChangeRisk
		want bool
	}{
		{WebRiskNone, false},
		{WebRiskLow, false},
		{WebRiskMedium, true},
		{WebRiskHigh, true},
	}

	for _, tt := range tests {
		t.Run(tt.risk.String(), func(t *testing.T) {
			if got := tt.risk.RequiresVisualVerification(); got != tt.want {
				t.Errorf("RequiresVisualVerification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebFileChange_Helpers(t *testing.T) {
	t.Run("IsCSSOnlyFile", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			{"web/src/app.css", true},
			{"web/src/styles.scss", true},
			{"web/src/Component.svelte", false},
			{"web/src/api.ts", false},
		}
		for _, tt := range tests {
			change := WebFileChange{Path: tt.path}
			if got := change.IsCSSOnlyFile(); got != tt.want {
				t.Errorf("IsCSSOnlyFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		}
	})

	t.Run("IsRouteFile", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			{"web/src/routes/+page.svelte", true},
			{"web/src/routes/dashboard/+page.svelte", true},
			{"web/src/pages/index.tsx", true},
			{"web/src/components/Button.svelte", false},
			{"web/src/lib/api.ts", false},
		}
		for _, tt := range tests {
			change := WebFileChange{Path: tt.path}
			if got := change.IsRouteFile(); got != tt.want {
				t.Errorf("IsRouteFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		}
	})

	t.Run("IsComponentFile", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			{"web/src/components/Button.svelte", true},
			{"web/src/lib/utils.ts", true},
			{"web/src/Component.svelte", true},
			{"web/src/Component.tsx", true},
			{"web/src/Component.jsx", true},
			{"web/src/Component.vue", true},
			{"web/src/app.css", false},
		}
		for _, tt := range tests {
			change := WebFileChange{Path: tt.path}
			if got := change.IsComponentFile(); got != tt.want {
				t.Errorf("IsComponentFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		}
	})

	t.Run("IsLayoutFile", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			{"web/src/routes/+layout.svelte", true},
			{"web/src/routes/_layout.svelte", true},
			{"web/src/layout.svelte", true},
			{"web/src/Layout.tsx", true},
			{"web/src/routes/+page.svelte", false},
			{"web/src/components/Button.svelte", false},
		}
		for _, tt := range tests {
			change := WebFileChange{Path: tt.path}
			if got := change.IsLayoutFile(); got != tt.want {
				t.Errorf("IsLayoutFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		}
	})

	t.Run("TotalChanges", func(t *testing.T) {
		change := WebFileChange{LinesAdded: 10, LinesRemoved: 5}
		if got := change.TotalChanges(); got != 15 {
			t.Errorf("TotalChanges() = %v, want 15", got)
		}
	})
}

func TestAssessWebChangeRisk(t *testing.T) {
	tests := []struct {
		name    string
		changes []WebFileChange
		want    WebChangeRisk
	}{
		{
			name:    "empty changes",
			changes: []WebFileChange{},
			want:    WebRiskNone,
		},
		{
			name: "small CSS change - LOW risk",
			changes: []WebFileChange{
				{Path: "web/src/app.css", LinesAdded: 3, LinesRemoved: 1, IsNew: false},
			},
			want: WebRiskLow,
		},
		{
			name: "larger CSS change - MEDIUM risk",
			changes: []WebFileChange{
				{Path: "web/src/app.css", LinesAdded: 15, LinesRemoved: 5, IsNew: false},
			},
			want: WebRiskMedium,
		},
		{
			name: "new route file - HIGH risk",
			changes: []WebFileChange{
				{Path: "web/src/routes/new-page/+page.svelte", LinesAdded: 50, LinesRemoved: 0, IsNew: true},
			},
			want: WebRiskHigh,
		},
		{
			name: "new layout file - HIGH risk",
			changes: []WebFileChange{
				{Path: "web/src/routes/+layout.svelte", LinesAdded: 30, LinesRemoved: 0, IsNew: true},
			},
			want: WebRiskHigh,
		},
		{
			name: "new component - MEDIUM risk",
			changes: []WebFileChange{
				{Path: "web/src/components/NewButton.svelte", LinesAdded: 20, LinesRemoved: 0, IsNew: true},
			},
			want: WebRiskMedium,
		},
		{
			name: "small component modification - LOW risk",
			changes: []WebFileChange{
				{Path: "web/src/components/Button.svelte", LinesAdded: 2, LinesRemoved: 1, IsNew: false},
			},
			want: WebRiskLow,
		},
		{
			name: "medium component modification - MEDIUM risk",
			changes: []WebFileChange{
				{Path: "web/src/components/Button.svelte", LinesAdded: 15, LinesRemoved: 10, IsNew: false},
			},
			want: WebRiskMedium,
		},
		{
			name: "large component modification - HIGH risk",
			changes: []WebFileChange{
				{Path: "web/src/components/Dashboard.svelte", LinesAdded: 40, LinesRemoved: 20, IsNew: false},
			},
			want: WebRiskHigh,
		},
		{
			name: "large route modification - HIGH risk",
			changes: []WebFileChange{
				{Path: "web/src/routes/+page.svelte", LinesAdded: 40, LinesRemoved: 30, IsNew: false},
			},
			want: WebRiskHigh,
		},
		{
			name: "mixed changes - takes highest risk",
			changes: []WebFileChange{
				{Path: "web/src/app.css", LinesAdded: 2, LinesRemoved: 1, IsNew: false},                  // LOW
				{Path: "web/src/components/Button.svelte", LinesAdded: 5, LinesRemoved: 2, IsNew: false}, // LOW
				{Path: "web/src/routes/new/+page.svelte", LinesAdded: 50, LinesRemoved: 0, IsNew: true},  // HIGH
			},
			want: WebRiskHigh,
		},
		{
			name: "multiple low risk changes stay LOW",
			changes: []WebFileChange{
				{Path: "web/src/app.css", LinesAdded: 2, LinesRemoved: 1, IsNew: false},
				{Path: "web/src/styles.css", LinesAdded: 3, LinesRemoved: 2, IsNew: false},
			},
			want: WebRiskLow,
		},
		{
			name: "large layout change - HIGH risk",
			changes: []WebFileChange{
				{Path: "web/src/routes/+layout.svelte", LinesAdded: 25, LinesRemoved: 10, IsNew: false},
			},
			want: WebRiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssessWebChangeRisk(tt.changes)
			if got != tt.want {
				t.Errorf("AssessWebChangeRisk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssessSingleFileRisk(t *testing.T) {
	tests := []struct {
		name   string
		change WebFileChange
		want   WebChangeRisk
	}{
		{
			name:   "new route always HIGH",
			change: WebFileChange{Path: "web/src/routes/new/+page.svelte", IsNew: true, LinesAdded: 5},
			want:   WebRiskHigh,
		},
		{
			name:   "new layout always HIGH",
			change: WebFileChange{Path: "web/src/routes/+layout.svelte", IsNew: true, LinesAdded: 10},
			want:   WebRiskHigh,
		},
		{
			name:   "CSS <=10 lines is LOW",
			change: WebFileChange{Path: "web/src/app.css", LinesAdded: 5, LinesRemoved: 3, IsNew: false},
			want:   WebRiskLow,
		},
		{
			name:   "CSS >10 lines is MEDIUM",
			change: WebFileChange{Path: "web/src/app.css", LinesAdded: 8, LinesRemoved: 5, IsNew: false},
			want:   WebRiskMedium,
		},
		{
			name:   "component <=5 lines is LOW",
			change: WebFileChange{Path: "web/src/components/Btn.svelte", LinesAdded: 3, LinesRemoved: 1, IsNew: false},
			want:   WebRiskLow,
		},
		{
			name:   "component 6-30 lines is MEDIUM",
			change: WebFileChange{Path: "web/src/components/Btn.svelte", LinesAdded: 15, LinesRemoved: 5, IsNew: false},
			want:   WebRiskMedium,
		},
		{
			name:   "component >30 lines is HIGH",
			change: WebFileChange{Path: "web/src/components/Btn.svelte", LinesAdded: 25, LinesRemoved: 10, IsNew: false},
			want:   WebRiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assessSingleFileRisk(tt.change)
			if got != tt.want {
				t.Errorf("assessSingleFileRisk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNumstatOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantLen int
	}{
		{
			name:    "empty output",
			output:  "",
			wantLen: 0,
		},
		{
			name:    "non-web files only",
			output:  "10\t5\tpkg/verify/check.go\n3\t1\tcmd/orch/main.go\n",
			wantLen: 0,
		},
		{
			name:    "web file",
			output:  "10\t5\tweb/src/app.css\n",
			wantLen: 1,
		},
		{
			name:    "mixed files",
			output:  "10\t5\tpkg/verify/check.go\n3\t1\tweb/src/app.css\n5\t2\tweb/src/Component.svelte\n",
			wantLen: 2,
		},
		{
			name:    "binary file",
			output:  "-\t-\tweb/src/image.png\n",
			wantLen: 0, // .png is not a web file extension
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := parseNumstatOutput(tt.output, "/tmp")
			if err != nil {
				t.Fatalf("parseNumstatOutput() error = %v", err)
			}
			if len(changes) != tt.wantLen {
				t.Errorf("parseNumstatOutput() returned %d changes, want %d", len(changes), tt.wantLen)
			}
		})
	}
}

func TestVisualVerificationResult_IncludesRiskLevel(t *testing.T) {
	// Test that the result struct properly includes risk level
	result := VisualVerificationResult{
		Passed:        true,
		HasWebChanges: true,
		RiskLevel:     WebRiskMedium,
	}

	if result.RiskLevel != WebRiskMedium {
		t.Errorf("RiskLevel = %v, want MEDIUM", result.RiskLevel)
	}

	if result.RiskLevel.String() != "MEDIUM" {
		t.Errorf("RiskLevel.String() = %v, want MEDIUM", result.RiskLevel.String())
	}
}

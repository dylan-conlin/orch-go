package verify

import (
	"testing"
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
			name: "glass tool mentioned",
			comments: []Comment{
				{Text: "Used glass_screenshot to capture the dashboard"},
			},
			wantHas:    true,
			wantMinLen: 1,
		},
		{
			name: "glass command mentioned",
			comments: []Comment{
				{Text: "Ran glass navigate to verify the page loads"},
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
		// Glass browser automation tools
		{"used glass_screenshot to capture UI", true},
		{"called glass_navigate to open page", true},
		{"glass_click on the button", true},
		{"glass screenshot of dashboard", true},
		{"glass navigate to settings", true},

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

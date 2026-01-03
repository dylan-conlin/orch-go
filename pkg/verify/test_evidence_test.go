package verify

import (
	"testing"
	"time"
)

func TestIsSkillRequiringTestEvidence(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		want      bool
	}{
		// Skills requiring test evidence
		{"feature-impl requires", "feature-impl", true},
		{"systematic-debugging requires", "systematic-debugging", true},
		{"reliability-testing requires", "reliability-testing", true},

		// Skills excluded from test evidence
		{"investigation excluded", "investigation", false},
		{"architect excluded", "architect", false},
		{"research excluded", "research", false},
		{"design-session excluded", "design-session", false},
		{"codebase-audit excluded", "codebase-audit", false},
		{"issue-creation excluded", "issue-creation", false},
		{"writing-skills excluded", "writing-skills", false},

		// Edge cases
		{"empty skill", "", false},
		{"unknown skill", "unknown-skill", false},
		{"case insensitive", "Feature-Impl", true},
		{"case insensitive lower", "FEATURE-IMPL", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSkillRequiringTestEvidence(tt.skillName)
			if got != tt.want {
				t.Errorf("IsSkillRequiringTestEvidence(%q) = %v, want %v", tt.skillName, got, tt.want)
			}
		})
	}
}

func TestHasTestExecutionEvidence(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     bool
		wantLen  int // Expected minimum number of evidence items
	}{
		// Go test patterns
		{
			name: "go test PASS",
			comments: []Comment{
				{Text: "Tests: go test ./pkg/... - PASS"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "go test ok output",
			comments: []Comment{
				{Text: "ok  github.com/example/pkg  0.123s"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "go test with count",
			comments: []Comment{
				{Text: "Tests: go test ./... - PASS (12 tests in 0.8s)"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "go test --- PASS",
			comments: []Comment{
				{Text: "--- PASS: TestSomething (0.00s)"},
			},
			want:    true,
			wantLen: 1,
		},

		// npm/yarn/bun test patterns
		{
			name: "npm test passed",
			comments: []Comment{
				{Text: "npm test - passed"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "jest style output",
			comments: []Comment{
				{Text: "Tests: 15 passed, 0 failed"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "vitest style output",
			comments: []Comment{
				{Text: "15 passing, 0 failing"},
			},
			want:    true,
			wantLen: 1,
		},

		// pytest patterns
		{
			name: "pytest output",
			comments: []Comment{
				{Text: "pytest - 15 passed"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "pytest summary line",
			comments: []Comment{
				{Text: "======= 15 passed, 0 warnings ======="},
			},
			want:    true,
			wantLen: 1,
		},

		// cargo test patterns
		{
			name: "cargo test ok",
			comments: []Comment{
				{Text: "cargo test - ok"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "cargo test result",
			comments: []Comment{
				{Text: "test result: ok. 15 passed; 0 failed"},
			},
			want:    true,
			wantLen: 1,
		},

		// Generic patterns - require counts
		{
			name: "all N tests passed",
			comments: []Comment{
				{Text: "all 15 tests passed"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "N tests passed",
			comments: []Comment{
				{Text: "15 tests passed"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "ran tests in time",
			comments: []Comment{
				{Text: "ran 15 tests in 2.3s"},
			},
			want:    true,
			wantLen: 1,
		},
		{
			name: "playwright passed",
			comments: []Comment{
				{Text: "playwright test - 5 passed (2s)"},
			},
			want:    true,
			wantLen: 1,
		},

		// False positives (should NOT count as evidence)
		// These are vague claims without quantifiable output (counts, timing, etc.)
		{
			name: "vague claim - tests pass",
			comments: []Comment{
				{Text: "tests pass"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - tests passed",
			comments: []Comment{
				{Text: "tests passed"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - all tests pass",
			comments: []Comment{
				{Text: "all tests pass"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - all tests passed",
			comments: []Comment{
				{Text: "all tests passed"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - the tests pass",
			comments: []Comment{
				{Text: "the tests pass"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - verified tests pass",
			comments: []Comment{
				{Text: "verified tests pass"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - confirmed tests pass",
			comments: []Comment{
				{Text: "confirmed tests pass"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - tests passing",
			comments: []Comment{
				{Text: "tests passing"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - tests are passing",
			comments: []Comment{
				{Text: "tests are passing"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - tests succeeded",
			comments: []Comment{
				{Text: "tests succeeded"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim - tests completed successfully",
			comments: []Comment{
				{Text: "tests completed successfully"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "expectation - tests should pass",
			comments: []Comment{
				{Text: "tests should pass after this change"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "expectation - tests will pass",
			comments: []Comment{
				{Text: "tests will pass"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "vague claim in sentence - all tests pass after changes",
			comments: []Comment{
				{Text: "all tests pass after changes"},
			},
			want:    false,
			wantLen: 0,
		},

		// Edge cases
		{
			name:     "no comments",
			comments: []Comment{},
			want:     false,
			wantLen:  0,
		},
		{
			name: "unrelated comments",
			comments: []Comment{
				{Text: "Phase: Planning - Analyzing codebase"},
				{Text: "Phase: Implementing - Adding feature"},
			},
			want:    false,
			wantLen: 0,
		},
		{
			name: "mixed valid and invalid",
			comments: []Comment{
				{Text: "tests pass"},                        // Invalid (vague)
				{Text: "go test ./... - PASS (5 tests)"},    // Valid
			},
			want:    true,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, evidence := HasTestExecutionEvidence(tt.comments)
			if got != tt.want {
				t.Errorf("HasTestExecutionEvidence() = %v, want %v", got, tt.want)
			}
			if len(evidence) < tt.wantLen {
				t.Errorf("HasTestExecutionEvidence() evidence count = %d, want >= %d", len(evidence), tt.wantLen)
			}
		})
	}
}

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		// Code files
		{"go file", "pkg/verify/check.go", true},
		{"python file", "scripts/process.py", true},
		{"typescript file", "src/app.ts", true},
		{"javascript file", "lib/utils.js", true},
		{"rust file", "src/main.rs", true},
		{"svelte file", "web/src/App.svelte", true},

		// Test files (should NOT require tests)
		{"go test file", "pkg/verify/check_test.go", false},
		{"js test file", "src/app.test.js", false},
		{"js spec file", "src/app.spec.ts", false},
		{"python test file", "tests/test_utils.py", false},

		// Non-code files
		{"markdown", "README.md", false},
		{"yaml config", "config.yaml", false},
		{"json config", "package.json", false},
		{"text file", "notes.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCodeFile(tt.filePath)
			if got != tt.want {
				t.Errorf("isCodeFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestHasCodeChangesInFiles(t *testing.T) {
	tests := []struct {
		name      string
		gitOutput string
		want      bool
	}{
		{
			name:      "has go code changes",
			gitOutput: "pkg/verify/check.go\npkg/verify/test_evidence.go\n",
			want:      true,
		},
		{
			name:      "only config changes",
			gitOutput: "config.yaml\npackage.json\n",
			want:      false,
		},
		{
			name:      "only test changes",
			gitOutput: "pkg/verify/check_test.go\npkg/verify/test_evidence_test.go\n",
			want:      false,
		},
		{
			name:      "mixed code and config",
			gitOutput: "pkg/verify/check.go\nconfig.yaml\n",
			want:      true,
		},
		{
			name:      "empty output",
			gitOutput: "",
			want:      false,
		},
		{
			name:      "whitespace only",
			gitOutput: "   \n\n  \n",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCodeChangesInFiles(tt.gitOutput)
			if got != tt.want {
				t.Errorf("hasCodeChangesInFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestEvidencePatternMatching(t *testing.T) {
	// Additional tests for specific patterns to ensure regex correctness
	testCases := []struct {
		name    string
		comment string
		want    bool
	}{
		// Go patterns
		{"go test with dash", "go test ./... - PASS", true},
		{"go test with endash", "go test ./... – PASS", true},
		{"go test with emdash", "go test ./... — PASS", true},
		{"go test no dash", "go test ./... PASS", true},
		{"ok package timing", "ok  github.com/pkg/test  1.234s", true},
		{"PASS count", "PASS: 15", true},
		{"tests passed count", "15 tests passed", true},

		// npm/yarn patterns
		{"npm success", "npm test - success", true},
		{"yarn passed", "yarn test - passed", true},
		{"bun success", "bun test - success", true},
		{"Test Suites passed", "Test Suites: 5 passed, 0 failed", true},

		// pytest patterns
		{"pytest dashes", "========= 10 passed ==========", true},
		{"pytest with warnings", "10 passed, 2 warnings", true},

		// Edge cases that should not match
		{"partial match test", "testing something", false},
		{"partial pass", "passenger list", false},

		// Vague claims that should NOT match (false positives)
		// These are exactly what caused verification theater in the 4026cb69 case
		{"vague all tests pass", "all tests pass", false},
		{"vague all tests passed", "all tests passed", false},
		{"vague tests pass", "tests pass", false},
		{"vague tests passed", "tests passed", false},
		{"vague the tests pass", "the tests pass", false},
		{"vague tests passing", "tests passing", false},
		{"vague tests are passing", "tests are passing", false},
		{"vague verified tests pass", "verified tests pass", false},
		{"vague confirmed tests pass", "confirmed tests pass", false},
		{"vague tests succeeded", "tests succeeded", false},
		{"vague tests completed successfully", "tests completed successfully", false},
		{"vague tests will pass", "tests will pass", false},
		{"vague tests should pass", "tests should pass", false},

		// Claims with context but no counts (still vague)
		{"vague with context", "all tests pass after changes", false},
		{"vague with reason", "tests pass because I fixed the bug", false},

		// Valid patterns WITH counts
		{"valid all N tests pass", "all 15 tests pass", true},
		{"valid all N tests passed", "all 42 tests passed", true},
		{"valid N tests passed", "15 tests passed", true},
		{"valid ran N tests", "ran 10 tests in 1.5s", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			comments := []Comment{{Text: tc.comment}}
			got, _ := HasTestExecutionEvidence(comments)
			if got != tc.want {
				t.Errorf("Pattern test for %q: got %v, want %v", tc.comment, got, tc.want)
			}
		})
	}
}

func TestHasCodeChangesSinceSpawn(t *testing.T) {
	// Note: This test uses the actual git repo, so results depend on repo state.
	// The key behavior we're testing is the fallback logic and that it handles
	// zero time correctly.
	
	tests := []struct {
		name      string
		spawnTime time.Time
		desc      string
	}{
		{
			name:      "zero spawn time falls back to recent commits",
			spawnTime: time.Time{},
			desc:      "Should fall back to HasCodeChangesInRecentCommits",
		},
		{
			name:      "future spawn time returns false (no commits since)",
			spawnTime: time.Now().Add(24 * time.Hour),
			desc:      "No commits can exist since a future time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use current directory as project dir (this test file's repo)
			projectDir := "."
			result := HasCodeChangesSinceSpawn(projectDir, tt.spawnTime)
			
			// For future spawn time, we expect false (no commits since future)
			if tt.spawnTime.After(time.Now()) && result {
				t.Errorf("HasCodeChangesSinceSpawn with future time = true, want false")
			}
			// For zero time, we can't test exact result but it shouldn't panic
			t.Logf("%s: result=%v", tt.desc, result)
		})
	}
}

func TestMarkdownOnlyChangesScenario(t *testing.T) {
	// This test verifies the hasCodeChangesInFiles correctly identifies
	// markdown-only changes as NOT requiring test evidence
	tests := []struct {
		name      string
		gitOutput string
		want      bool
		desc      string
	}{
		{
			name:      "markdown only - single file",
			gitOutput: "README.md\n",
			want:      false,
			desc:      "Single markdown file should not require tests",
		},
		{
			name:      "markdown only - multiple files",
			gitOutput: "README.md\ndocs/DESIGN.md\n.kb/investigations/2025-01-01-test.md\n",
			want:      false,
			desc:      "Multiple markdown files should not require tests",
		},
		{
			name:      "markdown plus template files",
			gitOutput: "SKILL.md\npkg/claudemd/templates/SYNTHESIS.md\n",
			want:      false,
			desc:      "Markdown templates should not require tests",
		},
		{
			name:      "markdown with code file",
			gitOutput: "README.md\npkg/verify/check.go\n",
			want:      true,
			desc:      "Mixed markdown and code should require tests",
		},
		{
			name:      "only config files",
			gitOutput: "config.yaml\npackage.json\n.gitignore\n",
			want:      false,
			desc:      "Config files should not require tests",
		},
		{
			name:      "markdown and config only",
			gitOutput: "README.md\nconfig.yaml\n",
			want:      false,
			desc:      "Markdown and config together should not require tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCodeChangesInFiles(tt.gitOutput)
			if got != tt.want {
				t.Errorf("hasCodeChangesInFiles() = %v, want %v (%s)", got, tt.want, tt.desc)
			}
		})
	}
}

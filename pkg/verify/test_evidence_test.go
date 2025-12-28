package verify

import (
	"testing"
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

		// Generic patterns
		{
			name: "all tests passed",
			comments: []Comment{
				{Text: "all 15 tests passed"},
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
		{
			name: "vague claim - tests pass",
			comments: []Comment{
				{Text: "tests pass"},
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
			name: "expectation - tests should pass",
			comments: []Comment{
				{Text: "tests should pass after this change"},
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

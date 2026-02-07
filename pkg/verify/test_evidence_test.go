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
				{Text: "tests pass"},                     // Invalid (vague)
				{Text: "go test ./... - PASS (5 tests)"}, // Valid
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

func TestHasCodeChangesSinceSpawnForWorkspace(t *testing.T) {
	// This tests the workspace-filtered version that prevents concurrent agent
	// commits from triggering false positives

	t.Run("empty workspace path falls back to all commits", func(t *testing.T) {
		// With empty workspace, should behave like original HasCodeChangesSinceSpawn
		projectDir := "."
		spawnTime := time.Now().Add(24 * time.Hour) // Future time = no commits

		result := HasCodeChangesSinceSpawnForWorkspace(projectDir, spawnTime, "")
		if result {
			t.Error("Expected false for future spawn time with empty workspace")
		}
	})

	t.Run("non-existent workspace returns false", func(t *testing.T) {
		// If workspace doesn't exist in any commits, no code changes should be detected
		projectDir := "."
		spawnTime := time.Now().Add(-time.Hour) // Recent time
		nonExistentWorkspace := "/nonexistent/workspace/path"

		result := HasCodeChangesSinceSpawnForWorkspace(projectDir, spawnTime, nonExistentWorkspace)
		if result {
			t.Error("Expected false for non-existent workspace - no commits touch it")
		}
	})

	t.Run("zero spawn time falls back to recent commits", func(t *testing.T) {
		projectDir := "."
		zeroTime := time.Time{}

		// Should fall back to HasCodeChangesInRecentCommits, which may or may not
		// find code changes depending on repo state
		result := HasCodeChangesSinceSpawnForWorkspace(projectDir, zeroTime, "")
		t.Logf("Zero spawn time fallback result: %v", result)
		// Just verify it doesn't panic
	})
}

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{"lowercase md", "README.md", true},
		{"uppercase MD", "README.MD", true},
		{"mixed case Md", "README.Md", true},
		{"nested markdown", "docs/guide/DESIGN.md", true},
		{"kb investigation", ".kb/investigations/2025-01-01-test.md", true},
		{"synthesis", ".orch/workspace/test/SYNTHESIS.md", true},
		{"go file", "pkg/verify/check.go", false},
		{"markdown in name", "markdown-parser.go", false},
		{"no extension", "README", false},
		{"json file", "package.json", false},
		{"yaml file", "config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMarkdownFile(tt.filePath)
			if got != tt.want {
				t.Errorf("isMarkdownFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestIsFileOutsideProject(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		projectDir string
		want       bool
	}{
		// Relative paths
		{"relative in project", "pkg/verify/check.go", "/project", false},
		{"relative outside project", "../other/file.go", "/project", true},
		{"double parent", "../../file.go", "/project", true},

		// Absolute paths
		{"absolute in project", "/project/pkg/verify/check.go", "/project", false},
		{"absolute outside project", "/other/file.go", "/project", true},
		{"home directory skill", "/Users/user/.claude/skills/test/SKILL.md", "/project", true},
		{"kb in other project", "/Users/user/other/.kb/test.md", "/project", true},

		// Edge cases
		{"empty project dir", "pkg/file.go", "", false},
		{"project in relative", "project/file.go", "/project", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFileOutsideProject(tt.filePath, tt.projectDir)
			if got != tt.want {
				t.Errorf("isFileOutsideProject(%q, %q) = %v, want %v", tt.filePath, tt.projectDir, got, tt.want)
			}
		})
	}
}

func TestAreAllFilesMarkdown(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantAll   bool
		wantCount int
	}{
		{
			name:      "single markdown",
			files:     []string{"README.md"},
			wantAll:   true,
			wantCount: 1,
		},
		{
			name:      "multiple markdown",
			files:     []string{"README.md", "docs/DESIGN.md", ".kb/test.md"},
			wantAll:   true,
			wantCount: 3,
		},
		{
			name:      "mixed files",
			files:     []string{"README.md", "main.go"},
			wantAll:   false,
			wantCount: 2,
		},
		{
			name:      "no markdown",
			files:     []string{"main.go", "config.yaml"},
			wantAll:   false,
			wantCount: 2,
		},
		{
			name:      "empty list",
			files:     []string{},
			wantAll:   true,
			wantCount: 0,
		},
		{
			name:      "nil list",
			files:     nil,
			wantAll:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAll, gotCount := areAllFilesMarkdown(tt.files)
			if gotAll != tt.wantAll {
				t.Errorf("areAllFilesMarkdown() allMd = %v, want %v", gotAll, tt.wantAll)
			}
			if gotCount != tt.wantCount {
				t.Errorf("areAllFilesMarkdown() count = %d, want %d", gotCount, tt.wantCount)
			}
		})
	}
}

func TestAreAllFilesOutsideProject(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		projectDir string
		wantAll    bool
		wantCount  int
	}{
		{
			name:       "all outside (relative)",
			files:      []string{"../other/file.md", "../../test/file.go"},
			projectDir: "/project",
			wantAll:    true,
			wantCount:  2,
		},
		{
			name:       "all outside (absolute)",
			files:      []string{"/other/file.md", "/home/user/file.go"},
			projectDir: "/project",
			wantAll:    true,
			wantCount:  2,
		},
		{
			name:       "mixed inside and outside",
			files:      []string{"pkg/file.go", "../other/file.md"},
			projectDir: "/project",
			wantAll:    false,
			wantCount:  2,
		},
		{
			name:       "all inside",
			files:      []string{"pkg/file.go", "cmd/main.go"},
			projectDir: "/project",
			wantAll:    false,
			wantCount:  2,
		},
		{
			name:       "empty list",
			files:      []string{},
			projectDir: "/project",
			wantAll:    true,
			wantCount:  0,
		},
		{
			name:       "empty project dir",
			files:      []string{"../file.go"},
			projectDir: "",
			wantAll:    true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAll, gotCount := areAllFilesOutsideProject(tt.files, tt.projectDir)
			if gotAll != tt.wantAll {
				t.Errorf("areAllFilesOutsideProject() allOutside = %v, want %v", gotAll, tt.wantAll)
			}
			if gotCount != tt.wantCount {
				t.Errorf("areAllFilesOutsideProject() count = %d, want %d", gotCount, tt.wantCount)
			}
		})
	}
}

func TestParseFileList(t *testing.T) {
	tests := []struct {
		name      string
		gitOutput string
		want      []string
	}{
		{
			name:      "simple list",
			gitOutput: "file1.go\nfile2.go\n",
			want:      []string{"file1.go", "file2.go"},
		},
		{
			name:      "with whitespace",
			gitOutput: "  file1.go  \n\n  file2.go\n\n",
			want:      []string{"file1.go", "file2.go"},
		},
		{
			name:      "empty output",
			gitOutput: "",
			want:      nil,
		},
		{
			name:      "whitespace only",
			gitOutput: "  \n\n  \n",
			want:      nil,
		},
		{
			name:      "paths with directories",
			gitOutput: "pkg/verify/check.go\ncmd/orch/main.go\n",
			want:      []string{"pkg/verify/check.go", "cmd/orch/main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFileList(tt.gitOutput)
			if len(got) != len(tt.want) {
				t.Errorf("parseFileList() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseFileList()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestTestEvidenceResultExemptFields(t *testing.T) {
	// Test that the exemption fields are properly tracked in TestEvidenceResult
	result := TestEvidenceResult{
		Passed:               true,
		MarkdownOnlyExempt:   true,
		OutsideProjectExempt: false,
	}

	if !result.MarkdownOnlyExempt {
		t.Error("MarkdownOnlyExempt should be true")
	}
	if result.OutsideProjectExempt {
		t.Error("OutsideProjectExempt should be false")
	}

	// Test the other way
	result2 := TestEvidenceResult{
		Passed:               true,
		MarkdownOnlyExempt:   false,
		OutsideProjectExempt: true,
	}

	if result2.MarkdownOnlyExempt {
		t.Error("MarkdownOnlyExempt should be false")
	}
	if !result2.OutsideProjectExempt {
		t.Error("OutsideProjectExempt should be true")
	}
}

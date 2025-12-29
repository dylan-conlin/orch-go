// changelog_test.go - Tests for changelog command
package main

import (
	"testing"
	"time"
)

func TestCategorizeCommitByFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "skills directory",
			files:    []string{"skills/worker/feature-impl/SKILL.md"},
			expected: "skills",
		},
		{
			name:     "nested skills",
			files:    []string{"foo/skills/bar/test.md"},
			expected: "skills",
		},
		{
			name:     "kb directory",
			files:    []string{".kb/investigations/test.md"},
			expected: "kb",
		},
		{
			name:     "cmd directory",
			files:    []string{"cmd/orch/main.go", "cmd/orch/changelog.go"},
			expected: "cmd",
		},
		{
			name:     "pkg directory",
			files:    []string{"pkg/spawn/config.go"},
			expected: "pkg",
		},
		{
			name:     "web directory",
			files:    []string{"web/src/routes/+page.svelte"},
			expected: "web",
		},
		{
			name:     "src directory (web)",
			files:    []string{"src/components/Button.tsx"},
			expected: "web",
		},
		{
			name:     "docs directory",
			files:    []string{"docs/README.md"},
			expected: "docs",
		},
		{
			name:     "config files",
			files:    []string{"go.mod", "package.json", "Makefile"},
			expected: "config",
		},
		{
			name:     "yaml config",
			files:    []string{".orch/config.yaml"},
			expected: "config",
		},
		{
			name:     "mixed - cmd wins",
			files:    []string{"cmd/orch/main.go", "README.md"},
			expected: "cmd",
		},
		{
			name:     "other",
			files:    []string{"README.md", "LICENSE"},
			expected: "other",
		},
		{
			name:     "empty files",
			files:    []string{},
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeCommitByFiles(tt.files)
			if result != tt.expected {
				t.Errorf("categorizeCommitByFiles(%v) = %q, want %q", tt.files, result, tt.expected)
			}
		})
	}
}

func TestParseGitLog(t *testing.T) {
	// Sample git log output with format: hash|subject|author|date (plus file names)
	output := `abc12345|feat: add changelog command|Dylan Conlin|2025-12-29T10:00:00-08:00
cmd/orch/changelog.go
cmd/orch/changelog_test.go

def67890|fix: handle missing repos|Dylan Conlin|2025-12-29T09:00:00-08:00
pkg/spawn/ecosystem.go`

	commits, err := parseGitLog(output, "orch-go")
	if err != nil {
		t.Fatalf("parseGitLog failed: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(commits))
	}

	// Check first commit
	if commits[0].Hash != "abc12345" {
		t.Errorf("first commit hash = %q, want %q", commits[0].Hash, "abc12345")
	}
	if commits[0].Subject != "feat: add changelog command" {
		t.Errorf("first commit subject = %q, want %q", commits[0].Subject, "feat: add changelog command")
	}
	if commits[0].Author != "Dylan Conlin" {
		t.Errorf("first commit author = %q, want %q", commits[0].Author, "Dylan Conlin")
	}
	if commits[0].Repo != "orch-go" {
		t.Errorf("first commit repo = %q, want %q", commits[0].Repo, "orch-go")
	}
	if len(commits[0].Files) != 2 {
		t.Errorf("first commit files count = %d, want %d", len(commits[0].Files), 2)
	}
	if commits[0].Category != "cmd" {
		t.Errorf("first commit category = %q, want %q", commits[0].Category, "cmd")
	}

	// Check second commit
	if commits[1].Hash != "def67890" {
		t.Errorf("second commit hash = %q, want %q", commits[1].Hash, "def67890")
	}
	if commits[1].Category != "pkg" {
		t.Errorf("second commit category = %q, want %q", commits[1].Category, "pkg")
	}
}

func TestParseGitLogEmpty(t *testing.T) {
	commits, err := parseGitLog("", "orch-go")
	if err != nil {
		t.Fatalf("parseGitLog failed on empty input: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for empty input, got %d", len(commits))
	}
}

func TestParseGitLogWhitespace(t *testing.T) {
	commits, err := parseGitLog("  \n\n  \n", "orch-go")
	if err != nil {
		t.Fatalf("parseGitLog failed on whitespace input: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for whitespace input, got %d", len(commits))
	}
}

func TestGetCategoryIcon(t *testing.T) {
	tests := []struct {
		category string
		expected string
	}{
		{"skills", "🎯"},
		{"kb", "📚"},
		{"cmd", "⚡"},
		{"pkg", "📦"},
		{"web", "🌐"},
		{"docs", "📝"},
		{"config", "⚙️"},
		{"other", "📄"},
		{"unknown", "📄"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := getCategoryIcon(tt.category)
			if result != tt.expected {
				t.Errorf("getCategoryIcon(%q) = %q, want %q", tt.category, result, tt.expected)
			}
		})
	}
}

func TestCommitInfoDateStr(t *testing.T) {
	// Verify date string format
	commit := CommitInfo{
		Hash:    "abc12345",
		Subject: "test commit",
		Date:    time.Date(2025, 12, 29, 10, 0, 0, 0, time.UTC),
		DateStr: "2025-12-29",
	}

	if commit.DateStr != "2025-12-29" {
		t.Errorf("commit.DateStr = %q, want %q", commit.DateStr, "2025-12-29")
	}
}

func TestChangelogResultStructure(t *testing.T) {
	// Test that ChangelogResult has expected fields
	result := ChangelogResult{
		DateRange: DateRange{
			Start: "2025-12-22",
			End:   "2025-12-29",
		},
		TotalCommits: 100,
		RepoCount:    5,
		MissingRepos: []string{"missing-repo"},
		CommitsByDate: map[string][]CommitInfo{
			"2025-12-29": {
				{Hash: "abc", Subject: "test", Repo: "orch-go", Category: "cmd"},
			},
		},
		CommitsByCategory: map[string]int{
			"cmd": 50,
			"pkg": 30,
		},
		RepoStats: map[string]int{
			"orch-go": 80,
			"kb-cli":  20,
		},
	}

	if result.TotalCommits != 100 {
		t.Errorf("TotalCommits = %d, want 100", result.TotalCommits)
	}
	if len(result.MissingRepos) != 1 {
		t.Errorf("MissingRepos count = %d, want 1", len(result.MissingRepos))
	}
	if result.CommitsByCategory["cmd"] != 50 {
		t.Errorf("CommitsByCategory[cmd] = %d, want 50", result.CommitsByCategory["cmd"])
	}
}

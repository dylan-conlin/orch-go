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
		{"skill-behavioral", "🎯"},
		{"skill-docs", "📖"},
		{"kb", "📚"},
		{"decision-record", "📜"},
		{"investigation", "🔍"},
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

// Tests for semantic parsing

func TestParseConventionalCommit(t *testing.T) {
	tests := []struct {
		name       string
		subject    string
		wantType   string
		wantBreak  bool
	}{
		{
			name:      "simple feat",
			subject:   "feat: add new feature",
			wantType:  "feat",
			wantBreak: false,
		},
		{
			name:      "fix with scope",
			subject:   "fix(auth): resolve login issue",
			wantType:  "fix",
			wantBreak: false,
		},
		{
			name:      "docs",
			subject:   "docs: update README",
			wantType:  "docs",
			wantBreak: false,
		},
		{
			name:      "breaking with exclamation",
			subject:   "feat!: breaking API change",
			wantType:  "feat",
			wantBreak: true,
		},
		{
			name:      "breaking with BREAKING prefix",
			subject:   "BREAKING: remove deprecated API",
			wantType:  "",
			wantBreak: true,
		},
		{
			name:      "breaking in message",
			subject:   "feat: change API BREAKING CHANGE",
			wantType:  "feat",
			wantBreak: true,
		},
		{
			name:      "refactor",
			subject:   "refactor: cleanup code",
			wantType:  "refactor",
			wantBreak: false,
		},
		{
			name:      "chore",
			subject:   "chore: update dependencies",
			wantType:  "chore",
			wantBreak: false,
		},
		{
			name:      "no conventional format",
			subject:   "Update readme file",
			wantType:  "",
			wantBreak: false,
		},
		{
			name:      "test commit",
			subject:   "test: add unit tests",
			wantType:  "test",
			wantBreak: false,
		},
		{
			name:      "perf commit",
			subject:   "perf: optimize query",
			wantType:  "perf",
			wantBreak: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotBreak := parseConventionalCommit(tt.subject)
			if gotType != tt.wantType {
				t.Errorf("parseConventionalCommit(%q) type = %q, want %q", tt.subject, gotType, tt.wantType)
			}
			if gotBreak != tt.wantBreak {
				t.Errorf("parseConventionalCommit(%q) breaking = %v, want %v", tt.subject, gotBreak, tt.wantBreak)
			}
		})
	}
}

func TestInferChangeType(t *testing.T) {
	tests := []struct {
		name       string
		commitType string
		files      []string
		want       ChangeType
	}{
		{
			name:       "docs commit type",
			commitType: "docs",
			files:      []string{"README.md"},
			want:       ChangeTypeDocumentation,
		},
		{
			name:       "feat commit type",
			commitType: "feat",
			files:      []string{"cmd/main.go"},
			want:       ChangeTypeBehavioral,
		},
		{
			name:       "fix commit type",
			commitType: "fix",
			files:      []string{"pkg/auth/auth.go"},
			want:       ChangeTypeBehavioral,
		},
		{
			name:       "chore commit type",
			commitType: "chore",
			files:      []string{"Makefile"},
			want:       ChangeTypeStructural,
		},
		{
			name:       "infer from markdown files",
			commitType: "",
			files:      []string{"docs/guide.md", "README.md"},
			want:       ChangeTypeDocumentation,
		},
		{
			name:       "infer from go files",
			commitType: "",
			files:      []string{"main.go", "pkg/util/util.go"},
			want:       ChangeTypeBehavioral,
		},
		{
			name:       "infer from config files",
			commitType: "",
			files:      []string{"config.yaml", "go.mod"},
			want:       ChangeTypeStructural,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferChangeType(tt.commitType, tt.files)
			if got != tt.want {
				t.Errorf("inferChangeType(%q, %v) = %q, want %q", tt.commitType, tt.files, got, tt.want)
			}
		})
	}
}

func TestInferBlastRadius(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  BlastRadius
	}{
		{
			name:  "single file local",
			files: []string{"main.go"},
			want:  BlastRadiusLocal,
		},
		{
			name:  "single skill local",
			files: []string{"skills/worker/feature-impl/SKILL.md"},
			want:  BlastRadiusLocal,
		},
		{
			name: "multiple skills cross-skill",
			files: []string{
				"skills/worker/feature-impl/SKILL.md",
				"skills/worker/investigation/SKILL.md",
			},
			want: BlastRadiusCrossSkill,
		},
		{
			name:  "spawn system infrastructure",
			files: []string{"pkg/spawn/context.go"},
			want:  BlastRadiusInfrastructure,
		},
		{
			name:  "skill.yaml infrastructure",
			files: []string{"skills/worker/feature-impl/.skillc/skill.yaml"},
			want:  BlastRadiusInfrastructure,
		},
		{
			name:  "verify package infrastructure",
			files: []string{"pkg/verify/skill_outputs.go"},
			want:  BlastRadiusInfrastructure,
		},
		{
			name:  "SPAWN_CONTEXT infrastructure",
			files: []string{"templates/SPAWN_CONTEXT.md"},
			want:  BlastRadiusInfrastructure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferBlastRadius(tt.files)
			if got != tt.want {
				t.Errorf("inferBlastRadius(%v) = %q, want %q", tt.files, got, tt.want)
			}
		})
	}
}

func TestInferSemanticCategory(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "decision record",
			files: []string{".kb/decisions/2025-12-30-some-decision.md"},
			want:  "decision-record",
		},
		{
			name:  "investigation",
			files: []string{".kb/investigations/2025-12-30-inv-something.md"},
			want:  "investigation",
		},
		{
			name:  "skill behavioral",
			files: []string{"skills/worker/feature-impl/SKILL.md", "skills/worker/feature-impl/config.go"},
			want:  "skill-behavioral",
		},
		{
			name:  "skill docs only",
			files: []string{"skills/worker/feature-impl/README.md", "skills/worker/feature-impl/SKILL.md"},
			want:  "skill-docs",
		},
		{
			name:  "regular files",
			files: []string{"cmd/main.go", "pkg/util.go"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferSemanticCategory(tt.files)
			if got != tt.want {
				t.Errorf("inferSemanticCategory(%v) = %q, want %q", tt.files, got, tt.want)
			}
		})
	}
}

func TestGenerateSemanticLabel(t *testing.T) {
	tests := []struct {
		name string
		info SemanticInfo
		want string
	}{
		{
			name: "breaking behavioral",
			info: SemanticInfo{
				ChangeType:  ChangeTypeBehavioral,
				BlastRadius: BlastRadiusLocal,
				IsBreaking:  true,
			},
			want: "[BREAKING | behavioral]",
		},
		{
			name: "docs local",
			info: SemanticInfo{
				ChangeType:  ChangeTypeDocumentation,
				BlastRadius: BlastRadiusLocal,
				IsBreaking:  false,
			},
			want: "[docs]",
		},
		{
			name: "structural cross-skill",
			info: SemanticInfo{
				ChangeType:  ChangeTypeStructural,
				BlastRadius: BlastRadiusCrossSkill,
				IsBreaking:  false,
			},
			want: "[structural | cross-skill]",
		},
		{
			name: "behavioral infrastructure",
			info: SemanticInfo{
				ChangeType:  ChangeTypeBehavioral,
				BlastRadius: BlastRadiusInfrastructure,
				IsBreaking:  false,
			},
			want: "[behavioral | infrastructure]",
		},
		{
			name: "unknown local",
			info: SemanticInfo{
				ChangeType:  ChangeTypeUnknown,
				BlastRadius: BlastRadiusLocal,
				IsBreaking:  false,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSemanticLabel(tt.info)
			if got != tt.want {
				t.Errorf("generateSemanticLabel(%+v) = %q, want %q", tt.info, got, tt.want)
			}
		})
	}
}

func TestParseSemanticInfo(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		files   []string
		wantType    ChangeType
		wantRadius  BlastRadius
		wantBreak   bool
	}{
		{
			name:       "feat with behavioral files",
			subject:    "feat: add new feature",
			files:      []string{"cmd/main.go"},
			wantType:   ChangeTypeBehavioral,
			wantRadius: BlastRadiusLocal,
			wantBreak:  false,
		},
		{
			name:       "breaking change",
			subject:    "feat!: breaking API change",
			files:      []string{"pkg/api/api.go"},
			wantType:   ChangeTypeBehavioral,
			wantRadius: BlastRadiusLocal,
			wantBreak:  true,
		},
		{
			name:       "docs with markdown",
			subject:    "docs: update documentation",
			files:      []string{"docs/README.md", "docs/guide.md"},
			wantType:   ChangeTypeDocumentation,
			wantRadius: BlastRadiusLocal,
			wantBreak:  false,
		},
		{
			name:       "infrastructure change",
			subject:    "refactor: update spawn system",
			files:      []string{"pkg/spawn/context.go", "pkg/spawn/config.go"},
			wantType:   ChangeTypeBehavioral,
			wantRadius: BlastRadiusInfrastructure,
			wantBreak:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSemanticInfo(tt.subject, tt.files)
			if got.ChangeType != tt.wantType {
				t.Errorf("parseSemanticInfo() ChangeType = %q, want %q", got.ChangeType, tt.wantType)
			}
			if got.BlastRadius != tt.wantRadius {
				t.Errorf("parseSemanticInfo() BlastRadius = %q, want %q", got.BlastRadius, tt.wantRadius)
			}
			if got.IsBreaking != tt.wantBreak {
				t.Errorf("parseSemanticInfo() IsBreaking = %v, want %v", got.IsBreaking, tt.wantBreak)
			}
		})
	}
}

func TestParseGitLogWithSemanticInfo(t *testing.T) {
	// Test that parseGitLog populates SemanticInfo
	output := `abc12345|feat!: breaking API change|Dylan Conlin|2025-12-29T10:00:00-08:00
cmd/orch/api.go

def67890|docs: update README|Dylan Conlin|2025-12-29T09:00:00-08:00
README.md`

	commits, err := parseGitLog(output, "orch-go")
	if err != nil {
		t.Fatalf("parseGitLog failed: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(commits))
	}

	// Check first commit (breaking)
	if !commits[0].SemanticInfo.IsBreaking {
		t.Errorf("first commit should be marked as breaking")
	}
	if commits[0].SemanticInfo.CommitType != "feat" {
		t.Errorf("first commit type = %q, want %q", commits[0].SemanticInfo.CommitType, "feat")
	}

	// Check second commit (docs)
	if commits[1].SemanticInfo.ChangeType != ChangeTypeDocumentation {
		t.Errorf("second commit change type = %q, want %q", commits[1].SemanticInfo.ChangeType, ChangeTypeDocumentation)
	}
}

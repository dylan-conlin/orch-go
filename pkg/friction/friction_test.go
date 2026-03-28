package friction

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCategory(t *testing.T) {
	tests := []struct {
		input   string
		wantCat string
		wantMsg string
	}{
		{"tooling: git stash failed", "tooling", "git stash failed"},
		{"ceremony: Re-probing a claim", "ceremony", "Re-probing a claim"},
		{"bug: autorebuild reverted edits", "bug", "autorebuild reverted edits"},
		{"gap: OpenCode not running", "gap", "OpenCode not running"},
		{"Tooling: upper case", "tooling", "upper case"},
		{"no colon here", "unknown", "no colon here"},
	}
	for _, tt := range tests {
		cat, msg := parseCategory(tt.input)
		if cat != tt.wantCat {
			t.Errorf("parseCategory(%q) cat = %q, want %q", tt.input, cat, tt.wantCat)
		}
		if msg != tt.wantMsg {
			t.Errorf("parseCategory(%q) msg = %q, want %q", tt.input, msg, tt.wantMsg)
		}
	}
}

func TestExtractSkill(t *testing.T) {
	tests := []struct {
		assignee string
		want     string
	}{
		{"og-feat-something-27mar-abc1", "feat"},
		{"og-debug-fix-stuff-20mar-1234", "debug"},
		{"og-inv-investigate-thing-22mar-5678", "inv"},
		{"og-arch-design-thing-26mar-9abc", "arch"},
		{"og-research-topic-26mar-def0", "research"},
		{"og-work-task-20mar-1111", "work"},
		{"", "unknown"},
		{"singleword", "unknown"},
	}
	for _, tt := range tests {
		got := extractSkill(tt.assignee)
		if got != tt.want {
			t.Errorf("extractSkill(%q) = %q, want %q", tt.assignee, got, tt.want)
		}
	}
}

func TestParseJSONL(t *testing.T) {
	jsonl := `{"id":"test-1","assignee":"og-feat-test-28mar-aaaa","comments":[{"text":"Phase: Planning","created_at":"2026-03-28T10:00:00-07:00"},{"text":"Friction: tooling: git stash broke","created_at":"2026-03-28T10:05:00-07:00"},{"text":"Friction: none","created_at":"2026-03-28T10:10:00-07:00"}]}
{"id":"test-2","assignee":"og-debug-fix-28mar-bbbb","comments":[{"text":"Friction: bug: compile error","created_at":"2026-03-28T11:00:00-07:00"},{"text":"Friction: none","created_at":"2026-03-28T11:05:00-07:00"}]}
{"id":"test-3","assignee":"","comments":[{"text":"Friction: none","created_at":"2026-03-28T12:00:00-07:00"}]}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "issues.jsonl")
	if err := os.WriteFile(path, []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	entries, noneCount, err := ParseJSONL(path, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Errorf("got %d friction entries, want 2", len(entries))
	}
	if noneCount != 3 {
		t.Errorf("got noneCount=%d, want 3", noneCount)
	}

	// Check first entry
	if entries[0].Category != "tooling" {
		t.Errorf("entries[0].Category = %q, want tooling", entries[0].Category)
	}
	if entries[0].Skill != "feat" {
		t.Errorf("entries[0].Skill = %q, want feat", entries[0].Skill)
	}
	if entries[0].IssueID != "test-1" {
		t.Errorf("entries[0].IssueID = %q, want test-1", entries[0].IssueID)
	}

	// Check second entry
	if entries[1].Category != "bug" {
		t.Errorf("entries[1].Category = %q, want bug", entries[1].Category)
	}
	if entries[1].Skill != "debug" {
		t.Errorf("entries[1].Skill = %q, want debug", entries[1].Skill)
	}
}

func TestParseJSONLWithSince(t *testing.T) {
	jsonl := `{"id":"old","assignee":"og-feat-old-01jan-0001","comments":[{"text":"Friction: tooling: old issue","created_at":"2026-01-01T10:00:00-07:00"}]}
{"id":"new","assignee":"og-feat-new-28mar-0002","comments":[{"text":"Friction: bug: new issue","created_at":"2026-03-28T10:00:00-07:00"}]}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "issues.jsonl")
	if err := os.WriteFile(path, []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	since, _ := time.Parse(time.RFC3339, "2026-03-01T00:00:00-07:00")
	entries, _, err := ParseJSONL(path, since)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1 (only recent)", len(entries))
	}
	if len(entries) > 0 && entries[0].IssueID != "new" {
		t.Errorf("expected new issue, got %q", entries[0].IssueID)
	}
}

func TestAggregate(t *testing.T) {
	entries := []Entry{
		{IssueID: "a", Skill: "feat", Category: "tooling", Message: "git stash broke", CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)},
		{IssueID: "b", Skill: "feat", Category: "tooling", Message: "hook deleted file", CreatedAt: time.Date(2026, 3, 20, 11, 0, 0, 0, time.UTC)},
		{IssueID: "c", Skill: "debug", Category: "ceremony", Message: "governance blocked edit", CreatedAt: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)},
		{IssueID: "d", Skill: "inv", Category: "bug", Message: "compile error from other agent", CreatedAt: time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)},
	}
	noneCount := 10

	report := Aggregate(entries, noneCount, 7)

	if report.FrictionCount != 4 {
		t.Errorf("FrictionCount = %d, want 4", report.FrictionCount)
	}
	if report.NoneCount != 10 {
		t.Errorf("NoneCount = %d, want 10", report.NoneCount)
	}
	if report.TotalComments != 14 {
		t.Errorf("TotalComments = %d, want 14", report.TotalComments)
	}
	if report.TotalIssues != 4 {
		t.Errorf("TotalIssues = %d, want 4", report.TotalIssues)
	}

	// Categories should be sorted by count
	if len(report.Categories) < 1 {
		t.Fatal("no categories")
	}
	if report.Categories[0].Category != "tooling" {
		t.Errorf("top category = %q, want tooling", report.Categories[0].Category)
	}
	if report.Categories[0].Count != 2 {
		t.Errorf("tooling count = %d, want 2", report.Categories[0].Count)
	}
}

func TestParseJSONLFull(t *testing.T) {
	jsonl := `{"id":"test-1","assignee":"og-feat-test-28mar-aaaa","comments":[{"text":"Friction: tooling: git broke","created_at":"2026-03-28T10:00:00-07:00"},{"text":"Friction: none","created_at":"2026-03-28T10:10:00-07:00"}]}
{"id":"test-2","assignee":"og-debug-fix-28mar-bbbb","comments":[{"text":"Friction: none","created_at":"2026-03-28T11:00:00-07:00"}]}
{"id":"test-3","assignee":"og-inv-thing-28mar-cccc","comments":[{"text":"Friction: bug: stale files","created_at":"2026-03-28T12:00:00-07:00"},{"text":"Friction: none","created_at":"2026-03-28T12:05:00-07:00"}]}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "issues.jsonl")
	if err := os.WriteFile(path, []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	entries, noneBySkill, err := ParseJSONLFull(path, time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Errorf("got %d friction entries, want 2", len(entries))
	}
	if noneBySkill["feat"] != 1 {
		t.Errorf("feat none count = %d, want 1", noneBySkill["feat"])
	}
	if noneBySkill["debug"] != 1 {
		t.Errorf("debug none count = %d, want 1", noneBySkill["debug"])
	}
	if noneBySkill["inv"] != 1 {
		t.Errorf("inv none count = %d, want 1", noneBySkill["inv"])
	}

	rates := ComputeSkillRatesWithNone(entries, noneBySkill)
	// feat: 1 friction, 1 none = 50%
	// debug: 0 friction, 1 none = 0%
	// inv: 1 friction, 1 none = 50%
	for _, r := range rates {
		switch r.Skill {
		case "feat":
			if r.Rate < 0.49 || r.Rate > 0.51 {
				t.Errorf("feat rate = %.2f, want ~0.50", r.Rate)
			}
		case "debug":
			if r.Rate != 0 {
				t.Errorf("debug rate = %.2f, want 0", r.Rate)
			}
		case "inv":
			if r.Rate < 0.49 || r.Rate > 0.51 {
				t.Errorf("inv rate = %.2f, want ~0.50", r.Rate)
			}
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("this is a long string that exceeds limit", 20); len(got) != 20 {
		t.Errorf("truncate long len = %d, want 20", len(got))
	}
}

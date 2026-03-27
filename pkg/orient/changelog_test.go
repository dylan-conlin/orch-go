package orient

import (
	"strings"
	"testing"
)

func TestParseGitLog(t *testing.T) {
	// git log --format="%h|%s" output
	input := `abc1234|feat: add orient command
def5678|fix: spawn timeout handling
111aaaa|refactor: extract model package`

	entries := ParseGitLog(input, 10)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Hash != "abc1234" {
		t.Errorf("expected hash abc1234, got %q", entries[0].Hash)
	}
	if entries[0].Subject != "feat: add orient command" {
		t.Errorf("expected subject 'feat: add orient command', got %q", entries[0].Subject)
	}
	if entries[2].Hash != "111aaaa" {
		t.Errorf("expected hash 111aaaa, got %q", entries[2].Hash)
	}
}

func TestParseGitLog_Empty(t *testing.T) {
	entries := ParseGitLog("", 10)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseGitLog_Limit(t *testing.T) {
	input := `aaa|commit 1
bbb|commit 2
ccc|commit 3
ddd|commit 4
eee|commit 5`

	entries := ParseGitLog(input, 3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (limit), got %d", len(entries))
	}
	if entries[2].Hash != "ccc" {
		t.Errorf("expected third entry hash 'ccc', got %q", entries[2].Hash)
	}
}

func TestParseGitLog_MalformedLines(t *testing.T) {
	input := `aaa|good commit
bad line without separator
bbb|another good commit`

	entries := ParseGitLog(input, 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (skip malformed), got %d", len(entries))
	}
}

func TestFormatChangelog(t *testing.T) {
	entries := []ChangelogEntry{
		{Hash: "abc1234", Subject: "feat: add orient command"},
		{Hash: "def5678", Subject: "fix: spawn timeout handling"},
	}

	output := FormatChangelog(entries, "2026-03-04")

	if !strings.Contains(output, "Changelog (since 2026-03-04):") {
		t.Error("missing changelog header with date")
	}
	if !strings.Contains(output, "abc1234") {
		t.Error("missing first commit hash")
	}
	if !strings.Contains(output, "feat: add orient command") {
		t.Error("missing first commit subject")
	}
	if !strings.Contains(output, "def5678") {
		t.Error("missing second commit hash")
	}
}

func TestFormatChangelog_Empty(t *testing.T) {
	output := FormatChangelog(nil, "2026-03-04")
	if output != "" {
		t.Errorf("expected empty string for nil entries, got %q", output)
	}
}

func TestFormatChangelog_NoDate(t *testing.T) {
	entries := []ChangelogEntry{
		{Hash: "abc1234", Subject: "feat: something"},
	}
	output := FormatChangelog(entries, "")
	if !strings.Contains(output, "Changelog (recent):") {
		t.Error("missing fallback header when no date")
	}
}

func TestFormatHealth_WithChangelog(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
		Changelog: []ChangelogEntry{
			{Hash: "abc1234", Subject: "feat: add feature"},
		},
		PreviousSession: &DebriefSummary{Date: "2026-03-04"},
	}

	output := FormatHealth(data)

	if !strings.Contains(output, "Changelog") {
		t.Error("missing changelog section in health output")
	}
	if !strings.Contains(output, "abc1234") {
		t.Error("missing commit hash in health output")
	}
}

func TestFormatHealth_NoChangelog(t *testing.T) {
	data := &OrientationData{
		Throughput: Throughput{Days: 1},
	}

	output := FormatHealth(data)

	if strings.Contains(output, "Changelog") {
		t.Error("changelog section should not appear when empty")
	}
}

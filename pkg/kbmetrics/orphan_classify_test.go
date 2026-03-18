package kbmetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyOrphan_Empty(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "inv-empty.md")

	// Template-only file
	content := `<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

# Investigation: Empty Template

**Status:** In Progress
**Created:** 2026-01-15
**Beads ID:** orch-go-xxxxx

## What I Tried

- [ ] Step 1
- [ ] Step 2

## What I Observed

## Next Steps
`
	os.WriteFile(f, []byte(content), 0644)

	cat := ClassifyOrphan(f)
	if cat != CategoryEmpty {
		t.Errorf("ClassifyOrphan(empty template) = %q, want %q", cat, CategoryEmpty)
	}
}

func TestClassifyOrphan_NegativeResult(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "inv-negative.md")

	content := `# Investigation: Check if SSE parsing is broken

**Status:** Complete
**TLDR:** Already fixed in commit abc123. Not a bug — works as expected.

## What I Tried

1. Reproduced the SSE stream
2. Verified event parsing

## What I Observed

The SSE parsing works correctly. The issue was already resolved in a prior commit.
No action needed.

## Conclusion

Already fixed. No change required.
`
	os.WriteFile(f, []byte(content), 0644)

	cat := ClassifyOrphan(f)
	if cat != CategoryNegativeResult {
		t.Errorf("ClassifyOrphan(negative result) = %q, want %q", cat, CategoryNegativeResult)
	}
}

func TestClassifyOrphan_NegativeResult_HypothesisRejected(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "inv-hypothesis.md")

	content := `# Investigation: Memory leak in daemon

**Status:** Complete

## What I Tried

Profiled the daemon over 24h with pprof.
Set up goroutine tracking and heap snapshots every hour.
Monitored RSS growth vs Go heap growth independently.

## What I Observed

No memory leak detected. Hypothesis disproven — the goroutine count is stable.
The growth we saw was the OS page cache, not the Go heap.
RSS grew from 50MB to 120MB but Go heap stayed at 30MB.
The OS was caching file reads from the investigation scanner.

## Conclusion

False positive. The daemon memory is stable.
`
	os.WriteFile(f, []byte(content), 0644)

	cat := ClassifyOrphan(f)
	if cat != CategoryNegativeResult {
		t.Errorf("ClassifyOrphan(hypothesis rejected) = %q, want %q", cat, CategoryNegativeResult)
	}
}

func TestClassifyOrphan_Superseded(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "inv-superseded.md")

	content := `# Investigation: Old auth middleware design

**Status:** Complete

## What I Tried

Analyzed the middleware chain for token validation.

## What I Observed

Found three potential approaches for session handling.
This work was useful but the approach changed.

## Note

Superseded by the 2026-02-15 auth rewrite investigation which took a different
direction based on compliance requirements.
`
	os.WriteFile(f, []byte(content), 0644)

	cat := ClassifyOrphan(f)
	if cat != CategorySuperseded {
		t.Errorf("ClassifyOrphan(superseded) = %q, want %q", cat, CategorySuperseded)
	}
}

func TestClassifyOrphan_PositiveUnlinked(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "inv-positive.md")

	content := `# Investigation: Daemon spawn deduplication

**Status:** Complete
**Created:** 2026-02-10

## What I Tried

1. Examined daemon spawn flow
2. Traced duplicate detection logic
3. Measured false positive rates

## What I Observed

The daemon's spawn deduplication has a 15% false positive rate due to
fuzzy title matching. When two issues have similar titles but different
scopes, the dedup logic incorrectly blocks the second spawn.

Root cause: Levenshtein distance threshold of 0.3 is too aggressive.
A threshold of 0.5 would reduce false positives to ~3% based on
backtesting against the last 200 spawn decisions.

## Recommendation

Raise the Levenshtein threshold from 0.3 to 0.5 and add scope-aware
comparison (check issue labels, not just titles).
`
	os.WriteFile(f, []byte(content), 0644)

	cat := ClassifyOrphan(f)
	if cat != CategoryPositiveUnlinked {
		t.Errorf("ClassifyOrphan(positive unlinked) = %q, want %q", cat, CategoryPositiveUnlinked)
	}
}

func TestClassifyOrphan_UnreadableFile(t *testing.T) {
	cat := ClassifyOrphan("/nonexistent/file.md")
	if cat != CategoryPositiveUnlinked {
		t.Errorf("ClassifyOrphan(nonexistent) = %q, want %q", cat, CategoryPositiveUnlinked)
	}
}

func TestIsEmptyInvestigation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "just headers",
			content: `# Investigation

## What I Tried

## What I Observed

## Next Steps
`,
			want: true,
		},
		{
			name: "template with checkboxes",
			content: `# Investigation

**Status:** In Progress

- [ ] Step 1
- [ ] Step 2
- [ ] Step 3
`,
			want: true,
		},
		{
			name: "real content",
			content: `# Investigation: Real findings

**Status:** Complete

## What I Tried

I examined the daemon spawn logic and traced the dedup path.
The spawn tracker maintains a map of recent spawns keyed by issue ID.
When a new spawn arrives, it checks this map plus a fuzzy title match.

## What I Observed

The dedup has a 15% false positive rate because the Levenshtein
distance threshold is too low at 0.3. Backtesting shows 0.5 would
reduce false positives to 3% without increasing duplicates.

## Recommendation

Raise the threshold to 0.5.
`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmptyInvestigation(tt.content)
			if got != tt.want {
				t.Errorf("isEmptyInvestigation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNegativeResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "already fixed and works",
			content: "**TLDR:** Already fixed in prior commit. Works as expected now.",
			want:    true, // 2 signals: "already fixed" + "works as expected"
		},
		{
			name:    "not a bug",
			content: "After investigation, this is not a bug. No action needed.",
			want:    true, // 2 signals: "not a bug" + "no action needed"
		},
		{
			name:    "positive finding",
			content: "Found a race condition in the token refresh logic. Multiple goroutines can call refresh simultaneously.",
			want:    false,
		},
		{
			name:    "single signal insufficient",
			content: "No action needed — the metric was already being tracked by the existing pipeline.",
			want:    false, // only 1 signal: "no action needed"
		},
		{
			name:    "two signals in conclusion",
			content: "The feature already exists in the codebase. No change needed for this.",
			want:    true, // 2 signals: "already exists" + "no change needed"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNegativeResult(tt.content)
			if got != tt.want {
				t.Errorf("isNegativeResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSuperseded(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "superseded by",
			content: "This investigation was superseded by the March rewrite.",
			want:    true,
		},
		{
			name:    "replaced by",
			content: "The old approach was replaced by the new probe system.",
			want:    true,
		},
		{
			name:    "merged into",
			content: "Findings merged into the coaching-plugin synthesis.",
			want:    true,
		},
		{
			name:    "not superseded",
			content: "Found important performance regression in the SSE client.",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSuperseded(tt.content)
			if got != tt.want {
				t.Errorf("isSuperseded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeStratifiedOrphanRate(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	invDir := filepath.Join(kbDir, "investigations")
	modelDir := filepath.Join(kbDir, "models", "test")
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(modelDir, 0755)

	// Connected investigation (referenced by model)
	os.WriteFile(filepath.Join(invDir, "inv-connected.md"),
		[]byte("# Connected investigation with real findings about spawn dedup.\nLots of content here about the spawn deduplication logic.\n"), 0644)

	// Empty orphan
	os.WriteFile(filepath.Join(invDir, "inv-empty.md"),
		[]byte("# Investigation\n\n## What I Tried\n\n## What I Observed\n"), 0644)

	// Negative result orphan (needs 2+ signals and >= 5 content lines)
	os.WriteFile(filepath.Join(invDir, "inv-negative.md"),
		[]byte("# Check SSE bug\n\n**TLDR:** Already fixed in prior commit.\n\nThe SSE parsing works correctly now.\nTested with curl and verified events are parsed.\nThe original reporter confirmed it works as expected.\nNo action needed — closing as resolved.\nThis was already implemented in the December push.\n"), 0644)

	// Superseded orphan (needs >= 5 content lines)
	os.WriteFile(filepath.Join(invDir, "inv-old.md"),
		[]byte("# Old auth design\n\nAnalyzed the middleware chain for token validation.\nFound three potential approaches for session handling.\nDocumented trade-offs between JWT and sessions.\nThis work was useful but the approach changed.\nSuperseded by the February auth rewrite investigation.\nThe new investigation took a different direction.\n"), 0644)

	// Positive unlinked orphan
	os.WriteFile(filepath.Join(invDir, "inv-positive.md"),
		[]byte("# Daemon spawn dedup analysis\n\nFound 15% false positive rate in dedup logic.\nThe Levenshtein threshold of 0.3 is too aggressive.\nBacktesting shows 0.5 reduces FP to 3%.\nRecommend raising threshold.\nThis finding should inform the daemon model.\nBut nobody linked it.\nRoot cause is fuzzy matching on titles only.\nShould also compare labels and scope.\nFix is straightforward: two-line change in trigger.go.\nWould improve spawn throughput by 12%.\n"), 0644)

	// Model references one investigation
	modelContent := "# Model\nBased on .kb/investigations/inv-connected.md\n"
	os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(modelContent), 0644)

	report, err := ComputeStratifiedOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}

	if report.Total != 5 {
		t.Errorf("Total = %d, want 5", report.Total)
	}
	if report.Connected != 1 {
		t.Errorf("Connected = %d, want 1", report.Connected)
	}
	if report.Orphaned != 4 {
		t.Errorf("Orphaned = %d, want 4", report.Orphaned)
	}

	if report.Categories[CategoryEmpty] != 1 {
		t.Errorf("Empty = %d, want 1", report.Categories[CategoryEmpty])
	}
	if report.Categories[CategoryNegativeResult] != 1 {
		t.Errorf("NegativeResult = %d, want 1", report.Categories[CategoryNegativeResult])
	}
	if report.Categories[CategorySuperseded] != 1 {
		t.Errorf("Superseded = %d, want 1", report.Categories[CategorySuperseded])
	}
	if report.Categories[CategoryPositiveUnlinked] != 1 {
		t.Errorf("PositiveUnlinked = %d, want 1", report.Categories[CategoryPositiveUnlinked])
	}
}

func TestComputeStratifiedOrphanRate_Empty(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "investigations"), 0755)

	report, err := ComputeStratifiedOrphanRate(kbDir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 0 {
		t.Errorf("Total = %d, want 0", report.Total)
	}
}

func TestStratifiedOrphanReport_CategoryRate(t *testing.T) {
	report := &StratifiedOrphanReport{
		OrphanReport: OrphanReport{Orphaned: 100},
		Categories: map[OrphanCategory]int{
			CategoryEmpty:           10,
			CategoryNegativeResult:  30,
			CategorySuperseded:      20,
			CategoryPositiveUnlinked: 40,
		},
	}

	if rate := report.CategoryRate(CategoryEmpty); rate != 10.0 {
		t.Errorf("CategoryRate(empty) = %f, want 10.0", rate)
	}
	if rate := report.CategoryRate(CategoryPositiveUnlinked); rate != 40.0 {
		t.Errorf("CategoryRate(positive-unlinked) = %f, want 40.0", rate)
	}
}

func TestStratifiedOrphanReport_CategoryRate_ZeroOrphans(t *testing.T) {
	report := &StratifiedOrphanReport{
		OrphanReport: OrphanReport{Orphaned: 0},
		Categories:   make(map[OrphanCategory]int),
	}

	if rate := report.CategoryRate(CategoryEmpty); rate != 0 {
		t.Errorf("CategoryRate(empty) = %f, want 0", rate)
	}
}

func TestStratifiedSummary(t *testing.T) {
	report := &StratifiedOrphanReport{
		OrphanReport: OrphanReport{
			Total:      100,
			Connected:  10,
			Orphaned:   90,
			OrphanRate: 90.0,
		},
		Categories: map[OrphanCategory]int{
			CategoryEmpty:           20,
			CategoryNegativeResult:  15,
			CategorySuperseded:      10,
			CategoryPositiveUnlinked: 45,
		},
	}

	summary := report.StratifiedSummary()
	if summary == "" {
		t.Error("StratifiedSummary() returned empty")
	}
	if !strings.Contains(summary, "empty") {
		t.Error("Summary missing 'empty' category")
	}
	if !strings.Contains(summary, "positive-unlinked") {
		t.Error("Summary missing 'positive-unlinked' category")
	}
	if !strings.Contains(summary, "Actionable") {
		t.Error("Summary missing actionable line")
	}
}

func TestStratifiedSummary_Zero(t *testing.T) {
	report := &StratifiedOrphanReport{
		OrphanReport: OrphanReport{},
		Categories:   make(map[OrphanCategory]int),
	}
	if summary := report.StratifiedSummary(); summary != "" {
		t.Errorf("StratifiedSummary() = %q, want empty for zero total", summary)
	}
}

package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestMaxInvestigationsInWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var dates []time.Time

	// 51 investigations across a 20-day span should trip the 30-day window.
	for i := 0; i < 51; i++ {
		dates = append(dates, start.AddDate(0, 0, i%20))
	}

	got := maxInvestigationsInWindow(dates, 30)
	if got != 51 {
		t.Fatalf("maxInvestigationsInWindow() = %d, want 51", got)
	}
}

func TestApplyInvestigationCircuitBreaker_FiltersFeaturesWhenActive(t *testing.T) {
	projectDir := t.TempDir()
	writeInvestigationFiles(t, projectDir, ".kb/investigations", 51, "2026-01-")

	issues := []Issue{
		{ID: "a", IssueType: "feature"},
		{ID: "b", IssueType: "task"},
		{ID: "c", IssueType: "bug"},
	}

	filtered, err := applyInvestigationCircuitBreaker(issues, projectDir, false, time.Now().UTC())
	if err != nil {
		t.Fatalf("applyInvestigationCircuitBreaker() error = %v", err)
	}

	if len(filtered) != 2 {
		t.Fatalf("filtered issue count = %d, want 2", len(filtered))
	}
	for _, issue := range filtered {
		if issue.IssueType == "feature" {
			t.Fatalf("feature issue %q should be filtered when circuit breaker is active", issue.ID)
		}
	}
}

func TestApplyInvestigationCircuitBreaker_OverrideAllowsFeatures(t *testing.T) {
	projectDir := t.TempDir()
	writeInvestigationFiles(t, projectDir, ".kb/investigations", 51, "2026-01-")

	issues := []Issue{{ID: "f", IssueType: "feature"}}

	filtered, err := applyInvestigationCircuitBreaker(issues, projectDir, true, time.Now().UTC())
	if err != nil {
		t.Fatalf("applyInvestigationCircuitBreaker() error = %v", err)
	}

	if len(filtered) != 1 || filtered[0].IssueType != "feature" {
		t.Fatalf("override should preserve feature issues, got %+v", filtered)
	}
}

func TestApplyInvestigationCircuitBreaker_IgnoresArchivedInvestigations(t *testing.T) {
	projectDir := t.TempDir()
	writeInvestigationFiles(t, projectDir, ".kb/investigations/archived", 80, "2026-01-")

	issues := []Issue{{ID: "f", IssueType: "feature"}}

	filtered, err := applyInvestigationCircuitBreaker(issues, projectDir, false, time.Now().UTC())
	if err != nil {
		t.Fatalf("applyInvestigationCircuitBreaker() error = %v", err)
	}

	if len(filtered) != 1 || filtered[0].IssueType != "feature" {
		t.Fatalf("archived investigations should not trigger gate, got %+v", filtered)
	}
}

func writeInvestigationFiles(t *testing.T, projectDir, relDir string, count int, datePrefix string) {
	t.Helper()

	baseDir := filepath.Join(projectDir, relDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", baseDir, err)
	}

	for i := 0; i < count; i++ {
		day := (i % 28) + 1
		filename := datePrefix + pad2(day) + "-inv-test-" + strconv.Itoa(i) + ".md"
		path := filepath.Join(baseDir, filename)
		if err := os.WriteFile(path, []byte("# test\n"), 0644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
}

func pad2(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

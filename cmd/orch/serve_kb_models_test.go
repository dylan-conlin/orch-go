package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanKBModelProbesAggregatesModelsQueueAndSummary(t *testing.T) {
	projectDir := t.TempDir()
	modelsDir := filepath.Join(projectDir, ".kb", "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("failed to create models dir: %v", err)
	}

	today := time.Now()
	oldDate := today.AddDate(0, 0, -45).Format("2006-01-02")
	recentA := today.AddDate(0, 0, -2).Format("2006-01-02")
	recentB := today.AddDate(0, 0, -1).Format("2006-01-02")
	recentC := today.AddDate(0, 0, -3).Format("2006-01-02")

	mergedName := writeProbeFixture(t, projectDir, "needs-review", recentA, "merged-extends", "extends", "Merged extends claim")
	_ = writeProbeFixture(t, projectDir, "needs-review", recentB, "unmerged-contradicts", "contradicts", "Unmerged contradiction claim")

	for i, date := range []string{recentA, recentB, recentC} {
		writeProbeFixture(t, projectDir, "well-validated", date, fmt.Sprintf("confirm-%d", i+1), "confirms", "Confirming claim")
	}

	writeProbeFixture(t, projectDir, "stale-model", oldDate, "old-confirm", "confirms", "Old claim")

	writeModelFixture(t, projectDir, "needs-review", fmt.Sprintf("**Recent Probes:**\n- `probes/%s.md`", mergedName))
	writeModelFixture(t, projectDir, "well-validated", "")
	writeModelFixture(t, projectDir, "stale-model", "")

	if err := os.WriteFile(filepath.Join(modelsDir, "README.md"), []byte("# Models"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "PHASE4_REVIEW.md"), []byte("# Phase"), 0644); err != nil {
		t.Fatalf("failed to write PHASE4_REVIEW.md: %v", err)
	}

	models, queue, summary, err := scanKBModelProbes(projectDir, 30)
	if err != nil {
		t.Fatalf("scanKBModelProbes returned error: %v", err)
	}

	if summary.ModelsTotal != 3 {
		t.Fatalf("expected 3 models, got %d", summary.ModelsTotal)
	}
	if summary.ProbesTotal != 6 {
		t.Fatalf("expected 6 probes total, got %d", summary.ProbesTotal)
	}
	if summary.NeedsReview != 1 || summary.Stale != 1 || summary.WellValidated != 1 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}

	if len(queue) != 1 {
		t.Fatalf("expected one queue item, got %d", len(queue))
	}
	if queue[0].Model != "needs-review" {
		t.Fatalf("expected queue model needs-review, got %q", queue[0].Model)
	}
	if queue[0].Verdict != "contradicts" {
		t.Fatalf("expected queue verdict contradicts, got %q", queue[0].Verdict)
	}
	if queue[0].Claim != "Unmerged contradiction claim" {
		t.Fatalf("expected extracted claim, got %q", queue[0].Claim)
	}

	byName := map[string]KBModelProbeModel{}
	for _, model := range models {
		byName[model.Name] = model
	}

	if byName["needs-review"].Status != "needs_review" {
		t.Fatalf("expected needs-review model status needs_review, got %q", byName["needs-review"].Status)
	}
	if byName["needs-review"].UnmergedCount != 1 {
		t.Fatalf("expected needs-review unmerged_count 1, got %d", byName["needs-review"].UnmergedCount)
	}
	if byName["well-validated"].Status != "well_validated" {
		t.Fatalf("expected well-validated model status well_validated, got %q", byName["well-validated"].Status)
	}
	if byName["stale-model"].Status != "stale" {
		t.Fatalf("expected stale-model status stale, got %q", byName["stale-model"].Status)
	}
}

func TestParseProbeFieldsFromSectionContent(t *testing.T) {
	content := `# Probe

## Question

Expected first claim line.
Follow-up detail line.

## Notes

**Verdict:** contradicts

## Model Impact

**Verdict:** extends - useful nuance
`

	if got := parseProbeClaimFromContent(content); got != "Expected first claim line." {
		t.Fatalf("expected first question line, got %q", got)
	}
	if got := parseProbeVerdictFromContent(content); got != "extends" {
		t.Fatalf("expected verdict from Model Impact section, got %q", got)
	}
}

func TestHandleKBModelProbesRejectsInvalidStaleDays(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/kb/model-probes?stale_days=bad", nil)
	rec := httptest.NewRecorder()

	newTestServer().handleKBModelProbes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp KBModelProbesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == "" {
		t.Fatal("expected error for invalid stale_days")
	}
}

func TestHandleKBModelProbesMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/kb/model-probes", nil)
	rec := httptest.NewRecorder()

	newTestServer().handleKBModelProbes(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}

func writeModelFixture(t *testing.T, projectDir, name, extra string) {
	t.Helper()
	content := fmt.Sprintf("# Model: %s\n\n**Last Updated:** %s\n", name, time.Now().Format("2006-01-02"))
	if extra != "" {
		content += "\n" + extra + "\n"
	}

	path := filepath.Join(projectDir, ".kb", "models", name+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write model fixture: %v", err)
	}
}

func writeProbeFixture(t *testing.T, projectDir, model, date, slug, verdict, question string) string {
	t.Helper()

	probesDir := filepath.Join(projectDir, ".kb", "models", model, "probes")
	if err := os.MkdirAll(probesDir, 0755); err != nil {
		t.Fatalf("failed to create probes dir: %v", err)
	}

	name := fmt.Sprintf("%s-%s", date, slug)
	path := filepath.Join(probesDir, name+".md")
	content := fmt.Sprintf(`# Probe

**Date:** %s

## Question

%s

More detail.

## Model Impact

**Verdict:** %s — fixture
`, date, question, verdict)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write probe fixture: %v", err)
	}

	return name
}

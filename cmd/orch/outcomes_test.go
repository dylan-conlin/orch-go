package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

func TestBuildOutcomeReport(t *testing.T) {
	projectDir, _ := setupOutcomeFixture(t)

	report, err := buildOutcomeReport(projectDir)
	if err != nil {
		t.Fatalf("buildOutcomeReport returned error: %v", err)
	}

	if report.DataQuality.IssuesTotal != 3 {
		t.Fatalf("expected 3 issues, got %d", report.DataQuality.IssuesTotal)
	}

	bySkill := map[string]OutcomeSkillCounts{}
	for _, row := range report.CountsBySkill {
		bySkill[row.Skill] = row
	}

	feature := bySkill["feature-impl"]
	if feature.Completion != 1 || feature.Abandoned != 1 || feature.Open != 0 || feature.Total != 2 {
		t.Fatalf("unexpected feature-impl counts: %+v", feature)
	}

	debug := bySkill["systematic-debugging"]
	if debug.Open != 1 || debug.Total != 1 {
		t.Fatalf("unexpected systematic-debugging counts: %+v", debug)
	}

	if report.DurationDistribution.Samples != 2 {
		t.Fatalf("expected 2 duration samples, got %d", report.DurationDistribution.Samples)
	}

	if math.Abs(report.DurationDistribution.P50Minutes-15.0) > 0.01 {
		t.Fatalf("expected p50=15, got %.2f", report.DurationDistribution.P50Minutes)
	}

	if math.Abs(report.DurationDistribution.P90Minutes-19.0) > 0.01 {
		t.Fatalf("expected p90=19, got %.2f", report.DurationDistribution.P90Minutes)
	}

	if math.Abs(report.DurationDistribution.P95Minutes-19.5) > 0.01 {
		t.Fatalf("expected p95=19.5, got %.2f", report.DurationDistribution.P95Minutes)
	}

	reasons := map[string]int{}
	for _, item := range report.AbandonmentReasons {
		reasons[item.Reason] = item.Count
	}

	if reasons["stuck"] != 2 {
		t.Fatalf("expected reason 'stuck' count=2, got %d", reasons["stuck"])
	}

	if reasons["unknown"] != 1 {
		t.Fatalf("expected reason 'unknown' count=1, got %d", reasons["unknown"])
	}

	if report.InvestigationToModel.TotalInvestigations != 2 {
		t.Fatalf("expected 2 active investigations, got %d", report.InvestigationToModel.TotalInvestigations)
	}

	if report.InvestigationToModel.InvestigationsWithModelCitation != 1 {
		t.Fatalf("expected 1 investigation with model citation, got %d", report.InvestigationToModel.InvestigationsWithModelCitation)
	}

	if math.Abs(report.InvestigationToModel.ThroughputRate-50.0) > 0.01 {
		t.Fatalf("expected throughput=50%%, got %.2f", report.InvestigationToModel.ThroughputRate)
	}
}

func TestHandleOutcomes(t *testing.T) {
	projectDir, _ := setupOutcomeFixture(t)

	s := newTestServer()
	s.SourceDir = projectDir

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/outcomes", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var report OutcomeReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(report.CountsBySkill) == 0 {
		t.Fatal("expected non-empty counts_by_skill")
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/outcomes", nil)
	postW := httptest.NewRecorder()
	mux.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", postW.Code)
	}
}

func setupOutcomeFixture(t *testing.T) (string, string) {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(projectDir, ".beads"), 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	base := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	issues := []beads.Issue{
		{
			ID:          "orch-go-1",
			Title:       "Feature issue",
			Status:      "closed",
			IssueType:   "feature",
			ClosedAt:    base.Add(10 * time.Minute).Format(time.RFC3339),
			CloseReason: "Completed",
		},
		{
			ID:        "orch-go-2",
			Title:     "Bug issue",
			Status:    "open",
			IssueType: "bug",
		},
		{
			ID:          "orch-go-3",
			Title:       "Task issue",
			Status:      "closed",
			IssueType:   "task",
			ClosedAt:    base.Add(40 * time.Minute).Format(time.RFC3339),
			CloseReason: "Auto-abandoned due to timeout",
		},
	}

	writeIssuesJSONL(t, filepath.Join(projectDir, ".beads", "issues.jsonl"), issues)

	writeWorkspaceMetadata(t, filepath.Join(projectDir, ".orch", "workspace", "ws-one"), "orch-go-1", base)
	writeWorkspaceMetadata(t, filepath.Join(projectDir, ".orch", "workspace", "archived", "ws-three"), "orch-go-3", base.Add(20*time.Minute))

	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "investigations", "archived"), 0755); err != nil {
		t.Fatalf("failed to create investigations directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".kb", "models"), 0755); err != nil {
		t.Fatalf("failed to create models directory: %v", err)
	}

	invOne := "2026-02-01-inv-one.md"
	invTwo := "2026-02-02-inv-two.md"
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "investigations", invOne), []byte("# one\n"), 0644); err != nil {
		t.Fatalf("failed to write investigation one: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "investigations", invTwo), []byte("# two\n"), 0644); err != nil {
		t.Fatalf("failed to write investigation two: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "investigations", "archived", "2026-01-01-inv-archived.md"), []byte("# archived\n"), 0644); err != nil {
		t.Fatalf("failed to write archived investigation: %v", err)
	}

	modelBody := "References: 2026-02-01-inv-one.md\n"
	if err := os.WriteFile(filepath.Join(projectDir, ".kb", "models", "model.md"), []byte(modelBody), 0644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(homeDir, ".orch"), 0755); err != nil {
		t.Fatalf("failed to create home .orch directory: %v", err)
	}
	writeAbandonmentEvents(t, filepath.Join(homeDir, ".orch", "events.jsonl"), base)

	return projectDir, homeDir
}

func writeIssuesJSONL(t *testing.T, path string, issues []beads.Issue) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create issues file: %v", err)
	}
	defer f.Close()

	for _, issue := range issues {
		line, err := json.Marshal(issue)
		if err != nil {
			t.Fatalf("failed to marshal issue: %v", err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			t.Fatalf("failed to write issue line: %v", err)
		}
	}
}

func writeWorkspaceMetadata(t *testing.T, workspacePath, beadsID string, spawnTime time.Time) {
	t.Helper()

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workspacePath, ".beads_id"), []byte(beadsID+"\n"), 0644); err != nil {
		t.Fatalf("failed to write .beads_id: %v", err)
	}

	spawnValue := fmt.Sprintf("%d\n", spawnTime.UnixNano())
	if err := os.WriteFile(filepath.Join(workspacePath, ".spawn_time"), []byte(spawnValue), 0644); err != nil {
		t.Fatalf("failed to write .spawn_time: %v", err)
	}
}

func writeAbandonmentEvents(t *testing.T, eventsPath string, base time.Time) {
	t.Helper()

	events := []map[string]interface{}{
		{
			"type":      "agent.abandoned",
			"timestamp": base.Add(1 * time.Hour).Unix(),
			"data": map[string]interface{}{
				"beads_id": "orch-go-3",
				"reason":   "stuck",
			},
		},
		{
			"type":      "agent.abandoned",
			"timestamp": base.Add(2 * time.Hour).Unix(),
			"data": map[string]interface{}{
				"beads_id": "orch-go-4",
			},
		},
		{
			"type":      "agent.abandoned",
			"timestamp": base.Add(3 * time.Hour).Unix(),
			"data": map[string]interface{}{
				"beads_id": "orch-go-5",
				"reason":   "stuck",
			},
		},
	}

	f, err := os.Create(eventsPath)
	if err != nil {
		t.Fatalf("failed to create events file: %v", err)
	}
	defer f.Close()

	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			t.Fatalf("failed to write event line: %v", err)
		}
	}
}

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFormatOperatorHealthSummaryIncludesAllCards(t *testing.T) {
	ratio := 2.4
	report := &OperatorHealthResponse{
		GeneratedAt: "2026-02-07T12:00:00Z",
		CrashFreeStreak: crashFreeStreakMetric{
			Status:            operatorHealthStatusHealthy,
			CurrentStreak:     "4d 2h",
			CurrentStreakDays: 4,
			TargetDays:        7,
			ProgressPercent:   57.1,
		},
		ResourceCeilings: resourceCeilingsMetric{
			Status: operatorHealthStatusWarning,
			Baseline: resourceMetrics{
				Goroutines:          10,
				HeapBytes:           1024,
				ChildProcesses:      2,
				OpenFileDescriptors: 12,
			},
			Current: resourceMetrics{
				Goroutines:          14,
				HeapBytes:           4096,
				ChildProcesses:      3,
				OpenFileDescriptors: 16,
			},
			CeilingMultiplier: 2,
			Breached:          false,
		},
		DefectClassClusters: defectClassClustersMetric{
			Status: operatorHealthStatusWarning,
			TopClasses: []defectClassClusterItem{
				{DefectClass: "resource-leak", Count: 8},
			},
		},
		AgentHealthRatio7d: agentHealthRatioMetric{
			Status:                    operatorHealthStatusWarning,
			WindowDays:                7,
			Completions:               12,
			Abandonments:              5,
			CompletionShare:           0.705,
			CompletionsPerAbandonment: &ratio,
		},
		ProcessCensus: processCensusMetric{
			Status:         operatorHealthStatusCritical,
			ChildProcesses: 6,
			OrphanedCount:  2,
			OrphanedProcesses: []orphanProcessEntry{
				{PID: 1234, Command: "bun"},
			},
		},
	}

	output := formatOperatorHealthSummary(report)

	requiredCards := []string{
		"Crash-Free Streak",
		"Resource Ceilings",
		"Defect Clusters (30d)",
		"Agent Health Ratio (7d)",
		"Process Census",
	}

	for _, card := range requiredCards {
		if !strings.Contains(output, card) {
			t.Fatalf("expected output to include card %q", card)
		}
	}

	if !strings.Contains(output, "\x1b[32m") {
		t.Fatalf("expected output to include green ANSI color code")
	}
	if !strings.Contains(output, "\x1b[33m") {
		t.Fatalf("expected output to include yellow ANSI color code")
	}
	if !strings.Contains(output, "\x1b[31m") {
		t.Fatalf("expected output to include red ANSI color code")
	}
}

func TestFetchOperatorHealthReportSuccess(t *testing.T) {
	var requestedPath string
	var requestedProject string

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedProject = r.URL.Query().Get("project")

		response := OperatorHealthResponse{
			GeneratedAt: "2026-02-07T12:00:00Z",
			CrashFreeStreak: crashFreeStreakMetric{
				Status:        operatorHealthStatusHealthy,
				CurrentStreak: "2d",
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode test response: %v", err)
		}
	}))
	defer ts.Close()

	report, err := fetchOperatorHealthReport(ts.Client(), ts.URL, "/tmp/project-a")
	if err != nil {
		t.Fatalf("fetchOperatorHealthReport returned error: %v", err)
	}

	if requestedPath != "/api/operator-health" {
		t.Fatalf("expected path /api/operator-health, got %q", requestedPath)
	}
	if requestedProject != "/tmp/project-a" {
		t.Fatalf("expected project query to be /tmp/project-a, got %q", requestedProject)
	}
	if report.GeneratedAt != "2026-02-07T12:00:00Z" {
		t.Fatalf("expected generated_at to round-trip, got %q", report.GeneratedAt)
	}
}

func TestRunHealthWithClientReturnsHelpfulErrorWhenServeUnavailable(t *testing.T) {
	client := &http.Client{Timeout: 100 * time.Millisecond}

	err := runHealthWithClient(client, "https://127.0.0.1:1", "/tmp/project-b")
	if err == nil {
		t.Fatal("expected runHealthWithClient to return an error")
	}

	errText := err.Error()
	if !strings.Contains(errText, "orch serve") {
		t.Fatalf("expected error to mention orch serve, got: %s", errText)
	}
	if !strings.Contains(errText, "orch-dashboard start") {
		t.Fatalf("expected error to include startup guidance, got: %s", errText)
	}
}

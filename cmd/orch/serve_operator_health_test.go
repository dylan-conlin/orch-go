package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/stability"
)

func TestBuildCrashFreeStreakMetricIgnoresAgentAbandonedForLastRecovery(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	if err := os.MkdirAll(filepath.Join(homeDir, ".orch"), 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	now := time.Now().UTC()
	healthy := true
	path := stability.DefaultPath()
	entries := []stability.Entry{
		{
			Type:     stability.TypeSnapshot,
			Ts:       now.Add(-2 * time.Hour).Unix(),
			Healthy:  &healthy,
			Services: map[string]bool{"OpenCode": true},
		},
		{
			Type:   stability.TypeIntervention,
			Ts:     now.Add(-1 * time.Hour).Unix(),
			Source: stability.SourceManualRecovery,
			Detail: "OpenCode restarted manually",
		},
		{
			Type:    stability.TypeIntervention,
			Ts:      now.Add(-30 * time.Minute).Unix(),
			Source:  stability.SourceAgentAbandoned,
			Detail:  "orch-go-999 abandoned",
			BeadsID: "orch-go-999",
		},
	}
	writeStabilityEntries(t, path, entries)

	metric, err := buildCrashFreeStreakMetric()
	if err != nil {
		t.Fatalf("buildCrashFreeStreakMetric returned error: %v", err)
	}

	if metric.LastIntervention == nil {
		t.Fatal("expected last_intervention to be set")
	}

	if metric.LastIntervention.Source != stability.SourceManualRecovery {
		t.Fatalf("expected last_intervention source %q, got %q", stability.SourceManualRecovery, metric.LastIntervention.Source)
	}
}

func TestHandleOperatorHealthMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/operator-health", nil)
	rec := httptest.NewRecorder()

	newTestServer().handleOperatorHealth(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestHandleOperatorHealthReturnsExpectedMetrics(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	if err := os.MkdirAll(filepath.Join(homeDir, ".orch"), 0755); err != nil {
		t.Fatalf("failed to create .orch dir: %v", err)
	}

	stabilityPath := filepath.Join(homeDir, ".orch", "stability.jsonl")
	recorder := stability.NewRecorder(stabilityPath)
	if err := recorder.RecordSnapshot(true, map[string]bool{"OpenCode": true}); err != nil {
		t.Fatalf("failed to write stability snapshot: %v", err)
	}
	if err := recorder.RecordIntervention(stability.SourceDoctorFix, "manual recovery test", []string{"OpenCode"}, "orch-go-123"); err != nil {
		t.Fatalf("failed to write stability intervention: %v", err)
	}

	eventsPath := filepath.Join(homeDir, ".orch", "events.jsonl")
	eventsData := []byte(
		"{\"type\":\"session.spawned\",\"session_id\":\"sess-1\",\"timestamp\":" + itoa(time.Now().Add(-2*time.Hour).Unix()) + ",\"data\":{\"skill\":\"feature-impl\",\"beads_id\":\"orch-go-111\"}}\n" +
			"{\"type\":\"agent.completed\",\"timestamp\":" + itoa(time.Now().Add(-90*time.Minute).Unix()) + ",\"data\":{\"beads_id\":\"orch-go-111\"}}\n" +
			"{\"type\":\"agent.abandoned\",\"timestamp\":" + itoa(time.Now().Add(-30*time.Minute).Unix()) + ",\"data\":{\"beads_id\":\"orch-go-222\"}}\n",
	)
	if err := os.WriteFile(eventsPath, eventsData, 0644); err != nil {
		t.Fatalf("failed to write events file: %v", err)
	}

	projectDir := t.TempDir()

	samples := []resourceSample{
		{metrics: resourceMetrics{Goroutines: 8, HeapBytes: 80, ChildProcesses: 2, OpenFileDescriptors: 6}},
		{metrics: resourceMetrics{Goroutines: 10, HeapBytes: 90, ChildProcesses: 4, OpenFileDescriptors: 8}},
	}
	sampleIndex := 0
	monitor := newResourceMonitorWithSampler(nil, func() resourceSample {
		if sampleIndex >= len(samples) {
			return samples[len(samples)-1]
		}
		sample := samples[sampleIndex]
		sampleIndex++
		return sample
	})

	s := newTestServer()
	s.ResourceMonitor = monitor

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/operator-health?project="+projectDir, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response OperatorHealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.GeneratedAt == "" {
		t.Fatalf("expected generated_at to be set")
	}

	if response.ResourceCeilings.Current.ChildProcesses != 4 {
		t.Fatalf("expected current child processes 4, got %d", response.ResourceCeilings.Current.ChildProcesses)
	}

	if response.AgentHealthRatio7d.Completions != 1 {
		t.Fatalf("expected completions=1, got %d", response.AgentHealthRatio7d.Completions)
	}

	if response.AgentHealthRatio7d.Abandonments != 1 {
		t.Fatalf("expected abandonments=1, got %d", response.AgentHealthRatio7d.Abandonments)
	}

	if response.CrashFreeStreak.Status == "" {
		t.Fatalf("expected crash_free_streak.status to be set")
	}
}

func TestIsOrchRelatedProcess(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    string
		want    bool
	}{
		// Legitimate PPID=1 processes (should return false - not orch orphans)
		{
			name:    "overmind should not be flagged as orch orphan",
			command: "overmind",
			args:    "start -f Procfile",
			want:    false,
		},
		{
			name:    "tmux should not be flagged as orch orphan",
			command: "tmux",
			args:    "new-session -d -s main",
			want:    false,
		},
		{
			name:    "macOS system process should not be flagged",
			command: "/System/Library/PrivateFrameworks/CoreServicesInternal",
			args:    "",
			want:    false,
		},
		{
			name:    "vite dev server should not be flagged as orphan",
			command: "node",
			args:    "/path/to/vite/bin/vite.js dev",
			want:    false,
		},
		{
			name:    "launchd opencode serve binary should not be flagged as orphan",
			command: "/Users/user/.bun/bin/opencode",
			args:    "serve --port 4096",
			want:    false,
		},
		{
			name:    "sketchybar helper script should not be flagged as orphan",
			command: "zsh",
			args:    "/Users/user/.config/sketchybar/helpers/orch-status.sh",
			want:    false,
		},

		// Actual orch-related processes (should return true - potential orphans)
		{
			name:    "bun process with .orch in path should be flagged",
			command: "bun",
			args:    "/Users/user/project/.orch/workspace/agent/script.js",
			want:    true,
		},
		{
			name:    "opencode process should be flagged",
			command: "opencode",
			args:    "--port 4096",
			want:    true,
		},
		{
			name:    "orch binary should be flagged",
			command: "orch",
			args:    "serve --daemon",
			want:    true,
		},
		{
			name:    "node process with opencode should be flagged",
			command: "node",
			args:    "/path/to/opencode/server.js",
			want:    true,
		},
		{
			name:    "bun with run --attach should be flagged",
			command: "bun",
			args:    "run --attach session-123",
			want:    true,
		},

		// Unrelated processes (should return false)
		{
			name:    "unrelated process should not be flagged",
			command: "firefox",
			args:    "https://example.com",
			want:    false,
		},
		{
			name:    "launchd should not be flagged",
			command: "launchd",
			args:    "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOrchRelatedProcess(tt.command, tt.args)
			if got != tt.want {
				t.Errorf("isOrchRelatedProcess(%q, %q) = %v, want %v", tt.command, tt.args, got, tt.want)
			}
		})
	}
}

func TestParseOrphanedOrchProcessesSkipsSelfPID(t *testing.T) {
	output := `101 1 orch orch serve --port 3348
102 1 bun bun run --attach session-123
103 2 orch orch serve --daemon`

	orphans, err := parseOrphanedOrchProcesses(output, 20, 101)
	if err != nil {
		t.Fatalf("parseOrphanedOrchProcesses returned error: %v", err)
	}

	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan after self-PID exclusion, got %d", len(orphans))
	}

	if orphans[0].PID != 102 {
		t.Fatalf("expected PID 102, got %d", orphans[0].PID)
	}
}

func TestParseOrphanedOrchProcessesKeepsOtherOrchServe(t *testing.T) {
	output := `201 1 orch orch serve --port 3348`

	orphans, err := parseOrphanedOrchProcesses(output, 20, 999)
	if err != nil {
		t.Fatalf("parseOrphanedOrchProcesses returned error: %v", err)
	}

	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}

	if orphans[0].PID != 201 {
		t.Fatalf("expected PID 201, got %d", orphans[0].PID)
	}
}

func writeStabilityEntries(t *testing.T, path string, entries []stability.Entry) {
	t.Helper()

	encoded := make([]byte, 0, len(entries)*128)
	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("failed to marshal stability entry: %v", err)
		}
		encoded = append(encoded, line...)
		encoded = append(encoded, '\n')
	}

	if err := os.WriteFile(path, encoded, 0644); err != nil {
		t.Fatalf("failed to write stability entries: %v", err)
	}
}

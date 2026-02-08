package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

func TestDetectResourceBreachesRequiresStrictlyGreaterThanTwoX(t *testing.T) {
	baseline := resourceMetrics{
		Goroutines:          10,
		HeapBytes:           100,
		ChildProcesses:      2,
		OpenFileDescriptors: 8,
	}
	current := resourceMetrics{
		Goroutines:          20,
		HeapBytes:           200,
		ChildProcesses:      4,
		OpenFileDescriptors: 16,
	}

	breaches := detectResourceBreaches(baseline, current)
	if len(breaches) != 0 {
		t.Fatalf("expected no breaches at exactly 2x baseline, got %d", len(breaches))
	}
}

func TestResourceMonitorLogsOnlyOnBreachTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")
	logger := events.NewLogger(logPath)

	samples := []resourceSample{
		{metrics: resourceMetrics{Goroutines: 10, HeapBytes: 100, ChildProcesses: 1, OpenFileDescriptors: 5}}, // baseline
		{metrics: resourceMetrics{Goroutines: 25, HeapBytes: 100, ChildProcesses: 1, OpenFileDescriptors: 5}}, // breach
		{metrics: resourceMetrics{Goroutines: 27, HeapBytes: 100, ChildProcesses: 1, OpenFileDescriptors: 5}}, // still breached
		{metrics: resourceMetrics{Goroutines: 15, HeapBytes: 100, ChildProcesses: 1, OpenFileDescriptors: 5}}, // recovered
		{metrics: resourceMetrics{Goroutines: 23, HeapBytes: 100, ChildProcesses: 1, OpenFileDescriptors: 5}}, // breach again
	}

	index := 0
	monitor := newResourceMonitorWithSampler(logger, func() resourceSample {
		if index >= len(samples) {
			return samples[len(samples)-1]
		}
		sample := samples[index]
		index++
		return sample
	})

	monitor.sampleAndCheck()
	monitor.sampleAndCheck()
	monitor.sampleAndCheck()
	monitor.sampleAndCheck()

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 breach events, got %d", len(lines))
	}

	for _, line := range lines {
		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("failed to parse event JSON: %v", err)
		}
		if event.Type != events.EventTypeResourceCeilingBreach {
			t.Fatalf("expected event type %q, got %q", events.EventTypeResourceCeilingBreach, event.Type)
		}
	}
}

func TestHealthEndpointIncludesResourceBaselineAndCurrent(t *testing.T) {
	samples := []resourceSample{
		{metrics: resourceMetrics{Goroutines: 8, HeapBytes: 80, ChildProcesses: 1, OpenFileDescriptors: 6}},  // baseline
		{metrics: resourceMetrics{Goroutines: 10, HeapBytes: 90, ChildProcesses: 1, OpenFileDescriptors: 7}}, // current
	}

	index := 0
	monitor := newResourceMonitorWithSampler(nil, func() resourceSample {
		if index >= len(samples) {
			return samples[len(samples)-1]
		}
		sample := samples[index]
		index++
		return sample
	})

	s := newTestServer()
	s.ResourceMonitor = monitor

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Status    string               `json:"status"`
		Resources resourceHealthReport `json:"resources"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode /health response: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", response.Status)
	}
	if response.Resources.Baseline.Goroutines != 8 {
		t.Fatalf("expected baseline goroutines 8, got %d", response.Resources.Baseline.Goroutines)
	}
	if response.Resources.Current.Goroutines != 10 {
		t.Fatalf("expected current goroutines 10, got %d", response.Resources.Current.Goroutines)
	}
	if response.Resources.CeilingMultiplier != resourceCeilingMultiplier {
		t.Fatalf("expected ceiling multiplier %d, got %d", resourceCeilingMultiplier, response.Resources.CeilingMultiplier)
	}
}

package coaching

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadMetrics(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		limit        int
		wantCount    int
		wantLastType string
		wantErr      bool
	}{
		{
			name: "read last 2 of 3 metrics",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}
{"timestamp":"2026-02-14T10:01:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
{"timestamp":"2026-02-14T10:02:00Z","session_id":"sess1","metric_type":"frame_collapse","value":1}
`,
			limit:        2,
			wantCount:    2,
			wantLastType: "frame_collapse",
			wantErr:      false,
		},
		{
			name: "limit larger than available metrics",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}
{"timestamp":"2026-02-14T10:01:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
`,
			limit:        100,
			wantCount:    2,
			wantLastType: "analysis_paralysis",
			wantErr:      false,
		},
		{
			name:      "empty file",
			content:   "",
			limit:     10,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "file with blank lines",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}

{"timestamp":"2026-02-14T10:01:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
`,
			limit:        10,
			wantCount:    2,
			wantLastType: "analysis_paralysis",
			wantErr:      false,
		},
		{
			name: "file with malformed line",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}
this is not json
{"timestamp":"2026-02-14T10:01:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
`,
			limit:        10,
			wantCount:    2,
			wantLastType: "analysis_paralysis",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "metrics.jsonl")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Test ReadMetrics
			got, err := ReadMetrics(tmpFile, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("ReadMetrics() got %d metrics, want %d", len(got), tt.wantCount)
			}

			if tt.wantCount > 0 && got[len(got)-1].Type != tt.wantLastType {
				t.Errorf("ReadMetrics() last metric type = %v, want %v", got[len(got)-1].Type, tt.wantLastType)
			}
		})
	}
}

func TestReadMetrics_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist.jsonl")

	got, err := ReadMetrics(nonExistent, 10)
	if err != nil {
		t.Errorf("ReadMetrics() should return empty slice for non-existent file, got error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ReadMetrics() should return empty slice for non-existent file, got %d metrics", len(got))
	}
}

func TestReadMetricsSince(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		since     time.Time
		wantCount int
		wantTypes []string
	}{
		{
			name: "filter by time",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}
{"timestamp":"2026-02-14T10:30:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
{"timestamp":"2026-02-14T11:00:00Z","session_id":"sess1","metric_type":"frame_collapse","value":1}
`,
			since:     time.Date(2026, 2, 14, 10, 15, 0, 0, time.UTC),
			wantCount: 2,
			wantTypes: []string{"analysis_paralysis", "frame_collapse"},
		},
		{
			name: "no metrics after time",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}
{"timestamp":"2026-02-14T10:30:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
`,
			since:     time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC),
			wantCount: 0,
		},
		{
			name: "all metrics after time",
			content: `{"timestamp":"2026-02-14T10:00:00Z","session_id":"sess1","metric_type":"action_ratio","value":0.5}
{"timestamp":"2026-02-14T10:30:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
`,
			since:     time.Date(2026, 2, 14, 9, 0, 0, 0, time.UTC),
			wantCount: 2,
			wantTypes: []string{"action_ratio", "analysis_paralysis"},
		},
		{
			name: "skip malformed timestamps",
			content: `{"timestamp":"invalid","session_id":"sess1","metric_type":"action_ratio","value":0.5}
{"timestamp":"2026-02-14T10:30:00Z","session_id":"sess1","metric_type":"analysis_paralysis","value":2}
`,
			since:     time.Date(2026, 2, 14, 9, 0, 0, 0, time.UTC),
			wantCount: 1,
			wantTypes: []string{"analysis_paralysis"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "metrics.jsonl")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Test ReadMetricsSince
			got, err := ReadMetricsSince(tmpFile, tt.since)
			if err != nil {
				t.Errorf("ReadMetricsSince() error = %v", err)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("ReadMetricsSince() got %d metrics, want %d", len(got), tt.wantCount)
			}

			// Verify metric types
			for i, want := range tt.wantTypes {
				if i >= len(got) {
					break
				}
				if got[i].Type != want {
					t.Errorf("ReadMetricsSince() metric[%d].Type = %v, want %v", i, got[i].Type, want)
				}
			}
		})
	}
}

func TestReadMetricsSince_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist.jsonl")

	got, err := ReadMetricsSince(nonExistent, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Errorf("ReadMetricsSince() should return empty slice for non-existent file, got error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ReadMetricsSince() should return empty slice for non-existent file, got %d metrics", len(got))
	}
}

func TestAggregateByType(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name    string
		metrics []Metric
		since   time.Time
		want    map[string]MetricSummary
	}{
		{
			name: "aggregate multiple types",
			metrics: []Metric{
				{Timestamp: oneHourAgo.Format(time.RFC3339), Type: "action_ratio", Value: 0.5},
				{Timestamp: oneHourAgo.Add(10 * time.Minute).Format(time.RFC3339), Type: "action_ratio", Value: 0.7},
				{Timestamp: oneHourAgo.Add(20 * time.Minute).Format(time.RFC3339), Type: "analysis_paralysis", Value: 2},
			},
			since: twoHoursAgo,
			want: map[string]MetricSummary{
				"action_ratio": {
					Type:         "action_ratio",
					Count:        2,
					LatestValue:  0.7,
					AverageValue: 0.6,
				},
				"analysis_paralysis": {
					Type:         "analysis_paralysis",
					Count:        1,
					LatestValue:  2,
					AverageValue: 2,
				},
			},
		},
		{
			name: "filter by time",
			metrics: []Metric{
				{Timestamp: twoHoursAgo.Format(time.RFC3339), Type: "action_ratio", Value: 0.3},
				{Timestamp: oneHourAgo.Format(time.RFC3339), Type: "action_ratio", Value: 0.7},
			},
			since: twoHoursAgo.Add(30 * time.Minute),
			want: map[string]MetricSummary{
				"action_ratio": {
					Type:         "action_ratio",
					Count:        1,
					LatestValue:  0.7,
					AverageValue: 0.7,
				},
			},
		},
		{
			name:    "empty metrics",
			metrics: []Metric{},
			since:   twoHoursAgo,
			want:    map[string]MetricSummary{},
		},
		{
			name: "skip invalid timestamps",
			metrics: []Metric{
				{Timestamp: "invalid", Type: "action_ratio", Value: 0.5},
				{Timestamp: oneHourAgo.Format(time.RFC3339), Type: "action_ratio", Value: 0.7},
			},
			since: twoHoursAgo,
			want: map[string]MetricSummary{
				"action_ratio": {
					Type:         "action_ratio",
					Count:        1,
					LatestValue:  0.7,
					AverageValue: 0.7,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateByType(tt.metrics, tt.since)

			if len(got) != len(tt.want) {
				t.Errorf("AggregateByType() got %d types, want %d", len(got), len(tt.want))
			}

			for typ, want := range tt.want {
				summary, exists := got[typ]
				if !exists {
					t.Errorf("AggregateByType() missing type %s", typ)
					continue
				}

				if summary.Type != want.Type {
					t.Errorf("AggregateByType()[%s].Type = %v, want %v", typ, summary.Type, want.Type)
				}
				if summary.Count != want.Count {
					t.Errorf("AggregateByType()[%s].Count = %v, want %v", typ, summary.Count, want.Count)
				}
				if summary.LatestValue != want.LatestValue {
					t.Errorf("AggregateByType()[%s].LatestValue = %v, want %v", typ, summary.LatestValue, want.LatestValue)
				}
				if summary.AverageValue != want.AverageValue {
					t.Errorf("AggregateByType()[%s].AverageValue = %v, want %v", typ, summary.AverageValue, want.AverageValue)
				}
			}
		})
	}
}

func TestFormatTextSummary(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		summary    map[string]MetricSummary
		wantOutput []string
	}{
		{
			name: "format single metric",
			summary: map[string]MetricSummary{
				"action_ratio": {
					Type:         "action_ratio",
					Count:        5,
					LatestValue:  0.7,
					AverageValue: 0.65,
					LastSeen:     now.Add(-10 * time.Minute),
				},
			},
			wantOutput: []string{"action_ratio:", "5 events", "10m ago", "0.65"},
		},
		{
			name: "format multiple metrics sorted",
			summary: map[string]MetricSummary{
				"z_metric": {
					Type:         "z_metric",
					Count:        1,
					AverageValue: 1.0,
					LastSeen:     now.Add(-1 * time.Hour),
				},
				"a_metric": {
					Type:         "a_metric",
					Count:        2,
					AverageValue: 2.0,
					LastSeen:     now.Add(-30 * time.Minute),
				},
			},
			wantOutput: []string{"a_metric:", "z_metric:"},
		},
		{
			name:       "empty summary",
			summary:    map[string]MetricSummary{},
			wantOutput: []string{"No metrics found"},
		},
		{
			name: "recent metric shows 'just now'",
			summary: map[string]MetricSummary{
				"recent": {
					Type:         "recent",
					Count:        1,
					AverageValue: 1.0,
					LastSeen:     now.Add(-30 * time.Second),
				},
			},
			wantOutput: []string{"recent:", "just now"},
		},
		{
			name: "old metric shows days",
			summary: map[string]MetricSummary{
				"old": {
					Type:         "old",
					Count:        1,
					AverageValue: 1.0,
					LastSeen:     now.Add(-48 * time.Hour),
				},
			},
			wantOutput: []string{"old:", "2d ago"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTextSummary(tt.summary)

			for _, want := range tt.wantOutput {
				if !strings.Contains(got, want) {
					t.Errorf("FormatTextSummary() output missing '%s'\nGot: %s", want, got)
				}
			}
		})
	}
}

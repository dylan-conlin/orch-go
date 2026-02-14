package coaching

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// Metric represents a single behavioral metric entry.
type Metric struct {
	Timestamp string                 `json:"timestamp"`
	SessionID string                 `json:"session_id,omitempty"`
	Type      string                 `json:"metric_type"`
	Value     float64                `json:"value"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// MetricSummary represents aggregated statistics for a metric type.
type MetricSummary struct {
	Type         string    `json:"type"`
	Count        int       `json:"count"`
	LatestValue  float64   `json:"latest_value"`
	AverageValue float64   `json:"average_value"`
	LastSeen     time.Time `json:"last_seen"`
}

// ReadMetrics reads metrics from a JSONL file, returning the last N entries.
// If the file doesn't exist, returns an empty slice.
func ReadMetrics(path string, limit int) ([]Metric, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Metric{}, nil // No metrics yet
		}
		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	var metrics []Metric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var metric Metric
		if err := json.Unmarshal([]byte(line), &metric); err != nil {
			// Skip malformed lines
			continue
		}

		metrics = append(metrics, metric)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading metrics: %w", err)
	}

	// Return last N lines
	if len(metrics) > limit {
		metrics = metrics[len(metrics)-limit:]
	}

	return metrics, nil
}

// ReadMetricsSince reads all metrics from a JSONL file that occurred after the given time.
// If the file doesn't exist, returns an empty slice.
func ReadMetricsSince(path string, since time.Time) ([]Metric, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Metric{}, nil // No metrics yet
		}
		return nil, fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	var metrics []Metric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var metric Metric
		if err := json.Unmarshal([]byte(line), &metric); err != nil {
			// Skip malformed lines
			continue
		}

		// Parse timestamp and filter by time
		metricTime, err := time.Parse(time.RFC3339, metric.Timestamp)
		if err != nil {
			// Skip metrics with invalid timestamps
			continue
		}

		if metricTime.After(since) {
			metrics = append(metrics, metric)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading metrics: %w", err)
	}

	return metrics, nil
}

// AggregateByType aggregates metrics by type, calculating count, average, and latest value.
// Only metrics after the given time are included.
func AggregateByType(metrics []Metric, since time.Time) map[string]MetricSummary {
	result := make(map[string]MetricSummary)

	for _, m := range metrics {
		// Parse timestamp to filter by time
		metricTime, err := time.Parse(time.RFC3339, m.Timestamp)
		if err != nil {
			// Skip metrics with invalid timestamps
			continue
		}

		if !metricTime.After(since) {
			continue
		}

		// Get or create summary for this type
		summary, exists := result[m.Type]
		if !exists {
			summary = MetricSummary{
				Type: m.Type,
			}
		}

		// Update aggregations
		summary.Count++
		summary.LatestValue = m.Value
		summary.AverageValue = ((summary.AverageValue * float64(summary.Count-1)) + m.Value) / float64(summary.Count)

		// Update last seen time
		if metricTime.After(summary.LastSeen) {
			summary.LastSeen = metricTime
		}

		result[m.Type] = summary
	}

	return result
}

// FormatTextSummary formats metric summaries as human-readable text for CLI output.
func FormatTextSummary(summary map[string]MetricSummary) string {
	if len(summary) == 0 {
		return "No metrics found"
	}

	// Sort metric types for consistent output
	var types []string
	for t := range summary {
		types = append(types, t)
	}
	sort.Strings(types)

	var output string
	for _, t := range types {
		s := summary[t]
		timeSince := time.Since(s.LastSeen)
		var timeStr string
		if timeSince < time.Minute {
			timeStr = "just now"
		} else if timeSince < time.Hour {
			timeStr = fmt.Sprintf("%dm ago", int(timeSince.Minutes()))
		} else if timeSince < 24*time.Hour {
			timeStr = fmt.Sprintf("%dh ago", int(timeSince.Hours()))
		} else {
			timeStr = fmt.Sprintf("%dd ago", int(timeSince.Hours()/24))
		}

		output += fmt.Sprintf("  %-25s %d events (last: %s, avg: %.2f)\n",
			t+":", s.Count, timeStr, s.AverageValue)
	}

	return output
}

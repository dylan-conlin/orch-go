package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TrialResult captures the outcome of a single scenario trial.
type TrialResult struct {
	Scenario   string        `json:"scenario"`
	Trial      int           `json:"trial"`
	Model      string        `json:"model,omitempty"`
	Tier       string        `json:"tier,omitempty"` // "light" or "full" — from scenario config
	BeadsID    string        `json:"beads_id,omitempty"`
	Status     string        `json:"status"` // "pass", "fail", "error", "timeout"
	Reworks    int           `json:"reworks"`
	Duration   time.Duration `json:"duration_ns"`
	DurationS  float64       `json:"duration_s"`
	EvalOutput string        `json:"eval_output,omitempty"`
	Error      string        `json:"error,omitempty"`
	StartedAt  time.Time     `json:"started_at"`
}

// RunResult captures the outcome of a full benchmark run.
type RunResult struct {
	Name      string        `json:"name"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration_ns"`
	DurationS float64       `json:"duration_s"`
	Trials    []TrialResult `json:"trials"`
}

// Summary returns aggregate statistics for the benchmark run.
func (r *RunResult) Summary() RunSummary {
	s := RunSummary{Name: r.Name, Total: len(r.Trials)}
	var totalDur time.Duration
	for _, t := range r.Trials {
		totalDur += t.Duration
		switch t.Status {
		case "pass":
			s.Passed++
		case "fail":
			s.Failed++
		case "error":
			s.Errors++
		case "timeout":
			s.Timeouts++
		}
		s.TotalReworks += t.Reworks
	}
	if s.Total > 0 {
		s.PassRate = float64(s.Passed) / float64(s.Total)
		s.AvgDuration = totalDur / time.Duration(s.Total)
	}
	return s
}

// RunSummary provides aggregate statistics.
type RunSummary struct {
	Name         string        `json:"name"`
	Total        int           `json:"total"`
	Passed       int           `json:"passed"`
	Failed       int           `json:"failed"`
	Errors       int           `json:"errors"`
	Timeouts     int           `json:"timeouts"`
	PassRate     float64       `json:"pass_rate"`
	TotalReworks int           `json:"total_reworks"`
	AvgDuration  time.Duration `json:"avg_duration_ns"`
}

// WriteJSONL writes trial results as JSONL to the given path.
func (r *RunResult) WriteJSONL(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating results file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, trial := range r.Trials {
		if err := enc.Encode(trial); err != nil {
			return fmt.Errorf("encoding trial result: %w", err)
		}
	}
	return nil
}

// WriteSummary writes the summary as JSON to the given path.
func (r *RunResult) WriteSummary(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}

	summary := r.Summary()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling summary: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

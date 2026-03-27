package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RunMetadata captures context for resuming or comparing benchmark runs.
type RunMetadata struct {
	RunID     string    `json:"run_id"`
	GitSHA    string    `json:"git_sha,omitempty"`
	GitBranch string    `json:"git_branch,omitempty"`
	StartedAt time.Time `json:"started_at"`
	ConfigRef string    `json:"config_ref,omitempty"` // path to config snapshot in run dir
}

// ModelSummary provides per-model aggregate statistics.
type ModelSummary struct {
	Model        string        `json:"model"`
	Total        int           `json:"total"`
	Passed       int           `json:"passed"`
	Failed       int           `json:"failed"`
	Errors       int           `json:"errors"`
	Timeouts     int           `json:"timeouts"`
	PassRate     float64       `json:"pass_rate"`
	TotalReworks int           `json:"total_reworks"`
	AvgDuration  time.Duration `json:"avg_duration_ns"`
	AvgDurationS float64       `json:"avg_duration_s"`
}

// ScenarioSummary provides per-scenario aggregate statistics.
type ScenarioSummary struct {
	Scenario     string        `json:"scenario"`
	Model        string        `json:"model,omitempty"`
	Total        int           `json:"total"`
	Passed       int           `json:"passed"`
	Failed       int           `json:"failed"`
	Errors       int           `json:"errors"`
	Timeouts     int           `json:"timeouts"`
	PassRate     float64       `json:"pass_rate"`
	TotalReworks int           `json:"total_reworks"`
	AvgDuration  time.Duration `json:"avg_duration_ns"`
	AvgDurationS float64       `json:"avg_duration_s"`
}

// ComplianceSignals captures protocol-compliance metrics derived from trial results.
type ComplianceSignals struct {
	FirstPassRate      float64 `json:"first_pass_rate"`       // passed on first attempt (no reworks)
	ReworkRecoveryRate float64 `json:"rework_recovery_rate"`  // of trials that reworked, how many recovered
	StallRate          float64 `json:"stall_rate"`            // timeout rate
	ErrorRate          float64 `json:"error_rate"`            // infrastructure error rate
	ReworkRate         float64 `json:"rework_rate"`           // fraction that needed any rework
}

// VerdictSignal captures the evaluation of a single threshold.
type VerdictSignal struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Status    string  `json:"status"` // "pass", "fail", "warn"
	Message   string  `json:"message"`
}

// Verdict captures the threshold-based evaluation of a benchmark run.
type Verdict struct {
	Overall string          `json:"overall"` // "PASS", "FAIL", "WARN"
	Signals []VerdictSignal `json:"signals"`
}

// Report is the top-level benchmark report combining all artifacts.
type Report struct {
	Metadata          RunMetadata                  `json:"metadata"`
	Summary           RunSummary                   `json:"summary"`
	ModelSummaries    []ModelSummary               `json:"model_summaries"`
	ScenarioSummaries []ScenarioSummary            `json:"scenario_summaries"`
	Compliance        ComplianceSignals            `json:"compliance"`
	TierCompliance    map[string]ComplianceSignals `json:"tier_compliance,omitempty"`
	Verdict           Verdict                      `json:"verdict"`
	Thresholds        Thresholds                   `json:"thresholds"`
}

// ModelSummaries computes per-model aggregate statistics from run results.
func ModelSummaries(r *RunResult) []ModelSummary {
	groups := map[string][]TrialResult{}
	for _, t := range r.Trials {
		model := t.Model
		if model == "" {
			model = "(default)"
		}
		groups[model] = append(groups[model], t)
	}

	var summaries []ModelSummary
	for model, trials := range groups {
		s := ModelSummary{Model: model, Total: len(trials)}
		var totalDur time.Duration
		for _, t := range trials {
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
			s.AvgDurationS = s.AvgDuration.Seconds()
		}
		summaries = append(summaries, s)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Model < summaries[j].Model
	})
	return summaries
}

// ScenarioSummaries computes per-scenario aggregate statistics from run results.
func ScenarioSummaries(r *RunResult) []ScenarioSummary {
	groups := map[string][]TrialResult{}
	modelByScenario := map[string]string{}
	for _, t := range r.Trials {
		groups[t.Scenario] = append(groups[t.Scenario], t)
		if t.Model != "" {
			modelByScenario[t.Scenario] = t.Model
		}
	}

	var summaries []ScenarioSummary
	for scenario, trials := range groups {
		s := ScenarioSummary{
			Scenario: scenario,
			Model:    modelByScenario[scenario],
			Total:    len(trials),
		}
		var totalDur time.Duration
		for _, t := range trials {
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
			s.AvgDurationS = s.AvgDuration.Seconds()
		}
		summaries = append(summaries, s)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Scenario < summaries[j].Scenario
	})
	return summaries
}

// ComputeCompliance derives protocol-compliance signals from trial results.
func ComputeCompliance(r *RunResult) ComplianceSignals {
	total := len(r.Trials)
	if total == 0 {
		return ComplianceSignals{}
	}

	var firstPass, reworked, reworkRecovered, timeouts, errors int
	for _, t := range r.Trials {
		if t.Status == "pass" && t.Reworks == 0 {
			firstPass++
		}
		if t.Reworks > 0 {
			reworked++
			if t.Status == "pass" {
				reworkRecovered++
			}
		}
		if t.Status == "timeout" {
			timeouts++
		}
		if t.Status == "error" {
			errors++
		}
	}

	s := ComplianceSignals{
		FirstPassRate: float64(firstPass) / float64(total),
		StallRate:     float64(timeouts) / float64(total),
		ErrorRate:     float64(errors) / float64(total),
		ReworkRate:    float64(reworked) / float64(total),
	}
	if reworked > 0 {
		s.ReworkRecoveryRate = float64(reworkRecovered) / float64(reworked)
	}
	return s
}

// ComputeComplianceByTier derives compliance signals partitioned by spawn tier.
// Trials with no tier set are grouped under "(untiered)".
func ComputeComplianceByTier(r *RunResult) map[string]ComplianceSignals {
	groups := map[string][]TrialResult{}
	for _, t := range r.Trials {
		tier := t.Tier
		if tier == "" {
			tier = "(untiered)"
		}
		groups[tier] = append(groups[tier], t)
	}

	result := make(map[string]ComplianceSignals, len(groups))
	for tier, trials := range groups {
		sub := &RunResult{Trials: trials}
		result[tier] = ComputeCompliance(sub)
	}
	return result
}

// EvaluateVerdict checks run results against configurable thresholds.
func EvaluateVerdict(r *RunResult, th Thresholds) Verdict {
	summary := r.Summary()
	compliance := ComputeCompliance(r)

	var signals []VerdictSignal
	hasFail := false
	hasWarn := false

	// Pass rate check
	passSignal := VerdictSignal{
		Name:      "pass_rate",
		Value:     summary.PassRate,
		Threshold: th.PassRate,
	}
	if summary.PassRate >= th.PassRate {
		passSignal.Status = "pass"
		passSignal.Message = fmt.Sprintf("pass rate %.0f%% >= %.0f%% threshold", summary.PassRate*100, th.PassRate*100)
	} else {
		passSignal.Status = "fail"
		passSignal.Message = fmt.Sprintf("pass rate %.0f%% < %.0f%% threshold", summary.PassRate*100, th.PassRate*100)
		hasFail = true
	}
	signals = append(signals, passSignal)

	// Error rate check
	errorSignal := VerdictSignal{
		Name:      "error_rate",
		Value:     compliance.ErrorRate,
		Threshold: th.MaxErrorRate,
	}
	if compliance.ErrorRate <= th.MaxErrorRate {
		errorSignal.Status = "pass"
		errorSignal.Message = fmt.Sprintf("error rate %.0f%% <= %.0f%% threshold", compliance.ErrorRate*100, th.MaxErrorRate*100)
	} else {
		errorSignal.Status = "fail"
		errorSignal.Message = fmt.Sprintf("error rate %.0f%% > %.0f%% threshold", compliance.ErrorRate*100, th.MaxErrorRate*100)
		hasFail = true
	}
	signals = append(signals, errorSignal)

	// Rework rate check (warning only)
	reworkSignal := VerdictSignal{
		Name:      "rework_rate",
		Value:     compliance.ReworkRate,
		Threshold: th.MaxReworkRate,
	}
	if compliance.ReworkRate <= th.MaxReworkRate {
		reworkSignal.Status = "pass"
		reworkSignal.Message = fmt.Sprintf("rework rate %.0f%% <= %.0f%% threshold", compliance.ReworkRate*100, th.MaxReworkRate*100)
	} else {
		reworkSignal.Status = "warn"
		reworkSignal.Message = fmt.Sprintf("rework rate %.0f%% > %.0f%% threshold", compliance.ReworkRate*100, th.MaxReworkRate*100)
		hasWarn = true
	}
	signals = append(signals, reworkSignal)

	overall := "PASS"
	if hasFail {
		overall = "FAIL"
	} else if hasWarn {
		overall = "WARN"
	}

	return Verdict{Overall: overall, Signals: signals}
}

// GenerateReport builds a complete benchmark report from run results.
func GenerateReport(r *RunResult, cfg *Config, meta RunMetadata) *Report {
	tierCompliance := ComputeComplianceByTier(r)
	// Omit tier_compliance if all trials are untiered (no useful partition).
	if len(tierCompliance) == 1 {
		if _, ok := tierCompliance["(untiered)"]; ok {
			tierCompliance = nil
		}
	}
	return &Report{
		Metadata:          meta,
		Summary:           r.Summary(),
		ModelSummaries:    ModelSummaries(r),
		ScenarioSummaries: ScenarioSummaries(r),
		Compliance:        ComputeCompliance(r),
		TierCompliance:    tierCompliance,
		Verdict:           EvaluateVerdict(r, cfg.Thresholds),
		Thresholds:        cfg.Thresholds,
	}
}

// WriteReport writes the report as JSON to the given path.
func WriteReport(report *Report, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating report dir: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// WriteConfigSnapshot writes a YAML snapshot of the config to the given path.
func WriteConfigSnapshot(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// FormatReport returns a human-readable text summary of the report.
func FormatReport(report *Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "═══ Benchmark Report: %s ═══\n", report.Summary.Name)
	fmt.Fprintf(&b, "Run: %s  Branch: %s  SHA: %s\n\n", report.Metadata.RunID, report.Metadata.GitBranch, report.Metadata.GitSHA)

	// Overall summary
	fmt.Fprintf(&b, "── Summary ──\n")
	fmt.Fprintf(&b, "  Total: %d  Pass: %d  Fail: %d  Error: %d  Timeout: %d\n",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed, report.Summary.Errors, report.Summary.Timeouts)
	fmt.Fprintf(&b, "  Pass rate: %.0f%%  Reworks: %d\n\n",
		report.Summary.PassRate*100, report.Summary.TotalReworks)

	// Per-model breakdown
	if len(report.ModelSummaries) > 0 {
		fmt.Fprintf(&b, "── Per-Model Pass Rates ──\n")
		for _, m := range report.ModelSummaries {
			fmt.Fprintf(&b, "  %-12s  %d/%d (%.0f%%)  reworks: %d  avg: %s\n",
				m.Model, m.Passed, m.Total, m.PassRate*100, m.TotalReworks, m.AvgDuration.Round(time.Second))
		}
		b.WriteString("\n")
	}

	// Per-scenario breakdown
	if len(report.ScenarioSummaries) > 0 {
		fmt.Fprintf(&b, "── Per-Scenario Pass Rates ──\n")
		for _, s := range report.ScenarioSummaries {
			modelTag := ""
			if s.Model != "" {
				modelTag = fmt.Sprintf(" [%s]", s.Model)
			}
			fmt.Fprintf(&b, "  %-20s%s  %d/%d (%.0f%%)\n",
				s.Scenario, modelTag, s.Passed, s.Total, s.PassRate*100)
		}
		b.WriteString("\n")
	}

	// Compliance signals
	fmt.Fprintf(&b, "── Compliance Signals ──\n")
	fmt.Fprintf(&b, "  First-pass rate:      %.0f%%\n", report.Compliance.FirstPassRate*100)
	fmt.Fprintf(&b, "  Rework recovery rate: %.0f%%\n", report.Compliance.ReworkRecoveryRate*100)
	fmt.Fprintf(&b, "  Stall rate:           %.0f%%\n", report.Compliance.StallRate*100)
	fmt.Fprintf(&b, "  Error rate:           %.0f%%\n", report.Compliance.ErrorRate*100)
	fmt.Fprintf(&b, "  Rework rate:          %.0f%%\n", report.Compliance.ReworkRate*100)

	// Per-tier compliance breakdown
	if len(report.TierCompliance) > 0 {
		// Sort tier names for deterministic output
		tiers := make([]string, 0, len(report.TierCompliance))
		for tier := range report.TierCompliance {
			tiers = append(tiers, tier)
		}
		sort.Strings(tiers)

		b.WriteString("\n── Per-Tier Compliance ──\n")
		for _, tier := range tiers {
			tc := report.TierCompliance[tier]
			fmt.Fprintf(&b, "  [%s] first-pass: %.0f%%  stall: %.0f%%  error: %.0f%%  rework: %.0f%%\n",
				tier, tc.FirstPassRate*100, tc.StallRate*100, tc.ErrorRate*100, tc.ReworkRate*100)
		}
	}
	b.WriteString("\n")

	// Verdict
	fmt.Fprintf(&b, "── Verdict: %s ──\n", report.Verdict.Overall)
	for _, s := range report.Verdict.Signals {
		icon := "✓"
		if s.Status == "fail" {
			icon = "✗"
		} else if s.Status == "warn" {
			icon = "!"
		}
		fmt.Fprintf(&b, "  %s %s\n", icon, s.Message)
	}

	return b.String()
}

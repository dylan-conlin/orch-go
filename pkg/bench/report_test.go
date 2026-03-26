package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sampleRunResult() *RunResult {
	return &RunResult{
		Name:      "reliability-suite",
		StartedAt: time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC),
		Duration:  5 * time.Minute,
		DurationS: 300.0,
		Trials: []TrialResult{
			{Scenario: "feature-add", Trial: 1, Model: "opus", Status: "pass", Duration: 60 * time.Second, Reworks: 0},
			{Scenario: "feature-add", Trial: 2, Model: "opus", Status: "pass", Duration: 90 * time.Second, Reworks: 1},
			{Scenario: "feature-add", Trial: 3, Model: "opus", Status: "fail", Duration: 120 * time.Second, Reworks: 2},
			{Scenario: "bug-fix", Trial: 1, Model: "sonnet", Status: "pass", Duration: 30 * time.Second, Reworks: 0},
			{Scenario: "bug-fix", Trial: 2, Model: "sonnet", Status: "timeout", Duration: 180 * time.Second, Reworks: 0},
			{Scenario: "bug-fix", Trial: 3, Model: "sonnet", Status: "error", Duration: 5 * time.Second, Reworks: 0, Error: "spawn failed"},
		},
	}
}

func TestModelSummaries(t *testing.T) {
	r := sampleRunResult()
	summaries := ModelSummaries(r)

	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}

	// Find opus summary
	var opus, sonnet *ModelSummary
	for i := range summaries {
		switch summaries[i].Model {
		case "opus":
			opus = &summaries[i]
		case "sonnet":
			sonnet = &summaries[i]
		}
	}

	if opus == nil {
		t.Fatal("missing opus summary")
	}
	if opus.Total != 3 {
		t.Errorf("opus.Total = %d, want 3", opus.Total)
	}
	if opus.Passed != 2 {
		t.Errorf("opus.Passed = %d, want 2", opus.Passed)
	}
	if opus.Failed != 1 {
		t.Errorf("opus.Failed = %d, want 1", opus.Failed)
	}
	wantRate := 2.0 / 3.0
	if opus.PassRate != wantRate {
		t.Errorf("opus.PassRate = %f, want %f", opus.PassRate, wantRate)
	}
	if opus.TotalReworks != 3 {
		t.Errorf("opus.TotalReworks = %d, want 3", opus.TotalReworks)
	}

	if sonnet == nil {
		t.Fatal("missing sonnet summary")
	}
	if sonnet.Total != 3 {
		t.Errorf("sonnet.Total = %d, want 3", sonnet.Total)
	}
	if sonnet.Passed != 1 {
		t.Errorf("sonnet.Passed = %d, want 1", sonnet.Passed)
	}
	if sonnet.Timeouts != 1 {
		t.Errorf("sonnet.Timeouts = %d, want 1", sonnet.Timeouts)
	}
	if sonnet.Errors != 1 {
		t.Errorf("sonnet.Errors = %d, want 1", sonnet.Errors)
	}
}

func TestModelSummaries_EmptyModel(t *testing.T) {
	r := &RunResult{
		Name: "no-model",
		Trials: []TrialResult{
			{Scenario: "a", Trial: 1, Model: "", Status: "pass", Duration: time.Second},
		},
	}
	summaries := ModelSummaries(r)
	if len(summaries) != 1 {
		t.Fatalf("len = %d, want 1", len(summaries))
	}
	if summaries[0].Model != "(default)" {
		t.Errorf("Model = %q, want %q", summaries[0].Model, "(default)")
	}
}

func TestScenarioSummaries(t *testing.T) {
	r := sampleRunResult()
	summaries := ScenarioSummaries(r)

	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}

	var featureAdd, bugFix *ScenarioSummary
	for i := range summaries {
		switch summaries[i].Scenario {
		case "feature-add":
			featureAdd = &summaries[i]
		case "bug-fix":
			bugFix = &summaries[i]
		}
	}

	if featureAdd == nil {
		t.Fatal("missing feature-add summary")
	}
	if featureAdd.Total != 3 {
		t.Errorf("feature-add.Total = %d, want 3", featureAdd.Total)
	}
	if featureAdd.Passed != 2 {
		t.Errorf("feature-add.Passed = %d, want 2", featureAdd.Passed)
	}

	if bugFix == nil {
		t.Fatal("missing bug-fix summary")
	}
	if bugFix.Total != 3 {
		t.Errorf("bug-fix.Total = %d, want 3", bugFix.Total)
	}
	if bugFix.Passed != 1 {
		t.Errorf("bug-fix.Passed = %d, want 1", bugFix.Passed)
	}
}

func TestComplianceSignals(t *testing.T) {
	r := sampleRunResult()
	signals := ComputeCompliance(r)

	// 3 of 6 passed on first attempt (no reworks): feature-add t1, bug-fix t1, bug-fix t2 (timeout), bug-fix t3 (error)
	// Actually: first_pass = pass AND reworks == 0 → feature-add t1, bug-fix t1 = 2 out of 6
	wantFirstPass := 2.0 / 6.0
	if signals.FirstPassRate != wantFirstPass {
		t.Errorf("FirstPassRate = %f, want %f", signals.FirstPassRate, wantFirstPass)
	}

	// Rework recovery: trials that had reworks > 0 and ended pass
	// feature-add t2 (reworks=1, pass) out of feature-add t2 + t3 (reworks>0) = 1/2
	wantRecovery := 1.0 / 2.0
	if signals.ReworkRecoveryRate != wantRecovery {
		t.Errorf("ReworkRecoveryRate = %f, want %f", signals.ReworkRecoveryRate, wantRecovery)
	}

	// Stall rate: timeouts / total = 1/6
	wantStall := 1.0 / 6.0
	if signals.StallRate != wantStall {
		t.Errorf("StallRate = %f, want %f", signals.StallRate, wantStall)
	}

	// Error rate: errors / total = 1/6
	wantError := 1.0 / 6.0
	if signals.ErrorRate != wantError {
		t.Errorf("ErrorRate = %f, want %f", signals.ErrorRate, wantError)
	}

	// Rework rate: trials that needed rework / total = 2/6
	wantReworkRate := 2.0 / 6.0
	if signals.ReworkRate != wantReworkRate {
		t.Errorf("ReworkRate = %f, want %f", signals.ReworkRate, wantReworkRate)
	}
}

func TestComplianceSignals_NoTrials(t *testing.T) {
	r := &RunResult{Name: "empty"}
	signals := ComputeCompliance(r)
	if signals.FirstPassRate != 0 {
		t.Errorf("FirstPassRate = %f, want 0", signals.FirstPassRate)
	}
	if signals.ReworkRecoveryRate != 0 {
		t.Errorf("ReworkRecoveryRate = %f, want 0", signals.ReworkRecoveryRate)
	}
}

func TestEvaluateVerdict_Pass(t *testing.T) {
	th := Thresholds{PassRate: 0.5, MaxErrorRate: 0.2, MaxReworkRate: 0.5}
	r := &RunResult{
		Name: "good",
		Trials: []TrialResult{
			{Status: "pass", Reworks: 0},
			{Status: "pass", Reworks: 0},
			{Status: "fail", Reworks: 1},
		},
	}
	v := EvaluateVerdict(r, th)
	if v.Overall != "PASS" {
		t.Errorf("Overall = %q, want PASS", v.Overall)
	}
}

func TestEvaluateVerdict_Fail_LowPassRate(t *testing.T) {
	th := Thresholds{PassRate: 0.8, MaxErrorRate: 0.2, MaxReworkRate: 0.5}
	r := &RunResult{
		Name: "bad",
		Trials: []TrialResult{
			{Status: "pass"},
			{Status: "fail"},
			{Status: "fail"},
			{Status: "fail"},
		},
	}
	v := EvaluateVerdict(r, th)
	if v.Overall != "FAIL" {
		t.Errorf("Overall = %q, want FAIL", v.Overall)
	}

	// Should have a signal for pass_rate
	found := false
	for _, s := range v.Signals {
		if s.Name == "pass_rate" && s.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Error("expected pass_rate signal with status=fail")
	}
}

func TestEvaluateVerdict_Fail_HighErrorRate(t *testing.T) {
	th := Thresholds{PassRate: 0.1, MaxErrorRate: 0.1, MaxReworkRate: 0.5}
	r := &RunResult{
		Name: "errors",
		Trials: []TrialResult{
			{Status: "pass"},
			{Status: "error"},
			{Status: "error"},
		},
	}
	v := EvaluateVerdict(r, th)
	if v.Overall != "FAIL" {
		t.Errorf("Overall = %q, want FAIL", v.Overall)
	}
}

func TestEvaluateVerdict_Warn_HighReworkRate(t *testing.T) {
	th := Thresholds{PassRate: 0.5, MaxErrorRate: 0.5, MaxReworkRate: 0.3}
	r := &RunResult{
		Name: "rework-heavy",
		Trials: []TrialResult{
			{Status: "pass", Reworks: 0},
			{Status: "pass", Reworks: 1},
			{Status: "pass", Reworks: 2},
		},
	}
	v := EvaluateVerdict(r, th)
	if v.Overall != "WARN" {
		t.Errorf("Overall = %q, want WARN", v.Overall)
	}
}

func TestGenerateReport(t *testing.T) {
	r := sampleRunResult()
	cfg := &Config{
		Name:       "reliability-suite",
		Trials:     3,
		Parallel:   2,
		Thresholds: Thresholds{PassRate: 0.5, MaxErrorRate: 0.2, MaxReworkRate: 0.5},
		Scenarios: []Scenario{
			{Name: "feature-add", Skill: "feature-impl", Task: "add endpoint", Eval: "go test ./...", Model: "opus"},
			{Name: "bug-fix", Skill: "systematic-debugging", Task: "fix bug", Eval: "make test", Model: "sonnet"},
		},
	}
	meta := RunMetadata{
		RunID:     "20260326-100000",
		GitSHA:    "abc123",
		GitBranch: "master",
		StartedAt: r.StartedAt,
	}

	report := GenerateReport(r, cfg, meta)

	if report.Metadata.RunID != "20260326-100000" {
		t.Errorf("RunID = %q", report.Metadata.RunID)
	}
	if report.Metadata.GitSHA != "abc123" {
		t.Errorf("GitSHA = %q", report.Metadata.GitSHA)
	}
	if len(report.ModelSummaries) != 2 {
		t.Errorf("ModelSummaries = %d, want 2", len(report.ModelSummaries))
	}
	if len(report.ScenarioSummaries) != 2 {
		t.Errorf("ScenarioSummaries = %d, want 2", len(report.ScenarioSummaries))
	}
	if report.Verdict.Overall == "" {
		t.Error("Verdict.Overall is empty")
	}
	if report.Compliance.FirstPassRate == 0 && report.Compliance.ErrorRate == 0 {
		t.Error("Compliance signals all zero")
	}
}

func TestWriteReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	report := &Report{
		Metadata: RunMetadata{RunID: "test-run"},
		Summary:  RunSummary{Name: "test", Total: 1, Passed: 1, PassRate: 1.0},
		Verdict:  Verdict{Overall: "PASS"},
	}

	if err := WriteReport(report, path); err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	var loaded Report
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if loaded.Metadata.RunID != "test-run" {
		t.Errorf("RunID = %q", loaded.Metadata.RunID)
	}
	if loaded.Verdict.Overall != "PASS" {
		t.Errorf("Verdict = %q", loaded.Verdict.Overall)
	}
}

func TestFormatReport(t *testing.T) {
	report := &Report{
		Metadata: RunMetadata{RunID: "20260326-100000", GitSHA: "abc123", GitBranch: "master"},
		Summary:  RunSummary{Name: "test", Total: 6, Passed: 3, Failed: 1, Errors: 1, Timeouts: 1, PassRate: 0.5, TotalReworks: 3},
		ModelSummaries: []ModelSummary{
			{Model: "opus", Total: 3, Passed: 2, PassRate: 0.667},
			{Model: "sonnet", Total: 3, Passed: 1, PassRate: 0.333},
		},
		Compliance: ComplianceSignals{FirstPassRate: 0.333, ReworkRecoveryRate: 0.5, StallRate: 0.167, ErrorRate: 0.167, ReworkRate: 0.333},
		Verdict: Verdict{
			Overall: "WARN",
			Signals: []VerdictSignal{
				{Name: "pass_rate", Value: 0.5, Threshold: 0.5, Status: "pass"},
				{Name: "error_rate", Value: 0.167, Threshold: 0.2, Status: "pass"},
				{Name: "rework_rate", Value: 0.333, Threshold: 0.3, Status: "warn"},
			},
		},
	}

	output := FormatReport(report)

	// Should contain key sections
	for _, want := range []string{"Benchmark Report", "opus", "sonnet", "WARN", "Compliance", "Verdict"} {
		if !containsString(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestWriteConfigSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Name:       "test-suite",
		Trials:     3,
		Parallel:   2,
		Thresholds: Thresholds{PassRate: 0.8},
		Scenarios: []Scenario{
			{Name: "a", Skill: "s", Task: "t", Eval: "e", Timeout: "30m"},
		},
	}

	if err := WriteConfigSnapshot(cfg, path); err != nil {
		t.Fatalf("WriteConfigSnapshot failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if len(data) == 0 {
		t.Error("config snapshot is empty")
	}

	// Should round-trip
	loaded, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("ParseConfig on snapshot: %v", err)
	}
	if loaded.Name != "test-suite" {
		t.Errorf("Name = %q", loaded.Name)
	}
}

func containsString(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && // avoid trivial matches
		findSubstring(s, sub)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

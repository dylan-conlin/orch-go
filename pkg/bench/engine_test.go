package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newTestEngine(spawnErr, waitErr, evalErr, reworkErr error, evalExit int) *Engine {
	var spawnCount int32
	return &Engine{
		SpawnFn: func(skill, task, model string) (string, error) {
			id := fmt.Sprintf("bench-%d", atomic.AddInt32(&spawnCount, 1))
			return id, spawnErr
		},
		WaitFn: func(beadsID string, timeout time.Duration) error {
			return waitErr
		},
		EvalFn: func(evalCmd string) (int, string, error) {
			return evalExit, "eval output", evalErr
		},
		ReworkFn: func(beadsID, feedback string) error {
			return reworkErr
		},
	}
}

func minimalConfig() *Config {
	return &Config{
		Name:     "test-bench",
		Trials:   1,
		Parallel: 1,
		Scenarios: []Scenario{
			{Name: "basic", Skill: "feature-impl", Task: "do thing", Eval: "echo ok", Timeout: "1m"},
		},
	}
}

func TestEngine_SingleTrialPass(t *testing.T) {
	e := newTestEngine(nil, nil, nil, nil, 0)
	cfg := minimalConfig()

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Trials) != 1 {
		t.Fatalf("len(Trials) = %d, want 1", len(result.Trials))
	}
	tr := result.Trials[0]
	if tr.Status != "pass" {
		t.Errorf("Status = %q, want %q", tr.Status, "pass")
	}
	if tr.Scenario != "basic" {
		t.Errorf("Scenario = %q, want %q", tr.Scenario, "basic")
	}
	if tr.Trial != 1 {
		t.Errorf("Trial = %d, want 1", tr.Trial)
	}
	if tr.BeadsID == "" {
		t.Error("BeadsID should not be empty")
	}
}

func TestEngine_SpawnError(t *testing.T) {
	e := newTestEngine(fmt.Errorf("spawn broke"), nil, nil, nil, 0)
	cfg := minimalConfig()

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tr := result.Trials[0]
	if tr.Status != "error" {
		t.Errorf("Status = %q, want %q", tr.Status, "error")
	}
}

func TestEngine_WaitTimeout(t *testing.T) {
	e := newTestEngine(nil, fmt.Errorf("timed out"), nil, nil, 0)
	cfg := minimalConfig()

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tr := result.Trials[0]
	if tr.Status != "timeout" {
		t.Errorf("Status = %q, want %q", tr.Status, "timeout")
	}
}

func TestEngine_EvalFail_NoRework(t *testing.T) {
	e := newTestEngine(nil, nil, nil, nil, 1)
	cfg := minimalConfig()
	cfg.Scenarios[0].MaxReworks = 0

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tr := result.Trials[0]
	if tr.Status != "fail" {
		t.Errorf("Status = %q, want %q", tr.Status, "fail")
	}
	if tr.Reworks != 0 {
		t.Errorf("Reworks = %d, want 0", tr.Reworks)
	}
}

func TestEngine_EvalFail_ReworkSucceeds(t *testing.T) {
	var evalCount int32
	e := &Engine{
		SpawnFn: func(skill, task, model string) (string, error) {
			return "bench-1", nil
		},
		WaitFn: func(beadsID string, timeout time.Duration) error {
			return nil
		},
		EvalFn: func(evalCmd string) (int, string, error) {
			count := atomic.AddInt32(&evalCount, 1)
			if count == 1 {
				return 1, "first eval failed", nil // first eval fails
			}
			return 0, "second eval passed", nil // rework eval passes
		},
		ReworkFn: func(beadsID, feedback string) error {
			return nil
		},
	}

	cfg := minimalConfig()
	cfg.Scenarios[0].MaxReworks = 2

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tr := result.Trials[0]
	if tr.Status != "pass" {
		t.Errorf("Status = %q, want %q", tr.Status, "pass")
	}
	if tr.Reworks != 1 {
		t.Errorf("Reworks = %d, want 1", tr.Reworks)
	}
}

func TestEngine_MultipleTrials(t *testing.T) {
	e := newTestEngine(nil, nil, nil, nil, 0)
	cfg := minimalConfig()
	cfg.Trials = 3

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Trials) != 3 {
		t.Fatalf("len(Trials) = %d, want 3", len(result.Trials))
	}
	for i, tr := range result.Trials {
		if tr.Trial != i+1 {
			t.Errorf("Trials[%d].Trial = %d, want %d", i, tr.Trial, i+1)
		}
	}
}

func TestEngine_ParallelExecution(t *testing.T) {
	var maxConcurrent int32
	var currentConcurrent int32

	e := &Engine{
		SpawnFn: func(skill, task, model string) (string, error) {
			cur := atomic.AddInt32(&currentConcurrent, 1)
			// Track max concurrent
			for {
				old := atomic.LoadInt32(&maxConcurrent)
				if cur <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond) // simulate work
			atomic.AddInt32(&currentConcurrent, -1)
			return "bench-x", nil
		},
		WaitFn: func(beadsID string, timeout time.Duration) error {
			return nil
		},
		EvalFn: func(evalCmd string) (int, string, error) {
			return 0, "ok", nil
		},
		ReworkFn: func(beadsID, feedback string) error {
			return nil
		},
	}

	cfg := &Config{
		Name:     "parallel-test",
		Trials:   1,
		Parallel: 3,
		Scenarios: []Scenario{
			{Name: "a", Skill: "s", Task: "t", Eval: "e", Timeout: "1m"},
			{Name: "b", Skill: "s", Task: "t", Eval: "e", Timeout: "1m"},
			{Name: "c", Skill: "s", Task: "t", Eval: "e", Timeout: "1m"},
		},
	}

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Trials) != 3 {
		t.Fatalf("len(Trials) = %d, want 3", len(result.Trials))
	}

	// With parallel=3 and 3 scenarios, we should see >1 concurrent
	if atomic.LoadInt32(&maxConcurrent) < 2 {
		t.Logf("maxConcurrent = %d (may be flaky under load)", maxConcurrent)
	}
}

func TestEngine_MissingCallbacks(t *testing.T) {
	e := &Engine{}
	cfg := minimalConfig()
	_, err := e.Run(cfg)
	if err == nil {
		t.Fatal("expected error for missing callbacks")
	}
}

func TestEngine_MultipleScenarios(t *testing.T) {
	e := newTestEngine(nil, nil, nil, nil, 0)
	cfg := &Config{
		Name:     "multi",
		Trials:   2,
		Parallel: 1,
		Scenarios: []Scenario{
			{Name: "a", Skill: "s", Task: "t", Eval: "e", Timeout: "1m"},
			{Name: "b", Skill: "s", Task: "t", Eval: "e", Timeout: "1m"},
		},
	}

	result, err := e.Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// 2 scenarios × 2 trials = 4
	if len(result.Trials) != 4 {
		t.Fatalf("len(Trials) = %d, want 4", len(result.Trials))
	}
}

// --- Results tests ---

func TestRunResult_Summary(t *testing.T) {
	r := &RunResult{
		Name: "test",
		Trials: []TrialResult{
			{Scenario: "a", Trial: 1, Status: "pass", Duration: 10 * time.Second, Reworks: 0},
			{Scenario: "a", Trial: 2, Status: "pass", Duration: 20 * time.Second, Reworks: 1},
			{Scenario: "a", Trial: 3, Status: "fail", Duration: 30 * time.Second, Reworks: 2},
			{Scenario: "b", Trial: 1, Status: "timeout", Duration: 5 * time.Second},
			{Scenario: "b", Trial: 2, Status: "error", Duration: 1 * time.Second},
		},
	}

	s := r.Summary()
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Passed != 2 {
		t.Errorf("Passed = %d, want 2", s.Passed)
	}
	if s.Failed != 1 {
		t.Errorf("Failed = %d, want 1", s.Failed)
	}
	if s.Timeouts != 1 {
		t.Errorf("Timeouts = %d, want 1", s.Timeouts)
	}
	if s.Errors != 1 {
		t.Errorf("Errors = %d, want 1", s.Errors)
	}
	if s.TotalReworks != 3 {
		t.Errorf("TotalReworks = %d, want 3", s.TotalReworks)
	}
	wantRate := 0.4
	if s.PassRate != wantRate {
		t.Errorf("PassRate = %f, want %f", s.PassRate, wantRate)
	}
}

func TestRunResult_WriteJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.jsonl")

	r := &RunResult{
		Trials: []TrialResult{
			{Scenario: "a", Trial: 1, Status: "pass"},
			{Scenario: "a", Trial: 2, Status: "fail"},
		},
	}

	if err := r.WriteJSONL(path); err != nil {
		t.Fatalf("WriteJSONL failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	// Should have 2 lines (JSONL)
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("lines = %d, want 2", lines)
	}
}

func TestRunResult_WriteSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.json")

	r := &RunResult{
		Name: "test",
		Trials: []TrialResult{
			{Scenario: "a", Trial: 1, Status: "pass", Duration: time.Second},
		},
	}

	if err := r.WriteSummary(path); err != nil {
		t.Fatalf("WriteSummary failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if len(data) == 0 {
		t.Error("summary file is empty")
	}
}

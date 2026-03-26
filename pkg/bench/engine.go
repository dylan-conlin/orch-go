package bench

import (
	"fmt"
	"sync"
	"time"
)

// SpawnFunc spawns an agent and returns the beads ID.
type SpawnFunc func(skill, task, model string) (beadsID string, err error)

// WaitFunc blocks until the agent reaches Phase: Complete or times out.
type WaitFunc func(beadsID string, timeout time.Duration) error

// EvalFunc runs the eval command and returns (exitCode, output, error).
type EvalFunc func(evalCmd string) (exitCode int, output string, err error)

// ReworkFunc spawns a rework iteration with feedback.
type ReworkFunc func(beadsID, feedback string) error

// Engine executes benchmark scenarios using spawn/wait/eval/rework primitives.
type Engine struct {
	SpawnFn  SpawnFunc
	WaitFn   WaitFunc
	EvalFn   EvalFunc
	ReworkFn ReworkFunc
}

// Run executes all scenarios × trials from the config, respecting concurrency limits.
func (e *Engine) Run(cfg *Config) (*RunResult, error) {
	if err := e.validate(); err != nil {
		return nil, err
	}

	result := &RunResult{
		Name:      cfg.Name,
		StartedAt: time.Now(),
	}

	type work struct {
		scenario Scenario
		trial    int
	}

	// Build work items: each scenario × each trial
	var items []work
	for _, s := range cfg.Scenarios {
		for t := 1; t <= cfg.Trials; t++ {
			items = append(items, work{scenario: s, trial: t})
		}
	}

	// Execute with concurrency control
	results := make([]TrialResult, len(items))
	sem := make(chan struct{}, cfg.Parallel)
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(idx int, w work) {
			defer wg.Done()
			defer func() { <-sem }() // release
			results[idx] = e.runTrial(w.scenario, w.trial)
		}(i, item)
	}

	wg.Wait()

	result.Trials = results
	result.Duration = time.Since(result.StartedAt)
	result.DurationS = result.Duration.Seconds()

	return result, nil
}

// runTrial executes a single scenario trial: spawn → wait → eval → (rework loop).
func (e *Engine) runTrial(s Scenario, trial int) TrialResult {
	start := time.Now()
	tr := TrialResult{
		Scenario:  s.Name,
		Trial:     trial,
		StartedAt: start,
	}

	// Parse timeout
	timeout, err := time.ParseDuration(s.Timeout)
	if err != nil {
		timeout = 30 * time.Minute
	}

	// Phase 1: Spawn
	beadsID, err := e.SpawnFn(s.Skill, s.Task, s.Model)
	if err != nil {
		tr.Status = "error"
		tr.Error = fmt.Sprintf("spawn failed: %v", err)
		tr.Duration = time.Since(start)
		tr.DurationS = tr.Duration.Seconds()
		return tr
	}
	tr.BeadsID = beadsID

	// Phase 2: Wait for completion
	if err := e.WaitFn(beadsID, timeout); err != nil {
		tr.Status = "timeout"
		tr.Error = fmt.Sprintf("wait failed: %v", err)
		tr.Duration = time.Since(start)
		tr.DurationS = tr.Duration.Seconds()
		return tr
	}

	// Phase 3: Eval
	exitCode, output, err := e.EvalFn(s.Eval)
	tr.EvalOutput = output
	if err != nil {
		tr.Status = "error"
		tr.Error = fmt.Sprintf("eval failed: %v", err)
		tr.Duration = time.Since(start)
		tr.DurationS = tr.Duration.Seconds()
		return tr
	}

	if exitCode == 0 {
		tr.Status = "pass"
		tr.Duration = time.Since(start)
		tr.DurationS = tr.Duration.Seconds()
		return tr
	}

	// Phase 4: Rework loop (if configured)
	for rework := 1; rework <= s.MaxReworks; rework++ {
		tr.Reworks = rework
		feedback := fmt.Sprintf("Benchmark eval failed (exit %d, attempt %d/%d):\n%s",
			exitCode, rework, s.MaxReworks, truncate(output, 4000))

		if err := e.ReworkFn(beadsID, feedback); err != nil {
			tr.Status = "error"
			tr.Error = fmt.Sprintf("rework %d failed: %v", rework, err)
			tr.Duration = time.Since(start)
			tr.DurationS = tr.Duration.Seconds()
			return tr
		}

		// Wait for rework completion
		if err := e.WaitFn(beadsID, timeout); err != nil {
			tr.Status = "timeout"
			tr.Error = fmt.Sprintf("rework %d wait failed: %v", rework, err)
			tr.Duration = time.Since(start)
			tr.DurationS = tr.Duration.Seconds()
			return tr
		}

		// Re-eval
		exitCode, output, err = e.EvalFn(s.Eval)
		tr.EvalOutput = output
		if err != nil {
			tr.Status = "error"
			tr.Error = fmt.Sprintf("rework %d eval failed: %v", rework, err)
			tr.Duration = time.Since(start)
			tr.DurationS = tr.Duration.Seconds()
			return tr
		}

		if exitCode == 0 {
			tr.Status = "pass"
			tr.Duration = time.Since(start)
			tr.DurationS = tr.Duration.Seconds()
			return tr
		}
	}

	// All rework attempts exhausted
	tr.Status = "fail"
	tr.Duration = time.Since(start)
	tr.DurationS = tr.Duration.Seconds()
	return tr
}

func (e *Engine) validate() error {
	if e.SpawnFn == nil {
		return fmt.Errorf("bench engine: SpawnFn is required")
	}
	if e.WaitFn == nil {
		return fmt.Errorf("bench engine: WaitFn is required")
	}
	if e.EvalFn == nil {
		return fmt.Errorf("bench engine: EvalFn is required")
	}
	if e.ReworkFn == nil {
		return fmt.Errorf("bench engine: ReworkFn is required")
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... (truncated %d bytes)", len(s)-max)
}

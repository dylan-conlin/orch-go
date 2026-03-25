// Package orch provides the loop controller for --loop spawn mode.
// The loop controller composes existing primitives (wait, eval, rework)
// into an automated iteration cycle with exit-code-based completion criteria.
package orch

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// WaitFunc blocks until the agent reaches Phase: Complete.
type WaitFunc func(beadsID, projectDir string) error

// EvalFunc runs the eval command and returns (exitCode, output, error).
// Exit code 0 = pass, non-zero = fail, error = command couldn't execute.
type EvalFunc func(evalCmd, projectDir string) (int, string, error)

// LabelFunc adds a label to a beads issue.
type LabelFunc func(beadsID, label, projectDir string) error

// LoopConfig holds configuration for the loop controller.
type LoopConfig struct {
	BeadsID      string // Issue being iterated on
	EvalCommand  string // Shell command; exit 0 = done, non-zero = continue
	MaxIter      int    // Maximum iterations (default 3)
	ProjectDir   string // Project directory for beads operations
	PollInterval time.Duration // How often to check for phase completion
	PollTimeout  time.Duration // Max time to wait per iteration

	// Optional function overrides for testability.
	// When nil, production defaults are used.
	WaitFn   WaitFunc  // Override: wait for phase complete
	EvalFn   EvalFunc  // Override: run eval command
	LabelFn  LabelFunc // Override: add label
}

// LoopResult contains the outcome of a loop run.
type LoopResult struct {
	Iterations int    // Total iterations executed
	EvalPassed bool   // Whether the final eval passed (exit 0)
	LastOutput string // Stdout+stderr from the last eval run
}

// ReworkFunc is the signature for the rework callback.
// The loop controller calls this to spawn a new iteration with feedback.
type ReworkFunc func(beadsID, feedback string) error

// RunLoop executes the loop cycle: wait → eval → {complete if pass, rework if fail}.
// It blocks until the eval command exits 0 or maxIter iterations are exhausted.
func RunLoop(cfg LoopConfig, reworkFn ReworkFunc) (*LoopResult, error) {
	if err := validateLoopConfig(cfg); err != nil {
		return nil, err
	}

	// Resolve function callbacks: use injected or production defaults
	waitFn := cfg.WaitFn
	if waitFn == nil {
		waitFn = func(beadsID, projectDir string) error {
			return waitForPhaseComplete(cfg)
		}
	}
	evalFn := cfg.EvalFn
	if evalFn == nil {
		evalFn = func(evalCmd, projectDir string) (int, string, error) {
			output, exitCode, err := runEvalCommand(evalCmd)
			return exitCode, output, err
		}
	}
	labelFn := cfg.LabelFn
	if labelFn == nil {
		labelFn = verify.AddLabel
	}

	logger := events.NewLogger(events.DefaultLogPath())
	result := &LoopResult{}

	// Add loop:managed label to prevent daemon interference
	if err := labelFn(cfg.BeadsID, "loop:managed", cfg.ProjectDir); err != nil {
		fmt.Printf("Warning: failed to add loop:managed label: %v\n", err)
	}

	for i := 1; i <= cfg.MaxIter; i++ {
		result.Iterations = i
		fmt.Printf("\n--- Loop iteration %d/%d for %s ---\n", i, cfg.MaxIter, cfg.BeadsID)

		// Phase 1: Wait for agent to report Phase: Complete
		if err := waitFn(cfg.BeadsID, cfg.ProjectDir); err != nil {
			return result, fmt.Errorf("iteration %d: wait failed: %w", i, err)
		}

		// Phase 2: Run eval command
		exitCode, evalOutput, err := evalFn(cfg.EvalCommand, cfg.ProjectDir)
		result.LastOutput = evalOutput

		// Log iteration event
		logLoopIteration(logger, cfg.BeadsID, i, exitCode, evalOutput)

		if err != nil && exitCode < 0 {
			// Eval command failed to execute (not just non-zero exit)
			return result, fmt.Errorf("iteration %d: eval command failed to execute: %w", i, err)
		}

		if exitCode == 0 {
			// Eval passed — loop complete
			result.EvalPassed = true
			fmt.Printf("Eval passed on iteration %d — loop complete\n", i)
			logLoopComplete(logger, cfg.BeadsID, i, true, "eval_passed")
			return result, nil
		}

		// Eval failed — rework if we have iterations left
		if i >= cfg.MaxIter {
			fmt.Printf("Eval failed on iteration %d (max reached) — stopping\n", i)
			logLoopComplete(logger, cfg.BeadsID, i, false, "max_iterations")
			return result, nil
		}

		// Build rework feedback from eval output
		feedback := buildReworkFeedback(evalOutput, exitCode, i)
		fmt.Printf("Eval failed (exit %d) — spawning rework iteration %d\n", exitCode, i+1)

		if err := reworkFn(cfg.BeadsID, feedback); err != nil {
			return result, fmt.Errorf("iteration %d: rework failed: %w", i, err)
		}
	}

	return result, nil
}

func validateLoopConfig(cfg LoopConfig) error {
	if cfg.BeadsID == "" {
		return fmt.Errorf("loop: beads ID is required")
	}
	if strings.TrimSpace(cfg.EvalCommand) == "" {
		return fmt.Errorf("loop: eval command is required (--loop-eval)")
	}
	if cfg.MaxIter < 1 {
		return fmt.Errorf("loop: max iterations must be >= 1 (got %d)", cfg.MaxIter)
	}
	return nil
}

// waitForPhaseComplete polls beads comments until "Phase: Complete" is detected.
func waitForPhaseComplete(cfg LoopConfig) error {
	interval := cfg.PollInterval
	if interval == 0 {
		interval = 5 * time.Second
	}
	timeout := cfg.PollTimeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	start := time.Now()
	var lastPhase string

	for {
		status, err := verify.GetPhaseStatus(cfg.BeadsID, cfg.ProjectDir)
		if err == nil && status.Found {
			currentPhase := status.Phase
			if currentPhase != lastPhase {
				fmt.Printf("  Agent phase: %s\n", currentPhase)
				lastPhase = currentPhase
			}

			if strings.EqualFold(currentPhase, "Complete") {
				return nil
			}
		}

		elapsed := time.Since(start)
		if elapsed >= timeout {
			return fmt.Errorf("timeout after %s waiting for Phase: Complete (last phase: %s)", timeout, lastPhase)
		}

		time.Sleep(interval)
	}
}

// runEvalCommand executes the eval command and returns stdout+stderr, exit code, and error.
// Exit code -1 means the command failed to start.
func runEvalCommand(command string) (string, int, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return outputStr, exitErr.ExitCode(), nil
		}
		return outputStr, -1, err
	}

	return outputStr, 0, nil
}

// buildReworkFeedback constructs the feedback message for a rework iteration.
func buildReworkFeedback(evalOutput string, exitCode, iteration int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Loop iteration %d: eval command failed (exit code %d).\n", iteration, exitCode))
	sb.WriteString("Fix the issues identified below and try again.\n\n")
	sb.WriteString("--- EVAL OUTPUT ---\n")
	// Truncate very long eval output to keep rework context manageable
	const maxOutputLen = 4000
	if len(evalOutput) > maxOutputLen {
		sb.WriteString(evalOutput[:maxOutputLen])
		sb.WriteString(fmt.Sprintf("\n... (truncated %d bytes)\n", len(evalOutput)-maxOutputLen))
	} else {
		sb.WriteString(evalOutput)
	}
	sb.WriteString("--- END EVAL OUTPUT ---\n")
	return sb.String()
}

func logLoopIteration(logger *events.Logger, beadsID string, iteration, exitCode int, evalOutput string) {
	// Truncate output for the event log
	truncOutput := evalOutput
	if len(truncOutput) > 1000 {
		truncOutput = truncOutput[:1000] + "..."
	}

	_ = logger.Log(events.Event{
		Type:      events.EventTypeLoopIteration,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":    beadsID,
			"iteration":   iteration,
			"exit_code":   exitCode,
			"eval_passed": exitCode == 0,
			"eval_output": truncOutput,
		},
	})
}

func logLoopComplete(logger *events.Logger, beadsID string, iterations int, evalPassed bool, reason string) {
	_ = logger.Log(events.Event{
		Type:      events.EventTypeLoopComplete,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id":    beadsID,
			"iterations":  iterations,
			"eval_passed": evalPassed,
			"reason":      reason,
		},
	})
}

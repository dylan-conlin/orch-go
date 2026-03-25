package orch

import (
	"fmt"
	"testing"
)

func TestRunLoop_PassOnFirstEval(t *testing.T) {
	evalCalls := 0
	reworkCalls := 0

	cfg := LoopConfig{
		BeadsID:     "test-123",
		EvalCommand: "true",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			evalCalls++
			return 0, "all tests pass", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		reworkCalls++
		return nil
	}

	result, err := RunLoop(cfg, reworkFn)
	if err != nil {
		t.Fatalf("RunLoop returned error: %v", err)
	}
	if result.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", result.Iterations)
	}
	if !result.EvalPassed {
		t.Error("expected EvalPassed=true")
	}
	if evalCalls != 1 {
		t.Errorf("expected 1 eval call, got %d", evalCalls)
	}
	if reworkCalls != 0 {
		t.Errorf("expected 0 rework calls, got %d", reworkCalls)
	}
}

func TestRunLoop_PassAfterRework(t *testing.T) {
	evalCalls := 0
	reworkCalls := 0

	cfg := LoopConfig{
		BeadsID:     "test-456",
		EvalCommand: "go test ./...",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			evalCalls++
			if evalCalls == 1 {
				return 1, "FAIL: TestFoo", nil
			}
			return 0, "ok", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		reworkCalls++
		return nil
	}

	result, err := RunLoop(cfg, reworkFn)
	if err != nil {
		t.Fatalf("RunLoop returned error: %v", err)
	}
	if result.Iterations != 2 {
		t.Errorf("expected 2 iterations, got %d", result.Iterations)
	}
	if !result.EvalPassed {
		t.Error("expected EvalPassed=true")
	}
	if reworkCalls != 1 {
		t.Errorf("expected 1 rework call, got %d", reworkCalls)
	}
}

func TestRunLoop_MaxIterationsReached(t *testing.T) {
	reworkCalls := 0

	cfg := LoopConfig{
		BeadsID:     "test-789",
		EvalCommand: "go test ./...",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			return 1, "FAIL", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		reworkCalls++
		return nil
	}

	result, err := RunLoop(cfg, reworkFn)
	if err != nil {
		t.Fatalf("RunLoop returned error: %v", err)
	}
	if result.Iterations != 3 {
		t.Errorf("expected 3 iterations, got %d", result.Iterations)
	}
	if result.EvalPassed {
		t.Error("expected EvalPassed=false")
	}
	if reworkCalls != 2 {
		t.Errorf("expected 2 rework calls (iterations 1 and 2), got %d", reworkCalls)
	}
}

func TestRunLoop_ReworkError(t *testing.T) {
	cfg := LoopConfig{
		BeadsID:     "test-err",
		EvalCommand: "go test",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			return 1, "FAIL", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		return fmt.Errorf("rework failed: spawn error")
	}

	_, err := RunLoop(cfg, reworkFn)
	if err == nil {
		t.Fatal("expected error from rework failure")
	}
}

func TestRunLoop_WaitError(t *testing.T) {
	cfg := LoopConfig{
		BeadsID:     "test-wait-err",
		EvalCommand: "true",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return fmt.Errorf("timeout waiting for agent")
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			return 0, "ok", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		return nil
	}

	_, err := RunLoop(cfg, reworkFn)
	if err == nil {
		t.Fatal("expected error from wait failure")
	}
}

func TestRunLoop_EvalCommandError(t *testing.T) {
	cfg := LoopConfig{
		BeadsID:     "test-eval-err",
		EvalCommand: "nonexistent-command",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			return -1, "", fmt.Errorf("command not found: nonexistent-command")
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		return nil
	}

	_, err := RunLoop(cfg, reworkFn)
	if err == nil {
		t.Fatal("expected error from eval command failure")
	}
}

func TestRunLoop_LoopManagedLabelAdded(t *testing.T) {
	var labelAdded string

	cfg := LoopConfig{
		BeadsID:     "test-label",
		EvalCommand: "go test",
		MaxIter:     3,
		ProjectDir:  "/tmp/test",
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			return 1, "FAIL", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			if label == "loop:managed" {
				labelAdded = label
			}
			return nil
		},
	}

	reworkFn := func(beadsID, feedback string) error {
		return nil
	}

	RunLoop(cfg, reworkFn)
	if labelAdded != "loop:managed" {
		t.Error("expected loop:managed label to be added")
	}
}

func TestRunLoop_ValidationErrors(t *testing.T) {
	reworkFn := func(beadsID, feedback string) error { return nil }

	tests := []struct {
		name string
		cfg  LoopConfig
	}{
		{"empty beads ID", LoopConfig{EvalCommand: "true", MaxIter: 3}},
		{"empty eval command", LoopConfig{BeadsID: "x", MaxIter: 3}},
		{"zero max iter", LoopConfig{BeadsID: "x", EvalCommand: "true", MaxIter: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunLoop(tt.cfg, reworkFn)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestBuildReworkFeedback(t *testing.T) {
	feedback := buildReworkFeedback("FAIL: TestFoo\n  got: 1\n  want: 2", 1, 1)
	if feedback == "" {
		t.Fatal("expected non-empty feedback")
	}
	if len(feedback) < 20 {
		t.Errorf("feedback too short: %s", feedback)
	}
}

func TestBuildReworkFeedback_Truncation(t *testing.T) {
	longOutput := ""
	for i := 0; i < 5000; i++ {
		longOutput += "x"
	}
	feedback := buildReworkFeedback(longOutput, 1, 1)
	if len(feedback) > 4500 {
		t.Errorf("feedback should be truncated, got %d chars", len(feedback))
	}
}

func TestRunEvalCommand_Success(t *testing.T) {
	output, exitCode, err := runEvalCommand("echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if output != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", output)
	}
}

func TestRunEvalCommand_NonZeroExit(t *testing.T) {
	_, exitCode, err := runEvalCommand("exit 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

func TestRunLoop_SingleIterationMax(t *testing.T) {
	reworkCalls := 0
	cfg := LoopConfig{
		BeadsID:     "test-single",
		EvalCommand: "false",
		MaxIter:     1,
		WaitFn: func(beadsID, projectDir string) error {
			return nil
		},
		EvalFn: func(evalCmd, projectDir string) (int, string, error) {
			return 1, "nope", nil
		},
		LabelFn: func(beadsID, label, projectDir string) error {
			return nil
		},
	}

	result, err := RunLoop(cfg, func(beadsID, feedback string) error {
		reworkCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EvalPassed {
		t.Error("expected eval to not pass")
	}
	if result.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", result.Iterations)
	}
	if reworkCalls != 0 {
		t.Errorf("expected 0 rework calls when max=1, got %d", reworkCalls)
	}
}

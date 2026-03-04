package daemon

import (
	"context"
	"os"
	"os/exec"
	"time"
)

const (
	// BdCommandTimeout is the default timeout for bd CLI commands.
	// Prevents unkillable lock pileups when bd hangs on JSONL lock.
	// 30 seconds is generous — most bd commands complete in <1s.
	BdCommandTimeout = 30 * time.Second
)

// runBdCommand executes a bd CLI command with a timeout.
// Returns stdout and any error (including context.DeadlineExceeded on timeout).
func runBdCommand(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), BdCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bd", args...)
	cmd.Env = os.Environ()
	return cmd.Output()
}

// runBdCommandInDir executes a bd CLI command with timeout in a specific directory.
func runBdCommandInDir(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), BdCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bd", args...)
	cmd.Env = os.Environ()
	cmd.Dir = dir
	return cmd.Output()
}

// runBdCommandCombined executes a bd CLI command with timeout, capturing stdout+stderr.
func runBdCommandCombined(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), BdCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bd", args...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

// runBdCommandCombinedInDir executes a bd CLI command with timeout in a specific directory.
func runBdCommandCombinedInDir(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), BdCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bd", args...)
	cmd.Env = os.Environ()
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

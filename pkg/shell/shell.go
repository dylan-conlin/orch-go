// Package shell provides an abstraction for executing shell commands with
// better error handling, timeout support, and testability.
//
// This package wraps os/exec to provide a consistent interface for running
// external commands throughout the codebase. It supports:
//   - Context-based timeout and cancellation
//   - Configurable working directory
//   - Custom environment variables
//   - Structured error handling with exit codes
//   - Mock implementation for testing
package shell

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Runner defines the interface for executing shell commands.
// Use this interface for dependency injection to enable testing.
type Runner interface {
	// Run executes a command and returns the combined stdout and stderr output.
	// Returns an error if the command fails to start or exits with non-zero status.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)

	// RunWithStdin executes a command with the given stdin input.
	RunWithStdin(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error)

	// Output executes a command and returns only stdout.
	// Stderr is discarded unless the command fails.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)

	// Start starts a command but does not wait for it to complete.
	// Returns a handle to the running command.
	Start(ctx context.Context, name string, args ...string) (Command, error)
}

// Command represents a running command.
type Command interface {
	// Wait waits for the command to complete.
	Wait() error

	// Kill kills the command process.
	Kill() error

	// Pid returns the process ID.
	Pid() int
}

// ExitError wraps exec.ExitError to provide additional context.
type ExitError struct {
	Cmd      string   // Command name
	Args     []string // Command arguments
	ExitCode int      // Exit code
	Stderr   []byte   // Stderr output if available
	Err      error    // Underlying error
}

func (e *ExitError) Error() string {
	if len(e.Stderr) > 0 {
		return fmt.Sprintf("command %q failed with exit code %d: %s", e.Cmd, e.ExitCode, string(e.Stderr))
	}
	return fmt.Sprintf("command %q failed with exit code %d", e.Cmd, e.ExitCode)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// Option is a functional option for configuring DefaultRunner.
type Option func(*DefaultRunner)

// WithDir sets the working directory for command execution.
func WithDir(dir string) Option {
	return func(r *DefaultRunner) {
		r.dir = dir
	}
}

// WithEnv sets the environment variables for command execution.
// If not set, the current process environment is inherited.
func WithEnv(env []string) Option {
	return func(r *DefaultRunner) {
		r.env = env
	}
}

// WithTimeout sets the default timeout for command execution.
// A zero value means no timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(r *DefaultRunner) {
		r.timeout = timeout
	}
}

// DefaultRunner implements Runner using os/exec.
type DefaultRunner struct {
	dir     string
	env     []string
	timeout time.Duration
}

// New creates a new DefaultRunner with the given options.
func New(opts ...Option) *DefaultRunner {
	r := &DefaultRunner{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes a command and returns the combined stdout and stderr output.
func (r *DefaultRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.run(ctx, nil, true, name, args...)
}

// RunWithStdin executes a command with the given stdin input.
func (r *DefaultRunner) RunWithStdin(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	return r.run(ctx, stdin, true, name, args...)
}

// Output executes a command and returns only stdout.
func (r *DefaultRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.run(ctx, nil, false, name, args...)
}

func (r *DefaultRunner) run(ctx context.Context, stdin []byte, combineOutput bool, name string, args ...string) ([]byte, error) {
	// Apply default timeout if set and context doesn't have a deadline
	if r.timeout > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, r.timeout)
			defer cancel()
		}
	}

	cmd := exec.CommandContext(ctx, name, args...)
	r.configureCmd(cmd)

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	if combineOutput {
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), r.wrapError(name, args, err, stderr.Bytes())
	}

	return stdout.Bytes(), nil
}

// Start starts a command but does not wait for it to complete.
func (r *DefaultRunner) Start(ctx context.Context, name string, args ...string) (Command, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	r.configureCmd(cmd)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command %q: %w", name, err)
	}

	return &defaultCommand{cmd: cmd}, nil
}

func (r *DefaultRunner) configureCmd(cmd *exec.Cmd) {
	if r.dir != "" {
		cmd.Dir = r.dir
	}
	if r.env != nil {
		cmd.Env = r.env
	} else {
		cmd.Env = os.Environ()
	}
}

func (r *DefaultRunner) wrapError(name string, args []string, err error, stderr []byte) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return &ExitError{
			Cmd:      name,
			Args:     args,
			ExitCode: exitErr.ExitCode(),
			Stderr:   stderr,
			Err:      err,
		}
	}
	return fmt.Errorf("command %q failed: %w", name, err)
}

// defaultCommand wraps exec.Cmd to implement Command interface.
type defaultCommand struct {
	cmd *exec.Cmd
}

func (c *defaultCommand) Wait() error {
	return c.cmd.Wait()
}

func (c *defaultCommand) Kill() error {
	if c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

func (c *defaultCommand) Pid() int {
	if c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}

// Ensure DefaultRunner implements Runner.
var _ Runner = (*DefaultRunner)(nil)

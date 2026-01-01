package shell

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// MockRunner is a test double for Runner that allows configuring responses.
type MockRunner struct {
	mu sync.Mutex

	// Responses maps command signatures to their responses.
	// The key is "name arg1 arg2 ..." and the value is the response.
	Responses map[string]MockResponse

	// DefaultResponse is returned when no matching response is found.
	// If nil, an error is returned for unknown commands.
	DefaultResponse *MockResponse

	// Calls records all commands that were executed.
	Calls []MockCall

	// StrictMode when true returns an error for commands not in Responses.
	// When false, returns empty output for unknown commands.
	StrictMode bool
}

// MockResponse represents a configured response for a command.
type MockResponse struct {
	Output   []byte
	Err      error
	ExitCode int // Only used if Err is nil but you want to simulate exit code
}

// MockCall records a single command execution.
type MockCall struct {
	Name  string
	Args  []string
	Stdin []byte
}

// NewMockRunner creates a new MockRunner with an empty response map.
func NewMockRunner() *MockRunner {
	return &MockRunner{
		Responses: make(map[string]MockResponse),
		Calls:     make([]MockCall, 0),
	}
}

// AddResponse adds a response for a specific command.
// The signature should be "name arg1 arg2 ..." (space-separated).
func (m *MockRunner) AddResponse(signature string, output []byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses[signature] = MockResponse{Output: output, Err: err}
}

// AddExitCodeResponse adds a response with a specific exit code.
func (m *MockRunner) AddExitCodeResponse(signature string, output []byte, exitCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if exitCode != 0 {
		m.Responses[signature] = MockResponse{
			Output: output,
			Err: &ExitError{
				Cmd:      strings.Fields(signature)[0],
				ExitCode: exitCode,
				Stderr:   output,
			},
		}
	} else {
		m.Responses[signature] = MockResponse{Output: output}
	}
}

// Run implements Runner.Run.
func (m *MockRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.execute(ctx, nil, name, args...)
}

// RunWithStdin implements Runner.RunWithStdin.
func (m *MockRunner) RunWithStdin(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	return m.execute(ctx, stdin, name, args...)
}

// Output implements Runner.Output.
func (m *MockRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.execute(ctx, nil, name, args...)
}

func (m *MockRunner) execute(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.Calls = append(m.Calls, MockCall{
		Name:  name,
		Args:  args,
		Stdin: stdin,
	})

	// Check context
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Build signature
	signature := m.buildSignature(name, args)

	// Look for exact match
	if resp, ok := m.Responses[signature]; ok {
		return resp.Output, resp.Err
	}

	// Try command name only (for flexible matching)
	if resp, ok := m.Responses[name]; ok {
		return resp.Output, resp.Err
	}

	// Use default response if available
	if m.DefaultResponse != nil {
		return m.DefaultResponse.Output, m.DefaultResponse.Err
	}

	// Strict mode returns error
	if m.StrictMode {
		return nil, fmt.Errorf("mock: no response configured for command %q", signature)
	}

	// Permissive mode returns empty output
	return nil, nil
}

// Start implements Runner.Start.
func (m *MockRunner) Start(ctx context.Context, name string, args ...string) (Command, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.Calls = append(m.Calls, MockCall{
		Name: name,
		Args: args,
	})

	// Check context
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Build signature
	signature := m.buildSignature(name, args)

	// Look for response to determine if start should fail
	if resp, ok := m.Responses[signature]; ok && resp.Err != nil {
		return nil, resp.Err
	}

	return &mockCommand{}, nil
}

func (m *MockRunner) buildSignature(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

// Reset clears all recorded calls.
func (m *MockRunner) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = make([]MockCall, 0)
}

// CallCount returns the number of times a command was executed.
// If signature is empty, returns total call count.
func (m *MockRunner) CallCount(signature string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if signature == "" {
		return len(m.Calls)
	}

	count := 0
	for _, call := range m.Calls {
		callSig := m.buildSignature(call.Name, call.Args)
		if callSig == signature || call.Name == signature {
			count++
		}
	}
	return count
}

// WasCalled returns true if the command was executed at least once.
func (m *MockRunner) WasCalled(signature string) bool {
	return m.CallCount(signature) > 0
}

// LastCall returns the most recent call, or nil if no calls were made.
func (m *MockRunner) LastCall() *MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Calls) == 0 {
		return nil
	}
	return &m.Calls[len(m.Calls)-1]
}

// mockCommand is a no-op implementation of Command for testing.
type mockCommand struct {
	waited bool
	killed bool
}

func (c *mockCommand) Wait() error {
	c.waited = true
	return nil
}

func (c *mockCommand) Kill() error {
	c.killed = true
	return nil
}

func (c *mockCommand) Pid() int {
	return 12345 // Fake PID for testing
}

// Ensure MockRunner implements Runner.
var _ Runner = (*MockRunner)(nil)

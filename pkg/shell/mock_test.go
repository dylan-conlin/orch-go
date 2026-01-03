package shell

import (
	"context"
	"testing"
)

func TestMockRunner_Run(t *testing.T) {
	m := NewMockRunner()
	m.AddResponse("echo hello", []byte("hello\n"), nil)

	output, err := m.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if string(output) != "hello\n" {
		t.Errorf("expected %q, got %q", "hello\n", string(output))
	}
}

func TestMockRunner_RecordsCalls(t *testing.T) {
	m := NewMockRunner()

	m.Run(context.Background(), "git", "status")
	m.Run(context.Background(), "git", "log")

	if len(m.Calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(m.Calls))
	}

	if m.Calls[0].Name != "git" || len(m.Calls[0].Args) != 1 || m.Calls[0].Args[0] != "status" {
		t.Errorf("unexpected first call: %+v", m.Calls[0])
	}

	if m.Calls[1].Name != "git" || len(m.Calls[1].Args) != 1 || m.Calls[1].Args[0] != "log" {
		t.Errorf("unexpected second call: %+v", m.Calls[1])
	}
}

func TestMockRunner_StrictMode(t *testing.T) {
	m := NewMockRunner()
	m.StrictMode = true

	_, err := m.Run(context.Background(), "unknown", "command")
	if err == nil {
		t.Error("expected error in strict mode for unknown command")
	}
}

func TestMockRunner_PermissiveMode(t *testing.T) {
	m := NewMockRunner()
	m.StrictMode = false // default

	output, err := m.Run(context.Background(), "unknown", "command")
	if err != nil {
		t.Errorf("unexpected error in permissive mode: %v", err)
	}

	if output != nil {
		t.Errorf("expected nil output, got %q", string(output))
	}
}

func TestMockRunner_DefaultResponse(t *testing.T) {
	m := NewMockRunner()
	m.DefaultResponse = &MockResponse{Output: []byte("default")}

	output, err := m.Run(context.Background(), "any", "command")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if string(output) != "default" {
		t.Errorf("expected %q, got %q", "default", string(output))
	}
}

func TestMockRunner_ExitCodeResponse(t *testing.T) {
	m := NewMockRunner()
	m.AddExitCodeResponse("git status", []byte("error output"), 1)

	output, err := m.Run(context.Background(), "git", "status")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}

	if exitErr.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode)
	}

	if string(output) != "error output" {
		t.Errorf("expected %q, got %q", "error output", string(output))
	}
}

func TestMockRunner_ContextCanceled(t *testing.T) {
	m := NewMockRunner()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Run(ctx, "echo", "test")
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMockRunner_CallCount(t *testing.T) {
	m := NewMockRunner()

	m.Run(context.Background(), "git", "status")
	m.Run(context.Background(), "git", "status")
	m.Run(context.Background(), "git", "log")

	if count := m.CallCount(""); count != 3 {
		t.Errorf("expected total count 3, got %d", count)
	}

	if count := m.CallCount("git status"); count != 2 {
		t.Errorf("expected 'git status' count 2, got %d", count)
	}

	if count := m.CallCount("git log"); count != 1 {
		t.Errorf("expected 'git log' count 1, got %d", count)
	}
}

func TestMockRunner_WasCalled(t *testing.T) {
	m := NewMockRunner()

	m.Run(context.Background(), "git", "status")

	if !m.WasCalled("git status") {
		t.Error("expected WasCalled to return true for 'git status'")
	}

	if m.WasCalled("git log") {
		t.Error("expected WasCalled to return false for 'git log'")
	}
}

func TestMockRunner_LastCall(t *testing.T) {
	m := NewMockRunner()

	if m.LastCall() != nil {
		t.Error("expected nil LastCall when no calls made")
	}

	m.Run(context.Background(), "first", "command")
	m.Run(context.Background(), "second", "command")

	last := m.LastCall()
	if last == nil {
		t.Fatal("expected non-nil LastCall")
	}

	if last.Name != "second" {
		t.Errorf("expected last call name %q, got %q", "second", last.Name)
	}
}

func TestMockRunner_Reset(t *testing.T) {
	m := NewMockRunner()

	m.Run(context.Background(), "git", "status")
	m.Reset()

	if len(m.Calls) != 0 {
		t.Errorf("expected 0 calls after reset, got %d", len(m.Calls))
	}
}

func TestMockRunner_RunWithStdin(t *testing.T) {
	m := NewMockRunner()
	m.AddResponse("cat", []byte("stdin content"), nil)

	stdin := []byte("test input")
	output, err := m.RunWithStdin(context.Background(), stdin, "cat")
	if err != nil {
		t.Fatalf("RunWithStdin failed: %v", err)
	}

	if string(output) != "stdin content" {
		t.Errorf("expected %q, got %q", "stdin content", string(output))
	}

	// Verify stdin was recorded
	if m.Calls[0].Stdin == nil || string(m.Calls[0].Stdin) != "test input" {
		t.Errorf("stdin not recorded correctly: %v", m.Calls[0].Stdin)
	}
}

func TestMockRunner_Output(t *testing.T) {
	m := NewMockRunner()
	m.AddResponse("echo test", []byte("test\n"), nil)

	output, err := m.Output(context.Background(), "echo", "test")
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	if string(output) != "test\n" {
		t.Errorf("expected %q, got %q", "test\n", string(output))
	}
}

func TestMockRunner_Start(t *testing.T) {
	m := NewMockRunner()

	cmd, err := m.Start(context.Background(), "sleep", "1")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify the call was recorded
	if !m.WasCalled("sleep 1") {
		t.Error("expected Start to record call")
	}

	// Test the mock command
	if pid := cmd.Pid(); pid != 12345 {
		t.Errorf("expected fake PID 12345, got %d", pid)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("Wait failed: %v", err)
	}

	if err := cmd.Kill(); err != nil {
		t.Errorf("Kill failed: %v", err)
	}
}

func TestMockRunner_StartError(t *testing.T) {
	m := NewMockRunner()
	m.AddResponse("fail start", nil, &ExitError{Cmd: "fail", ExitCode: 1})

	_, err := m.Start(context.Background(), "fail", "start")
	if err == nil {
		t.Error("expected Start to fail when error configured")
	}
}

func TestMockRunner_CommandNameOnlyMatch(t *testing.T) {
	m := NewMockRunner()
	// Only configure response for command name, not full signature
	m.AddResponse("git", []byte("git response"), nil)

	// Should match regardless of args
	output, err := m.Run(context.Background(), "git", "any", "args")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if string(output) != "git response" {
		t.Errorf("expected %q, got %q", "git response", string(output))
	}
}

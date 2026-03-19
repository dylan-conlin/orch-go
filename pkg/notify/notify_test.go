package notify

import (
	"testing"
	"time"
)

// TestNotifyCompletion tests the NotifyCompletion function.
func TestNotifyCompletion(t *testing.T) {
	// Create a mock notifier to track calls
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: true}

	// Test with workspace name
	err := notifier.SessionComplete("ses_abc123", "og-feat-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mock was called
	if mock.callCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount)
	}
	if mock.lastTitle != "Agent Complete: og-feat-test" {
		t.Errorf("expected title 'Agent Complete: og-feat-test', got '%s'", mock.lastTitle)
	}
	if mock.lastMessage != "Session ses_abc123 completed" {
		t.Errorf("expected message 'Session ses_abc123 completed', got '%s'", mock.lastMessage)
	}
}

// TestNotifyCompletionNoWorkspace tests completion without workspace name.
func TestNotifyCompletionNoWorkspace(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: true}

	err := notifier.SessionComplete("ses_xyz789", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use session ID when no workspace name
	if mock.lastTitle != "Agent Complete: ses_xyz789" {
		t.Errorf("expected title 'Agent Complete: ses_xyz789', got '%s'", mock.lastTitle)
	}
}

// TestNotifyError tests error notification.
func TestNotifyError(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: true}

	err := notifier.Error("Test error message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastTitle != "OpenCode Error" {
		t.Errorf("expected title 'OpenCode Error', got '%s'", mock.lastTitle)
	}
	if mock.lastMessage != "Test error message" {
		t.Errorf("expected message 'Test error message', got '%s'", mock.lastMessage)
	}
}

// TestDefaultNotifier tests creating the default notifier.
func TestDefaultNotifier(t *testing.T) {
	notifier := Default()
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}
	if notifier.backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

// TestNotificationsDisabled tests that notifications are skipped when disabled.
func TestNotificationsDisabled(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: false}

	// SessionComplete should not call backend
	err := notifier.SessionComplete("ses_abc123", "og-feat-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.callCount != 0 {
		t.Errorf("expected 0 calls when disabled, got %d", mock.callCount)
	}

	// Error should not call backend
	err = notifier.Error("Test error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.callCount != 0 {
		t.Errorf("expected 0 calls when disabled, got %d", mock.callCount)
	}
}

// TestIsEnabled tests the IsEnabled method.
func TestIsEnabled(t *testing.T) {
	enabledNotifier := &Notifier{backend: nil, enabled: true}
	if !enabledNotifier.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}

	disabledNotifier := &Notifier{backend: nil, enabled: false}
	if disabledNotifier.IsEnabled() {
		t.Error("IsEnabled() = true, want false")
	}
}

// TestSetEnabled tests the SetEnabled method.
func TestSetEnabled(t *testing.T) {
	notifier := &Notifier{backend: nil, enabled: true}

	notifier.SetEnabled(false)
	if notifier.IsEnabled() {
		t.Error("after SetEnabled(false), IsEnabled() = true, want false")
	}

	notifier.SetEnabled(true)
	if !notifier.IsEnabled() {
		t.Error("after SetEnabled(true), IsEnabled() = false, want true")
	}
}

// TestNewNotifierEnabledByDefault tests that New() creates enabled notifier.
func TestNewNotifierEnabledByDefault(t *testing.T) {
	mock := &MockNotifier{}
	notifier := New(mock)
	if !notifier.IsEnabled() {
		t.Error("New() should create enabled notifier")
	}
}

// TestDaemonStuck tests the DaemonStuck notification.
func TestDaemonStuck(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: true}

	err := notifier.DaemonStuck(5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount)
	}
	if mock.lastTitle != "Daemon Stuck" {
		t.Errorf("expected title 'Daemon Stuck', got '%s'", mock.lastTitle)
	}
	expected := "All 5/5 slots full — no spawns or completions in 10+ min"
	if mock.lastMessage != expected {
		t.Errorf("expected message %q, got %q", expected, mock.lastMessage)
	}
}

// TestDaemonStuckDisabled tests DaemonStuck is skipped when disabled.
func TestDaemonStuckDisabled(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: false}

	err := notifier.DaemonStuck(5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.callCount != 0 {
		t.Errorf("expected 0 calls when disabled, got %d", mock.callCount)
	}
}

// TestAgentUnresponsive tests the AgentUnresponsive notification.
func TestAgentUnresponsive(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: true}

	err := notifier.AgentUnresponsive("scs-sp-abc", 12*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount)
	}
	if mock.lastTitle != "Agent Unresponsive: scs-sp-abc" {
		t.Errorf("expected title 'Agent Unresponsive: scs-sp-abc', got '%s'", mock.lastTitle)
	}
	expected := "No phase reported in 12m0s — may need respawn"
	if mock.lastMessage != expected {
		t.Errorf("expected message %q, got %q", expected, mock.lastMessage)
	}
}

// TestAgentUnresponsiveDisabled tests AgentUnresponsive is skipped when disabled.
func TestAgentUnresponsiveDisabled(t *testing.T) {
	mock := &MockNotifier{}
	notifier := &Notifier{backend: mock, enabled: false}

	err := notifier.AgentUnresponsive("scs-sp-abc", 12*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.callCount != 0 {
		t.Errorf("expected 0 calls when disabled, got %d", mock.callCount)
	}
}

// MockNotifier is a mock implementation for testing.
type MockNotifier struct {
	callCount   int
	lastTitle   string
	lastMessage string
	lastIcon    string
	shouldError error
}

func (m *MockNotifier) Notify(title, message, icon string) error {
	m.callCount++
	m.lastTitle = title
	m.lastMessage = message
	m.lastIcon = icon
	return m.shouldError
}

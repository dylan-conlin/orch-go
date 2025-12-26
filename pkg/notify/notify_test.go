package notify

import (
	"testing"
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

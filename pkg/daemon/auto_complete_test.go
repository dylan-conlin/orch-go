package daemon

import (
	"fmt"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func TestProcessCompletion_AutoTier_DoesNotCallAutoCompleterWhenVerificationFails(t *testing.T) {
	// Auto-tier agent in a nonexistent project dir — verification will fail.
	// AutoCompleter should NOT be called when verification fails.
	orchCompleteCalled := false
	d := &Daemon{
		AutoCompleter: &mockAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:    "orch-go-test1",
		Title:      "Test auto-complete",
		ProjectDir: "/nonexistent/dir",
	}

	config := CompletionConfig{
		ProjectDir: "/nonexistent/dir",
	}

	result := d.ProcessCompletion(agent, config)

	if orchCompleteCalled {
		t.Error("AutoCompleter.Complete should NOT be called when verification fails")
	}
	if result.AutoCompleted {
		t.Error("result.AutoCompleted should be false when verification fails")
	}
	if result.Error == nil {
		t.Error("expected error from failed verification")
	}
}

func TestProcessCompletion_ReviewTier_DoesNotAutoComplete(t *testing.T) {
	// Review-tier agents should never trigger auto-complete,
	// even when verification fails at a different step.
	orchCompleteCalled := false
	d := &Daemon{
		AutoCompleter: &mockAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:    "orch-go-test3",
		Title:      "Test review-tier",
		ProjectDir: "/nonexistent",
	}

	config := CompletionConfig{
		ProjectDir: "/nonexistent",
	}

	_ = d.ProcessCompletion(agent, config)

	if orchCompleteCalled {
		t.Error("AutoCompleter.Complete should NOT be called for review-tier agent")
	}
}

func TestAutoCompleter_Interface(t *testing.T) {
	// Verify OrcCompleter implements AutoCompleter
	var _ AutoCompleter = (*OrcCompleter)(nil)
	var _ AutoCompleter = (*mockAutoCompleter)(nil)
}

func TestAutoCompleteResult_Fields(t *testing.T) {
	result := CompletionResult{
		BeadsID:       "test-123",
		Processed:     true,
		AutoCompleted: true,
		CloseReason:   "Phase: Complete - auto-completed",
	}

	if !result.AutoCompleted {
		t.Error("AutoCompleted should be true")
	}
	if !result.Processed {
		t.Error("Processed should be true")
	}
}

func TestAutoCompleteResult_NotAutoCompleted_ByDefault(t *testing.T) {
	result := CompletionResult{
		BeadsID:   "test-456",
		Processed: true,
	}

	if result.AutoCompleted {
		t.Error("AutoCompleted should be false by default")
	}
}

func TestReviewTierDefaultsForAutoTier(t *testing.T) {
	// Verify that the skills expected to be auto-tier actually are
	autoSkills := []string{"capture-knowledge", "issue-creation"}
	for _, skill := range autoSkills {
		tier := spawn.DefaultReviewTier(skill, "")
		if tier != spawn.ReviewAuto {
			t.Errorf("DefaultReviewTier(%q) = %q, want %q", skill, tier, spawn.ReviewAuto)
		}
	}

	// Verify non-auto skills are NOT auto
	nonAutoSkills := []string{"feature-impl", "investigation", "architect"}
	for _, skill := range nonAutoSkills {
		tier := spawn.DefaultReviewTier(skill, "")
		if tier == spawn.ReviewAuto {
			t.Errorf("DefaultReviewTier(%q) should NOT be auto", skill)
		}
	}
}

func TestMockAutoCompleter_DefaultReturnsNil(t *testing.T) {
	m := &mockAutoCompleter{}
	if err := m.Complete("test", "/dir"); err != nil {
		t.Errorf("default mockAutoCompleter.Complete should return nil, got: %v", err)
	}
}

func TestMockAutoCompleter_ReturnsError(t *testing.T) {
	m := &mockAutoCompleter{
		CompleteFunc: func(beadsID, workdir string) error {
			return fmt.Errorf("gate failure")
		},
	}
	err := m.Complete("test", "/dir")
	if err == nil {
		t.Error("expected error from mockAutoCompleter")
	}
	if err.Error() != "gate failure" {
		t.Errorf("error = %q, want 'gate failure'", err.Error())
	}
}

// mockAutoCompleter implements AutoCompleter for tests.
type mockAutoCompleter struct {
	CompleteFunc func(beadsID, workdir string) error
}

func (m *mockAutoCompleter) Complete(beadsID, workdir string) error {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(beadsID, workdir)
	}
	return nil
}

// Verify interface compliance.
var _ AutoCompleter = (*mockAutoCompleter)(nil)

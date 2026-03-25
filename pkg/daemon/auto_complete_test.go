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

func TestLightAutoCompleter_Interface(t *testing.T) {
	// Verify OrcCompleter implements LightAutoCompleter
	var _ LightAutoCompleter = (*OrcCompleter)(nil)
	var _ LightAutoCompleter = (*mockLightAutoCompleter)(nil)
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

func TestReviewTierDefaultsForScanTier(t *testing.T) {
	// Scan-tier skills should now auto-complete (not require human review).
	// Verify they're classified as scan tier.
	scanSkills := []string{"investigation", "probe", "research", "codebase-audit", "design-session", "ux-audit"}
	for _, skill := range scanSkills {
		tier := spawn.DefaultReviewTier(skill, "")
		if tier != spawn.ReviewScan {
			t.Errorf("DefaultReviewTier(%q) = %q, want %q", skill, tier, spawn.ReviewScan)
		}
	}

	// Review-tier skills should NOT be auto-completable
	reviewSkills := []string{"feature-impl", "systematic-debugging", "architect"}
	for _, skill := range reviewSkills {
		tier := spawn.DefaultReviewTier(skill, "")
		if tier != spawn.ReviewReview {
			t.Errorf("DefaultReviewTier(%q) = %q, want %q", skill, tier, spawn.ReviewReview)
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

// --- Effort label tests ---

func TestIsEffortSmall(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   bool
	}{
		{"effort:small present", []string{"triage:ready", "effort:small"}, true},
		{"effort:small only", []string{"effort:small"}, true},
		{"effort:medium present", []string{"effort:medium"}, false},
		{"effort:large present", []string{"effort:large"}, false},
		{"no effort label", []string{"triage:ready"}, false},
		{"empty labels", []string{}, false},
		{"nil labels", nil, false},
		{"case insensitive", []string{"Effort:Small"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEffortSmall(tt.labels)
			if got != tt.want {
				t.Errorf("IsEffortSmall(%v) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}

func TestHasEffortLabel(t *testing.T) {
	labels := []string{"effort:small", "triage:ready"}
	if !HasEffortLabel(labels, LabelEffortSmall) {
		t.Error("expected HasEffortLabel to find effort:small")
	}
	if HasEffortLabel(labels, LabelEffortMedium) {
		t.Error("expected HasEffortLabel not to find effort:medium")
	}
}

func TestEffortLabelConstants(t *testing.T) {
	if LabelEffortSmall != "effort:small" {
		t.Errorf("LabelEffortSmall = %q, want 'effort:small'", LabelEffortSmall)
	}
	if LabelEffortMedium != "effort:medium" {
		t.Errorf("LabelEffortMedium = %q, want 'effort:medium'", LabelEffortMedium)
	}
	if LabelEffortLarge != "effort:large" {
		t.Errorf("LabelEffortLarge = %q, want 'effort:large'", LabelEffortLarge)
	}
}

func TestProcessCompletion_EffortSmall_DoesNotCallAutoCompleterWhenVerificationFails(t *testing.T) {
	// effort:small agent in nonexistent project dir — verification will fail.
	// AutoCompleter should NOT be called when verification fails.
	orchCompleteCalled := false
	d := &Daemon{
		AutoCompleter: &mockLightAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
			CompleteLightFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:    "orch-go-test-small1",
		Title:      "Test effort:small auto-complete",
		ProjectDir: "/nonexistent/dir",
		Labels:     []string{"effort:small"},
	}

	config := CompletionConfig{
		ProjectDir: "/nonexistent/dir",
	}

	result := d.ProcessCompletion(agent, config)

	if orchCompleteCalled {
		t.Error("AutoCompleter should NOT be called when verification fails (effort:small)")
	}
	if result.AutoCompleted {
		t.Error("result.AutoCompleted should be false when verification fails")
	}
	if result.Error == nil {
		t.Error("expected error from failed verification")
	}
}

func TestProcessCompletion_EffortMedium_GetsReadyReviewLabel(t *testing.T) {
	// effort:medium should NOT auto-complete — should follow normal ready-review path.
	orchCompleteCalled := false
	d := &Daemon{
		AutoCompleter: &mockLightAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
			CompleteLightFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:    "orch-go-test-medium1",
		Title:      "Test effort:medium",
		ProjectDir: "/nonexistent",
		Labels:     []string{"effort:medium"},
	}

	config := CompletionConfig{
		ProjectDir: "/nonexistent",
	}

	// Will fail at verification (nonexistent dir), but AutoCompleter
	// should not be called for effort:medium
	_ = d.ProcessCompletion(agent, config)

	if orchCompleteCalled {
		t.Error("AutoCompleter should NOT be called for effort:medium agent")
	}
}

func TestProcessCompletion_NoEffortLabel_GetsReadyReviewLabel(t *testing.T) {
	// No effort label should NOT auto-complete — should follow normal ready-review path.
	orchCompleteCalled := false
	d := &Daemon{
		AutoCompleter: &mockLightAutoCompleter{
			CompleteFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
			CompleteLightFunc: func(beadsID, workdir string) error {
				orchCompleteCalled = true
				return nil
			},
		},
	}

	agent := CompletedAgent{
		BeadsID:    "orch-go-test-noeffort",
		Title:      "Test no effort label",
		ProjectDir: "/nonexistent",
		Labels:     []string{"triage:ready"},
	}

	config := CompletionConfig{
		ProjectDir: "/nonexistent",
	}

	_ = d.ProcessCompletion(agent, config)

	if orchCompleteCalled {
		t.Error("AutoCompleter should NOT be called for agent without effort:small label")
	}
}

func TestCompletedAgent_LabelsField(t *testing.T) {
	agent := CompletedAgent{
		BeadsID: "orch-go-123",
		Title:   "Test",
		Labels:  []string{"effort:small", "triage:ready"},
	}

	if len(agent.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(agent.Labels))
	}
	if !IsEffortSmall(agent.Labels) {
		t.Error("expected agent labels to contain effort:small")
	}
}

// --- Mocks ---

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

// mockLightAutoCompleter implements LightAutoCompleter for tests.
type mockLightAutoCompleter struct {
	CompleteFunc      func(beadsID, workdir string) error
	CompleteLightFunc func(beadsID, workdir string) error
}

func (m *mockLightAutoCompleter) Complete(beadsID, workdir string) error {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(beadsID, workdir)
	}
	return nil
}

func (m *mockLightAutoCompleter) CompleteLight(beadsID, workdir string) error {
	if m.CompleteLightFunc != nil {
		return m.CompleteLightFunc(beadsID, workdir)
	}
	return nil
}

// mockHeadlessAutoCompleter implements HeadlessAutoCompleter for tests.
type mockHeadlessAutoCompleter struct {
	CompleteFunc         func(beadsID, workdir string) error
	CompleteLightFunc    func(beadsID, workdir string) error
	CompleteHeadlessFunc func(beadsID, workdir string) error
}

func (m *mockHeadlessAutoCompleter) Complete(beadsID, workdir string) error {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(beadsID, workdir)
	}
	return nil
}

func (m *mockHeadlessAutoCompleter) CompleteLight(beadsID, workdir string) error {
	if m.CompleteLightFunc != nil {
		return m.CompleteLightFunc(beadsID, workdir)
	}
	return nil
}

func (m *mockHeadlessAutoCompleter) CompleteHeadless(beadsID, workdir string) error {
	if m.CompleteHeadlessFunc != nil {
		return m.CompleteHeadlessFunc(beadsID, workdir)
	}
	return nil
}

// Verify interface compliance.
var _ AutoCompleter = (*mockAutoCompleter)(nil)
var _ LightAutoCompleter = (*mockLightAutoCompleter)(nil)
var _ HeadlessAutoCompleter = (*mockHeadlessAutoCompleter)(nil)
var _ HeadlessAutoCompleter = (*OrcCompleter)(nil)

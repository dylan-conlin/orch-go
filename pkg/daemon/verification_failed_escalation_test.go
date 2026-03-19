package daemon

import (
	"testing"
	"time"
)

func TestRunPeriodicVerificationFailedEscalation_NotDue(t *testing.T) {
	d := New()
	// Set last run to recent — should not be due
	d.Scheduler.SetLastRun(TaskVerificationFailedEscalation, time.Now())

	result := d.RunPeriodicVerificationFailedEscalation()
	if result != nil {
		t.Error("expected nil result when not due")
	}
}

func TestRunPeriodicVerificationFailedEscalation_DueOnFirstRun(t *testing.T) {
	d := New()
	// Never run before — should be due, but will fail to connect to beads (no daemon)
	result := d.RunPeriodicVerificationFailedEscalation()
	if result == nil {
		t.Fatal("expected non-nil result on first run")
	}
	// Should get an error since beads isn't running in test
	if result.Error == nil {
		// If beads IS running, we get a real result — just verify it has a message
		if result.Message == "" {
			t.Error("expected non-empty message")
		}
	}
}

func TestVerificationFailedEscalationResult_Snapshot(t *testing.T) {
	result := &VerificationFailedEscalationResult{
		EscalatedCount: 2,
		ScannedCount:   5,
	}
	snapshot := result.Snapshot()
	if snapshot.EscalatedCount != 2 {
		t.Errorf("EscalatedCount = %d, want 2", snapshot.EscalatedCount)
	}
	if snapshot.ScannedCount != 5 {
		t.Errorf("ScannedCount = %d, want 5", snapshot.ScannedCount)
	}
	if snapshot.LastCheck.IsZero() {
		t.Error("LastCheck should not be zero")
	}
}

func TestEscalationLabelConstants(t *testing.T) {
	if LabelVerificationFailed != "daemon:verification-failed" {
		t.Errorf("LabelVerificationFailed = %q, want 'daemon:verification-failed'", LabelVerificationFailed)
	}
	if LabelReadyReview != "daemon:ready-review" {
		t.Errorf("LabelReadyReview = %q, want 'daemon:ready-review'", LabelReadyReview)
	}
	if LabelTriageReview != "triage:review" {
		t.Errorf("LabelTriageReview = %q, want 'triage:review'", LabelTriageReview)
	}
}

func TestSchedulerRegistration_VerificationFailedEscalation(t *testing.T) {
	d := New()
	// Verify the task is registered and due on first run
	if !d.Scheduler.IsDue(TaskVerificationFailedEscalation) {
		t.Error("TaskVerificationFailedEscalation should be due on first run")
	}
}

func TestSchedulerRegistration_LightweightCleanup(t *testing.T) {
	d := New()
	if !d.Scheduler.IsDue(TaskLightweightCleanup) {
		t.Error("TaskLightweightCleanup should be due on first run")
	}
}

func TestTaskConstants(t *testing.T) {
	if TaskVerificationFailedEscalation != "verification_failed_escalation" {
		t.Errorf("TaskVerificationFailedEscalation = %q, want 'verification_failed_escalation'",
			TaskVerificationFailedEscalation)
	}
	if TaskLightweightCleanup != "lightweight_cleanup" {
		t.Errorf("TaskLightweightCleanup = %q, want 'lightweight_cleanup'",
			TaskLightweightCleanup)
	}
}

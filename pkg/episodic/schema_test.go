package episodic

import (
	"testing"
	"time"
)

func TestActionMemoryValidateSuccess(t *testing.T) {
	now := time.Now().UTC()
	memory := ActionMemory{
		ID:        "am_1",
		Boundary:  BoundarySpawn,
		Project:   "orch-go",
		Workspace: "og-feat-test",
		SessionID: "ses_123",
		BeadsID:   "orch-go-123",
		Action: Action{
			Type:  "lifecycle",
			Name:  "session.spawned",
			Input: "{}",
		},
		Outcome: Outcome{
			Status:  OutcomeSuccess,
			Summary: "Session spawned",
		},
		Evidence: Evidence{
			Kind:      EvidenceKindEventsJSONL,
			Pointer:   "~/.orch/events.jsonl#type=session.spawned",
			Timestamp: now.Unix(),
			Hash:      "sha256:abc",
		},
		Confidence:      0.9,
		ExpiresAt:       now.Add(time.Hour),
		ValidationState: ValidationPending,
		CreatedAt:       now,
	}

	if err := memory.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestActionMemoryValidateRequiredFields(t *testing.T) {
	memory := ActionMemory{}
	memory.SetDefaults(time.Now().UTC())
	memory.Action = Action{}
	memory.Outcome = Outcome{}
	memory.Evidence = Evidence{}
	memory.Confidence = 1.2

	if err := memory.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestExpiryForBoundary(t *testing.T) {
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)
	spawnExpiry := ExpiryForBoundary(BoundarySpawn, now)
	verificationExpiry := ExpiryForBoundary(BoundaryVerification, now)

	if spawnExpiry.Sub(now) != 24*time.Hour {
		t.Fatalf("spawn expiry = %v, want 24h", spawnExpiry.Sub(now))
	}
	if verificationExpiry.Sub(now) != 14*24*time.Hour {
		t.Fatalf("verification expiry = %v, want 14d", verificationExpiry.Sub(now))
	}
}

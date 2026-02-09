package episodic

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreAppendReadPrune(t *testing.T) {
	now := time.Now().UTC()
	store := NewStore(filepath.Join(t.TempDir(), "action-memory.jsonl"))

	active := ActionMemory{
		ID:              "am_active",
		Boundary:        BoundaryCommand,
		Project:         "orch-go",
		Workspace:       "og-feat-1",
		SessionID:       "ses_1",
		BeadsID:         "orch-go-1",
		Action:          Action{Type: "command", Name: "session.send", Input: "ping"},
		Outcome:         Outcome{Status: OutcomeSuccess, Summary: "sent"},
		Evidence:        Evidence{Kind: EvidenceKindEventsJSONL, Pointer: "ptr", Timestamp: now.Unix(), Hash: "sha256:1"},
		Confidence:      0.8,
		ValidationState: ValidationPending,
		ExpiresAt:       now.Add(time.Hour),
		CreatedAt:       now,
	}
	expired := active
	expired.ID = "am_expired"
	expired.ExpiresAt = now.Add(-time.Hour)

	if err := store.AppendMany([]ActionMemory{active, expired}); err != nil {
		t.Fatalf("AppendMany failed: %v", err)
	}

	entries, err := store.Read(Filter{})
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].ID != "am_active" {
		t.Fatalf("entry ID = %s, want am_active", entries[0].ID)
	}
}

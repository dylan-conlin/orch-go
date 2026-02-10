package daemon

import (
	"fmt"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

func TestCleanupResultFields(t *testing.T) {
	result := CleanupResult{
		SessionsDeleted:        1,
		WorkspacesArchived:     2,
		InvestigationsArchived: 3,
		Message:                "test",
	}

	if result.SessionsDeleted != 1 {
		t.Errorf("Expected SessionsDeleted=1, got %d", result.SessionsDeleted)
	}
	if result.WorkspacesArchived != 2 {
		t.Errorf("Expected WorkspacesArchived=2, got %d", result.WorkspacesArchived)
	}
	if result.InvestigationsArchived != 3 {
		t.Errorf("Expected InvestigationsArchived=3, got %d", result.InvestigationsArchived)
	}
}

type fakeSessionReaperClient struct {
	sessions   []opencode.Session
	processing map[string]bool
	deleteErr  map[string]error
	deleted    []string
}

func (f *fakeSessionReaperClient) ListSessions(string) ([]opencode.Session, error) {
	return f.sessions, nil
}

func (f *fakeSessionReaperClient) IsSessionProcessing(sessionID string) bool {
	if f.processing == nil {
		return false
	}
	return f.processing[sessionID]
}

func (f *fakeSessionReaperClient) DeleteSession(sessionID string) error {
	if err, ok := f.deleteErr[sessionID]; ok {
		return err
	}
	f.deleted = append(f.deleted, sessionID)
	return nil
}

func TestReapIdleUntrackedSessions(t *testing.T) {
	now := time.Now()
	old := now.Add(-45 * time.Minute).UnixMilli()
	recent := now.Add(-10 * time.Minute).UnixMilli()

	client := &fakeSessionReaperClient{
		sessions: []opencode.Session{
			{ID: "tracked", Title: "worker [orch-go-abc123]", Time: opencode.SessionTime{Updated: old}},
			{ID: "untracked", Title: "adhoc [orch-go-untracked-123]", Time: opencode.SessionTime{Updated: old}},
			{ID: "untitled", Title: "scratch-session", Time: opencode.SessionTime{Updated: old}},
			{ID: "recent", Title: "recent [orch-go-untracked-999]", Time: opencode.SessionTime{Updated: recent}},
			{ID: "processing", Title: "busy [orch-go-untracked-888]", Time: opencode.SessionTime{Updated: old}},
			{ID: "orchestrator", Title: "meta-orchestrator", Time: opencode.SessionTime{Updated: old}},
		},
		processing: map[string]bool{"processing": true},
	}

	deleted, err := reapIdleUntrackedSessions(client, 30*time.Minute, true, now)
	if err != nil {
		t.Fatalf("reapIdleUntrackedSessions() error = %v", err)
	}

	if deleted != 2 {
		t.Fatalf("reapIdleUntrackedSessions() deleted = %d, want 2", deleted)
	}

	deletedSet := map[string]bool{}
	for _, id := range client.deleted {
		deletedSet[id] = true
	}
	if !deletedSet["untracked"] || !deletedSet["untitled"] {
		t.Fatalf("deleted sessions = %v, want untracked and untitled", client.deleted)
	}
	if deletedSet["tracked"] || deletedSet["recent"] || deletedSet["processing"] || deletedSet["orchestrator"] {
		t.Fatalf("deleted sessions contains unexpected IDs: %v", client.deleted)
	}
}

func TestReapIdleUntrackedSessions_IgnoreNotFoundDelete(t *testing.T) {
	now := time.Now()
	client := &fakeSessionReaperClient{
		sessions: []opencode.Session{
			{ID: "gone", Title: "ghost [orch-go-untracked-1]", Time: opencode.SessionTime{Updated: now.Add(-40 * time.Minute).UnixMilli()}},
		},
		deleteErr: map[string]error{"gone": fmt.Errorf("failed to delete session: status 404: not found")},
	}

	deleted, err := reapIdleUntrackedSessions(client, 30*time.Minute, false, now)
	if err != nil {
		t.Fatalf("reapIdleUntrackedSessions() error = %v", err)
	}
	if deleted != 0 {
		t.Fatalf("reapIdleUntrackedSessions() deleted = %d, want 0", deleted)
	}
}

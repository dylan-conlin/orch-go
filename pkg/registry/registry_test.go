package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_ModeFields(t *testing.T) {
	// Setup temp registry
	tempDir, err := os.MkdirTemp("", "orch-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	regPath := filepath.Join(tempDir, "agent-registry.json")
	reg, err := New(regPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// 1. Test registering a claude mode agent
	claudeAgent := &Agent{
		ID:         "test-claude",
		BeadsID:    "beads-1",
		Mode:       "claude",
		TmuxWindow: "window-1",
		ProjectDir: "/tmp/project",
		Skill:      "investigation",
	}

	if err := reg.Register(claudeAgent); err != nil {
		t.Fatalf("failed to register claude agent: %v", err)
	}

	// 2. Test registering an opencode mode agent
	opencodeAgent := &Agent{
		ID:        "test-opencode",
		BeadsID:   "beads-2",
		Mode:      "opencode",
		SessionID: "ses_123",
	}

	if err := reg.Register(opencodeAgent); err != nil {
		t.Fatalf("failed to register opencode agent: %v", err)
	}

	// Save and reload
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	reg2, err := New(regPath)
	if err != nil {
		t.Fatalf("failed to reload registry: %v", err)
	}

	// Verify claude agent
	a1 := reg2.Find("test-claude")
	if a1 == nil {
		t.Fatal("claude agent not found")
	}
	if a1.Mode != "claude" || a1.TmuxWindow != "window-1" {
		t.Errorf("claude agent fields mismatch: mode=%s, window=%s", a1.Mode, a1.TmuxWindow)
	}

	// Verify opencode agent
	a2 := reg2.Find("test-opencode")
	if a2 == nil {
		t.Fatal("opencode agent not found")
	}
	if a2.Mode != "opencode" || a2.SessionID != "ses_123" {
		t.Errorf("opencode agent fields mismatch: mode=%s, session=%s", a2.Mode, a2.SessionID)
	}

	// 3. Test slot reuse
	reg2.Abandon("test-claude")
	if err := reg2.Save(); err != nil {
		t.Fatalf("failed to save after abandon: %v", err)
	}

	// Re-register test-claude with different metadata
	newClaude := &Agent{
		ID:         "test-claude",
		BeadsID:    "beads-1-new",
		Mode:       "claude",
		TmuxWindow: "window-2",
	}

	if err := reg2.Register(newClaude); err != nil {
		t.Fatalf("failed to re-register claude agent: %v", err)
	}

	a1new := reg2.Find("test-claude")
	if a1new.BeadsID != "beads-1-new" || a1new.TmuxWindow != "window-2" {
		t.Errorf("re-registered agent fields mismatch: beads=%s, window=%s", a1new.BeadsID, a1new.TmuxWindow)
	}
}

func TestRegistry_ListAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "orch-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	regPath := filepath.Join(tempDir, "agent-registry.json")
	reg, err := New(regPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Register agents with different statuses
	agents := []*Agent{
		{ID: "active-1", BeadsID: "b1", Mode: "opencode"},
		{ID: "active-2", BeadsID: "b2", Mode: "opencode"},
	}
	for _, a := range agents {
		if err := reg.Register(a); err != nil {
			t.Fatalf("Register() error: %v", err)
		}
	}

	// Abandon one
	reg.Abandon("active-1")

	// ListAgents should only return active
	active := reg.ListAgents()
	if len(active) != 1 {
		t.Errorf("ListAgents() = %d, want 1", len(active))
	}

	// ListAll should return all
	all := reg.ListAll()
	if len(all) != 2 {
		t.Errorf("ListAll() = %d, want 2", len(all))
	}
}

func TestRegistry_Purge(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "orch-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	regPath := filepath.Join(tempDir, "agent-registry.json")
	reg, err := New(regPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	agents := []*Agent{
		{ID: "tracked-1", BeadsID: "orch-go-111", Mode: "opencode"},
		{ID: "untracked-1", BeadsID: "orch-go-untracked-999", Mode: "opencode"},
		{ID: "tracked-2", BeadsID: "orch-go-222", Mode: "opencode"},
		{ID: "untracked-2", BeadsID: "orch-go-untracked-888", Mode: "opencode"},
	}
	for _, a := range agents {
		if err := reg.Register(a); err != nil {
			t.Fatalf("Register() error: %v", err)
		}
	}

	// Purge untracked entries
	removed := reg.Purge(func(a *Agent) bool {
		return len(a.BeadsID) > 0 && a.BeadsID[len(a.BeadsID)-3:] != "111" && a.BeadsID[len(a.BeadsID)-3:] != "222"
	})
	if removed != 2 {
		t.Errorf("Purge() removed %d, want 2", removed)
	}

	// Save and reload
	if err := reg.SaveSkipMerge(); err != nil {
		t.Fatalf("SaveSkipMerge() error: %v", err)
	}

	reg2, err := New(regPath)
	if err != nil {
		t.Fatalf("New() reload error: %v", err)
	}

	all := reg2.ListAll()
	if len(all) != 2 {
		t.Fatalf("After purge reload: ListAll() = %d, want 2", len(all))
	}

	// Verify correct entries survived
	for _, a := range all {
		if a.ID != "tracked-1" && a.ID != "tracked-2" {
			t.Errorf("Unexpected agent survived: %s", a.ID)
		}
	}
}

func TestRegistry_PurgeAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "orch-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	regPath := filepath.Join(tempDir, "agent-registry.json")
	reg, err := New(regPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	if err := reg.Register(&Agent{ID: "a1", BeadsID: "b1", Mode: "opencode"}); err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	// Purge everything
	removed := reg.Purge(func(a *Agent) bool { return true })
	if removed != 1 {
		t.Errorf("Purge(all) removed %d, want 1", removed)
	}

	all := reg.ListAll()
	if len(all) != 0 {
		t.Errorf("After purge all: ListAll() = %d, want 0", len(all))
	}
}

func TestRegistry_PurgeNone(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "orch-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	regPath := filepath.Join(tempDir, "agent-registry.json")
	reg, err := New(regPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	if err := reg.Register(&Agent{ID: "a1", BeadsID: "b1", Mode: "opencode"}); err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	// Purge nothing
	removed := reg.Purge(func(a *Agent) bool { return false })
	if removed != 0 {
		t.Errorf("Purge(none) removed %d, want 0", removed)
	}

	all := reg.ListAll()
	if len(all) != 1 {
		t.Errorf("After purge none: ListAll() = %d, want 1", len(all))
	}
}

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

package main

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/agent"
)

// TestAdaptersImplementInterfaces verifies that all adapters satisfy
// the pkg/agent lifecycle interfaces at compile time. The compile-time
// checks in lifecycle_adapters.go (var _ agent.X = (*adapter)(nil))
// already enforce this, but this test documents the expectation.
func TestAdaptersImplementInterfaces(t *testing.T) {
	// These type assertions verify interface compliance.
	// They would fail at compile time if any method is missing.
	var _ agent.BeadsClient = (*beadsAdapter)(nil)
	var _ agent.OpenCodeClient = (*openCodeAdapter)(nil)
	var _ agent.TmuxClient = (*tmuxAdapter)(nil)
	var _ agent.EventLogger = (*eventLoggerAdapter)(nil)
	var _ agent.WorkspaceManager = (*workspaceAdapter)(nil)
}

// TestBuildLifecycleManager verifies the factory constructs a valid manager.
func TestBuildLifecycleManager(t *testing.T) {
	lm := buildLifecycleManager("/tmp/project", "http://localhost:4096", "test-agent", "proj-123")
	if lm == nil {
		t.Fatal("buildLifecycleManager returned nil")
	}
}

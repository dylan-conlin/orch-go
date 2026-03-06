package main

import (
	"os"
	"testing"
	"time"
)

func TestContextBroadcaster_SubscribeAndBroadcast(t *testing.T) {
	b := &contextBroadcaster{
		clients: make(map[chan ContextAPIResponse]struct{}),
	}

	// Subscribe two clients
	ch1 := b.subscribe()
	ch2 := b.subscribe()

	// Broadcast a context change
	ctx := ContextAPIResponse{
		Cwd:        "/Users/test/project",
		ProjectDir: "/Users/test/project",
		Project:    "project",
	}
	b.broadcast(ctx)

	// Both clients should receive the event
	select {
	case received := <-ch1:
		if received.Project != "project" {
			t.Errorf("ch1: expected project 'project', got %q", received.Project)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("ch1: timed out waiting for broadcast")
	}

	select {
	case received := <-ch2:
		if received.Project != "project" {
			t.Errorf("ch2: expected project 'project', got %q", received.Project)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("ch2: timed out waiting for broadcast")
	}

	// Unsubscribe ch1 and broadcast again - only ch2 should receive
	b.unsubscribe(ch1)

	ctx2 := ContextAPIResponse{Project: "other-project"}
	b.broadcast(ctx2)

	select {
	case received := <-ch2:
		if received.Project != "other-project" {
			t.Errorf("ch2: expected 'other-project', got %q", received.Project)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("ch2: timed out waiting for second broadcast")
	}

	b.unsubscribe(ch2)
}

func TestContextBroadcaster_NonBlockingBroadcast(t *testing.T) {
	b := &contextBroadcaster{
		clients: make(map[chan ContextAPIResponse]struct{}),
	}

	// Subscribe a client but don't read from it
	ch := b.subscribe()

	// Fill the buffer (channel has buffer of 1)
	b.broadcast(ContextAPIResponse{Project: "first"})

	// Second broadcast should not block (dropped because channel is full)
	done := make(chan struct{})
	go func() {
		b.broadcast(ContextAPIResponse{Project: "second"})
		close(done)
	}()

	select {
	case <-done:
		// Good - broadcast didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("broadcast blocked on full channel")
	}

	// The channel should have the first message
	select {
	case received := <-ch:
		if received.Project != "first" {
			t.Errorf("expected 'first', got %q", received.Project)
		}
	default:
		t.Error("expected message in channel")
	}

	b.unsubscribe(ch)
}

func TestContextCache_Invalidate(t *testing.T) {
	c := &contextCache{
		ttl: 10 * time.Second,
	}

	// Set cached value
	c.mu.Lock()
	c.cwd = "/some/path"
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	// Verify cache is valid
	c.mu.RLock()
	if time.Since(c.fetchedAt) >= c.ttl {
		t.Error("cache should be valid before invalidation")
	}
	c.mu.RUnlock()

	// Invalidate
	c.invalidate()

	// Verify cache is now stale
	c.mu.RLock()
	if c.fetchedAt.IsZero() == false && time.Since(c.fetchedAt) < c.ttl {
		t.Error("cache should be stale after invalidation")
	}
	c.mu.RUnlock()
}

func TestBuildContextResponse(t *testing.T) {
	resp := buildContextResponse(
		"/Users/test/Documents/personal/orch-go/cmd/orch",
		"/Users/test/Documents/personal/orch-go",
	)

	if resp.Project != "orch-go" {
		t.Errorf("expected project 'orch-go', got %q", resp.Project)
	}

	if resp.Cwd != "/Users/test/Documents/personal/orch-go/cmd/orch" {
		t.Errorf("unexpected cwd: %q", resp.Cwd)
	}

	// orch-go should have included projects from multi-project config
	if len(resp.IncludedProjects) < 2 {
		t.Errorf("expected multiple included projects for orch-go, got %d", len(resp.IncludedProjects))
	}
}

func TestBuildContextResponse_BeadsPrefixDiffersFromDirName(t *testing.T) {
	// Create a temp dir simulating scs-special-projects with issue-prefix: scs-sp
	tmpDir := t.TempDir()
	projectDir := tmpDir + "/scs-special-projects"
	beadsDir := projectDir + "/.beads"

	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beadsDir+"/config.yaml", []byte("issue-prefix: \"scs-sp\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := buildContextResponse(projectDir, projectDir)

	if resp.Project != "scs-special-projects" {
		t.Errorf("expected project 'scs-special-projects', got %q", resp.Project)
	}

	// IncludedProjects should contain both the dir name and the beads prefix
	found := false
	for _, p := range resp.IncludedProjects {
		if p == "scs-sp" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'scs-sp' in included_projects, got %v", resp.IncludedProjects)
	}

	// Should also contain the dir name itself
	foundDir := false
	for _, p := range resp.IncludedProjects {
		if p == "scs-special-projects" {
			foundDir = true
			break
		}
	}
	if !foundDir {
		t.Errorf("expected 'scs-special-projects' in included_projects, got %v", resp.IncludedProjects)
	}
}

func TestBuildContextResponse_BeadsPrefixSameAsDirName(t *testing.T) {
	// When beads prefix matches dir name, no extra entry should be added
	tmpDir := t.TempDir()
	projectDir := tmpDir + "/my-project"
	beadsDir := projectDir + "/.beads"

	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beadsDir+"/config.yaml", []byte("issue-prefix: \"my-project\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := buildContextResponse(projectDir, projectDir)

	if resp.Project != "my-project" {
		t.Errorf("expected project 'my-project', got %q", resp.Project)
	}

	// Should only have "my-project" (no duplicate)
	if len(resp.IncludedProjects) != 1 {
		t.Errorf("expected 1 included project, got %d: %v", len(resp.IncludedProjects), resp.IncludedProjects)
	}
}

func TestReadBeadsIssuePrefix(t *testing.T) {
	t.Run("reads prefix from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := tmpDir + "/.beads"
		os.MkdirAll(beadsDir, 0o755)
		os.WriteFile(beadsDir+"/config.yaml", []byte("issue-prefix: \"scs-sp\"\n"), 0o644)

		prefix := readBeadsIssuePrefix(tmpDir)
		if prefix != "scs-sp" {
			t.Errorf("expected 'scs-sp', got %q", prefix)
		}
	})

	t.Run("returns empty for no beads dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		prefix := readBeadsIssuePrefix(tmpDir)
		if prefix != "" {
			t.Errorf("expected empty, got %q", prefix)
		}
	})

	t.Run("returns empty for no config", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.MkdirAll(tmpDir+"/.beads", 0o755)
		prefix := readBeadsIssuePrefix(tmpDir)
		if prefix != "" {
			t.Errorf("expected empty, got %q", prefix)
		}
	})

	t.Run("handles unquoted prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		beadsDir := tmpDir + "/.beads"
		os.MkdirAll(beadsDir, 0o755)
		os.WriteFile(beadsDir+"/config.yaml", []byte("issue-prefix: pw\n"), 0o644)

		prefix := readBeadsIssuePrefix(tmpDir)
		if prefix != "pw" {
			t.Errorf("expected 'pw', got %q", prefix)
		}
	})
}

package beads

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPool(t *testing.T) {
	pool := NewPool()
	if pool == nil {
		t.Fatal("NewPool() returned nil")
	}
	if pool.clients == nil {
		t.Error("pool.clients map is nil")
	}
}

func TestPoolGetOrCreate_Default(t *testing.T) {
	pool := NewPool()

	// Default (empty) project should use DefaultDir
	tempDir := t.TempDir()
	DefaultDir = tempDir

	// GetOrCreate without socket should return nil client (not an error)
	// because the pool gracefully handles missing daemons
	client := pool.GetOrCreate("")
	// Should be nil since no socket exists
	if client != nil {
		t.Logf("GetOrCreate returned client (daemon may be running): %v", client)
	}
}

func TestPoolGetOrCreate_SameProject(t *testing.T) {
	pool := NewPool()

	// Create temp dir with fake .beads structure
	tempDir := t.TempDir()
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Multiple calls with same project should return same client reference
	client1 := pool.GetOrCreate(tempDir)
	client2 := pool.GetOrCreate(tempDir)

	// Both should be nil or both should be same pointer
	if client1 != client2 {
		t.Error("GetOrCreate should return same client for same project")
	}
}

func TestPoolGetOrCreate_DifferentProjects(t *testing.T) {
	pool := NewPool()

	// Create two temp dirs
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Create .beads directories
	if err := os.MkdirAll(filepath.Join(tempDir1, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir2, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Should track different projects separately
	_ = pool.GetOrCreate(tempDir1)
	_ = pool.GetOrCreate(tempDir2)

	// Pool should have entries for both directories
	pool.mu.Lock()
	hasFirst := pool.clients[tempDir1] != nil || pool.attempted[tempDir1]
	hasSecond := pool.clients[tempDir2] != nil || pool.attempted[tempDir2]
	pool.mu.Unlock()

	if !hasFirst {
		t.Error("pool should track first directory")
	}
	if !hasSecond {
		t.Error("pool should track second directory")
	}
}

func TestPoolStats(t *testing.T) {
	pool := NewPool()

	// Create temp dirs
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Create .beads directories
	if err := os.MkdirAll(filepath.Join(tempDir1, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir2, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Access two different projects
	_ = pool.GetOrCreate(tempDir1)
	_ = pool.GetOrCreate(tempDir2)

	// Get stats
	connCount, attemptCount, dirs := pool.Stats()

	// Both should be attempted (no daemon so likely not connected)
	if attemptCount < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attemptCount)
	}

	// Should have directories tracked
	if len(dirs) < 2 {
		t.Errorf("expected at least 2 directories tracked, got %d", len(dirs))
	}

	t.Logf("Pool stats: connected=%d, attempted=%d, dirs=%v", connCount, attemptCount, dirs)
}

func TestPoolCloseAll(t *testing.T) {
	pool := NewPool()

	// Create a temp dir
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Access the project
	_ = pool.GetOrCreate(tempDir)

	// Close all
	pool.CloseAll()

	// Pool should be empty after close
	pool.mu.Lock()
	count := len(pool.clients)
	pool.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 clients after CloseAll, got %d", count)
	}
}

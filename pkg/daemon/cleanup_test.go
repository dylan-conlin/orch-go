package daemon

import (
	"testing"
	"time"
)

func TestRunPeriodicCleanupRunsWhenDue(t *testing.T) {
	called := 0
	d := &Daemon{
		Config: Config{
			CleanupEnabled:  true,
			CleanupInterval: time.Minute,
		},
		cleanupFunc: func(config Config) (int, string, error) {
			called++
			return 2, "Closed 2 stale tmux windows", nil
		},
	}

	result := d.RunPeriodicCleanup()
	if called != 1 {
		t.Fatalf("RunPeriodicCleanup should call cleanup func once, got %d", called)
	}
	if result == nil {
		t.Fatal("RunPeriodicCleanup should return result when due")
	}
	if result.Deleted != 2 {
		t.Fatalf("CleanupResult.Deleted = %d, want 2", result.Deleted)
	}
	if result.Message == "" {
		t.Fatal("CleanupResult.Message should not be empty")
	}
	if d.lastCleanup.IsZero() {
		t.Fatal("lastCleanup should be updated after successful cleanup")
	}
}

func TestRunPeriodicCleanupSkipsWhenNotDue(t *testing.T) {
	called := 0
	d := &Daemon{
		Config: Config{
			CleanupEnabled:  true,
			CleanupInterval: time.Hour,
		},
		lastCleanup: time.Now(),
		cleanupFunc: func(config Config) (int, string, error) {
			called++
			return 1, "Closed 1 stale tmux window", nil
		},
	}

	result := d.RunPeriodicCleanup()
	if result != nil {
		t.Fatal("RunPeriodicCleanup should return nil when cleanup is not due")
	}
	if called != 0 {
		t.Fatalf("RunPeriodicCleanup should not call cleanup func, got %d calls", called)
	}
}

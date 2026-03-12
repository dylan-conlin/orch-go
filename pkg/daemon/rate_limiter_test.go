package daemon

import (
	"strings"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	r := NewRateLimiter(20)

	if r.MaxPerHour != 20 {
		t.Errorf("NewRateLimiter(20).MaxPerHour = %d, want 20", r.MaxPerHour)
	}
	if len(r.SpawnHistory) != 0 {
		t.Errorf("NewRateLimiter(20).SpawnHistory should be empty, got %d entries", len(r.SpawnHistory))
	}
	if r.nowFunc == nil {
		t.Error("NewRateLimiter(20).nowFunc should not be nil")
	}
}

func TestRateLimiter_CanSpawn_NoLimit(t *testing.T) {
	r := NewRateLimiter(0) // No limit

	canSpawn, count, msg := r.CanSpawn()
	if !canSpawn {
		t.Error("CanSpawn() should return true when no limit is set")
	}
	if count != 0 {
		t.Errorf("CanSpawn() count = %d, want 0 (no tracking)", count)
	}
	if msg != "" {
		t.Errorf("CanSpawn() msg = %q, want empty", msg)
	}
}

func TestRateLimiter_CanSpawn_BelowLimit(t *testing.T) {
	r := NewRateLimiter(5)

	// Record 3 spawns
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	canSpawn, count, msg := r.CanSpawn()
	if !canSpawn {
		t.Error("CanSpawn() should return true when below limit")
	}
	if count != 3 {
		t.Errorf("CanSpawn() count = %d, want 3", count)
	}
	if msg != "" {
		t.Errorf("CanSpawn() msg = %q, want empty", msg)
	}
}

func TestRateLimiter_CanSpawn_AtLimit(t *testing.T) {
	r := NewRateLimiter(3)

	// Record exactly 3 spawns
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	canSpawn, count, msg := r.CanSpawn()
	if canSpawn {
		t.Error("CanSpawn() should return false when at limit")
	}
	if count != 3 {
		t.Errorf("CanSpawn() count = %d, want 3", count)
	}
	if msg == "" {
		t.Error("CanSpawn() should return a message when at limit")
	}
}

func TestRateLimiter_CanSpawn_ExpiredHistory(t *testing.T) {
	r := NewRateLimiter(3)

	// Use a mock time function
	baseTime := time.Now()
	r.nowFunc = func() time.Time { return baseTime }

	// Record 3 spawns at base time
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	// Move time forward by more than an hour
	r.nowFunc = func() time.Time { return baseTime.Add(61 * time.Minute) }

	// Old spawns should be expired
	canSpawn, count, _ := r.CanSpawn()
	if !canSpawn {
		t.Error("CanSpawn() should return true when old spawns are expired")
	}
	if count != 0 {
		t.Errorf("CanSpawn() count = %d, want 0 (expired)", count)
	}
}

func TestRateLimiter_RecordSpawn(t *testing.T) {
	r := NewRateLimiter(10)

	r.RecordSpawn()
	if len(r.SpawnHistory) != 1 {
		t.Errorf("RecordSpawn() should add one entry, got %d", len(r.SpawnHistory))
	}

	r.RecordSpawn()
	r.RecordSpawn()
	if len(r.SpawnHistory) != 3 {
		t.Errorf("RecordSpawn() should have 3 entries, got %d", len(r.SpawnHistory))
	}
}

func TestRateLimiter_SpawnsRemaining(t *testing.T) {
	tests := []struct {
		name     string
		max      int
		spawns   int
		wantLeft int
	}{
		{"no limit", 0, 10, 100},
		{"none used", 5, 0, 5},
		{"some used", 10, 3, 7},
		{"all used", 5, 5, 0},
		{"over limit", 3, 5, 0}, // Can't have negative remaining
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRateLimiter(tt.max)
			for i := 0; i < tt.spawns; i++ {
				r.RecordSpawn()
			}

			got := r.SpawnsRemaining()
			if got != tt.wantLeft {
				t.Errorf("SpawnsRemaining() = %d, want %d", got, tt.wantLeft)
			}
		})
	}
}

func TestRateLimiter_Status(t *testing.T) {
	r := NewRateLimiter(10)
	for i := 0; i < 3; i++ {
		r.RecordSpawn()
	}

	status := r.Status()
	if status.MaxPerHour != 10 {
		t.Errorf("Status().MaxPerHour = %d, want 10", status.MaxPerHour)
	}
	if status.SpawnsLastHour != 3 {
		t.Errorf("Status().SpawnsLastHour = %d, want 3", status.SpawnsLastHour)
	}
	if status.SpawnsRemaining != 7 {
		t.Errorf("Status().SpawnsRemaining = %d, want 7", status.SpawnsRemaining)
	}
	if status.LimitReached {
		t.Error("Status().LimitReached should be false")
	}
}

func TestDaemon_OnceExcluding_RateLimited(t *testing.T) {
	d := &Daemon{
		Config: Config{MaxSpawnsPerHour: 2},
		Issues: &mockIssueQuerier{ListReadyIssuesFunc: func() ([]Issue, error) {
			return []Issue{
				{ID: "proj-1", Title: "First", Priority: 0, IssueType: "feature", Status: "open"},
			}, nil
		}},
		Spawner:       &mockSpawner{SpawnWorkFunc: func(id, model, workdir, account string) error { return nil }},
		StatusUpdater: &mockIssueUpdater{UpdateStatusFunc: func(beadsID string, status string) error {
			return nil // Mock: always succeed
		}},
	}
	d.RateLimiter = NewRateLimiter(2)

	// First spawn should succeed
	result, err := d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("First spawn should be processed")
	}

	// Second spawn should succeed
	result, err = d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if !result.Processed {
		t.Error("Second spawn should be processed")
	}

	// Third spawn should be rate limited
	result, err = d.OnceExcluding(nil)
	if err != nil {
		t.Fatalf("OnceExcluding() unexpected error: %v", err)
	}
	if result.Processed {
		t.Error("Third spawn should be rate limited")
	}
	if result.Message == "" || !strings.Contains(result.Message, "Rate limited") {
		t.Errorf("Rate limited message expected, got: %q", result.Message)
	}
}

package daemon

import (
	"testing"
	"time"
)

func TestPeriodicScheduler_IsDue_UnregisteredTask(t *testing.T) {
	s := NewPeriodicScheduler()
	if s.IsDue("nonexistent") {
		t.Error("IsDue should return false for unregistered task")
	}
}

func TestPeriodicScheduler_IsDue_DisabledTask(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", false, time.Hour)
	if s.IsDue("cleanup") {
		t.Error("IsDue should return false for disabled task")
	}
}

func TestPeriodicScheduler_IsDue_ZeroInterval(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, 0)
	if s.IsDue("cleanup") {
		t.Error("IsDue should return false for zero interval")
	}
}

func TestPeriodicScheduler_IsDue_NeverRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)
	if !s.IsDue("cleanup") {
		t.Error("IsDue should return true when task has never run")
	}
}

func TestPeriodicScheduler_IsDue_IntervalElapsed(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)
	s.tasks["cleanup"].lastRun = time.Now().Add(-2 * time.Hour)
	if !s.IsDue("cleanup") {
		t.Error("IsDue should return true when interval has elapsed")
	}
}

func TestPeriodicScheduler_IsDue_IntervalNotElapsed(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)
	s.tasks["cleanup"].lastRun = time.Now().Add(-30 * time.Minute)
	if s.IsDue("cleanup") {
		t.Error("IsDue should return false when interval has not elapsed")
	}
}

func TestPeriodicScheduler_MarkRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)

	if !s.IsDue("cleanup") {
		t.Fatal("Should be due before first run")
	}

	s.MarkRun("cleanup")

	if s.IsDue("cleanup") {
		t.Error("Should not be due immediately after MarkRun")
	}
}

func TestPeriodicScheduler_LastRunTime_NeverRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)

	if !s.LastRunTime("cleanup").IsZero() {
		t.Error("LastRunTime should be zero when never run")
	}
}

func TestPeriodicScheduler_LastRunTime_AfterRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)

	before := time.Now()
	s.MarkRun("cleanup")
	after := time.Now()

	last := s.LastRunTime("cleanup")
	if last.Before(before) || last.After(after) {
		t.Errorf("LastRunTime = %v, want between %v and %v", last, before, after)
	}
}

func TestPeriodicScheduler_LastRunTime_Unregistered(t *testing.T) {
	s := NewPeriodicScheduler()
	if !s.LastRunTime("nonexistent").IsZero() {
		t.Error("LastRunTime should return zero for unregistered task")
	}
}

func TestPeriodicScheduler_NextRunTime_Disabled(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", false, time.Hour)

	if !s.NextRunTime("cleanup").IsZero() {
		t.Error("NextRunTime should return zero when disabled")
	}
}

func TestPeriodicScheduler_NextRunTime_NeverRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)

	next := s.NextRunTime("cleanup")
	if time.Until(next) > time.Second {
		t.Error("NextRunTime should return ~now when never run")
	}
}

func TestPeriodicScheduler_NextRunTime_AfterRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)

	now := time.Now()
	s.tasks["cleanup"].lastRun = now

	next := s.NextRunTime("cleanup")
	expected := now.Add(time.Hour)
	if next.Sub(expected).Abs() > time.Second {
		t.Errorf("NextRunTime = %v, want ~%v", next, expected)
	}
}

func TestPeriodicScheduler_NextRunTime_Unregistered(t *testing.T) {
	s := NewPeriodicScheduler()
	if !s.NextRunTime("nonexistent").IsZero() {
		t.Error("NextRunTime should return zero for unregistered task")
	}
}

func TestPeriodicScheduler_MarkRun_Unregistered(t *testing.T) {
	s := NewPeriodicScheduler()
	// Should not panic
	s.MarkRun("nonexistent")
}

func TestPeriodicScheduler_MultipleTasks(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("fast", true, time.Minute)
	s.Register("slow", true, time.Hour)

	// Both due initially
	if !s.IsDue("fast") {
		t.Error("fast should be due")
	}
	if !s.IsDue("slow") {
		t.Error("slow should be due")
	}

	// Mark fast as run
	s.MarkRun("fast")
	if s.IsDue("fast") {
		t.Error("fast should not be due after MarkRun")
	}
	if !s.IsDue("slow") {
		t.Error("slow should still be due")
	}
}

func TestPeriodicScheduler_SetLastRun(t *testing.T) {
	s := NewPeriodicScheduler()
	s.Register("cleanup", true, time.Hour)

	past := time.Now().Add(-2 * time.Hour)
	s.SetLastRun("cleanup", past)

	if !s.LastRunTime("cleanup").Equal(past) {
		t.Errorf("SetLastRun didn't work: got %v, want %v", s.LastRunTime("cleanup"), past)
	}
	if !s.IsDue("cleanup") {
		t.Error("Should be due after setting lastRun to 2 hours ago")
	}
}

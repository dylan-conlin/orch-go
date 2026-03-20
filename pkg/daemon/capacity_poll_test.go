package daemon

import (
	"errors"
	"testing"
	"time"
)

type mockCapacityPollService struct {
	count int
	err   error
}

func (m *mockCapacityPollService) PollAndCache() (int, error) {
	return m.count, m.err
}

func TestRunPeriodicCapacityPoll_NotDue(t *testing.T) {
	d := &Daemon{
		Scheduler: NewPeriodicScheduler(),
	}
	d.Scheduler.Register(TaskCapacityPoll, true, 5*time.Minute)
	d.Scheduler.SetLastRun(TaskCapacityPoll, time.Now()) // just ran

	result := d.RunPeriodicCapacityPoll()
	if result != nil {
		t.Error("expected nil when not due")
	}
}

func TestRunPeriodicCapacityPoll_Success(t *testing.T) {
	d := &Daemon{
		Scheduler:    NewPeriodicScheduler(),
		CapacityPoll: &mockCapacityPollService{count: 2},
	}
	d.Scheduler.Register(TaskCapacityPoll, true, 5*time.Minute)

	result := d.RunPeriodicCapacityPoll()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.AccountCount != 2 {
		t.Errorf("AccountCount = %d, want 2", result.AccountCount)
	}
	if result.Message != "capacity cache updated" {
		t.Errorf("Message = %q, want %q", result.Message, "capacity cache updated")
	}
}

func TestRunPeriodicCapacityPoll_Error(t *testing.T) {
	d := &Daemon{
		Scheduler:    NewPeriodicScheduler(),
		CapacityPoll: &mockCapacityPollService{count: 1, err: errors.New("api failure")},
	}
	d.Scheduler.Register(TaskCapacityPoll, true, 5*time.Minute)

	result := d.RunPeriodicCapacityPoll()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
	if result.AccountCount != 1 {
		t.Errorf("AccountCount = %d, want 1", result.AccountCount)
	}
}

func TestRunPeriodicCapacityPoll_MarksRun(t *testing.T) {
	d := &Daemon{
		Scheduler:    NewPeriodicScheduler(),
		CapacityPoll: &mockCapacityPollService{count: 2},
	}
	d.Scheduler.Register(TaskCapacityPoll, true, 5*time.Minute)

	d.RunPeriodicCapacityPoll()

	if d.Scheduler.LastRunTime(TaskCapacityPoll).IsZero() {
		t.Error("expected LastRunTime to be set after running")
	}
}

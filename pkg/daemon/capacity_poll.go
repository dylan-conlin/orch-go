package daemon

import (
	"github.com/dylan-conlin/orch-go/pkg/account"
)

// CapacityPollResult holds the result of a periodic capacity poll.
type CapacityPollResult struct {
	Error        error
	Message      string
	AccountCount int
}

// CapacityPollService abstracts the capacity polling for testability.
type CapacityPollService interface {
	PollAndCache() (int, error)
}

// defaultCapacityPollService polls the Anthropic API and writes to the file cache.
type defaultCapacityPollService struct{}

func (s *defaultCapacityPollService) PollAndCache() (int, error) {
	accounts, err := account.ListAccountsWithCapacity()
	if err != nil {
		return 0, err
	}

	cachePath := account.DefaultCapacityFileCachePath()
	if err := account.WriteCapacityFileCache(cachePath, accounts); err != nil {
		return len(accounts), err
	}

	return len(accounts), nil
}

// RunPeriodicCapacityPoll polls account capacity and writes to file cache.
// Returns nil if not due.
func (d *Daemon) RunPeriodicCapacityPoll() *CapacityPollResult {
	if !d.Scheduler.IsDue(TaskCapacityPoll) {
		return nil
	}

	svc := d.CapacityPoll
	if svc == nil {
		svc = &defaultCapacityPollService{}
	}

	count, err := svc.PollAndCache()
	d.Scheduler.MarkRun(TaskCapacityPoll)

	if err != nil {
		return &CapacityPollResult{
			Error:        err,
			Message:      err.Error(),
			AccountCount: count,
		}
	}

	return &CapacityPollResult{
		AccountCount: count,
		Message:      "capacity cache updated",
	}
}

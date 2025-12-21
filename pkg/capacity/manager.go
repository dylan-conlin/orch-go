// Package capacity provides multi-account capacity coordination for spawning agents.
package capacity

import (
	"context"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
)

// Event types for capacity changes
const (
	EventAcquired  = "capacity.acquired"
	EventReleased  = "capacity.released"
	EventLow       = "capacity.low"
	EventExhausted = "capacity.exhausted"
)

// CapacityFunc is a function that fetches capacity for an account.
// This abstraction allows mocking in tests.
type CapacityFunc func(name string) (*account.CapacityInfo, error)

// EventHandler is a function that handles capacity events.
type EventHandler func(Event)

// Event represents a capacity change event.
type Event struct {
	Type      string
	Timestamp time.Time
	Account   string
	Data      map[string]interface{}
}

// Options configures the CapacityManager behavior.
type Options struct {
	// Threshold is the minimum remaining capacity % to use an account (default 20).
	Threshold float64
	// MaxPerAcct is the maximum concurrent agents per account (default 3).
	MaxPerAcct int
	// QueueTimeout is how long to wait in queue when all accounts busy (default 5m).
	QueueTimeout time.Duration
	// RefreshRate is how often to refresh capacity data (default 1m).
	RefreshRate time.Duration
	// EventHandler is called for capacity events.
	EventHandler EventHandler
}

// DefaultOptions returns sensible defaults for CapacityManager.
func DefaultOptions() Options {
	return Options{
		Threshold:    20,
		MaxPerAcct:   3,
		QueueTimeout: 5 * time.Minute,
		RefreshRate:  time.Minute,
	}
}

// AccountState tracks the state of a single account.
type AccountState struct {
	Name          string
	Email         string
	Capacity      *account.CapacityInfo
	InFlight      int
	TotalAcquired int
	LastUsed      time.Time
	LastRefresh   time.Time
}

// Slot represents an acquired agent slot.
type Slot struct {
	AccountName string
	AcquiredAt  time.Time
}

// CapacityManager coordinates spawning across multiple accounts.
type CapacityManager struct {
	mu           sync.Mutex
	cond         *sync.Cond
	accounts     map[string]*AccountState
	accountNames []string // ordered list for consistent iteration
	capacityFunc CapacityFunc
	opts         Options
}

// New creates a new CapacityManager with the given accounts.
// It uses the real account.GetAccountCapacity function.
func New(accountNames []string, opts Options) *CapacityManager {
	return NewWithCapacityFunc(accountNames, account.GetAccountCapacity, opts)
}

// NewWithCapacityFunc creates a CapacityManager with a custom capacity function.
// This is useful for testing with mock capacity data.
func NewWithCapacityFunc(accountNames []string, capacityFunc CapacityFunc, opts Options) *CapacityManager {
	m := &CapacityManager{
		accounts:     make(map[string]*AccountState),
		accountNames: accountNames,
		capacityFunc: capacityFunc,
		opts:         opts,
	}
	m.cond = sync.NewCond(&m.mu)

	// Initialize account states
	for _, name := range accountNames {
		m.accounts[name] = &AccountState{
			Name: name,
		}
	}

	// Initial capacity fetch
	m.refreshCapacityLocked()

	return m
}

// AcquireSlot gets an available slot from the least-used account.
// If all accounts are exhausted, it queues until one becomes available or context times out.
func (m *CapacityManager) AcquireSlot(ctx context.Context) (*Slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		// Check context first
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Try to find an available account
		acct := m.selectBestAccountLocked()
		if acct != nil {
			// Got one - increment in-flight and return
			acct.InFlight++
			acct.TotalAcquired++
			acct.LastUsed = time.Now()

			slot := &Slot{
				AccountName: acct.Name,
				AcquiredAt:  time.Now(),
			}

			// Emit event
			if m.opts.EventHandler != nil {
				m.opts.EventHandler(Event{
					Type:      EventAcquired,
					Timestamp: time.Now(),
					Account:   acct.Name,
					Data: map[string]interface{}{
						"in_flight":     acct.InFlight,
						"five_hour_pct": acct.Capacity.FiveHourRemaining,
					},
				})
			}

			return slot, nil
		}

		// No account available - emit exhausted event
		if m.opts.EventHandler != nil {
			m.opts.EventHandler(Event{
				Type:      EventExhausted,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"reason": "all accounts at capacity",
				},
			})
		}

		// Wait for a release or timeout
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				m.mu.Lock()
				m.cond.Broadcast()
				m.mu.Unlock()
			case <-done:
			}
		}()

		m.cond.Wait()
		close(done)
	}
}

// ReleaseSlot marks a slot as complete, decrementing the in-flight count.
func (m *CapacityManager) ReleaseSlot(slot *Slot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	acct, ok := m.accounts[slot.AccountName]
	if !ok {
		return nil // slot for unknown account - ignore
	}

	if acct.InFlight > 0 {
		acct.InFlight--
	}

	// Emit event
	if m.opts.EventHandler != nil {
		m.opts.EventHandler(Event{
			Type:      EventReleased,
			Timestamp: time.Now(),
			Account:   slot.AccountName,
			Data: map[string]interface{}{
				"in_flight": acct.InFlight,
				"held_for":  time.Since(slot.AcquiredAt).String(),
			},
		})
	}

	// Wake up any waiters
	m.cond.Broadcast()

	return nil
}

// Status returns the current state of all accounts.
func (m *CapacityManager) Status() []AccountState {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]AccountState, 0, len(m.accounts))
	for _, name := range m.accountNames {
		acct := m.accounts[name]
		// Copy to avoid external mutation
		state := AccountState{
			Name:          acct.Name,
			Email:         acct.Email,
			InFlight:      acct.InFlight,
			TotalAcquired: acct.TotalAcquired,
			LastUsed:      acct.LastUsed,
			LastRefresh:   acct.LastRefresh,
		}
		if acct.Capacity != nil {
			// Copy capacity
			cap := *acct.Capacity
			state.Capacity = &cap
		}
		result = append(result, state)
	}

	return result
}

// Refresh updates capacity data for all accounts.
func (m *CapacityManager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.refreshCapacityLocked()
	return nil
}

// selectBestAccountLocked finds the account with most remaining capacity that is:
// 1. Above the threshold
// 2. Below the max in-flight limit
// Must be called with lock held.
func (m *CapacityManager) selectBestAccountLocked() *AccountState {
	var best *AccountState
	var bestRemaining float64 = -1

	for _, name := range m.accountNames {
		acct := m.accounts[name]

		// Skip if no capacity data
		if acct.Capacity == nil {
			continue
		}

		// Skip if capacity has error
		if acct.Capacity.Error != "" {
			continue
		}

		// Skip if at max concurrent
		if acct.InFlight >= m.opts.MaxPerAcct {
			continue
		}

		// Skip if below threshold
		remaining := min(acct.Capacity.FiveHourRemaining, acct.Capacity.SevenDayRemaining)
		if remaining <= m.opts.Threshold {
			continue
		}

		// Select the one with most remaining
		if remaining > bestRemaining {
			best = acct
			bestRemaining = remaining
		}
	}

	return best
}

// refreshCapacityLocked fetches fresh capacity for all accounts.
// Must be called with lock held.
func (m *CapacityManager) refreshCapacityLocked() {
	now := time.Now()

	for _, name := range m.accountNames {
		acct := m.accounts[name]

		capacity, _ := m.capacityFunc(name)
		if capacity != nil {
			acct.Capacity = capacity
			if capacity.Email != "" {
				acct.Email = capacity.Email
			}
		}
		acct.LastRefresh = now
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

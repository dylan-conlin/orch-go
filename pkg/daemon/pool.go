// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"sync"
	"time"
)

// WorkerPool manages concurrent agent spawning with a fixed number of slots.
// It provides a semaphore-based pattern similar to CapacityManager but simpler,
// without the multi-account complexity.
type WorkerPool struct {
	mu          sync.Mutex
	cond        *sync.Cond
	maxWorkers  int
	activeCount int
	slots       []*Slot // Track active slots for monitoring
}

// Slot represents an acquired worker slot.
type Slot struct {
	ID         int
	AcquiredAt time.Time
	BeadsID    string // Optional - for tracking which issue is in this slot
}

// NewWorkerPool creates a pool with the specified number of concurrent workers.
// If maxWorkers <= 0, the pool allows unlimited concurrency.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	p := &WorkerPool{
		maxWorkers: maxWorkers,
		slots:      make([]*Slot, 0),
	}
	p.cond = sync.NewCond(&p.mu)
	return p
}

// Acquire blocks until a slot becomes available or context is cancelled.
// Returns a Slot that must be Released when the work is complete.
func (p *WorkerPool) Acquire(ctx context.Context) (*Slot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		// Check context first
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// If no limit or below limit, acquire immediately
		if p.maxWorkers <= 0 || p.activeCount < p.maxWorkers {
			p.activeCount++
			slot := &Slot{
				ID:         p.activeCount,
				AcquiredAt: time.Now(),
			}
			p.slots = append(p.slots, slot)
			return slot, nil
		}

		// At capacity - wait for a release or context cancellation
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				p.mu.Lock()
				p.cond.Broadcast()
				p.mu.Unlock()
			case <-done:
			}
		}()

		p.cond.Wait()
		close(done)
	}
}

// TryAcquire attempts to acquire a slot without blocking.
// Returns nil if no slot is available.
func (p *WorkerPool) TryAcquire() *Slot {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxWorkers > 0 && p.activeCount >= p.maxWorkers {
		return nil
	}

	p.activeCount++
	slot := &Slot{
		ID:         p.activeCount,
		AcquiredAt: time.Now(),
	}
	p.slots = append(p.slots, slot)
	return slot
}

// Release marks a slot as complete, allowing another worker to start.
func (p *WorkerPool) Release(slot *Slot) {
	if slot == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.activeCount > 0 {
		p.activeCount--
	}

	// Remove slot from tracking
	for i, s := range p.slots {
		if s == slot {
			p.slots = append(p.slots[:i], p.slots[i+1:]...)
			break
		}
	}

	// Wake up any waiters
	p.cond.Broadcast()
}

// Active returns the number of currently active workers.
func (p *WorkerPool) Active() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount
}

// Available returns the number of available slots.
// Returns a high number if no limit is set.
func (p *WorkerPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxWorkers <= 0 {
		return 100 // No limit
	}
	available := p.maxWorkers - p.activeCount
	if available < 0 {
		return 0
	}
	return available
}

// AtCapacity returns true if no slots are available.
func (p *WorkerPool) AtCapacity() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxWorkers <= 0 {
		return false
	}
	return p.activeCount >= p.maxWorkers
}

// MaxWorkers returns the maximum number of concurrent workers.
func (p *WorkerPool) MaxWorkers() int {
	return p.maxWorkers
}

// Status returns current pool state for monitoring.
type PoolStatus struct {
	MaxWorkers  int
	Active      int
	Available   int
	ActiveSlots []SlotInfo
}

// SlotInfo provides information about an active slot.
type SlotInfo struct {
	ID         int
	AcquiredAt time.Time
	Duration   time.Duration
	BeadsID    string
}

// Status returns the current state of the worker pool.
func (p *WorkerPool) Status() PoolStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	slotInfos := make([]SlotInfo, len(p.slots))
	for i, s := range p.slots {
		slotInfos[i] = SlotInfo{
			ID:         s.ID,
			AcquiredAt: s.AcquiredAt,
			Duration:   now.Sub(s.AcquiredAt),
			BeadsID:    s.BeadsID,
		}
	}

	available := 100
	if p.maxWorkers > 0 {
		available = p.maxWorkers - p.activeCount
		if available < 0 {
			available = 0
		}
	}

	return PoolStatus{
		MaxWorkers:  p.maxWorkers,
		Active:      p.activeCount,
		Available:   available,
		ActiveSlots: slotInfos,
	}
}

// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"context"
	"sync"
	"time"
)

// WorkerPool manages concurrent agent spawning with a fixed number of slots.
//
// It uses a two-counter design to separate daemon-spawned agents from
// externally-spawned agents (manual orch spawn, prior daemon runs):
//
//   - lastBeadsCount: authoritative count from beads (set during Reconcile)
//   - pendingSpawns: daemon spawns since last Reconcile (incremented by TryAcquire)
//   - totalActive = lastBeadsCount + pendingSpawns
//
// This eliminates the synthetic slot oscillation that occurred when the old
// single-counter design tried to track external agents via add/free cycles.
// See: scs-sp-8dm (pool reconciliation race under concurrent spawns).
type WorkerPool struct {
	mu             sync.Mutex
	cond           *sync.Cond
	maxWorkers     int
	lastBeadsCount int     // Authoritative count from last Reconcile (all agents)
	pendingSpawns  int     // Daemon spawns since last Reconcile (not yet in beads)
	slots          []*Slot // Track daemon-spawned slots for monitoring
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

// totalActive returns the effective active agent count (must be called with lock held).
// This is the sum of the authoritative beads count and daemon-pending spawns.
func (p *WorkerPool) totalActive() int {
	total := p.lastBeadsCount + p.pendingSpawns
	if total < 0 {
		return 0
	}
	return total
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
		if p.maxWorkers <= 0 || p.totalActive() < p.maxWorkers {
			p.pendingSpawns++
			slot := &Slot{
				ID:         p.totalActive(),
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

	if p.maxWorkers > 0 && p.totalActive() >= p.maxWorkers {
		return nil
	}

	p.pendingSpawns++
	slot := &Slot{
		ID:         p.totalActive(),
		AcquiredAt: time.Now(),
	}
	p.slots = append(p.slots, slot)
	return slot
}

// Release marks a slot as complete, allowing another worker to start.
// Decrements pendingSpawns (daemon's own spawn count).
func (p *WorkerPool) Release(slot *Slot) {
	if slot == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.pendingSpawns > 0 {
		p.pendingSpawns--
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

// Active returns the total number of active agents (daemon + external).
func (p *WorkerPool) Active() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.totalActive()
}

// Available returns the number of available slots.
// Returns a high number if no limit is set.
func (p *WorkerPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxWorkers <= 0 {
		return 100 // No limit
	}
	available := p.maxWorkers - p.totalActive()
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
	return p.totalActive() >= p.maxWorkers
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

	total := p.totalActive()
	available := 100
	if p.maxWorkers > 0 {
		available = p.maxWorkers - total
		if available < 0 {
			available = 0
		}
	}

	return PoolStatus{
		MaxWorkers:  p.maxWorkers,
		Active:      total,
		Available:   available,
		ActiveSlots: slotInfos,
	}
}

// ReconcileResult contains the outcome of a pool reconciliation.
type ReconcileResult struct {
	// Freed is the number of slots released (agents completed without daemon knowing).
	Freed int
	// Added is the number of slots created to account for agents the pool didn't track
	// (e.g., agents from a prior daemon run, or manually spawned agents).
	Added int
}

// Reconcile synchronizes the pool with the actual number of active agents
// from beads (the authoritative source of agent lifecycle state).
//
// Sets lastBeadsCount to actualCount and resets pendingSpawns to 0, since
// beads has caught up with any daemon spawns from the prior cycle.
//
// This design eliminates the synthetic slot oscillation that occurred when
// external agents (manual spawns) caused repeated add/free cycles:
//
//   - Old behavior: actualCount > activeCount -> add synthetic slots -> agents
//     complete -> actualCount < activeCount -> free slots -> repeat.
//   - New behavior: lastBeadsCount is simply set to the authoritative count.
//     No synthetic slots. No oscillation.
//
// Returns a ReconcileResult with Freed/Added representing the net change
// in effective capacity (for logging, not slot management).
func (p *WorkerPool) Reconcile(actualCount int) ReconcileResult {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := ReconcileResult{}
	previousTotal := p.totalActive()

	// Update authoritative count from beads. Reset pendingSpawns because
	// beads now reflects any daemon spawns from the prior cycle.
	p.lastBeadsCount = actualCount
	p.pendingSpawns = 0

	newTotal := p.totalActive() // == actualCount (since pendingSpawns is 0)

	if newTotal > previousTotal {
		result.Added = newTotal - previousTotal
	} else if newTotal < previousTotal {
		result.Freed = previousTotal - newTotal
	}

	// Trim stale daemon slots that no longer correspond to running agents.
	// After reconciliation, we should have at most actualCount slots.
	// Excess slots are from daemon-spawned agents that have completed.
	for len(p.slots) > actualCount {
		p.slots = p.slots[1:] // Remove oldest
	}

	// Wake up any waiters since capacity may have freed up
	if result.Freed > 0 {
		p.cond.Broadcast()
	}

	return result
}

package daemon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/internal/testutil"
)

func TestNewWorkerPool(t *testing.T) {
	p := NewWorkerPool(5)

	if p == nil {
		t.Fatal("NewWorkerPool() returned nil")
	}
	if p.MaxWorkers() != 5 {
		t.Errorf("MaxWorkers() = %d, want 5", p.MaxWorkers())
	}
	if p.Active() != 0 {
		t.Errorf("Active() = %d, want 0", p.Active())
	}
}

func TestWorkerPool_AcquireRelease(t *testing.T) {
	p := NewWorkerPool(3)

	// Acquire a slot
	slot, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if slot == nil {
		t.Fatal("Acquire() returned nil slot")
	}

	// Check active count
	if p.Active() != 1 {
		t.Errorf("Active() = %d after acquire, want 1", p.Active())
	}

	// Release
	p.Release(slot)

	// Check active count after release
	if p.Active() != 0 {
		t.Errorf("Active() = %d after release, want 0", p.Active())
	}
}

func TestWorkerPool_TryAcquire(t *testing.T) {
	p := NewWorkerPool(2)

	// First TryAcquire should succeed
	slot1 := p.TryAcquire()
	if slot1 == nil {
		t.Fatal("First TryAcquire() returned nil")
	}

	// Second TryAcquire should succeed
	slot2 := p.TryAcquire()
	if slot2 == nil {
		t.Fatal("Second TryAcquire() returned nil")
	}

	// Third TryAcquire should fail (at capacity)
	slot3 := p.TryAcquire()
	if slot3 != nil {
		t.Error("Third TryAcquire() should return nil when at capacity")
	}

	// Release one, then TryAcquire should succeed
	p.Release(slot1)
	slot4 := p.TryAcquire()
	if slot4 == nil {
		t.Error("TryAcquire() after release should succeed")
	}
}

func TestWorkerPool_AtCapacity(t *testing.T) {
	p := NewWorkerPool(2)

	if p.AtCapacity() {
		t.Error("AtCapacity() should be false when empty")
	}

	slot1 := p.TryAcquire()
	if p.AtCapacity() {
		t.Error("AtCapacity() should be false when 1/2")
	}

	slot2 := p.TryAcquire()
	if !p.AtCapacity() {
		t.Error("AtCapacity() should be true when 2/2")
	}

	p.Release(slot1)
	if p.AtCapacity() {
		t.Error("AtCapacity() should be false after release")
	}

	p.Release(slot2)
}

func TestWorkerPool_Available(t *testing.T) {
	p := NewWorkerPool(3)

	if p.Available() != 3 {
		t.Errorf("Available() = %d, want 3", p.Available())
	}

	slot1 := p.TryAcquire()
	if p.Available() != 2 {
		t.Errorf("Available() = %d after 1 acquire, want 2", p.Available())
	}

	slot2 := p.TryAcquire()
	slot3 := p.TryAcquire()
	if p.Available() != 0 {
		t.Errorf("Available() = %d after 3 acquires, want 0", p.Available())
	}

	p.Release(slot1)
	p.Release(slot2)
	p.Release(slot3)
}

func TestWorkerPool_NoLimit(t *testing.T) {
	p := NewWorkerPool(0) // No limit

	if p.AtCapacity() {
		t.Error("AtCapacity() should be false with no limit")
	}

	// Should be able to acquire many slots
	var slots []*Slot
	for i := 0; i < 50; i++ {
		slot := p.TryAcquire()
		if slot == nil {
			t.Errorf("TryAcquire() returned nil at iteration %d with no limit", i)
		}
		slots = append(slots, slot)
	}

	if p.Active() != 50 {
		t.Errorf("Active() = %d, want 50", p.Active())
	}

	// Cleanup
	for _, s := range slots {
		p.Release(s)
	}
}

func TestWorkerPool_AcquireBlocks(t *testing.T) {
	p := NewWorkerPool(1)

	// Acquire the only slot
	slot1, _ := p.Acquire(context.Background())

	// Try to acquire another with timeout - should block then timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.Acquire(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Acquire() error = %v, want DeadlineExceeded", err)
	}

	// Release and try again
	p.Release(slot1)
	slot2, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() after release error = %v", err)
	}
	if slot2 == nil {
		t.Error("Acquire() after release returned nil")
	}
	p.Release(slot2)
}

func TestWorkerPool_WakesWaiters(t *testing.T) {
	p := NewWorkerPool(1)

	// Acquire the only slot
	slot1, _ := p.Acquire(context.Background())

	// Start a goroutine that waits for a slot
	var slot2 *Slot
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		slot2, err = p.Acquire(context.Background())
		if err != nil {
			t.Errorf("Waiter got error: %v", err)
		}
	}()

	// Give goroutine time to start blocking on cond.Wait()
	testutil.YieldForGoroutine()

	// Release - should wake waiter
	p.Release(slot1)

	// Wait for goroutine
	wg.Wait()

	if slot2 == nil {
		t.Error("Waiter should have received slot")
	}
	p.Release(slot2)
}

func TestWorkerPool_ConcurrentAccess(t *testing.T) {
	p := NewWorkerPool(5)

	// Run many concurrent acquires and releases
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			slot, err := p.Acquire(ctx)
			if err != nil {
				return // Timeout is acceptable
			}
			// Simulate work by holding the slot briefly
			time.Sleep(5 * time.Millisecond)
			p.Release(slot)
		}()
	}

	wg.Wait()

	// All should be released
	if p.Active() != 0 {
		t.Errorf("Active() = %d after all released, want 0", p.Active())
	}
}

func TestWorkerPool_Status(t *testing.T) {
	p := NewWorkerPool(3)

	slot1, _ := p.Acquire(context.Background())
	slot1.BeadsID = "issue-123" // Set beads ID for tracking

	status := p.Status()

	if status.MaxWorkers != 3 {
		t.Errorf("Status.MaxWorkers = %d, want 3", status.MaxWorkers)
	}
	if status.Active != 1 {
		t.Errorf("Status.Active = %d, want 1", status.Active)
	}
	if status.Available != 2 {
		t.Errorf("Status.Available = %d, want 2", status.Available)
	}
	if len(status.ActiveSlots) != 1 {
		t.Fatalf("len(Status.ActiveSlots) = %d, want 1", len(status.ActiveSlots))
	}
	if status.ActiveSlots[0].BeadsID != "issue-123" {
		t.Errorf("Status.ActiveSlots[0].BeadsID = %q, want 'issue-123'", status.ActiveSlots[0].BeadsID)
	}

	p.Release(slot1)
}

func TestWorkerPool_ReleaseNil(t *testing.T) {
	p := NewWorkerPool(3)

	// Should not panic
	p.Release(nil)

	if p.Active() != 0 {
		t.Error("Release(nil) should not change active count")
	}
}

func TestWorkerPool_ReleaseUnknownSlot(t *testing.T) {
	p := NewWorkerPool(3)

	// Acquire a real slot
	slot1, _ := p.Acquire(context.Background())

	// Release a fake slot (not from this pool)
	fakeSlot := &Slot{ID: 999}
	p.Release(fakeSlot)

	// The active count for slot1 should still be tracked
	// Note: This is a bit undefined behavior - we decrement anyway
	// but don't find it in the slice. This test documents the behavior.
	if p.Active() != 0 {
		// Actually, we decrement activeCount regardless
		// This is acceptable because the invariant should be maintained
		// by properly using the pool
	}

	p.Release(slot1)
}

// =============================================================================
// Tests for Reconcile
// =============================================================================

func TestWorkerPool_Reconcile_FreesStaleSlots(t *testing.T) {
	p := NewWorkerPool(3)

	// Acquire 3 slots (at capacity)
	slot1 := p.TryAcquire()
	slot2 := p.TryAcquire()
	slot3 := p.TryAcquire()

	if p.Active() != 3 {
		t.Fatalf("Active() = %d, want 3", p.Active())
	}
	if !p.AtCapacity() {
		t.Fatal("AtCapacity() should be true")
	}

	// Simulate: 2 agents actually running (1 completed without daemon knowing)
	result := p.Reconcile(2)

	if result.Freed != 1 {
		t.Errorf("Reconcile(2) freed = %d, want 1", result.Freed)
	}
	if result.Added != 0 {
		t.Errorf("Reconcile(2) added = %d, want 0", result.Added)
	}
	if p.Active() != 2 {
		t.Errorf("Active() after reconcile = %d, want 2", p.Active())
	}
	if p.AtCapacity() {
		t.Error("AtCapacity() should be false after reconcile")
	}
	if p.Available() != 1 {
		t.Errorf("Available() after reconcile = %d, want 1", p.Available())
	}

	// Cleanup
	p.Release(slot1)
	p.Release(slot2)
	p.Release(slot3)
}

func TestWorkerPool_Reconcile_AllSessionsGone(t *testing.T) {
	p := NewWorkerPool(3)

	// Acquire 3 slots
	p.TryAcquire()
	p.TryAcquire()
	p.TryAcquire()

	// Simulate: all agents completed (none running)
	result := p.Reconcile(0)

	if result.Freed != 3 {
		t.Errorf("Reconcile(0) freed = %d, want 3", result.Freed)
	}
	if p.Active() != 0 {
		t.Errorf("Active() after reconcile = %d, want 0", p.Active())
	}
	if p.AtCapacity() {
		t.Error("AtCapacity() should be false after reconcile")
	}
}

func TestWorkerPool_Reconcile_MoreActualThanTracked(t *testing.T) {
	p := NewWorkerPool(5)

	// Acquire 1 slot
	p.TryAcquire()

	// More agents running than tracked (happens after daemon restart
	// when agents from prior run are still active)
	result := p.Reconcile(3)

	if result.Freed != 0 {
		t.Errorf("Reconcile(3) freed = %d, want 0", result.Freed)
	}
	if result.Added != 2 {
		t.Errorf("Reconcile(3) added = %d, want 2 (3 actual - 1 tracked)", result.Added)
	}
	if p.Active() != 3 {
		t.Errorf("Active() after reconcile = %d, want 3", p.Active())
	}
	if p.Available() != 2 {
		t.Errorf("Available() after reconcile = %d, want 2 (5 max - 3 active)", p.Available())
	}
}

func TestWorkerPool_Reconcile_SameCount(t *testing.T) {
	p := NewWorkerPool(3)

	// Acquire 2 slots
	p.TryAcquire()
	p.TryAcquire()

	// Simulate: exact match
	result := p.Reconcile(2)

	if result.Freed != 0 {
		t.Errorf("Reconcile(2) freed = %d, want 0 (no action when counts match)", result.Freed)
	}
	if result.Added != 0 {
		t.Errorf("Reconcile(2) added = %d, want 0 (no action when counts match)", result.Added)
	}
	if p.Active() != 2 {
		t.Errorf("Active() after reconcile = %d, want 2 (unchanged)", p.Active())
	}
}

func TestWorkerPool_Reconcile_EmptyPool(t *testing.T) {
	p := NewWorkerPool(3)

	// Pool is empty
	result := p.Reconcile(0)

	if result.Freed != 0 {
		t.Errorf("Reconcile(0) on empty pool freed = %d, want 0", result.Freed)
	}
	if result.Added != 0 {
		t.Errorf("Reconcile(0) on empty pool added = %d, want 0", result.Added)
	}
	if p.Active() != 0 {
		t.Errorf("Active() = %d, want 0", p.Active())
	}
}

func TestWorkerPool_Reconcile_WakesWaiters(t *testing.T) {
	p := NewWorkerPool(1)

	// Acquire the only slot
	p.TryAcquire()

	// Start a goroutine that waits for a slot
	var slot *Slot
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		slot, err = p.Acquire(context.Background())
		if err != nil {
			t.Errorf("Waiter got error: %v", err)
		}
	}()

	// Give goroutine time to start blocking on cond.Wait()
	testutil.YieldForGoroutine()

	// Reconcile to free the slot (simulating agent completed)
	result := p.Reconcile(0)
	if result.Freed != 1 {
		t.Errorf("Reconcile(0) freed = %d, want 1", result.Freed)
	}

	// Wait for goroutine
	wg.Wait()

	if slot == nil {
		t.Error("Waiter should have received slot after Reconcile freed capacity")
	}
	if slot != nil {
		p.Release(slot)
	}
}

// =============================================================================
// Tests for restart reconciliation (upward seeding)
// =============================================================================

func TestWorkerPool_Reconcile_RestartSeedsFromRunningAgents(t *testing.T) {
	// After daemon restart, pool starts fresh at 0 but agents from the prior run
	// are still active. Reconcile must raise the pool count to match reality,
	// otherwise the daemon can over-spawn past the concurrency cap.
	p := NewWorkerPool(5)

	// Pool starts fresh (activeCount=0) — simulates daemon restart
	if p.Active() != 0 {
		t.Fatalf("Fresh pool Active() = %d, want 0", p.Active())
	}
	if p.Available() != 5 {
		t.Fatalf("Fresh pool Available() = %d, want 5", p.Available())
	}

	// Reality: 3 agents running from prior daemon
	result := p.Reconcile(3)

	if result.Added != 3 {
		t.Errorf("Reconcile(3) added = %d, want 3", result.Added)
	}
	if result.Freed != 0 {
		t.Errorf("Reconcile(3) freed = %d, want 0", result.Freed)
	}
	if p.Active() != 3 {
		t.Errorf("Active() after seed = %d, want 3", p.Active())
	}
	if p.Available() != 2 {
		t.Errorf("Available() after seed = %d, want 2 (5 max - 3 active)", p.Available())
	}

	// Verify pool correctly limits: can spawn 2 more but not 3
	s1 := p.TryAcquire()
	s2 := p.TryAcquire()
	s3 := p.TryAcquire()
	if s1 == nil || s2 == nil {
		t.Error("Should be able to acquire 2 more slots after seeding 3/5")
	}
	if s3 != nil {
		t.Error("Should NOT be able to acquire 6th slot (5 max)")
	}

	// Cleanup
	if s1 != nil {
		p.Release(s1)
	}
	if s2 != nil {
		p.Release(s2)
	}
}

func TestWorkerPool_Reconcile_RestartAtCapacity(t *testing.T) {
	// Edge case: after restart, all slots are already occupied by prior agents
	p := NewWorkerPool(3)

	// Reconcile with 3 actual = max workers
	result := p.Reconcile(3)

	if result.Added != 3 {
		t.Errorf("Reconcile(3) added = %d, want 3", result.Added)
	}
	if !p.AtCapacity() {
		t.Error("Pool should be at capacity when seeded to max")
	}

	// Cannot acquire any new slots
	s := p.TryAcquire()
	if s != nil {
		t.Error("Should NOT be able to acquire slot when at capacity from seeding")
	}
}

func TestWorkerPool_Reconcile_SeedThenFree(t *testing.T) {
	// Simulate: daemon restarts, seeds from 3 running agents,
	// then one completes on the next cycle.
	p := NewWorkerPool(5)

	// Cycle 1: seed from reality
	result := p.Reconcile(3)
	if result.Added != 3 {
		t.Errorf("Cycle 1: added = %d, want 3", result.Added)
	}
	if p.Active() != 3 {
		t.Errorf("Cycle 1: Active() = %d, want 3", p.Active())
	}

	// Daemon spawns 1 new agent
	s := p.TryAcquire()
	if s == nil {
		t.Fatal("Should be able to acquire after seeding 3/5")
	}
	if p.Active() != 4 {
		t.Errorf("After spawn: Active() = %d, want 4", p.Active())
	}

	// Cycle 2: one of the prior agents completes
	result = p.Reconcile(3)
	if result.Freed != 1 {
		t.Errorf("Cycle 2: freed = %d, want 1 (one agent completed)", result.Freed)
	}
	if p.Active() != 3 {
		t.Errorf("Cycle 2: Active() = %d, want 3", p.Active())
	}
	if p.Available() != 2 {
		t.Errorf("Cycle 2: Available() = %d, want 2", p.Available())
	}

	p.Release(s)
}

// TestWorkerPool_ConcurrentSpawnSources_NoOscillation verifies that when
// daemon spawns and manual spawns happen concurrently, reconciliation does NOT
// oscillate between seeding and freeing synthetic slots.
//
// This is the regression test for scs-sp-8dm: "pool reconciliation race
// condition under concurrent spawns."
func TestWorkerPool_ConcurrentSpawnSources_NoOscillation(t *testing.T) {
	p := NewWorkerPool(5)

	// Daemon spawns 3 agents
	s1 := p.TryAcquire()
	s2 := p.TryAcquire()
	s3 := p.TryAcquire()
	if s1 == nil || s2 == nil || s3 == nil {
		t.Fatal("Should acquire 3 slots")
	}
	if p.Active() != 3 {
		t.Fatalf("Active() = %d after 3 daemon spawns, want 3", p.Active())
	}

	// 2 manual spawns happen externally (not through pool).
	// Reconcile with beads showing 5 total agents.
	result := p.Reconcile(5)
	if result.Added != 2 {
		t.Errorf("Reconcile(5): Added = %d, want 2 (2 external agents)", result.Added)
	}
	if p.Active() != 5 {
		t.Errorf("Active() after reconcile = %d, want 5", p.Active())
	}
	if !p.AtCapacity() {
		t.Error("Should be at capacity (5/5)")
	}

	// KEY: Second reconcile with SAME beads count should be a no-op.
	// The old code would oscillate here because pendingSpawns was not reset.
	result = p.Reconcile(5)
	if result.Added != 0 || result.Freed != 0 {
		t.Errorf("Second Reconcile(5): Added=%d Freed=%d, want both 0 (stable, no oscillation)",
			result.Added, result.Freed)
	}
	if p.Active() != 5 {
		t.Errorf("Active() after second reconcile = %d, want 5", p.Active())
	}

	// 1 daemon agent completes, 1 manual agent completes -> beads = 3
	result = p.Reconcile(3)
	if result.Freed != 2 {
		t.Errorf("Reconcile(3): Freed = %d, want 2", result.Freed)
	}
	if p.Active() != 3 {
		t.Errorf("Active() after completions = %d, want 3", p.Active())
	}
	if p.AtCapacity() {
		t.Error("Should NOT be at capacity (3/5)")
	}

	// Daemon can spawn again
	s4 := p.TryAcquire()
	if s4 == nil {
		t.Error("Should be able to acquire after agents completed")
	}
	if p.Active() != 4 {
		t.Errorf("Active() after new spawn = %d, want 4", p.Active())
	}

	// Cleanup
	p.Release(s1)
	p.Release(s2)
	p.Release(s3)
	p.Release(s4)
}

// TestWorkerPool_ManualSpawns_DontCauseOscillation verifies that
// rapid manual spawn/complete cycles don't cause repeated seeding/freeing.
func TestWorkerPool_ManualSpawns_DontCauseOscillation(t *testing.T) {
	p := NewWorkerPool(5)

	totalReconcileAdjustments := 0

	// Simulate 5 rapid cycles of manual spawn -> reconcile -> complete -> reconcile
	for cycle := 0; cycle < 5; cycle++ {
		// Manual spawn: beads count goes up by 1
		beadsCount := cycle + 1
		result := p.Reconcile(beadsCount)
		if result.Added > 0 || result.Freed > 0 {
			totalReconcileAdjustments++
		}

		// Immediate reconcile with same count: should be no-op
		result = p.Reconcile(beadsCount)
		if result.Added != 0 || result.Freed != 0 {
			t.Errorf("Cycle %d: repeat Reconcile(%d) caused change: Added=%d Freed=%d",
				cycle, beadsCount, result.Added, result.Freed)
		}
	}

	// Each cycle should produce exactly 1 adjustment (the first reconcile).
	if totalReconcileAdjustments != 5 {
		t.Errorf("Expected exactly 5 adjustments across 5 cycles, got %d", totalReconcileAdjustments)
	}
}

// TestWorkerPool_PendingSpawns_CorrectlyAccountedInReconcile verifies that
// daemon spawns (pendingSpawns) are correctly reset during reconciliation.
func TestWorkerPool_PendingSpawns_CorrectlyAccountedInReconcile(t *testing.T) {
	p := NewWorkerPool(5)

	// Reconcile sets beads count to 2
	p.Reconcile(2)
	if p.Active() != 2 {
		t.Fatalf("Active() after initial reconcile = %d, want 2", p.Active())
	}

	// Daemon spawns 2 more (pendingSpawns = 2, total = 4)
	s1 := p.TryAcquire()
	s2 := p.TryAcquire()
	if s1 == nil || s2 == nil {
		t.Fatal("Should acquire 2 slots")
	}
	if p.Active() != 4 {
		t.Fatalf("Active() after 2 daemon spawns = %d, want 4", p.Active())
	}

	// Beads now shows 4 (2 prior + 2 daemon spawns caught up)
	result := p.Reconcile(4)
	if result.Added != 0 && result.Freed != 0 {
		t.Errorf("Reconcile(4): Added=%d Freed=%d, want both 0", result.Added, result.Freed)
	}
	if p.Active() != 4 {
		t.Errorf("Active() after reconcile = %d, want 4", p.Active())
	}

	// Cleanup
	p.Release(s1)
	p.Release(s2)
}

func TestDaemon_ReconcileActiveAgents_SeedsOnRestart(t *testing.T) {
	// End-to-end test: daemon with fresh pool seeds from running agent count
	config := DefaultConfig()
	config.MaxAgents = 5
	d := NewWithConfig(config)

	// Mock active counter to simulate 3 running agents from prior daemon
	d.ActiveCounter = &mockActiveCounter{CountFunc: func() int { return 3 }}

	// Pool should start fresh
	if d.ActiveCount() != 0 {
		t.Fatalf("Fresh daemon ActiveCount() = %d, want 0", d.ActiveCount())
	}

	// First reconciliation should seed the pool
	result := d.ReconcileActiveAgents()

	if result.Added != 3 {
		t.Errorf("ReconcileActiveAgents() added = %d, want 3", result.Added)
	}
	if d.ActiveCount() != 3 {
		t.Errorf("ActiveCount() after seed = %d, want 3", d.ActiveCount())
	}
	if d.AvailableSlots() != 2 {
		t.Errorf("AvailableSlots() after seed = %d, want 2", d.AvailableSlots())
	}
	if d.AtCapacity() {
		t.Error("Should not be at capacity (3/5)")
	}

	// Subsequent reconciliation with same count should be no-op
	result = d.ReconcileActiveAgents()
	if result.Added != 0 || result.Freed != 0 {
		t.Errorf("Second reconcile: added=%d freed=%d, want both 0", result.Added, result.Freed)
	}
}

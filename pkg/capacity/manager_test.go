package capacity

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
)

// mockCapacityFunc creates a capacity function that returns preset values
func mockCapacityFunc(capacities map[string]*account.CapacityInfo) CapacityFunc {
	return func(name string) (*account.CapacityInfo, error) {
		if c, ok := capacities[name]; ok {
			return c, nil
		}
		return &account.CapacityInfo{Error: "account not found"}, nil
	}
}

func TestNewManager(t *testing.T) {
	accounts := []string{"personal", "work"}
	m := New(accounts, DefaultOptions())

	if m == nil {
		t.Fatal("New() returned nil")
	}

	if len(m.accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(m.accounts))
	}
}

func TestAcquireSlot_SelectsLeastUsedAccount(t *testing.T) {
	accounts := []string{"personal", "work"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 30, SevenDayRemaining: 40}, // 30% remaining
		"work":     {FiveHourRemaining: 60, SevenDayRemaining: 70}, // 60% remaining
	}

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), DefaultOptions())

	slot, err := m.AcquireSlot(context.Background())
	if err != nil {
		t.Fatalf("AcquireSlot() error = %v", err)
	}

	// Should select "work" as it has more remaining capacity
	if slot.AccountName != "work" {
		t.Errorf("Expected account 'work', got %q", slot.AccountName)
	}
}

func TestAcquireSlot_SkipsAccountsAboveThreshold(t *testing.T) {
	accounts := []string{"personal", "work"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 30, SevenDayRemaining: 30}, // 30% remaining (below 80% usage)
		"work":     {FiveHourRemaining: 10, SevenDayRemaining: 10}, // 10% remaining (above 80% usage)
	}

	opts := DefaultOptions()
	opts.Threshold = 20 // Only use accounts with >20% remaining

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	slot, err := m.AcquireSlot(context.Background())
	if err != nil {
		t.Fatalf("AcquireSlot() error = %v", err)
	}

	// Should select "personal" as "work" is below threshold
	if slot.AccountName != "personal" {
		t.Errorf("Expected account 'personal', got %q", slot.AccountName)
	}
}

func TestReleaseSlot_DecrementsInFlight(t *testing.T) {
	accounts := []string{"personal"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50},
	}

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), DefaultOptions())

	slot, err := m.AcquireSlot(context.Background())
	if err != nil {
		t.Fatalf("AcquireSlot() error = %v", err)
	}

	// Check in-flight count
	status := m.Status()
	if status[0].InFlight != 1 {
		t.Errorf("Expected InFlight = 1, got %d", status[0].InFlight)
	}

	// Release
	if err := m.ReleaseSlot(slot); err != nil {
		t.Fatalf("ReleaseSlot() error = %v", err)
	}

	// Check in-flight decreased
	status = m.Status()
	if status[0].InFlight != 0 {
		t.Errorf("Expected InFlight = 0 after release, got %d", status[0].InFlight)
	}
}

func TestAcquireSlot_RespectsMaxPerAccount(t *testing.T) {
	accounts := []string{"personal", "work"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50},
		"work":     {FiveHourRemaining: 40, SevenDayRemaining: 40},
	}

	opts := DefaultOptions()
	opts.MaxPerAcct = 2 // Only 2 concurrent per account

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	// Acquire 2 slots on "personal" (highest capacity)
	slot1, _ := m.AcquireSlot(context.Background())
	slot2, _ := m.AcquireSlot(context.Background())

	if slot1.AccountName != "personal" || slot2.AccountName != "personal" {
		t.Errorf("First 2 slots should be on personal")
	}

	// Third slot should go to "work" since personal is at max
	slot3, err := m.AcquireSlot(context.Background())
	if err != nil {
		t.Fatalf("AcquireSlot() error = %v", err)
	}

	if slot3.AccountName != "work" {
		t.Errorf("Third slot should be on 'work', got %q", slot3.AccountName)
	}
}

func TestAcquireSlot_QueuesWhenExhausted(t *testing.T) {
	accounts := []string{"personal"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50},
	}

	opts := DefaultOptions()
	opts.MaxPerAcct = 1
	opts.QueueTimeout = 100 * time.Millisecond

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	// Acquire the only slot
	slot1, _ := m.AcquireSlot(context.Background())

	// Try to acquire another - should queue and timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := m.AcquireSlot(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}

	// Release first slot
	m.ReleaseSlot(slot1)

	// Now should succeed
	slot2, err := m.AcquireSlot(context.Background())
	if err != nil {
		t.Fatalf("After release, AcquireSlot() error = %v", err)
	}
	if slot2.AccountName != "personal" {
		t.Errorf("Expected personal, got %q", slot2.AccountName)
	}
}

func TestAcquireSlot_WakesWaitersOnRelease(t *testing.T) {
	accounts := []string{"personal"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50},
	}

	opts := DefaultOptions()
	opts.MaxPerAcct = 1
	opts.QueueTimeout = 5 * time.Second

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	// Acquire the only slot
	slot1, _ := m.AcquireSlot(context.Background())

	// Start a goroutine that will wait for slot
	var slot2 *Slot
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		slot2, err = m.AcquireSlot(context.Background())
		if err != nil {
			t.Errorf("Waiter got error: %v", err)
		}
	}()

	// Give goroutine time to start waiting
	time.Sleep(10 * time.Millisecond)

	// Release - should wake waiter
	m.ReleaseSlot(slot1)

	// Wait for goroutine
	wg.Wait()

	if slot2 == nil {
		t.Fatal("Waiter should have gotten slot after release")
	}
	if slot2.AccountName != "personal" {
		t.Errorf("Waiter got wrong account: %q", slot2.AccountName)
	}
}

func TestConcurrentAccess(t *testing.T) {
	accounts := []string{"personal", "work", "backup"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50},
		"work":     {FiveHourRemaining: 40, SevenDayRemaining: 40},
		"backup":   {FiveHourRemaining: 30, SevenDayRemaining: 30},
	}

	opts := DefaultOptions()
	opts.MaxPerAcct = 2

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	// Run many concurrent acquires and releases
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slot, err := m.AcquireSlot(context.Background())
			if err != nil {
				return // might timeout
			}
			// Hold briefly
			time.Sleep(5 * time.Millisecond)
			m.ReleaseSlot(slot)
		}()
	}

	wg.Wait()

	// All should be released
	status := m.Status()
	totalInFlight := 0
	for _, s := range status {
		totalInFlight += s.InFlight
	}
	if totalInFlight != 0 {
		t.Errorf("Expected 0 in-flight after all released, got %d", totalInFlight)
	}
}

func TestEventEmission(t *testing.T) {
	accounts := []string{"personal"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50},
	}

	var events []Event
	var mu sync.Mutex

	opts := DefaultOptions()
	opts.EventHandler = func(e Event) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	slot, _ := m.AcquireSlot(context.Background())
	m.ReleaseSlot(slot)

	mu.Lock()
	defer mu.Unlock()

	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events, got %d", len(events))
	}

	if events[0].Type != EventAcquired {
		t.Errorf("First event should be %q, got %q", EventAcquired, events[0].Type)
	}
	if events[1].Type != EventReleased {
		t.Errorf("Second event should be %q, got %q", EventReleased, events[1].Type)
	}
}

func TestStatus(t *testing.T) {
	accounts := []string{"personal", "work"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 50, SevenDayRemaining: 50, Email: "p@example.com"},
		"work":     {FiveHourRemaining: 40, SevenDayRemaining: 40, Email: "w@example.com"},
	}

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), DefaultOptions())

	status := m.Status()
	if len(status) != 2 {
		t.Fatalf("Expected 2 accounts in status, got %d", len(status))
	}

	// Check that status contains both accounts
	found := make(map[string]bool)
	for _, s := range status {
		found[s.Name] = true
	}
	if !found["personal"] || !found["work"] {
		t.Errorf("Status should contain both accounts, got %v", status)
	}
}

func TestRefresh(t *testing.T) {
	accounts := []string{"personal"}

	callCount := 0
	capacityFn := func(name string) (*account.CapacityInfo, error) {
		callCount++
		return &account.CapacityInfo{
			FiveHourRemaining: float64(50 - callCount*10), // Decreases each call
			SevenDayRemaining: 50,
		}, nil
	}

	m := NewWithCapacityFunc(accounts, capacityFn, DefaultOptions())

	// Initial status (triggers first capacity fetch)
	status1 := m.Status()
	firstRemaining := status1[0].Capacity.FiveHourRemaining

	// Force refresh
	if err := m.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Check capacity updated
	status2 := m.Status()
	secondRemaining := status2[0].Capacity.FiveHourRemaining

	if secondRemaining >= firstRemaining {
		t.Errorf("Expected capacity to decrease after refresh")
	}
}

func TestAcquireSlot_NoHealthyAccounts(t *testing.T) {
	accounts := []string{"personal", "work"}
	capacities := map[string]*account.CapacityInfo{
		"personal": {FiveHourRemaining: 5, SevenDayRemaining: 5}, // Below threshold
		"work":     {FiveHourRemaining: 3, SevenDayRemaining: 3}, // Below threshold
	}

	opts := DefaultOptions()
	opts.Threshold = 20 // Need >20% remaining
	opts.QueueTimeout = 50 * time.Millisecond

	m := NewWithCapacityFunc(accounts, mockCapacityFunc(capacities), opts)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := m.AcquireSlot(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded when no healthy accounts, got %v", err)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Threshold != 20 {
		t.Errorf("Default Threshold = %v, want 20", opts.Threshold)
	}
	if opts.MaxPerAcct != 3 {
		t.Errorf("Default MaxPerAcct = %d, want 3", opts.MaxPerAcct)
	}
	if opts.QueueTimeout != 5*time.Minute {
		t.Errorf("Default QueueTimeout = %v, want 5m", opts.QueueTimeout)
	}
	if opts.RefreshRate != time.Minute {
		t.Errorf("Default RefreshRate = %v, want 1m", opts.RefreshRate)
	}
}

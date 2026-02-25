package account

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewCapacityCache(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("NewCapacityCache() returned nil")
	}
}

func TestCapacityCache_GetAndSet(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)

	// Cache miss returns nil
	got := cache.Get("work")
	if got != nil {
		t.Errorf("Get(work) on empty cache = %v, want nil", got)
	}

	// Set and get
	info := &CapacityInfo{
		FiveHourRemaining: 80,
		SevenDayRemaining: 60,
	}
	cache.Set("work", info)

	got = cache.Get("work")
	if got == nil {
		t.Fatal("Get(work) after Set returned nil")
	}
	if got.FiveHourRemaining != 80 {
		t.Errorf("FiveHourRemaining = %v, want 80", got.FiveHourRemaining)
	}
	if got.SevenDayRemaining != 60 {
		t.Errorf("SevenDayRemaining = %v, want 60", got.SevenDayRemaining)
	}
}

func TestCapacityCache_TTLExpiry(t *testing.T) {
	// Use a very short TTL for testing
	cache := NewCapacityCache(50 * time.Millisecond)

	info := &CapacityInfo{
		FiveHourRemaining: 80,
		SevenDayRemaining: 60,
	}
	cache.Set("work", info)

	// Should be available immediately
	got := cache.Get("work")
	if got == nil {
		t.Fatal("Get(work) immediately after Set returned nil")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should be expired now
	got = cache.Get("work")
	if got != nil {
		t.Errorf("Get(work) after TTL expiry = %v, want nil", got)
	}
}

func TestCapacityCache_Invalidate(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)

	info := &CapacityInfo{
		FiveHourRemaining: 80,
		SevenDayRemaining: 60,
	}
	cache.Set("work", info)

	// Verify it's cached
	if cache.Get("work") == nil {
		t.Fatal("Get(work) should return cached value")
	}

	// Invalidate
	cache.Invalidate("work")

	// Should be gone
	if cache.Get("work") != nil {
		t.Error("Get(work) after Invalidate should return nil")
	}
}

func TestCapacityCache_InvalidateNonexistent(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)

	// Should not panic
	cache.Invalidate("nonexistent")
}

func TestCapacityCache_MultipleAccounts(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)

	cache.Set("work", &CapacityInfo{FiveHourRemaining: 80, SevenDayRemaining: 60})
	cache.Set("personal", &CapacityInfo{FiveHourRemaining: 95, SevenDayRemaining: 88})

	work := cache.Get("work")
	personal := cache.Get("personal")

	if work == nil || personal == nil {
		t.Fatal("Both accounts should be cached")
	}

	if work.FiveHourRemaining != 80 {
		t.Errorf("work.FiveHourRemaining = %v, want 80", work.FiveHourRemaining)
	}
	if personal.FiveHourRemaining != 95 {
		t.Errorf("personal.FiveHourRemaining = %v, want 95", personal.FiveHourRemaining)
	}
}

func TestCapacityCache_Overwrite(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)

	cache.Set("work", &CapacityInfo{FiveHourRemaining: 80})
	cache.Set("work", &CapacityInfo{FiveHourRemaining: 50})

	got := cache.Get("work")
	if got == nil {
		t.Fatal("Get(work) returned nil after overwrite")
	}
	if got.FiveHourRemaining != 50 {
		t.Errorf("FiveHourRemaining after overwrite = %v, want 50", got.FiveHourRemaining)
	}
}

func TestCapacityCache_ConcurrentAccess(t *testing.T) {
	cache := NewCapacityCache(5 * time.Minute)
	var wg sync.WaitGroup

	// Concurrent writes and reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("account-%d", i%5)
			cache.Set(name, &CapacityInfo{FiveHourRemaining: float64(i)})
			cache.Get(name)
			if i%10 == 0 {
				cache.Invalidate(name)
			}
		}(i)
	}

	wg.Wait()
	// No panics = success
}

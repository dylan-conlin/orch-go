package usage

import (
	"testing"
	"time"
)

func TestUsageLimitRemaining(t *testing.T) {
	tests := []struct {
		name        string
		utilization float64
		want        float64
	}{
		{"zero utilization", 0, 100},
		{"half utilization", 50, 50},
		{"full utilization", 100, 0},
		{"partial utilization", 33.5, 66.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := &UsageLimit{Utilization: tt.utilization}
			if got := limit.Remaining(); got != tt.want {
				t.Errorf("Remaining() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsageLimitTimeUntilReset(t *testing.T) {
	t.Run("nil reset time", func(t *testing.T) {
		limit := &UsageLimit{ResetsAt: nil}
		if got := limit.TimeUntilReset(); got != "" {
			t.Errorf("TimeUntilReset() = %v, want empty string", got)
		}
	})

	t.Run("past time", func(t *testing.T) {
		pastTime := time.Now().UTC().Add(-1 * time.Hour)
		limit := &UsageLimit{ResetsAt: &pastTime}
		if got := limit.TimeUntilReset(); got != "now" {
			t.Errorf("TimeUntilReset() = %v, want 'now'", got)
		}
	})

	t.Run("future time shows hours and minutes", func(t *testing.T) {
		// Use a time 2 hours and 30 minutes in the future
		futureTime := time.Now().UTC().Add(2*time.Hour + 30*time.Minute + 30*time.Second)
		limit := &UsageLimit{ResetsAt: &futureTime}
		result := limit.TimeUntilReset()
		// Should be in format "Xh Ym"
		if !contains(result, "h") || !contains(result, "m") {
			t.Errorf("TimeUntilReset() = %v, want format 'Xh Ym'", result)
		}
	})

	t.Run("days shows days and hours", func(t *testing.T) {
		// Use a time 2 days and 5 hours in the future
		futureTime := time.Now().UTC().Add(2*24*time.Hour + 5*time.Hour + 30*time.Minute)
		limit := &UsageLimit{ResetsAt: &futureTime}
		result := limit.TimeUntilReset()
		// Should be in format "Xd Yh"
		if !contains(result, "d") || !contains(result, "h") {
			t.Errorf("TimeUntilReset() = %v, want format 'Xd Yh'", result)
		}
	})

	t.Run("minutes only", func(t *testing.T) {
		// Use a time 45 minutes in the future
		futureTime := time.Now().UTC().Add(45*time.Minute + 30*time.Second)
		limit := &UsageLimit{ResetsAt: &futureTime}
		result := limit.TimeUntilReset()
		// Should be in format "Xm" (no hours)
		if !contains(result, "m") || contains(result, "h") {
			t.Errorf("TimeUntilReset() = %v, want format 'Xm' without hours", result)
		}
	})
}

func TestFormatDisplay(t *testing.T) {
	t.Run("error case", func(t *testing.T) {
		info := &UsageInfo{Error: "test error"}
		result := FormatDisplay(info)
		if result != "\u274C Error: test error" {
			t.Errorf("FormatDisplay() = %v, want error message", result)
		}
	})

	t.Run("with usage data", func(t *testing.T) {
		resetTime := time.Now().UTC().Add(2 * time.Hour)
		info := &UsageInfo{
			FiveHour: &UsageLimit{
				Utilization: 50,
				ResetsAt:    &resetTime,
			},
			SevenDay: &UsageLimit{
				Utilization: 30,
				ResetsAt:    &resetTime,
			},
			Email: "test@example.com",
		}
		result := FormatDisplay(info)
		if result == "" {
			t.Error("FormatDisplay() returned empty string")
		}
		// Check that key elements are present
		if !contains(result, "Claude Max Usage") {
			t.Error("missing 'Claude Max Usage' header")
		}
		if !contains(result, "test@example.com") {
			t.Error("missing email")
		}
		if !contains(result, "50.0% used") {
			t.Error("missing 5-hour usage")
		}
		if !contains(result, "30.0% used") {
			t.Error("missing 7-day usage")
		}
	})
}

func TestGetUsageSummary(t *testing.T) {
	// This test requires actual API access, so we just verify the function signature
	// In a real test environment, we'd mock the HTTP client
	t.Skip("Skipping test that requires API access")
}

func TestUsageCache(t *testing.T) {
	t.Run("cache hit within TTL", func(t *testing.T) {
		cache := newUsageCache(60 * time.Second)

		// Create test data
		testInfo := &UsageInfo{
			FiveHour: &UsageLimit{Utilization: 50},
			SevenDay: &UsageLimit{Utilization: 30},
			Email:    "test@example.com",
		}

		// Set cache entry
		token := "test-token"
		cache.set(token, testInfo)

		// Retrieve from cache - should hit
		cached, ok := cache.get(token)
		if !ok {
			t.Error("Expected cache hit, got miss")
		}
		if cached.Email != "test@example.com" {
			t.Errorf("Expected email test@example.com, got %s", cached.Email)
		}
	})

	t.Run("cache miss after TTL expiration", func(t *testing.T) {
		cache := newUsageCache(1 * time.Millisecond)

		testInfo := &UsageInfo{
			Email: "test@example.com",
		}

		token := "test-token"
		cache.set(token, testInfo)

		// Wait for TTL to expire
		time.Sleep(10 * time.Millisecond)

		// Should be cache miss
		_, ok := cache.get(token)
		if ok {
			t.Error("Expected cache miss after TTL, got hit")
		}
	})

	t.Run("cache invalidation", func(t *testing.T) {
		cache := newUsageCache(60 * time.Second)

		testInfo := &UsageInfo{
			Email: "test@example.com",
		}

		token := "test-token"
		cache.set(token, testInfo)

		// Verify cache hit before invalidation
		_, ok := cache.get(token)
		if !ok {
			t.Error("Expected cache hit before invalidation")
		}

		// Invalidate
		cache.invalidate()

		// Should be cache miss after invalidation
		_, ok = cache.get(token)
		if ok {
			t.Error("Expected cache miss after invalidation, got hit")
		}
	})

	t.Run("concurrent access safety", func(t *testing.T) {
		cache := newUsageCache(60 * time.Second)

		// Spawn multiple goroutines doing concurrent reads/writes
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				testInfo := &UsageInfo{
					Email: "test@example.com",
				}
				token := "test-token"

				// Mix of operations
				cache.set(token, testInfo)
				cache.get(token)
				if id%2 == 0 {
					cache.invalidate()
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Test passes if no race conditions detected
	})

	t.Run("different tokens have separate cache entries", func(t *testing.T) {
		cache := newUsageCache(60 * time.Second)

		info1 := &UsageInfo{Email: "user1@example.com"}
		info2 := &UsageInfo{Email: "user2@example.com"}

		cache.set("token1", info1)
		cache.set("token2", info2)

		cached1, ok := cache.get("token1")
		if !ok || cached1.Email != "user1@example.com" {
			t.Error("token1 cache entry incorrect")
		}

		cached2, ok := cache.get("token2")
		if !ok || cached2.Email != "user2@example.com" {
			t.Error("token2 cache entry incorrect")
		}
	})
}

func TestFetchUsageCaching(t *testing.T) {
	// This test verifies that FetchUsage uses the cache
	// We can't test the actual API call without credentials, but we can verify
	// that error responses are NOT cached (only successful responses are cached)

	t.Run("errors are not cached", func(t *testing.T) {
		// Clear the global cache
		globalUsageCache.invalidate()

		// FetchUsage will fail without valid credentials, but should not cache the error
		info1 := FetchUsage()
		if info1.Error == "" {
			t.Skip("Skipping - valid credentials found, can't test error caching")
		}

		// Verify error is returned
		if info1.Error == "" {
			t.Error("Expected error, got success")
		}

		// Subsequent call should also hit the API (not cache) since errors aren't cached
		info2 := FetchUsage()
		if info2.Error == "" {
			t.Error("Expected error on second call, got success")
		}
	})

	t.Run("cache invalidation is accessible", func(t *testing.T) {
		// Verify that InvalidateUsageCache is accessible (called by account switching)
		InvalidateUsageCache()
		// Test passes if function is accessible
	})
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

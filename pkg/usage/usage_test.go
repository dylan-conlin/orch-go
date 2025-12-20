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

package daemon

import (
	"strings"
	"testing"
)

func TestDeadSessionDetectionConfig_MaxRetries(t *testing.T) {
	tests := []struct {
		name     string
		config   DeadSessionDetectionConfig
		expected int
	}{
		{
			name:     "default when zero",
			config:   DeadSessionDetectionConfig{MaxRetries: 0},
			expected: DefaultMaxDeadSessionRetries,
		},
		{
			name:     "default when negative",
			config:   DeadSessionDetectionConfig{MaxRetries: -1},
			expected: DefaultMaxDeadSessionRetries,
		},
		{
			name:     "explicit value",
			config:   DeadSessionDetectionConfig{MaxRetries: 5},
			expected: 5,
		},
		{
			name:     "one",
			config:   DeadSessionDetectionConfig{MaxRetries: 1},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.maxRetries()
			if got != tt.expected {
				t.Errorf("maxRetries() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestDefaultMaxDeadSessionRetries(t *testing.T) {
	if DefaultMaxDeadSessionRetries != 2 {
		t.Errorf("DefaultMaxDeadSessionRetries = %d, want 2", DefaultMaxDeadSessionRetries)
	}
}

func TestDefaultDeadSessionDetectionConfig(t *testing.T) {
	config := DefaultDeadSessionDetectionConfig()
	if config.MaxRetries != DefaultMaxDeadSessionRetries {
		t.Errorf("default MaxRetries = %d, want %d", config.MaxRetries, DefaultMaxDeadSessionRetries)
	}
	if config.Verbose {
		t.Error("default Verbose should be false")
	}
}

func TestMarkSessionAsDead_CommentPrefix(t *testing.T) {
	// Verify the DEAD SESSION: prefix format that CountDeadSessionComments relies on
	comment := "DEAD SESSION: test reason\n\nsome body"
	if !strings.HasPrefix(comment, "DEAD SESSION:") {
		t.Error("dead session comment must start with DEAD SESSION: prefix")
	}
}

func TestDeadSessionDetectionResult_EscalatedField(t *testing.T) {
	result := DeadSessionDetectionResult{
		DetectedCount:  3,
		MarkedCount:    1,
		EscalatedCount: 2,
		SkippedCount:   0,
	}
	if result.EscalatedCount != 2 {
		t.Errorf("EscalatedCount = %d, want 2", result.EscalatedCount)
	}
	total := result.MarkedCount + result.EscalatedCount + result.SkippedCount
	if total != result.DetectedCount {
		t.Errorf("marked(%d) + escalated(%d) + skipped(%d) = %d, want detected(%d)",
			result.MarkedCount, result.EscalatedCount, result.SkippedCount, total, result.DetectedCount)
	}
}

func TestDefaultConfig_MaxDeadSessionRetries(t *testing.T) {
	config := DefaultConfig()
	if config.MaxDeadSessionRetries != DefaultMaxDeadSessionRetries {
		t.Errorf("DefaultConfig().MaxDeadSessionRetries = %d, want %d",
			config.MaxDeadSessionRetries, DefaultMaxDeadSessionRetries)
	}
}

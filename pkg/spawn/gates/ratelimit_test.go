package gates

import (
	"os"
	"strconv"
	"testing"
)

func TestDefaultUsageThresholds(t *testing.T) {
	thresholds := DefaultUsageThresholds()

	if thresholds.WarnThreshold != 80 {
		t.Errorf("WarnThreshold = %v, want 80", thresholds.WarnThreshold)
	}
	if thresholds.BlockThreshold != 95 {
		t.Errorf("BlockThreshold = %v, want 95", thresholds.BlockThreshold)
	}
}

func TestUsageThresholdsFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		warnEnv   string
		blockEnv  string
		wantWarn  float64
		wantBlock float64
	}{
		{
			name:      "no env vars - use defaults",
			warnEnv:   "",
			blockEnv:  "",
			wantWarn:  80,
			wantBlock: 95,
		},
		{
			name:      "custom warn threshold",
			warnEnv:   "70",
			blockEnv:  "",
			wantWarn:  70,
			wantBlock: 95,
		},
		{
			name:      "custom block threshold",
			warnEnv:   "",
			blockEnv:  "90",
			wantWarn:  80,
			wantBlock: 90,
		},
		{
			name:      "both custom",
			warnEnv:   "75",
			blockEnv:  "92",
			wantWarn:  75,
			wantBlock: 92,
		},
		{
			name:      "invalid env - use defaults",
			warnEnv:   "not-a-number",
			blockEnv:  "invalid",
			wantWarn:  80,
			wantBlock: 95,
		},
		{
			name:      "out of range - use defaults",
			warnEnv:   "150",
			blockEnv:  "-10",
			wantWarn:  80,
			wantBlock: 95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origWarn := os.Getenv("ORCH_USAGE_WARN_THRESHOLD")
			origBlock := os.Getenv("ORCH_USAGE_BLOCK_THRESHOLD")
			defer func() {
				if origWarn == "" {
					os.Unsetenv("ORCH_USAGE_WARN_THRESHOLD")
				} else {
					os.Setenv("ORCH_USAGE_WARN_THRESHOLD", origWarn)
				}
				if origBlock == "" {
					os.Unsetenv("ORCH_USAGE_BLOCK_THRESHOLD")
				} else {
					os.Setenv("ORCH_USAGE_BLOCK_THRESHOLD", origBlock)
				}
			}()

			if tt.warnEnv != "" {
				os.Setenv("ORCH_USAGE_WARN_THRESHOLD", tt.warnEnv)
			} else {
				os.Unsetenv("ORCH_USAGE_WARN_THRESHOLD")
			}
			if tt.blockEnv != "" {
				os.Setenv("ORCH_USAGE_BLOCK_THRESHOLD", tt.blockEnv)
			} else {
				os.Unsetenv("ORCH_USAGE_BLOCK_THRESHOLD")
			}

			// Replicate the threshold parsing logic from CheckRateLimit
			thresholds := DefaultUsageThresholds()
			if envVal := os.Getenv("ORCH_USAGE_WARN_THRESHOLD"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
					thresholds.WarnThreshold = val
				}
			}
			if envVal := os.Getenv("ORCH_USAGE_BLOCK_THRESHOLD"); envVal != "" {
				if val, err := strconv.ParseFloat(envVal, 64); err == nil && val > 0 && val <= 100 {
					thresholds.BlockThreshold = val
				}
			}

			if thresholds.WarnThreshold != tt.wantWarn {
				t.Errorf("WarnThreshold = %v, want %v", thresholds.WarnThreshold, tt.wantWarn)
			}
			if thresholds.BlockThreshold != tt.wantBlock {
				t.Errorf("BlockThreshold = %v, want %v", thresholds.BlockThreshold, tt.wantBlock)
			}
		})
	}
}

func TestUsageCheckResult_Fields(t *testing.T) {
	// Verify struct can represent all gate states
	result := &UsageCheckResult{
		Warning:      "test warning",
		Blocked:      true,
		BlockReason:  "over limit",
		Switched:     true,
		SwitchReason: "auto-switched",
	}

	if result.Warning != "test warning" {
		t.Error("Warning field not set")
	}
	if !result.Blocked {
		t.Error("Blocked field not set")
	}
	if !result.Switched {
		t.Error("Switched field not set")
	}
}

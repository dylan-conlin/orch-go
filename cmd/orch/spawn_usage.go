// Package main provides usage monitoring and rate limit management for spawn commands.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// determineSpawnTier determines the spawn tier based on flags, skill defaults, and global config.
// Priority: --light flag > --full flag > skill default > userconfig.default_tier > TierFull (conservative)
func determineSpawnTier(skillName string, lightFlag, fullFlag bool) string {
	globalDefault := ""
	cfg, err := userconfig.Load()
	if err == nil && cfg.GetDefaultTier() != "" {
		globalDefault = cfg.GetDefaultTier()
	}

	return determineSpawnTierWithGlobalDefault(skillName, lightFlag, fullFlag, globalDefault)
}

func determineSpawnTierWithGlobalDefault(skillName string, lightFlag, fullFlag bool, globalDefault string) string {
	// Explicit flags take precedence
	if lightFlag {
		return spawn.TierLight
	}
	if fullFlag {
		return spawn.TierFull
	}

	// Skill-specific default takes precedence over global fallback
	if tier, ok := spawn.SkillTierDefaults[skillName]; ok {
		return tier
	}

	// Global default applies only when skill has no declared default
	if globalDefault == spawn.TierLight || globalDefault == spawn.TierFull {
		return globalDefault
	}

	// Conservative fallback for unknown skills
	return spawn.TierFull
}

// UsageThresholds defines the thresholds for proactive rate limit monitoring.
// These are checked BEFORE spawn to warn or block based on current usage.
type UsageThresholds struct {
	// WarnThreshold is the usage % above which to show a warning (default 80).
	WarnThreshold float64
	// BlockThreshold is the usage % above which to block spawn unless auto-switch succeeds (default 95).
	BlockThreshold float64
}

// DefaultUsageThresholds returns the default proactive monitoring thresholds.
func DefaultUsageThresholds() UsageThresholds {
	return UsageThresholds{
		WarnThreshold:  80,
		BlockThreshold: 95,
	}
}

// UsageCheckResult contains the result of a pre-spawn usage check.
type UsageCheckResult struct {
	// Warning is set if usage exceeds warning threshold.
	Warning string
	// Blocked is true if spawn should be blocked (usage critical and switch failed).
	Blocked bool
	// BlockReason explains why spawn was blocked.
	BlockReason string
	// Switched is true if account was auto-switched.
	Switched bool
	// SwitchedToAccount is the saved account name selected by auto-switch.
	SwitchedToAccount string
	// SwitchReason explains the switch.
	SwitchReason string
	// CapacityInfo is the current account capacity (for telemetry).
	CapacityInfo *account.CapacityInfo
}

// checkUsageBeforeSpawn performs proactive rate limit monitoring.
// It checks usage BEFORE spawn and:
// 1. Warns at 80% usage (5h or weekly)
// 2. Attempts auto-switch at 95% usage
// 3. Blocks spawn at 95% if auto-switch fails
//
// Returns UsageCheckResult for telemetry and a blocking error if spawn should not proceed.
func checkUsageBeforeSpawn() (*UsageCheckResult, error) {
	result := &UsageCheckResult{}

	// Get thresholds from environment or use defaults
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

	// Get current account capacity
	capacity, err := account.GetCurrentCapacity()
	if err != nil {
		// Log warning but don't block - can't check capacity
		fmt.Fprintf(os.Stderr, "Warning: could not check usage: %v\n", err)
		return result, nil
	}

	if capacity.Error != "" {
		fmt.Fprintf(os.Stderr, "Warning: usage check failed: %s\n", capacity.Error)
		return result, nil
	}

	result.CapacityInfo = capacity

	// Determine effective usage (use the tighter constraint)
	fiveHourUsed := capacity.FiveHourUsed
	weeklyUsed := capacity.SevenDayUsed
	effectiveUsage := fiveHourUsed
	usageType := "5h session"
	if weeklyUsed > fiveHourUsed {
		effectiveUsage = weeklyUsed
		usageType = "weekly"
	}

	// Check for blocking threshold (95%)
	if effectiveUsage >= thresholds.BlockThreshold {
		// Try auto-switch first
		switchResult, switchErr := tryAutoSwitchForSpawn()
		if switchErr == nil && switchResult.Switched {
			result.Switched = true
			result.SwitchedToAccount = switchResult.ToAccount
			result.SwitchReason = switchResult.Reason
			// Update capacity after switch
			newCapacity, _ := account.GetCurrentCapacity()
			if newCapacity != nil && newCapacity.Error == "" {
				result.CapacityInfo = newCapacity
			}
			fmt.Printf("🔄 Auto-switched account: %s\n", switchResult.Reason)
			return result, nil
		}

		// Switch failed or no alternate account - block spawn
		result.Blocked = true
		result.BlockReason = fmt.Sprintf("usage critical: %s at %.1f%% (threshold: %.0f%%)", usageType, effectiveUsage, thresholds.BlockThreshold)

		// Log the blocked spawn for pattern analysis
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.blocked.rate_limit",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"five_hour_used": fiveHourUsed,
				"weekly_used":    weeklyUsed,
				"threshold":      thresholds.BlockThreshold,
				"switch_failed":  switchErr != nil || !switchResult.Switched,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log blocked spawn: %v\n", err)
		}

		return result, fmt.Errorf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│  🛑 SPAWN BLOCKED: Rate Limit Critical                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  Current usage: %s at %.1f%%                                                │
│  Block threshold: %.0f%%                                                     │
│                                                                             │
│  Auto-switch failed: No alternate account with sufficient headroom.         │
│                                                                             │
│  Options:                                                                   │
│    • Wait for limit to reset (see 'orch usage' for reset time)              │
│    • Add another account: orch account add <name>                           │
│    • Override: ORCH_USAGE_BLOCK_THRESHOLD=100 orch spawn ...                │
└─────────────────────────────────────────────────────────────────────────────┘
`, usageType, effectiveUsage, thresholds.BlockThreshold)
	}

	// Check for warning threshold (80%)
	if effectiveUsage >= thresholds.WarnThreshold {
		result.Warning = fmt.Sprintf("⚠️  Usage warning: %s at %.1f%% (warn at %.0f%%, block at %.0f%%)", usageType, effectiveUsage, thresholds.WarnThreshold, thresholds.BlockThreshold)
		fmt.Fprintf(os.Stderr, "%s\n", result.Warning)

		// Log the warning for pattern analysis
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type:      "spawn.warning.rate_limit",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"five_hour_used": fiveHourUsed,
				"weekly_used":    weeklyUsed,
				"threshold":      thresholds.WarnThreshold,
			},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log usage warning: %v\n", err)
		}
	}

	return result, nil
}

// tryAutoSwitchForSpawn attempts to auto-switch to a better account for spawning.
// This is called when usage is at blocking threshold.
func tryAutoSwitchForSpawn() (*account.AutoSwitchResult, error) {
	// Use lower thresholds to trigger switch more aggressively
	thresholds := account.AutoSwitchThresholds{
		FiveHourThreshold: 90, // Lower than default 80 since we're already at 95
		WeeklyThreshold:   90, // Lower than default 90
		MinHeadroomDelta:  5,  // Lower delta requirement for emergency switch
	}

	return account.AutoSwitchIfNeeded(thresholds)
}

// Package verify provides verification helpers for agent completion.
package verify

import (
	"os/exec"
	"strings"
)

// Context limit thresholds (in tokens)
const (
	// ContextWarningThreshold is 75% of 1M context limit
	ContextWarningThreshold = 750000

	// ContextCriticalThreshold is 90% of 1M context limit
	ContextCriticalThreshold = 900000
)

// ContextRiskLevel indicates the severity of context exhaustion risk.
type ContextRiskLevel string

const (
	// RiskNone indicates no context exhaustion risk
	RiskNone ContextRiskLevel = ""

	// RiskWarning indicates high token usage (>75% of limit)
	RiskWarning ContextRiskLevel = "warning"

	// RiskCritical indicates very high token usage (>90% of limit)
	RiskCritical ContextRiskLevel = "critical"
)

// ContextExhaustionRisk represents the risk of an agent exhausting its context.
type ContextExhaustionRisk struct {
	// Level indicates the severity of the risk
	Level ContextRiskLevel `json:"level,omitempty"`

	// TokenUsage is the total tokens used by the agent
	TokenUsage int `json:"token_usage,omitempty"`

	// TokenPercent is the percentage of context limit used (0-100)
	TokenPercent float64 `json:"token_percent,omitempty"`

	// HasUncommittedWork indicates if the agent has uncommitted changes
	HasUncommittedWork bool `json:"has_uncommitted_work,omitempty"`

	// UncommittedCount is the number of uncommitted files
	UncommittedCount int `json:"uncommitted_count,omitempty"`

	// Reason provides a human-readable explanation of the risk
	Reason string `json:"reason,omitempty"`
}

// AssessContextRisk evaluates the context exhaustion risk for an agent.
// It combines token usage with uncommitted work detection to determine risk level.
//
// Parameters:
//   - totalTokens: The total tokens used by the agent (input + output)
//   - projectDir: The project directory to check for uncommitted changes
//   - isProcessing: Whether the agent is currently processing (active)
//
// Returns a ContextExhaustionRisk with the assessed risk level and details.
func AssessContextRisk(totalTokens int, projectDir string, isProcessing bool) ContextExhaustionRisk {
	risk := ContextExhaustionRisk{
		TokenUsage:   totalTokens,
		TokenPercent: float64(totalTokens) / 1000000 * 100, // Assume 1M context limit
	}

	// Check for uncommitted work
	if projectDir != "" {
		hasUncommitted, count := HasUncommittedWork(projectDir)
		risk.HasUncommittedWork = hasUncommitted
		risk.UncommittedCount = count
	}

	// Determine risk level based on token usage
	if totalTokens >= ContextCriticalThreshold {
		risk.Level = RiskCritical
		if risk.HasUncommittedWork {
			risk.Reason = "Critical: >90% context used with uncommitted work"
		} else {
			risk.Reason = "Critical: >90% context used"
		}
	} else if totalTokens >= ContextWarningThreshold {
		// Only warn if there's uncommitted work (otherwise just high usage is normal)
		if risk.HasUncommittedWork {
			risk.Level = RiskWarning
			risk.Reason = "High context usage with uncommitted work"
		}
	} else {
		// Low token usage - only flag if there's a lot of uncommitted work
		// and agent is idle (might have crashed/exited)
		if risk.HasUncommittedWork && risk.UncommittedCount >= 5 && !isProcessing {
			risk.Level = RiskWarning
			risk.Reason = "Agent idle with significant uncommitted work"
		}
	}

	return risk
}

// HasUncommittedWork checks if a git repository has uncommitted changes.
// Returns (hasChanges, changeCount).
//
// This function runs `git status --porcelain` in the given directory
// and parses the output to determine if there are uncommitted changes.
func HasUncommittedWork(projectDir string) (bool, int) {
	if projectDir == "" {
		return false, 0
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Git command failed - assume no uncommitted work
		// (could be not a git repo, or other issue)
		return false, 0
	}

	changes := strings.TrimSpace(string(output))
	if changes == "" {
		return false, 0
	}

	// Count the number of changed files
	lines := strings.Split(changes, "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			count++
		}
	}

	return true, count
}

// FormatRiskStatus returns a concise string representation of the risk.
// Used for display in status tables.
func (r ContextExhaustionRisk) FormatRiskStatus() string {
	switch r.Level {
	case RiskCritical:
		if r.HasUncommittedWork {
			return "CRITICAL"
		}
		return "HIGH-CTX"
	case RiskWarning:
		if r.HasUncommittedWork {
			return "AT-RISK"
		}
		return "HIGH-TOK"
	default:
		if r.HasUncommittedWork {
			return "UNCOMMIT"
		}
		return ""
	}
}

// FormatRiskEmoji returns an emoji indicator for the risk level.
func (r ContextExhaustionRisk) FormatRiskEmoji() string {
	switch r.Level {
	case RiskCritical:
		return "🚨"
	case RiskWarning:
		return "⚠️"
	default:
		return ""
	}
}

// IsAtRisk returns true if there's any level of risk.
func (r ContextExhaustionRisk) IsAtRisk() bool {
	return r.Level != RiskNone
}

// ShouldAlert returns true if this risk warrants immediate attention.
// Used by monitor to determine when to send notifications.
func (r ContextExhaustionRisk) ShouldAlert() bool {
	// Alert on critical risk, or warning with uncommitted work
	return r.Level == RiskCritical || (r.Level == RiskWarning && r.HasUncommittedWork)
}

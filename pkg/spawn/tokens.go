// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"strings"
)

// Token estimation constants.
// Note: CharsPerToken is defined in kbcontext.go (value: 4)
const (
	// DefaultTokenWarningThreshold is the default threshold for warning about context size.
	// Claude's context window is 200k tokens. We warn at 100k to leave room for:
	// - Agent's working memory during the session
	// - Tool results and file contents that get added
	// - Back-and-forth conversation
	DefaultTokenWarningThreshold = 100000

	// DefaultTokenErrorThreshold is the default threshold for blocking spawn.
	// At 150k estimated tokens, the spawn context alone is too large.
	DefaultTokenErrorThreshold = 150000
)

// TokenEstimate represents an estimate of token usage for spawn context.
type TokenEstimate struct {
	// CharCount is the total character count of the content.
	CharCount int

	// EstimatedTokens is the estimated token count (chars / CharsPerToken).
	EstimatedTokens int

	// WarningThreshold is the threshold at which warnings should be shown.
	WarningThreshold int

	// ErrorThreshold is the threshold at which spawn should be blocked.
	ErrorThreshold int

	// Components breaks down token estimates by component.
	Components map[string]int
}

// ExceedsWarning returns true if estimated tokens exceed the warning threshold.
func (e *TokenEstimate) ExceedsWarning() bool {
	return e.EstimatedTokens >= e.WarningThreshold
}

// ExceedsError returns true if estimated tokens exceed the error threshold.
func (e *TokenEstimate) ExceedsError() bool {
	return e.EstimatedTokens >= e.ErrorThreshold
}

// UtilizationPercent returns the percentage of warning threshold used.
func (e *TokenEstimate) UtilizationPercent() float64 {
	if e.WarningThreshold == 0 {
		return 0
	}
	return float64(e.EstimatedTokens) / float64(e.WarningThreshold) * 100
}

// FormatSummary returns a human-readable summary of the token estimate.
func (e *TokenEstimate) FormatSummary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Token estimate: ~%dk tokens", e.EstimatedTokens/1000))
	sb.WriteString(fmt.Sprintf(" (%.0f%% of warning threshold)\n", e.UtilizationPercent()))

	// Show component breakdown for large contexts
	if e.EstimatedTokens > 10000 && len(e.Components) > 0 {
		sb.WriteString("  Components:\n")
		for name, tokens := range e.Components {
			if tokens > 1000 {
				sb.WriteString(fmt.Sprintf("    - %s: ~%dk\n", name, tokens/1000))
			}
		}
	}

	return sb.String()
}

// EstimateTokens estimates the token count from a character count.
// Uses CharsPerToken constant defined in kbcontext.go.
func EstimateTokens(charCount int) int {
	return charCount / CharsPerToken
}

// EstimateContextTokens estimates the token count for a spawn context configuration.
// It analyzes the individual components to provide a breakdown.
func EstimateContextTokens(cfg *Config) *TokenEstimate {
	components := make(map[string]int)

	// Estimate base template size (the template structure without dynamic content)
	baseTemplateSize := 3000 // ~750 tokens for the base template
	components["template"] = EstimateTokens(baseTemplateSize)

	// Task description
	taskSize := len(cfg.Task)
	components["task"] = EstimateTokens(taskSize)

	// Skill content (usually the largest component)
	skillSize := len(cfg.SkillContent)
	if skillSize > 0 {
		components["skill"] = EstimateTokens(skillSize)
	}

	// KB context
	kbContextSize := len(cfg.KBContext)
	if kbContextSize > 0 {
		components["kb_context"] = EstimateTokens(kbContextSize)
	}

	// Server context
	serverContextSize := len(cfg.ServerContext)
	if serverContextSize > 0 {
		components["server_context"] = EstimateTokens(serverContextSize)
	}

	// Calculate total
	totalChars := baseTemplateSize + taskSize + skillSize + kbContextSize + serverContextSize
	totalTokens := 0
	for _, tokens := range components {
		totalTokens += tokens
	}

	return &TokenEstimate{
		CharCount:        totalChars,
		EstimatedTokens:  totalTokens,
		WarningThreshold: DefaultTokenWarningThreshold,
		ErrorThreshold:   DefaultTokenErrorThreshold,
		Components:       components,
	}
}

// EstimateContentTokens estimates tokens from generated content string.
// This is useful for post-generation validation.
func EstimateContentTokens(content string) *TokenEstimate {
	charCount := len(content)
	tokens := EstimateTokens(charCount)

	return &TokenEstimate{
		CharCount:        charCount,
		EstimatedTokens:  tokens,
		WarningThreshold: DefaultTokenWarningThreshold,
		ErrorThreshold:   DefaultTokenErrorThreshold,
		Components:       map[string]int{"content": tokens},
	}
}

// ValidateContextSize checks if the spawn context is within acceptable limits.
// Returns nil if OK, or an error describing the issue.
func ValidateContextSize(cfg *Config) error {
	estimate := EstimateContextTokens(cfg)

	if estimate.ExceedsError() {
		return &ContextTooLargeError{
			Estimate:         estimate,
			LargestComponent: findLargestComponent(estimate.Components),
		}
	}

	return nil
}

// ContextTooLargeError is returned when spawn context exceeds size limits.
type ContextTooLargeError struct {
	Estimate         *TokenEstimate
	LargestComponent string
}

func (e *ContextTooLargeError) Error() string {
	return fmt.Sprintf(
		"spawn context too large: ~%dk tokens (limit: %dk). Largest component: %s (~%dk tokens)",
		e.Estimate.EstimatedTokens/1000,
		e.Estimate.ErrorThreshold/1000,
		e.LargestComponent,
		e.Estimate.Components[e.LargestComponent]/1000,
	)
}

// findLargestComponent returns the name of the largest component.
func findLargestComponent(components map[string]int) string {
	var largest string
	var maxTokens int

	for name, tokens := range components {
		if tokens > maxTokens {
			maxTokens = tokens
			largest = name
		}
	}

	return largest
}

// ShouldWarnAboutSize returns true and a warning message if context size is concerning.
// Returns false and empty string if context size is acceptable.
func ShouldWarnAboutSize(cfg *Config) (bool, string) {
	estimate := EstimateContextTokens(cfg)

	if !estimate.ExceedsWarning() {
		return false, ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚠️  Large spawn context: ~%dk tokens (%.0f%% of safe limit)\n",
		estimate.EstimatedTokens/1000,
		estimate.UtilizationPercent()))

	// Provide specific guidance based on largest component
	largest := findLargestComponent(estimate.Components)
	sb.WriteString(fmt.Sprintf("   → Largest component: %s (~%dk tokens)\n",
		largest, estimate.Components[largest]/1000))
	switch largest {
	case "skill":
		sb.WriteString("   → Consider using a more focused skill or --skip-artifact-check\n")
	case "kb_context":
		sb.WriteString("   → Consider --skip-artifact-check to reduce KB context\n")
	}

	sb.WriteString("   → Agent may run out of context during work\n")

	return true, sb.String()
}

// Package cost provides cost calculation for AI model usage.
package cost

import (
	"fmt"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
)

// Sonnet45Pricing represents Sonnet 4.5 pricing as of January 2026.
// Rates per 1,000,000 tokens.
type Sonnet45Pricing struct {
	InputPerMillion      float64 // $3.00 per 1M input tokens
	OutputPerMillion     float64 // $15.00 per 1M output tokens
	ReasoningPerMillion  float64 // $3.00 per 1M reasoning tokens (same as input)
	CacheReadPerMillion  float64 // $0.30 per 1M cache read tokens
	CacheWritePerMillion float64 // $3.75 per 1M cache write tokens
}

// DefaultSonnet45Pricing returns the current Sonnet 4.5 pricing.
func DefaultSonnet45Pricing() Sonnet45Pricing {
	return Sonnet45Pricing{
		InputPerMillion:      3.00,
		OutputPerMillion:     15.00,
		ReasoningPerMillion:  3.00,
		CacheReadPerMillion:  0.30,
		CacheWritePerMillion: 3.75,
	}
}

// CostResult represents calculated cost for token usage.
type CostResult struct {
	InputCost       float64 `json:"input_cost"`        // Cost for input tokens in USD
	OutputCost      float64 `json:"output_cost"`       // Cost for output tokens in USD
	ReasoningCost   float64 `json:"reasoning_cost"`    // Cost for reasoning tokens in USD
	CacheReadCost   float64 `json:"cache_read_cost"`   // Cost for cache read tokens in USD
	CacheWriteCost  float64 `json:"cache_write_cost"`  // Cost for cache write tokens in USD (estimated)
	TotalCost       float64 `json:"total_cost"`        // Total cost in USD
	InputTokens     int     `json:"input_tokens"`      // Input token count
	OutputTokens    int     `json:"output_tokens"`     // Output token count
	ReasoningTokens int     `json:"reasoning_tokens"`  // Reasoning token count
	CacheReadTokens int     `json:"cache_read_tokens"` // Cache read token count
}

// CalculateCost calculates the cost for given token stats using Sonnet 4.5 pricing.
// Note: Cache write tokens are estimated as 30% of input tokens based on typical usage patterns.
func CalculateCost(stats opencode.TokenStats, pricing Sonnet45Pricing) CostResult {
	// Calculate costs for each token type
	inputCost := (float64(stats.InputTokens) / 1_000_000.0) * pricing.InputPerMillion
	outputCost := (float64(stats.OutputTokens) / 1_000_000.0) * pricing.OutputPerMillion
	reasoningCost := (float64(stats.ReasoningTokens) / 1_000_000.0) * pricing.ReasoningPerMillion
	cacheReadCost := (float64(stats.CacheReadTokens) / 1_000_000.0) * pricing.CacheReadPerMillion

	// Estimate cache write tokens as 30% of input tokens (typical pattern)
	cacheWriteTokens := int(float64(stats.InputTokens) * 0.3)
	cacheWriteCost := (float64(cacheWriteTokens) / 1_000_000.0) * pricing.CacheWritePerMillion

	totalCost := inputCost + outputCost + reasoningCost + cacheReadCost + cacheWriteCost

	return CostResult{
		InputCost:       roundToCents(inputCost),
		OutputCost:      roundToCents(outputCost),
		ReasoningCost:   roundToCents(reasoningCost),
		CacheReadCost:   roundToCents(cacheReadCost),
		CacheWriteCost:  roundToCents(cacheWriteCost),
		TotalCost:       roundToCents(totalCost),
		InputTokens:     stats.InputTokens,
		OutputTokens:    stats.OutputTokens,
		ReasoningTokens: stats.ReasoningTokens,
		CacheReadTokens: stats.CacheReadTokens,
	}
}

// CalculateCostFromStats is a convenience function that uses default pricing.
func CalculateCostFromStats(stats opencode.TokenStats) CostResult {
	return CalculateCost(stats, DefaultSonnet45Pricing())
}

// DailyCost represents cost aggregated by day.
type DailyCost struct {
	Date      string  `json:"date"`       // YYYY-MM-DD
	TotalCost float64 `json:"total_cost"` // Total cost for the day in USD
	Count     int     `json:"count"`      // Number of sessions included
}

// MonthlyCost represents cost aggregated by month.
type MonthlyCost struct {
	Month     string  `json:"month"`      // YYYY-MM
	TotalCost float64 `json:"total_cost"` // Total cost for the month in USD
	Count     int     `json:"count"`      // Number of sessions included
}

// AgentCost represents cost for a specific agent/session.
type AgentCost struct {
	SessionID   string     `json:"session_id"`
	BeadsID     string     `json:"beads_id,omitempty"`
	Skill       string     `json:"skill,omitempty"`
	Task        string     `json:"task,omitempty"`
	Cost        CostResult `json:"cost"`
	SpawnedAt   time.Time  `json:"spawned_at"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
}

// roundToCents rounds a float to 2 decimal places (cents).
func roundToCents(value float64) float64 {
	return float64(int(value*100+0.5)) / 100.0
}

// FormatCost formats a cost value as a string with dollar sign.
func FormatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

// GetBudgetColor returns a color class based on cost relative to monthly budget.
// Green: < $100, Yellow: $100-$180, Red: > $180 (approaching $200 Max subscription)
func GetBudgetColor(monthlyCost float64) string {
	if monthlyCost < 100.0 {
		return "green"
	} else if monthlyCost < 180.0 {
		return "yellow"
	}
	return "red"
}

// GetBudgetEmoji returns an emoji based on cost relative to monthly budget.
func GetBudgetEmoji(monthlyCost float64) string {
	if monthlyCost < 100.0 {
		return "🟢"
	} else if monthlyCost < 180.0 {
		return "🟡"
	}
	return "🔴"
}

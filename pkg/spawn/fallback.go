package spawn

import (
	"fmt"
	"sort"

	"github.com/dylan-conlin/orch-go/pkg/account"
)

// FallbackAction describes which fallback path the system should take.
type FallbackAction string

const (
	// FallbackNone means no fallback is needed — use the default model.
	FallbackNone FallbackAction = "none"
	// FallbackAlternateAccount means switch to another Anthropic account that has Opus headroom.
	FallbackAlternateAccount FallbackAction = "alternate-account"
	// FallbackSonnet means downgrade to Sonnet on the same Claude backend.
	FallbackSonnet FallbackAction = "sonnet"
	// FallbackGPT means route to GPT-5.4 (only valid for feature-impl).
	FallbackGPT FallbackAction = "gpt"
	// FallbackEscalate means stop and surface the constraint to the operator.
	FallbackEscalate FallbackAction = "escalate"
)

// FallbackInput provides the context needed for a fallback decision.
type FallbackInput struct {
	// Skill is the skill being spawned (e.g., "architect", "feature-impl").
	Skill string
	// CurrentCapacity is the capacity of the current Anthropic account.
	CurrentCapacity *account.CapacityInfo
	// AlternateCapacity maps account names to their capacity info.
	// Used to check if another account has Opus headroom.
	AlternateCapacity map[string]*account.CapacityInfo
}

// FallbackResult describes the recommended fallback action with an operator-visible reason.
type FallbackResult struct {
	// Action is the fallback path to take.
	Action FallbackAction
	// Reason is a human-readable explanation of why this fallback was chosen.
	// Always populated when Action != FallbackNone.
	Reason string
	// AccountName is set when Action == FallbackAlternateAccount.
	AccountName string
	// ModelAlias is set when Action == FallbackSonnet or FallbackGPT.
	ModelAlias string
}

// reasoningHeavySkills are skills that require Opus-level reasoning and should
// not be silently routed to GPT-5.4. This set mirrors skillModelMapping in
// pkg/daemon/skill_inference.go.
var reasoningHeavySkills = map[string]bool{
	"systematic-debugging": true,
	"investigation":        true,
	"architect":            true,
	"codebase-audit":       true,
	"research":             true,
}

// IsReasoningHeavySkill returns true if the skill requires Opus-level reasoning.
func IsReasoningHeavySkill(skill string) bool {
	return reasoningHeavySkills[skill]
}

// FallbackDecision implements the Opus rate-limit fallback cascade.
//
// The cascade priority order is:
//  1. Stay on Opus by switching to the healthiest alternate Anthropic account
//  2. Downgrade to Sonnet on Claude backend (when Opus exhausted but Claude healthy)
//  3. Use GPT-5.4 for feature-impl only (when Anthropic path is unhealthy)
//  4. Escalate for reasoning-heavy skills (stop, don't silently downgrade)
//
// Non-reasoning skills (feature-impl) skip Opus-specific checks because they
// already default to Sonnet via the resolve pipeline.
func FallbackDecision(input FallbackInput) FallbackResult {
	// No capacity data = can't make a routing decision, escalate
	if input.CurrentCapacity == nil || input.CurrentCapacity.Error != "" {
		return FallbackResult{
			Action: FallbackEscalate,
			Reason: "capacity data unavailable — cannot determine safe fallback",
		}
	}

	isReasoning := IsReasoningHeavySkill(input.Skill)

	// Non-reasoning skills (feature-impl) don't use Opus, so Opus exhaustion
	// is irrelevant. Only check if the Anthropic path itself is unhealthy.
	if !isReasoning {
		if input.CurrentCapacity.IsHealthy() {
			return FallbackResult{Action: FallbackNone}
		}
		// Anthropic path unhealthy — feature-impl can overflow to GPT-5.4
		if input.CurrentCapacity.IsCritical() {
			return FallbackResult{
				Action:    FallbackGPT,
				ModelAlias: "gpt-5.4",
				Reason: fmt.Sprintf("Anthropic capacity critical (5h: %.0f%% remaining, weekly: %.0f%% remaining) — routing feature-impl to GPT-5.4",
					input.CurrentCapacity.FiveHourRemaining, input.CurrentCapacity.SevenDayRemaining),
			}
		}
		return FallbackResult{Action: FallbackNone}
	}

	// === Reasoning-heavy skill path ===

	// If Opus is healthy, no fallback needed
	if input.CurrentCapacity.IsOpusHealthy() && input.CurrentCapacity.IsHealthy() {
		return FallbackResult{Action: FallbackNone}
	}

	// Step 1: Check if an alternate account has Opus headroom
	if best, ok := bestAlternateForOpus(input.AlternateCapacity); ok {
		return FallbackResult{
			Action:      FallbackAlternateAccount,
			AccountName: best,
			Reason: fmt.Sprintf("Opus exhausted on current account (%.0f%% remaining) — switching to %s which has %.0f%% Opus headroom",
				input.CurrentCapacity.SevenDayOpusRemaining, best, input.AlternateCapacity[best].SevenDayOpusRemaining),
		}
	}

	// Step 2: If Opus is exhausted but generic Claude is healthy, downgrade to Sonnet
	if input.CurrentCapacity.IsHealthy() && !input.CurrentCapacity.IsOpusHealthy() {
		return FallbackResult{
			Action:    FallbackSonnet,
			ModelAlias: "sonnet",
			Reason: fmt.Sprintf("Opus weekly capacity exhausted (%.0f%% remaining) but Claude capacity healthy (5h: %.0f%%, weekly: %.0f%%) — downgrading to Sonnet",
				input.CurrentCapacity.SevenDayOpusRemaining, input.CurrentCapacity.FiveHourRemaining, input.CurrentCapacity.SevenDayRemaining),
		}
	}

	// Step 3 & 4: Anthropic path unhealthy — reasoning skills cannot safely cross to GPT
	return FallbackResult{
		Action: FallbackEscalate,
		Reason: fmt.Sprintf("Anthropic capacity unhealthy (5h: %.0f%% remaining, weekly: %.0f%% remaining) and skill %q requires Opus-level reasoning — cannot safely route to GPT-5.4",
			input.CurrentCapacity.FiveHourRemaining, input.CurrentCapacity.SevenDayRemaining, input.Skill),
	}
}

// bestAlternateForOpus finds the alternate account with the most Opus headroom.
// Returns the account name and true if one has healthy Opus capacity.
func bestAlternateForOpus(alternates map[string]*account.CapacityInfo) (string, bool) {
	if len(alternates) == 0 {
		return "", false
	}

	type candidate struct {
		name    string
		opusRem float64
	}

	var candidates []candidate
	for name, cap := range alternates {
		if cap != nil && cap.Error == "" && cap.IsOpusHealthy() {
			candidates = append(candidates, candidate{name: name, opusRem: cap.SevenDayOpusRemaining})
		}
	}

	if len(candidates) == 0 {
		return "", false
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].opusRem != candidates[j].opusRem {
			return candidates[i].opusRem > candidates[j].opusRem
		}
		return candidates[i].name < candidates[j].name
	})

	return candidates[0].name, true
}

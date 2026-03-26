package spawn

import (
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/account"
)

func TestFallbackDecision(t *testing.T) {
	// Helper to build capacity with Opus-specific fields.
	mkCap := func(fiveHourRem, weeklyRem, opusRem float64) *account.CapacityInfo {
		return &account.CapacityInfo{
			FiveHourRemaining:      fiveHourRem,
			SevenDayRemaining:      weeklyRem,
			SevenDayOpusRemaining:  opusRem,
			SevenDayOpusUsed:       100 - opusRem,
		}
	}

	t.Run("healthy opus returns no fallback", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "architect",
			CurrentCapacity: mkCap(80, 60, 50),
		})
		if result.Action != FallbackNone {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackNone)
		}
	})

	t.Run("opus exhausted but claude healthy -> sonnet for reasoning skill", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "investigation",
			CurrentCapacity: mkCap(50, 40, 2),
		})
		if result.Action != FallbackSonnet {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackSonnet)
		}
		if result.Reason == "" {
			t.Fatal("Reason should be non-empty")
		}
	})

	t.Run("opus exhausted but claude healthy -> sonnet for feature-impl too", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "feature-impl",
			CurrentCapacity: mkCap(50, 40, 2),
		})
		// feature-impl doesn't normally use Opus, so this should be no-op
		if result.Action != FallbackNone {
			t.Fatalf("Action = %q, want %q for feature-impl (already uses Sonnet)", result.Action, FallbackNone)
		}
	})

	t.Run("alternate account has opus headroom -> switch account", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "architect",
			CurrentCapacity: mkCap(50, 40, 2),
			AlternateCapacity: map[string]*account.CapacityInfo{
				"personal": mkCap(70, 60, 45),
			},
		})
		if result.Action != FallbackAlternateAccount {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackAlternateAccount)
		}
		if result.AccountName != "personal" {
			t.Fatalf("AccountName = %q, want %q", result.AccountName, "personal")
		}
	})

	t.Run("anthropic path unhealthy + feature-impl -> gpt fallback", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "feature-impl",
			CurrentCapacity: mkCap(3, 2, 0),
		})
		if result.Action != FallbackGPT {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackGPT)
		}
	})

	t.Run("anthropic path unhealthy + reasoning skill -> escalate", func(t *testing.T) {
		for _, skill := range []string{"architect", "investigation", "systematic-debugging", "research", "codebase-audit"} {
			result := FallbackDecision(FallbackInput{
				Skill:           skill,
				CurrentCapacity: mkCap(3, 2, 0),
			})
			if result.Action != FallbackEscalate {
				t.Fatalf("skill=%s: Action = %q, want %q", skill, result.Action, FallbackEscalate)
			}
		}
	})

	t.Run("nil capacity returns escalate", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "architect",
			CurrentCapacity: nil,
		})
		if result.Action != FallbackEscalate {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackEscalate)
		}
	})

	t.Run("capacity error returns escalate", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "architect",
			CurrentCapacity: &account.CapacityInfo{Error: "auth failed"},
		})
		if result.Action != FallbackEscalate {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackEscalate)
		}
	})

	t.Run("alternate account chosen is the one with most opus headroom", func(t *testing.T) {
		result := FallbackDecision(FallbackInput{
			Skill:           "architect",
			CurrentCapacity: mkCap(50, 40, 2),
			AlternateCapacity: map[string]*account.CapacityInfo{
				"alpha": mkCap(60, 50, 30),
				"beta":  mkCap(70, 60, 55),
			},
		})
		if result.Action != FallbackAlternateAccount {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackAlternateAccount)
		}
		if result.AccountName != "beta" {
			t.Fatalf("AccountName = %q, want %q (highest opus headroom)", result.AccountName, "beta")
		}
	})

	t.Run("feature-impl skips opus-specific checks entirely", func(t *testing.T) {
		// feature-impl uses Sonnet by default, so Opus exhaustion doesn't matter.
		// Only Anthropic-wide exhaustion triggers fallback.
		result := FallbackDecision(FallbackInput{
			Skill:           "feature-impl",
			CurrentCapacity: mkCap(50, 40, 0), // Opus totally gone, but claude healthy
		})
		if result.Action != FallbackNone {
			t.Fatalf("Action = %q, want %q", result.Action, FallbackNone)
		}
	})
}

func TestIsReasoningHeavySkill(t *testing.T) {
	reasoning := []string{"architect", "investigation", "systematic-debugging", "research", "codebase-audit"}
	for _, s := range reasoning {
		if !IsReasoningHeavySkill(s) {
			t.Errorf("IsReasoningHeavySkill(%q) = false, want true", s)
		}
	}
	nonReasoning := []string{"feature-impl", "issue-creation", "", "unknown"}
	for _, s := range nonReasoning {
		if IsReasoningHeavySkill(s) {
			t.Errorf("IsReasoningHeavySkill(%q) = true, want false", s)
		}
	}
}

package model

import "strings"

const (
	// ProfileStrictComplete is for models that reliably report phase transitions
	// and completion without additional prompting.
	ProfileStrictComplete = "strict-complete"
	// ProfileNeedsNudge is for models that benefit from explicit completion nudges.
	ProfileNeedsNudge = "needs-nudge"
)

// BehaviorProfile describes model-specific execution behavior traits.
type BehaviorProfile struct {
	Name string

	// NeedsCompletionNudge indicates the model often benefits from explicit
	// reminders to emit completion signaling (Phase: Complete).
	NeedsCompletionNudge bool

	// ReliablePhaseReporting indicates whether the model reliably reports phase
	// transitions (Planning/Implementing/Testing/Complete) without reminders.
	ReliablePhaseReporting bool

	// NeedsExplicitGitCommit indicates the model does not naturally run
	// git add && git commit after completing work. Without explicit instruction,
	// work exists only as uncommitted changes that cannot be integrated.
	NeedsExplicitGitCommit bool
}

var strictCompleteProfile = BehaviorProfile{
	Name:                   ProfileStrictComplete,
	NeedsCompletionNudge:   false,
	ReliablePhaseReporting: true,
	NeedsExplicitGitCommit: false,
}

var needsNudgeProfile = BehaviorProfile{
	Name:                   ProfileNeedsNudge,
	NeedsCompletionNudge:   true,
	ReliablePhaseReporting: false,
	NeedsExplicitGitCommit: true,
}

// behaviorProfileOverrides maps model aliases and canonical model IDs to
// behavior profiles. This gives us explicit control for known model names.
var behaviorProfileOverrides = map[string]BehaviorProfile{
	// Anthropic (reliable)
	"opus":                                 strictCompleteProfile,
	"sonnet":                               strictCompleteProfile,
	"haiku":                                strictCompleteProfile,
	"claude-opus-4-6":                      strictCompleteProfile,
	"claude-opus-4-5-20251101":             strictCompleteProfile,
	"claude-sonnet-4-5-20250929":           strictCompleteProfile,
	"claude-haiku-4-5-20251001":            strictCompleteProfile,
	"anthropic/claude-opus-4-6":            strictCompleteProfile,
	"anthropic/claude-opus-4-5-20251101":   strictCompleteProfile,
	"anthropic/claude-sonnet-4-5-20250929": strictCompleteProfile,

	// Google/OpenAI/DeepSeek/Alibaba (needs nudge)
	"flash":                         needsNudgeProfile,
	"pro":                           needsNudgeProfile,
	"gpt":                           needsNudgeProfile,
	"codex":                         needsNudgeProfile,
	"deepseek":                      needsNudgeProfile,
	"qwen":                          needsNudgeProfile,
	"gemini-3-flash-preview":        needsNudgeProfile,
	"gemini-2.5-flash":              needsNudgeProfile,
	"gemini-2.5-pro":                needsNudgeProfile,
	"gpt-5.3-codex":                 needsNudgeProfile,
	"gpt-5.2":                       needsNudgeProfile,
	"gpt-5.2-codex":                 needsNudgeProfile,
	"gpt-5-mini-20251130":           needsNudgeProfile,
	"gpt-4o":                        needsNudgeProfile,
	"gpt-4o-mini":                   needsNudgeProfile,
	"deepseek-chat":                 needsNudgeProfile,
	"deepseek-v3.2":                 needsNudgeProfile,
	"deepseek-r1":                   needsNudgeProfile,
	"qwen3-max":                     needsNudgeProfile,
	"qwen3-max-2026-01-23":          needsNudgeProfile,
	"google/gemini-3-flash-preview": needsNudgeProfile,
	"google/gemini-2.5-flash":       needsNudgeProfile,
	"google/gemini-2.5-pro":         needsNudgeProfile,
	"openai/gpt-5.3-codex":          needsNudgeProfile,
	"openai/gpt-5.2":                needsNudgeProfile,
	"openai/gpt-4o":                 needsNudgeProfile,
	"deepseek/deepseek-chat":        needsNudgeProfile,
	"alibaba/qwen3-max":             needsNudgeProfile,
}

// ResolveBehaviorProfile resolves a model spec (alias, provider/model, or raw model ID)
// to a behavior profile.
func ResolveBehaviorProfile(spec string) BehaviorProfile {
	normalized := strings.ToLower(strings.TrimSpace(spec))
	if normalized == "" {
		return strictCompleteProfile
	}

	if profile, ok := behaviorProfileOverrides[normalized]; ok {
		return profile
	}

	resolved := Resolve(spec)
	if profile, ok := behaviorProfileOverrides[strings.ToLower(resolved.Format())]; ok {
		return profile
	}
	if profile, ok := behaviorProfileOverrides[strings.ToLower(resolved.ModelID)]; ok {
		return profile
	}

	// Provider-level fallback when a specific model isn't mapped yet.
	switch strings.ToLower(resolved.Provider) {
	case "anthropic":
		return strictCompleteProfile
	case "google", "openai", "deepseek", "alibaba":
		return needsNudgeProfile
	default:
		// Conservative default: prefer nudging over missed completion signaling.
		return needsNudgeProfile
	}
}

package events

import "strings"

const (
	AbandonmentReasonStalled          = "stalled"
	AbandonmentReasonInfra            = "infra"
	AbandonmentReasonRateLimit        = "rate-limit"
	AbandonmentReasonScopeMismatch    = "scope-mismatch"
	AbandonmentReasonManualCleanup    = "manual-cleanup"
	AbandonmentReasonContextExhausted = "context-exhaustion"
	AbandonmentReasonUnknown          = "unknown"
)

var abandonmentReasonTaxonomy = []string{
	AbandonmentReasonStalled,
	AbandonmentReasonInfra,
	AbandonmentReasonRateLimit,
	AbandonmentReasonScopeMismatch,
	AbandonmentReasonManualCleanup,
	AbandonmentReasonContextExhausted,
}

var abandonmentReasonAliases = map[string]string{
	"stalled":            AbandonmentReasonStalled,
	"stuck":              AbandonmentReasonStalled,
	"dead-session":       AbandonmentReasonStalled,
	"dead_session":       AbandonmentReasonStalled,
	"auto-abandoned":     AbandonmentReasonStalled,
	"infra":              AbandonmentReasonInfra,
	"infrastructure":     AbandonmentReasonInfra,
	"service-failure":    AbandonmentReasonInfra,
	"service_failure":    AbandonmentReasonInfra,
	"rate-limit":         AbandonmentReasonRateLimit,
	"rate_limit":         AbandonmentReasonRateLimit,
	"ratelimit":          AbandonmentReasonRateLimit,
	"quota":              AbandonmentReasonRateLimit,
	"scope-mismatch":     AbandonmentReasonScopeMismatch,
	"scope_mismatch":     AbandonmentReasonScopeMismatch,
	"requirements-gap":   AbandonmentReasonScopeMismatch,
	"manual-cleanup":     AbandonmentReasonManualCleanup,
	"manual_cleanup":     AbandonmentReasonManualCleanup,
	"cleanup":            AbandonmentReasonManualCleanup,
	"operator":           AbandonmentReasonManualCleanup,
	"context-exhaustion": AbandonmentReasonContextExhausted,
	"context_exhaustion": AbandonmentReasonContextExhausted,
	"context-window":     AbandonmentReasonContextExhausted,
	"context_window":     AbandonmentReasonContextExhausted,
	"token-limit":        AbandonmentReasonContextExhausted,
}

type abandonmentReasonKeywordMatcher struct {
	reasonCode string
	keywords   []string
}

var abandonmentReasonKeywordMatchers = []abandonmentReasonKeywordMatcher{
	{reasonCode: AbandonmentReasonContextExhausted, keywords: []string{"context window", "context exhausted", "context-exhausted", "too much context", "too many tokens", "token limit", "max tokens"}},
	{reasonCode: AbandonmentReasonRateLimit, keywords: []string{"rate limit", "rate-limit", "429", "quota", "throttl", "usage limit", "credits exhausted"}},
	{reasonCode: AbandonmentReasonScopeMismatch, keywords: []string{"scope mismatch", "scope-mismatch", "out of scope", "requirements mismatch", "unclear requirements", "wrong task", "wrong issue"}},
	{reasonCode: AbandonmentReasonStalled, keywords: []string{"stalled", "stuck", "dead session", "dead-session", "frozen", "hung", "no progress", "stuck in loop", "auto-abandoned"}},
	{reasonCode: AbandonmentReasonInfra, keywords: []string{"infra", "infrastructure", "server crash", "daemon crash", "opencode crash", "auth failed", "network", "connection refused", "connection reset", "tls"}},
	{reasonCode: AbandonmentReasonManualCleanup, keywords: []string{"manual cleanup", "manual-cleanup", "cleanup", "clean up", "operator", "cancelled by user", "cancelled by operator"}},
}

// AbandonmentReasonTaxonomy returns the valid abandonment reason codes.
func AbandonmentReasonTaxonomy() []string {
	result := make([]string, len(abandonmentReasonTaxonomy))
	copy(result, abandonmentReasonTaxonomy)
	return result
}

// NormalizeAbandonmentReasonCode canonicalizes a reason code alias to a taxonomy value.
// Returns empty string when the code is not recognized.
func NormalizeAbandonmentReasonCode(code string) string {
	normalized := strings.TrimSpace(strings.ToLower(code))
	if normalized == "" {
		return ""
	}
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Join(strings.Fields(normalized), "-")
	if canonical, ok := abandonmentReasonAliases[normalized]; ok {
		return canonical
	}
	return ""
}

// InferAbandonmentReasonCode infers a taxonomy code from free-form reason text.
// Returns empty string when no taxonomy category can be inferred.
func InferAbandonmentReasonCode(reason string) string {
	cleaned := strings.TrimSpace(strings.ToLower(reason))
	if cleaned == "" {
		return ""
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	for _, matcher := range abandonmentReasonKeywordMatchers {
		for _, keyword := range matcher.keywords {
			if strings.Contains(cleaned, keyword) {
				return matcher.reasonCode
			}
		}
	}

	return ""
}

// IsValidAbandonmentReasonCode returns true when code belongs to the taxonomy.
func IsValidAbandonmentReasonCode(code string) bool {
	return NormalizeAbandonmentReasonCode(code) != ""
}

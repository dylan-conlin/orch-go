package kbmetrics

import (
	"regexp"
	"strings"
)

// EvidenceTier represents the declared evidence strength for a claim.
type EvidenceTier int

const (
	TierUnclassified EvidenceTier = iota
	TierAssumed                   // No direct evidence
	TierHypothesis                // Plausible synthesis, not directly tested
	TierObserved                  // Single experiment or observation, one context
	TierReplicated                // Reproduced across >1 independent context
	TierValidated                 // Replicated + external confirmation
)

func (t EvidenceTier) String() string {
	switch t {
	case TierAssumed:
		return "assumed"
	case TierHypothesis:
		return "working-hypothesis"
	case TierObserved:
		return "observed"
	case TierReplicated:
		return "replicated"
	case TierValidated:
		return "validated"
	default:
		return "unclassified"
	}
}

// ClassifyTier maps an **Evidence quality:** annotation string to a tier.
func ClassifyTier(annotation string) EvidenceTier {
	lower := strings.ToLower(annotation)

	switch {
	case strings.Contains(lower, "validated"):
		return TierValidated
	case strings.Contains(lower, "replicated"),
		strings.Contains(lower, "multi-source"):
		return TierReplicated
	case strings.Contains(lower, "observed"),
		strings.Contains(lower, "single-source"):
		return TierObserved
	case strings.Contains(lower, "working-hypothesis"),
		strings.Contains(lower, "working hypothesis"):
		return TierHypothesis
	case strings.Contains(lower, "assumed"):
		return TierAssumed
	default:
		return TierUnclassified
	}
}

// DriftInput is a claim with its declared tier, ready for drift analysis.
type DriftInput struct {
	ClaimText string
	Tier      EvidenceTier
	ClaimLine int
}

// DriftFlag is a detected mismatch between declared tier and claim language.
type DriftFlag struct {
	Line     int
	Claim    string
	Tier     EvidenceTier
	Triggers []string // which overclaim phrases were found
}

// Overclaim indicators: phrases that imply generality, absoluteness, or
// universal validity. These are only problematic at lower evidence tiers.
var overclaimPatterns = []struct {
	re       *regexp.Regexp
	label    string
	minTier  EvidenceTier // tier at which this language becomes acceptable
}{
	{regexp.MustCompile(`(?i)\bfundamentally\b`), "fundamentally", TierValidated},
	{regexp.MustCompile(`(?i)\buniversally\b`), "universally", TierValidated},
	{regexp.MustCompile(`(?i)\balways\b`), "always", TierReplicated},
	{regexp.MustCompile(`(?i)\bnever\b`), "never", TierReplicated},
	{regexp.MustCompile(`(?i)\bproven\b`), "proven", TierValidated},
	{regexp.MustCompile(`(?i)\bvalidated\s+generally\b`), "validated generally", TierValidated},
	{regexp.MustCompile(`(?i)\bcompletely\b`), "completely", TierReplicated},
	{regexp.MustCompile(`(?i)\b100%\s+reliability\b`), "100% reliability", TierReplicated},
	{regexp.MustCompile(`(?i)\bgeneral\s+to\s+any\b`), "general to any", TierValidated},
	{regexp.MustCompile(`(?i)\ball\s+(?:systems|frameworks|approaches)\b`), "all systems/frameworks", TierValidated},
	{regexp.MustCompile(`(?i)\binherently\b`), "inherently", TierValidated},
	{regexp.MustCompile(`(?i)\bimpossible\b`), "impossible", TierValidated},
}

// DetectDrift checks each claim for language that exceeds its declared
// evidence tier. Returns flags for claims where prose strength exceeds
// the declared tier.
func DetectDrift(claims []DriftInput) []DriftFlag {
	var flags []DriftFlag

	for _, c := range claims {
		var triggers []string
		for _, p := range overclaimPatterns {
			if c.Tier < p.minTier && p.re.MatchString(c.ClaimText) {
				triggers = append(triggers, p.label)
			}
		}
		if len(triggers) > 0 {
			flags = append(flags, DriftFlag{
				Line:     c.ClaimLine,
				Claim:    truncate(c.ClaimText, 120),
				Tier:     c.Tier,
				Triggers: triggers,
			})
		}
	}

	return flags
}

package verify

import "fmt"

// SkipConfig holds the configuration for which verification gates to skip.
type SkipConfig struct {
	TestEvidence         bool
	Visual               bool
	GitDiff              bool
	Synthesis            bool
	Build                bool
	Constraint           bool
	PhaseGate            bool
	SkillOutput          bool
	DecisionPatch        bool
	PhaseComplete        bool
	HandoffContent       bool
	ExplainBack          bool
	Accretion            bool
	ArchitecturalChoices bool
	SelfReview           bool
	ProbeModelMerge      bool
	Reason               string // Required reason for skips
}

// HasAnySkip returns true if any skip flag is set.
func (c SkipConfig) HasAnySkip() bool {
	return c.TestEvidence || c.Visual || c.GitDiff || c.Synthesis ||
		c.Build || c.Constraint || c.PhaseGate || c.SkillOutput ||
		c.DecisionPatch || c.PhaseComplete || c.HandoffContent || c.ExplainBack ||
		c.Accretion || c.ArchitecturalChoices || c.SelfReview ||
		c.ProbeModelMerge
}

// SkippedGates returns a list of gate names that are being skipped.
func (c SkipConfig) SkippedGates() []string {
	var gates []string
	if c.TestEvidence {
		gates = append(gates, GateTestEvidence)
	}
	if c.Visual {
		gates = append(gates, GateVisualVerify)
	}
	if c.GitDiff {
		gates = append(gates, GateGitDiff)
	}
	if c.Synthesis {
		gates = append(gates, GateSynthesis)
	}
	if c.Build {
		gates = append(gates, GateBuild)
	}
	if c.Constraint {
		gates = append(gates, GateConstraint)
	}
	if c.PhaseGate {
		gates = append(gates, GatePhaseGate)
	}
	if c.SkillOutput {
		gates = append(gates, GateSkillOutput)
	}
	if c.DecisionPatch {
		gates = append(gates, GateDecisionPatchLimit)
	}
	if c.PhaseComplete {
		gates = append(gates, GatePhaseComplete)
	}
	if c.HandoffContent {
		gates = append(gates, GateHandoffContent)
	}
	if c.ExplainBack {
		gates = append(gates, GateExplainBack)
	}
	if c.Accretion {
		gates = append(gates, GateAccretion)
	}
	if c.ArchitecturalChoices {
		gates = append(gates, GateArchitecturalChoices)
	}
	if c.SelfReview {
		gates = append(gates, GateSelfReview)
	}
	if c.ProbeModelMerge {
		gates = append(gates, GateProbeModelMerge)
	}
	return gates
}

// ShouldSkipGate returns true if the given gate should be skipped.
func (c SkipConfig) ShouldSkipGate(gate string) bool {
	switch gate {
	case GateTestEvidence:
		return c.TestEvidence
	case GateVisualVerify:
		return c.Visual
	case GateGitDiff:
		return c.GitDiff
	case GateSynthesis:
		return c.Synthesis
	case GateBuild:
		return c.Build
	case GateConstraint:
		return c.Constraint
	case GatePhaseGate:
		return c.PhaseGate
	case GateSkillOutput:
		return c.SkillOutput
	case GateDecisionPatchLimit:
		return c.DecisionPatch
	case GatePhaseComplete:
		return c.PhaseComplete
	case GateHandoffContent:
		return c.HandoffContent
	case GateExplainBack:
		return c.ExplainBack
	case GateAccretion:
		return c.Accretion
	case GateArchitecturalChoices:
		return c.ArchitecturalChoices
	case GateSelfReview:
		return c.SelfReview
	case GateProbeModelMerge:
		return c.ProbeModelMerge
	default:
		return false
	}
}

// ValidateSkipFlags validates that Reason is provided when skip flags are used.
func ValidateSkipFlags(skipConfig SkipConfig) error {
	if !skipConfig.HasAnySkip() {
		return nil
	}

	if skipConfig.Reason == "" {
		return fmt.Errorf("--skip-reason is required when using --skip-* flags")
	}

	if len(skipConfig.Reason) < 10 {
		return fmt.Errorf("--skip-reason must be at least 10 characters (got %d)", len(skipConfig.Reason))
	}

	return nil
}

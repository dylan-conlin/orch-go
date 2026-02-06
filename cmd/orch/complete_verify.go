// Package main provides verification logic for the complete command.
// Extracted from complete_cmd.go for maintainability.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// SkipConfig holds the configuration for which verification gates to skip.
type SkipConfig struct {
	TestEvidence    bool
	Visual          bool
	GitDiff         bool
	Synthesis       bool
	Build           bool
	Constraint      bool
	PhaseGate       bool
	SkillOutput     bool
	DecisionPatch   bool
	PhaseComplete   bool
	HandoffContent  bool
	DashboardHealth bool
	Reason          string // Required reason for skips
}

// hasAnySkip returns true if any skip flag is set.
func (c SkipConfig) hasAnySkip() bool {
	return c.TestEvidence || c.Visual || c.GitDiff || c.Synthesis ||
		c.Build || c.Constraint || c.PhaseGate || c.SkillOutput ||
		c.DecisionPatch || c.PhaseComplete || c.HandoffContent || c.DashboardHealth
}

// skippedGates returns a list of gate names that are being skipped.
func (c SkipConfig) skippedGates() []string {
	var gates []string
	if c.TestEvidence {
		gates = append(gates, verify.GateTestEvidence)
	}
	if c.Visual {
		gates = append(gates, verify.GateVisualVerify)
	}
	if c.GitDiff {
		gates = append(gates, verify.GateGitDiff)
	}
	if c.Synthesis {
		gates = append(gates, verify.GateSynthesis)
	}
	if c.Build {
		gates = append(gates, verify.GateBuild)
	}
	if c.Constraint {
		gates = append(gates, verify.GateConstraint)
	}
	if c.PhaseGate {
		gates = append(gates, verify.GatePhaseGate)
	}
	if c.SkillOutput {
		gates = append(gates, verify.GateSkillOutput)
	}
	if c.DecisionPatch {
		gates = append(gates, verify.GateDecisionPatchLimit)
	}
	if c.PhaseComplete {
		gates = append(gates, verify.GatePhaseComplete)
	}
	if c.HandoffContent {
		gates = append(gates, verify.GateHandoffContent)
	}
	if c.DashboardHealth {
		gates = append(gates, verify.GateDashboardHealth)
	}
	return gates
}

// shouldSkipGate returns true if the given gate should be skipped.
func (c SkipConfig) shouldSkipGate(gate string) bool {
	switch gate {
	case verify.GateTestEvidence:
		return c.TestEvidence
	case verify.GateVisualVerify:
		return c.Visual
	case verify.GateGitDiff:
		return c.GitDiff
	case verify.GateSynthesis:
		return c.Synthesis
	case verify.GateBuild:
		return c.Build
	case verify.GateConstraint:
		return c.Constraint
	case verify.GatePhaseGate:
		return c.PhaseGate
	case verify.GateSkillOutput:
		return c.SkillOutput
	case verify.GateDecisionPatchLimit:
		return c.DecisionPatch
	case verify.GatePhaseComplete:
		return c.PhaseComplete
	case verify.GateHandoffContent:
		return c.HandoffContent
	case verify.GateDashboardHealth:
		return c.DashboardHealth
	default:
		return false
	}
}

// getSkipConfig builds the skip configuration from command-line flags.
func getSkipConfig() SkipConfig {
	return SkipConfig{
		TestEvidence:    completeSkipTestEvidence,
		Visual:          completeSkipVisual,
		GitDiff:         completeSkipGitDiff,
		Synthesis:       completeSkipSynthesis,
		Build:           completeSkipBuild,
		Constraint:      completeSkipConstraint,
		PhaseGate:       completeSkipPhaseGate,
		SkillOutput:     completeSkipSkillOutput,
		DecisionPatch:   completeSkipDecisionPatch,
		PhaseComplete:   completeSkipPhaseComplete,
		HandoffContent:  completeSkipHandoffContent,
		DashboardHealth: completeSkipDashboardHealth,
		Reason:          completeSkipReason,
	}
}

// validateSkipFlags validates that --skip-reason is provided when --skip-* flags are used.
func validateSkipFlags(skipConfig SkipConfig) error {
	if !skipConfig.hasAnySkip() {
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

// logSkipEvents logs verification.bypassed events for all skipped gates.
func logSkipEvents(skipConfig SkipConfig, beadsID, workspace, skill string) {
	logger := events.NewLogger(events.DefaultLogPath())
	for _, gate := range skipConfig.skippedGates() {
		if err := logger.LogVerificationBypassed(events.VerificationBypassedData{
			BeadsID:   beadsID,
			Workspace: workspace,
			Gate:      gate,
			Reason:    skipConfig.Reason,
			Skill:     skill,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log bypass event for %s: %v\n", gate, err)
		}
	}
}

// recordBuildSkipMemory persists a build gate skip decision for future completions.
// When the orchestrator uses --skip-build, this records the reason so subsequent
// completions auto-skip the build gate without requiring --skip-build again.
func recordBuildSkipMemory(skipConfig SkipConfig, projectDir, identifier string) {
	if !skipConfig.Build {
		return
	}
	if err := verify.RecordBuildSkip(projectDir, skipConfig.Reason, identifier); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to persist build skip memory: %v\n", err)
	} else {
		fmt.Printf("Build skip memory saved (expires in %v)\n", verify.BuildSkipDuration)
	}
}

// printGateResults prints a formatted summary of gate results showing which passed and failed,
// with error details for failures, and the specific --skip flags needed to bypass them.
func printGateResults(results []verify.GateResult, failed []string) {
	// Build a set of failed gates for quick lookup
	failedSet := make(map[string]bool, len(failed))
	for _, g := range failed {
		failedSet[g] = true
	}

	// Print per-gate results
	for _, gr := range results {
		name := verify.GateDisplayName(gr.Gate)
		if gr.Passed {
			fmt.Fprintf(os.Stderr, "  \033[32m✓\033[0m %s\n", name)
		} else {
			// Truncate long error messages to keep output readable
			errMsg := gr.Error
			if len(errMsg) > 120 {
				errMsg = errMsg[:117] + "..."
			}
			fmt.Fprintf(os.Stderr, "  \033[31m✗\033[0m %s: %s\n", name, errMsg)
		}
	}

	// Print skip flags for failing gates
	if len(failed) > 0 {
		var flags []string
		for _, g := range failed {
			flags = append(flags, verify.GateSkipFlag(g))
		}
		fmt.Fprintf(os.Stderr, "\nSkip failing gates with:\n")
		fmt.Fprintf(os.Stderr, "  %s --skip-reason '<reason>'\n", strings.Join(flags, " "))
	}
}

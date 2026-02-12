// Package main provides verification logic for the complete command.
// Extracted from complete_cmd.go for maintainability.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/episodic"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// SkipConfig holds the configuration for which verification gates to skip.
type SkipConfig struct {
	TestEvidence         bool
	ModelConnection      bool
	Visual               bool
	GitDiff              bool
	Synthesis            bool
	Build                bool
	Constraint           bool
	PhaseGate            bool
	SkillOutput          bool
	DecisionPatch        bool
	PhaseComplete        bool
	AgentRunning         bool
	HandoffContent       bool
	DashboardHealth      bool
	VerificationSpec     bool
	CommitEvidence       bool
	OrchestratorOverride []string // Gate names to override (allows core gate bypass with elevated logging)
	Reason               string // Required reason for skips
	BatchMode            bool   // Batch mode: skip all Tier 2 (quality) gates
}

// hasAnySkip returns true if any skip flag is set (including batch mode and orchestrator-override).
func (c SkipConfig) hasAnySkip() bool {
	return c.BatchMode || len(c.OrchestratorOverride) > 0 || c.TestEvidence || c.ModelConnection || c.Visual || c.GitDiff || c.Synthesis ||
		c.Build || c.Constraint || c.PhaseGate || c.SkillOutput ||
		c.DecisionPatch || c.PhaseComplete || c.AgentRunning || c.HandoffContent || c.DashboardHealth || c.VerificationSpec || c.CommitEvidence
}

// skippedGates returns a list of gate names that are being skipped.
func (c SkipConfig) skippedGates() []string {
	var gates []string
	gates = append(gates, c.OrchestratorOverride...)
	if c.TestEvidence {
		gates = append(gates, verify.GateTestEvidence)
	}
	if c.ModelConnection {
		gates = append(gates, verify.GateModelConnection)
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
	if c.AgentRunning {
		gates = append(gates, verify.GateAgentRunning)
	}
	if c.HandoffContent {
		gates = append(gates, verify.GateHandoffContent)
	}
	if c.DashboardHealth {
		gates = append(gates, verify.GateDashboardHealth)
	}
	if c.VerificationSpec {
		gates = append(gates, verify.GateVerificationSpec)
	}
	if c.CommitEvidence {
		gates = append(gates, verify.GateCommitEvidence)
	}
	return gates
}

// shouldSkipGate returns true if the given gate should be skipped.
// In batch mode, all Tier 2 (quality) gates are automatically skipped.
// Orchestrator override bypasses core gate protection for the single named gate.
// Core gates (Tier 1) are never skippable via --skip-* flags — they block completion unconditionally.
func (c SkipConfig) shouldSkipGate(gate string) bool {
	// Orchestrator override: elevated privilege to bypass any gate (including core gates)
	for _, override := range c.OrchestratorOverride {
		if override == gate {
			return true
		}
	}
	// Core gates cannot be skipped via --skip-* flags (only via orchestrator-override or --force)
	if verify.IsCoreGate(gate) {
		return false
	}
	if c.BatchMode && verify.IsQualityGate(gate) {
		return true
	}
	switch gate {
	case verify.GateTestEvidence:
		return c.TestEvidence
	case verify.GateModelConnection:
		return c.ModelConnection
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
	case verify.GateAgentRunning:
		return c.AgentRunning
	case verify.GateHandoffContent:
		return c.HandoffContent
	case verify.GateDashboardHealth:
		return c.DashboardHealth
	case verify.GateVerificationSpec:
		return c.VerificationSpec
	case verify.GateCommitEvidence:
		return c.CommitEvidence
	default:
		return false
	}
}

// getSkipConfig builds the skip configuration from command-line flags.
func getSkipConfig() SkipConfig {
	return SkipConfig{
		TestEvidence:         completeSkipTestEvidence,
		ModelConnection:      completeSkipModelConnection,
		Visual:               completeSkipVisual,
		GitDiff:              completeSkipGitDiff,
		Synthesis:            completeSkipSynthesis,
		Build:                completeSkipBuild,
		Constraint:           completeSkipConstraint,
		PhaseGate:            completeSkipPhaseGate,
		SkillOutput:          completeSkipSkillOutput,
		DecisionPatch:        completeSkipDecisionPatch,
		PhaseComplete:        completeSkipPhaseComplete,
		AgentRunning:         completeSkipAgentRunning,
		HandoffContent:       completeSkipHandoffContent,
		DashboardHealth:      completeSkipDashboardHealth,
		VerificationSpec:     completeSkipVerificationSpec,
		CommitEvidence:       completeSkipCommitEvidence,
		OrchestratorOverride: parseOrchestratorOverride(completeOrchestratorOverride),
		Reason:               completeSkipReason,
		BatchMode:            completeBatch,
	}
}

// parseOrchestratorOverride parses a comma-separated string of gate names into a slice.
// Returns nil for empty input. Trims whitespace around each gate name.
func parseOrchestratorOverride(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var gates []string
	for _, p := range parts {
		g := strings.TrimSpace(p)
		if g != "" {
			gates = append(gates, g)
		}
	}
	return gates
}

// buildBatchSkipConfig creates a SkipConfig for batch mode (used by batch-complete command).
func buildBatchSkipConfig() SkipConfig {
	return SkipConfig{
		BatchMode: true,
		Reason:    "batch mode - core gates only",
	}
}

// validateSkipFlags validates that --skip-reason is provided when --skip-* flags are used.
// Batch mode does not require --skip-reason (the reason is implicit).
// Orchestrator override requires --reason and validates the gate name.
// Core gates (Tier 1) cannot be skipped via --skip-* flags (only via orchestrator-override).
func validateSkipFlags(skipConfig SkipConfig) error {
	if skipConfig.BatchMode {
		return nil
	}

	// Orchestrator override: validate gate names and reason
	if len(skipConfig.OrchestratorOverride) > 0 {
		if skipConfig.Reason == "" {
			return fmt.Errorf("--reason is required when using --orchestrator-override")
		}
		if len(skipConfig.Reason) < 10 {
			return fmt.Errorf("--reason must be at least 10 characters (got %d)", len(skipConfig.Reason))
		}
		// Validate that each gate name is a known gate
		for _, gate := range skipConfig.OrchestratorOverride {
			if !isValidGateName(gate) {
				return fmt.Errorf("unknown gate name for --orchestrator-override: %s (valid gates: phase_complete, commit_evidence, synthesis, test_evidence, git_diff, build, visual_verification, model_connection, etc.)", gate)
			}
		}
		return nil
	}

	// Check for attempts to skip core gates via --skip-* — these are never allowed
	coreSkips := skipConfig.coreGateSkips()
	if len(coreSkips) > 0 {
		return fmt.Errorf("core gates cannot be skipped: %s (use --orchestrator-override <gate-name> --reason '<justification>' to bypass with elevated logging)", strings.Join(coreSkips, ", "))
	}

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

// isValidGateName returns true if the given gate name is a known gate constant.
func isValidGateName(gateName string) bool {
	validGates := map[string]bool{
		verify.GatePhaseComplete:      true,
		verify.GateSynthesis:          true,
		verify.GateHandoffContent:     true,
		verify.GateConstraint:         true,
		verify.GatePhaseGate:          true,
		verify.GateSkillOutput:        true,
		verify.GateVisualVerify:       true,
		verify.GateTestEvidence:       true,
		verify.GateModelConnection:    true,
		verify.GateVerificationSpec:   true,
		verify.GateGitDiff:            true,
		verify.GateBuild:              true,
		verify.GateDecisionPatchLimit: true,
		verify.GateDashboardHealth:    true,
		verify.GateAgentRunning:       true,
		verify.GateCommitEvidence:     true,
	}
	return validGates[gateName]
}

// coreGateSkips returns the names of core gates that the skip config attempts to skip.
func (c SkipConfig) coreGateSkips() []string {
	var skips []string
	coreChecks := []struct {
		flag bool
		gate string
	}{
		{c.PhaseComplete, verify.GatePhaseComplete},
		{c.CommitEvidence, verify.GateCommitEvidence},
		{c.Synthesis, verify.GateSynthesis},
		{c.TestEvidence, verify.GateTestEvidence},
		{c.GitDiff, verify.GateGitDiff},
	}
	for _, check := range coreChecks {
		if check.flag {
			skips = append(skips, check.gate)
		}
	}
	return skips
}

// logSkipEvents logs verification.bypassed events for all skipped gates.
func logSkipEvents(skipConfig SkipConfig, beadsID, workspace, skill string) {
	logger := events.NewLogger(events.DefaultLogPath())
	for _, gate := range skipConfig.skippedGates() {
		event := events.Event{
			Type:      events.EventTypeVerificationBypassed,
			SessionID: beadsID,
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"beads_id":  beadsID,
				"workspace": workspace,
				"gate":      gate,
				"reason":    skipConfig.Reason,
				"skill":     skill,
			},
		}
		recordEpisodicEvent(event, episodic.Context{
			Boundary:  episodic.BoundaryVerification,
			Project:   projectFromCWD(),
			Workspace: workspace,
			BeadsID:   beadsID,
		})

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

// persistGateSkipMemory records skip decisions for the given gates.
// This is the single implementation for gate skip memory persistence,
// used by both recordGateSkipMemory (skip-flag path) and applySkipFiltering (pipeline path).
func persistGateSkipMemory(gates []string, reason, projectDir, identifier string) {
	for _, gate := range gates {
		if err := verify.RecordGateSkip(projectDir, gate, reason, identifier); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to persist gate skip memory for %s: %v\n", gate, err)
		} else {
			fmt.Printf("Gate skip memory saved for %s (expires in %v)\n", gate, verify.GateSkipDuration)
		}
	}
}

// recordGateSkipMemory persists gate skip decisions for future completions.
// When the orchestrator uses --skip-* flags, this records each skip reason so
// subsequent completions auto-skip those gates without requiring --skip-* again.
func recordGateSkipMemory(skipConfig SkipConfig, projectDir, identifier string) {
	persistGateSkipMemory(skipConfig.skippedGates(), skipConfig.Reason, projectDir, identifier)
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

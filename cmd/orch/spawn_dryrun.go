// Package main provides the spawn dry-run command.
// This file contains:
// - runSpawnDryRun for validating spawn plans without execution
// - formatting helpers for dry-run output
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/skills"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// runSpawnDryRun validates skill loading, context generation, and resolved settings
// without creating beads issues, writing workspace files, or dispatching the spawn.
func runSpawnDryRun(serverURL, skillName, task string) error {
	// Validate --mode flag early
	if err := orch.ValidateMode(orch.Mode); err != nil {
		return err
	}

	// Validate flags
	if spawnVerifyLevel != "" && !spawn.IsValidVerifyLevel(spawnVerifyLevel) {
		return fmt.Errorf("invalid --verify-level %q: must be V0, V1, V2, or V3", spawnVerifyLevel)
	}
	if spawnReviewTier != "" && !spawn.IsValidReviewTier(spawnReviewTier) {
		return fmt.Errorf("invalid --review-tier %q: must be auto, scan, review, or deep", spawnReviewTier)
	}
	if spawnEffort != "" && !spawn.IsValidEffort(spawnEffort) {
		return fmt.Errorf("invalid --effort %q: must be low, medium, or high", spawnEffort)
	}

	// Resolve project directory
	projectDir, projectName, err := orch.ResolveProjectDirectory(spawnWorkdir)
	if err != nil {
		return err
	}

	// Load skill
	loader := skills.DefaultLoader()
	skillContent, err := loader.LoadSkillWithDependencies(skillName)
	if err != nil {
		return fmt.Errorf("skill loading failed: %w", err)
	}
	rawSkillContent, _ := loader.LoadSkillContent(skillName)
	var isOrchestrator, isMetaOrchestrator bool
	if rawSkillContent != "" {
		if metadata, parseErr := skills.ParseSkillMetadata(rawSkillContent); parseErr == nil {
			isOrchestrator = metadata.SkillType == "policy" || metadata.SkillType == "orchestrator"
		}
	}
	isMetaOrchestrator = skillName == "meta-orchestrator"

	// Generate workspace name (for display only, not created)
	workspaceName := spawn.GenerateWorkspaceName(projectName, skillName, task, spawn.WorkspaceNameOptions{
		IsMetaOrchestrator: isMetaOrchestrator,
		IsOrchestrator:     isOrchestrator,
	})

	// Apply section filtering
	skillContent = skills.FilterSkillSections(skillContent, buildSectionFilter(spawnPhases, orch.Mode))

	// Resolve spawn settings
	projectCfg, projectMeta, err := config.LoadWithMeta(projectDir)
	if err != nil {
		projectCfg = nil
		projectMeta = nil
	}
	userCfg, userMeta, warning := loadUserConfigWithMetaAndWarning()
	if warning != "" {
		fmt.Fprint(os.Stderr, warning)
		userCfg = nil
		userMeta = nil
	}

	beadsLabels := loadBeadsLabels(spawnIssue, projectDir)
	resolveInput := spawn.ResolveInput{
		CLI: spawn.CLISettings{
			Backend:       spawnBackendFlag,
			Model:         spawnModel,
			Mode:          orch.Mode,
			ModeSet:       spawnModeSet,
			Validation:    spawnValidation,
			ValidationSet: spawnValidationSet,
			MCP:           spawnMCP,
			Account:       spawnAccount,
			Effort:        spawnEffort,
			Light:         spawnLight,
			Full:          spawnFull,
			Headless:      spawnHeadless,
			Tmux:          spawnTmux,
		},
		BeadsLabels:            beadsLabels,
		ProjectConfig:          projectCfg,
		ProjectConfigMeta:      projectMetaFromConfig(projectMeta),
		UserConfig:             userCfg,
		UserConfigMeta:         userMetaFromConfig(userMeta),
		Task:                   task,
		BeadsID:                spawnIssue,
		SkillName:              skillName,
		IsOrchestrator:         isOrchestrator,
		InfrastructureDetected: orch.IsInfrastructureWork(task, spawnIssue),
		CapacityFetcher:        buildCapacityFetcher(),
	}
	resolved, err := orch.ResolveSpawnSettings(resolveInput)
	if err != nil {
		return err
	}

	// Gather KB context (read-only, no side effects)
	kbContext, gapAnalysis, _, _, _, err := orch.GatherSpawnContext(skillContent, task, spawnOrientationFrame, spawnIssue, projectDir, workspaceName, skillName, spawnSkipArtifactCheck, spawnGateOnGap, spawnSkipGapGate, spawnGapThreshold)
	if err != nil {
		return err
	}

	// Estimate token size
	cfg := orch.BuildSpawnConfig(&orch.SpawnContext{
		Task:               task,
		SkillName:          skillName,
		ProjectDir:         projectDir,
		ProjectName:        projectName,
		WorkspaceName:      workspaceName,
		SkillContent:       skillContent,
		BeadsID:            spawnIssue,
		IsOrchestrator:     isOrchestrator,
		IsMetaOrchestrator: isMetaOrchestrator,
		ResolvedModel:      resolved.Model,
		ResolvedSettings:   resolved.Settings,
		KBContext:          kbContext,
		GapAnalysis:        gapAnalysis,
		Tier:               resolved.Settings.Tier.Value,
	}, spawnPhases, resolved.Settings.Mode.Value, resolved.Settings.Validation.Value, resolved.Settings.MCP.Value, resolved.Settings.BrowserTool.Value, spawnNoTrack, spawnSkipArtifactCheck, spawnReason)

	tokenEstimate := spawn.EstimateContextTokens(cfg)

	// Print spawn plan
	fmt.Println("=== SPAWN PLAN (dry-run) ===")
	fmt.Println()
	fmt.Printf("  Task:       %s\n", task)
	fmt.Printf("  Skill:      %s\n", skillName)
	fmt.Printf("  Project:    %s (%s)\n", projectName, projectDir)
	fmt.Printf("  Workspace:  %s\n", workspaceName)
	fmt.Println()
	fmt.Println("--- Resolved Settings ---")
	printSetting("Backend", resolved.Settings.Backend)
	printSetting("Model", resolved.Settings.Model)
	printSetting("Tier", resolved.Settings.Tier)
	printSetting("Spawn Mode", resolved.Settings.SpawnMode)
	printSetting("Mode", resolved.Settings.Mode)
	printSetting("Validation", resolved.Settings.Validation)
	printSetting("Account", resolved.Settings.Account)
	printSetting("Effort", resolved.Settings.Effort)
	if resolved.Settings.MCP.Value != "" {
		printSetting("MCP", resolved.Settings.MCP)
	}
	if resolved.Settings.BrowserTool.Value != "" {
		printSetting("Browser", resolved.Settings.BrowserTool)
	}
	fmt.Println()
	fmt.Println("--- Context ---")
	fmt.Printf("  Skill content:  %d chars\n", len(skillContent))
	fmt.Printf("  KB context:     %d chars\n", len(kbContext))
	fmt.Printf("  Context quality: %s\n", formatContextQualitySummary(gapAnalysis))
	fmt.Printf("  Token estimate: ~%d tokens\n", tokenEstimate.EstimatedTokens)
	if tokenEstimate.ExceedsWarning() {
		fmt.Printf("  ⚠️  Exceeds warning threshold (%d tokens)\n", tokenEstimate.WarningThreshold)
	}
	if tokenEstimate.ExceedsError() {
		fmt.Printf("  🚨 Exceeds error threshold (%d tokens) — spawn would be BLOCKED\n", tokenEstimate.ErrorThreshold)
	}
	fmt.Println()
	fmt.Println("--- Flags ---")
	fmt.Printf("  --no-track:     %v\n", spawnNoTrack)
	fmt.Printf("  --issue:        %s\n", valueOrNone(spawnIssue))
	fmt.Printf("  --phases:       %s\n", valueOrNone(spawnPhases))
	if spawnMaxTurns > 0 {
		fmt.Printf("  --max-turns:    %d\n", spawnMaxTurns)
	}

	// Show routing impact when provider-driven routing was applied
	routingImpact := spawn.BuildRoutingImpact(resolved.Settings)
	if routingImpact.Triggered {
		fmt.Println()
		fmt.Println("--- Routing Impact ---")
		fmt.Printf("  Trigger:          %s\n", routingImpact.Trigger)
		if routingImpact.PreviousBackend != "" {
			fmt.Printf("  Previous backend: %s\n", routingImpact.PreviousBackend)
		}
		fmt.Printf("  Resolved backend: %s\n", routingImpact.ResolvedBackend)
		if routingImpact.PreviousModel != "" {
			fmt.Printf("  Previous model:   %s\n", routingImpact.PreviousModel)
		}
		fmt.Printf("  Resolved model:   %s\n", routingImpact.ResolvedModel)
		if routingImpact.Provider != "" {
			fmt.Printf("  Provider:         %s\n", routingImpact.Provider)
		}
		fmt.Printf("  Automatic:        %v\n", routingImpact.Automatic)
		fmt.Printf("  Explanation:      %s\n", routingImpact.Explanation)
	}

	if len(resolved.Settings.Warnings) > 0 {
		fmt.Println()
		fmt.Println("--- Warnings ---")
		for _, w := range resolved.Settings.Warnings {
			fmt.Printf("  ⚠️  %s\n", w)
		}
	}

	fmt.Println()
	fmt.Println("Dry run complete. No spawn executed, no issues created, no files written.")
	return nil
}

// printSetting formats a resolved setting with its source for dry-run output.
func printSetting(label string, s spawn.ResolvedSetting) {
	source := string(s.Source)
	if s.Detail != "" {
		source += " (" + s.Detail + ")"
	}
	fmt.Printf("  %-14s %s (source: %s)\n", label+":", s.Value, source)
}

// valueOrNone returns the value or "(none)" if empty.
func valueOrNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// formatContextQualitySummary formats a gap analysis into a human-readable summary.
func formatContextQualitySummary(gapAnalysis *spawn.GapAnalysis) string {
	if gapAnalysis == nil {
		return "not checked"
	}
	quality := gapAnalysis.ContextQuality
	var indicator, label string
	switch {
	case quality == 0:
		indicator = "🚨"
		label = "CRITICAL - No context"
	case quality < 20:
		indicator = "⚠️"
		label = "poor"
	case quality < 40:
		indicator = "⚠️"
		label = "limited"
	case quality < 60:
		indicator = "📊"
		label = "moderate"
	case quality < 80:
		indicator = "✓"
		label = "good"
	default:
		indicator = "✓"
		label = "excellent"
	}
	summary := fmt.Sprintf("%s %d/100 (%s)", indicator, quality, label)
	if gapAnalysis.MatchStats.TotalMatches > 0 {
		summary += fmt.Sprintf(" - %d matches", gapAnalysis.MatchStats.TotalMatches)
		if gapAnalysis.MatchStats.ConstraintCount > 0 {
			summary += fmt.Sprintf(" (%d constraints)", gapAnalysis.MatchStats.ConstraintCount)
		}
	}
	return summary
}

// buildSectionFilter creates a SectionFilter from spawn flags.
// Returns nil when no filtering is needed (backward compatible).
func buildSectionFilter(phases, mode string) *skills.SectionFilter {
	var phasesList []string
	if phases != "" {
		for _, p := range strings.Split(phases, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				phasesList = append(phasesList, p)
			}
		}
	}

	filter := &skills.SectionFilter{
		Phases: phasesList,
		Mode:   mode,
	}
	if filter.IsEmpty() {
		return nil
	}
	return filter
}

// Package main provides shared helper functions for spawn and work commands.
// This file contains:
// - config loading and conversion helpers
// - spawn mode application
// - scaffolding management
// - misc utility functions used across spawn files
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/display"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

func projectMetaFromConfig(meta *config.ConfigMeta) spawn.ProjectConfigMeta {
	if meta == nil {
		return spawn.ProjectConfigMeta{}
	}
	return spawn.ProjectConfigMeta{
		SpawnMode:     meta.Explicit["spawn_mode"],
		ClaudeModel:   meta.ExplicitClaude["model"],
		OpenCodeModel: meta.ExplicitOpenCode["model"],
		Models:        meta.Explicit["models"],
	}
}

func userMetaFromConfig(meta *userconfig.ConfigMeta) spawn.UserConfigMeta {
	if meta == nil {
		return spawn.UserConfigMeta{}
	}
	return spawn.UserConfigMeta{
		Backend:                meta.Explicit["backend"],
		DefaultModel:           meta.Explicit["default_model"],
		DefaultTier:            meta.Explicit["default_tier"],
		Models:                 meta.Explicit["models"],
		AllowAnthropicOpenCode: meta.Explicit["allow_anthropic_opencode"],
	}
}

func formatUserConfigLoadWarning(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf(
		"Warning: failed to load user config %s: %v\n"+
			"         Using defaults; user config preferences (backend/default_model) will be ignored.\n",
		userconfig.ConfigPath(),
		err,
	)
}

func loadUserConfigAndWarning() (*userconfig.Config, string) {
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, formatUserConfigLoadWarning(err)
	}
	return cfg, ""
}

func loadUserConfigWithMetaAndWarning() (*userconfig.Config, *userconfig.ConfigMeta, string) {
	cfg, meta, err := userconfig.LoadWithMeta()
	if err != nil {
		return nil, nil, formatUserConfigLoadWarning(err)
	}
	return cfg, meta, ""
}

func applyResolvedSpawnMode(input *orch.SpawnInput, spawnMode string) {
	if input == nil || input.Attach {
		return
	}
	switch spawnMode {
	case spawn.SpawnModeInline:
		input.Inline = true
		input.Headless = false
		input.Tmux = false
	case spawn.SpawnModeTmux:
		input.Tmux = true
		input.Inline = false
		input.Headless = false
	case spawn.SpawnModeHeadless:
		input.Headless = true
		input.Inline = false
		input.Tmux = false
	}
}

// hotspotFilesFromResult extracts all matched file paths from a hotspot result.
// Returns nil if result is nil or has no matched hotspots.
func hotspotFilesFromResult(result *gates.HotspotResult) []string {
	if result == nil {
		return nil
	}
	return result.MatchedFiles
}

// dirExists returns true if the path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ensureOrchScaffolding checks for required scaffolding (.orch, .beads) and optionally auto-initializes.
// Returns nil if scaffolding exists or was successfully created.
// Returns an error with guidance if scaffolding is missing and auto-init is not enabled.
func ensureOrchScaffolding(projectDir string, autoInit bool, noTrack bool) error {
	beadsDir := filepath.Join(projectDir, ".beads")
	beadsExists := dirExists(beadsDir)

	// If beads exists or tracking is disabled, we're good
	if beadsExists || noTrack {
		return nil
	}

	// Beads is missing and tracking is enabled
	// If auto-init is enabled, run initialization
	if autoInit {
		fmt.Println("Auto-initializing orch scaffolding...")

		// Run init with appropriate flags (skip CLAUDE.md and tmuxinator for minimal init)
		result, err := initProject(projectDir, false, false, false, true, true, "", "")
		if err != nil {
			return fmt.Errorf("auto-init failed: %w", err)
		}

		// Print minimal summary
		if len(result.DirsCreated) > 0 {
			fmt.Printf("Created: %s\n", strings.Join(result.DirsCreated, ", "))
		}
		if result.BeadsInitiated {
			fmt.Println("Beads initialized (.beads/)")
		}
		if result.KBInitiated {
			fmt.Println("KB initialized (.kb/)")
		}

		return nil
	}

	// Not auto-init, provide helpful error message
	return fmt.Errorf("missing beads tracking (.beads/ not initialized)\n\nTo fix, run one of:\n  orch init           # Full initialization (recommended)\n  orch spawn --auto-init ...  # Auto-init during spawn")
}

// Test-only wrapper functions that call pkg/orch versions.
// These exist only to support existing tests.

func formatSessionTitle(workspaceName, beadsID string) string {
	if beadsID == "" {
		return workspaceName
	}
	return fmt.Sprintf("%s [%s]", workspaceName, beadsID)
}

func stripANSI(s string) string { return display.StripANSI(s) }

func validateModeModelCombo(backend string, resolvedModel model.ModelSpec) error {
	if backend == "opencode" && strings.Contains(strings.ToLower(resolvedModel.ModelID), "opus") {
		return fmt.Errorf(`Warning: opencode backend with opus model may fail (auth blocked).
  Recommendation: Use --model sonnet (default) or let auto-selection use claude backend`)
	}
	return nil
}

func determineBeadsID(projectName, skillName, task, spawnIssueFlag string, noTrack bool, createBeadsFn func(string, string, string, string) (string, error), dir string) (string, error) {
	if spawnIssueFlag != "" {
		return resolveShortBeadsID(spawnIssueFlag)
	}
	if noTrack {
		// Create a real beads issue with tier:lightweight label instead of synthetic ID.
		beadsID, err := createBeadsFn(projectName, skillName, task, dir)
		if err != nil {
			return "", fmt.Errorf("failed to create lightweight beads issue: %w", err)
		}
		return beadsID, nil
	}
	beadsID, err := createBeadsFn(projectName, skillName, task, dir)
	if err != nil {
		return "", fmt.Errorf("failed to create beads issue: %w", err)
	}
	return beadsID, nil
}

func addUsageInfoToEventData(eventData map[string]interface{}, usageInfo *spawn.UsageInfo) {
	if usageInfo == nil {
		return
	}
	eventData["usage_5h_used"] = usageInfo.FiveHourUsed
	eventData["usage_weekly_used"] = usageInfo.SevenDayUsed
	if usageInfo.AccountEmail != "" {
		eventData["usage_account"] = usageInfo.AccountEmail
	}
	if usageInfo.AutoSwitched {
		eventData["usage_auto_switched"] = true
		eventData["usage_switch_reason"] = usageInfo.SwitchReason
	}
}


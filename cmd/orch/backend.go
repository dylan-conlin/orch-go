package main

import (
	"fmt"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

// BackendResolution contains the resolved backend and any advisory warnings.
type BackendResolution struct {
	Backend  string   // "claude", "opencode", or "docker"
	Warnings []string // Advisory messages (infrastructure, compatibility, etc.)
	Reason   string   // Why this backend was selected (for logging/debugging)
}

// resolveBackend determines which backend to use for spawning.
// Priority: 1) explicit flags, 2) project config, 3) global config, 4) default opencode
//
// This function consolidates all backend selection logic into a single place with
// a clear, documented priority chain. Infrastructure detection is advisory-only.
func resolveBackend(
	backendFlag string, // --backend flag value ("claude", "opencode", or "")
	opusFlag bool, // --opus flag
	modelFlag string, // --model flag (for compatibility warnings)
	projCfg *config.Config, // .orch/config.yaml in project
	globalCfg *userconfig.Config, // ~/.orch/config.yaml
	task string, // Task description (for infrastructure detection)
	beadsID string, // Beads issue ID (for infrastructure detection)
) BackendResolution {
	var result BackendResolution

	// Priority 1: Explicit --backend flag (highest priority, user knows what they want)
	if backendFlag != "" {
		if backendFlag != "claude" && backendFlag != "opencode" && backendFlag != "docker" {
			// Invalid value - warn and fall through to other resolution
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Invalid --backend value %q ignored (must be 'claude', 'opencode', or 'docker')", backendFlag))
		} else {
			result.Backend = backendFlag
			result.Reason = fmt.Sprintf("--backend %s flag", backendFlag)
			return addInfrastructureWarning(result, task, beadsID)
		}
	}

	// Priority 2: Explicit --opus flag (implies claude backend)
	if opusFlag {
		result.Backend = "claude"
		result.Reason = "--opus flag (implies claude backend)"
		return addInfrastructureWarning(result, task, beadsID)
	}

	// Priority 3: Project config (.orch/config.yaml in project directory)
	if projCfg != nil && projCfg.SpawnMode != "" {
		if projCfg.SpawnMode == "claude" || projCfg.SpawnMode == "opencode" {
			result.Backend = projCfg.SpawnMode
			result.Reason = fmt.Sprintf("project config (spawn_mode: %s)", projCfg.SpawnMode)
			return addInfrastructureWarning(result, task, beadsID)
		}
		// Invalid config value - warn and fall through
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Invalid project spawn_mode %q ignored", projCfg.SpawnMode))
	}

	// Priority 4: Global config (~/.orch/config.yaml)
	// Uses the existing "backend" field which defaults to "opencode"
	if globalCfg != nil && globalCfg.Backend != "" {
		if globalCfg.Backend == "claude" || globalCfg.Backend == "opencode" {
			result.Backend = globalCfg.Backend
			result.Reason = fmt.Sprintf("global config (backend: %s)", globalCfg.Backend)
			return addInfrastructureWarning(result, task, beadsID)
		}
		// Invalid config value - warn and fall through
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Invalid global backend %q ignored", globalCfg.Backend))
	}

	// Priority 5: Default to opencode (cost optimization)
	result.Backend = "opencode"
	result.Reason = "default (opencode for cost optimization)"
	return addInfrastructureWarning(result, task, beadsID)
}

// addInfrastructureWarning checks for critical infrastructure work and adds advisory warning.
// NEVER overrides the backend - warnings only.
func addInfrastructureWarning(result BackendResolution, task, beadsID string) BackendResolution {
	if !isCriticalInfrastructureWork(task, beadsID) {
		return result
	}

	if result.Backend == "opencode" {
		result.Warnings = append(result.Warnings,
			"",
			"  Critical infrastructure work detected (may restart OpenCode server)",
			"  Agent may die if server restarts. Consider: --backend claude --tmux",
			"")
	}
	return result
}

// validateBackendModelCompatibility checks for known-bad combinations.
// Returns warning message if there's an issue, empty string if OK.
func validateBackendModelCompatibility(backend, modelFlag string) string {
	if backend == "opencode" && strings.Contains(strings.ToLower(modelFlag), "opus") {
		return "  opus model with opencode backend may fail (auth issues). Consider: --backend claude"
	}
	return ""
}

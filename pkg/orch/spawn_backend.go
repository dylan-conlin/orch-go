package orch

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/model"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
)

func DetermineSpawnBackend(resolvedModel model.ModelSpec, task, beadsID, projectDir, backendFlag, spawnModel string) (string, error) {
	projCfg, projMeta, _ := config.LoadWithMeta(projectDir)
	projectSpawnModeExplicit := projMeta != nil && projMeta.Explicit["spawn_mode"]
	userCfg, userMeta, _ := userconfig.LoadWithMeta()
	userCfgExplicit := userMeta != nil && userMeta.Explicit["backend"] && userCfg != nil && userCfg.Backend != ""
	userDefaultModelExplicit := userMeta != nil && userMeta.Explicit["default_model"] && userCfg != nil && userCfg.DefaultModel != ""
	backend := "opencode"
	explicitBackend := backendFlag != ""
	explicitModel := spawnModel != "" || userDefaultModelExplicit
	if explicitBackend {
		backend = backendFlag
		if backend != "claude" && backend != "opencode" {
			return "", fmt.Errorf("invalid --backend value: %s (must be 'claude' or 'opencode')", backend)
		}
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting explicit --backend %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if explicitModel {
		modelName := spawnModel
		if modelName == "" && userDefaultModelExplicit {
			modelName = userCfg.DefaultModel
		}
		if modelName == "" {
			modelName = resolvedModel.Format()
		}
		if projCfg != nil && projectSpawnModeExplicit && projCfg.SpawnMode != "" {
			backend = projCfg.SpawnMode
		} else if userCfgExplicit {
			backend = userCfg.Backend
		}
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting explicit model %s (backend: %s)\n", modelName, backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if projCfg != nil && projectSpawnModeExplicit && projCfg.SpawnMode != "" {
		backend = projCfg.SpawnMode
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting project spawn_mode %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if userCfgExplicit {
		backend = userCfg.Backend
		if isInfrastructureWork(task, beadsID) && backend != "claude" {
			fmt.Fprintf(os.Stderr, "⚠️  Infrastructure work detected but respecting user config backend %s\n", backend)
			fmt.Fprintf(os.Stderr, "   Recommendation: Use --backend claude for infrastructure work to survive server restarts.\n")
		}
	} else if isInfrastructureWork(task, beadsID) {
		backend = "claude"
		fmt.Println("🔧 Infrastructure work detected - auto-applying escape hatch (--backend claude --tmux)")
		fmt.Println("   This ensures the agent survives OpenCode server restarts.")
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type: "spawn.infrastructure_detected", Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{"task": task, "beads_id": beadsID, "skill": ""},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log infrastructure detection: %v\n", err)
		}
	}
	if err := validateModeModelCombo(backend, resolvedModel); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
	}
	return backend, nil
}

func isInfrastructureWork(task string, beadsID string) bool {
	infrastructureKeywords := []string{
		"opencode", "orch-go", "pkg/spawn", "pkg/opencode", "pkg/verify", "pkg/state",
		"cmd/orch", "spawn_cmd.go", "serve.go", "status.go", "main.go", "dashboard",
		"agent-card", "agents.ts", "daemon.ts", "skillc", "skill.yaml", "SPAWN_CONTEXT",
		"spawn system", "spawn logic", "spawn template", "orchestration infrastructure", "orchestration system",
	}
	taskLower := strings.ToLower(task)
	for _, keyword := range infrastructureKeywords {
		if strings.Contains(taskLower, keyword) {
			return true
		}
	}
	if beadsID != "" {
		issue, err := verify.GetIssue(beadsID)
		if err == nil {
			titleLower := strings.ToLower(issue.Title)
			for _, keyword := range infrastructureKeywords {
				if strings.Contains(titleLower, keyword) {
					return true
				}
			}
			descLower := strings.ToLower(issue.Description)
			for _, keyword := range infrastructureKeywords {
				if strings.Contains(descLower, keyword) {
					return true
				}
			}
		}
	}
	return false
}

func IsInfrastructureWork(task string, beadsID string) bool {
	return isInfrastructureWork(task, beadsID)
}

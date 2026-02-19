// Package main provides the rework command for spawning a rework agent.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/orch"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/spawn/gates"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	reworkModel        string
	reworkSkill        string
	reworkTmux         bool
	reworkBypassTriage bool
	reworkForce        bool
	reworkWorkdir      string
)

var reworkCmd = &cobra.Command{
	Use:   "rework [beads-id] [feedback]",
	Short: "Spawn a rework agent for a completed issue",
	Long: `Spawn a new agent with structured rework context from a prior attempt.

This command reopens the original beads issue, records a REWORK comment,
and spawns a fresh workspace with rework context embedded in SPAWN_CONTEXT.md.

Manual rework requires --bypass-triage (consistent with manual spawn friction).

Examples:
  orch-go rework orch-go-123 "Missing tests and docs" --bypass-triage
  orch-go rework orch-go-123 "Fix sorting output" --bypass-triage --tmux
  orch-go rework orch-go-123 "Use Opus for deeper analysis" --bypass-triage --model opus`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		beadsID := args[0]
		feedback := strings.Join(args[1:], " ")
		return runRework(beadsID, feedback)
	},
}

func init() {
	reworkCmd.Flags().StringVar(&reworkModel, "model", "", "Override model for rework agent")
	reworkCmd.Flags().StringVar(&reworkSkill, "skill", "", "Override skill for rework agent")
	reworkCmd.Flags().BoolVar(&reworkTmux, "tmux", false, "Run in tmux window for visual monitoring")
	reworkCmd.Flags().BoolVar(&reworkBypassTriage, "bypass-triage", false, "Acknowledge manual rework bypasses daemon-driven triage workflow")
	reworkCmd.Flags().BoolVar(&reworkForce, "force", false, "Override safety checks (e.g., issue not closed)")
	reworkCmd.Flags().StringVar(&reworkWorkdir, "workdir", "", "Target project directory (for cross-project rework)")
	orch.RegisterModeFlag(reworkCmd)
}

func runRework(beadsID, feedback string) error {
	if err := orch.ValidateMode(orch.Mode); err != nil {
		return err
	}
	if strings.TrimSpace(feedback) == "" {
		return fmt.Errorf("feedback message is required")
	}
	if !reworkBypassTriage {
		return fmt.Errorf("manual rework requires --bypass-triage")
	}

	projectDir, projectName, err := orch.ResolveProjectDirectory(reworkWorkdir)
	if err != nil {
		return err
	}
	beads.DefaultDir = projectDir

	issue, err := verify.GetIssue(beadsID)
	if err != nil {
		projectHint := formatProjectMismatchHint(projectDir, beadsID)
		if projectHint != "" {
			return fmt.Errorf("failed to get beads issue %s: %w\n\n%s", beadsID, err, projectHint)
		}
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	if issue.Status != "closed" && !reworkForce {
		return fmt.Errorf("issue %s is %s (rework requires closed issue). Use --force to override", beadsID, issue.Status)
	}

	task := issue.Title
	if issue.Description != "" {
		task = issue.Title + "\n\n" + issue.Description
	}

	priorWorkspace, err := spawn.FindArchivedWorkspaceByBeadsID(projectDir, beadsID)
	if err != nil {
		if wsPath, _ := findWorkspaceByBeadsID(projectDir, beadsID); wsPath != "" {
			if !reworkForce {
				return fmt.Errorf("prior workspace not archived for %s. Run orch complete or use --force to rework from active workspace", beadsID)
			}
			priorWorkspace = wsPath
		} else {
			return err
		}
	}

	manifest := spawn.ReadAgentManifestWithFallback(priorWorkspace)

	skillName := strings.TrimSpace(reworkSkill)
	if skillName == "" {
		skillName = strings.TrimSpace(manifest.Skill)
	}
	if skillName == "" {
		inferred, err := orch.InferSkillFromIssueType(issue.IssueType)
		if err != nil {
			return fmt.Errorf("could not infer skill from issue type: %w", err)
		}
		skillName = inferred
	}

	input := &orch.SpawnInput{
		ServerURL:    serverURL,
		SkillName:    skillName,
		Task:         task,
		Inline:       false,
		Headless:     false,
		Tmux:         reworkTmux,
		Attach:       false,
		DaemonDriven: false,
	}

	hotspotCheckFunc := func(dir, t string) (*gates.HotspotResult, error) {
		result, err := RunHotspotCheckForSpawn(dir, t)
		if err != nil || result == nil {
			return nil, err
		}
		return &gates.HotspotResult{HasHotspots: result.HasHotspots, Warning: result.Warning}, nil
	}

	usageCheckResult, err := orch.RunPreFlightChecks(input, projectDir, reworkBypassTriage, false, "", 0, extractBeadsIDFromTitle, hotspotCheckFunc)
	if err != nil {
		return err
	}

	skillContent, workspaceName, isOrchestrator, isMetaOrchestrator, err := orch.LoadSkillAndGenerateWorkspace(skillName, projectName, task, projectDir, false, false, ensureOrchScaffolding)
	if err != nil {
		return err
	}
	if isOrchestrator || isMetaOrchestrator {
		return fmt.Errorf("orch rework only supports worker skills")
	}

	modelFlag := strings.TrimSpace(reworkModel)
	if modelFlag == "" {
		modelFlag = strings.TrimSpace(manifest.Model)
	}
	projectCfg, projectMeta, err := config.LoadWithMeta(projectDir)
	if err != nil {
		projectCfg = nil
		projectMeta = nil
	}
	userCfg, userMeta, err := userconfig.LoadWithMeta()
	if err != nil {
		userCfg = nil
		userMeta = nil
	}
	beadsLabels := loadBeadsLabels(beadsID)
	manifestTier := strings.TrimSpace(manifest.Tier)
	resolveInput := spawn.ResolveInput{
		CLI: spawn.CLISettings{
			Backend:       "",
			Model:         modelFlag,
			Mode:          orch.Mode,
			ModeSet:       false,
			Validation:    "tests",
			ValidationSet: false,
			MCP:           "",
			Light:         strings.EqualFold(manifestTier, spawn.TierLight),
			Full:          strings.EqualFold(manifestTier, spawn.TierFull),
			Headless:      false,
			Tmux:          reworkTmux,
			Inline:        false,
		},
		BeadsLabels:            beadsLabels,
		ProjectConfig:          projectCfg,
		ProjectConfigMeta:      projectMetaFromConfig(projectMeta),
		UserConfig:             userCfg,
		UserConfigMeta:         userMetaFromConfig(userMeta),
		Task:                   task,
		BeadsID:                beadsID,
		SkillName:              skillName,
		IsOrchestrator:         false,
		InfrastructureDetected: orch.IsInfrastructureWork(task, beadsID),
	}
	resolved, err := orch.ResolveSpawnSettings(resolveInput)
	if err != nil {
		return err
	}
	applyResolvedSpawnMode(input, resolved.Settings.SpawnMode.Value)

	kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, err := orch.GatherSpawnContext(skillContent, task, beadsID, projectDir, workspaceName, skillName, false, false, false, 0)
	if err != nil {
		return err
	}

	isBug, reproSteps := orch.ExtractBugReproInfo(beadsID, false)
	usageInfo := orch.BuildUsageInfo(usageCheckResult)

	reworkCount, err := spawn.CountReworks(beadsID, projectDir)
	if err != nil {
		return fmt.Errorf("failed to count rework attempts: %w", err)
	}
	reworkNumber := reworkCount + 1

	priorSynthesis := ""
	synthesisPath := filepath.Join(priorWorkspace, "SYNTHESIS.md")
	if summary, err := spawn.ExtractReworkSummary(synthesisPath); err == nil {
		priorSynthesis = summary
	} else {
		priorSynthesis = fmt.Sprintf("No prior SYNTHESIS.md summary available (%v)", err)
	}

	if err := verify.UpdateIssueStatus(beadsID, "open"); err != nil {
		return fmt.Errorf("failed to reopen issue: %w", err)
	}

	reworkComment := fmt.Sprintf("REWORK #%d: %s", reworkNumber, feedback)
	if err := addReworkComment(beadsID, reworkComment, projectDir); err != nil {
		return fmt.Errorf("failed to add rework comment: %w", err)
	}

	if err := verify.AddLabel(beadsID, fmt.Sprintf("rework:%d", reworkNumber)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add rework label: %v\n", err)
	}

	if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to set issue status to in_progress: %v\n", err)
	}

	ctx := &orch.SpawnContext{
		Task:              task,
		OrientationFrame:  "",
		SkillName:         skillName,
		ProjectDir:        projectDir,
		ProjectName:       projectName,
		WorkspaceName:     workspaceName,
		SkillContent:      skillContent,
		BeadsID:           beadsID,
		ResolvedModel:     resolved.Model,
		KBContext:         kbContext,
		GapAnalysis:       gapAnalysis,
		HasInjectedModels: hasInjectedModels,
		PrimaryModelPath:  primaryModelPath,
		IsBug:             isBug,
		ReproSteps:        reproSteps,
		ReworkFeedback:    feedback,
		ReworkNumber:      reworkNumber,
		PriorSynthesis:    priorSynthesis,
		PriorWorkspace:    priorWorkspace,
		UsageInfo:         usageInfo,
		SpawnBackend:      resolved.Settings.Backend.Value,
		Tier:              resolved.Settings.Tier.Value,
	}

	cfg := orch.BuildSpawnConfig(ctx, "", resolved.Settings.Mode.Value, resolved.Settings.Validation.Value, resolved.Settings.MCP.Value, false, false)
	minimalPrompt, err := orch.ValidateAndWriteContext(cfg, false)
	if err != nil {
		return err
	}

	if err := orch.DispatchSpawn(input, cfg, minimalPrompt, beadsID, skillName, task, serverURL); err != nil {
		return err
	}

	logger := events.NewLogger(events.DefaultLogPath())
	logErr := logger.LogAgentReworked(events.AgentReworkedData{
		BeadsID:        beadsID,
		PriorWorkspace: priorWorkspace,
		NewWorkspace:   cfg.WorkspacePath(),
		ReworkNumber:   reworkNumber,
		Feedback:       feedback,
		Skill:          skillName,
		Model:          resolved.Model.Format(),
	})
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log rework event: %v\n", logErr)
	}

	return nil
}

func addReworkComment(beadsID, comment, projectDir string) error {
	socketPath, err := beads.FindSocketPath(projectDir)
	if err == nil {
		opts := []beads.Option{beads.WithAutoReconnect(3)}
		if projectDir != "" {
			opts = append(opts, beads.WithCwd(projectDir))
		}
		client := beads.NewClient(socketPath, opts...)
		if err := client.Connect(); err == nil {
			defer client.Close()
			if err := client.AddComment(beadsID, "orchestrator", comment); err == nil {
				return nil
			}
		}
	}

	return beads.FallbackAddComment(beadsID, comment)
}

func formatProjectMismatchHint(projectDir, beadsID string) string {
	projectName := filepath.Base(projectDir)
	issuePrefix := strings.Split(beadsID, "-")[0]
	if len(strings.Split(beadsID, "-")) > 1 {
		issuePrefix = strings.Join(strings.Split(beadsID, "-")[:len(strings.Split(beadsID, "-"))-1], "-")
	}
	if issuePrefix != projectName {
		return fmt.Sprintf("Hint: The issue ID suggests it belongs to project '%s', but you're in '%s'.\nTry: orch rework %s --workdir ~/path/to/%s", issuePrefix, projectName, beadsID, issuePrefix)
	}
	return ""
}

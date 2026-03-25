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

// ReworkParams holds parameters for programmatic rework invocation.
// Used by the loop controller to call rework without relying on cobra globals.
type ReworkParams struct {
	BeadsID       string
	Feedback      string
	Model         string
	Skill         string
	Tmux          bool
	BypassTriage  bool
	Force         bool
	Workdir       string
	ServerURL     string
}

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
	params := ReworkParams{
		BeadsID:      beadsID,
		Feedback:     feedback,
		Model:        reworkModel,
		Skill:        reworkSkill,
		Tmux:         reworkTmux,
		BypassTriage: reworkBypassTriage,
		Force:        reworkForce,
		Workdir:      reworkWorkdir,
		ServerURL:    serverURL,
	}
	return runReworkWithParams(params)
}

// runReworkWithParams is the core rework implementation that accepts a struct parameter.
// This enables programmatic invocation from the loop controller.
func runReworkWithParams(params ReworkParams) error {
	beadsID := params.BeadsID
	feedback := params.Feedback

	if err := orch.ValidateMode(orch.Mode); err != nil {
		return err
	}
	if strings.TrimSpace(feedback) == "" {
		return fmt.Errorf("feedback message is required")
	}
	if !params.BypassTriage {
		return fmt.Errorf("manual rework requires --bypass-triage")
	}

	projectDir, projectName, err := orch.ResolveProjectDirectory(params.Workdir)
	if err != nil {
		return err
	}
	issue, err := verify.GetIssue(beadsID, projectDir)
	if err != nil {
		projectHint := formatProjectMismatchHint(projectDir, beadsID)
		if projectHint != "" {
			return fmt.Errorf("failed to get beads issue %s: %w\n\n%s", beadsID, err, projectHint)
		}
		return fmt.Errorf("failed to get beads issue: %w", err)
	}

	if issue.Status != "closed" && !params.Force {
		return fmt.Errorf("issue %s is %s (rework requires closed issue). Use --force to override", beadsID, issue.Status)
	}

	task := issue.Title
	reworkOrientationFrame := issue.Description
	if issue.Description != "" {
		task = issue.Title + "\n\n" + issue.Description
	}

	priorWorkspace, err := spawn.FindArchivedWorkspaceByBeadsID(projectDir, beadsID)
	if err != nil {
		if wsPath, _ := findWorkspaceByBeadsID(projectDir, beadsID); wsPath != "" {
			if !params.Force {
				return fmt.Errorf("prior workspace not archived for %s. Run orch complete or use --force to rework from active workspace", beadsID)
			}
			priorWorkspace = wsPath
		} else {
			return err
		}
	}

	manifest := spawn.ReadAgentManifestWithFallback(priorWorkspace)

	skillName := strings.TrimSpace(params.Skill)
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

	srvURL := params.ServerURL
	if srvURL == "" {
		srvURL = serverURL
	}

	input := &orch.SpawnInput{
		ServerURL:    srvURL,
		SkillName:    skillName,
		Task:         task,
		IssueID:      beadsID,
		Inline:       false,
		Headless:     false,
		Tmux:         params.Tmux,
		Attach:       false,
		DaemonDriven: false,
	}

	hotspotCheckFunc := func(dir, t string) (*gates.HotspotResult, error) {
		result, err := RunHotspotCheckForSpawn(dir, t)
		if err != nil || result == nil {
			return nil, err
		}
		var matchedFiles []string
		for _, h := range result.MatchedHotspots {
			matchedFiles = append(matchedFiles, h.Path)
		}
		return &gates.HotspotResult{
			HasHotspots:        result.HasHotspots,
			HasCriticalHotspot: result.HasCriticalHotspot,
			Warning:            result.Warning,
			CriticalFiles:      result.CriticalFiles,
			MatchedFiles:       matchedFiles,
		}, nil
	}

	agreementsCheckFunc := buildAgreementsChecker()
	openQuestionCheckFunc := buildOpenQuestionChecker()
	hotspotResult, _, _, err := orch.RunPreFlightChecks(input, projectDir, params.BypassTriage, "", hotspotCheckFunc, agreementsCheckFunc, openQuestionCheckFunc)
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

	modelFlag := strings.TrimSpace(params.Model)
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
	beadsLabels := loadBeadsLabels(beadsID, projectDir)
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
			Tmux:          params.Tmux,
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
		CapacityFetcher:        buildCapacityFetcher(),
	}
	resolved, err := orch.ResolveSpawnSettings(resolveInput)
	if err != nil {
		return err
	}
	applyResolvedSpawnMode(input, resolved.Settings.SpawnMode.Value)

	kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, _, err := orch.GatherSpawnContext(skillContent, task, reworkOrientationFrame, beadsID, projectDir, workspaceName, skillName, false, false, false, 0)
	if err != nil {
		return err
	}

	isBug, reproSteps := orch.ExtractBugReproInfo(beadsID, false)
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

	// Archive prior workspace before starting new work.
	// Unified lifecycle cleanup discipline: rework is a state transition that must
	// clean prior artifacts. Without this, old workspaces accumulate until orch clean.
	if priorWorkspace != "" {
		isArchived := strings.Contains(priorWorkspace, "/archived/")
		if !isArchived {
			if archivedPath, err := archiveWorkspace(priorWorkspace, projectDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to archive prior workspace: %v\n", err)
			} else {
				fmt.Printf("Archived prior workspace: %s\n", filepath.Base(archivedPath))
				// Update priorWorkspace to archived location for rework context below
				priorWorkspace = archivedPath
			}
		}
	}

	if err := verify.UpdateIssueStatus(beadsID, "open", projectDir); err != nil {
		return fmt.Errorf("failed to reopen issue: %w", err)
	}

	reworkComment := fmt.Sprintf("REWORK #%d: %s", reworkNumber, feedback)
	if err := addReworkComment(beadsID, reworkComment, projectDir); err != nil {
		return fmt.Errorf("failed to add rework comment: %w", err)
	}

	if err := verify.AddLabel(beadsID, fmt.Sprintf("rework:%d", reworkNumber), projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add rework label: %v\n", err)
	}

	if err := verify.UpdateIssueStatus(beadsID, "in_progress", projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to set issue status to in_progress: %v\n", err)
	}

	ctx := &orch.SpawnContext{
		Task:              task,
		SkillName:         skillName,
		ProjectDir:        projectDir,
		ProjectName:       projectName,
		WorkspaceName:     workspaceName,
		SkillContent:      skillContent,
		BeadsID:           beadsID,
		ResolvedModel:     resolved.Model,
		ResolvedSettings:  resolved.Settings,
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
		SpawnBackend:      resolved.Settings.Backend.Value,
		Tier:              resolved.Settings.Tier.Value,
		HotspotArea:          hotspotResult != nil && hotspotResult.HasHotspots,
		HotspotFiles:         hotspotFilesFromResult(hotspotResult),
		HotspotDefectClasses: DefectClassesForHotspots(hotspotFilesFromResult(hotspotResult)),
		OpsecSandbox:         projectCfg != nil && projectCfg.Opsec.Sandbox,
		OpsecPort:            resolveOpsecPort(projectCfg),
	}

	cfg := orch.BuildSpawnConfig(ctx, "", resolved.Settings.Mode.Value, resolved.Settings.Validation.Value, resolved.Settings.MCP.Value, resolved.Settings.BrowserTool.Value, false, false, "")

	// OPSEC gate: verify proxy is running before allowing sandboxed spawns
	if err := spawn.CheckOpsecProxy(cfg.OpsecSandbox, cfg.OpsecPort); err != nil {
		return fmt.Errorf("rework blocked: %w", err)
	}

	minimalPrompt, rollback, err := orch.ValidateAndWriteContext(cfg, false)
	if err != nil {
		return err
	}

	if err := orch.DispatchSpawn(input, cfg, minimalPrompt, beadsID, skillName, task, srvURL); err != nil {
		if rollback != nil {
			rollback()
		}
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

	return beads.FallbackAddComment(beadsID, comment, projectDir)
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

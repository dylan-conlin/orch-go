package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/model"
)

// getGitBaseline returns the current git commit SHA for the project directory.
// Returns empty string if not in a git repository or if git command fails.
func getGitBaseline(projectDir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getGitBranch(projectDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(output))
	if branch == "HEAD" {
		return ""
	}
	return branch
}

// CreateScreenshotsDir creates the screenshots/ subdirectory in a workspace.
// This directory is for agent-produced visual artifacts.
func CreateScreenshotsDir(workspacePath string) error {
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	if err := os.MkdirAll(screenshotsPath, 0755); err != nil {
		return fmt.Errorf("failed to create screenshots directory: %w", err)
	}
	return nil
}

// contextData holds template data for SPAWN_CONTEXT.md.
type contextData struct {
	Task                     string
	BeadsID                  string
	RecentValidatedEpisodes  string
	ProjectDir               string
	WorkspaceName            string
	SkillName                string
	SkillContent             string
	InvestigationSlug        string
	Phases                   string
	Mode                     string
	Validation               string
	InvestigationType        string
	KBContext                string
	Tier                     string
	ServerContext            string
	BloatWarnings            string
	NoTrack                  bool
	IsBug                    bool
	ReproSteps               string
	IsInfrastructureTouching bool
	DesignWorkspace          string
	DesignMockupPath         string
	DesignPromptPath         string
	DesignNotes              string
	IssueComments            []IssueComment
	IsInvestigationSkill     bool
	FailureContext           *FailureContext
	HasInjectedModels        bool
	ModelBehaviorProfile     string
	NeedsCompletionNudge     bool
	ReliablePhaseReporting   bool
	NeedsExplicitGitCommit   bool
}

func buildContextData(cfg *Config) contextData {
	// Generate investigation slug from task.
	slug := generateSlug(cfg.Task, 5)

	// Generate server context if enabled.
	serverContext := cfg.ServerContext
	if cfg.IncludeServers && serverContext == "" {
		serverContext = GenerateServerContext(cfg.ProjectDir)
	}

	// Check for bloated files mentioned in the task.
	bloatWarnings := ""
	if cfg.ProjectDir != "" {
		warnings := CheckBloatedFiles(cfg.Task, cfg.ProjectDir)
		bloatWarnings = GenerateBloatWarningSection(warnings)
	}

	// Detect model presence from explicit flag or formatted KB context content.
	// Some model entries only include references (no injected summary blocks), so we
	// detect both the models section header and injected-claims marker line.
	hasModels := cfg.HasInjectedModels ||
		strings.Contains(cfg.KBContext, "### Models (synthesized understanding)") ||
		strings.Contains(cfg.KBContext, "Your findings should confirm, contradict, or extend the claims above.")

	behaviorProfileName := ""
	behaviorProfile := model.BehaviorProfile{}
	if strings.TrimSpace(cfg.Model) != "" {
		behaviorProfile = model.ResolveBehaviorProfile(cfg.Model)
		behaviorProfileName = behaviorProfile.Name
	}

	return contextData{
		Task:                     cfg.Task,
		BeadsID:                  cfg.BeadsID,
		RecentValidatedEpisodes:  GenerateRecentValidatedEpisodesSection(cfg),
		ProjectDir:               cfg.ProjectDir,
		WorkspaceName:            cfg.WorkspaceName,
		SkillName:                cfg.SkillName,
		SkillContent:             prepareSkillContent(cfg),
		InvestigationSlug:        slug,
		Phases:                   cfg.Phases,
		Mode:                     cfg.Mode,
		Validation:               cfg.Validation,
		InvestigationType:        cfg.InvestigationType,
		KBContext:                cfg.KBContext,
		Tier:                     cfg.Tier,
		ServerContext:            serverContext,
		BloatWarnings:            bloatWarnings,
		NoTrack:                  cfg.NoTrack,
		IsBug:                    cfg.IsBug,
		ReproSteps:               cfg.ReproSteps,
		IsInfrastructureTouching: cfg.IsInfrastructureTouching,
		DesignWorkspace:          cfg.DesignWorkspace,
		DesignMockupPath:         cfg.DesignMockupPath,
		DesignPromptPath:         cfg.DesignPromptPath,
		DesignNotes:              cfg.DesignNotes,
		IssueComments:            cfg.IssueComments,
		IsInvestigationSkill:     IsInvestigationSkill(cfg.SkillName),
		FailureContext:           cfg.FailureContext,
		HasInjectedModels:        hasModels,
		ModelBehaviorProfile:     behaviorProfileName,
		NeedsCompletionNudge:     behaviorProfile.NeedsCompletionNudge,
		ReliablePhaseReporting:   behaviorProfile.ReliablePhaseReporting,
		NeedsExplicitGitCommit:   behaviorProfile.NeedsExplicitGitCommit,
	}
}

// WriteContext writes the SPAWN_CONTEXT.md file to the workspace.
// For orchestrator-type skills (IsOrchestrator=true), it delegates to
// WriteOrchestratorContext which generates ORCHESTRATOR_CONTEXT.md instead.
// For meta-orchestrator skills (IsMetaOrchestrator=true), it delegates to
// WriteMetaOrchestratorContext which generates META_ORCHESTRATOR_CONTEXT.md.
func WriteContext(cfg *Config) error {
	attemptID := strings.TrimSpace(cfg.AttemptID)
	if attemptID == "" {
		var err error
		attemptID, err = GenerateAttemptID()
		if err != nil {
			return fmt.Errorf("failed to generate attempt ID: %w", err)
		}
		cfg.AttemptID = attemptID
	}

	if cfg.IsMetaOrchestrator {
		if err := WriteMetaOrchestratorContext(cfg); err != nil {
			return err
		}
		if err := WriteAttemptID(cfg.WorkspacePath(), attemptID); err != nil {
			return fmt.Errorf("failed to write attempt ID file: %w", err)
		}
		return nil
	}
	if cfg.IsOrchestrator {
		if err := WriteOrchestratorContext(cfg); err != nil {
			return err
		}
		if err := WriteAttemptID(cfg.WorkspacePath(), attemptID); err != nil {
			return fmt.Errorf("failed to write attempt ID file: %w", err)
		}
		return nil
	}

	sourceProjectDir := cfg.ProjectDir
	runtimeDir := cfg.RuntimeDir()
	gitWorktreeDir := sourceProjectDir
	gitBranch := getGitBranch(sourceProjectDir)

	if cfg.SpawnMode == "opencode" {
		worktreeDir, branch, err := CreateWorktree(sourceProjectDir, cfg.WorkspaceName)
		if err != nil {
			return fmt.Errorf("failed to create git worktree: %w", err)
		}
		cfg.CWD = worktreeDir
		runtimeDir = worktreeDir
		gitWorktreeDir = worktreeDir
		gitBranch = branch
	} else {
		if strings.TrimSpace(runtimeDir) == "" {
			runtimeDir = sourceProjectDir
		}
		cfg.CWD = runtimeDir
		gitWorktreeDir = runtimeDir
		if gitBranch == "" {
			gitBranch = getGitBranch(runtimeDir)
		}
	}

	content, err := GenerateContext(cfg)
	if err != nil {
		return err
	}
	content, err = applySpawnContextBudget(cfg, content)
	if err != nil {
		return fmt.Errorf("failed to enforce context budget: %w", err)
	}

	// Ensure SYNTHESIS.md template exists in the project (only for full tier).
	if cfg.Tier != TierLight {
		if err := EnsureSynthesisTemplate(cfg.ProjectDir); err != nil {
			return fmt.Errorf("failed to ensure synthesis template: %w", err)
		}
	}

	workspacePath := cfg.WorkspacePath()
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	if err := CreateScreenshotsDir(workspacePath); err != nil {
		return err
	}

	if err := WriteVerificationSpecSkeleton(cfg); err != nil {
		return fmt.Errorf("failed to write verification spec skeleton: %w", err)
	}

	contextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if err := os.WriteFile(contextPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	if err := WriteTier(workspacePath, cfg.Tier); err != nil {
		return fmt.Errorf("failed to write tier file: %w", err)
	}

	spawnTime := time.Now()
	if err := WriteSpawnTime(workspacePath, spawnTime); err != nil {
		return fmt.Errorf("failed to write spawn time file: %w", err)
	}

	if err := WriteAttemptID(workspacePath, attemptID); err != nil {
		return fmt.Errorf("failed to write attempt ID file: %w", err)
	}

	if cfg.BeadsID != "" {
		beadsIDPath := filepath.Join(workspacePath, ".beads_id")
		if err := os.WriteFile(beadsIDPath, []byte(cfg.BeadsID), 0644); err != nil {
			return fmt.Errorf("failed to write beads ID file: %w", err)
		}
	}

	if cfg.SpawnMode != "" {
		spawnModePath := filepath.Join(workspacePath, ".spawn_mode")
		if err := os.WriteFile(spawnModePath, []byte(cfg.SpawnMode), 0644); err != nil {
			return fmt.Errorf("failed to write spawn mode file: %w", err)
		}
	}

	manifest := AgentManifest{
		WorkspaceName:    cfg.WorkspaceName,
		Skill:            cfg.SkillName,
		BeadsID:          cfg.BeadsID,
		SourceProjectDir: sourceProjectDir,
		ProjectDir:       sourceProjectDir,
		GitWorktreeDir:   gitWorktreeDir,
		GitBranch:        gitBranch,
		GitBaseline:      getGitBaseline(sourceProjectDir),
		SpawnTime:        spawnTime.Format(time.RFC3339),
		Tier:             cfg.Tier,
		SpawnMode:        cfg.SpawnMode,
		Model:            cfg.Model,
	}
	if err := WriteAgentManifest(workspacePath, manifest); err != nil {
		return fmt.Errorf("failed to write agent manifest: %w", err)
	}

	// Copy spawn artifacts to the worktree so the agent can find them in its CWD.
	// The canonical copies remain in workspacePath; these are delivery copies.
	if err := copySpawnArtifactsToWorktree(cfg); err != nil {
		return fmt.Errorf("failed to copy spawn artifacts to worktree: %w", err)
	}

	return nil
}

// copySpawnArtifactsToWorktree copies key spawn artifacts from the workspace directory
// to the agent's worktree directory so the agent can find them in its CWD.
// This is only needed when CWD differs from ProjectDir (i.e., git worktree spawns).
// The canonical copies remain in .orch/workspace/{name}/.
func copySpawnArtifactsToWorktree(cfg *Config) error {
	worktreeDir := cfg.RuntimeDir()
	workspacePath := cfg.WorkspacePath()

	// Only copy when the runtime directory differs from the workspace path parent.
	// This means a worktree was created and the agent runs in a different directory.
	if worktreeDir == cfg.ProjectDir || worktreeDir == "" {
		return nil
	}

	artifacts := []string{
		"SPAWN_CONTEXT.md",
		AgentManifestFilename,
		verificationSpecFileName,
	}

	for _, name := range artifacts {
		src := filepath.Join(workspacePath, name)
		dst := filepath.Join(worktreeDir, name)

		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip optional artifacts that weren't generated
			}
			return fmt.Errorf("failed to read %s: %w", name, err)
		}

		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s to worktree: %w", name, err)
		}
	}

	return nil
}

// EnsureSynthesisTemplate ensures the SYNTHESIS.md template exists in the project.
func EnsureSynthesisTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "SYNTHESIS.md")

	if _, err := os.Stat(templatePath); err == nil {
		return nil
	}

	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	if err := os.WriteFile(templatePath, []byte(DefaultSynthesisTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write synthesis template: %w", err)
	}

	return nil
}

// MinimalPrompt generates the minimal prompt for opencode run.
// For meta-orchestrator skills, it points to META_ORCHESTRATOR_CONTEXT.md.
// For orchestrator-type skills, it points to ORCHESTRATOR_CONTEXT.md instead.
func MinimalPrompt(cfg *Config) string {
	if cfg.IsMetaOrchestrator {
		return MinimalMetaOrchestratorPrompt(cfg)
	}
	if cfg.IsOrchestrator {
		return MinimalOrchestratorPrompt(cfg)
	}
	// When a worktree is in use, SPAWN_CONTEXT.md is copied there.
	// Point the agent to its CWD copy for reliable discovery.
	runtimeDir := cfg.RuntimeDir()
	if runtimeDir != "" && runtimeDir != cfg.ProjectDir {
		return fmt.Sprintf(
			"Read your spawn context from %s/SPAWN_CONTEXT.md and begin the task.",
			runtimeDir,
		)
	}
	return fmt.Sprintf(
		"Read your spawn context from %s/.orch/workspace/%s/SPAWN_CONTEXT.md and begin the task.",
		cfg.ProjectDir,
		cfg.WorkspaceName,
	)
}

// GenerateInvestigationSlug creates a slug for the investigation file name.
func GenerateInvestigationSlug(task string) string {
	slug := generateSlug(task, 5)
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s-inv-%s", date, slug)
}

// EnsureFailureReportTemplate ensures the FAILURE_REPORT.md template exists in the project.
func EnsureFailureReportTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "FAILURE_REPORT.md")

	if _, err := os.Stat(templatePath); err == nil {
		return nil
	}

	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	if err := os.WriteFile(templatePath, []byte(DefaultFailureReportTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write failure report template: %w", err)
	}

	return nil
}

// WriteFailureReport generates and writes a FAILURE_REPORT.md to the workspace.
// Returns the path to the written file.
func WriteFailureReport(workspacePath, workspaceName, beadsID, reason, task string) (string, error) {
	content := generateFailureReport(workspaceName, beadsID, reason, task)

	reportPath := filepath.Join(workspacePath, "FAILURE_REPORT.md")
	if err := os.WriteFile(reportPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write failure report: %w", err)
	}

	return reportPath, nil
}

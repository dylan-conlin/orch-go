package spawn

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

// MetaOrchestratorContextTemplate is the template for META_ORCHESTRATOR_CONTEXT.md.
// This is used for meta-orchestrator skills (skill-name: meta-orchestrator).
// Key differences from ORCHESTRATOR_CONTEXT.md:
// - Interactive session framing ("managing orchestrator sessions" not "work toward goal")
// - First action: check orch status for sessions to complete or review
// - No SESSION_HANDOFF.md requirement (stay interactive and available)
// - Emphasizes: spawn orchestrators, review handoffs, stay available
const MetaOrchestratorContextTemplate = `# Meta-Orchestrator Session Context

**Role:** You are managing orchestrator sessions
**Skill:** {{.SkillName}}
**Project:** {{.ProjectDir}}
**Started:** {{.StartTime}}

---

## Session Type

You are a **meta-orchestrator** - you manage orchestrator sessions, not worker tasks.

**Key distinction:**
- Orchestrators spawn workers and review their work
- Meta-orchestrators spawn orchestrators and review their session handoffs
- You operate one level above orchestrators

**Your frame:**
- Orchestrator sessions are your "work units" (like workers are to orchestrators)
- You spawn orchestrators to accomplish goals, then review their SESSION_HANDOFF.md
- Stay interactive and available - don't try to accomplish everything and exit

---
{{if .PriorHandoffPath}}
## Prior Session Context

**A previous meta-orchestrator session completed with a handoff.**

Read the prior session's context to understand what was accomplished and what's pending:

` + "```" + `
{{.PriorHandoffPath}}
` + "```" + `

**First action:** Read this file to pick up context from where the previous session left off.

---
{{end}}
## First Actions

Within your first 3 tool calls:
{{if .PriorHandoffPath}}1. **Read the prior SESSION_HANDOFF.md** at {{.PriorHandoffPath}}
2. Run ` + "`orch status`" + ` to check for orchestrator sessions needing completion or review
3. Review pending handoffs or ask Dylan what to do next
{{else}}1. Run ` + "`orch status`" + ` to check for orchestrator sessions needing completion or review
2. Run ` + "`orch review`" + ` if any sessions are pending review
3. Ask Dylan what orchestrator sessions should be spawned (or review pending handoffs)
{{end}}
**Do NOT:**
- Start reading code files (that's worker behavior)
- Try to accomplish a task yourself (spawn an orchestrator for that)
- Produce artifacts and exit (stay interactive)

---

## What You Do

**Spawn orchestrator sessions:**
` + "```bash" + `
# Spawn an orchestrator session with a goal
orch spawn orchestrator "goal description"
` + "```" + `

**Review completed orchestrator sessions:**
` + "```bash" + `
# Check for sessions to review
orch status
orch review

# Complete a session after reviewing its handoff
orch complete <session-id>
` + "```" + `

**Stay available:**
- After spawning or reviewing, ask Dylan what's next
- Don't try to complete everything in one session
- Your value is in managing orchestration, not doing work

---

## What You Don't Do

**Never drop to orchestrator level:**
- Don't spawn workers directly (spawn an orchestrator to do that)
- Don't manage beads issues directly (orchestrators do that)
- Don't review worker SYNTHESIS.md (orchestrators do that)

**Never drop to worker level:**
- Don't read code to understand it (investigation skill does that)
- Don't implement features (feature-impl skill does that)
- Don't debug issues (systematic-debugging skill does that)

**The test:** If you're about to read a file other than:
- SESSION_HANDOFF.md (reviewing orchestrator output)
- orch status output
- This context file

→ STOP. You're dropping levels. Spawn an orchestrator instead.

---

## Authority

**You decide:**
- Which orchestrator sessions to spawn
- When to complete orchestrator sessions
- How to synthesize findings across sessions
- Session prioritization and sequencing

**Escalate when:**
- Strategic direction unclear (what goals should orchestrators pursue?)
- Major scope changes (significant expansion of orchestration targets)
- Resource constraints (too many parallel orchestrator sessions)
- Dylan needs to weigh in on priorities

---

## Session Behavior

**Interactive mode (default):**
- Stay available for conversation with Dylan
- Don't produce completion artifacts and exit
- Ask questions and wait for direction

**No SESSION_HANDOFF.md required:**
- Unlike spawned orchestrators, you don't need to produce handoff artifacts
- Your session is ongoing until Dylan ends it
- Focus on being responsive, not completing tasks
{{if .SkillContent}}
---

## Skill Guidance

{{.SkillContent}}
{{end}}
---

## Context Available

- Global: ~/.claude/CLAUDE.md
- Project: {{.ProjectDir}}/CLAUDE.md
{{if .KBContext}}
{{.KBContext}}
{{end}}
{{if .ServerContext}}
{{.ServerContext}}
{{end}}
{{if .RegisteredProjects}}
{{.RegisteredProjects}}
{{end}}
---

## Workspace

Your workspace is: {{.WorkspacePath}}

**No required artifacts** - meta-orchestrators stay interactive.

**Optional artifacts:**
- Notes on orchestration decisions
- Cross-session synthesis

---

**Remember:** You are a meta-orchestrator. Spawn orchestrators. Review handoffs. Stay available. Never do work yourself.
`

// metaOrchestratorContextData holds template data for META_ORCHESTRATOR_CONTEXT.md.
type metaOrchestratorContextData struct {
	SkillName          string
	SkillContent       string
	ProjectDir         string
	WorkspacePath      string
	WorkspaceName      string
	StartTime          string
	KBContext          string
	ServerContext      string
	RegisteredProjects string
	PriorHandoffPath   string // Path to prior meta-orchestrator's SESSION_HANDOFF.md
}

// GenerateMetaOrchestratorContext generates the META_ORCHESTRATOR_CONTEXT.md content.
func GenerateMetaOrchestratorContext(cfg *Config) (string, error) {
	tmpl, err := template.New("meta_orchestrator_context").Parse(MetaOrchestratorContextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse meta-orchestrator template: %w", err)
	}

	// Generate server context if enabled
	serverContext := cfg.ServerContext
	if cfg.IncludeServers && serverContext == "" {
		serverContext = GenerateServerContext(cfg.ProjectDir)
	}

	// Generate registered projects context for meta-orchestrators
	registeredProjects := cfg.RegisteredProjects
	if registeredProjects == "" {
		registeredProjects = GenerateRegisteredProjectsContext()
	}

	// Find prior meta-orchestrator SESSION_HANDOFF.md if not already set
	priorHandoffPath := cfg.PriorHandoffPath
	if priorHandoffPath == "" {
		priorHandoffPath = findPriorMetaOrchestratorHandoffExcluding(cfg.ProjectDir, cfg.WorkspaceName)
	}

	data := metaOrchestratorContextData{
		SkillName:          cfg.SkillName,
		SkillContent:       cfg.SkillContent,
		ProjectDir:         cfg.ProjectDir,
		WorkspacePath:      cfg.WorkspacePath(),
		WorkspaceName:      cfg.WorkspaceName,
		StartTime:          time.Now().Format("2006-01-02 15:04"),
		KBContext:          cfg.KBContext,
		ServerContext:      serverContext,
		RegisteredProjects: registeredProjects,
		PriorHandoffPath:   priorHandoffPath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute meta-orchestrator template: %w", err)
	}

	return buf.String(), nil
}

// WriteMetaOrchestratorContext writes the META_ORCHESTRATOR_CONTEXT.md file to the workspace.
func WriteMetaOrchestratorContext(cfg *Config) error {
	content, err := GenerateMetaOrchestratorContext(cfg)
	if err != nil {
		return err
	}

	// Create workspace directory
	workspacePath := cfg.WorkspacePath()
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create screenshots subdirectory for agent-produced visual artifacts
	if err := CreateScreenshotsDir(workspacePath); err != nil {
		return err
	}

	// Write context file (named META_ORCHESTRATOR_CONTEXT.md for meta-orchestrator spawns)
	contextPath := filepath.Join(workspacePath, "META_ORCHESTRATOR_CONTEXT.md")
	if err := os.WriteFile(contextPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write meta-orchestrator context file: %w", err)
	}

	// Write spawn time for tracking
	if err := WriteSpawnTime(workspacePath, time.Now()); err != nil {
		return fmt.Errorf("failed to write spawn time file: %w", err)
	}

	// Write tier file for programmatic detection (orch complete, orch status, etc.)
	// Meta-orchestrators use "orchestrator" tier (same verification rules as orchestrators)
	if err := WriteTier(workspacePath, "orchestrator"); err != nil {
		return fmt.Errorf("failed to write tier file: %w", err)
	}

	// Write meta-orchestrator marker file for orch complete to detect
	// Note: .tier file with "orchestrator" value is now the primary detection mechanism
	// Keep .meta-orchestrator for backwards compatibility and to distinguish from regular orchestrators
	markerPath := filepath.Join(workspacePath, ".meta-orchestrator")
	if err := os.WriteFile(markerPath, []byte("meta-orchestrator-spawn"), 0644); err != nil {
		return fmt.Errorf("failed to write meta-orchestrator marker: %w", err)
	}

	// Write workspace name file for lookup during orch complete (meta-orchestrators don't have beads IDs)
	workspaceNamePath := filepath.Join(workspacePath, ".workspace_name")
	if err := os.WriteFile(workspaceNamePath, []byte(cfg.WorkspaceName), 0644); err != nil {
		return fmt.Errorf("failed to write workspace name file: %w", err)
	}

	// Note: Meta-orchestrators do NOT write .beads_id - they don't use beads tracking
	// They also don't require SESSION_HANDOFF.md (stay interactive)

	return nil
}

// MinimalMetaOrchestratorPrompt generates the minimal prompt for meta-orchestrator spawns.
func MinimalMetaOrchestratorPrompt(cfg *Config) string {
	return fmt.Sprintf(
		"Read your session context from %s/.orch/workspace/%s/META_ORCHESTRATOR_CONTEXT.md and begin managing orchestrator sessions.",
		cfg.ProjectDir,
		cfg.WorkspaceName,
	)
}

// FindPriorMetaOrchestratorHandoff finds the most recent completed meta-orchestrator
// SESSION_HANDOFF.md in the target project. This allows new meta-orchestrator sessions
// to pick up context from where the prior session left off.
//
// It searches both .orch/workspace/ and .orch/workspace-archive/ directories,
// looking for workspaces with:
// - .meta-orchestrator marker file (indicating meta-orchestrator workspace)
// - SESSION_HANDOFF.md (indicating completed session)
//
// Returns the path to the most recent SESSION_HANDOFF.md, or empty string if none found.
// The "most recent" is determined by the .spawn_time file in the workspace.
func FindPriorMetaOrchestratorHandoff(projectDir string) string {
	// Exclude the current workspace being created (will be passed in as part of cfg later)
	return findPriorMetaOrchestratorHandoffExcluding(projectDir, "")
}

// findPriorMetaOrchestratorHandoffExcluding finds the most recent meta-orchestrator
// SESSION_HANDOFF.md, excluding the specified workspace name.
func findPriorMetaOrchestratorHandoffExcluding(projectDir, excludeWorkspace string) string {
	type workspaceInfo struct {
		handoffPath string
		spawnTime   time.Time
	}

	var candidates []workspaceInfo

	// Search both workspace and workspace-archive directories
	searchDirs := []string{
		filepath.Join(projectDir, ".orch", "workspace"),
		filepath.Join(projectDir, ".orch", "workspace-archive"),
	}

	for _, searchDir := range searchDirs {
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			continue // Directory may not exist
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			workspaceName := entry.Name()

			// Skip the workspace we're excluding (the one being created)
			if excludeWorkspace != "" && workspaceName == excludeWorkspace {
				continue
			}

			workspacePath := filepath.Join(searchDir, workspaceName)

			// Check for meta-orchestrator marker
			markerPath := filepath.Join(workspacePath, ".meta-orchestrator")
			if _, err := os.Stat(markerPath); os.IsNotExist(err) {
				continue // Not a meta-orchestrator workspace
			}

			// Check for SESSION_HANDOFF.md
			handoffPath := filepath.Join(workspacePath, "SESSION_HANDOFF.md")
			if info, err := os.Stat(handoffPath); err != nil || info.Size() == 0 {
				continue // No handoff or empty file
			}

			// Get spawn time for sorting
			spawnTime := getSpawnTime(workspacePath)

			candidates = append(candidates, workspaceInfo{
				handoffPath: handoffPath,
				spawnTime:   spawnTime,
			})
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort by spawn time (most recent first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].spawnTime.After(candidates[j].spawnTime)
	})

	return candidates[0].handoffPath
}

// getSpawnTime reads the .spawn_time file and parses the timestamp.
// Returns zero time if file doesn't exist or can't be parsed.
func getSpawnTime(workspacePath string) time.Time {
	spawnTimePath := filepath.Join(workspacePath, ".spawn_time")
	content, err := os.ReadFile(spawnTimePath)
	if err != nil {
		return time.Time{}
	}

	// Parse the timestamp (stored as Unix nanoseconds)
	var nanos int64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(content)), "%d", &nanos); err != nil {
		return time.Time{}
	}

	return time.Unix(0, nanos)
}

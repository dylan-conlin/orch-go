package spawn

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// OrchestratorContextTemplate is the template for ORCHESTRATOR_CONTEXT.md.
// This is used for orchestrator-type skills (skill-type: policy/orchestrator).
// Key differences from SPAWN_CONTEXT.md:
// - No beads tracking instructions (orchestrators manage sessions, not issues)
// - SESSION_HANDOFF.md instead of SYNTHESIS.md requirement
// - Session goal focus instead of task focus
// - orch session end instead of /exit
const OrchestratorContextTemplate = `# Orchestrator Session Context

**Session Goal:** {{.SessionGoal}}
**Skill:** {{.SkillName}}
**Project:** {{.ProjectDir}}
**Started:** {{.StartTime}}

---

## Session Type

You are a **spawned orchestrator** - an orchestrator session that was spawned to accomplish a specific goal.
This is different from interactive orchestrator sessions in that:
- You have a defined goal to accomplish
- You should produce a SESSION_HANDOFF.md before ending
- You use ` + "`orch session end`" + ` when complete (not ` + "`/exit`" + `)

---

## First Actions

Within your first 3 tool calls:
1. Read the orchestrator skill guidance (already embedded below)
2. Check current project state: ` + "`orch status`" + ` and ` + "`bd ready`" + `
3. Begin working toward your session goal

---

## Session Scope

**Goal:** {{.SessionGoal}}

**Authority:**
- Spawn worker agents via ` + "`orch spawn`" + `
- Complete agents via ` + "`orch complete`" + `
- Manage beads issues (create, close, label)
- Make tactical decisions about execution approach
- Synthesize findings from completed agents

**Escalate When:**
- Strategic direction unclear (multiple valid high-level approaches)
- Scope significantly larger than expected
- Blocked by missing permissions or access
- Need human judgment on priorities

---

## Completion Protocol

When you've accomplished your session goal:

1. **Create SESSION_HANDOFF.md** in workspace: {{.WorkspacePath}}/SESSION_HANDOFF.md
   - Summary of what was accomplished
   - Active agents and their status
   - Pending work and recommendations
   - Context for the next session
{{if .HasSessionHandoffTemplate}}
   **Template available:** Use SESSION_HANDOFF.template.md in your workspace as the structure for your handoff.
{{end}}
2. **Run:** ` + "`orch session end`" + `
   - This triggers reflection prompts and cleanup
   - Ensures proper session state transitions

**Do NOT use ` + "`/exit`" + `** - that's for worker agents. Orchestrators use ` + "`orch session end`" + `.

---
{{if .SkillContent}}
## Skill Guidance

{{.SkillContent}}

---
{{end}}
## Context Available

- Global: ~/.claude/CLAUDE.md
- Project: {{.ProjectDir}}/CLAUDE.md
{{if .KBContext}}
{{.KBContext}}
{{end}}
{{if .ServerContext}}
{{.ServerContext}}
{{end}}
---

## Workspace

Your workspace is: {{.WorkspacePath}}

**Required artifacts:**
- SESSION_HANDOFF.md (before session end)

**Optional artifacts:**
- Investigation files in .kb/investigations/
- Decision records in .kb/decisions/

---

**Remember:** You are an orchestrator. Delegate spawnable work to agents. Focus on coordination, synthesis, and decision-making.
`

// orchestratorContextData holds template data for ORCHESTRATOR_CONTEXT.md.
type orchestratorContextData struct {
	SessionGoal               string
	SkillName                 string
	SkillContent              string
	ProjectDir                string
	WorkspacePath             string
	WorkspaceName             string
	StartTime                 string
	KBContext                 string
	ServerContext             string
	HasSessionHandoffTemplate bool
}

// GenerateOrchestratorContext generates the ORCHESTRATOR_CONTEXT.md content.
func GenerateOrchestratorContext(cfg *Config) (string, error) {
	tmpl, err := template.New("orchestrator_context").Parse(OrchestratorContextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse orchestrator template: %w", err)
	}

	// Generate server context if enabled
	serverContext := cfg.ServerContext
	if cfg.IncludeServers && serverContext == "" {
		serverContext = GenerateServerContext(cfg.ProjectDir)
	}

	data := orchestratorContextData{
		SessionGoal:               cfg.SessionGoal,
		SkillName:                 cfg.SkillName,
		SkillContent:              cfg.SkillContent,
		ProjectDir:                cfg.ProjectDir,
		WorkspacePath:             cfg.WorkspacePath(),
		WorkspaceName:             cfg.WorkspaceName,
		StartTime:                 time.Now().Format("2006-01-02 15:04"),
		KBContext:                 cfg.KBContext,
		ServerContext:             serverContext,
		HasSessionHandoffTemplate: cfg.HasSessionHandoffTemplate,
	}

	// Use Task as SessionGoal if SessionGoal not explicitly set
	if data.SessionGoal == "" {
		data.SessionGoal = cfg.Task
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute orchestrator template: %w", err)
	}

	return buf.String(), nil
}

// WriteOrchestratorContext writes the ORCHESTRATOR_CONTEXT.md file to the workspace.
func WriteOrchestratorContext(cfg *Config) error {
	// Create workspace directory
	workspacePath := cfg.WorkspacePath()
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Copy SESSION_HANDOFF.md template to workspace if it exists
	cfg.HasSessionHandoffTemplate = copySessionHandoffTemplate(cfg.ProjectDir, workspacePath)

	content, err := GenerateOrchestratorContext(cfg)
	if err != nil {
		return err
	}

	// Write context file (named ORCHESTRATOR_CONTEXT.md for orchestrator spawns)
	contextPath := filepath.Join(workspacePath, "ORCHESTRATOR_CONTEXT.md")
	if err := os.WriteFile(contextPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write orchestrator context file: %w", err)
	}

	// Write spawn time for tracking
	if err := WriteSpawnTime(workspacePath, time.Now()); err != nil {
		return fmt.Errorf("failed to write spawn time file: %w", err)
	}

	// Write orchestrator marker file for orch complete to detect
	markerPath := filepath.Join(workspacePath, ".orchestrator")
	if err := os.WriteFile(markerPath, []byte("orchestrator-spawn"), 0644); err != nil {
		return fmt.Errorf("failed to write orchestrator marker: %w", err)
	}

	// Write beads ID file for workspace lookup during orch complete
	if cfg.BeadsID != "" {
		beadsIDPath := filepath.Join(workspacePath, ".beads_id")
		if err := os.WriteFile(beadsIDPath, []byte(cfg.BeadsID), 0644); err != nil {
			return fmt.Errorf("failed to write beads ID file: %w", err)
		}
	}

	return nil
}

// copySessionHandoffTemplate copies the SESSION_HANDOFF.md template from
// .orch/templates/ to the workspace as SESSION_HANDOFF.template.md.
// Returns true if the template was copied, false if it doesn't exist.
func copySessionHandoffTemplate(projectDir, workspacePath string) bool {
	templatePath := filepath.Join(projectDir, ".orch", "templates", "SESSION_HANDOFF.md")

	// Check if template exists
	content, err := os.ReadFile(templatePath)
	if err != nil {
		// Template doesn't exist - that's fine, it's optional
		return false
	}

	// Copy to workspace with .template.md suffix
	destPath := filepath.Join(workspacePath, "SESSION_HANDOFF.template.md")
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		// Log but don't fail - template is optional
		return false
	}

	return true
}

// MinimalOrchestratorPrompt generates the minimal prompt for orchestrator spawns.
func MinimalOrchestratorPrompt(cfg *Config) string {
	return fmt.Sprintf(
		"Read your session context from %s/.orch/workspace/%s/ORCHESTRATOR_CONTEXT.md and begin working toward your session goal.",
		cfg.ProjectDir,
		cfg.WorkspaceName,
	)
}

// DefaultSessionHandoffTemplate is the embedded SESSION_HANDOFF.md template content.
const DefaultSessionHandoffTemplate = `# Session Handoff

**Orchestrator Session:** {workspace-name}
**Goal:** {session-goal}
**Duration:** {start-time} → {end-time}
**Outcome:** {accomplished | partial | blocked}

---

## Summary

[1-2 sentence summary of what was accomplished during this session]

---

## Work Completed

### Agents Spawned
- ` + "`{agent-name}`" + ` - {status} - {brief outcome}

### Issues Closed
- ` + "`{issue-id}`" + ` - {reason}

### Decisions Made
- {Decision 1}
- {Decision 2}

---

## Active Work

### Running Agents
- ` + "`{agent-name}`" + ` - {current phase} - {ETA if known}

### Pending Issues
- ` + "`{issue-id}`" + ` - {why pending}

---

## Recommendations for Next Session

**Immediate Priority:**
- {What to do first when resuming}

**Pending Decisions:**
- {Decisions that need human input}

**Blocked Items:**
- {What's blocked and why}

---

## Context to Remember

- {Key insight 1}
- {Key insight 2}
- {Important constraint discovered}

---

## Session Metadata

**Skill:** {skill-name}
**Workspace:** ` + "`" + `.orch/workspace/{workspace-name}/` + "`" + `
`

// EnsureSessionHandoffTemplate ensures the SESSION_HANDOFF.md template exists.
func EnsureSessionHandoffTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "SESSION_HANDOFF.md")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template exists
	}

	// Create templates directory if needed
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Write the default template
	if err := os.WriteFile(templatePath, []byte(DefaultSessionHandoffTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write session handoff template: %w", err)
	}

	return nil
}

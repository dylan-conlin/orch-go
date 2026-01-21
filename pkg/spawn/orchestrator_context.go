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
// - SYNTHESIS.md instead of SYNTHESIS.md requirement
// - Session goal focus instead of task focus
// - Waits for level above to complete (no /exit)
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
- You produce a SYNTHESIS.md when goal is reached
- You WAIT after writing SYNTHESIS.md - the level above (meta-orchestrator or Dylan) runs ` + "`orch complete`" + ` to close your session

---

## First Actions

Within your first 5 tool calls:
1. Read the orchestrator skill guidance (already embedded below)
2. Check current project state: ` + "`orch status`" + ` and ` + "`bd ready`" + `
3. **Fill SYNTHESIS.md sections:** Open {{.WorkspacePath}}/SYNTHESIS.md and fill in:
   - TLDR (initial framing of session goal)
   - "Where We Started" in Focus Progress section (current state at session start)
4. Begin working toward your session goal

**Progressive Documentation:** SYNTHESIS.md has been pre-created with metadata. Fill sections AS YOU WORK, not at the end.

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

1. **Create SYNTHESIS.md** in workspace: {{.WorkspacePath}}/SYNTHESIS.md
   - Summary of what was accomplished
   - Active agents and their status
   - Pending work and recommendations
   - Context for the next session
{{if .HasSynthesisTemplate}}
   **Template available:** Use SYNTHESIS.template.md in your workspace as the structure for your synthesis.
{{end}}
2. **WAIT** - Do not exit or try to end your session
   - The level above (meta-orchestrator or Dylan) will run ` + "`orch complete`" + ` to close your session
   - Your SYNTHESIS.md signals you are ready for completion
   - If you need to signal completion, you can say "SYNTHESIS.md written, ready for completion"

**Do NOT use ` + "`/exit`" + `** - spawned orchestrators wait for the level above to complete them.

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
{{if .RegisteredProjects}}
{{.RegisteredProjects}}
{{end}}
---

## Workspace

Your workspace is: {{.WorkspacePath}}

**Required artifacts:**
- SYNTHESIS.md (before session end)

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
	RegisteredProjects        string
	HasSynthesisTemplate bool
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

	// Generate registered projects context for orchestrators
	registeredProjects := cfg.RegisteredProjects
	if registeredProjects == "" {
		registeredProjects = GenerateRegisteredProjectsContext()
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
		RegisteredProjects:        registeredProjects,
		HasSynthesisTemplate: cfg.HasSynthesisTemplate,
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

	// Create screenshots subdirectory for agent-produced visual artifacts
	if err := CreateScreenshotsDir(workspacePath); err != nil {
		return err
	}

	// Copy SYNTHESIS.md template to workspace if it exists (as reference)
	cfg.HasSynthesisTemplate = copySynthesisTemplate(cfg.ProjectDir, workspacePath)

	// Pre-create SYNTHESIS.md with metadata filled
	// This encourages progressive documentation - orchestrators fill as they work
	startTime := time.Now().Format("2006-01-02 15:04")
	sessionGoal := cfg.SessionGoal
	if sessionGoal == "" {
		sessionGoal = cfg.Task
	}
	if err := writePreFilledSynthesis(workspacePath, cfg.WorkspaceName, sessionGoal, startTime); err != nil {
		return fmt.Errorf("failed to write pre-filled synthesis: %w", err)
	}

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

	// Write tier file for programmatic detection (orch complete, orch status, etc.)
	// Uses "orchestrator" tier which has special verification rules (SYNTHESIS.md, no beads)
	if err := WriteTier(workspacePath, "orchestrator"); err != nil {
		return fmt.Errorf("failed to write tier file: %w", err)
	}

	// Write orchestrator marker file for orch complete to detect
	// Note: .tier file with "orchestrator" value is now the primary detection mechanism
	// Keep .orchestrator for backwards compatibility with existing workspaces
	markerPath := filepath.Join(workspacePath, ".orchestrator")
	if err := os.WriteFile(markerPath, []byte("orchestrator-spawn"), 0644); err != nil {
		return fmt.Errorf("failed to write orchestrator marker: %w", err)
	}

	// Write workspace name file for lookup during orch complete (orchestrators don't have beads IDs)
	workspaceNamePath := filepath.Join(workspacePath, ".workspace_name")
	if err := os.WriteFile(workspaceNamePath, []byte(cfg.WorkspaceName), 0644); err != nil {
		return fmt.Errorf("failed to write workspace name file: %w", err)
	}

	// Note: Orchestrators do NOT write .beads_id - they don't use beads tracking
	// SYNTHESIS.md is the completion signal, not Phase: Complete

	return nil
}

// copySynthesisTemplate copies the SYNTHESIS.md template from
// .orch/templates/ to the workspace as SYNTHESIS.template.md.
// Returns true if the template was copied, false if it doesn't exist.
func copySynthesisTemplate(projectDir, workspacePath string) bool {
	templatePath := filepath.Join(projectDir, ".orch", "templates", "SYNTHESIS.md")

	// Check if template exists
	content, err := os.ReadFile(templatePath)
	if err != nil {
		// Template doesn't exist - that's fine, it's optional
		return false
	}

	// Copy to workspace with .template.md suffix
	destPath := filepath.Join(workspacePath, "SYNTHESIS.template.md")
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

// DefaultSynthesisTemplate is the embedded SYNTHESIS.md template content.
const DefaultSynthesisTemplate = `# Synthesis

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

// PreFilledSynthesisTemplate is used to pre-create SYNTHESIS.md with metadata.
// This encourages progressive documentation - fill as you work, not at the end.
const PreFilledSynthesisTemplate = `# Synthesis

**Orchestrator:** {{.WorkspaceName}}
**Focus:** {{.SessionGoal}}
**Duration:** {{.StartTime}} → {end-time}
**Outcome:** {success | partial | blocked | failed}

---

<!--
## Progressive Documentation (READ THIS FIRST)

**This file has been pre-created with metadata. Fill sections AS YOU WORK.**

**Within first 5 tool calls:**
1. Fill TLDR (initial framing of what you're trying to accomplish)
2. Fill "Where We Started" (current state at session start)

**During work:**
- Add to Spawns table as you spawn/complete agents
- Add to Evidence as you observe patterns
- Capture Friction immediately (you'll rationalize it away later)

**Before completion:**
- Synthesize Knowledge section
- Fill Next section with recommendations
- Update TLDR to reflect what actually happened
- Update Outcome field
-->

## TLDR

[Fill within first 5 tool calls: What is this session trying to accomplish?]

---

## Spawns (Agents Managed)

### Completed
| Agent | Issue | Skill | Outcome | Key Finding |
|-------|-------|-------|---------|-------------|
| {workspace} | {beads-id} | {skill} | {success/partial/failed} | {1-line insight} |

### Still Running
| Agent | Issue | Skill | Phase | ETA |
|-------|-------|-------|-------|-----|
| {workspace} | {beads-id} | {skill} | {phase} | {estimate} |

### Blocked/Failed
| Agent | Issue | Blocker | Next Step |
|-------|-------|---------|-----------|
| {workspace} | {beads-id} | {what blocked} | {spawn-fresh/escalate/defer} |

---

## Evidence (What Was Observed)

### Patterns Across Agents
- [Pattern 1: e.g., "3 agents hit the same auth issue"]

### Completions
- **{beads-id}:** {what SYNTHESIS.md revealed}

### System Behavior
- [Observation about orch/beads/kb tooling]

---

## Knowledge (What Was Learned)

### Decisions Made
- **{topic}:** {decision} because {rationale}

### Constraints Discovered
- {constraint} - why it matters

### Externalized
- ` + "`kn decide \"X\" --reason \"Y\"`" + ` - [if applicable]
- ` + "`.kb/decisions/YYYY-MM-DD-*.md`" + ` - [if created]

### Artifacts Created
- [list any investigations, decisions, or other artifacts]

---

## Friction (What Was Harder Than It Should Be)

<!--
Capture frustrations AS THEY HAPPEN. You'll rationalize them away later.
-->

### Tooling Friction
- [Tool gap or UX issue]

### Context Friction
- [Information that should have been surfaced but wasn't]

### Skill/Spawn Friction
- [Skill guidance was unclear or wrong]

*(If smooth session: "No significant friction observed")*

---

## Focus Progress

### Where We Started
[Fill within first 5 tool calls: What is the current state before you begin working?]

### Where We Ended
- {state of focus goal now}
- {what shifted or became clearer}

### Scope Changes
- [If focus shifted mid-session, note why]

---

## Next (What Should Happen)

**Recommendation:** {continue-focus | shift-focus | escalate | pause}

### If Continue Focus
**Immediate:** {first thing next orchestrator should do}
**Then:** {subsequent priorities}
**Context to reload:**
- {key file or artifact to read}

### If Shift Focus
**New focus:** {recommended focus}
**Why shift:** {rationale}

### If Escalate
**Question for meta-orchestrator:** {what needs decision}
**Recommendation:** {which option and why}

### If Pause
**Why pausing:** {rationale}
**Resume conditions:** {what needs to happen before resuming}

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- [Question 1 - why it's interesting]

**System improvement ideas:**
- [Tooling or process idea]

*(If nothing emerged: "Focused session, no unexplored territory")*

---

## Session Metadata

**Agents spawned:** {count}
**Agents completed:** {count}
**Issues closed:** {list}
**Issues created:** {list}

**Workspace:** ` + "`.orch/workspace/{{.WorkspaceName}}/`" + `
`

// preFilledSynthesisData holds template data for pre-filled SYNTHESIS.md.
type preFilledSynthesisData struct {
	WorkspaceName string
	SessionGoal   string
	StartTime     string
}

// GeneratePreFilledSynthesis generates SYNTHESIS.md with metadata pre-filled.
func GeneratePreFilledSynthesis(workspaceName, sessionGoal, startTime string) (string, error) {
	tmpl, err := template.New("synthesis").Parse(PreFilledSynthesisTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse synthesis template: %w", err)
	}

	data := preFilledSynthesisData{
		WorkspaceName: workspaceName,
		SessionGoal:   sessionGoal,
		StartTime:     startTime,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute synthesis template: %w", err)
	}

	return buf.String(), nil
}

// writePreFilledSynthesis writes SYNTHESIS.md with metadata pre-filled.
func writePreFilledSynthesis(workspacePath, workspaceName, sessionGoal, startTime string) error {
	content, err := GeneratePreFilledSynthesis(workspaceName, sessionGoal, startTime)
	if err != nil {
		return err
	}

	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	if err := os.WriteFile(synthesisPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write pre-filled synthesis: %w", err)
	}

	return nil
}

// EnsureSynthesisTemplate ensures the SYNTHESIS.md template exists.
func EnsureSynthesisTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "SYNTHESIS.md")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template exists
	}

	// Create templates directory if needed
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Write the default template
	if err := os.WriteFile(templatePath, []byte(DefaultSynthesisTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write synthesis template: %w", err)
	}

	return nil
}

package spawn

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// SpawnContextTemplate is the basic structure for SPAWN_CONTEXT.md.
// This is a simplified version of the Python template.
const SpawnContextTemplate = `TASK: {{.Task}}

🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via ` + "`bd comment {{.BeadsID}} \"Phase: Planning - [brief description]\"`" + `
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary of deliverables]\"`" + `
3. Run: ` + "`/exit`" + ` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.


CONTEXT: [See task description]

PROJECT_DIR: {{.ProjectDir}}

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours


AUTHORITY:
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be {{.ProjectDir}})
2. **SET UP investigation file:** Run ` + "`kb create investigation {{.InvestigationSlug}}`" + ` to create from template
   - This creates: ` + "`.kb/investigations/simple/YYYY-MM-DD-{{.InvestigationSlug}}.md`" + `
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running ` + "`kb create`" + `, report the actual path via:
     ` + "`bd comment {{.BeadsID}} \"investigation_path: /path/to/file.md\"`" + `
     (This allows orch complete to verify the correct file)
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
6. **CREATE SYNTHESIS.md:** Before completing, create ` + "`SYNTHESIS.md`" + ` in your workspace: {{.ProjectDir}}/.orch/workspace/{{.WorkspaceName}}/SYNTHESIS.md
   - Use the template from: {{.ProjectDir}}/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.

STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input

## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **{{.BeadsID}}**

**Use ` + "`bd comment`" + ` for progress updates instead of workspace-only tracking:**

` + "```bash" + `
# Report progress at phase transitions
bd comment {{.BeadsID}} "Phase: Planning - Analyzing codebase structure"
bd comment {{.BeadsID}} "Phase: Implementing - Adding authentication middleware"
bd comment {{.BeadsID}} "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment {{.BeadsID}} "BLOCKED: Need clarification on API contract"

# Report questions
bd comment {{.BeadsID}} "QUESTION: Should we use JWT or session-based auth?"
` + "```" + `

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via ` + "`bd show {{.BeadsID}}`" + `.

⛔ **NEVER run ` + "`bd close`" + `** - Only the orchestrator closes issues via ` + "`orch complete`" + `.
   - Workers report ` + "`Phase: Complete`" + `, orchestrator verifies and closes
   - Running ` + "`bd close`" + ` bypasses verification and breaks tracking

{{if .SkillContent}}
## SKILL GUIDANCE ({{.SkillName}})

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

{{.SkillContent}}

---
{{end}}
{{if .Phases}}
FEATURE-IMPL CONFIGURATION:
Phases: {{.Phases}}
Mode: {{if .Mode}}{{.Mode}}{{else}}tdd{{end}}
Validation: {{if .Validation}}{{.Validation}}{{else}}tests{{end}}

Follow phase guidance from the feature-impl skill.
{{end}}
{{if .InvestigationType}}
INVESTIGATION CONFIGURATION:
Type: {{.InvestigationType}}

Create investigation file in .kb/investigations/{{.InvestigationType}}/ subdirectory.
Follow investigation skill guidance for {{.InvestigationType}} investigations.
{{end}}

CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: {{.ProjectDir}}/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + `
3. ` + "`/exit`" + `

⚠️ Your work is NOT complete until you run both commands.
`

// contextData holds template data for SPAWN_CONTEXT.md.
type contextData struct {
	Task              string
	BeadsID           string
	ProjectDir        string
	WorkspaceName     string
	SkillName         string
	SkillContent      string
	InvestigationSlug string
	Phases            string
	Mode              string
	Validation        string
	InvestigationType string
}

// GenerateContext generates the SPAWN_CONTEXT.md content.
func GenerateContext(cfg *Config) (string, error) {
	tmpl, err := template.New("spawn_context").Parse(SpawnContextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Generate investigation slug from task
	slug := generateSlug(cfg.Task, 5)

	data := contextData{
		Task:              cfg.Task,
		BeadsID:           cfg.BeadsID,
		ProjectDir:        cfg.ProjectDir,
		WorkspaceName:     cfg.WorkspaceName,
		SkillName:         cfg.SkillName,
		SkillContent:      cfg.SkillContent,
		InvestigationSlug: slug,
		Phases:            cfg.Phases,
		Mode:              cfg.Mode,
		Validation:        cfg.Validation,
		InvestigationType: cfg.InvestigationType,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// WriteContext writes the SPAWN_CONTEXT.md file to the workspace.
func WriteContext(cfg *Config) error {
	content, err := GenerateContext(cfg)
	if err != nil {
		return err
	}

	// Ensure SYNTHESIS.md template exists in the project
	if err := EnsureSynthesisTemplate(cfg.ProjectDir); err != nil {
		return fmt.Errorf("failed to ensure synthesis template: %w", err)
	}

	// Create workspace directory
	workspacePath := cfg.WorkspacePath()
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Write context file
	contextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
	if err := os.WriteFile(contextPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	return nil
}

// EnsureSynthesisTemplate ensures the SYNTHESIS.md template exists in the project.
// If the project doesn't have .orch/templates/SYNTHESIS.md, it creates one from
// the embedded default template.
func EnsureSynthesisTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "SYNTHESIS.md")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template exists, nothing to do
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

// DefaultSynthesisTemplate is the embedded SYNTHESIS.md template content.
// This is used as a fallback when a project doesn't have its own template.
const DefaultSynthesisTemplate = `# Session Synthesis

**Agent:** {workspace-name}
**Issue:** {beads-id}
**Duration:** {start-time} → {end-time}
**Outcome:** {success | partial | blocked | failed}

---

## TLDR

[1-2 sentence summary. What was the goal? What was achieved?]

---

## Delta (What Changed)

### Files Created
- ` + "`path/to/file.go`" + ` - Brief description

### Files Modified
- ` + "`path/to/existing.go`" + ` - What was changed

### Commits
- ` + "`abc1234`" + ` - Commit message summary

---

## Evidence (What Was Observed)

- Observation 1 with source reference (file:line or command output)
- Observation 2 with source reference
- Key finding that informed decisions

### Tests Run
` + "```bash" + `
# Command and result
go test ./... 
# PASS: all tests passing
` + "```" + `

---

## Knowledge (What Was Learned)

### New Artifacts
- ` + "`.kb/investigations/YYYY-MM-DD-*.md`" + ` - Brief description

### Decisions Made
- Decision 1: [choice] because [rationale]

### Constraints Discovered
- Constraint 1 - Why it matters

### Externalized via ` + "`kn`" + `
- ` + "`kn decide \"X\" --reason \"Y\"`" + ` - [if applicable]
- ` + "`kn constraint \"X\" --reason \"Y\"`" + ` - [if applicable]
- ` + "`kn tried \"X\" --failed \"Y\"`" + ` - [if applicable]

---

## Next (What Should Happen)

**Recommendation:** {close | spawn-follow-up | escalate | resume}

### If Close
- [ ] All deliverables complete
- [ ] Tests passing
- [ ] Investigation file has ` + "`**Phase:** Complete`" + `
- [ ] Ready for ` + "`orch complete {issue-id}`" + `

### If Spawn Follow-up
**Issue:** {new-issue-title}
**Skill:** {recommended-skill}
**Context:**
` + "```" + `
{Brief context for next agent - 2-3 sentences max}
` + "```" + `

### If Escalate
**Question:** {what needs decision from orchestrator}
**Options:**
1. {option A} - pros/cons
2. {option B} - pros/cons

**Recommendation:** {which option and why}

### If Resume
**Next Step:** {what to do when resuming}
**Blocker:** {what prevented completion}
**Context to Reload:**
- {key file to re-read}
- {state to remember}

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- [Question 1 - why it's interesting]
- [Question 2 - why it's interesting]

**Areas worth exploring further:**
- [Area 1]
- [Area 2]

**What remains unclear:**
- [Uncertainty 1]
- [Uncertainty 2]

*(If nothing emerged, note: "Straightforward session, no unexplored territory")*

---

## Session Metadata

**Skill:** {skill-name}
**Model:** {model-used}
**Workspace:** ` + "`.orch/workspace/{workspace-name}/`" + `
**Investigation:** ` + "`.kb/investigations/YYYY-MM-DD-*.md`" + `
**Beads:** ` + "`bd show {issue-id}`" + `
`

// MinimalPrompt generates the minimal prompt for opencode run.
func MinimalPrompt(cfg *Config) string {
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

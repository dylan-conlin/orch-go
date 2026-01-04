package spawn

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// Pre-compiled regex patterns for context.go
var (
	regexBeadsSectionHeader    = regexp.MustCompile(`(?i)^#+\s*(report\s+(via|to)\s+beads|beads\s+(progress\s+)?tracking)`)
	regexNextSectionHeader     = regexp.MustCompile(`^#{1,6}\s+[A-Z]`)
	regexBeadsReportedCriteria = regexp.MustCompile(`(?i)\*\*Reported\*\*.*bd\s+comment`)
	regexBeadsIDPlaceholder    = regexp.MustCompile(`bd\s+(comment|close|show)\s+<beads-id>`)
	regexMultiNewline          = regexp.MustCompile(`\n{3,}`)
)

// SpawnContextTemplate is the basic structure for SPAWN_CONTEXT.md.
// This is a simplified version of the Python template.
const SpawnContextTemplate = `TASK: {{.Task}}
{{if .Tier}}
SPAWN TIER: {{.Tier}}
{{if eq .Tier "light"}}
⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.
{{else}}
📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.
{{end}}
{{end}}
{{if .KBContext}}
{{.KBContext}}
{{end}}
{{if .IsBug}}
## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
{{.ReproSteps}}

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: ` + "`bd comment {{.BeadsID}} \"Reproduction verified: [describe test performed]\"`" + `

⚠️ A bug fix is only complete when the original reproduction steps pass.
{{end}}
{{if .NoTrack}}
📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:
{{if eq .Tier "light"}}
1. Run: ` + "`/exit`" + ` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: ` + "`/exit`" + ` to close the agent session
{{end}}
{{else}}
🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via ` + "`bd comment {{.BeadsID}} \"Phase: Planning - [brief description]\"`" + `
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
{{if eq .Tier "light"}}
1. Run: ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary of deliverables]\"`" + `
2. Run: ` + "`/exit`" + ` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary of deliverables]\"`" + `
3. Run: ` + "`/exit`" + ` to close the agent session
{{end}}
⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.
{{end}}

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

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
{{if .NoTrack}}1. Document it in your investigation file: "CONSTRAINT: [what constraint] - [why considering workaround]"
2. Include the constraint and your reasoning in SYNTHESIS.md
{{else}}1. Surface it first: ` + "`bd comment {{.BeadsID}} \"CONSTRAINT: [what constraint] - [why considering workaround]\"`" + `
2. Wait for orchestrator acknowledgment before proceeding
{{end}}3. The accountability is a feature, not a cost

This applies to:
- System constraints discovered during work (e.g., API limits, tool limitations)
- Architectural patterns that seem inconvenient for your task
- Process requirements that feel like overhead
- Prior decisions (from ` + "`kb context`" + `) that conflict with your approach

**Why:** Working around constraints without surfacing them:
- Prevents the system from learning about recurring friction
- Bypasses stakeholders who should know about the limitation
- Creates hidden technical debt

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be {{.ProjectDir}})
2. **SET UP investigation file:** Run ` + "`kb create investigation {{.InvestigationSlug}}`" + ` to create from template
   - This creates: ` + "`.kb/investigations/simple/YYYY-MM-DD-{{.InvestigationSlug}}.md`" + `
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
{{if not .NoTrack}}
   - **IMPORTANT:** After running ` + "`kb create`" + `, report the actual path via:
     ` + "`bd comment {{.BeadsID}} \"investigation_path: /path/to/file.md\"`" + `
     (This allows orch complete to verify the correct file)
{{end}}
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
{{if ne .Tier "light"}}
6. **CREATE SYNTHESIS.md:** Before completing, create ` + "`SYNTHESIS.md`" + ` in your workspace: {{.ProjectDir}}/.orch/workspace/{{.WorkspaceName}}/SYNTHESIS.md
   - Use the template from: {{.ProjectDir}}/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.
{{else}}
6. ⚡ SYNTHESIS.md is NOT required (light tier spawn).
{{end}}

STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input
{{if not .NoTrack}}

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
{{end}}

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
{{if .ServerContext}}

{{.ServerContext}}
{{end}}
🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
{{if .NoTrack}}
{{if eq .Tier "light"}}
1. ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. ` + "`/exit`" + `
{{end}}
{{else}}
{{if eq .Tier "light"}}
1. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + `
2. ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + `
3. ` + "`/exit`" + `
{{end}}
{{end}}
⚠️ Your work is NOT complete until you run these commands.
`

// StripBeadsInstructions removes beads-specific instructions from skill content.
// This is used when NoTrack=true to avoid confusing agents with beads commands
// that won't work (since there's no beads issue to track against).
//
// Removes:
// - Code blocks containing `bd comment` or `bd close` commands
// - "Report via Beads" sections
// - Lines containing `<beads-id>` placeholders
// - Completion criteria mentioning beads reporting
func StripBeadsInstructions(content string) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string
	inBeadsCodeBlock := false
	skipUntilNextSection := false
	inCodeBlockDuringSkip := false // Track if we entered a code block while skipping



	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track code block state even while skipping
		// This prevents us from ending skip mode inside a code block
		if strings.HasPrefix(trimmedLine, "```") {
			if skipUntilNextSection {
				inCodeBlockDuringSkip = !inCodeBlockDuringSkip
			}
		}

		// Check if we're starting a beads-related section
		if regexBeadsSectionHeader.MatchString(line) {
			skipUntilNextSection = true
			inCodeBlockDuringSkip = false // Reset code block tracking
			continue
		}

		// Check if we've reached a new section (exit beads section)
		// But ONLY if we're not inside a code block
		if skipUntilNextSection && !inCodeBlockDuringSkip && regexNextSectionHeader.MatchString(line) && !regexBeadsSectionHeader.MatchString(line) {
			skipUntilNextSection = false
			// Include this line (the new section header)
		}

		// Skip lines while in beads section
		if skipUntilNextSection {
			continue
		}

		_ = i // Silence unused variable warning

		// Check for code blocks containing beads commands
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inBeadsCodeBlock {
				// End of beads code block - skip the closing ```
				inBeadsCodeBlock = false
				continue
			}
			// Look ahead to see if this code block contains beads commands
			hasBeadsCommand := false
			for j := i + 1; j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "```"); j++ {
				if regexBeadsIDPlaceholder.MatchString(lines[j]) {
					hasBeadsCommand = true
					break
				}
			}
			if hasBeadsCommand {
				inBeadsCodeBlock = true
				continue
			}
		}

		// Skip lines inside beads code blocks
		if inBeadsCodeBlock {
			continue
		}

		// Skip individual lines with beads completion criteria
		if regexBeadsReportedCriteria.MatchString(line) {
			continue
		}

		// Skip lines that are just beads commands with <beads-id>
		if regexBeadsIDPlaceholder.MatchString(line) && strings.TrimSpace(line) != "" {
			// Only skip if it's a standalone command line (not part of documentation)
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "bd ") || strings.HasPrefix(trimmed, "- `bd ") {
				continue
			}
		}

		result = append(result, line)
	}

	// Clean up excessive blank lines that may result from stripping
	output := strings.Join(result, "\n")
	// Replace 3+ consecutive newlines with 2
	output = regexMultiNewline.ReplaceAllString(output, "\n\n")

	return output
}

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
	KBContext         string
	Tier              string
	ServerContext     string
	NoTrack           bool   // When true, omit beads instructions from spawn context
	IsBug             bool   // When true, this is a bug issue with reproduction info
	ReproSteps        string // Reproduction steps from bug issue
}

// GenerateContext generates the SPAWN_CONTEXT.md content.
func GenerateContext(cfg *Config) (string, error) {
	tmpl, err := template.New("spawn_context").Parse(SpawnContextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Generate investigation slug from task
	slug := generateSlug(cfg.Task, 5)

	// Generate server context if enabled
	serverContext := cfg.ServerContext
	if cfg.IncludeServers && serverContext == "" {
		serverContext = GenerateServerContext(cfg.ProjectDir)
	}

	// Strip beads instructions from skill content when NoTrack is true
	// This prevents confusing agents with beads commands that won't work
	skillContent := cfg.SkillContent
	if cfg.NoTrack && skillContent != "" {
		skillContent = StripBeadsInstructions(skillContent)
	}

	data := contextData{
		Task:              cfg.Task,
		BeadsID:           cfg.BeadsID,
		ProjectDir:        cfg.ProjectDir,
		WorkspaceName:     cfg.WorkspaceName,
		SkillName:         cfg.SkillName,
		SkillContent:      skillContent,
		InvestigationSlug: slug,
		Phases:            cfg.Phases,
		Mode:              cfg.Mode,
		Validation:        cfg.Validation,
		InvestigationType: cfg.InvestigationType,
		KBContext:         cfg.KBContext,
		Tier:              cfg.Tier,
		ServerContext:     serverContext,
		NoTrack:           cfg.NoTrack,
		IsBug:             cfg.IsBug,
		ReproSteps:        cfg.ReproSteps,
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

	// Ensure SYNTHESIS.md template exists in the project (only for full tier)
	if cfg.Tier != TierLight {
		if err := EnsureSynthesisTemplate(cfg.ProjectDir); err != nil {
			return fmt.Errorf("failed to ensure synthesis template: %w", err)
		}
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

	// Write tier metadata file for orch complete to read
	if err := WriteTier(workspacePath, cfg.Tier); err != nil {
		return fmt.Errorf("failed to write tier file: %w", err)
	}

	// Write spawn time for constraint verification scoping
	// Constraints should only match files created after this spawn, not pre-existing files
	if err := WriteSpawnTime(workspacePath, time.Now()); err != nil {
		return fmt.Errorf("failed to write spawn time file: %w", err)
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

// EnsureFailureReportTemplate ensures the FAILURE_REPORT.md template exists in the project.
// If the project doesn't have .orch/templates/FAILURE_REPORT.md, it creates one from
// the embedded default template.
func EnsureFailureReportTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "FAILURE_REPORT.md")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template exists, nothing to do
	}

	// Create templates directory if needed
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Write the default template
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

// generateFailureReport creates the content for a FAILURE_REPORT.md file.
func generateFailureReport(workspaceName, beadsID, reason, task string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	return fmt.Sprintf(`# Failure Report

**Agent:** %s
**Issue:** %s
**Abandoned:** %s
**Reason:** %s

---

## Context

**Task:** %s

**What was attempted:**
[Brief description of what the agent was trying to do]

---

## Failure Summary

**Primary Cause:** %s

**Details:**
[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]

---

## Progress Made

### Completed Steps
- [ ] [Step 1 - if any]

### Partial Progress
- [What was started but not finished]

### Artifacts Created
- [List any files created before abandonment]

---

## Learnings

**What worked:**
- [Things that went well before failure]

**What didn't work:**
- [Approaches that failed or caused issues]

**Root cause analysis:**
- [If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]

---

## Recovery Recommendations

**Can this be retried?** yes

**If yes, what should be different:**
- [Suggestion 1 - different approach]
- [Suggestion 2 - smaller scope]
- [Suggestion 3 - additional context needed]

**If spawning a new agent:**
`+"```"+`
orch spawn {skill} "{adjusted-task}" --issue %s
`+"```"+`

**Context to provide:**
- [Key insight 1 for next agent]
- [Key insight 2 for next agent]
- [Pitfall to avoid]

---

## Session Metadata

**Workspace:** `+"`"+`.orch/workspace/%s/`+"`"+`
**Beads:** `+"`"+`bd show %s`+"`"+`
`, workspaceName, beadsID, timestamp, reason, task, reason, beadsID, workspaceName, beadsID)
}

// DefaultFailureReportTemplate is the embedded FAILURE_REPORT.md template content.
// This is used as a fallback when a project doesn't have its own template.
const DefaultFailureReportTemplate = `# Failure Report

**Agent:** {workspace-name}
**Issue:** {beads-id}
**Abandoned:** {timestamp}
**Reason:** {reason}

---

## Context

**Task:** {original-task}

**Skill:** {skill-name}

**What was attempted:**
[Brief description of what the agent was trying to do]

---

## Failure Summary

**Primary Cause:** {stuck | frozen | out-of-context | error | blocked | other}

**Details:**
[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]

---

## Progress Made

### Completed Steps
- [ ] [Step 1 - if any]
- [ ] [Step 2 - if any]

### Partial Progress
- [What was started but not finished]

### Artifacts Created
- [List any files created before abandonment]

---

## Learnings

**What worked:**
- [Things that went well before failure]

**What didn't work:**
- [Approaches that failed or caused issues]

**Root cause analysis:**
- [If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]

---

## Recovery Recommendations

**Can this be retried?** {yes | no | partially}

**If yes, what should be different:**
- [Suggestion 1 - different approach]
- [Suggestion 2 - smaller scope]
- [Suggestion 3 - additional context needed]

**If spawning a new agent:**
` + "```" + `
orch spawn {skill} "{adjusted-task}" --issue {beads-id}
` + "```" + `

**Context to provide:**
- [Key insight 1 for next agent]
- [Key insight 2 for next agent]
- [Pitfall to avoid]

---

## Session Metadata

**Workspace:** ` + "`" + `.orch/workspace/{workspace-name}/` + "`" + `
**Beads:** ` + "`" + `bd show {issue-id}` + "`" + `
**Time Spent:** {approximate-duration}
`

// GenerateServerContext creates the server context section for SPAWN_CONTEXT.md.
// Returns empty string if no servers are configured for the project.
func GenerateServerContext(projectDir string) string {
	cfg, err := config.Load(projectDir)
	if err != nil {
		return "" // No config or can't load - skip silently
	}

	if len(cfg.Servers) == 0 {
		return "" // No servers configured
	}

	// Get project name from directory
	projectName := filepath.Base(projectDir)
	sessionName := tmux.GetWorkersSessionName(projectName)

	// Check if servers are running
	running := tmux.SessionExists(sessionName)
	status := "stopped"
	if running {
		status = "running"
	}

	// Build server list
	var serverLines []string
	for service, port := range cfg.Servers {
		serverLines = append(serverLines, fmt.Sprintf("- **%s:** http://localhost:%d", service, port))
	}

	// Format the context section
	var sb strings.Builder
	sb.WriteString("## LOCAL SERVERS\n\n")
	sb.WriteString(fmt.Sprintf("**Project:** %s\n", projectName))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", status))
	sb.WriteString("**Ports:**\n")
	for _, line := range serverLines {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\n**Quick commands:**\n")
	sb.WriteString(fmt.Sprintf("- Start servers: `orch servers start %s`\n", projectName))
	sb.WriteString(fmt.Sprintf("- Stop servers: `orch servers stop %s`\n", projectName))
	sb.WriteString(fmt.Sprintf("- Open in browser: `orch servers open %s`\n", projectName))
	sb.WriteString("\n")

	return sb.String()
}

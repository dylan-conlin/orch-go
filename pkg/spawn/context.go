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

	"github.com/dylan-conlin/orch-go/pkg/action"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
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
{{if .EcosystemContext}}
## LOCAL PROJECT ECOSYSTEM

The following local projects are part of Dylan's orchestration ecosystem. These are LOCAL repositories on this machine - do NOT search GitHub for them.

{{.EcosystemContext}}
{{end}}
{{if .BehavioralPatterns}}
## BEHAVIORAL PATTERNS WARNING

The following patterns have been detected from prior agent sessions. These are futile actions that agents have repeatedly attempted without success:

{{.BehavioralPatterns}}

**Why this matters:** These patterns indicate files or targets that don't exist, commands that fail, or approaches that don't work. Avoid repeating these actions. If you need similar functionality, try alternative approaches or ask for clarification.
{{end}}
{{if .NoTrack}}
📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 PHASE REPORTING (WORKSPACE FILE):
Since bd comment is not available, report phase via workspace file:
` + "`echo 'Planning' > {{.WorkspacePath}}/.phase`" + `

Update this file at phase transitions:
- Planning → Implementing → Testing → Complete

This enables orchestrator visibility for untracked agents.

🚨 SESSION COMPLETE PROTOCOL:
When your work is done (all deliverables ready), complete in this EXACT order:
{{if eq .Tier "light"}}
1. Run: ` + "`echo 'Complete' > {{.WorkspacePath}}/.phase`" + ` (report phase FIRST - before commit)
2. Commit any final changes
3. Run: ` + "`/exit`" + ` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Run: ` + "`echo 'Complete' > {{.WorkspacePath}}/.phase`" + ` (report phase FIRST - before commit)
2. Ensure SYNTHESIS.md is created
3. Commit all changes (including SYNTHESIS.md)
4. Run: ` + "`/exit`" + ` to close the agent session
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
When your work is done (all deliverables ready), complete in this EXACT order:
{{if eq .Tier "light"}}
1. Run: ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary of deliverables]\"`" + ` (report phase FIRST - before commit)
2. Commit any final changes
3. Run: ` + "`/exit`" + ` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Run: ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary of deliverables]\"`" + ` (report phase FIRST - before commit)
2. Ensure SYNTHESIS.md is created
3. Commit all changes (including SYNTHESIS.md)
4. Run: ` + "`/exit`" + ` to close the agent session
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
{{if .RequiresInvestigationFile}}
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
4. **CHECK LINEAGE:** Before marking complete, run ` + "`kb context \"<your-topic>\"`" + ` to check if any prior investigation might be superseded by your work.
   - If yes: Fill the **Supersedes:** field in your investigation with the path to the prior artifact
   - Consider whether the prior investigation should have **Superseded-By:** updated (mention in completion comment)
5. Update Status: field when done (Active → Complete)
6. [Task-specific deliverables]
{{if ne .Tier "light"}}
7. **CREATE SYNTHESIS.md:** Before completing, create ` + "`SYNTHESIS.md`" + ` in your workspace: {{.ProjectDir}}/.orch/workspace/{{.WorkspaceName}}/SYNTHESIS.md
   - Use the template from: {{.ProjectDir}}/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.
{{else}}
7. ⚡ SYNTHESIS.md is NOT required (light tier spawn).
{{end}}
{{else}}
2. [Task-specific deliverables]
{{if ne .Tier "light"}}
3. **CREATE SYNTHESIS.md:** Before completing, create ` + "`SYNTHESIS.md`" + ` in your workspace: {{.ProjectDir}}/.orch/workspace/{{.WorkspaceName}}/SYNTHESIS.md
   - Use the template from: {{.ProjectDir}}/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.
{{else}}
3. ⚡ SYNTHESIS.md is NOT required (light tier spawn).
{{end}}
{{end}}

STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input

EXECUTION BEHAVIOR:
**Listing steps is NOT a stopping point.** If you write "Next steps:" or a numbered action list, execute them immediately. Do not wait for confirmation. Silent waiting is a bug - you are either working, explicitly blocked (Status: BLOCKED), or done (Phase: Complete).
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
{{if not .NoTrack}}
## VERIFICATION REQUIREMENTS (orch complete gates)

**Why this exists:** The orchestrator runs ` + "`orch complete`" + ` which verifies your work before closing the issue. These are the checks that WILL BLOCK completion if not satisfied. Capture evidence proactively.

### For ALL skills:

- **Phase: Complete** - Must report via beads comment (checked automatically)

### For code-producing skills (feature-impl, systematic-debugging, reliability-testing):

1. **Git commits exist** - At least one commit must exist since spawn time. No commits = work not done.

2. **Test execution evidence** - Beads comments must contain actual test output, NOT just claims.
   
   ✅ **Good** (quantifiable output):
   ` + "```bash" + `
   bd comment {{.BeadsID}} "Tests: go test ./pkg/... - PASS (12 tests in 0.8s)"
   bd comment {{.BeadsID}} "Tests: npm test - 15 passing, 0 failing"
   bd comment {{.BeadsID}} "Tests: pytest - 8 passed in 2.3s"
   ` + "```" + `
   
   ❌ **Bad** (vague claims that will be REJECTED):
   - "tests pass" / "all tests pass"
   - "verified tests pass"
   - "tests are passing"

3. **Visual verification (if web/ files modified)** - Screenshots or browser verification evidence required.
   ` + "```bash" + `
   bd comment {{.BeadsID}} "Visual verification: screenshot captured showing [description]"
   bd comment {{.BeadsID}} "Tested in browser - [what was verified]"
   ` + "```" + `
   
   Use Playwright MCP (` + "`browser_take_screenshot`" + `) or Glass tools (` + "`glass_screenshot`" + `) to capture evidence.

### Evidence capture timing:

- **Run tests** → Immediately capture output in beads comment
- **Modify web/ files** → Capture screenshot before moving to next task
- **Complete implementation** → Verify git status shows commits

**The test:** Before marking Phase: Complete, ask yourself: "Can orch complete find quantifiable evidence of my work?"
{{end}}

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
When your work is done (all deliverables ready), complete in this EXACT order:
{{if .NoTrack}}
{{if eq .Tier "light"}}
1. ` + "`echo 'Complete' > {{.WorkspacePath}}/.phase`" + ` (report phase FIRST - before commit)
2. Commit any final changes
3. ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. ` + "`echo 'Complete' > {{.WorkspacePath}}/.phase`" + ` (report phase FIRST - before commit)
2. Ensure SYNTHESIS.md is created
3. Commit all changes (including SYNTHESIS.md)
4. ` + "`/exit`" + `
{{end}}
{{else}}
{{if eq .Tier "light"}}
1. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + ` (report phase FIRST - before commit)
2. Commit any final changes
3. ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + ` (report phase FIRST - before commit)
2. Ensure SYNTHESIS.md is created
3. Commit all changes (including SYNTHESIS.md)
4. ` + "`/exit`" + `
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

	// Pattern to match "### Report via Beads" or similar section headers
	beadsSectionPattern := regexp.MustCompile(`(?i)^#+\s*(report\s+(via|to)\s+beads|beads\s+(progress\s+)?tracking)`)
	// Pattern to match the next section header (any heading)
	// Must start with 1-6 # followed by a space and an uppercase letter
	nextSectionPattern := regexp.MustCompile(`^#{1,6}\s+[A-Z]`)
	// Pattern to match completion criteria line with beads reporting
	beadsReportedPattern := regexp.MustCompile(`(?i)\*\*Reported\*\*.*bd\s+comment`)
	// Pattern to match lines with <beads-id> placeholder in code context
	beadsIDPattern := regexp.MustCompile(`bd\s+(comment|close|show)\s+<beads-id>`)

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
		if beadsSectionPattern.MatchString(line) {
			skipUntilNextSection = true
			inCodeBlockDuringSkip = false // Reset code block tracking
			continue
		}

		// Check if we've reached a new section (exit beads section)
		// But ONLY if we're not inside a code block
		if skipUntilNextSection && !inCodeBlockDuringSkip && nextSectionPattern.MatchString(line) && !beadsSectionPattern.MatchString(line) {
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
				if beadsIDPattern.MatchString(lines[j]) {
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
		if beadsReportedPattern.MatchString(line) {
			continue
		}

		// Skip lines that are just beads commands with <beads-id>
		if beadsIDPattern.MatchString(line) && strings.TrimSpace(line) != "" {
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
	multiNewline := regexp.MustCompile(`\n{3,}`)
	output = multiNewline.ReplaceAllString(output, "\n\n")

	return output
}

// contextData holds template data for SPAWN_CONTEXT.md.
type contextData struct {
	Task                     string
	BeadsID                  string
	ProjectDir               string
	WorkspaceName            string
	WorkspacePath            string // Full absolute path to workspace directory
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
	EcosystemContext         string // Local project ecosystem from ~/.orch/ECOSYSTEM.md
	BehavioralPatterns       string // Detected behavioral patterns from action-log.jsonl
	NoTrack                  bool   // When true, omit beads instructions from spawn context
	RequiresInvestigationFile bool   // When true, include investigation file setup instructions
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

	// Generate ecosystem context (auto-inject local project registry)
	// Only include for projects that are part of Dylan's orchestration ecosystem.
	// Use provided context if available, otherwise auto-generate from ~/.orch/ECOSYSTEM.md.
	ecosystemContext := cfg.EcosystemContext
	if ecosystemContext == "" && IsEcosystemRepo(cfg.Project) {
		ecosystemContext = GenerateEcosystemContext()
	}

	// Strip beads instructions from skill content when NoTrack is true
	// This prevents confusing agents with beads commands that won't work
	skillContent := cfg.SkillContent
	if cfg.NoTrack && skillContent != "" {
		skillContent = StripBeadsInstructions(skillContent)
	}

	// Generate behavioral patterns context (auto-inject detected patterns)
	// Use provided context if available, otherwise auto-generate from action-log.jsonl
	// Filter by project directory to prevent cross-project noise
	behavioralPatterns := cfg.BehavioralPatterns
	if behavioralPatterns == "" {
		behavioralPatterns = GenerateBehavioralPatternsContextForProject(cfg.WorkspaceName, cfg.ProjectDir)
	}

	// Compute full workspace path for template (needed for phase file in --no-track spawns)
	workspacePath := filepath.Join(cfg.ProjectDir, ".orch", "workspace", cfg.WorkspaceName)

	// Determine if this skill/phase combination requires an investigation file
	requiresInvestigationFile := RequiresInvestigationFile(cfg.SkillName, cfg.Phases)

	data := contextData{
		Task:                      cfg.Task,
		BeadsID:                   cfg.BeadsID,
		ProjectDir:                cfg.ProjectDir,
		WorkspaceName:             cfg.WorkspaceName,
		WorkspacePath:             workspacePath,
		SkillName:                 cfg.SkillName,
		SkillContent:              skillContent,
		InvestigationSlug:         slug,
		Phases:                    cfg.Phases,
		Mode:                      cfg.Mode,
		Validation:                cfg.Validation,
		InvestigationType:         cfg.InvestigationType,
		KBContext:                 cfg.KBContext,
		Tier:                      cfg.Tier,
		ServerContext:             serverContext,
		EcosystemContext:          ecosystemContext,
		BehavioralPatterns:        behavioralPatterns,
		NoTrack:                   cfg.NoTrack,
		RequiresInvestigationFile: requiresInvestigationFile,
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

	// Log spawn telemetry to events.jsonl for observability
	// This captures context size, kb context stats, and other spawn metrics
	telemetry := CollectSpawnTelemetry(cfg, content, nil)
	if err := LogSpawnTelemetry(DefaultTelemetryLogPath(), telemetry); err != nil {
		// Don't fail the spawn on telemetry error - just warn
		// Telemetry is observability, not critical path
		fmt.Fprintf(os.Stderr, "Warning: failed to log spawn telemetry: %v\n", err)
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

### Externalized via ` + "`kb quick`" + `
- ` + "`kb quick decide \"X\" --reason \"Y\"`" + ` - [if applicable]
- ` + "`kb quick constrain \"X\" --reason \"Y\"`" + ` - [if applicable]
- ` + "`kb quick tried \"X\" --failed \"Y\"`" + ` - [if applicable]

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

// NOTE: EcosystemFilePath, GenerateEcosystemContext, ExtractQuickReference,
// and IsEcosystemRepo are now defined in ecosystem.go

// GenerateBehavioralPatternsContext loads action patterns from action-log.jsonl
// and formats them for inclusion in SPAWN_CONTEXT.md. This surfaces futile action
// patterns to agents so they don't repeat the same mistakes.
//
// Returns empty string if no patterns are detected or if loading fails.
// Patterns are filtered to those relevant to the current workspace/session.
// Note: Use GenerateBehavioralPatternsContextForProject for project-filtered patterns.
func GenerateBehavioralPatternsContext(workspaceName string) string {
	return GenerateBehavioralPatternsContextForProject(workspaceName, "")
}

// GenerateBehavioralPatternsContextForProject loads action patterns from action-log.jsonl
// filtered to a specific project directory. This prevents cross-project noise by only
// showing patterns relevant to the project being worked on.
//
// If projectDir is empty, returns all patterns (global view).
func GenerateBehavioralPatternsContextForProject(workspaceName, projectDir string) string {
	tracker, err := action.LoadTracker("")
	if err != nil {
		return "" // Fail silently - don't block spawn on action log issues
	}

	// Filter events by project directory if specified
	var filteredEvents []action.ActionEvent
	if projectDir != "" {
		for _, e := range tracker.Events {
			// Check if target path or workspace path is within project directory
			if strings.HasPrefix(e.Target, projectDir) || (e.Workspace != "" && strings.HasPrefix(e.Workspace, projectDir)) {
				filteredEvents = append(filteredEvents, e)
			}
		}
		// If no project-specific patterns, fall back to global
		if len(filteredEvents) == 0 {
			filteredEvents = tracker.Events
		}
		tracker = &action.Tracker{Events: filteredEvents}
	}

	patterns := tracker.FindPatterns()
	if len(patterns) == 0 {
		return "" // No patterns detected
	}

	var sb strings.Builder

	// Limit to top 5 most frequent patterns to avoid context bloat
	maxPatterns := 5
	if len(patterns) < maxPatterns {
		maxPatterns = len(patterns)
	}

	for i := 0; i < maxPatterns; i++ {
		p := patterns[i]

		// Format icon based on severity
		icon := "⚠️"
		if p.Count >= 5 {
			icon = "🚫" // Strongly discourage
		}

		// Format the pattern warning
		outcomeDesc := ""
		switch p.Outcome {
		case action.OutcomeEmpty:
			outcomeDesc = "returns empty"
		case action.OutcomeError:
			outcomeDesc = "fails with error"
		case action.OutcomeFallback:
			outcomeDesc = "requires fallback"
		default:
			outcomeDesc = "fails"
		}

		sb.WriteString(fmt.Sprintf("%s **%s** on `%s` %s (%dx in past week)\n",
			icon, p.Tool, p.Target, outcomeDesc, p.Count))
	}

	if len(patterns) > 5 {
		sb.WriteString(fmt.Sprintf("\n... and %d more patterns (run `orch patterns` to see all)\n", len(patterns)-5))
	}

	return sb.String()
}

// GenerateBehavioralPatternsContextForWorkspace loads action patterns filtered
// to a specific workspace. This is useful for showing patterns relevant to
// the current agent's work context.
func GenerateBehavioralPatternsContextForWorkspace(workspaceName string) string {
	tracker, err := action.LoadTracker("")
	if err != nil {
		return ""
	}

	// Filter events to those matching this workspace
	var workspaceEvents []action.ActionEvent
	for _, e := range tracker.Events {
		if e.Workspace != "" && strings.Contains(e.Workspace, workspaceName) {
			workspaceEvents = append(workspaceEvents, e)
		}
	}

	if len(workspaceEvents) == 0 {
		// Fall back to global patterns if no workspace-specific ones
		return GenerateBehavioralPatternsContext(workspaceName)
	}

	// Create a tracker with filtered events and find patterns
	filteredTracker := &action.Tracker{Events: workspaceEvents}
	patterns := filteredTracker.FindPatterns()

	if len(patterns) == 0 {
		// Fall back to global patterns
		return GenerateBehavioralPatternsContext(workspaceName)
	}

	var sb strings.Builder
	sb.WriteString("**Patterns specific to similar workspaces:**\n\n")

	maxPatterns := 3
	if len(patterns) < maxPatterns {
		maxPatterns = len(patterns)
	}

	for i := 0; i < maxPatterns; i++ {
		p := patterns[i]
		outcomeDesc := ""
		switch p.Outcome {
		case action.OutcomeEmpty:
			outcomeDesc = "returns empty"
		case action.OutcomeError:
			outcomeDesc = "fails"
		case action.OutcomeFallback:
			outcomeDesc = "requires fallback"
		default:
			outcomeDesc = "fails"
		}

		sb.WriteString(fmt.Sprintf("- **%s** on `%s` %s (%dx)\n", p.Tool, p.Target, outcomeDesc, p.Count))
	}

	return sb.String()
}

// FailureReportStatus represents the result of checking a FAILURE_REPORT.md file.
type FailureReportStatus struct {
	Exists              bool     // Whether a FAILURE_REPORT.md exists
	FilePath            string   // Path to the FAILURE_REPORT.md file
	WorkspaceName       string   // Name of the workspace containing the report
	IsFilled            bool     // Whether key sections have been filled out
	UnfilledSections    []string // Names of sections that still have placeholders
	WhatWasAttempted    bool     // Whether "What was attempted" has been filled
	Details             bool     // Whether "Details" under Failure Summary has been filled
	RootCauseAnalysis   bool     // Whether "Root cause analysis" has been filled
	WhatShouldDifferent bool     // Whether "If yes, what should be different" has been filled
}

// Placeholders that indicate unfilled sections in FAILURE_REPORT.md.
// These are the template placeholders that should be replaced with actual content.
var failureReportPlaceholders = []string{
	"[Brief description of what the agent was trying to do]",
	"[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]",
	"[If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]",
	"[Suggestion 1 - different approach]",
}

// CheckFailureReport checks if a FAILURE_REPORT.md exists for the given beads ID
// and whether it has been filled out. Returns a status struct describing the state.
//
// This implements the "Gate Over Remind" principle: we don't just remind that the
// file exists, we gate respawning on it being filled out properly.
func CheckFailureReport(projectDir, beadsID string) *FailureReportStatus {
	status := &FailureReportStatus{}

	// Find workspace directory for this beads ID
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return status // No workspace directory, no failure report
	}

	// Look for FAILURE_REPORT.md in any workspace that was spawned for this beads ID
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspacePath := filepath.Join(workspaceDir, entry.Name())
		reportPath := filepath.Join(workspacePath, "FAILURE_REPORT.md")

		// Check if FAILURE_REPORT.md exists
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			continue
		}

		// Found a FAILURE_REPORT.md - check if it's for this beads ID
		// First check the SPAWN_CONTEXT.md to confirm this workspace was for this issue
		spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
		if content, err := os.ReadFile(spawnContextPath); err == nil {
			contentStr := string(content)
			// Look for "spawned from beads issue: **{beadsID}**" or "beads issue: **{beadsID}**"
			if !strings.Contains(contentStr, beadsID) {
				continue // This workspace is for a different issue
			}
		} else {
			continue // Can't read SPAWN_CONTEXT.md, skip this workspace
		}

		// This workspace was for our beads ID and has a FAILURE_REPORT.md
		status.Exists = true
		status.FilePath = reportPath
		status.WorkspaceName = entry.Name()

		// Read and analyze the failure report
		content, err := os.ReadFile(reportPath)
		if err != nil {
			return status // Exists but can't read - report as unfilled
		}

		contentStr := string(content)
		status.IsFilled = true // Assume filled until we find placeholders

		// Check each key section for placeholder content
		for _, placeholder := range failureReportPlaceholders {
			if strings.Contains(contentStr, placeholder) {
				status.IsFilled = false

				// Determine which section this placeholder belongs to
				switch placeholder {
				case "[Brief description of what the agent was trying to do]":
					status.UnfilledSections = append(status.UnfilledSections, "What was attempted")
				case "[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]":
					status.UnfilledSections = append(status.UnfilledSections, "Details")
				case "[If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]":
					status.UnfilledSections = append(status.UnfilledSections, "Root cause analysis")
				case "[Suggestion 1 - different approach]":
					status.UnfilledSections = append(status.UnfilledSections, "What should be different")
				}
			}
		}

		// Set individual field flags
		status.WhatWasAttempted = !strings.Contains(contentStr, "[Brief description of what the agent was trying to do]")
		status.Details = !strings.Contains(contentStr, "[Describe what went wrong - symptoms observed, errors encountered, or why the agent was stuck]")
		status.RootCauseAnalysis = !strings.Contains(contentStr, "[If known, why did this fail? External dependency? Tool issue? Scope creep? Context exhaustion?]")
		status.WhatShouldDifferent = !strings.Contains(contentStr, "[Suggestion 1 - different approach]")

		// We found the failure report for this issue - return status
		return status
	}

	return status // No failure report found
}

// FormatFailureReportGateError formats a user-friendly error message when
// the failure report gate blocks a spawn.
func FormatFailureReportGateError(status *FailureReportStatus, beadsID string) string {
	var sb strings.Builder

	sb.WriteString("⛔ FAILURE_REPORT.md has unfilled sections\n\n")
	sb.WriteString("Before respawning, fill out the failure report to capture learning:\n\n")
	sb.WriteString(fmt.Sprintf("  File: %s\n\n", status.FilePath))
	sb.WriteString("  Unfilled sections:\n")
	for _, section := range status.UnfilledSections {
		sb.WriteString(fmt.Sprintf("    - %s\n", section))
	}
	sb.WriteString("\nWhy this gate exists:\n")
	sb.WriteString("  • Prevents repeating the same failures\n")
	sb.WriteString("  • Captures context for the next agent\n")
	sb.WriteString("  • Creates institutional memory about what didn't work\n")
	sb.WriteString("\nTo bypass (not recommended):\n")
	sb.WriteString(fmt.Sprintf("  orch spawn --skip-failure-review --issue %s ...\n", beadsID))

	return sb.String()
}

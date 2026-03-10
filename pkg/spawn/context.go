package spawn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// getGitBaseline returns the current git commit SHA for the project directory.
// Returns empty string if not in a git repository or if git command fails.
// This is used as the baseline for git-based change detection during verification.
func getGitBaseline(projectDir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		// Not in a git repo or git command failed - return empty
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Pre-compiled regex patterns for context.go
var (
	regexBeadsSectionHeader    = regexp.MustCompile(`(?i)^#+\s*(report\s+(via|to)\s+beads|beads\s+(progress\s+)?tracking)`)
	regexNextSectionHeader     = regexp.MustCompile(`^#{1,6}\s+[A-Z]`)
	regexBeadsReportedCriteria = regexp.MustCompile(`(?i)\*\*Reported\*\*.*bd\s+comment`)
	regexBeadsIDPlaceholder    = regexp.MustCompile(`bd\s+(comment|close|show)\s+<beads-id>`)
	regexMultiNewline          = regexp.MustCompile(`\n{3,}`)
	regexSessionScope          = regexp.MustCompile(`(?mi)^\s*session\s+scope:\s*([^\r\n]+)`)
	// regexBeadsIDInText matches beads-like IDs in text: project-prefix followed by hyphen and digits.
	// Examples: pw-8972, orch-go-1141, pw-123
	regexBeadsIDInText = regexp.MustCompile(`\b([a-z][\w-]*-\d+)\b`)
)

// ParseScopeFromTask extracts a session scope value from a task description.
// Looks for patterns like "SESSION SCOPE: Small" or "Session scope: Large".
// Returns the lowercase first word of the scope value (e.g., "small", "medium", "large"),
// or empty string if no scope is found.
func ParseScopeFromTask(task string) string {
	matches := regexSessionScope.FindStringSubmatch(task)
	if len(matches) < 2 {
		return ""
	}
	scope := strings.TrimSpace(strings.ToLower(matches[1]))
	if scope == "" {
		return ""
	}
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// ResolveScope determines the session scope for a spawn.
// Priority: explicit scope parameter > parsed from task > default "medium".
func ResolveScope(explicitScope, task string) string {
	if explicitScope != "" {
		return strings.ToLower(explicitScope)
	}
	if parsed := ParseScopeFromTask(task); parsed != "" {
		return parsed
	}
	return ScopeMedium
}

// CreateScreenshotsDir creates the screenshots/ subdirectory in a workspace.
// This directory is for agent-produced visual artifacts (e.g., UI screenshots for verification).
func CreateScreenshotsDir(workspacePath string) error {
	screenshotsPath := filepath.Join(workspacePath, "screenshots")
	if err := os.MkdirAll(screenshotsPath, 0755); err != nil {
		return fmt.Errorf("failed to create screenshots directory: %w", err)
	}
	return nil
}

// SpawnContextTemplate is the basic structure for SPAWN_CONTEXT.md.
// This is a simplified version of the Python template.
const SpawnContextTemplate = `TASK: {{.Task}}
{{if .OrientationFrame}}
ORIENTATION_FRAME: {{.OrientationFrame}}
{{end}}
{{- if .IntentType}}
INTENT_TYPE: {{.IntentType}}
{{end}}
{{if .DesignWorkspace}}
## DESIGN REFERENCE

This feature implementation is based on an approved design from: **{{.DesignWorkspace}}**
{{if .DesignMockupPath}}
**Mockup:** {{.DesignMockupPath}}
{{end}}
{{if .DesignPromptPath}}
**Design Prompt:** {{.DesignPromptPath}}
{{end}}
{{if .DesignNotes}}
**Design Notes:**
{{.DesignNotes}}
{{end}}

Use these design artifacts as the source of truth for UI layout, styling, and user experience.

{{end}}
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
{{if .ConfigResolution}}
## CONFIG RESOLUTION

{{.ConfigResolution}}

{{end}}
{{if .ReworkFeedback}}
## 🔄 REWORK CONTEXT (Attempt #{{.ReworkNumber}})

**This is rework** — a prior agent attempted this task but the work was insufficient.

### What Was Wrong
{{.ReworkFeedback}}

### Prior Attempt Summary
{{.PriorSynthesis}}

### Prior Workspace
Full prior artifacts at: {{.PriorWorkspace}}

### Rework Instructions
1. Read the feedback above carefully
2. Read the prior SYNTHESIS.md for full context on what was tried
3. Focus specifically on the identified gaps
4. Do NOT re-do work that was correct — build on it
5. Report via ` + "`bd comment {{.BeadsID}} \"Phase: Planning - Rework #{{.ReworkNumber}}: [brief plan]\"`" + `

{{end}}
{{if .PriorCompletions}}
{{.PriorCompletions}}
{{end}}
{{if .KBContext}}
{{.KBContext}}
{{end}}
{{if .ClusterSummary}}
{{.ClusterSummary}}
{{end}}
{{if .HotspotArea}}
## HOTSPOT AREA WARNING

⚠️ This task targets files in a **hotspot area** (high churn, complexity, or coupling).
{{if .HotspotFiles}}
**Hotspot files:**
{{range .HotspotFiles}}- ` + "`{{.}}`" + `
{{end}}{{end}}{{if .HotspotDefectClasses}}
**Likely defect classes (watch for these patterns):**
{{range .HotspotDefectClasses}}- {{.}}
{{end}}
*Reference: .kb/models/defect-class-taxonomy/model.md*
{{end}}
**Investigation routing:** If your findings affect these files, recommend ` + "`architect`" + ` follow-up instead of direct ` + "`feature-impl`" + `. Hotspot areas require architectural review before implementation changes.
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
{{else}}
🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. (Allowed) Read this SPAWN_CONTEXT.md file (your first tool call may be this read)
2. Immediately report via ` + "`bd comment {{.BeadsID}} \"Phase: Planning - [brief description]\"`" + `
3. Read relevant codebase context for your task and begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.
{{end}}

{{if not .NoTrack}}
VERIFICATION REQUIREMENTS (ORCH COMPLETE):
Your work is verified in two human gates before closing:
- Gate 1 (explain-back): orchestrator must explain what was built and why.
- Gate 2 (behavioral, Tier 1 only): orchestrator confirms behavior is verified.
Provide clear Phase: Complete summary and VERIFICATION_SPEC.yaml evidence to support both gates.
{{end}}

CONTEXT: [See task description]

PROJECT_DIR: {{.ProjectDir}}

{{if eq .Scope "small"}}SESSION SCOPE: Small (text edits only)
- Lightweight session - focus on precise, targeted changes
{{else if eq .Scope "large"}}SESSION SCOPE: Large (estimated [4-6h+])
- Extended session
- Recommend checkpoint after Phase 1 if session exceeds 3 hours
{{else}}SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours
{{end}}


AUTHORITY:
Authority delegation rules are provided via skill guidance (worker-base skill).
**Full criteria:** See ` + "`.kb/guides/decision-authority.md`" + ` for the complete decision tree and examples.

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
{{if .ProducesInvestigation}}
{{if .HasInjectedModels}}
2. **SET UP probe file:** This is confirmatory work against an existing model.
   - Model content was injected in PRIOR KNOWLEDGE section above
   - Create probe file in model's probes/ directory (use the absolute path from the ` + "`See:`" + ` reference)
   - Use probe template structure: Question, What I Tested, What I Observed, Model Impact
   - Your probe should confirm, contradict, or extend the model's claims
{{if .CrossRepoModelDir}}
   - ⚠️ **CROSS-REPO MODEL:** The model lives in ` + "`{{.CrossRepoModelDir}}`" + `, NOT in your working directory (` + "`{{.ProjectDir}}`" + `).
     Create the probe file using the **absolute path** from the model's ` + "`See:`" + ` reference in PRIOR KNOWLEDGE above.
     Your workspace and SYNTHESIS.md remain in {{.ProjectDir}}.
{{end}}
{{if not .NoTrack}}
   - **IMPORTANT:** After creating probe file, report the **absolute** path via:
     ` + "`bd comment {{.BeadsID}} \"probe_path: /path/to/probe.md\"`" + `
{{end}}
{{else}}
2. **SET UP investigation file:** Run ` + "`kb create investigation {{.InvestigationSlug}} --model <model-name>`" + ` to create from template (or ` + "`--orphan`" + ` if no model applies)
   - This creates: ` + "`.kb/investigations/simple/YYYY-MM-DD-{{.InvestigationSlug}}.md`" + `
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
{{if not .NoTrack}}
   - **IMPORTANT:** After running ` + "`kb create`" + `, report the actual path via:
     ` + "`bd comment {{.BeadsID}} \"investigation_path: /path/to/file.md\"`" + `
     (This allows orch complete to verify the correct file)
{{end}}
{{end}}
{{if .HasInjectedModels}}
3. **UPDATE probe file** as you work:
   - Question: What model claim are you testing?
   - What I Tested: Actual command/code run (not just code review)
   - What I Observed: Actual output/behavior
   - Model Impact: Confirms/contradicts/extends which invariant
{{else}}
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
{{end}}
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
{{else}}
2. [Task-specific deliverables]
{{end}}
{{if ne .Tier "light"}}
{{if .ProducesInvestigation}}6{{else}}3{{end}}. **CREATE SYNTHESIS.md:** Before completing, create ` + "`SYNTHESIS.md`" + ` in your workspace: {{.ProjectDir}}/.orch/workspace/{{.WorkspaceName}}/SYNTHESIS.md
   - Use the template from: {{.ProjectDir}}/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.
{{else}}
{{if .ProducesInvestigation}}6{{else}}3{{end}}. ⚡ SYNTHESIS.md is NOT required (light tier spawn).
{{end}}

STATUS UPDATES:
{{if .ProducesInvestigation}}{{if .HasInjectedModels}}Update Status: field in your probe file:
- Status: Active (while working)
- Status: Complete (when done and committed)

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to probe file
- Add '**Status:** QUESTION - [question]' when needing input
{{else}}Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed)

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input
{{end}}{{else}}Track progress via beads comments.
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
{{if .BrowserAutomation}}
## BROWSER AUTOMATION (playwright-cli)

You have **playwright-cli** available for browser automation. Use it via Bash commands (NOT MCP).

**Quick reference:**
` + "```" + `bash
playwright-cli open <url>              # Open browser (daemon — stays running)
playwright-cli screenshot --filename /tmp/screenshot.png  # Take screenshot
playwright-cli click <ref>             # Click element by accessibility ref
playwright-cli fill <ref> "text"       # Fill input field
playwright-cli hover <ref>             # Hover over element
playwright-cli select <ref> "value"    # Select dropdown option
playwright-cli goto <url>              # Navigate to URL
playwright-cli eval "document.title"   # Run simple JS expression
playwright-cli console                 # View console logs
playwright-cli network                 # View network requests
playwright-cli close                   # Close browser session
` + "```" + `

**Key patterns:**
- **Snapshots:** Element refs (e3, e12) are in ` + "`.playwright-cli/page-*.yml`" + ` files — read them to get refs for interaction
- **Sessions:** Use ` + "`-s=name`" + ` for named sessions (e.g., ` + "`playwright-cli -s=dashboard open http://localhost:5188`" + `)
- **Speed:** First command ~1.7s (browser startup), subsequent commands ~0.15s (daemon IPC)
- **Screenshots:** Use ` + "`--full-page`" + ` flag for full page capture

**For complex JS:** Use ` + "`playwright-cli run-code \"async page => { ... }\"`" + ` instead of ` + "`eval`" + `.
{{end}}
🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.

{{if .NoTrack}}
{{if eq .Tier "light"}}
1. **COMMIT YOUR WORK:** ` + "`git add <files you changed> && git commit -m \"feat: [description]\"`" + `
2. **Session complete** — no further actions needed.

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. Create SYNTHESIS.md in your workspace
2. **COMMIT YOUR WORK:** ` + "`git add <files you changed> && git commit -m \"feat: [description]\"`" + `
3. **Session complete** — no further actions needed.
{{end}}
{{else}}
{{if eq .Tier "light"}}
1. **COMMIT YOUR WORK:** ` + "`git add <files you changed> && git commit -m \"feat: [description] ({{.BeadsID}})\"`" + `
2. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + `
3. **Session complete** — no further actions needed.

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
1. Create SYNTHESIS.md in your workspace
2. **COMMIT YOUR WORK:** ` + "`git add <files you changed> && git commit -m \"feat: [description] ({{.BeadsID}})\"`" + `
3. ` + "`bd comment {{.BeadsID}} \"Phase: Complete - [1-2 sentence summary]\"`" + `
4. **Session complete** — no further actions needed.
{{end}}
{{end}}

⛔ **NEVER run ` + "`git push`" + `** - Workers commit locally only.
{{if not .NoTrack}}⛔ **NEVER run ` + "`bd close`" + `** - Only the orchestrator closes issues via ` + "`orch complete`" + `.
{{end}}⚠️ Your work is NOT complete until Phase: Complete is reported (or all steps above are done for --no-track).
`

// skillContentData holds the data context for processing skill content templates.
// Skill content files (SKILL.md) can contain Go template variables that need to be
// replaced with spawn-specific values before injection into SPAWN_CONTEXT.md.
type skillContentData struct {
	BeadsID string // The beads issue ID for progress tracking
	Tier    string // Spawn tier: "light" or "full"
}

// ProcessSkillContentTemplate processes Go template variables in skill content.
// Skill content (from SKILL.md files) may contain template variables like {{.BeadsID}}
// and conditionals like {{if eq .Tier "light"}}. This function processes those
// templates using the spawn-specific data context before the skill content is
// injected into SPAWN_CONTEXT.md.
//
// If template parsing or execution fails, returns the original content unchanged
// (fail-open behavior to avoid breaking spawns for minor template issues).
func ProcessSkillContentTemplate(content string, beadsID string, tier string) string {
	if content == "" {
		return content
	}

	// Quick check: if content doesn't contain template syntax, skip processing
	if !strings.Contains(content, "{{") {
		return content
	}

	tmpl, err := template.New("skill_content").Parse(content)
	if err != nil {
		// Template parse error - return original content
		// This can happen if skill content has malformed templates
		return content
	}

	data := skillContentData{
		BeadsID: beadsID,
		Tier:    tier,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Template execution error - return original content
		return content
	}

	return buf.String()
}

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

// WriteSkillPromptFile writes compiled skill content to SKILL_PROMPT.md in the workspace directory.
// This file is used by --append-system-prompt "$(cat SKILL_PROMPT.md)" for system-level injection.
// The file path is set on cfg.SystemPromptFile for use by BuildClaudeLaunchCommand.
// Returns nil if cfg.SkillContent is empty (no skill to write).
func WriteSkillPromptFile(cfg *Config) error {
	if cfg.SkillContent == "" {
		return nil
	}

	skillContent := cfg.SkillContent
	// Strip beads instructions when NoTrack is true
	if cfg.NoTrack {
		skillContent = StripBeadsInstructions(skillContent)
	}
	// Process template variables (e.g., {{.BeadsID}})
	skillContent = ProcessSkillContentTemplate(skillContent, cfg.BeadsID, cfg.Tier)

	promptPath := filepath.Join(cfg.WorkspacePath(), "SKILL_PROMPT.md")
	if err := os.WriteFile(promptPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("failed to write SKILL_PROMPT.md: %w", err)
	}

	cfg.SystemPromptFile = promptPath
	return nil
}

// contextData holds template data for SPAWN_CONTEXT.md.
type contextData struct {
	Task                  string
	BeadsID               string
	ProjectDir            string
	WorkspaceName         string
	SkillName             string
	SkillContent          string
	InvestigationSlug     string
	ProducesInvestigation bool
	HasInjectedModels     bool   // When true, this spawn has model content injected for probing
	CrossRepoModelDir     string // When non-empty, model lives in a different repo than ProjectDir
	Phases                string
	Mode                  string
	Validation            string
	InvestigationType     string
	KBContext             string
	ClusterSummary        string // Area awareness: cluster summary from orch tree --cluster <area> --format summary
	ConfigResolution      string
	Tier                  string
	Scope                 string // Session scope: "small", "medium", or "large"
	ServerContext         string
	NoTrack               bool   // When true, omit beads instructions from spawn context
	IsBug                 bool   // When true, this is a bug issue with reproduction info
	ReproSteps            string // Reproduction steps from bug issue
	ReworkFeedback        string // Rework instructions from orchestrator
	ReworkNumber          int    // Rework attempt number
	PriorSynthesis        string // TLDR + Delta from prior SYNTHESIS.md
	PriorWorkspace        string // Path to archived prior workspace
	HotspotArea           bool     // Task targets a hotspot area
	HotspotFiles          []string // Files identified as hotspots
	HotspotDefectClasses  []string // Defect classes likely for the hotspot area
	DesignWorkspace       string   // Design workspace name for ui-design-session handoff
	DesignMockupPath      string   // Path to approved mockup
	DesignPromptPath      string   // Path to design prompt
	DesignNotes           string   // Notes from design session
	OrientationFrame      string   // Additional task context (from issue description), rendered as separate section
	IntentType            string   // Orchestrator's declared outcome type (experience, produce, compare, etc.)
	PriorCompletions      string   // Prior completed agent work on same issue
	BrowserAutomation     bool     // When true, playwright-cli browser automation is available
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

	// When SystemPromptFile is set, skill content is injected at system prompt level
	// via --append-system-prompt. Omit it from SPAWN_CONTEXT.md to prevent double-loading.
	var skillContent string
	if cfg.SystemPromptFile == "" {
		// User-level injection: embed skill content in SPAWN_CONTEXT.md (current behavior)
		skillContent = cfg.SkillContent
		if cfg.NoTrack && skillContent != "" {
			skillContent = StripBeadsInstructions(skillContent)
		}
		if skillContent != "" {
			skillContent = ProcessSkillContentTemplate(skillContent, cfg.BeadsID, cfg.Tier)
		}
	}
	// When SystemPromptFile is set, skillContent remains empty — skill content
	// is already written to SKILL_PROMPT.md and will be injected via CLI flag

	// Generate cluster summary for area awareness
	// Detect area from task description or beads issue labels
	clusterSummary := ""
	if detectedArea := DetectAreaFromTask(cfg.Task, cfg.BeadsID, cfg.ProjectDir); detectedArea != "" {
		if summary := GetClusterSummary(detectedArea, cfg.ProjectDir); summary != "" {
			clusterSummary = fmt.Sprintf("\n## AREA CONTEXT: %s\n\n%s\n", detectedArea, summary)
		}
	}

	data := contextData{
		Task:                  cfg.Task,
		BeadsID:               cfg.BeadsID,
		ProjectDir:            cfg.ProjectDir,
		WorkspaceName:         cfg.WorkspaceName,
		SkillName:             cfg.SkillName,
		SkillContent:          skillContent,
		InvestigationSlug:     slug,
		ProducesInvestigation: DefaultProducesInvestigationForSkill(cfg.SkillName, cfg.Phases),
		HasInjectedModels:     cfg.HasInjectedModels,
		CrossRepoModelDir:     cfg.CrossRepoModelDir,
		Phases:                cfg.Phases,
		Mode:                  cfg.Mode,
		Validation:            cfg.Validation,
		InvestigationType:     cfg.InvestigationType,
		KBContext:             cfg.KBContext,
		ClusterSummary:        clusterSummary,
		ConfigResolution:      FormatResolvedSpawnSettings(cfg.ResolvedSettings),
		Tier:                  cfg.Tier,
		Scope:                 ResolveScope(cfg.Scope, cfg.Task),
		ServerContext:         serverContext,
		NoTrack:               cfg.NoTrack,
		IsBug:                 cfg.IsBug,
		ReproSteps:            cfg.ReproSteps,
		ReworkFeedback:        cfg.ReworkFeedback,
		ReworkNumber:          cfg.ReworkNumber,
		PriorSynthesis:        cfg.PriorSynthesis,
		PriorWorkspace:        cfg.PriorWorkspace,
		HotspotArea:           cfg.HotspotArea,
		HotspotFiles:          cfg.HotspotFiles,
		HotspotDefectClasses:  cfg.HotspotDefectClasses,
		DesignWorkspace:       cfg.DesignWorkspace,
		DesignMockupPath:      cfg.DesignMockupPath,
		DesignPromptPath:      cfg.DesignPromptPath,
		DesignNotes:           cfg.DesignNotes,
		OrientationFrame:      cfg.OrientationFrame,
		IntentType:            cfg.IntentType,
		PriorCompletions:      cfg.PriorCompletions,
		BrowserAutomation:     cfg.BrowserTool == "playwright-cli",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// WriteContext writes the SPAWN_CONTEXT.md file to the workspace.
// For orchestrator-type skills (IsOrchestrator=true), it delegates to
// WriteOrchestratorContext which generates ORCHESTRATOR_CONTEXT.md instead.
// For meta-orchestrator skills (IsMetaOrchestrator=true), it delegates to
// WriteMetaOrchestratorContext which generates META_ORCHESTRATOR_CONTEXT.md.
func WriteContext(cfg *Config) error {
	// Route meta-orchestrator spawns to dedicated template (check first, more specific)
	if cfg.IsMetaOrchestrator {
		return WriteMetaOrchestratorContext(cfg)
	}

	// Route orchestrator spawns to dedicated template
	if cfg.IsOrchestrator {
		return WriteOrchestratorContext(cfg)
	}

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

	// Ensure PROBE.md template exists in the project (for probe-type spawns)
	if cfg.HasInjectedModels {
		if err := EnsureProbeTemplate(cfg.ProjectDir); err != nil {
			return fmt.Errorf("failed to ensure probe template: %w", err)
		}
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

	// Write beads ID file for workspace lookup during orch complete
	if cfg.BeadsID != "" {
		beadsIDPath := filepath.Join(workspacePath, ".beads_id")
		if err := os.WriteFile(beadsIDPath, []byte(cfg.BeadsID), 0644); err != nil {
			return fmt.Errorf("failed to write beads ID file: %w", err)
		}
	}

	// Write spawn mode file for orch complete to be mode-aware
	if cfg.SpawnMode != "" {
		spawnModePath := filepath.Join(workspacePath, ".spawn_mode")
		if err := os.WriteFile(spawnModePath, []byte(cfg.SpawnMode), 0644); err != nil {
			return fmt.Errorf("failed to write spawn mode file: %w", err)
		}
	}

	// Write agent manifest JSON for canonical agent identity and spawn-time metadata
	// This provides a single source of truth for git-based scoping and verification gates
	spawnTime := time.Now()
	manifest := AgentManifest{
		WorkspaceName: cfg.WorkspaceName,
		Skill:         cfg.SkillName,
		BeadsID:       cfg.BeadsID,
		ProjectDir:    cfg.ProjectDir,
		GitBaseline:   getGitBaseline(cfg.ProjectDir),
		SpawnTime:     spawnTime.Format(time.RFC3339),
		Tier:          cfg.Tier,
		SpawnMode:     cfg.SpawnMode,
		Model:         cfg.Model,
		VerifyLevel:   cfg.VerifyLevel,
		ReviewTier:    cfg.ReviewTier,
	}
	if err := WriteAgentManifest(workspacePath, manifest); err != nil {
		return fmt.Errorf("failed to write agent manifest: %w", err)
	}

	// Write prior workspace reference for rework spawns (if provided)
	if cfg.PriorWorkspace != "" {
		priorWorkspacePath := filepath.Join(workspacePath, ".prior_workspace")
		content := cfg.PriorWorkspace
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := os.WriteFile(priorWorkspacePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write prior workspace file: %w", err)
		}
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

// EnsureProbeTemplate ensures the PROBE.md template exists in the project.
// If the project doesn't have .orch/templates/PROBE.md, it creates one from
// the DefaultProbeTemplate in probes.go.
func EnsureProbeTemplate(projectDir string) error {
	templatesDir := filepath.Join(projectDir, ".orch", "templates")
	templatePath := filepath.Join(templatesDir, "PROBE.md")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template exists, nothing to do
	}

	// Create templates directory if needed
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Write the default template
	if err := os.WriteFile(templatePath, []byte(DefaultProbeTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write probe template: %w", err)
	}

	return nil
}

// extractProjectPrefix extracts the project prefix from a beads ID.
// Given "pw-8972", returns "pw". Given "orch-go-1141", returns "orch-go".
// The prefix is everything before the final hyphen-number sequence.
func extractProjectPrefix(beadsID string) string {
	// Find the last occurrence of -<digits> and take everything before it
	for i := len(beadsID) - 1; i >= 0; i-- {
		if beadsID[i] == '-' {
			// Check if everything after the hyphen is digits
			suffix := beadsID[i+1:]
			allDigits := true
			for _, c := range suffix {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits && len(suffix) > 0 {
				return beadsID[:i]
			}
		}
	}
	return beadsID
}

// ValidateBeadsIDConsistency checks if the task text references a beads ID
// from the same project that differs from the tracking beads ID.
// Returns a warning message if a mismatch is detected, empty string otherwise.
//
// This catches a class of spawn bugs where the task description references
// one issue (e.g., "fix pw-8972") but the --issue flag tracks a different
// issue (e.g., pw-8975), leading to confusing SPAWN_CONTEXT where the TASK
// line says one thing but bd comment instructions reference another.
func ValidateBeadsIDConsistency(task string, beadsID string) string {
	if beadsID == "" {
		return ""
	}

	trackingPrefix := extractProjectPrefix(beadsID)

	// Find all beads-like IDs in the task text
	matches := regexBeadsIDInText.FindAllString(strings.ToLower(task), -1)
	for _, match := range matches {
		matchPrefix := extractProjectPrefix(match)

		// Only check IDs from the same project (same prefix)
		if matchPrefix != trackingPrefix {
			continue
		}

		// Same project, check if it's the same ID
		if match != strings.ToLower(beadsID) {
			return fmt.Sprintf(
				"Warning: task text references %s but tracking issue is %s (same project prefix %q). "+
					"This may cause agent confusion — TASK line will say %s but bd comment instructions will use %s.",
				match, beadsID, trackingPrefix, match, beadsID,
			)
		}
	}

	return ""
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
// For meta-orchestrator skills, it points to META_ORCHESTRATOR_CONTEXT.md.
// For orchestrator-type skills, it points to ORCHESTRATOR_CONTEXT.md instead.
func MinimalPrompt(cfg *Config) string {
	if cfg.IsMetaOrchestrator {
		return MinimalMetaOrchestratorPrompt(cfg)
	}
	if cfg.IsOrchestrator {
		return MinimalOrchestratorPrompt(cfg)
	}
	return fmt.Sprintf(
		"Read your spawn context from %s/.orch/workspace/%s/SPAWN_CONTEXT.md. The instructions in SPAWN_CONTEXT.md are mandatory protocol. Your first tool call may read SPAWN_CONTEXT.md; immediately after reading, report Phase: Planning via the bd comment command specified there. Do not end a turn with narrative unless you are BLOCKED, have a QUESTION, or are COMPLETE. Continue making tool calls until all required deliverables (including Phase: Complete reporting and any required files) are done. Begin the task.",
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

// RegisteredProject represents a project registered with kb.
type RegisteredProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DetectAreaFromTask attempts to detect a knowledge area/cluster from the task description or beads issue.
// Returns empty string if no clear area is detected.
// Checks against known clusters in .kb/investigations/synthesized/ and model directories.
func DetectAreaFromTask(task string, beadsID string, projectDir string) string {
	// Get list of known clusters from filesystem
	kbDir := filepath.Join(projectDir, ".kb")

	// Check investigations/synthesized/ for clusters
	synthesizedDir := filepath.Join(kbDir, "investigations", "synthesized")
	var knownClusters []string
	if entries, err := os.ReadDir(synthesizedDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				knownClusters = append(knownClusters, entry.Name())
			}
		}
	}

	// Add model directories as potential clusters
	modelsDir := filepath.Join(kbDir, "models")
	if entries, err := os.ReadDir(modelsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				knownClusters = append(knownClusters, entry.Name())
			}
		}
	}

	// Always include "models" and "decisions" as default clusters
	knownClusters = append(knownClusters, "models", "decisions")

	// Check task description for cluster keywords
	taskLower := strings.ToLower(task)
	for _, cluster := range knownClusters {
		// Check if cluster name appears in task (word boundary match)
		// Use regex to match whole words only
		pattern := `\b` + regexp.QuoteMeta(strings.ToLower(cluster)) + `\b`
		if matched, _ := regexp.MatchString(pattern, taskLower); matched {
			return cluster
		}
	}

	// If beads issue is provided, check labels for area:* pattern
	if beadsID != "" {
		// Try to get beads issue and check labels
		socketPath, err := beads.FindSocketPath("")
		if err == nil {
			client := beads.NewClient(socketPath)
			if err := client.Connect(); err == nil {
				defer client.Close()
				if issue, err := client.Show(beadsID); err == nil {
					for _, label := range issue.Labels {
						if strings.HasPrefix(label, "area:") {
							area := strings.TrimPrefix(label, "area:")
							// Verify area exists as a known cluster
							for _, cluster := range knownClusters {
								if cluster == area {
									return area
								}
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// GetClusterSummary fetches a summary for a specific cluster using orch tree --format summary.
// Returns empty string if cluster not found or command fails.
func GetClusterSummary(clusterName string, projectDir string) string {
	if clusterName == "" {
		return ""
	}

	// Run orch tree --cluster <name> --format summary
	cmd := exec.Command("orch", "tree", "--cluster", clusterName, "--format", "summary")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// GenerateRegisteredProjectsContext creates the registered projects section for orchestrator contexts.
// Returns empty string if kb projects list fails or returns no projects.
func GenerateRegisteredProjectsContext() string {
	projects, err := GetRegisteredProjects()
	if err != nil || len(projects) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Registered Projects\n\n")
	sb.WriteString("These projects are registered with `kb` for cross-project orchestration:\n\n")
	sb.WriteString("| Project | Path |\n")
	sb.WriteString("|---------|------|\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", p.Name, p.Path))
	}
	sb.WriteString("\n**Usage:** `orch spawn --workdir <path> SKILL \"task\"`\n\n")

	return sb.String()
}

// GetRegisteredProjects fetches the list of registered projects from kb.
func GetRegisteredProjects() ([]RegisteredProject, error) {
	cmd := exec.Command("kb", "projects", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kb projects list failed: %w", err)
	}

	var projects []RegisteredProject
	if err := json.Unmarshal(output, &projects); err != nil {
		return nil, fmt.Errorf("failed to parse kb projects output: %w", err)
	}

	return projects, nil
}

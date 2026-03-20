package spawn

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
{{if .ArchitectDesign}}
## Architect Design

**This implementation is guided by a prior architect review.** Follow the design below.
Do NOT re-investigate or redesign — implement according to these specifications.

{{.ArchitectDesign}}

{{end}}
{{if .ClaimContext}}
{{.ClaimContext}}
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
{{if .GovernanceContext}}

{{.GovernanceContext}}
{{end}}
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
{{if .Explore}}
## EXPLORATION MODE CONFIGURATION

**Mode:** Exploration (decompose → parallelize → judge → synthesize{{if gt .ExploreDepth 1}} → iterate{{end}})
**Parent Skill:** {{.ExploreParentSkill}}
**Breadth:** {{.ExploreBreadth}} parallel workers
{{if gt .ExploreDepth 1}}**Depth:** {{.ExploreDepth}} (judge can trigger up to {{subtract .ExploreDepth 1}} re-exploration rounds)
{{end}}**Beads ID:** {{.BeadsID}}
{{if .ExploreJudgeModel}}**Judge Model:** {{.ExploreJudgeModel}} (cross-model judging — workers use default model, judge uses {{.ExploreJudgeModel}})
{{end}}
**Your role:** You are an exploration orchestrator. Your job is to:
1. **DECOMPOSE** the question into {{.ExploreBreadth}} independent subproblems
2. **SPAWN** workers for each subproblem using ` + "`orch spawn --bypass-triage --no-track --reason \"exploration worker\" {{.ExploreParentSkill}} \"subproblem\"`" + `
3. **WAIT** for all workers using ` + "`orch wait <beads-id> --timeout 30m`" + ` (or check tmux windows)
4. **COLLECT** findings from each worker's investigation/probe files
5. **JUDGE** — spawn a judge agent: ` + "`orch spawn --bypass-triage --no-track --reason \"exploration judge\"{{if .ExploreJudgeModel}} --model {{.ExploreJudgeModel}}{{end}} exploration-judge \"Evaluate sub-findings for: [question]\"`" + `
6. **SYNTHESIZE** using judge verdicts to weight findings (not concatenate)
{{- if gt .ExploreDepth 1}}
7. **ITERATE** — if judge found critical coverage gaps and depth budget remains, re-explore (see Iteration Protocol below)
{{- end}}

**Decomposition Rules:**
- Each subproblem MUST be independently answerable (no cross-dependencies)
- Subproblems should cover different aspects of the question
- Include the original question context in each worker's task
- Workers use the ` + "`{{.ExploreParentSkill}}`" + ` skill (they get the domain expertise)

**Judge Phase:**
Spawn a dedicated judge agent using the ` + "`exploration-judge`" + ` skill.{{if .ExploreJudgeModel}} Use ` + "`--model {{.ExploreJudgeModel}}`" + ` for cross-model judging.{{end}} Pass it:
- The original question
- Your decomposition plan
- All worker sub-findings (full text or file references)
The judge produces a ` + "`judge-verdict.yaml`" + ` with per-finding verdicts (accepted/contested/rejected),
contested findings analysis, and coverage gaps. Wait for judge completion before synthesizing.
{{if gt .ExploreDepth 1}}
**Iteration Protocol (Depth {{.ExploreDepth}}):**
After reading judge verdicts, check for critical coverage gaps. If the judge identified gaps with severity ` + "`critical`" + `:

1. **Check depth budget:** You start at depth 1. Each iteration increments depth. Stop at depth {{.ExploreDepth}}.
2. **Decompose gaps into subproblems:** Each critical gap from ` + "`coverage_gaps`" + ` becomes a new subproblem.
   - Only spawn workers for ` + "`critical`" + ` severity gaps (skip ` + "`moderate`" + ` and ` + "`minor`" + `)
   - Use the judge's ` + "`suggested_subproblem`" + ` text as the worker task
3. **Spawn gap workers:** Same pattern as initial workers, using ` + "`{{.ExploreParentSkill}}`" + ` skill
4. **Re-judge:** Spawn a new judge with ALL findings (original + gap-filling workers)
5. **Emit iteration event:** ` + "`orch emit exploration.iterated --beads-id {{.BeadsID}} --data '{\"iteration\":N,\"gaps_addressed\":G,\"new_workers\":W}'`" + `
6. **Synthesize:** Only after final iteration (no more critical gaps or depth exhausted)

**Iteration Decision Rules:**
- **Iterate** when: judge found ` + "`critical`" + ` coverage gaps AND depth < {{.ExploreDepth}}
- **Stop** when: no critical gaps, OR depth = {{.ExploreDepth}}, OR rate-limited
- **Never iterate** for: ` + "`moderate`" + ` or ` + "`minor`" + ` gaps (note them in synthesis instead)
- Each iteration should address fewer gaps (convergent). If gaps increase, stop and synthesize what you have.
{{end}}
**Synthesis Output:**
Write your synthesis to the investigation file (.kb/investigations/) or SYNTHESIS.md.
- **Weight by verdict:** accepted findings anchor synthesis, contested get dedicated discussion, rejected are noted but downweighted
- Contested findings (where workers disagree) are the most valuable — highlight them
- Gaps (from judge's coverage_gaps) should be explicitly noted
- Do NOT just concatenate — compose understanding from the parts
{{- if gt .ExploreDepth 1}}
- Note which findings came from iteration rounds (original vs gap-filling)
- Include iteration summary: how many rounds, what gaps were addressed
{{- end}}

**Cost Bounding:**
- Max {{.ExploreBreadth}} workers per round (enforced)
{{- if gt .ExploreDepth 1}}
- Max {{.ExploreDepth}} iterations (depth limit enforced)
- Total agent budget: up to {{.ExploreBreadth}} workers × {{.ExploreDepth}} rounds + {{.ExploreDepth}} judges + 1 synthesizer
{{- end}}
- Workers use --no-track (lightweight, no beads overhead)
- If rate-limited, reduce breadth rather than fail
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
	ArchitectDesign       string   // Architect SYNTHESIS.md content (from --architect-ref)
	DesignWorkspace       string   // Design workspace name for ui-design-session handoff
	DesignMockupPath      string   // Path to approved mockup
	DesignPromptPath      string   // Path to design prompt
	DesignNotes           string   // Notes from design session
	OrientationFrame      string   // Additional task context (from issue description), rendered as separate section
	IntentType            string   // Orchestrator's declared outcome type (experience, produce, compare, etc.)
	PriorCompletions      string   // Prior completed agent work on same issue
	BrowserAutomation     bool     // When true, playwright-cli browser automation is available
	Explore               bool     // When true, this is an exploration mode spawn
	ExploreBreadth        int      // Max parallel workers for exploration
	ExploreDepth          int      // Max iteration depth (1=single pass, N=judge triggers re-exploration)
	ExploreParentSkill    string   // Original skill (investigation/architect)
	ExploreJudgeModel     string   // Model override for judge agent (cross-model experiment)
	GovernanceContext     string   // Governance-protected paths section for pre-planning awareness
	ClaimContext          string   // Formatted claim probe context (claim text, falsifies_if, evidence)
}

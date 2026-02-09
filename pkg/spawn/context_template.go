package spawn

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// SpawnContextTemplate is the basic structure for SPAWN_CONTEXT.md.
const SpawnContextTemplate = `TASK: {{.Task}}
{{if .BloatWarnings}}
{{.BloatWarnings}}
{{end}}
{{if .FailureContext}}
{{if .FailureContext.IsRework}}
## 🚨 REWORK SPAWN - PRIOR ATTEMPT FAILED

⚠️ **This is a REWORK spawn.** A previous agent reported completion but the feature doesn't work.

**Failure Type:** {{.FailureContext.FailureType}}
**What Failed:** {{.FailureContext.Description}}
{{if .FailureContext.PriorAttemptContext}}
**Prior Attempt Context:**
{{.FailureContext.PriorAttemptContext}}
{{end}}

**Suggested Skill:** {{.FailureContext.SuggestedSkill}}
{{if ne .SkillName .FailureContext.SuggestedSkill}}
⚠️ You were spawned with ` + "`" + `{{.SkillName}}` + "`" + ` but the suggested skill for this failure type is ` + "`" + `{{.FailureContext.SuggestedSkill}}` + "`" + `.
Consider whether the current skill is appropriate or if re-spawning with the suggested skill would be more effective.
{{end}}

### What This Means

1. **Do NOT trust the prior agent's completion claim** - they said it was done, but it wasn't
2. **Verify the ORIGINAL requirement actually works** before claiming completion
3. **Focus on the failure type:**
   {{if eq .FailureContext.FailureType "verification"}}
   - Prior agent didn't properly verify their work
   - Ensure you have concrete evidence the feature works (not just tests passing)
   {{else if eq .FailureContext.FailureType "implementation"}}
   - The code has a bug - find and fix it
   - Use systematic debugging to identify the root cause
   {{else if eq .FailureContext.FailureType "spec"}}
   - The spec was wrong or incomplete
   - Investigate what the correct behavior should be first
   {{else if eq .FailureContext.FailureType "integration"}}
   - Feature works in isolation but fails in real context
   - Test in the full integration environment
   {{end}}

### Verification Requirement

Before reporting Phase: Complete, you MUST:
1. Reproduce the failure described above
2. Fix the root cause
3. Verify the feature works end-to-end (not just unit tests)
4. Document your verification evidence in the completion comment

{{end}}
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
{{if .ModelBehaviorProfile}}
MODEL BEHAVIOR PROFILE: {{.ModelBehaviorProfile}}
{{if .NeedsCompletionNudge}}
⚠️ This model often needs an explicit completion nudge.
{{if .NoTrack}}
Before /exit, explicitly state completion in your final message and confirm the checklist is done.
{{else}}
Before /exit, explicitly run ` + "`orch phase {{.BeadsID}} Complete \"[1-2 sentence summary]\"`" + ` and confirm it succeeds.
{{end}}
{{else if .ReliablePhaseReporting}}
✅ This model usually reports phases reliably. Still follow the completion checklist exactly.
{{end}}
{{end}}
{{if .KBContext}}
{{.KBContext}}
{{end}}
{{if .HasInjectedModels}}
## PROBE GUIDANCE (Model-Scoped Work)

**Models were injected above.** When your task involves testing, validating, or extending claims from an existing model, produce a **probe** (~30-50 lines) instead of a full investigation (~300 lines).

**Probe structure (all sections mandatory):**

1. **Question** — The specific claim or invariant being tested
2. **What I Tested** — Actual command run or code executed (NOT code review — you must run something)
3. **What I Observed** — Concrete output (paste actual results)
4. **Model Impact** — Verdict: confirms | contradicts | extends — which invariant, with 1-3 sentence explanation

**Template:** See ` + "`" + `{{.ProjectDir}}/.orch/templates/PROBE.md` + "`" + `

**When to use probe vs full investigation:**
- **Probe:** Model exists, you are confirming/extending its claims with targeted tests
- **Full investigation:** Novel exploration where no model exists, or model is fundamentally wrong

**Key discipline:** "What I Tested" must contain an actual command or code you executed. Reading code is not testing.
{{end}}
{{if .IsInfrastructureTouching}}
## RESOURCE LIFECYCLE AUDIT (REQUIRED)

Audit all resources this component creates (goroutines, subprocesses, connections, caches) and ensure each has bounded lifetime and cleanup on shutdown.
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
{{if .IssueComments}}
## ORCHESTRATOR NOTES

The following notes were added to this issue by the orchestrator after issue creation.
Read these carefully - they may contain important clarifications, constraints, or guidance.

{{range .IssueComments}}
**{{.Author}}** ({{.CreatedAt}}):
> {{.Text}}

{{end}}
{{end}}
{{if .NoTrack}}
📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
TERMINAL CHECKLIST (EXECUTE IN ORDER):
- [ ] NEVER run ` + "`git push`" + ` (orchestrator handles remote push)
- [ ] ` + "`git push`" + ` can trigger deploys that disrupt production systems

{{if eq .Tier "light"}}
- [ ] Run: ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
- [ ] Ensure SYNTHESIS.md is created and committed in your workspace
- [ ] Run: ` + "`/exit`" + `
{{end}}
{{else}}
🚨 FIRST 3 ACTIONS (ADVISORY - STRONGLY RECOMMENDED):
Suggested first actions for visibility:
1. Report phase: ` + "`orch phase {{.BeadsID}} Planning \"[brief description]\"`" + `
2. Read relevant codebase context for your task
3. Begin planning

**Note:** ` + "`orch phase`" + ` writes to SQLite (~1ms) for instant dashboard visibility AND automatically adds a bd comment for audit trail. One command handles both.

**Gates vs Advisory Checkpoints:**
- **Gates** (blocking): Enforced by ` + "`orch complete`" + ` code - will BLOCK completion if not satisfied (e.g., Phase: Complete, test evidence, visual verification for UI changes)
- **Advisory** (suggested): Best practices that help quality but aren't enforced - you can skip these without blocking completion

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
TERMINAL CHECKLIST (EXECUTE IN ORDER):
- [ ] NEVER run ` + "`git push`" + ` (orchestrator handles remote push)
- [ ] ` + "`git push`" + ` can trigger deploys that disrupt production systems

{{if eq .Tier "light"}}
- [ ] Run: ` + "`orch phase {{.BeadsID}} Complete \"[1-2 sentence summary of deliverables]\"`" + `
- [ ] Run: ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
- [ ] Ensure SYNTHESIS.md is created and committed in your workspace
- [ ] Run: ` + "`orch phase {{.BeadsID}} Complete \"[1-2 sentence summary of deliverables]\"`" + `
- [ ] Run: ` + "`/exit`" + `
{{end}}
If you do not emit Phase: Complete, your work will be lost
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
{{if .IsInvestigationSkill}}
2. **SET UP investigation deliverable:**
{{if .HasInjectedModels}}
   - Models are available in kb context above. For model-scoped validation/extension work, produce a **probe** instead of a standalone investigation.
   - Probe location: ` + "`.kb/models/<model-name>/probes/YYYY-MM-DD-<slug>.md`" + `
   - Probe template: ` + "`{{.ProjectDir}}/.orch/templates/PROBE.md`" + `
   - If you produce a probe, do NOT create a redundant standalone investigation file.
   - Use standalone investigation only when no model applies or the model is fundamentally wrong.
{{end}}
   - For standalone investigation workflow, run ` + "`kb create investigation {{.InvestigationSlug}} --defect-class <class>`" + ` to create from template
   - This creates: ` + "`.kb/investigations/YYYY-MM-DD-<type>-<slug>.md`" + `
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - Defect class is REQUIRED (mechanism, not symptom). Use one of:
     ` + "`unbounded-growth`" + `, ` + "`race-condition`" + `, ` + "`missing-cleanup`" + `, ` + "`state-corruption`" + `, ` + "`resource-leak`" + `, ` + "`configuration-drift`" + `, ` + "`integration-mismatch`" + `
   - If command fails, report to orchestrator immediately
{{if not .NoTrack}}
   - **IMPORTANT:** Report your actual deliverable path via:
     ` + "`bd comment {{.BeadsID}} \"investigation_path: /path/to/file.md\"`" + `
     (Use the probe or investigation path so orch complete can verify the correct file)
{{end}}
3. **UPDATE investigation deliverable** as you work:
{{if .HasInjectedModels}}
   - Probe workflow: Fill all 4 mandatory sections (Question, What I Tested, What I Observed, Model Impact)
   - Probe evidence must include concrete command/output results (reading code alone is insufficient)
{{end}}
   - Investigation workflow: Add TLDR at top (1-2 sentence summary of question and finding)
   - Investigation workflow: Fill sections (What I tried → What I observed → Test performed)
   - Investigation workflow: Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
{{else}}
2. [Task-specific deliverables from skill guidance]
{{end}}
{{if ne .Tier "light"}}
{{if .IsInvestigationSkill}}6{{else}}3{{end}}. **CREATE SYNTHESIS.md:** Before completing, create ` + "`SYNTHESIS.md`" + ` in your workspace: {{.ProjectDir}}/.orch/workspace/{{.WorkspaceName}}/SYNTHESIS.md
   - Use the template from: {{.ProjectDir}}/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.
{{else}}
{{if .IsInvestigationSkill}}6{{else}}3{{end}}. ⚡ SYNTHESIS.md is NOT required (light tier spawn).
{{end}}
{{if .IsInvestigationSkill}}
STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session
{{end}}

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input
{{if not .NoTrack}}

## PROGRESS TRACKING

You were spawned from beads issue: **{{.BeadsID}}**

**Phase reporting:** Use ` + "`orch phase`" + ` — one command that writes to SQLite (~1ms) for instant dashboard visibility AND automatically adds a bd comment for permanent audit trail.

` + "```bash" + `
# Report progress at phase transitions
orch phase {{.BeadsID}} Planning "Analyzing codebase structure"
orch phase {{.BeadsID}} Implementing "Adding authentication middleware"
orch phase {{.BeadsID}} Complete "All tests passing, ready for review"

# Report blockers immediately
orch phase {{.BeadsID}} BLOCKED "Need clarification on API contract"

# Report questions
orch phase {{.BeadsID}} QUESTION "Should we use JWT or session-based auth?"
` + "```" + `

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

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
TERMINAL CHECKLIST (EXECUTE IN ORDER):
- [ ] NEVER run ` + "`git push`" + ` (orchestrator handles remote push)
- [ ] ` + "`git push`" + ` can trigger deploys that disrupt production systems

{{if .NoTrack}}
{{if eq .Tier "light"}}
- [ ] ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
- [ ] Ensure SYNTHESIS.md is created and committed in your workspace
- [ ] ` + "`/exit`" + `
{{end}}
{{else}}
{{if eq .Tier "light"}}
- [ ] ` + "`orch phase {{.BeadsID}} Complete \"[1-2 sentence summary]\"`" + `
- [ ] ` + "`/exit`" + `

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.
{{else}}
- [ ] Ensure SYNTHESIS.md is created and committed in your workspace
- [ ] ` + "`orch phase {{.BeadsID}} Complete \"[1-2 sentence summary]\"`" + `
- [ ] ` + "`/exit`" + `
{{end}}
If you do not emit Phase: Complete, your work will be lost
{{end}}
⚠️ Your work is NOT complete until this checklist is finished.
`

// GenerateContext generates the SPAWN_CONTEXT.md content.
func GenerateContext(cfg *Config) (string, error) {
	tmpl, err := template.New("spawn_context").Parse(SpawnContextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := buildContextData(cfg)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
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

## Verification Contract

- **Spec:** ` + "`VERIFICATION_SPEC.yaml`" + ` (workspace root)
- **Key outcomes:**
  - ` + "`[verification-id]`" + ` - [pass | fail | pending] - Brief evidence or observed output
  - ` + "`[verification-id]`" + ` - [pass | fail | pending] - Brief evidence or observed output
- **Manual steps (if any):**
  - ` + "`[step]`" + ` - [completed | pending] - Note or owner

*(If no spec update was needed, state why.)*

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

## Issues Created

**Discovered work tracked during this session:**

- ` + "`orch-go-XXXXX`" + ` - Brief description of bug/task/feature/question
- ` + "`orch-go-YYYYY`" + ` - Brief description

*(If no discovered work: "No discovered work during this session")*

**Note:** Every session should either create issues for discovered work OR explicitly note that none was found. Bugs, tech debt, enhancement ideas, and strategic questions should all be tracked.

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

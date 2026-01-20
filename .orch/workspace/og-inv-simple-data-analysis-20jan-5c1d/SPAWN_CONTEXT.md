TASK: Simple data analysis test


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "simple data analysis"

### Constraints (MUST respect)
- Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads
  - Reason: Investigation found all registry data can be derived from primary sources

### Prior Decisions
- Session_id stored in workspace file not registry
  - Reason: Co-locates data with workspace, single writer, no lock contention
- Temporal density and repeated constraints are highest value reflection signals
  - Reason: Low noise, high actionability - tested against real kn/kb data
- Chronicle should be a view over existing artifacts, not new artifact type
  - Reason: Minimal taxonomy principle; source data already exists in git/kn/kb; value is in narrative synthesis not data capture
- orch status shows PHASE and TASK columns from beads data
  - Reason: Makes output actionable - users can immediately see what each agent is doing
- Use existing stores for UI improvements over backend changes
  - Reason: All necessary data (errorEvents, agentlogEvents) already exists in frontend stores; UI-only changes are faster to implement and deploy
- Skill constraints use HTML comment blocks for embedding
  - Reason: Simple extraction via regex, survives markdown processing, follows existing skillc pattern for metadata (checksum, source)
- Use '-' as display fallback for missing duration data in UI
  - Reason: Cleaner than 'N/A' or empty, signals missing data without breaking layout
- Use title attributes for tooltips in Svelte rather than custom tooltip component
  - Reason: Native browser tooltips work well for simple explanatory text, no extra dependencies needed

### Models (synthesized understanding)
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Dashboard Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md
- Daemon Autonomous Operation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation.md
- Decidability Graph
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/decidability-graph.md
- Beads Integration Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
- Dashboard Agent Status Calculation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-agent-status.md
- OpenCode Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-session-lifecycle.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/extract-patterns.md

### Guides (procedural knowledge)
- API Development Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/api-development.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md
- Background Services Performance Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/background-services-performance.md
- Two-Tier Sensing Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/two-tier-sensing-pattern.md
- OpenCode Plugin System Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode-plugins.md
- Beads Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Agent Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/agent-lifecycle.md

### Related Investigations
- Deep Analysis of OpenCode Coaching Plugin Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md
- Design Data Model Load Bearing
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-design-data-model-load-bearing.md
- Playwright MCP Performance Analysis - Where Does the Time Go?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-27-inv-playwright-mcp-performance-analysis-time.md
- Dashboard UI Structure Analysis and Refactor Plan
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-design-analyze-dashboard-ui-structure.md
- pkg/verify/check.go Structure Analysis and Extraction Plan
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-design-analyze-pkg-verify-check.md
- Registry Abandonment Workflow Validate Simple
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md
- orch handoff generates stale/incorrect data
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-orch-handoff-generates-stale-incorrect.md
- Add /api/errors Endpoint for Error Pattern Analysis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-add-api-errors-endpoint-error.md
- daemon.go Structure Analysis and Extraction Plan
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-design-analyze-pkg-daemon-daemon.md
- Dashboard Shows 0% When Usage Data Unavailable
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-dashboard-shows-usage-anthropic-api.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `/exit` to close the agent session



CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

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

**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Document it in your investigation file: "CONSTRAINT: [what constraint] - [why considering workaround]"
2. Include the constraint and your reasoning in SYNTHESIS.md
3. The accountability is a feature, not a cost

This applies to:
- System constraints discovered during work (e.g., API limits, tool limitations)
- Architectural patterns that seem inconvenient for your task
- Process requirements that feel like overhead
- Prior decisions (from `kb context`) that conflict with your approach

**Why:** Working around constraints without surfacing them:
- Prevents the system from learning about recurring friction
- Bypasses stakeholders who should know about the limitation
- Creates hidden technical debt

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation simple-data-analysis-test` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-simple-data-analysis-test.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-simple-data-analysis-20jan-5c1d/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (investigation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 7a352f8412b8 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/skills/src/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-19 15:42:20 -->

## Summary

**Purpose:** Common protocols shared by all worker skills. This is inherited by worker skills via dependencies.

---

# Worker Base Patterns

**Purpose:** Common protocols shared by all worker skills. This is inherited by worker skills via dependencies.

**What this provides:**
- Authority delegation (what you can decide vs escalate)
- Beads progress tracking (how to report via bd comment)
- Phase reporting (how to signal transitions)
- Exit/completion protocol (how to properly end a session)

---

## Authority Delegation

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

---

## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment {{.BeadsID}} "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

**Status Updates:**
Update Status: field in your workspace/investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed)

**Signal orchestrator when blocked:**
- Add `**Status:** BLOCKED - [reason]` to workspace
- Add `**Status:** QUESTION - [question]` when needing input

---

## Session Complete Protocol

**When your work is done (all deliverables ready), complete in this EXACT order:**

{{if eq .Tier "light"}}
1. Run: `bd comment {{.BeadsID}} "Phase: Complete - [1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
2. Commit any final changes
3. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Run: `bd comment {{.BeadsID}} "Phase: Complete - [1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
2. Ensure SYNTHESIS.md is created
3. Commit all changes (including SYNTHESIS.md)
4. Run: `/exit` to close the agent session
{{end}}

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

---

---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 97a7b3127364 -->
<!-- Source: .skillc -->
<!-- To modify: edit files in .skillc, then run: skillc build -->
<!-- Last compiled: 2026-01-19 10:40:12 -->

<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file with findings -->
<!-- /SKILL-CONSTRAINTS -->
## Summary

**Purpose:** Answer a question by testing, not by reasoning.

---

# Investigation Skill

**Purpose:** Answer a question by testing, not by reasoning.

## The One Rule

**You cannot conclude without testing.**

If you didn't run a test, you don't get to fill the Conclusion section.

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

- **Primary** (authoritative): Actual code, test output, observed behavior → This IS the evidence
- **Secondary** (claims to verify): Workspaces, investigations, decisions → Hypotheses to test

When an artifact says "X is not implemented," that's a hypothesis—search the codebase before concluding.

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for evidence hierarchy examples and common failures.

## Workflow

1. **Create file:** `kb create investigation {slug}`
2. **IMMEDIATE CHECKPOINT:** Fill Question, add Finding 1 ("Starting approach"), commit immediately
3. **TEST-FIRST GATE:** "What's the simplest test I can run right now?" (60-second rule)
4. Try things, observe what happens (add findings progressively)
5. **Run a test to validate your hypothesis**
6. Fill conclusion only if you tested
7. Final commit

**Why checkpoint immediately?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, no record of what was attempted.

**Reference:** See `~/.claude/skills/worker/investigation/reference/error-recovery.md` for handling fatal errors during exploration.

## D.E.K.N. Summary

**Every investigation file starts with a D.E.K.N. summary block.** Enables 30-second handoff to fresh Claude.

- **Delta:** What was discovered/answered
- **Evidence:** Primary evidence supporting conclusion
- **Knowledge:** What was learned (insights, constraints)
- **Next:** Recommended action

**Fill D.E.K.N. at the END, before marking Complete.**

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for D.E.K.N. examples.

## Template

Use `kb create investigation {slug}` to create. Required sections:
- **D.E.K.N. Summary** (Delta, Evidence, Knowledge, Next)
- **Question** and **Status**
- **Findings** (add progressively)
- **Test performed** (not "reviewed code" - actual test)
- **Conclusion** (only if you tested)

**Reference:** See `~/.claude/skills/worker/investigation/reference/template.md` for full structure and `reference/examples.md` for common failures.

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`

## Self-Review (Mandatory)

Before completing, verify investigation quality:

### Self-Review Checklist

- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
- [ ] **Question answered** - Investigation addresses the original question
- [ ] **File complete** - All sections filled (not "N/A" or "None")
- [ ] **D.E.K.N. filled** - Replaced placeholders in Summary section
- [ ] **Scope verified** - Ran `rg` to find all occurrences before concluding
- [ ] **NOT DONE claims verified** - If claiming incomplete, searched actual code

### Discovered Work

If you found bugs, tech debt, or enhancement ideas during investigation:
- Create beads issues: `bd create "description" --type bug|task|feature`
- Apply label: `bd label <id> triage:ready` or `triage:review`

**If no discoveries:** Note "No discovered work items" in completion comment.

**Reference:** See `~/.claude/skills/worker/investigation/reference/self-review-guide.md` for scope verification examples and discovered work procedures.

**Only proceed to commit after self-review passes.**

---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**
- `kb quick decide "X" --reason "Y"` (made a choice)
- `kb quick tried "X" --failed "Y"` (something failed)
- `kb quick constrain "X" --reason "Y"` (found a constraint)
- `kb quick question "X"` (open question)

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

**Reference:** See `~/.claude/skills/worker/investigation/reference/leave-it-better.md` for command examples.

---

## Completion

1. Self-review passed
2. Leave it Better completed (or noted why N/A)
3. D.E.K.N. summary filled (with **Promote to Decision** flag)
4. Report: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"` (FIRST - before commit)
5. Commit: `git add && git commit`
6. Exit: `/exit`

**Why report before commit?** If agent dies after commit but before reporting, orchestrator cannot detect completion.

---

**Remember:** Test before concluding.



---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.

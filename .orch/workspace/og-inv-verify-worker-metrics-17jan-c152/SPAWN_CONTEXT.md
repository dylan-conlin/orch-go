TASK: Verify worker metrics: perform 10 tool calls (e.g., read random files) and then check ~/.orch/coaching-metrics.jsonl for any entries with your session ID and metric_type 'context_usage'. This is a post-fix verification spawn. Verification protocol: Architect design was completed in orch-go-ls80s, now verifying implementation fix.


SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "verify worker metrics"

### Constraints (MUST respect)
- orch complete must verify SYNTHESIS.md exists and is not placeholder before closing issue
  - Reason: 70% of agents completed without synthesis in 24h chaos period
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- Worker agents must test spawn functionality via unit tests and code review, not end-to-end spawning
  - Reason: Prevents recursive spawn testing incidents while still enabling verification
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).

### Prior Decisions
- orch spawn context delivery is reliable
  - Reason: Verified that SPAWN_CONTEXT.md is correctly populated and accessible by the agent
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- When spawned for cross-repo work, verify work completion status before starting
  - Reason: Task orch-go-oo1f: spawned in orch-go for work in orch-knowledge. Template was already retired (commit 7430185) before agent fully engaged. Quick verification could have saved agent context.
- Headless spawn mode is production-ready
  - Reason: All 5 requirements verified working: status detection, monitoring, completion detection, error handling, user visibility. Investigation orch-go-0r2q confirmed no blockers exist.
- Spawn system verified functional for basic use cases
  - Reason: Test spawn successfully created workspace, loaded context, created investigation file via kb CLI
- Layer 2 skill constraint verification already implemented in orch-go
  - Reason: pkg/verify/constraint.go (211 lines) parses SKILL-CONSTRAINTS block from SPAWN_CONTEXT.md and verifies required/optional patterns via VerifyCompletionFull()
- Skill output verification parses skill.yaml directly
  - Reason: skillc verify CLI doesn't exist, so we parse outputs.required from skill.yaml files in Go
- orch complete gates on visual verification for web/ changes
  - Reason: Agents modifying web files must provide screenshot evidence or mention visual verification in beads comments. This ensures UI changes are actually verified before completion.

### Models (synthesized understanding)
- Orchestrator Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle.md
- Dashboard Agent Status Calculation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-agent-status.md
- Follow Orchestrator Mechanism
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/follow-orchestrator-mechanism.md
- Completion Verification Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification.md
- Escape Hatch Visibility Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/escape-hatch-visibility-architecture.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
- Daemon Autonomous Operation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation.md
- Dashboard Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture.md
- Workspace Lifecycle & Hierarchy
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/workspace-lifecycle-model.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md

### Guides (procedural knowledge)
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- OpenCode Plugin System Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode-plugins.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Code Extraction Patterns
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/code-extraction-patterns.md
- Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/status.md
- Two-Tier Sensing Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/two-tier-sensing-pattern.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md

### Related Investigations
- Missing Coaching Metrics and Frame Collapse Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-investigate-missing-coaching-metrics-frame.md
- Workers Attempting Restart Orch Servers
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-workers-attempting-restart-orch-servers.md
- Workers Pushing Remote No Push
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-16-inv-workers-pushing-remote-no-push.md
- Add Worker Specific Metrics Plugins
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-add-worker-specific-metrics-plugins.md
- Test Verify Daemon Skip Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-test-verify-daemon-skip-functionality.md
- Test Worker Naming
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-13-inv-test-worker-naming.md
- Debug Worker Filtering Coaching Ts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-10-inv-debug-worker-filtering-coaching-ts.md
- Add Worker Filtering Coaching Ts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-10-inv-add-worker-filtering-coaching-ts.md
- Verify Test First Gate Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-15-inv-verify-test-first-gate-implementation.md
- Orch Doctor Verify Dashboard Fetch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-14-inv-orch-doctor-verify-dashboard-fetch.md

### Failed Attempts (DO NOT repeat)
- orch clean to remove ghost sessions automatically

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-8hxsr "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


1. Run: `bd comment orch-go-8hxsr "Phase: Complete - [1-2 sentence summary of deliverables]"`
2. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.


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
1. Surface it first: `bd comment orch-go-8hxsr "CONSTRAINT: [what constraint] - [why considering workaround]"`
2. Wait for orchestrator acknowledgment before proceeding
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
2. **SET UP investigation file:** Run `kb create investigation verify-worker-metrics-perform-10` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-verify-worker-metrics-perform-10.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-8hxsr "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. ⚡ SYNTHESIS.md is NOT required (light tier spawn).


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-8hxsr**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-8hxsr "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-8hxsr "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-8hxsr "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-8hxsr "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-8hxsr "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-8hxsr`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (investigation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: b17aa8f7412f -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-17 01:46:54 -->


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



1. `bd comment orch-go-8hxsr "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.


⚠️ Your work is NOT complete until you run these commands.

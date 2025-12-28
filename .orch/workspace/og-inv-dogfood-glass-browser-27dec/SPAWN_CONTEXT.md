TASK: How to dogfood Glass browser automation effectively. Context: Dylan wants to start using Glass regularly to surface gaps, but current friction includes: 1) Opens separate Chrome window instead of user's browser (orch-go-cg9s), 2) Unclear integration status. Questions: What's current Glass setup? What's minimal dogfooding path? What needs fixing first? Check prior investigation og-inv-glass-integration-status-27dec. Goal: Actionable recommendation for dogfooding Glass this week.

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "how"

### Constraints (MUST respect)
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state
- Activity state should be ephemeral in UI
  - Reason: Real-time activity is meant to show current state, not history - keeping it ephemeral avoids state management complexity and storage costs
- Ask 'should we' before 'how do we' for strategic direction changes
  - Reason: Epic orch-go-erdw was created assuming skills-as-value was correct direction. Architect review revealed the premise was wrong - current separation is intentional design. Wasted work avoided by validating premise before execution.

### Prior Decisions
- Registry updates must happen before beads close in orch complete
  - Reason: Prevents inconsistent state where beads shows closed but registry shows active
- Implement 3-tier guardrail system: preflight checks, completion gates, daily reconciliation
  - Reason: Post-mortem showed 115 commits in 24h with 7 missing guardrails enabling runaway automation
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- orch status shows PHASE and TASK columns from beads data
  - Reason: Makes output actionable - users can immediately see what each agent is doing
- orch servers uses project-grouped output not service-grouped
  - Reason: Showing ports grouped by project (e.g., 'web:5173, api:3333') is more intuitive than separate rows per service because developers think in terms of projects not individual services
- Document existing capabilities before building new infrastructure
  - Reason: WebFetch investigation showed tool already exists - main gap was documentation not capability
- orch serve displayThreshold should match orch status (30min)
  - Reason: 6h threshold showed 25 stale sessions while orch status showed 0, causing dashboard noise
- kb context uses keyword matching, not semantic understanding - 'how would the system recommend...' questions require orchestrator synthesis
  - Reason: Tested kb context with various query formats: keyword queries (swarm, dashboard) returned results, but semantic queries (swarm map sorting, how should dashboard present agents) returned nothing. The pattern reveals desire for semantic query answering that would require LLM-based RAG.
- Usage display color thresholds: green <60%, yellow 60-80%, red >80%
  - Reason: Matches established UX patterns for warning levels, consistent with how other dashboards signal utilization
- Slide-out panel for agent card detail view
  - Reason: Preserves swarm grid context while showing detail; standard pattern for select-to-detail UX; modal blocks view, inline expand disrupts grid
- Show full TLDR on completed agent cards by removing line-clamp-2
  - Reason: Users want to see the complete summary at a glance instead of having to click for details
- Show collapsed section preview
  - Reason: Users can see agent tasks at a glance without expanding sections
- Usage progress bars show reset times inline with percentage
  - Reason: Reset time context is essential for orchestrator to know when to switch accounts or wait
- orch-go auth implementation is complete
  - Reason: Code review shows Go has OAuth login, refresh, switch, capacity tracking - all features from Python orch-cli
- SHOULD-HOW-EXECUTE sequence for strategic questions
  - Reason: Epic orch-go-erdw was created from 'how do we' without testing premise. Architect found premise wrong. Wasted work avoided if 'should we' asked first.
- Cross-project agent visibility requires extracting PROJECT_DIR from SPAWN_CONTEXT.md
  - Reason: Beads comments are per-project; to show phase info for agents from other projects, must query the correct project's beads database
- Dashboard is_processing visual indicators require status === 'active' check
  - Reason: SSE session.status events may not clear is_processing flag when agent completes, causing stale pulsing animation. Defensive check ensures only active agents show processing state.
- OpenCode handles OAuth auto-refresh via anthropic-auth plugin
  - Reason: Code inspection shows token check at fetch time - no orch implementation needed
- Dashboard synthesis review shows synthesis inline with actionable issue creation
  - Reason: Enables orchestrators to act on synthesis recommendations without leaving dashboard UI
- Error pattern analysis uses normalized message matching for grouping similar errors
  - Reason: Enables dashboard to show recurring patterns by truncating to 100 chars and trimming whitespace

### Related Investigations
- CLI orch complete Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-complete-command.md
- CLI orch spawn Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md
- CLI Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-status-command.md
- CLI Project Scaffolding and Build Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-project-scaffolding-build.md
- SSE Event Monitoring Client
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md
- Desktop Notifications on Completion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-desktop-notifications-completion.md
- Fix comment ID parsing - Comment.ID type mismatch
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-fix-comment-id-parsing-comment.md
- Fix SSE parsing - event type inside JSON data
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-fix-sse-parsing-event-type.md
- Set beads issue status to in_progress on spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-set-beads-issue-status-progress.md
- Update README with current CLI commands and usage
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-update-readme-current-cli-commands.md
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- Add /api/agentlog endpoint to serve.go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- Add CLI Commands for Focus, Drift, and Next
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-cli-commands-focus-drift.md
- Add --dry-run flag to daemon run command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-dry-run-flag-daemon.md
- Add Missing Spawn Flags
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-missing-spawn-flags-no.md
- Add orch review command for batch completion workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-orch-review-command-batch.md
- Add Usage/Capacity Tracking to Account Package
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-usage-capacity-tracking-account.md
- Add Wait Command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-wait-command-orch.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-7vsp "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-7vsp "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

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

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Surface it first: `bd comment orch-go-7vsp "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation dogfood-glass-browser-automation-effectively` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-dogfood-glass-browser-automation-effectively.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-7vsp "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-dogfood-glass-browser-27dec/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-7vsp**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-7vsp "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-7vsp "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-7vsp "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-7vsp "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-7vsp "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-7vsp`.

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
<!-- Checksum: c522ec6e6913 -->
<!-- Source: .skillc -->
<!-- To modify: edit files in .skillc, then run: skillc build -->
<!-- Last compiled: 2025-12-25 20:45:03 -->


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

| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| **Primary** (authoritative) | Actual code, test output, observed behavior | This IS the evidence |
| **Secondary** (claims to verify) | Workspaces, investigations, decisions | Hypotheses to test |

When an artifact says "X is not implemented," that's a hypothesis—not a finding to report. Search the codebase before concluding.

**The failure mode:** An agent reads a workspace claiming "feature X NOT DONE" and reports that as a finding without checking if feature X actually exists in the code.


## Workflow

1. Create investigation file: `kb create investigation {slug}`
2. Fill in your question
3. Try things, observe what happens
4. **Run a test to validate your hypothesis**
5. Fill conclusion only if you tested
6. Commit

## D.E.K.N. Summary

**Every investigation file starts with a D.E.K.N. summary block at the top.** This enables 30-second handoff to fresh Claude.

| Section | Purpose | Example |
|---------|---------|---------|
| **Delta** | What was discovered/answered | "Test-running guidance is missing from spawn prompts" |
| **Evidence** | Primary evidence supporting conclusion | "Searched 5 agent sessions - none ran tests" |
| **Knowledge** | What was learned (insights, constraints) | "Agents follow documentation literally" |
| **Next** | Recommended action | "Add test-running instruction to template" |

**Fill D.E.K.N. at the END of your investigation, before marking Complete.**


## Template

The template enforces the discipline. Use `kb create investigation {slug}` to create.

```markdown
## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered]
**Evidence:** [Primary evidence supporting conclusion]
**Knowledge:** [What was learned]
**Next:** [Recommended action]

---

# Investigation: [Topic]

**Question:** [What are you trying to figure out?]
**Status:** Active | Complete

## Findings
[Evidence gathered]

## Test performed
**Test:** [What you did to validate]
**Result:** [What happened]

## Conclusion
[Only fill if you tested]
```

## Common Failures

**"Logical verification" is not a test.**

Wrong:
```markdown
## Test performed
**Test:** Reviewed the code logic
**Result:** The implementation looks correct
```

Right:
```markdown
## Test performed
**Test:** Ran `time orch spawn investigation "test"` 5 times
**Result:** Average 6.2s, breakdown: 70ms orch overhead, 5.5s Claude startup
```

**Speculation is not a conclusion.**

Wrong:
```markdown
## Conclusion
Based on the code structure, the issue is likely X.
```

Right:
```markdown
## Conclusion
The test confirmed X is the cause. When I changed Y, the behavior changed to Z.
```

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`


## Self-Review (Mandatory)

Before completing, verify investigation quality:

### Scope Verification

**Did you scope the problem with rg before concluding?**

| Check | How | If Failed |
|-------|-----|-----------|
| **Problem scoped** | Ran `rg` to find all occurrences of the pattern being investigated | Run now, update findings |
| **Scope documented** | Investigation states "Found X occurrences in Y files" | Add concrete numbers |
| **Broader patterns checked** | Searched for variations/related patterns | Document what else exists |

**Examples:**
```bash
# Investigating "how does auth work?"
rg "authenticate|authorize|jwt|token" --type py -l  # Scope: which files touch auth

# Investigating "why does X fail?"
rg "error.*X|X.*error" --type py  # Find all error handling for X

# Investigating "where is config loaded?"
rg "config|settings|env" --type py -l  # Scope the config surface area
```

**Why this matters:** Investigations that don't scope the problem often miss the full picture. "I found one place that does X" is less useful than "X happens in 3 files: A, B, C."

---

### Investigation-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Real test performed** | Not "reviewed code" or "analyzed logic" | Go back and test |
| **Conclusion from evidence** | Based on test results, not speculation | Rewrite conclusion |
| **Question answered** | Original question has clear answer | Complete the investigation |
| **Reproducible** | Someone else could follow your steps | Add detail |

### Self-Review Checklist

- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
- [ ] **Question answered** - Investigation addresses the original question
- [ ] **File complete** - All sections filled (not "N/A" or "None")
- [ ] **D.E.K.N. filled** - Replaced placeholders in Summary section (Delta, Evidence, Knowledge, Next)
- [ ] **NOT DONE claims verified** - If claiming something is incomplete, searched actual files/code to confirm (not just artifact claims)

### Discovered Work Check

*During this investigation, did you discover any of the following?*

| Type | Examples | Action |
|------|----------|--------|
| **Bugs** | Broken functionality, edge cases that fail | `bd create "description" --type bug` |
| **Technical debt** | Workarounds, code that needs refactoring | `bd create "description" --type task` |
| **Enhancement ideas** | Better approaches, missing features | `bd create "description" --type feature` |
| **Documentation gaps** | Missing/outdated docs | Note in completion summary |

*When creating issues for discovered work, apply triage labels:*

| Confidence | Label | When to use |
|------------|-------|-------------|
| High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Lower | `triage:review` | Uncertain scope, needs orchestrator input |

Example:
```bash
bd create "Bug: edge case in validation" --type bug
bd label <issue-id> triage:ready  # or triage:review
```

**Checklist:**
- [ ] **Reviewed for discoveries** - Checked investigation for patterns, bugs, or ideas beyond original scope
- [ ] **Tracked if applicable** - Created beads issues for actionable items (or noted "No discoveries")
- [ ] **Included in summary** - Completion comment mentions discovered items (if any)

**If no discoveries:** Note "No discovered work items" in completion comment. This is common and acceptable.

**Why this matters:** Investigations often reveal issues beyond the original question. Beads issues ensure these discoveries surface in SessionStart context rather than getting buried in investigation files.

### Document in Investigation File

At the end of your investigation file, add:

```markdown
## Self-Review

- [ ] Real test performed (not code review)
- [ ] Conclusion from evidence (not speculation)
- [ ] Question answered
- [ ] File complete

**Self-Review Status:** PASSED / FAILED
```

**Only proceed to commit after self-review passes.**


---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kn decide` | `kn decide "Use Redis for sessions" --reason "Need distributed state"` |
| Tried something that failed | `kn tried` | `kn tried "SQLite for sessions" --failed "Race conditions"` |
| Discovered a constraint | `kn constrain` | `kn constrain "API requires idempotency" --reason "Retry logic"` |
| Found an open question | `kn question` | `kn question "Should we rate-limit per-user or per-IP?"` |

**Quick checklist:**
- [ ] Reflected on session: What did I learn that the next agent should know?
- [ ] Externalized at least one item via `kn` command

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."

---

## Completion

Before marking complete:

1. Self-review passed (see above)
2. **Leave it Better:** At least one `kn` command run OR noted as not applicable
3. `## Test performed` has a real test (not "reviewed code" or "analyzed logic")
4. `## Conclusion` is based on test results
5. D.E.K.N. summary filled (Delta, Evidence, Knowledge, Next)
6. `git add` and `git commit` the investigation file
7. Link artifact to beads issue: `kb link <investigation-file> --issue <beads-id>`
8. Report via beads: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"`
9. Close the beads issue: `bd close <beads-id> --reason "conclusion summary"`
10. Run `/exit` to close session

---

**Remember:** The old investigation system produced confident wrong conclusions. The fix is simple: test before concluding.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-7vsp "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

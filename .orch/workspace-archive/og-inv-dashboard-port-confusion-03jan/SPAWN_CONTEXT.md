TASK: Dashboard port confusion: orch serve runs on random ports (currently 3348), conflicts with other services on 5188. Why isn't there a consistent port? What's the intended architecture for orch serve vs opencode serve? Why do we keep hitting this?

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "dashboard"

### Constraints (MUST respect)
- Dashboard event panels max-h-64 for visibility without overwhelming layout
  - Reason: Doubled from 32px provides better event scanning while preserving agent grid visibility
- Dashboard must be fully usable at 666px width (half MacBook Pro screen). No horizontal scrolling. All critical info visible without scrolling.
  - Reason: Primary workflow is orchestrator CLI + dashboard side-by-side on MacBook Pro. Minimum width constraint - should expand gracefully on larger displays.
- OpenCode serve requires --port 4096 flag
  - Reason: Default is random port. Daemon, orch CLI, and dashboard all expect 4096.
- Dylan doesn't interact with dashboard directly - orchestrator uses Glass for all browser interactions
  - Reason: Pressure Over Compensation: forces gaps in Glass tooling to surface rather than routing around them
- UI features require browser verification before marking complete
  - Reason: Post-mortem og-work-dashboard-two-modes-27dec: SSR hydration bug passed build/typecheck but failed in browser; agent documented 'not browser-tested' but had no gate to force validation

### Prior Decisions
- Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management
  - Reason: Balances operational visibility (active work always visible) with historical debugging (expand sections as needed) and UI clarity (collapsed sections reduce clutter). Only approach that satisfies all three user contexts: development focus, debugging history, and health monitoring.
- orch serve displayThreshold should match orch status (30min)
  - Reason: 6h threshold showed 25 stale sessions while orch status showed 0, causing dashboard noise
- kb context uses keyword matching, not semantic understanding - 'how would the system recommend...' questions require orchestrator synthesis
  - Reason: Tested kb context with various query formats: keyword queries (swarm, dashboard) returned results, but semantic queries (swarm map sorting, how should dashboard present agents) returned nothing. The pattern reveals desire for semantic query answering that would require LLM-based RAG.
- 24-hour threshold for Recent vs Archive in dashboard
  - Reason: Balances operational focus (recent work visible) with history access (older work collapsed but accessible)
- Usage display color thresholds: green <60%, yellow 60-80%, red >80%
  - Reason: Matches established UX patterns for warning levels, consistent with how other dashboards signal utilization
- Dashboard integrations tiered: Beads+Focus high, Servers medium, KB/KN skip
  - Reason: Operational awareness purpose means actionable work queue > reference material
- Dashboard account name lookup uses email reverse-mapping from accounts.yaml
  - Reason: Provides meaningful account identifier (personal/work) instead of ambiguous email prefix
- Dashboard agent status derived from beads phase, not session time
  - Reason: Phase: Complete from beads comments is authoritative for completion status, session idle time is secondary
- Dashboard beads stats use bd stats --json API call
  - Reason: Provides comprehensive issue statistics with ready/blocked/open counts in single call
- Dashboard panel additions follow pattern: API endpoint in serve.go -> Svelte store -> page.svelte integration
  - Reason: Established during focus/beads/servers panel additions Dec 24
- Active agents should use stable sort (spawned_at) to prevent grid reordering from SSE updates
  - Reason: updated_at changes every second for active agents, causing constant visual churn in the dashboard grid
- Dashboard progressive disclosure is already fully implemented
  - Reason: Active/Recent/Archive sections with 24h threshold, localStorage persistence, count badges, and preview text all exist in current codebase
- Dashboard project filter follows skill filter pattern - state var, unique extraction, apply function, dropdown UI
  - Reason: Consistent pattern makes future filter additions predictable and maintainable
- Dashboard uses SYNTHESIS.md as fallback for untracked agent completion detection
  - Reason: Untracked agents have fake beads IDs that won't match real issues, so Phase: Complete check fails - workspace-based detection is the reliable fallback
- Dashboard is_processing visual indicators require status === 'active' check
  - Reason: SSE session.status events may not clear is_processing flag when agent completes, causing stale pulsing animation. Defensive check ensures only active agents show processing state.
- Dashboard gets lightweight acknowledgment actions (approve, reject, mark reviewed, priority bump); orchestrator keeps reasoning actions (spawn, synthesize, scope)
  - Reason: Control separation: if it requires judgment, orchestrator. If it's confirmation of something already decided, dashboard. Reduces context switching without blurring the thinking/seeing boundary.
- Dashboard daemon indicator uses file-based status (daemon-status.json) over IPC
  - Reason: Simple, decoupled approach - daemon writes status to file on each poll, dashboard reads it. No process coupling, works when daemon restarts, allows monitoring from any process.
- Dashboard synthesis review shows synthesis inline with actionable issue creation
  - Reason: Enables orchestrators to act on synthesis recommendations without leaving dashboard UI
- Error pattern analysis uses normalized message matching for grouping similar errors
  - Reason: Enables dashboard to show recurring patterns by truncating to 100 chars and trimming whitespace
- Dashboard queue visibility should use expandable section under stats bar
  - Reason: Consistent with CollapsibleSection pattern, respects 666px constraint, no context switching required

### Related Investigations
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Add /api/agentlog endpoint to serve.go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- Add Usage/Capacity Tracking to Account Package
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-usage-capacity-tracking-account.md
- Implement Synthesis Card Display in Swarm Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-synthesis-card-display-swarm.md
- Scaffold beads-ui v2 (Bun + SvelteKit 5 + shadcn-svelte)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scaffold-beads-ui-v2-bun.md
- Tmux Concurrent Epsilon Spawn Capability
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-epsilon.md
- Dashboard Agent Activity Visibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-dashboard-needs-better-agent-activity.md
- Deep Post-Mortem on 24 Hours of Development Chaos
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md
- Failure Mode Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md
- Dashboard Shows 0 Agents Despite API Returning 209
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-debug-dashboard-shows-0-agents-despite-api-returning-209.md
- Dashboard Agent Activity Visibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-dashboard-agent-activity-visibility.md
- Ideal Cross-Repo Setup for Dylan's Orchestration Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-design-ideal-cross-repo-setup.md
- orch handoff generates stale/incorrect data
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-orch-handoff-generates-stale-incorrect.md
- Review 18 Open Investigations from kb reflect
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-review-19-open-investigations-kb.md
- Audit Swarm Dashboard Web UI
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-audit-swarm-dashboard-web-ui.md
- Design Question Should Orch Servers
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md
- Design Question Should Swarm Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md
- Explore Options Centralized Server Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md
- Add Api Usage Endpoint Serve
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-add-api-usage-endpoint-serve.md
- Add Beads Stats Dashboard Stats
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-add-beads-stats-dashboard-stats.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

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
2. **SET UP investigation file:** Run `kb create investigation dashboard-port-confusion-orch-serve` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-dashboard-port-confusion-orch-serve.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-dashboard-port-confusion-03jan/SYNTHESIS.md
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

---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed, and whether you tested. Key discipline - you cannot conclude without testing.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 68f6b8415809 -->
<!-- Source: .skillc -->
<!-- To modify: edit files in .skillc, then run: skillc build -->
<!-- Last compiled: 2025-12-30 10:08:22 -->

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
2. **IMMEDIATE CHECKPOINT (before ANY exploration):**
   - Fill in your **Question** from SPAWN_CONTEXT
   - Add first finding: `### Finding 1: Starting approach` with your planned first step
   - **Commit immediately:** `git add .kb/investigations/*.md && git commit -m "investigation: {slug} - checkpoint"`
   - This ensures if you die, there's a trail of what you were attempting
3. Try things, observe what happens (add findings progressively)
4. **Run a test to validate your hypothesis**
5. Fill conclusion only if you tested
6. Final commit

**Why the immediate checkpoint?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, you leave only an empty template with no record of what was attempted.

## Error Recovery

**If you encounter a fatal error during exploration:**

1. **Before doing anything else**, add a finding to your investigation file:
   ```markdown
   ### Finding N: ERROR - [brief description]
   
   **Error:** [Full error message]
   
   **Context:** [What you were attempting when error occurred]
   
   **Significance:** [Why this blocks progress or what it reveals]
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

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.

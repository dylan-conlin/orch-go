TASK: Dashboard /api/agents still takes 20s on cold cache despite time/project filters - filters are passed (?since=12h&project=orch-go) but first request is 20.52s. Second request hits cache (111ms). Need to investigate why cold path is still slow. Screenshot shows Following toggle active, 12h filter applied. Prior fixes (Jan 7 f87bedf4) added investigation cache and restored 2h threshold but this regression suggests filters may not be applied before expensive operations.

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
- High patch density in a single area (5+ fix commits, 10+ conditions, duplicate logic) signals missing coherent model - spawn architect before more patches
  - Reason: Dashboard status logic accumulated 10+ conditions from incremental agent patches without anyone stepping back to design. Each patch was locally correct but globally incoherent. Discovered Jan 4 2026 after weeks of completion bugs.
- Dashboard SSE connections can exhaust HTTP/1.1 browser connection pool (6 per origin)
  - Reason: Two SSE endpoints (/api/events, /api/agentlog) consume long-lived connections. When combined with slow API responses, fetch requests queue as pending indefinitely. Permanent fix needed: HTTP/2, multiplexed SSE, or WebSocket.
- Hotspot thresholds: 5+ fix commits in 4 weeks OR 3+ investigations on same topic
  - Reason: Conservative thresholds based on dashboard status case study; can be tuned via --threshold flag after gathering real-world data
- High patch density in a single area (5+ fix commits, 10+ conditions) signals missing coherent model - spawn architect before more patches
  - Reason: Dashboard status logic accumulated 10+ conditions from incremental patches without stepping back to design

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





🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-gi3ty "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-gi3ty "Phase: Complete - [1-2 sentence summary of deliverables]"`
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

**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Surface it first: `bd comment orch-go-gi3ty "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation dashboard-api-agents-still-takes` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-dashboard-api-agents-still-takes.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-gi3ty "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-dashboard-api-agents-07jan-4f51/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-gi3ty**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-gi3ty "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-gi3ty "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-gi3ty "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-gi3ty "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-gi3ty "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-gi3ty`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (issue-creation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: issue-creation
skill-type: procedure
description: Transform symptoms into rich beads issues with Problem-Solution-Evidence structure. Investigates root cause before creating issue - the issue IS the deliverable, not an investigation file.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 29e003113d68 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/issue-creation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/issue-creation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/issue-creation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-07 14:44:12 -->


## Summary

**Purpose:** Transform vague symptoms into high-quality beads issues through targeted investigation.

---

# Issue Creation Skill

**Purpose:** Transform vague symptoms into high-quality beads issues through targeted investigation.

## The Core Insight

Quality issues come from understanding BEFORE creating, not from validation AFTER.

**Yegge's issues average 609 characters** with Problem-Solution-Evidence structure because someone understood the problem deeply first. Our goal: match that quality through systematic investigation.

## When to Use This Skill

| Use issue-creation | Use bd create directly |
|--------------------|------------------------|
| Symptom reported ("X seems broken") | Obvious, trivial issue |
| Root cause unclear | Already know exactly what's wrong |
| Multiple possible causes | Single clear fix needed |
| Worth 15-30 min investigation | <5 min to document |

**Rule:** If you'd need to investigate before fixing, use this skill.


## Workflow

### Phase 1: Understand the Symptom (5-10 min)

1. **Document the symptom exactly as reported**
   - What behavior was observed?
   - What was expected instead?
   - Any context (when it happens, frequency)?

2. **Reproduce the symptom (REQUIRED for bugs)**
   
   For bug reports, reproduction is mandatory before issue creation:
   
   ```bash
   # Attempt to trigger the symptom
   # Use EXACT commands/steps from the report
   # Document what you tried and what happened
   ```
   
   **Record working reproduction steps:**
   - Minimum: Command + observed output
   - Better: Full environment context + steps + output
   - Best: Deterministic repro with expected vs actual comparison
   
   **If reproduction succeeds:** Continue to Phase 2 with documented repro steps.
   
   **If reproduction fails after 3+ attempts:**
   - Document what you tried
   - Create as `--type investigation` instead of bug (see "Non-Reproducible Issues" below)
   - OR use `--no-repro --reason "explanation"` if you're confident it's still a bug

3. **Scope the problem area**
   ```bash
   # Find related code
   rg "keyword" --type py -l
   
   # Understand the area
   Read relevant files
   ```

### Phase 2: Investigate Root Cause (10-20 min)

**Goal:** Understand WHY, not just WHERE.

1. **Trace the code path**
   - Start from symptom manifestation
   - Follow the logic backward
   - Document each file and relevant lines

2. **Form hypothesis**
   - What could cause this behavior?
   - What evidence would confirm/deny?

3. **Test hypothesis**
   ```bash
   # Run specific tests, check logs, reproduce with variations
   ```

4. **Document file references**
   - `src/orch/spawn.py:142` - Where X happens
   - `src/orch/registry.py:89` - Related state management

### Phase 3: Create the Issue (5-10 min)

**Use the P-S-E structure:**

Write the full description first, then create with it inline:

```bash
# For bugs: MUST include --repro with verified reproduction steps
bd create "Clear title describing the problem" --type bug \
  --repro "1. Run 'orch status' 2. Observe: shows 27 active 3. Expected: 4 active" \
  --description "## Problem

[Full P-S-E content here - see template below]"

# For non-bug types: --repro not required
bd create "Clear title" --type feature --description "..."
```

**If bug cannot be reproduced (after 3+ documented attempts):**

```bash
# Option 1: Create as investigation instead (preferred)
bd create "Investigate: [symptom description]" --type investigation \
  --description "## Symptom

[What was reported]

## Reproduction Attempts

[What you tried, why it didn't reproduce]

## Next Steps

[What investigation is needed]"

# Option 2: Create as bug with skip reason (use sparingly)
bd create "title" --type bug \
  --no-repro --reason "One-time crash, no logs available" \
  --description "..."
```

**Alternative for very long descriptions:**

```bash
# Create issue first (with repro for bugs)
bd create "Clear title describing the problem" --type bug \
  --repro "Command X produces Y instead of Z"

# Add description via comment (non-interactive)
bd comments add <issue-id> "## Problem

[Full P-S-E content here]

## Evidence

[File references and reproduction steps]

## Context

[Impact and related issues]"
```

**Issue Template:**

```markdown
## Problem

[What is broken or wrong. Specific, observable behavior.]

[When/how it manifests. Frequency, conditions.]

## Evidence

[File:line references where the issue exists]
- `src/file.py:123` - [What's wrong here]
- `src/other.py:456` - [Related code]

[Reproduction steps if applicable]
1. Do X
2. Do Y
3. Observe Z (expected: W)

[Error messages, logs, or output]

## Context

[Why this matters. Impact on users/system.]

[Any related issues or prior attempts.]
```

### Phase 4: Apply Labels

**Confidence-based labeling:**

| Label | When to use |
|-------|-------------|
| `triage:ready` | High confidence in diagnosis, clear fix path |
| `triage:review` | Uncertain about root cause or fix approach |

```bash
# High confidence - daemon can auto-spawn
bd label <issue-id> triage:ready

# Lower confidence - human reviews first
bd label <issue-id> triage:review
```

**Additional labels as appropriate:**
- Type: `bug`, `feature`, `task`
- Priority: `P1`, `P2`, `P3` (if clearly emergent)
- Area: `auth`, `ui`, `api` etc. (project-specific)

## Non-Reproducible Issues

When a reported bug cannot be reproduced after thorough attempts:

**Decision tree:**

1. **Can you identify the root cause anyway?** (e.g., obvious race condition in code)
   → Create bug with `--no-repro --reason "Race condition identified in code, timing-dependent"`

2. **Is there strong evidence it's real?** (logs, screenshots, multiple reports)
   → Create bug with `--no-repro --reason "Evidence: [logs/screenshots/reports]"`

3. **Is more investigation needed?**
   → Create investigation instead:
   ```bash
   bd create "Investigate: [symptom]" --type investigation \
     --description "## Reported Symptom
   [What user reported]
   
   ## Reproduction Attempts
   - Tried: [approach 1] → Result: [outcome]
   - Tried: [approach 2] → Result: [outcome]
   - Tried: [approach 3] → Result: [outcome]
   
   ## Hypothesis
   [What might be causing this]
   
   ## Next Steps
   [What investigation is needed]"
   ```

4. **Is it likely user error or environment-specific?**
   → Ask reporter for more context before creating issue

**The key insight:** A bug you can't reproduce is harder to fix. Creating it as an investigation first ensures someone can dig deeper before committing to a fix approach.

## Quality Checklist

Before completing, verify:

- [ ] **Problem clear:** Someone unfamiliar could understand what's wrong
- [ ] **Evidence concrete:** File:line references, not just "somewhere in X"
- [ ] **Reproduction verified:** For bugs, --repro provided with tested steps (or --no-repro with valid reason)
- [ ] **Scoped:** Clear boundaries of what this issue covers
- [ ] **Labeled:** Appropriate triage label applied
- [ ] **>200 chars:** Rich description, not just a title

## Common Failures

**Shallow issue (DON'T):**
```markdown
Title: Fix polling bug
Description: Polling isn't working correctly.
```

**Rich issue (DO):**
```markdown
Title: Polling controller returns stale data after 60s timeout

## Problem

The polling controller in `src/controllers/poll.py` returns 
cached data even after the cache TTL expires. Users see 
outdated information until they hard-refresh.

## Evidence

- `src/controllers/poll.py:89` - Cache check doesn't account for TTL
- `src/cache/redis.py:142` - TTL is set but not used in fetch

Reproduction:
1. Load dashboard
2. Wait 90 seconds (past 60s TTL)
3. Trigger poll refresh
4. Observe: Data unchanged despite backend updates

## Context

Affects all users relying on real-time data. Discovered while 
investigating support ticket #4521.
```


## Completion

When finished (complete in this EXACT order):

1. Verify quality checklist passes
2. Note the created issue ID
3. Report via beads (FIRST - before any final commits):
   ```bash
   bd comment <spawn-issue-id> "Phase: Complete - Created issue <new-issue-id> with P-S-E structure"
   ```
4. Commit any remaining changes
5. Run `/exit`

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility.

**Note:** `bd close` is removed from agent responsibilities - only the orchestrator closes issues via `orch complete`.

## The Issue IS the Deliverable

Unlike investigation skill (produces investigation file), this skill produces a beads issue directly. The investigation happens, but it's internalized - the issue captures the understanding without a separate artifact.

**When to switch to investigation skill instead:**
- Understanding needed beyond single issue (architectural, cross-cutting)
- Knowledge should persist even if issue is closed
- Exploration without clear actionable outcome

---

**Remember:** 15-30 minutes of understanding BEFORE creating the issue produces dramatically better results than trying to add detail AFTER.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-gi3ty "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

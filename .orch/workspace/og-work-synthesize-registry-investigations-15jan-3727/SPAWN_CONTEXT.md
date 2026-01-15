TASK: Synthesize registry investigations (11)

## Synthesis Needed

Topic **registry** has accumulated 11 investigations that may benefit from consolidation.

### Investigations to synthesize:

- 2025-12-20-inv-plan-refactoring-pkg-registry-act.md
- 2025-12-21-inv-agents-being-marked-completed-registry.md
- 2025-12-21-inv-audit-all-registry-usage-orch.md
- 2025-12-21-inv-implement-port-allocation-registry-orch.md
- 2025-12-21-inv-registry-abandon-doesn-remove-agent.md
- 2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md
- 2026-01-06-inv-registry-population-issues-orch-status.md
- 2026-01-07-inv-investigate-registry-population-failures-root.md
- 2026-01-07-inv-registry-file-self-describing-header.md
- 2026-01-09-inv-add-mode-field-registry-schema.md
- 2026-01-08-inv-test-registry-fix-verify-slot.md

### Chronicle Timeline

```
2025-12-20 [INV] Add CLI Commands for Focus, Drift, and Next
2025-12-20 [INV] Add orch review command for batch completion wo...
2025-12-20 [INV] Add Wait Command to orch-go
2025-12-20 [INV] KB Search vs Grep Benchmark
2025-12-20 [INV] Compare orch-cli (Python) vs orch-go Features
... (run kb chronicle for full timeline)
```

### Suggested Action

1. Run `kb chronicle "registry"` to understand evolution
2. Identify patterns and contradictions across investigations
3. Run `kb create guide "registry"` to create authoritative reference


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "synthesize registry investigations"

### Constraints (MUST respect)
- tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: If both are stale/missing, fallback fails despite window existing
- tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: Both paths needed for resilience; if both stale/missing, fallback fails despite window existing
- Tmux fallback requires at least one valid path: current registry window_id OR beads ID in window name format [beads-id]
  - Reason: Dual dependency failure causes fallback to fail even when window exists (discovered iteration 5, confirmed iteration 10)
- orch tail tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: Dual-dependency failure causes fallback to fail when both are stale/missing
- orch-go agent state exists in four layers (OpenCode memory, OpenCode disk, registry, tmux)
  - Reason: Each layer has independent lifecycle - cleanup must touch all layers or ghosts accumulate
- Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads
  - Reason: Investigation found all registry data can be derived from primary sources
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- D.E.K.N. 'Next:' field must be updated when marking Status: Complete
  - Reason: Prevents stale investigations that mislead future agents
- LLM guidance compliance requires signal balance - overwhelming counter-patterns (56:13 ratio) drowns specific exceptions
  - Reason: Investigation found orchestrator skill has 4:1 ask-vs-act signal ratio causing autonomy guidance to fail
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).

### Prior Decisions
- Registry respawn workflow uses slot reuse pattern
  - Reason: Preserves single-entry-per-ID invariant while enabling abandon→respawn lifecycle
- Registry updates must happen before beads close in orch complete
  - Reason: Prevents inconsistent state where beads shows closed but registry shows active
- OpenCode ListSessions WITH x-opencode-directory header returns disk sessions, WITHOUT returns in-memory
  - Reason: Finding from investigation - explains 2 vs 238 session count discrepancy
- Investigations live in .kb/ not workspaces
  - Reason: kb context discoverability essential; SYNTHESIS.md bridges via investigation_path pointer
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- orch-go is primary CLI, orch-cli (Python) is reference/fallback
  - Reason: Go provides better primitives (single binary, OpenCode HTTP client, goroutines); Python taught requirements through 27k lines and 200+ investigations
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- Session_id stored in workspace file not registry
  - Reason: Co-locates data with workspace, single writer, no lock contention
- kb reflect uses single command with --type flag for four reflection modes
  - Reason: Consistent with kb pattern, most discoverable, extensible. Maps directly to signals from ws4z investigations.
- Self-reflection is signal-triggered not time-scheduled
  - Reason: Density thresholds (3+ investigations) produce actionable signals; time intervals (weekly review) produce noise. Per ws4z.8 investigation.
- orch clean to remove ghost sessions automatically
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

### Related Investigations
- Registry Abandonment Workflow Validate Simple
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md
- Registry Usage Audit in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md
- Synthesize Synthesis Investigations (26 Total)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-synthesis-investigations-26-synthesis.md
- Post-Synthesis Investigation Archival Workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-design-post-synthesis-investigation-archival.md
- Synthesis of 15 Skill Investigations (Dec 2025 - Jan 2026)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-synthesize-skill-investigations-15-synthesis.md
- Synthesize Session Investigations (10)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-session-investigations-10-synthesis.md
- Synthesis of 10 Completion Investigations
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-completion-investigations-10-synthesis.md
- Investigate Registry Population Failures Root
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-investigate-registry-population-failures-root.md
- Synthesize Orchestrator Investigations 28 Synthesis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-synthesize-orchestrator-investigations-28-synthesis.md
- Synthesize Orchestrator Investigations
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-synthesize-orchestrator-investigations.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.





🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-pi2k2 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-pi2k2 "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-pi2k2 "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation synthesize-registry-investigations-11-synthesis` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-synthesize-registry-investigations-11-synthesis.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-pi2k2 "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-synthesize-registry-investigations-15jan-3727/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-pi2k2**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-pi2k2 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-pi2k2 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-pi2k2 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-pi2k2 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-pi2k2 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-pi2k2`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (kb-reflect)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: kb-reflect
skill-type: procedure
description: Use when triaging knowledge hygiene findings from kb reflect output - handles synthesis (consolidate investigations), promote (kb quick entries to decisions), stale (uncited decisions), drift (outdated constraints), and open (pending actions). Spawnable for session start, after major work, or weekly maintenance.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: c45ada8cc8a9 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/kb-reflect/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/kb-reflect/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/kb-reflect/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-14 16:58:17 -->


## Summary

**Purpose:** Systematic review and triage of `kb reflect` output to maintain knowledge hygiene.

---

# KB Reflect Triage

**Purpose:** Systematic review and triage of `kb reflect` output to maintain knowledge hygiene.

## When to Run

| Trigger | Frequency | Focus |
|---------|-----------|-------|
| **Session start** | Every session | Quick scan for high-priority items (stale, drift) |
| **After major work** | Post-feature/post-sprint | Synthesis opportunities, open actions |
| **Weekly maintenance** | Weekly | Full triage of all categories |
| **Pre-release** | Before milestones | Drift detection, stale cleanup |

**Quick check (session start):**
```bash
kb reflect --type stale --limit 3    # Uncited decisions
kb reflect --type drift --limit 3    # Constraint violations
```

**Full triage:**
```bash
kb reflect                           # All categories
orch daemon reflect                  # Run and save for dashboard
```


## Finding Types Overview

| Type | What It Means | Default Action |
|------|---------------|----------------|
| **synthesis** | 3+ investigations on same topic | Consolidate into decision/guide |
| **promote** | kb quick entry worth preserving long-term | Promote to kb decision |
| **stale** | Decision with 0 citations, >7 days | Archive or refresh |
| **drift** | CLAUDE.md constraint diverging from practice | Update or remove |
| **open** | Investigation with pending Next: action | Complete or close |


## Decision Tree: Synthesis Findings

**Trigger:** Topic has 3+ investigations.

```
Topic with 3+ investigations
│
├─ Are findings still relevant?
│  ├─ NO → Archive older investigations
│  │       Move to .kb/investigations/archived/
│  │       (Keep 1-2 most recent if any value)
│  │
│  └─ YES → Do they contradict or build on each other?
│           │
│           ├─ CONTRADICT → Create decision record
│           │  Which approach is correct?
│           │  Document reasoning, supersede losers
│           │
│           └─ BUILD → Create chronicle/guide
│                      kb chronicle "topic"
│                      Extract common patterns
│                      Consider kb create guide
```

**Actions:**
```bash
# View evolution of topic
kb chronicle "topic"

# Create guide if reusable pattern emerged
kb create guide "topic-patterns"

# Create decision if choice was made
kb create decision "chose-approach-for-topic"

# Archive obsolete investigations
mkdir -p .kb/investigations/archived
git mv .kb/investigations/2025-*-old-topic.md .kb/investigations/archived/
```

**Consolidation vs Archiving:**

| If investigations show... | Action |
|---------------------------|--------|
| Same conclusion, different evidence | Keep 1 best, archive rest |
| Evolution of understanding | Create guide showing progression |
| Contradictory findings | Create decision resolving conflict |
| Obsolete approaches | Archive with note why obsolete |


## Decision Tree: Promote Findings

**Trigger:** kb quick entry worth long-term preservation.

```
kb quick entry flagged for promotion
│
├─ Is this a one-time decision or ongoing constraint?
│  ├─ ONE-TIME → Does it have architectural impact?
│  │  ├─ YES → kb promote <id> → creates decision
│  │  └─ NO → Leave as kb quick entry (sufficient)
│  │
│  └─ ONGOING → Is it project-specific or universal?
│              ├─ PROJECT → Add to project CLAUDE.md
│              └─ UNIVERSAL → Add to global CLAUDE.md
│                            OR kb create decision
```

**Actions:**
```bash
# Promote kb quick entry to kb decision
kb promote <kb-id>

# View kb quick entry before promoting
kb quick get <id>

# Add constraint to CLAUDE.md manually if universal
# (kb quick entries are quick; CLAUDE.md is authoritative)
```

**Promotion criteria:**
- Entry was referenced 3+ times in subsequent work
- Entry prevents a recurring mistake
- Entry represents a significant architectural choice
- Entry should outlast the project (universal learning)


## Decision Tree: Stale Findings

**Trigger:** Decision with 0 citations, older than 7 days.

```
Stale decision detected
│
├─ Is the decision still valid?
│  ├─ NO (superseded) → Archive or add Superseded-By
│  │  Option A: Move to archived/
│  │  Option B: Add **Superseded-By:** header
│  │
│  ├─ YES but forgotten → Refresh references
│  │  - Add citations to relevant code/docs
│  │  - Link from CLAUDE.md if applicable
│  │  - Update "Last reviewed" date
│  │
│  └─ YES and actively followed → No action needed
│     (Just hasn't been formally cited yet)
```

**Actions:**
```bash
# Archive superseded decision
mkdir -p .kb/decisions/archived
git mv .kb/decisions/2025-*-old-decision.md .kb/decisions/archived/

# Add superseded header (keeps history visible)
# Add to file: **Superseded-By:** path/to/new/decision.md

# Refresh by adding citation
# In code or docs, add: "See decision: path/to/decision.md"
```

**Archive vs Keep:**

| Situation | Disposition |
|-----------|-------------|
| Approach was abandoned | Archive with reason |
| Replaced by better approach | Add Superseded-By header |
| Still valid, just not cited | Keep, add citations if important |
| Was experimental, didn't pan out | Archive with lessons learned |


## Decision Tree: Drift Findings

**Trigger:** CLAUDE.md constraint may conflict with actual practice.

```
Constraint drift detected
│
├─ Is the constraint correct (practice is wrong)?
│  ├─ YES → Create issue to fix practice
│  │        bd create "Fix drift: practice violates constraint X"
│  │        Label: triage:ready
│  │
│  └─ NO → Is the practice correct (constraint outdated)?
│          ├─ YES → Update CLAUDE.md
│          │        Remove or modify constraint
│          │        Document why in commit message
│          │
│          └─ UNCLEAR → Investigate
│                       orch spawn investigation "is X constraint still valid"
```

**Actions:**
```bash
# Create issue if practice should change
bd create "Fix drift: code violates constraint X" --type task
bd label <id> triage:ready

# Update CLAUDE.md if constraint outdated
# Edit CLAUDE.md, commit with rationale

# Investigate if unclear
orch spawn investigation "validate constraint: X"
```

**Common drift patterns:**
- Constraint was aspirational, never enforced
- Constraint was valid for old architecture, not current
- Constraint conflicts with newer, higher-priority constraint
- Practice evolved but constraint wasn't updated


## Decision Tree: Open Findings

**Trigger:** Investigation has Next: action that wasn't completed.

```
Investigation with open Next: action
│
├─ Is the action still relevant?
│  ├─ NO → Update investigation
│  │       Change Next: to "None - superseded by X"
│  │       Set Status: Complete
│  │
│  └─ YES → Why wasn't it completed?
│           │
│           ├─ FORGOTTEN → Create beads issue
│           │  bd create "Complete: [action from investigation]"
│           │  Reference investigation in description
│           │
│           ├─ BLOCKED → Document blocker
│           │  Update investigation Status: Paused
│           │  Add **Blocker:** field
│           │
│           └─ IN PROGRESS → Update status
│                            Set Status: In Progress
│                            (Investigation should track this)
```

**Actions:**
```bash
# Create issue for forgotten action
bd create "Complete: [action]" --type task -d "From investigation: [path]"

# Update investigation to close it
# Edit investigation file:
# - Next: None (superseded by decision X)
# - Status: Complete

# Mark as paused with blocker
# Edit investigation file:
# - Status: Paused
# - **Blocker:** [what's blocking]
```


## Proposed Actions (REQUIRED)

**Purpose:** Transform findings into actionable proposals that orchestrator can approve/reject.

After triaging all findings, create a **Proposed Actions** section in your investigation file with structured proposals:

### Proposal Format

```markdown
## Proposed Actions

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| A1 | `.kb/investigations/2025-01-15-old-topic.md` | Superseded by decision X | [ ] |
| A2 | `.kb/decisions/2025-02-01-obsolete.md` | No longer valid (approach abandoned) | [ ] |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | decision | "Use approach X for Y" | Topic has 5 investigations, decision needed | [ ] |
| C2 | guide | "Patterns for Z" | Recurring pattern from 3+ investigations | [ ] |
| C3 | issue | "Fix drift: constraint X outdated" | Practice diverged from constraint | [ ] |

### Promote Actions
| ID | kb-id | To | Reason | Approved |
|----|-------|-------|--------|----------|
| P1 | `kb-abc123` | decision | Architectural impact, referenced 4 times | [ ] |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `CLAUDE.md` line 45 | Remove constraint X | Practice is correct, constraint outdated | [ ] |
| U2 | `.kb/investigations/2025-03-01-topic.md` | Set Status: Complete | Next action completed | [ ] |
```

### Generating Proposals

**For each finding type, apply the decision tree and output a proposal:**

| Finding Type | If Decision Tree Result Is... | Proposal Type |
|--------------|-------------------------------|---------------|
| synthesis | Archive older investigations | Archive |
| synthesis | Create decision/guide | Create |
| promote | Promote to kb | Promote |
| promote | Add to CLAUDE.md | Update |
| stale | Archive decision | Archive |
| stale | Refresh references | Update |
| drift | Fix practice | Create (issue) |
| drift | Update constraint | Update |
| open | Create beads issue | Create (issue) |
| open | Close investigation | Update |

### Proposal Guidelines

1. **Be specific:** Include exact file paths, line numbers, or issue IDs
2. **Explain why:** Each proposal needs a clear reason
3. **One action per row:** Don't combine multiple changes
4. **Order by priority:** High-impact proposals first
5. **Include draft content:** For Create actions, provide draft title and 1-sentence description

### Example Proposals

**From synthesis finding (3+ investigations on "auth"):**
```markdown
| C1 | decision | "Chose JWT over sessions for auth" | 4 investigations converged on JWT approach, need formal decision | [ ] |
| A1 | `.kb/investigations/2025-01-10-auth-sessions.md` | Superseded by JWT decision | [ ] |
```

**From drift finding (constraint violated):**
```markdown
| C2 | issue | "Fix drift: tests must run before commit" | Code merged without tests 3 times this week | [ ] |
```

**From stale finding (uncited decision):**
```markdown
| A2 | `.kb/decisions/2025-02-01-old-api.md` | API redesigned, decision obsolete | [ ] |
```

### Orchestrator Approval Workflow

After agent completes kb-reflect triage:

1. **Review proposals:** Check each row in Proposed Actions
2. **Mark approved:** Add `x` in Approved column: `[x]`
3. **Adjust if needed:** Modify title, description, or target
4. **Execute:** Agent or orchestrator executes approved proposals
5. **Skip declined:** Leave `[ ]` empty for proposals to skip

### Counting Proposals

At the end of Proposed Actions section, add summary:

```markdown
**Summary:** X proposals (Y archive, Z create, W promote, V update)
**High priority:** [list IDs of urgent proposals]
```


## Investigation Closure Procedure

**When closing an investigation (marking Status: Complete):**

1. **Fill D.E.K.N. summary** at top of file
   - Delta: What was discovered (1 sentence)
   - Evidence: Primary evidence (1 sentence)
   - Knowledge: What was learned (1 sentence)
   - Next: Recommended action or "None"

2. **Set Next: field** to one of:
   - "None" (no follow-up needed)
   - "Close - superseded by [X]" (another investigation covers this)
   - "Implement - see issue [beads-id]" (action tracked elsewhere)

3. **Set Status: Complete**

4. **Link to beads if applicable:**
   ```bash
   kb link <investigation-path> --issue <beads-id>
   ```

5. **Commit the changes:**
   ```bash
   git add .kb/investigations/
   git commit -m "Close investigation: [topic]"
   ```

**Improper closure patterns to avoid:**
- Setting Status: Complete without filling D.E.K.N.
- Leaving Next: as "[action needed]" when setting Complete
- Not committing the closure


## Triage Session Template

Use this structure for a triage session:

```markdown
# KB Reflect Triage - [Date]

## Summary

**Ran:** `kb reflect` at [time]
**Findings:** [X] synthesis, [Y] promote, [Z] stale, [W] drift, [V] open

## Dispositions

### Synthesis (X items)
- [ ] topic-1: [disposition - consolidate/archive/keep]
- [ ] topic-2: [disposition]

### Promote (Y items)
- [ ] kb-id-1: [disposition - promote/keep/skip]

### Stale (Z items)
- [ ] decision-1.md: [disposition - archive/refresh/keep]

### Drift (W items)
- [ ] constraint-1: [disposition - fix practice/update constraint/investigate]

### Open (V items)
- [ ] investigation-1.md: [disposition - create issue/close/update status]

## Proposed Actions

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| A1 | `[file path]` | [reason] | [ ] |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | decision/guide/issue | "[title]" | [1-sentence description] | [ ] |

### Promote Actions
| ID | kb-id | To | Reason | Approved |
|----|-------|-------|--------|----------|
| P1 | `[kb-id]` | decision/CLAUDE.md | [reason] | [ ] |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `[file/location]` | [change] | [reason] | [ ] |

**Summary:** X proposals (Y archive, Z create, W promote, V update)
**High priority:** [list IDs of urgent proposals]

## Actions Taken (After Approval)
- [Action 1]
- [Action 2]

## Next Triage
- Scheduled: [date]
- Focus: [any specific areas]
```


## Self-Review Checklist

Before marking triage complete:

- [ ] All findings reviewed (not just skimmed)
- [ ] Each finding has explicit disposition (action or keep)
- [ ] **Proposed Actions section completed** with structured proposals
- [ ] Each proposal has: target, type, reason, and draft content
- [ ] Proposals are prioritized (high-impact first)
- [ ] Investigation file documents decisions
- [ ] `kb quick` entries created for lessons learned
- [ ] Commits made for file changes

---

## Completion

Before marking complete (in this EXACT order):

1. All findings triaged with explicit disposition
2. **Proposed Actions section completed** with actionable proposals
3. Proposal summary included (X archive, Y create, Z promote, W update)
4. Investigation file completed with D.E.K.N.
5. Report via beads: `bd comment <beads-id> "Phase: Complete - triaged X findings, produced Y proposals for orchestrator review"` (FIRST - before commit)
6. `git add` and `git commit` changes
7. Run `/exit` to close session

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility.

**Key deliverable:** Orchestrator should be able to review proposals and mark `[x]` to approve without re-reading all findings.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-pi2k2 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

TASK: Session-end workflow for orchestrators. Current: 'Landing the Plane' focuses on git/cleanup. Missing: friction audit, gap capture, system reaction check. User says same reflection prompt every session. Question: What should session-end pattern look like? Checklist in skill? Separate section before git cleanup? What triggers it? What gets captured where?

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "session"

### Constraints (MUST respect)
- Concurrent agents trigger TPM throttling at >60% session usage
  - Reason: Observed performance degradation and user reports of hitting limits during swarm operations
- Session idle ≠ agent complete
  - Reason: Agents legitimately go idle during normal operation (loading, thinking, tool execution)
- OpenCode x-opencode-directory header returns ALL disk sessions, not just matching ones
  - Reason: API behavior is counterintuitive - without header returns in-memory only
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state
- OpenCode attach mode only creates sessions after first message received
  - Reason: TUI startup is not session creation - must send prompt before looking up session ID
- Stall detection: session.status=busy for >5min without message.part events indicates hung Claude API call
  - Reason: SSE monitoring pattern - healthy sessions emit regular message.part.updated events
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- No sacred cows - everything is replaceable at the right cost
  - Reason: Load-bearing (Provenance, Session Amnesia) requires equivalent replacement. Implementation (beads, orch, kn, kb) requires something better. Convention (paths, naming) requires friction < value. Nothing is immune from questioning.

### Prior Decisions
- orch-go tmux spawn is fire-and-forget - no session ID capture
  - Reason: opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring.
- OpenCode ListSessions WITH x-opencode-directory header returns disk sessions, WITHOUT returns in-memory
  - Reason: Finding from investigation - explains 2 vs 238 session count discrepancy
- Orchestrator sessions need SESSION_HANDOFF.md
  - Reason: Session amnesia applies to orchestrator work; skillc pattern provides mature template
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- Three-tier temporal model (ephemeral/persistent/operational) organizes artifact placement
  - Reason: Artifacts live where their lifecycle dictates - session-bound to workspace, project-lifetime to kb, work-in-progress to beads
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- Session_id stored in workspace file not registry
  - Reason: Co-locates data with workspace, single writer, no lock contention
- Constraint validity tested by implementation, not age
  - Reason: Constraints become outdated when code contradicts them (implementation supersession), not when time passes. Test with rg before assuming constraint holds.
- Session ID resolution pattern
  - Reason: Commands that need to find agents should use resolveSessionID or the runTail pattern: workspace files first, then API lookup, then tmux fallback
- Post-registry lifecycle uses 4 state sources: OpenCode sessions, tmux windows, beads issues, workspaces
  - Reason: Registry removed due to false positive completion detection; derived lookups replace central state
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- Use inline lineage metadata for project extraction, not centralized registry
  - Reason: Session Amnesia principle requires self-describing artifacts; centralized registry creates fragile external dependency
- Template ownership split by domain
  - Reason: kb-cli owns knowledge artifacts (investigation, decision, guide, research); orch-go owns orchestration artifacts (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT, SESSION_HANDOFF)
- kb templates use inline lineage metadata (extracted-from, supersedes, superseded-by)
  - Reason: Session Amnesia principle requires self-describing artifacts; headers appear in templates at ~/.kb/templates/
- CreateSession API now accepts model parameter for headless spawns
  - Reason: Headless mode was missing model selection capability - added Model field to CreateSessionRequest struct and threaded through CreateSession function to achieve parity with inline/tmux modes
- Tmux session existence is correct abstraction for 'orch servers' status - separate infrastructure from project servers
  - Reason: orch serve is persistent monitoring infrastructure, not ephemeral project server. Tmux lifecycle matches dev server lifecycle. Mixing them creates semantic confusion and false status reporting.
- Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management
  - Reason: Balances operational visibility (active work always visible) with historical debugging (expand sections as needed) and UI clarity (collapsed sections reduce clutter). Only approach that satisfies all three user contexts: development focus, debugging history, and health monitoring.
- OpenCode model selection is per-message, not per-session
  - Reason: Intentional design enabling mid-conversation model switching, cost optimization, and flexibility. Pass model to prompt/message calls, not session creation.
- Use x-opencode-env-ORCH_WORKER header for headless spawns
  - Reason: OpenCode HTTP API doesn't support env vars in request body, but follows x-opencode-* header pattern like x-opencode-directory, so we use x-opencode-env-ORCH_WORKER: 1 header when creating sessions via CreateSession API call
- orch serve displayThreshold should match orch status (30min)
  - Reason: 6h threshold showed 25 stale sessions while orch status showed 0, causing dashboard noise

### Related Investigations
- CLI orch complete Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-complete-command.md
- CLI orch spawn Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-spawn-command.md
- CLI Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-status-command.md
- CLI Project Scaffolding and Build Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-project-scaffolding-build.md
- OpenCode Client Package Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-opencode-session-management.md
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
- OpenCode POC - Spawn Session Via Go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- Add /api/agentlog endpoint to serve.go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- KB Search vs Grep Benchmark
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md
- Beta Flash Synthesis Protocol Design v3
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-beta-flash-synthesis-protocol-design.md
- Compare orch-cli (Python) vs orch-go Features
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md
- Enhance status command with swarm progress
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-status-command-swarm-progress.md

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

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation session-end-workflow-orchestrators-current` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-session-end-workflow-orchestrators-current.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-session-end-workflow-26dec/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (design-session)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: design-session
skill-type: procedure
description: Strategic scoping skill for turning vague ideas into actionable work. Gathers context autonomously, discusses scope interactively, then produces the appropriate artifact (epic, investigation, or decision) based on clarity level.
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: d3ec0b5b1fab -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/design-session/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/design-session/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/design-session/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2025-12-25 20:44:55 -->


## Summary

**Purpose:** Transform vague ideas into actionable, well-scoped work through structured context gathering and collaborative discussion.

---

# Design Session Skill

**Purpose:** Transform vague ideas into actionable, well-scoped work through structured context gathering and collaborative discussion.

---

## When to Use

| Use design-session | Use architect | Use investigation |
|-------------------|---------------|-------------------|
| "I want to add X" (vague scope) | "Should we do X?" (strategic choice) | "How does X work?" (understand existing) |
| Feature ideation | Trade-off analysis | Root cause analysis |
| Scope definition | System shaping | Codebase exploration |

**The key distinction:** Design-session is for *scoping work*, architect is for *strategic decisions*, investigation is for *understanding*.

---

## Workflow Overview

```
Phase 1: Context Gathering (Autonomous)
    ↓
Phase 2: Design Synthesis (Semi-Autonomous)
    ↓
Phase 3: Output Creation (Autonomous)
    ↓
One of: Epic | Investigation | Decision
```

---

## Phase 1: Context Gathering (Autonomous)

**Goal:** Understand existing context before discussing scope.

### 1.1 Gather Knowledge Context

```bash
# Find relevant knowledge (constraints, decisions, investigations)
kb context "<topic keywords>"

# Example
kb context "rate limiting"
kb context "user authentication"
```

### 1.2 Gather Issue Context

```bash
# Find related beads issues
bd list --labels "<area>" 2>/dev/null | head -20
bd ready | grep -i "<keyword>" | head -10

# Find blocked work that might be related
bd blocked | grep -i "<keyword>" | head -10
```

### 1.3 Gather Codebase Context (If Applicable)

```bash
# Find relevant code areas
rg "<keyword>" --type-list  # See available types
rg "<pattern>" --type py -l | head -10

# Read key files
# Use Read tool for files identified above
```

### 1.4 Document Findings

Create a structured summary of what you found:

```markdown
## Context Gathered

### Existing Knowledge
- [kn entries found]
- [relevant investigations]
- [applicable decisions]

### Related Issues
- [existing issues on topic]
- [blocked items that relate]

### Codebase State
- [relevant files/modules]
- [existing implementations]
```

**Report:** `bd comment <beads-id> "Phase: Context Gathering - Found [N] related items"`

---

## Phase 2: Design Synthesis (Semi-Autonomous)

**Goal:** Present findings, refine scope through discussion, determine output type.

### 2.1 Present Context Summary

Present your findings naturally (not using AskUserQuestion yet):

```markdown
Here's what I found relevant to [topic]:

**Existing Knowledge:**
- [summary of constraints/decisions]

**Related Work:**
- [existing issues]
- [blocked items]

**Current State:**
- [what exists in codebase]

**Initial Questions:**
1. [clarifying question 1]
2. [clarifying question 2]
```

### 2.2 Refine Scope Through Discussion

**Question patterns for scoping:**

| Question Type | When to Ask | Example |
|--------------|-------------|---------|
| **Boundary** | Scope unclear | "Should this include X or is that separate?" |
| **Priority** | Multiple options | "Which of these is most important to start?" |
| **Constraint** | Trade-offs exist | "Are you optimizing for speed or completeness?" |
| **Dependency** | Order matters | "Does this need X first, or can they be parallel?" |

**Use natural conversation for discussion.** Reserve AskUserQuestion for:
- Forcing explicit choice between options
- When multiple rounds haven't converged

### 2.3 Determine Output Type

Based on discussion, assess clarity level:

| Clarity Level | Output Type | Indicators |
|--------------|-------------|------------|
| **High** | Epic with children | Clear scope, decomposable into tasks, no major unknowns |
| **Medium** | Investigation | Some unknowns remain, need exploration first |
| **Low** | Decision | Architectural choice blocks progress |

**Decision tree:**

```
Can we list the specific tasks needed?
├── YES → Do we understand all the tasks well enough to implement?
│   ├── YES → Epic with children
│   └── NO → Investigation (to clarify unknowns)
└── NO → Is this blocked by a strategic choice?
    ├── YES → Decision artifact
    └── NO → Investigation (to discover tasks)
```

**Report:** `bd comment <beads-id> "Phase: Design Synthesis - Determined output: [type]"`

---

## Phase 3: Output Creation (Autonomous)

Based on the determined output type, follow the appropriate path.

---

### Path A: Epic with Children

**When:** Scope is clear, work decomposes into discrete tasks.

#### A.1 Create the Epic

```bash
bd create "Epic: [high-level goal]" \
  --type epic \
  --description "## Goal

[What this epic achieves]

## Scope

- [In scope item 1]
- [In scope item 2]

## Out of Scope

- [Explicitly excluded]

## Success Criteria

- [ ] [Testable criterion 1]
- [ ] [Testable criterion 2]"
```

#### A.2 Create Child Issues

For each discrete task:

```bash
bd create "[Task title]" \
  --type task \
  --parent <epic-id> \
  --description "## Context

Part of [epic-id]: [epic title]

## Task

[What needs to be done]

## Acceptance Criteria

- [ ] [Criterion 1]
- [ ] [Criterion 2]"
```

#### A.3 Set Up Dependencies (If Needed)

```bash
# If task B depends on task A
bd dep add <task-b-id> --blocks <task-a-id>

# Common patterns:
# - Design → Implementation
# - Backend → Frontend
# - Core → Extensions
```

#### A.4 Apply Labels

```bash
# For all children
bd label <issue-id> triage:ready  # Ready for work
bd label <issue-id> area:auth     # Area label
```

**Reference: Beads Epic Patterns**

| Pattern | When to Use | Example |
|---------|-------------|---------|
| **Sequential** | Tasks must be done in order | Design → Implement → Test |
| **Parallel** | Tasks can be done independently | Multiple unrelated features |
| **Diamond** | Multiple paths converge | Backend + Frontend → Integration |

---

### Path B: Investigation

**When:** Unknowns remain that need exploration before planning.

#### B.1 Create Investigation

```bash
kb create investigation design/<slug>
```

#### B.2 Document What's Known and Unknown

Fill the investigation template:

```markdown
# Design Investigation: [Topic]

**Date:** [today]
**Status:** Active

## Question

What do we need to understand before we can plan [topic]?

## What We Know

- [From context gathering]
- [From discussion]

## What We Need to Learn

1. [Unknown 1]
2. [Unknown 2]

## Proposed Exploration

- [ ] [Investigation step 1]
- [ ] [Investigation step 2]

## Next Steps

After this investigation, expect to produce:
- [ ] Epic with children (if unknowns resolved)
- [ ] Decision artifact (if choice needed)
```

#### B.3 Create Follow-Up Issue

```bash
bd create "Investigate: [topic] design unknowns" \
  --type task \
  --description "Investigation artifact: .kb/investigations/[date]-design-[slug].md

Complete the investigation, then create follow-up work based on findings."
```

---

### Path C: Decision

**When:** Architectural choice blocks progress.

#### C.1 Create Decision Artifact

```bash
kb create decision <slug>
```

#### C.2 Document the Choice Needed

Fill the decision template:

```markdown
# [Decision Title]

**Date:** [today]
**Status:** Proposed

## Context

[Why this decision is needed now]

## Question

[The specific architectural question]

## Options

### Option A: [Name]

**Description:** [How it works]
**Pros:** [Benefits]
**Cons:** [Drawbacks]

### Option B: [Name]

**Description:** [How it works]
**Pros:** [Benefits]
**Cons:** [Drawbacks]

## Recommendation

[If you have one, state it with reasoning]

## Decision

[Leave blank - Dylan decides]

## Consequences

[What changes based on the decision]
```

#### C.3 Create Decision Review Issue

```bash
bd create "Decision needed: [topic]" \
  --type task \
  --labels "triage:review" \
  --description "Decision artifact: .kb/decisions/[date]-[slug].md

Review and make the architectural choice before proceeding with implementation."
```

---

## Beads Mastery Reference

### Creating Epics with Children

```bash
# Step 1: Create epic
bd create "Epic: User authentication" --type epic --description "..."
# Returns: orch-abc123

# Step 2: Create children with parent reference
bd create "Design auth flow" --type task --parent orch-abc123
bd create "Implement login" --type task --parent orch-abc123
bd create "Add tests" --type task --parent orch-abc123
```

### Setting Up Dependencies

```bash
# Task B is blocked by Task A (A must finish first)
bd dep add <task-b> --blocks <task-a>

# Check dependency tree
bd dep tree <epic-id>

# Find issues blocked by something
bd blocked
```

### Labels for Triage

| Label | Meaning | When to Apply |
|-------|---------|---------------|
| `triage:ready` | Ready for work | Clear scope, no blockers |
| `triage:review` | Needs human review | Uncertainty, needs input |
| `area:*` | Component area | auth, ui, api, etc. |
| `skill:*` | Recommended skill | feature-impl, investigation, etc. |

```bash
# Apply labels
bd label <issue-id> triage:ready
bd label <issue-id> area:auth skill:feature-impl
```

### Epic Status Checking

```bash
# See epic completion status
bd epic status <epic-id>

# Close eligible epics (all children done)
bd epic close-eligible
```

---

## Self-Review (Mandatory)

Before completing, verify:

### Phase Completion

| Phase | Check | If Failed |
|-------|-------|-----------|
| **Context Gathering** | Ran kb context and bd queries? | Run now |
| **Design Synthesis** | Discussed scope, determined output type? | Present findings |
| **Output Creation** | Produced appropriate artifact? | Complete output |

### Output Quality

#### For Epics:
- [ ] Epic has clear goal and scope
- [ ] Children are discrete, implementable tasks
- [ ] Dependencies set where needed
- [ ] Labels applied (triage:ready or triage:review)
- [ ] Each child has acceptance criteria

#### For Investigations:
- [ ] Question clearly stated
- [ ] Known/unknown clearly separated
- [ ] Next steps defined
- [ ] Follow-up issue created

#### For Decisions:
- [ ] Context explains why now
- [ ] Options clearly presented
- [ ] Pros/cons for each option
- [ ] Review issue created

---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kn decide` | `kn decide "Scope auth to session-based only" --reason "OAuth deferred to phase 2"` |
| Tried something that failed | `kn tried` | `kn tried "Single mega-epic" --failed "Too large, decomposed into 3 smaller epics"` |
| Discovered a constraint | `kn constrain` | `kn constrain "Must maintain backward compat" --reason "Existing API consumers"` |
| Found an open question | `kn question` | `kn question "Should notifications be real-time or polling?"` |

**Quick checklist:**
- [ ] Reflected on session: What did I learn that the next agent should know?
- [ ] Externalized at least one item via `kn` command

**If nothing to externalize:** Note in completion comment: "Leave it Better: Scope decisions captured in artifacts, no additional knowledge to externalize."

---

## Completion Criteria

Before marking complete:

- [ ] Phase 1: Context gathered (or skip-context used)
- [ ] Phase 2: Scope discussed, output type determined
- [ ] Phase 3: Appropriate artifact produced
- [ ] **Leave it Better completed** - At least one `kn` command run OR noted as not applicable
- [ ] All changes committed to git
- [ ] Report via beads: `bd comment <beads-id> "Phase: Complete - Created [output type]: [summary]"`
- [ ] Close beads issue: `bd close <beads-id> --reason "[summary]"`
- [ ] Call `/exit` to close agent session

---

## Related Skills

- **architect** - Use for strategic decisions with trade-off analysis
- **investigation** - Use when "how does X work?" (understand, not scope)
- **issue-creation** - Use for single issues from symptoms
- **feature-impl** - Use after design-session produces actionable epic/tasks

---

## Common Patterns

### Pattern: Vague Feature Request

```
User: "We need better error handling"

Phase 1: Find existing error handling code, related issues
Phase 2: "Better" could mean many things - discuss scope
Phase 3: Likely → Epic with children (log errors, user messages, retry logic)
```

### Pattern: Technical Debt

```
User: "The auth system is a mess"

Phase 1: Audit auth code, find pain points
Phase 2: Prioritize which issues to address first
Phase 3: Could be → Investigation (if root cause unclear)
              → Epic (if clear what to fix)
```

### Pattern: New Feature Idea

```
User: "Can we add real-time notifications?"

Phase 1: Check existing notification code, related decisions
Phase 2: Scope: push vs pull, what triggers, where shown
Phase 3: Likely → Decision (architecture choice)
              → then Epic (implementation plan)
```






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.

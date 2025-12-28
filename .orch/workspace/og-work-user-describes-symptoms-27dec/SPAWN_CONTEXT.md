TASK: User describes symptoms using technical terms they're not sure about (e.g., 'hydration is slow'). Orchestrator takes the technical framing literally and spawns work on wrong problem. Real issue was backend latency, not hydration. How should we handle the gap between user's symptom description and their technical framing? This touches: communication protocol between human and orchestrator, when to validate premises vs trust framing, Evidence Hierarchy (user observation vs user hypothesis). Context: Premise Before Solution principle exists but wasn't applied.

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "user"

### Constraints (MUST respect)
- Concurrent agents trigger TPM throttling at >60% session usage
  - Reason: Observed performance degradation and user reports of hitting limits during swarm operations

### Prior Decisions
- Headless Swarm = batch execution + rate-limit management across accounts
  - Reason: User clarified scope: focus on concurrent agent spawning with capacity management, not distributed architecture or multi-model routing
- orch status shows PHASE and TASK columns from beads data
  - Reason: Makes output actionable - users can immediately see what each agent is doing
- Headless spawn mode is production-ready
  - Reason: All 5 requirements verified working: status detection, monitoring, completion detection, error handling, user visibility. Investigation orch-go-0r2q confirmed no blockers exist.
- Document all spawn modes in CLAUDE.md
  - Reason: Users need to see all options (headless default, --tmux opt-in, --inline blocking) to make informed spawn choices
- Help text should be organized by mode hierarchy (default → opt-in → blocking)
  - Reason: Users need to see default behavior first; mode grouping in examples improves discoverability and understanding
- Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management
  - Reason: Balances operational visibility (active work always visible) with historical debugging (expand sections as needed) and UI clarity (collapsed sections reduce clutter). Only approach that satisfies all three user contexts: development focus, debugging history, and health monitoring.
- Binary staleness should be prevented with make install
  - Reason: Project tracks ./orch binary in git, but build process creates build/orch. Users should run 'make install' to sync ~/bin/orch rather than manually copying to root directory.
- D.E.K.N. validation gates file output only
  - Reason: Allows draft preview workflow - users see prompts with stdout, then fill in content before saving with -o
- Agent card titles use TLDR for completed, task for active
  - Reason: Human-readable titles convey gist at a glance without requiring users to parse workspace naming conventions
- Show full TLDR on completed agent cards by removing line-clamp-2
  - Reason: Users want to see the complete summary at a glance instead of having to click for details
- Show collapsed section preview
  - Reason: Users can see agent tasks at a glance without expanding sections
- Global user config in pkg/userconfig reads ~/.orch/config.yaml
  - Reason: Needed unified place for user preferences like notifications. Follows same pattern as pkg/account for ~/.orch/accounts.yaml.
- Single-Agent Review Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-21-single-agent-review-command.md
- Template Ownership Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-22-template-ownership-model.md
- Orchestrator System Resource Visibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md

### Related Investigations
- Update README with current CLI commands and usage
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-update-readme-current-cli-commands.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- Add Missing Spawn Flags
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-missing-spawn-flags-no.md
- Add orch review command for batch completion workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-orch-review-command-batch.md
- KB Search vs Grep Benchmark
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md
- Compare orch-cli (Python) vs orch-go Features
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md
- Finalize Native Implementation for orch send
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-finalize-native-implementation-orch-send.md
- Implement Account Switch with OAuth Token Refresh
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-account-switch-oauth-token.md
- Port model flexibility and arbitrage to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md
- Migrate orch-go from tmux to HTTP API
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-migrate-orch-go-tmux-http.md
- Agent Registry for Persistent Tracking
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-agent-registry-persistent.md
- orch-go Add Question Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-question-command.md
- Recursive Research Loop Persona Confusion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-recursive-research-loop-persona-confusion.md
- Research: Gemini 2.0 Models (Flash, Pro, Experimental)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-research-gemini-2-0-models.md
- Scaffold beads-ui v2 (Bun + SvelteKit 5 + shadcn-svelte)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scaffold-beads-ui-v2-bun.md
- Scope Out Headless Swarm Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md
- tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-tmux-concurrent-zeta.md
- Update All Worker Skills with 'Leave it Better' Phase
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-update-all-worker-skills-include.md
- Update Investigation Skill to Use D.E.K.N. Summary
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-update-investigation-skill-use-summary.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-5pgu "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-5pgu "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-5pgu "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation user-describes-symptoms-technical-terms` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-user-describes-symptoms-technical-terms.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-5pgu "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-user-describes-symptoms-27dec/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-5pgu**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-5pgu "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-5pgu "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-5pgu "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-5pgu "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-5pgu "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-5pgu`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



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
<!-- Checksum: 988443a1c470 -->
<!-- Source: worker/design-session/.skillc -->
<!-- To modify: edit files in worker/design-session/.skillc, then run: skillc build -->
<!-- Last compiled: 2025-12-24 07:51:19 -->


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
2. `bd comment orch-go-5pgu "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

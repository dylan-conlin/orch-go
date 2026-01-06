TASK: Design-principles skill integration: 1) How should the skill be triggered/loaded (manual flag, project config, task detection)? 2) Standalone skill or merge into feature-impl UI phases? 3) Context window cost tradeoff (238 lines ~1-2K tokens)? 4) Should Dylan's personal design preferences override/extend upstream? 5) How does this interact with ui-mockup-generation? Context: We've evaluated https://github.com/Dammyjay93/claude-design-skill (cloned to /Users/dylanconlin/Documents/personal/claude-design-skill) - high quality 238-line policy skill with 4px grid, typography scales, 6 design personalities, anti-patterns. Fills gap in orch-ecosystem. See investigation: .kb/investigations/2026-01-05-design-claude-design-skill-evaluation.md

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



⚠️ Limited context (15/100) - agent may need to discover patterns during work

## PRIOR KNOWLEDGE (from kb context)

**Query:** "design principles skill"

### Related Investigations
- Claude Design Skill Evaluation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-05-design-claude-design-skill-evaluation.md

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
2. **SET UP investigation file:** Run `kb create investigation design-principles-skill-integration-skill` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-design-principles-skill-integration-skill.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-design-principles-skill-05jan-3932/SYNTHESIS.md
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
<!-- Checksum: 8daf42f85df1 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/design-session/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/design-session/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/design-session/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-05 12:52:51 -->

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

### 1.0 Review Foundational Principles

**Before scoping work, review:** `.kb/principles.md`

Key principles for design sessions:
- **Session amnesia** - Will this scoping help the next Claude resume?
- **Evolve by distinction** - When scope is unclear, ask "what are we conflating?"
- **Evidence hierarchy** - Code is truth; artifacts are hypotheses to verify

Consider which principles apply when making scoping decisions.

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

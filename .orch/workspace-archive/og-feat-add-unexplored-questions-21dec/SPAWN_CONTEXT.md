TASK: Add Unexplored Questions section to SYNTHESIS.md template. Add section before Session Metadata with three prompts: questions that emerged, areas worth exploring, things that remain unclear. Update orch review to display unexplored questions section. See design spec .kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md lines 449-469 for template text.

🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-ivtg.1 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-ivtg.1 "Phase: Complete - [1-2 sentence summary of deliverables]"`
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

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation add-unexplored-questions-section-synthesis` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-add-unexplored-questions-section-synthesis.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-ivtg.1 "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]
6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-add-unexplored-questions-21dec/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-ivtg.1**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-ivtg.1 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-ivtg.1 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-ivtg.1 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-ivtg.1 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-ivtg.1 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-ivtg.1`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## SKILL GUIDANCE (feature-impl)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: feature-impl
skill-type: procedure
audience: worker
spawnable: true
category: implementation
description: Unified feature implementation with configurable phases (investigation, clarifying-questions, design, implementation, validation, integration). Replaces test-driven-development, surgical-change, and feature-coordination skills. Use for any feature work with phases/mode/validation configured by orchestrator.

deliverables:
  investigation:
    required: false
    description: "Investigation file (.kb/investigations/YYYY-MM-DD-inv-{topic}.md). Required when investigation phase included in configuration."

  design:
    required: false
    description: "Design document (.kb/decisions/ or docs/plans/). Required when design phase included in configuration."

  tests:
    required: false
    description: "Test files (location varies by project). Required when mode=tdd or validation=tests."

  implementation:
    required: false
    description: "Source code changes. Required when implementation phase included in configuration."

  validation-evidence:
    required: false
    description: "Validation evidence (smoke test results, test output, screenshots). Required when validation != none. Report via bd comment."

verification:
  requirements:
    - "All configured phases completed (investigation findings, design docs, implementation, validation evidence as applicable)"
    - "Tests pass OR validation evidence documented via bd comment (automated tests, smoke test, or multi-phase validation)"
    - "Implementation matches design (if design phase used)"
    - "No regressions introduced (existing functionality still works)"
    - "All deliverables committed and reported via bd comment"
---

<!-- AUTO-GENERATED: Do not edit this file directly. Source: src/SKILL.md.template + src/phases/*.md. Build with: orch build --skills -->

> AUTO-GENERATED SKILL FILE
> Source: src/SKILL.md.template + src/phases/*.md
> Build command: orch build --skills
> Do NOT edit this file directly; edit the sources and rebuild.

# Feature Implementation (Unified Framework)

**For orchestrators:** Spawn via `orch spawn feature-impl "task" --phases "..." --mode ... --validation ...`

**For workers:** You've been spawned to implement a feature using a phased approach with specific configuration.

---

## Your Configuration

**Read from SPAWN_CONTEXT.md** to understand your configuration:

- **Phases:** Which phases you'll proceed through (e.g., `investigation,clarifying-questions,design,implementation,validation`)
- **Current Phase:** Determined by your progress (start with first configured phase)
- **Implementation Mode:** `tdd` or `direct` (only relevant if implementation phase included)
- **Validation Level:** `none`, `tests`, `smoke-test`, or `multi-phase` (only relevant if validation phase included)

**Example configuration:**
```
Phases: design, implementation, validation
Mode: tdd
Validation: smoke-test
```

**This means you will:**
1. Start with Design phase → create design document
2. Move to Implementation phase (TDD mode) → write tests first, then code
3. Finish with Validation phase (smoke-test level) → run tests + manual verification

---
## When to Use Investigation Phase

Investigation phase is OPTIONAL. Include it when:
- ✅ Unfamiliar codebase/subsystem
- ✅ Complex integration points unknown
- ✅ Prior investigation >3 months old

Skip investigation phase when:
- ❌ Architecture well understood
- ❌ Prior investigation exists and current
- ❌ Simple, isolated change

**Alternative:** For significant unknowns, spawn separate investigation agent first,
then spawn feature-impl with `--phases implementation,validation` and reference
to investigation findings.

---


## Deliverables

**Required based on configuration:**

| Configuration | Required |
|---------------|----------|
| investigation phase | Investigation file |
| design phase | Design document |
| implementation phase | Source code |
| mode=tdd | Tests (write first) |
| validation=tests | Tests |
| validation=smoke-test | Validation evidence (reported via bd comment) |
| validation=multi-phase | Phase checkpoints via bd comment |

**See phase guidance for details.**

---

## Workflow

Proceed through phases sequentially per your configuration.

**Phases:** Investigation → Clarifying Questions → Design → Implementation (TDD/direct) → Validation → Self-Review → Integration

**Follow current phase guidance below** (track progress via `bd comment <beads-id> "Phase: X - details"`).

---

## Step 0: Scope Enumeration (REQUIRED)

**Purpose:** Prevent "Section Blindness" - implementing only part of spawn context while missing other requirements.

**Before starting ANY phase work, you MUST:**

### 1. Read ENTIRE SPAWN_CONTEXT

Read your full SPAWN_CONTEXT.md from start to end. Don't skim. Don't stop at the first section that looks like work.

### 2. Enumerate ALL Requirements

List every discrete requirement/deliverable from the spawn context. Include:
- Explicit deliverables (files, code, tests, documentation)
- Implicit requirements (validation, testing, cleanup)
- Items from ALL sections (not just `## Implementation` or similar)

### 3. Report Scope via Beads

```bash
bd comment <beads-id> "Scope: 1. [requirement] 2. [requirement] 3. [requirement] ..."
```

**Example:**
```bash
bd comment ok-abc "Scope: 1. Add trace_path to scrape_job.metadata 2. Create Rails endpoint with CORS 3. Add UI link 4. Implement retention strategy (keep failures, first success per combo, random 5%) 5. Create cleanup rake task"
```

### 4. Flag Uncertainty

If any of these are true, add to your comment:
- Spawn context has multiple sections that look like separate work
- Requirements seem incomplete or unclear
- You're unsure if something is in scope

```bash
bd comment <beads-id> "Scope: 1. X 2. Y 3. Z -- QUESTION: Is the retention strategy section also in scope?"
```

**Wait for orchestrator confirmation if you flag uncertainty.**

---

### Why This Matters

Agents have repeatedly:
- Implemented `## Implementation` section while ignoring `## Retention Strategy`
- Claimed "Phase: Complete" with partial work
- Never flagged they were doing partial scope

Forcing explicit enumeration catches this BEFORE implementation begins.

---

### Completion Criteria (Step 0)

- [ ] Read full SPAWN_CONTEXT (all sections)
- [ ] Enumerated ALL requirements (not just one section)
- [ ] Reported scope via `bd comment`
- [ ] Flagged any uncertainty (if applicable)

**Once Step 0 complete → Proceed to your first configured phase.**

---

## Phase-Specific Guidance

**Below are the detailed instructions for each phase.** Read the section that matches your current phase.

---

<!-- SKILL-TEMPLATE: investigation -->
<!-- Auto-generated from src/phases/investigation.md -->

# Investigation Phase

**Purpose:** Understand the existing system before making changes.

**When you're in this phase:** Your SPAWN_CONTEXT specified investigation scope. Document findings progressively to inform design and implementation decisions.

---

## Deliverables

- **Investigation file:** `.kb/investigations/YYYY-MM-DD-inv-{kebab-case-description}.md`
- **Findings:** Evidence-Source-Significance pattern
- **Synthesis:** Connected insights, not just a list of facts

---

## Workflow

### 1. Create Investigation Template (Before Exploring)

**Critical:** Create template at START, not at end. Forces progressive documentation.

```bash
DATE=$(date +%Y-%m-%d)
SLUG="topic-in-kebab-case"  # From SPAWN_CONTEXT description
INVESTIGATION_FILE=${PROJECT_DIR}/.kb/investigations/${DATE}-inv-${SLUG}.md

mkdir -p ${PROJECT_DIR}/.kb/investigations
# Use investigation skill template or create from structure below
```

Template structure:
```markdown
# Investigation: [Specific Topic]

**Question:** [Precise question from SPAWN_CONTEXT]
**Started:** YYYY-MM-DD
**Updated:** YYYY-MM-DD
**Status:** In Progress
**Confidence:** Low

## Findings

[Add progressively as you explore]

## Synthesis

**Key Insights:**
- [Connect findings into patterns]

**Answer to Question:**
[Coherent answer based on findings]

## Confidence Assessment

**Current Confidence:** Low/Medium/High

**What's certain:**
- [Verified facts with evidence]

**What's uncertain:**
- [Known gaps - be honest]
```

### 2. Fill Question and Metadata

Edit investigation file with precise question from SPAWN_CONTEXT:
- **Question:** Specific, answerable question
- **Started:** Today's date
- **Status:** In Progress
- **Confidence:** Low (initial state)

### 3. Add Findings Progressively (As You Explore)

**After each discovery**, add a finding using this pattern:

```markdown
### Finding 1: [Brief description]

**Evidence:** [Concrete observation - code snippet, output, behavior]

**Source:** [File:line reference or command that produced evidence]

**Significance:** [Why this matters for implementation]
```

**Example:**
```markdown
### Finding 1: Authentication uses JWT tokens in HTTP-only cookies

**Evidence:** Found `Set-Cookie` header with `httpOnly=true` flag in login response. Token has 3-part structure (header.payload.signature).

**Source:** `src/auth/middleware.ts:45-67` and Chrome DevTools Network tab

**Significance:** Token can't be accessed by JavaScript (XSS protection), but sent automatically with requests (CSRF risk). Must implement CSRF protection for state-changing operations.
```

**Don't wait to write everything at end** - document as you go.

### 4. Update Synthesis After Each Cluster

**Every 3-5 findings**, update synthesis section to connect patterns:

```markdown
## Synthesis

**Key Insights:**
- Auth uses JWT tokens in HTTP-only cookies (Finding 1)
- Tokens expire after 15 min, refresh tokens last 7 days (Finding 2)
- Refresh endpoint at `/auth/refresh` extends session (Finding 3)

**Answer to Question:**
[Coherent explanation connecting all findings]
```

Progressive synthesis helps spot patterns as they emerge.

### 5. Update Confidence Assessment

As investigation progresses, update confidence honestly:

```markdown
## Confidence Assessment

**Current Confidence:** Medium

**Why this level?**
Explored main flow thoroughly, but haven't examined edge cases.

**What's certain:**
- [Verified facts]

**What's uncertain:**
- [Known gaps]

**To increase confidence:**
- [Actions to close gaps]
```

**Honest confidence > false certainty.** State what you don't know.

### 6. Mark Complete and Move to Clarifying Questions Phase

When investigation answers your question:

1. Update status and confidence:
   ```markdown
   **Status:** Complete
   **Confidence:** High
   ```

2. Commit investigation file:
   ```bash
   cd ${PROJECT_DIR}
   git add .kb/investigations/${DATE}-inv-${SLUG}.md
   git commit -m "investigation: ${SLUG}"
   ```

3. Report phase transition:
   ```bash
   bd comment <beads-id> "Phase: Clarifying Questions - Investigation complete, findings in [investigation file path]"
   ```

4. Output: "✅ Investigation complete, moving to Clarifying Questions phase"

---

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Progressive documentation** | Create template first, fill findings as you explore (not at end) |
| **Evidence-based** | Every finding needs concrete evidence (code, output, observation) |
| **Honest confidence** | State what you don't know - gaps are valuable information |
| **Clear sourcing** | Always include file:line or command that produced evidence |
| **Synthesis over list** | Connect findings into coherent answer, don't just list facts |

---

## Completion Criteria

Before moving to Design phase, verify:

- [ ] Investigation file created in `.kb/investigations/`
- [ ] Question answered with synthesis (not just list of findings)
- [ ] Each finding has Evidence + Source + Significance
- [ ] Confidence assessed honestly (gaps acknowledged)
- [ ] Key architectural constraints documented
- [ ] Dependencies identified for implementation
- [ ] Investigation file committed to git
- [ ] Workspace updated: Phase → Clarifying Questions

**If ANY box unchecked, investigation is NOT complete.**

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: clarifying-questions -->
<!-- Auto-generated from src/phases/clarifying-questions.md -->

# Clarifying Questions Phase

**Purpose:** Surface all ambiguities BEFORE design work begins.

**When you're in this phase:** Investigation (if any) is complete. Before starting design, explicitly identify and ask about any unclear requirements, edge cases, or integration concerns.

**Why this exists:** Asking questions during design is often too late - design decisions may already be influenced by assumptions. This phase creates a hard stop to surface ambiguities before investing effort in design.

---

## Deliverables

- **Questions documented:** All clarifying questions communicated via `bd comment`
- **Answers received:** Orchestrator or user has answered all blocking questions
- **No ambiguities:** Ready to proceed with design with clear understanding

---

## Workflow

### 1. Review What You Know

Before identifying gaps, summarize your current understanding:

**If investigation phase preceded this:**
- Read investigation file findings
- Note key architectural constraints discovered
- Identify integration points found
- List dependencies identified

**If no investigation phase:**
- Review SPAWN_CONTEXT requirements
- Note explicit constraints provided
- Identify stated scope boundaries

### 2. Identify Question Categories

Systematically consider each category for potential ambiguities:

| Category | Questions to Consider |
|----------|----------------------|
| **Edge Cases** | Empty inputs? Maximum limits? Concurrent access? Null/undefined handling? |
| **Error Handling** | What should happen when X fails? Retry behavior? User-facing error messages? |
| **Integration Points** | How does this connect to existing systems? API contracts? Data flow? |
| **Backward Compatibility** | Will this break existing functionality? Migration needed? Deprecation strategy? |
| **Performance** | Expected load? Response time requirements? Resource constraints? |
| **Security** | Authentication requirements? Authorization rules? Data sensitivity? |
| **Scope Boundaries** | What's explicitly out of scope? Deferred to future work? |

### 3. Document Questions

**Report questions via beads comment:**

```bash
bd comment <beads-id> "QUESTION: [question with context and default assumption]"
```

**Example:**
```bash
bd comment <beads-id> "QUESTION: Edge case - What should happen with empty input? Default assumption: return empty result"
bd comment <beads-id> "QUESTION: Integration - Should auth middleware apply to this endpoint? Default assumption: yes, standard auth"
```

**Include default assumptions** - this allows orchestrator to quickly confirm or correct rather than answering from scratch.

### 4. Ask Questions Using Directive-Guidance Pattern

**CRITICAL: Use directive-guidance, not quiz-style questions.**

Clarifying questions are about **confirming intent**, not testing knowledge. There is no "wrong" answer - the user's response defines the requirement.

**Pattern reference:** `~/.orch/patterns/directive-guidance.md`

**❌ DON'T present neutral options (quiz-style):**
```
"How should we handle flag conflicts?"
  1. Option A
  2. Option B
  3. Option C
```
This feels like a quiz where the user might give a "wrong" answer.

**✅ DO state your recommendation with reasoning:**
```
"I'm planning to error if both --json and --format are specified, since
explicit errors are clearer than magic precedence rules. Does that match
what you want, or would you prefer different behavior?"
```
This confirms intent - user can agree or redirect.

**When using AskUserQuestion tool:**
- **question:** State your planned approach, ask if it matches their intent
- **options:** Your recommendation (marked ⭐) + alternatives with tradeoffs
- **description:** Why you're recommending this approach

**Example AskUserQuestion usage:**
```
question: "I'm planning to return a 429 error for rate limit violations.
          Does that work, or would you prefer different behavior?"
options:
  - "⭐ 429 error (recommended)" - Clear feedback, standard HTTP semantics
  - "Queue requests" - Better UX but adds complexity
  - "Drop silently" - Simple but user gets no feedback
```

**For complex or open-ended questions**, report via `bd comment <beads-id> "AWAITING_ANSWERS: [details]"`.

**Do NOT proceed to design until questions are answered.**

### 5. Record Answers

When orchestrator responds, acknowledge via beads:

```bash
bd comment <beads-id> "Answers received: [summary]. Impact on design: [brief notes]"
```

### 6. Move to Design Phase

Once all questions resolved:

1. Report phase transition: `bd comment <beads-id> "Phase: Design - Questions resolved, proceeding with design"`

2. Output: "✅ Clarifying questions resolved, moving to Design phase"

---

## When Questions Are Not Needed

**Skip this phase (or complete quickly) when:**
- SPAWN_CONTEXT is highly detailed and explicit
- Following well-established patterns with no ambiguity
- Orchestrator pre-answered likely questions in spawn prompt
- Investigation phase already surfaced and resolved ambiguities

**Even then, quickly verify:** "Are there any edge cases, error handling, or integration concerns I should ask about?"

If genuinely nothing unclear → Document "No clarifying questions - requirements are clear" and proceed.

---

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Ask before design** | Questions during design means rework; questions before design saves time |
| **Confirm intent, don't quiz** | State your recommendation, ask if it matches intent - there's no "wrong" answer |
| **Default assumptions** | Always state what you'll assume - enables quick confirmation vs open-ended questions |
| **Structured categories** | Systematic review prevents missing important ambiguities |
| **Block on answers** | Don't proceed to design with unresolved ambiguities |
| **Document impact** | When answer received, note how it affects design approach |

---

## Completion Criteria

Before moving to Design phase, verify:

- [ ] All question categories reviewed (edge cases, errors, integration, compatibility, etc.)
- [ ] Questions communicated via `bd comment`
- [ ] Orchestrator answered all blocking questions
- [ ] Answers acknowledged via `bd comment`
- [ ] No remaining ambiguities that would affect design
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Design - Questions resolved"`

**If ANY box unchecked, clarifying questions phase is NOT complete.**

**Exception:** If genuinely no questions exist, report via `bd comment <beads-id> "Phase: Design - No clarifying questions needed"` and proceed.

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: design -->
<!-- Auto-generated from src/phases/design.md -->

# Design Phase

**Purpose:** Document architectural approach before implementation.

**When you're in this phase:** Investigation findings (if any) inform the design. Document approach, architecture, and testing strategy before writing code.

---

## Deliverables

- **Design document:** `docs/designs/YYYY-MM-DD-{kebab-case-feature}.md`
- **Testing strategy:** Clear plan for what needs tests
- **Architecture decision:** Chosen approach with trade-off analysis

---

## Workflow

### 1. Review Investigation Findings (If Investigation Phase Included)

If investigation phase preceded design:
- Read investigation file: `.kb/investigations/YYYY-MM-DD-*.md`
- Note key architectural constraints
- Identify integration points
- Understand dependencies

### 2. Determine if Design Exploration Needed

**Escalate to orchestrator for design exploration when:**
- ✅ Multiple viable technical approaches exist (e.g., library selection, architecture patterns)
- ✅ Significant trade-offs to evaluate (performance vs maintainability, complexity vs flexibility)
- ✅ Uncertainty about best approach based on investigation findings
- ✅ Novel problem domain without established patterns

**Proceed with design directly when:**
- ❌ Approach is obvious from investigation
- ❌ Following established patterns in codebase
- ❌ Simple/straightforward implementations
- ❌ Orchestrator already specified approach in SPAWN_CONTEXT

**If design exploration needed:**

Report via beads that you need design exploration:
```bash
bd comment <beads-id> "Status: BLOCKED - Multiple viable approaches, need design exploration before proceeding"
```

The orchestrator may spawn an interactive architect session (`orch spawn architect -i`) for collaborative design exploration. Wait for orchestrator response before proceeding.

### 3. Create Design Document

Create design file:
```bash
DATE=$(date +%Y-%m-%d)
SLUG="feature-name-in-kebab-case"
DESIGN_FILE=${PROJECT_DIR}/docs/designs/${DATE}-${SLUG}.md

mkdir -p ${PROJECT_DIR}/docs/designs
```

**Use the design document template:**

Full template available at: `~/.claude/skills/worker/feature-impl/reference/design-template.md`

**Key sections to include:**
- Problem statement with success criteria
- Approach and architectural decisions
- Data model (if applicable)
- UI/UX (if applicable)
- Testing strategy
- Security considerations
- Performance requirements
- Rollout plan
- Alternatives considered
- Open questions and references

### 4. Present Design for Orchestrator Review

**Report design summary via beads:**

```bash
bd comment <beads-id> "Design ready for review: docs/designs/YYYY-MM-DD-{slug}.md - [chosen approach]. Key decisions: [1-2 sentences]. Awaiting approval."
```

If design exploration was done (via architect session):
```bash
bd comment <beads-id> "Design ready: Evaluated 3 approaches, recommending [A] because [reasoning]. See docs/designs/... for details."
```

### 5. Get Orchestrator Approval

**Wait for orchestrator to:**
- Review design document
- Ask clarifying questions
- Approve approach OR suggest adjustments

**Do not proceed to implementation without approval.**

### 6. Move to Implementation

Once approved:

1. Update design doc status:
   ```markdown
   **Status:** Approved
   ```

2. Commit design document:
   ```bash
   cd ${PROJECT_DIR}
   git add docs/designs/${DATE}-${SLUG}.md
   git commit -m "design: ${SLUG}"
   ```

3. Report phase transition: `bd comment <beads-id> "Phase: Implementation - Design approved, beginning implementation"`

4. Output: "✅ Design approved, moving to Implementation phase"

---

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Design before code** | Architecture decisions documented upfront, not discovered during implementation |
| **Trade-offs explicit** | Document why chosen approach, what alternatives were rejected |
| **Testing strategy clear** | Know what needs tests before writing code |
| **Security by design** | Security considerations integrated, not bolted on |
| **Review before implementation** | Get approval on approach before investing time in code |

---

## Completion Criteria

Before moving to Implementation phase, verify:

- [ ] Design document created in `docs/designs/`
- [ ] Problem statement clear
- [ ] Approach documented with rationale
- [ ] Data model defined (if applicable)
- [ ] Testing strategy specified
- [ ] Security considerations addressed
- [ ] Performance requirements documented
- [ ] Alternatives considered (if design exploration was done)
- [ ] Orchestrator reviewed and approved design
- [ ] Design document committed to git
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Implementation - Design approved"`

**If ANY box unchecked, design is NOT complete.**

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: implementation-tdd -->
<!-- Auto-generated from src/phases/implementation-tdd.md -->

# Implementation Phase (TDD Mode)

**Purpose:** Implement feature using test-driven development.

**When to use:** Feature adds/changes behavior (APIs, business logic, UI interactions).

**Core principle:** If you didn't watch the test fail, you don't know if it tests the right thing.

---

## Pre-Implementation Exploration (REQUIRED)

**Before writing ANY code, you MUST explore the codebase.** This prevents incomplete implementations and missed integration points.

### Step 1: Explore with Task Tool

Use the Task tool with `subagent_type="Explore"` to understand the code you'll be changing:

```
Task(
  subagent_type="Explore",
  prompt="Find all files related to [feature area]. Identify:
    1. Files I'll need to modify
    2. Functions/classes that call or are called by this code
    3. Existing tests covering this functionality
    4. Edge cases visible in current implementation",
  description="Pre-impl exploration"
)
```

**What to explore:**
- Files you plan to modify (read them fully)
- Callers/importers of functions you'll change
- Related tests (understand what's already covered)
- Similar patterns in the codebase (how is this done elsewhere?)

### Step 2: Report Findings

After exploration, report via beads:

```bash
bd comment <beads-id> "Pre-impl exploration complete: [N] files to modify, [M] callers identified, [K] existing tests. Key integration points: [list]. Edge cases to handle: [list]"
```

### Step 3: Verify Readiness

Before proceeding, confirm:
- [ ] Read all files I'll modify
- [ ] Identified callers/dependencies
- [ ] Found existing tests
- [ ] Know the edge cases

**If exploration reveals complexity beyond task scope:** STOP and escalate to orchestrator.

**Why this matters:** Most bugs are integration issues and missed edge cases. Exploring BEFORE coding catches these when they're cheap to fix. Skipping exploration leads to incomplete implementations and rework.

---

## The Iron Law

```
NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST
```

Write code before the test? **Delete and start over**.

---

## TDD Cycle (Repeat for Each Unit of Behavior)

### 1. Write Failing Test (RED)
- Write one minimal test showing desired behavior
- Run test, verify it fails for correct reason
- Commit: `git commit -m "test: add failing test for [behavior]"`

### 2. Write Minimal Code (GREEN)
- Write simplest code to make test pass (no over-engineering)
- Verify all tests pass (no regressions)
- Commit: `git commit -m "feat: implement [behavior]"`

### 3. Refactor (REFACTOR)
- Clean up while staying green (remove duplication, improve names)
- Run tests again - verify still passing
- Commit (if refactored): `git commit -m "refactor: [what improved]"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/tdd-best-practices.md` for:
- Detailed TDD cycle steps with code examples
- Best practices (test-first, red-green-refactor, small cycles)
- Good test qualities
- Commit format examples

---

## UI Feature Requirements

**Critical:** Tests passing ≠ feature working (especially for UI)

**Mandatory smoke test for UI features:**
1. Load actual feature (browser, curl, or CLI)
2. Verify visually (rendering, styling, data, interactions, no errors)
3. Capture evidence (screenshot or output)
4. Report via beads: `bd comment <beads-id> "Smoke test passed - [verification summary]"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/tdd-best-practices.md` for:
- Why smoke tests matter (what tests don't verify)
- Real example of "tests pass" but feature broken
- Smoke test documentation template

---

## Red Flags - STOP and Start Over

**If doing any of these, delete code and restart:**
- Writing code before test
- Test passes immediately (didn't see failure)
- Rationalizing "just this once" or "tests later"
- "TDD is dogmatic, I'm being pragmatic"

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/tdd-best-practices.md` for complete red flags list.

---

## Completion Criteria

Before moving to Validation phase, verify:

- [ ] Every function/method has test that failed first
- [ ] All tests pass (green)
- [ ] UI smoke test complete (if UI feature)
- [ ] Test/impl commits separate
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Validation - Implementation complete, tests passing"`

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/tdd-best-practices.md` for complete checklist.

---

## When to Move to Validation Phase

Once completion criteria met → Report via `bd comment <beads-id> "Phase: Validation"` → Proceed to Validation phase

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: implementation-direct -->
<!-- Auto-generated from src/phases/implementation-direct.md -->

# Implementation Phase (Direct Mode)

**Purpose:** Implement non-behavioral changes directly without TDD overhead.

**When to use:** Refactoring, configuration, documentation, code cleanup, renaming, removing dead code

⚠️ **Critical:** If changing behavior (logic, features, bugs) → STOP and switch to TDD mode.

---

## Pre-Implementation Exploration (REQUIRED)

**Before making changes, you MUST explore the codebase.** This prevents accidental behavioral changes and regressions.

### Step 1: Explore with Task Tool

Use the Task tool with `subagent_type="Explore"` to understand impact:

```
Task(
  subagent_type="Explore",
  prompt="Find all usages of [code to modify]. Identify:
    1. All callers/importers of this code
    2. Tests that cover this functionality
    3. Whether this change affects behavior (logic, output, side effects)",
  description="Pre-impl impact check"
)
```

**What to explore:**
- All callers/dependents of code you'll change
- Existing tests (to verify no regressions)
- Whether change is truly non-behavioral

### Step 2: Confirm Non-Behavioral

After exploration, verify this is truly non-behavioral:
- ✅ Rename, extract helper (same behavior), config, docs, formatting, remove dead code
- ❌ Bug fix, new feature, logic change, error handling → **STOP, switch to TDD mode**

### Step 3: Report Findings

```bash
bd comment <beads-id> "Pre-impl exploration complete: [N] files to modify, [M] callers found. Confirmed non-behavioral: [reasoning]. Tests to verify: [list]"
```

**If exploration reveals behavioral impact:** STOP. Switch to TDD mode.

**If unsure about impact:** STOP and escalate. Ask orchestrator before proceeding.

---

## Workflow

### 1. Validate Scope

**Confirm non-behavioral:**
- ✅ Rename, extract helper (same behavior), config, docs, formatting, remove dead code
- ❌ Bug fix, new feature, logic change, error handling → Use TDD mode instead

**If unsure → use TDD mode (safer).**

### 2. Prepare Environment

1. Pull latest (`git pull origin main`)
2. Run existing tests (establish baseline)
3. Verify all tests pass before making changes

### 3. Make Changes

- Keep diffs focused (avoid opportunistic refactors)
- One change per commit (two max if tightly related)
- If scope expands beyond 2 files or 1 hour → pause and escalate

### 4. Verify No Regressions

1. Run tests again
2. Verify all tests still pass
3. Sanity check impacted area

### 5. Commit

```bash
git add [files]
git commit -m "[type]: [description]"
```

**Types:** `refactor` (restructuring), `chore` (config/tools), `docs` (documentation), `style` (formatting)

### 6. Move to Validation

Report via `bd comment <beads-id> "Phase: Validation - Direct mode changes complete, tests passing"`

---

## Guardrails

**Stop and escalate if:**
- Scope expands beyond 2 files or 1 hour
- Behavior changes detected → Switch to TDD mode
- Tests start failing unexpectedly
- Unclear if change is behavioral → Use TDD mode

---

## Completion Criteria

- [ ] Changes truly non-behavioral
- [ ] Existing tests still pass
- [ ] Scope ≤ 2 files and ≤ 1 hour
- [ ] Conventional commit format
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Validation"`

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: validation -->
<!-- Auto-generated from src/phases/validation.md -->

# Validation Phase

**Purpose:** Verify implementation works as intended.

**Validation level** determines workflow (read from SPAWN_CONTEXT configuration).

---

## Validation: none

**When to use:** Trivial changes where validation overhead exceeds value.

**Workflow:**

1. Confirm changes are complete
2. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
3. **Verify commit** - `git status` shows "nothing to commit"
4. Report completion: `bd comment <beads-id> "Phase: Complete - [brief summary]"`
5. Call /exit to close agent session

**That's it - no validation required.**

---

## Validation: tests

**When to use:** Standard validation for features with test suites.

**Workflow:**

1. **Run test suite** - Use project-specific test command (see reference for examples by language)
2. **Verify all tests pass** - All green, no errors/warnings, adequate coverage
3. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
4. **Verify commit** - `git status` shows "nothing to commit"
5. **Report completion** - `bd comment <beads-id> "Phase: Complete - Tests passing. [summary of test results]"`
6. **Call /exit** - Close agent session

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/validation-examples.md` for:
- Test commands by language (JavaScript, Python, Ruby, Rust, Go)

---

## Validation: smoke-test

**When to use:** Features with UI components, user-facing functionality, or integration points where automated tests don't verify end-to-end behavior.

**Workflow:**

1. **Run test suite** - First verify automated tests pass (see "Validation: tests")
2. **Load feature** - Start dev server, open browser/API client/CLI (see reference for commands)
3. **Verify manually** - Use checklist for Web UI, API, or CLI verification (see reference)
4. **Capture evidence** - Screenshot for UI, request/response for API/CLI
5. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
6. **Verify commit** - `git status` shows "nothing to commit"
7. **Report completion** - `bd comment <beads-id> "Phase: Complete - Smoke test passed. [verification summary]"`
8. **Call /exit** - Close agent session

**Critical:** Tests passing ≠ feature working. Always perform manual verification for user-facing features.

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/validation-examples.md` for:
- Commands to load feature (Web UI, API, CLI)
- Verification checklists (Web UI, API, CLI)

---

## Validation: multi-phase

**When to use:** Complex features with multiple phases where orchestrator needs to manually validate each phase before allowing next phase to proceed.

**Purpose:** Creates explicit checkpoint for orchestrator verification before next phase begins.

**Workflow:**

1. **Run test suite** - Verify automated tests pass
2. **Smoke test (if UI)** - Perform manual verification if feature includes UI
3. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
4. **Verify commit** - `git status` shows "nothing to commit"
5. **Report awaiting validation** - `bd comment <beads-id> "AWAITING_VALIDATION - [phase details, evidence summary]"`
6. **STOP** - Wait for orchestrator approval. DO NOT proceed to next phase
7. **After approval** - Report: `bd comment <beads-id> "Phase: Complete - [summary]"`
8. **Call /exit** - Close agent session

**Critical:** STOP and wait for explicit orchestrator approval. Do not proceed or mark complete without approval.

---

## When Validation Fails

**If tests fail or smoke test reveals issues:**

1. **Check agentlog for runtime errors:**
   ```bash
   agentlog errors
   ```
   This shows logged runtime errors from your implementation. Often reveals the root cause immediately (stack traces, assertion failures, uncaught exceptions).

2. **Analyze failure output** - Read test output carefully for specific assertion failures
3. **Return to Implementation** - Fix the issue, re-run tests
4. **Re-validate** - Repeat validation workflow after fix

---

## Common Issues

**See reference for detailed troubleshooting:**
- Tests pass but feature doesn't work (tests verify logic, not UI/integration)
- Smoke test reveals issues (return to Implementation, fix, re-validate)
- Multi-phase orchestrator finds issues (fix immediately, don't defend)

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/validation-examples.md` for detailed common issues and solutions.

---

## Completion Criteria

**For validation: none:**
- [ ] Changes complete
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: Phase: Complete

**For validation: tests:**
- [ ] Test suite passing
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: Phase: Complete with test results

**For validation: smoke-test:**
- [ ] Test suite passing
- [ ] Manual verification complete
- [ ] Evidence captured (screenshot/output)
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: Phase: Complete with verification summary

**For validation: multi-phase:**
- [ ] Test suite passing
- [ ] Smoke test complete (if UI)
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: AWAITING_VALIDATION
- [ ] Orchestrator manually tested and approved
- [ ] Reported via `bd comment`: Phase: Complete (after approval)

**If ANY box unchecked, validation is NOT complete.**

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: self-review -->
<!-- Auto-generated from src/phases/self-review.md -->

# Self-Review Phase

**Purpose:** Quality gate before completion. Catch anti-patterns, verify commit hygiene, ensure deliverables are complete.

**When you're in this phase:** Implementation and validation are done. Before marking complete, review your own work against quality standards.

---

## Why Self-Review Matters

Agents often mark work "complete" with:
- God objects introduced (files doing too much)
- Incomplete implementations (TODOs, placeholders)
- Poor commit hygiene (WIP commits, wrong types)
- Missing test coverage for edge cases
- Security issues (hardcoded secrets, injection vulnerabilities)
- **Orphaned code (components/functions that exist but aren't wired in)**
- **Demo/placeholder data that should be real data**

Self-review catches these before orchestrator sees them.

---

## Self-Review Checklist

**Perform each check. Document significant findings via `bd comment`.**

### 0. Scope Verification (For Refactoring/Migration Work)

**If your work involved renaming, refactoring, migrating, or changing patterns across the codebase:**

| Check | How | If Failed |
|-------|-----|-----------|
| **Scope was determined** | Ran `rg "old_pattern"` before starting to count all occurrences | Document scope retroactively, verify you found them all |
| **All instances updated** | Run `rg "old_pattern"` now - should return 0 matches | Find and fix remaining instances |
| **New pattern consistent** | Run `rg "new_pattern"` - count matches expected scope | Investigate mismatches |

**Examples:**
```bash
# Renaming a function
rg "oldFunctionName" --type py  # Should be 0
rg "newFunctionName" --type py  # Should match expected count

# Migrating path from .orch/ to .kb/
rg "\.orch/investigations" --type py  # Should be 0
rg "\.kb/investigations" --type py    # Should show new paths

# Updating config pattern
rg "old_config_key" --type yaml  # Should be 0
```

**Why this matters:** Partial migrations are worse than no migration - they create inconsistent state that's hard to debug later.

**Skip this section if:** Your work was purely additive (new files, new functions) with no changes to existing patterns.

---

### 1. Anti-Pattern Detection

Review your changes for common anti-patterns:

| Anti-Pattern | How to Check | If Found |
|--------------|--------------|----------|
| **God objects** | Any file >300 lines or doing multiple concerns? | Extract responsibilities |
| **Tight coupling** | Components directly instantiating dependencies? | Use dependency injection |
| **Magic values** | Hardcoded numbers/strings without explanation? | Extract to named constants |
| **Deep nesting** | Logic nested >3 levels? | Extract to helper functions |
| **Incomplete work** | Any TODO, FIXME, placeholder comments? | Complete or document as known limitation |

### 2. Security Review

Check for common security issues:

- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No SQL injection vulnerabilities (use parameterized queries)
- [ ] No XSS vulnerabilities (escape user input in output)
- [ ] No path traversal (validate file paths)
- [ ] No command injection (avoid shell commands with user input)

**If security issue found:** Fix immediately. Do not proceed.

### 3. Commit Hygiene

Review your commits:

```bash
git log --oneline -10
```

| Check | Standard | If Violated |
|-------|----------|-------------|
| **Conventional format** | `type: description` (feat, fix, refactor, test, docs, chore) | Amend or squash |
| **Atomic commits** | Each commit is one logical change | Squash related commits |
| **No WIP commits** | No "WIP", "temp", "fix typo" commits in history | Squash into meaningful commits |
| **Test/impl separation** | Test commits separate from implementation (TDD mode) | OK to have together if not TDD |

### 4. Test Coverage

Review test adequacy:

- [ ] Happy path tested (main functionality works)
- [ ] Edge cases covered (empty input, boundaries, nulls)
- [ ] Error paths tested (what happens when things fail)
- [ ] No test gaps for new code (every new function has test)

**For TDD mode:** You should already have this covered. Verify.

**For direct mode:** Verify existing tests still pass, no new behavioral code added without tests.

### 5. Documentation Check

- [ ] Public APIs have clear signatures (types, return values)
- [ ] Complex logic has inline comments explaining "why"
- [ ] No commented-out code left behind
- [ ] No debug statements (console.log, print, debugger)

### 6. Deliverables Verification

Cross-check against SPAWN_CONTEXT requirements:

- [ ] All required deliverables exist (investigation, design, tests, implementation as applicable)
- [ ] Deliverables are complete (not stubs or placeholders)
- [ ] Deliverables reported via `bd comment` with paths/summary

### 7. Integration Wiring Check (CRITICAL)

**New code MUST be wired into the system, not just exist in isolation.**

Components that exist but aren't connected are worse than no implementation - they create false confidence that work is done.

| Check | How | If Failed |
|-------|-----|-----------|
| **New modules imported** | Search for imports of your new files (`rg "import.*new-file"` or `rg "require.*new-file"`) | Wire into consuming code or delete orphaned file |
| **New functions called** | Search for calls to new functions (`rg "newFunctionName\("`) | Add calls or delete unused functions |
| **New exports used** | Check that exported symbols are imported elsewhere | Remove unused exports or wire them in |
| **New routes registered** | If adding endpoints, verify they appear in route registration | Register routes in app/router |
| **New components rendered** | If adding UI components, verify they're rendered somewhere | Add to parent component or page |
| **New config referenced** | If adding config options, verify they're read somewhere | Wire config into code that uses it |

**Examples:**
```bash
# New React component - verify it's rendered
rg "import.*NewComponent" --type tsx  # Should find at least one import
rg "<NewComponent" --type tsx          # Should find at least one render

# New API endpoint - verify it's registered
rg "router\.(get|post|put|delete).*\/new-endpoint"  # Should find registration

# New utility function - verify it's called
rg "newUtilFunction\(" --type ts  # Should find at least one call
```

**Why this matters:** "Code exists" ≠ "Feature works". A component that isn't wired in does nothing. Tests may even pass because the dead code path is never executed.

**Red flags (STOP and fix):**
- New file with 0 imports elsewhere
- New export with 0 consumers
- New route handler not in route registry
- New component not rendered anywhere
- New hook not called anywhere

### 8. Demo/Placeholder Data Ban (CRITICAL)

**Work is NOT complete if it contains demo, placeholder, or mock data that should be real.**

This is different from intentional test fixtures or development seeds. The check is: "Would this data cause problems in production?"

| Pattern | Examples | Action |
|---------|----------|--------|
| **Fake identities** | "John Doe", "Jane Smith", "Test User", "Admin User" | Replace with actual data source or configurable value |
| **Placeholder domains** | example.com, test.com, foo.bar, localhost hardcoded | Use environment variables or config |
| **Lorem ipsum** | "Lorem ipsum dolor sit amet...", placeholder text | Replace with real content or clear indication it's a template |
| **Magic numbers as data** | `price: 9.99`, `quantity: 100`, `id: 12345` hardcoded | Use actual data source or named constants with clear purpose |
| **Fake contact info** | "555-1234", "test@example.com", "123 Main St" | Use real data source or redact |
| **Hardcoded credentials** | Any username/password even for "testing" | Use environment variables |
| **Mock responses inline** | JSON blobs hardcoded instead of from API/DB | Wire to actual data source |

**How to check:**
```bash
# Common placeholder patterns
rg -i "john doe|jane smith|test user|admin user" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
rg -i "example\.com|test\.com|localhost" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
rg -i "lorem ipsum" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
rg "555-|123-456|test@|placeholder" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
```

**Exceptions (these are OK):**
- Test fixtures in `/test/`, `/tests/`, `/__tests__/`, `*.test.*`, `*.spec.*` directories
- Seed data explicitly marked as development-only
- Storybook stories or component demos
- Documentation examples

**If found in production code:** STOP. Replace with:
- Environment variable: `process.env.API_URL`
- Config file reference: `config.defaultEmail`
- Dynamic data: data from API/database
- Clear template marker: `"{{USER_NAME}}"` that gets replaced

**Work containing demo data in production paths is NOT complete.**

### 9. Discovered Work Check

*During this implementation, did you discover any of the following?*

| Type | Examples | Action |
|------|----------|--------|
| **Bugs** | Broken functionality, edge cases that fail | `bd create "description" --type bug` |
| **Technical debt** | Workarounds, code that needs refactoring | `bd create "description" --type task` |
| **Enhancement ideas** | Better approaches, missing features | `bd create "description" --type feature` |
| **Documentation gaps** | Missing/outdated docs | Note in completion summary |

**Triage labeling for daemon processing:**

When creating issues for discovered work, apply triage labels so the daemon can process them:

| Confidence | Label | When to use |
|------------|-------|-------------|
| High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Lower | `triage:review` | Uncertain scope, needs orchestrator input |

```bash
# Example: Creating and labeling a discovered bug
bd create "Edge case fails when input empty" --type bug
bd label <issue-id> triage:ready  # High confidence - daemon can auto-spawn

# Example: Uncertain discovery needs review
bd create "Potential performance issue in query" --type task
bd label <issue-id> triage:review  # Lower confidence - human reviews first
```

**Why triage labels matter:** Issues with `triage:ready` are automatically picked up by the daemon for autonomous processing. Without this label, discovered work requires manual intervention.

**Checklist:**
- [ ] **Reviewed for discoveries** - Checked work for patterns, bugs, or ideas beyond original scope
- [ ] **Tracked if applicable** - Created beads issues for actionable items (or noted "No discoveries")
- [ ] **Labeled for triage** - Applied `triage:ready` or `triage:review` based on confidence
- [ ] **Included in summary** - Completion comment mentions discovered items (if any)

**If no discoveries:** Note "No discovered work items" in completion comment. This is common and acceptable.

**Why this matters:** Implementation work often reveals issues beyond the original scope. Beads issues with triage labels ensure these discoveries surface and get processed autonomously rather than getting lost.

---

## Document Findings

**If self-review finds issues:**
1. Fix them before proceeding
2. Report via `bd comment <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report via `bd comment <beads-id> "Self-review passed - ready for completion"`

**Checklist summary (verify mentally, report issues only):**
- Anti-patterns: No god objects, tight coupling, magic values, deep nesting, incomplete work
- Security: No hardcoded secrets, no injection vulnerabilities, no XSS
- Commit hygiene: Conventional format, atomic commits, no WIP commits
- Test coverage: Happy path, edge cases, error paths
- Documentation: APIs documented, no debug statements, no commented-out code
- Deliverables: All required deliverables exist and complete
- **Integration wiring: New code imported/called/rendered somewhere (not orphaned)**
- **Demo data ban: No placeholder data in production code paths**
- Discovered work: Reviewed for discoveries, tracked or noted "No discoveries"

---

## If Issues Found

1. **Fix immediately** - Don't proceed with issues
2. **Commit fixes** - Use appropriate commit type (`fix:`, `refactor:`, `chore:`)
3. **Update checklist** - Mark items as resolved
4. **Re-run affected checks** - Verify fix didn't introduce new issues

---

## Completion Criteria

Before proceeding to mark work complete:

- [ ] All anti-pattern checks passed
- [ ] Security review passed (no vulnerabilities)
- [ ] Commit hygiene verified
- [ ] Test coverage adequate
- [ ] Documentation complete
- [ ] All deliverables verified
- [ ] **Integration wiring verified (new code connected to system, not orphaned)**
- [ ] **No demo/placeholder data in production code paths**
- [ ] Discovered work reviewed and tracked (or noted "No discoveries")
- [ ] Self-review passed and reported via bd comment

**If ANY box unchecked, self-review is NOT complete.**

---

## After Self-Review Passes

1. Report self-review status:
   ```bash
   bd comment <beads-id> "Self-review passed - ready for completion"
   ```

2. Proceed to mark complete:
   - Report: `bd comment <beads-id> "Phase: Complete - [deliverables summary]"`
   - Output: "✅ Self-review passed, work complete"
   - Call /exit to close agent session

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: integration -->
<!-- Auto-generated from src/phases/integration.md -->

# Integration Phase

**Purpose:** Combine multiple validated phases into cohesive feature.

**When to use:** Multi-phase features after all phases validated individually

**Prerequisites:** All dependent phases must be validated and approved by orchestrator.

---

## When Integration is Needed

**Use when:**
- ✅ Feature split into multiple phases (A, B, C...)
- ✅ Each phase validated independently
- ✅ Phases need to work together
- ✅ E2E testing required across phase boundaries

**Skip when:**
- ❌ Single-phase feature
- ❌ Phases completely independent
- ❌ Each phase already tested integration

---

## Workflow

### 1. Review Completed Phases

Review each phase via beads comments (`bd show <beads-id>`) to understand:
- What was implemented
- What was tested
- Open questions/concerns
- Integration points mentioned

### 2. Identify Integration Points

Map where phases interact:
- Data flow between components
- Shared state/configuration
- API contracts between modules
- Database schema dependencies
- Event handling across boundaries

### 3. Integration Testing

Write integration tests for cross-phase scenarios:
- Test interactions between phases
- Verify data flows correctly
- Confirm shared contracts work
- Check error handling across boundaries

### 4. E2E Verification

Test complete user flows across all phases:
1. Define end-to-end scenarios
2. Execute flows manually or automated
3. Verify all phases work together
4. Document test results

### 5. Performance Testing (If Applicable)

- Measure end-to-end performance
- Verify meets requirements
- Document metrics

### 6. Regression Testing

1. Run full test suite
2. Verify existing features still work
3. Check for unintended side effects

### 7. Document Results

Report via beads:
```bash
bd comment <beads-id> "Integration results: Phases [A,B,C] integrated. Integration tests: [X passing]. E2E: [verified]. Performance: [metrics if applicable]"
```

### 8. Final Smoke Test

Perform manual verification of complete feature:
- Use as end user would
- Test all flows across phases
- Verify UI polish
- Check error messages
- Confirm performance

### 9. Move to Validation

Report via `bd comment <beads-id> "Phase: Validation - Integration complete"`

---

## Completion Criteria

- [ ] All phases reviewed (via beads comment history)
- [ ] Integration points identified and documented
- [ ] Integration tests written and passing
- [ ] E2E tests cover complete user flows
- [ ] Performance requirements met (if applicable)
- [ ] No regressions (full test suite passing)
- [ ] Final smoke test passed
- [ ] Integration results reported via beads
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Validation"`

---

## Integration Deliverables

**Required:**
- Integration tests (cross-phase coverage)
- E2E tests (complete user flows)
- Integration documentation

**Optional:**
- Performance metrics
- Architecture diagram
- API documentation
- Deployment guide

<!-- /SKILL-TEMPLATE -->

---

## Phase Transitions

**After completing each phase:**

1. Report progress: `bd comment <beads-id> "Phase: <new-phase> - <brief summary>"`
2. Output: "✅ {Current Phase} complete, moving to {Next Phase}"
3. Continue to next phase guidance

**Beads comments are the primary progress log.** The orchestrator monitors via `bd show <beads-id>`.

---

## Progress Tracking (Beads-First)

**If SPAWN_CONTEXT includes a BEADS ISSUE section, use `bd comment` for progress:**

```bash
# Phase transitions (always comment)
bd comment <issue-id> "Phase: Implementing - Starting TDD cycle for auth middleware"

# Significant milestones
bd comment <issue-id> "Milestone: Core API endpoints complete, tests passing"

# Blockers (immediate)
bd comment <issue-id> "BLOCKED: Need API contract clarification from orchestrator"

# Questions
bd comment <issue-id> "QUESTION: Should validation errors return 400 or 422?"

# Completion
bd comment <issue-id> "Phase: Complete - All tests passing. Deliverables: src/auth/*.ts, tests/auth/*.test.ts"
```

**Why beads comments matter:**
- Orchestrator monitors progress via `bd show <beads-id>`
- Progress history persists across sessions
- Enables automated tracking and reporting
- Comments are searchable across all issues

---

## Special Cases

**Multi-Phase Validation:**
- When validation=multi-phase: STOP at validation, report via `bd comment <beads-id> "AWAITING_VALIDATION - [details]"`
- Wait for orchestrator approval before marking complete
- See Validation phase for details

**Mode Selection:**
- TDD mode: Behavioral changes (write tests first)
- Direct mode: Non-behavioral (refactoring, config, docs)
- If SPAWN_CONTEXT specifies mode, use that mode

**Skipped Phases:**
- If phase not in configuration, skip its guidance
- Follow only the phases specified in SPAWN_CONTEXT

---

## Completion Criteria

- [ ] **Step 0 completed:** Scope enumerated and reported via `bd comment`
- [ ] All configured phases completed
- [ ] Each phase's completion criteria met
- [ ] Self-review passed
- [ ] All deliverables created
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Final status reported: `bd comment <beads-id> "Phase: Complete - [summary of deliverables]"`

**If ANY unchecked, work is NOT complete.**

**Final step:** After all criteria met:
1. Close the beads issue: `bd close <beads-id> --reason "summary of what was done"`
2. Call /exit to close agent session

---

## Troubleshooting

**Stuck:** Re-read phase guidance, check SPAWN_CONTEXT. If blocked: `bd comment <beads-id> "BLOCKED: [reason]"`

**Unclear requirements:** `bd comment <beads-id> "QUESTION: [question]"` and wait for clarification

**Scope changes:** Document change, ask orchestrator via beads comment

---

## Related Skills

- **investigation**, **systematic-debugging**, **architect** (for design exploration), **record-decision**, **code-review**


---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-ivtg.1 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`

⚠️ Your work is NOT complete until you run both commands.

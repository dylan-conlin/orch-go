TASK: Go CLI: orch send command

🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-ph1.8 "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:
1. Run: `bd comment orch-go-ph1.8 "Phase: Complete - [1-2 sentence summary of deliverables]"`
2. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.


CONTEXT: [See task description]

ARCHITECTURE CONTEXT:
- **Orchestration Pattern:** Per-project orchestrators (Architecture B)
  - Multiple `.orch/` directories across projects (meta-orchestration, price-watch, context-driven-dev, etc.)
  - Each project has independent orchestration context
  - Dylan switches contexts via `cd` - not managing all projects from one instance
  - When in `/project-name/`, you ARE that project's orchestrator
- **Key Architectural Constraints:**
  - Projects are architecturally independent (loose coupling)
  - Cross-project dependencies = exception, not rule
  - Shared concerns extracted to libraries, not coordinated via meta-orchestrator

⚠️ **META-ORCHESTRATION TEMPLATE SYSTEM** (Critical if working on meta-orchestration):

**IF task involves these files/patterns:**
- .orch/CLAUDE.md updates
- Orchestrator guidance changes
- Pattern/workflow documentation
- Any file with <!-- ORCH-TEMPLATE: ... --> markers

**THEN you MUST understand the template build system:**

**Template Architecture (3 layers):**
1. **Source:** templates-src/orchestrator/*.md ← EDIT HERE
2. **Distribution:** ~/.orch/templates/orchestrator/*.md (synced via `orch build-global`)
3. **Consumption:** .orch/CLAUDE.md (rebuilt via `orch build --orchestrator`)

**Critical Rules:**
- ❌ NEVER edit .orch/CLAUDE.md sections between `<!-- ORCH-TEMPLATE: ... -->` markers
- ✅ ALWAYS edit source in templates-src/orchestrator/
- ✅ ALWAYS rebuild: `orch build-global && orch build --orchestrator`

**Before editing ANY file:**
```bash
grep "ORCH-TEMPLATE\|Auto-generated" <file>
```

**If file has template markers:**
1. Find source template path in the Auto-generated comment
2. Edit templates-src/orchestrator/[template-name].md
3. Run: `orch build-global` (sync source → distribution)
4. Run: `orch build --orchestrator` (regenerate .orch/CLAUDE.md)
5. Verify changes appear in .orch/CLAUDE.md

**Files that are NOT templates (safe to edit directly):**
- docs/*.md
- tools/orch/*.py
- templates-src/ files (these ARE the source)

**Why this matters:**
- Changes to template-generated sections get SILENTLY OVERWRITTEN on next build
- This is a recurring amnesia bug (see post-mortem: .orch/knowledge/spawning-lessons/2025-11-20-forgot-template-system-context-recurring.md)

**Reference:** .orch/CLAUDE.md lines 77-125 for template system documentation

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
2. **SET UP investigation file:** Run `kb create investigation cli-orch-send-command` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-cli-orch-send-command.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately
   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-ph1.8 "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)
3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input

## PRIOR INVESTIGATIONS (from kb)

*Relevant investigations and decisions discovered. Review for context.*

### CLI Project Scaffolding and Build Setup
- **Path:** `.kb/investigations/2025-12-19-inv-cli-project-scaffolding-build.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**TLDR:** Question: Set up Go CLI project structure with cobra, Makefile, and proper directory layou..._
  - _# Investigation: CLI Project Scaffolding and Build Setup_

### OpenCode Client Package Session Management
- **Path:** `.kb/investigations/2025-12-19-inv-client-opencode-session-management.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**TLDR:** Question: Can we create a reusable Go package for OpenCode session management? Answer: Yes..._
  - _# Investigation: OpenCode Client Package Session Management_

### SSE Event Monitoring Client
- **Path:** `.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**TLDR:** Question: Can we build an SSE client for real-time OpenCode event monitoring? Answer: Yes ..._
  - _# Investigation: SSE Event Monitoring Client_

### OpenCode POC - Spawn Session Via Go
- **Path:** `.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md`
- **Type:** investigations
- **Relevant excerpts:**
  - _**TLDR:** Question: Can we build a Go POC for OpenCode orchestration (spawn, monitor, Q&A)? Answer: ..._
  - _### Finding 1: OpenCode CLI supports --attach mode with JSON output_

*If these investigations are relevant, read the full files for detailed context.*

## ACTIVE SERVICES

*Running services on common dev ports. Use these for API calls/testing.*

- :3000 → com.docke (PID 88787) - likely dev server
- :3001 → com.docke (PID 88787) - likely dev server
- :3306 → mysqld (PID 26684) - likely dev server
- :3334 → bun (PID 89296) - likely dev server
- :4096 → opencode (PID 71798) - likely dev server
- :5000 → ControlCe (PID 754) - macOS Control Center (can ignore)
- :5173 → node (PID 89295) - likely dev server
- :5433 → com.docke (PID 88787) - likely dev server
- :8765 → Python (PID 25263) - likely API server


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-ph1.8**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-ph1.8 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-ph1.8 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-ph1.8 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-ph1.8 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-ph1.8 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-ph1.8`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking


## ADDITIONAL CONTEXT

BEADS ISSUE: orch-go-ph1.8

Issue Description:
Implement orch send for Q&A on sessions.

## Scope
- Send message to existing session
- Support both running and completed sessions
- Stream response to stdout

## Dependencies
- orch-go-ph1.1 (OpenCode session management)

## Implementation
From POC:
- opencode run --attach --session <id> --format json "message"
- Session context preserved - agent remembers conversation

## Current Python Reference
- src/orch/send_commands.py

## Deliverables
- cmd/orch/send.go - send command

## Validation
- orch send <session-id> "follow up question"
- Response shows agent understood context




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

FEATURE-IMPL CONFIGURATION:
Phases: implementation,validation
Mode: tdd
Validation: tests

Follow phase guidance from the feature-impl skill.

INVESTIGATION CONFIGURATION:
Type: simple

Create investigation file in .kb/investigations/simple/ subdirectory.
Follow investigation skill guidance for simple investigations.


ADDITIONAL DELIVERABLES:
- investigation:  (OPTIONAL)
- design:  (OPTIONAL)
- tests:  (OPTIONAL)
- implementation:  (OPTIONAL)
- validation-evidence:  (OPTIONAL)

COORDINATION ARTIFACT: .
(Investigation file is your deliverable - update Status when complete)


VERIFICATION REQUIRED:
- [ ] All configured phases completed (investigation findings, design docs, implementation, validation evidence as applicable)
- [ ] Tests pass OR validation evidence documented via bd comment (automated tests, smoke test, or multi-phase validation)
- [ ] Implementation matches design (if design phase used)
- [ ] No regressions introduced (existing functionality still works)
- [ ] All deliverables committed and reported via bd comment

IMPORTANT: Ensure these requirements are met before reporting Phase: Complete via `bd comment`.

CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md
- CDD: ~/orch-knowledge/docs/cdd-essentials.md
- Process guide: ~/.claude/skills/implementation/feature-impl/SKILL.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:
1. `bd comment orch-go-ph1.8 "Phase: Complete - [1-2 sentence summary]"`
2. `/exit`

⚠️ Your work is NOT complete until you run both commands.
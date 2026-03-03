# Implementation Phase (TDD Mode)

**Purpose:** Implement feature using test-driven development.

**When to use:** Feature adds/changes behavior (APIs, business logic, UI interactions).

**Core principle:** If you didn't watch the test fail, you don't know if it tests the right thing.

---

## Pre-Implementation Exploration (ADVISORY)

> **Note:** This is an **advisory checkpoint** - suggested prep work before TDD. Exploration helps you write better tests, but isn't enforced.

**Before writing code, explore the codebase.** This prevents incomplete implementations and missed integration points.

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

## UI Feature Requirements (web/ Changes)

**⚠️ CRITICAL: Tests passing ≠ feature working (especially for UI)**

If you modified ANY file in `web/` directory, visual verification is MANDATORY.

### Check for web/ Changes

```bash
git diff --name-only | grep "^web/"
```

**If this returns files → you MUST complete visual verification before Phase: Complete.**

### Visual Verification Workflow

1. **Rebuild and restart:**

   ```bash
   make install
   orch servers stop <project>
   orch servers start <project>
   ```

2. **Capture screenshot via playwright-cli:**
   ```bash
   playwright-cli open http://localhost:5188/your-page
   playwright-cli screenshot
   ```
   - Screenshot MUST show the UI changes you made

3. **Document evidence in beads:**
   ```bash
   bd comment <beads-id> "Visual verification: [describe what screenshot shows, key UI elements visible, state verified]"
   ```

### What to Verify Visually

- [ ] Component renders without errors
- [ ] Layout correct (no overlapping, proper spacing)
- [ ] Data displays correctly (not empty, not placeholder)
- [ ] Interactions work (clicks, hover states, transitions)
- [ ] No console errors in browser

**⛔ If `playwright-cli` is not available:**

```bash
bd comment <beads-id> "BLOCKED: UI changes require visual verification but playwright-cli is not available"
```

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
- [ ] Reported via beads with **actual test output**: `bd comment <beads-id> "Phase: Validation "Tests: <command> - <actual output>"`

**Test Evidence Requirement:**

- Format: `Tests: <command> - <actual output summary>`
- Good: `bd comment <id> "Phase: Validation - Tests: go test ./... - 12 passed, 0 failed"`
- Bad: `bd comment <id> "Phase: Validation - Implementation complete, tests passing"` (no evidence)

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/tdd-best-practices.md` for complete checklist.

---

## When to Move to Validation Phase

Once completion criteria met → Report via `bd comment <beads-id> "Phase: Validation "Tests: <command> - <output>"` → Proceed to Validation phase

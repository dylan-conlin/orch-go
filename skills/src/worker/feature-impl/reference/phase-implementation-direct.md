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
bd comments add <beads-id> "Pre-impl exploration complete: [N] files to modify, [M] callers found. Confirmed non-behavioral: [reasoning]. Tests to verify: [list]"
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

Report via `bd comments add <beads-id> "Phase: Validation - Tests: <command> - <actual output>"`

**Test Evidence Requirement:**
- Include actual test command and output summary
- Good: `Tests: go test ./... - 47 passed, 0 failed`
- Bad: `Direct mode changes complete, tests passing` (no evidence)

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
- [ ] Reported via beads with **actual test output**: `bd comments add <beads-id> "Phase: Validation - Tests: <cmd> - <output>"`

# Investigation: Multi-Phase Feature Orchestration Process Gaps

**Date:** 2025-11-18
**Status:** Complete
**Investigator:** Orchestrator (self-analysis)
**Sparked By:** Price-watch time-series comparison implementation session revealing systematic orchestration failures
**Confidence:** High (90%) - Based on live session analysis with clear failure patterns

---

## TLDR

Multi-phase feature development (4+ hours, multiple validation points) exposed critical orchestration gaps: missing validation checkpoints between phases, test coverage blindness (tests pass ≠ feature works), context management failures (tried to pivot instead of respawn), and phase discipline breakdown (jumped to Phase B without validating Phase A). Root cause: optimized for speed over correctness, lacked "show me it works" culture.

**Key insight:** Current orchestration patterns work for isolated tasks but fail for iterative, multi-session feature development requiring intermediate validation.

---

## Problem Statement

**Session context:** Implementing time-series price comparison view for price-watch project, structured as 4 phases (A: foundation 5-8h, B: UX wins 3-5h, C: competitor validation, D: quantity expansion).

**What broke:**
1. Phase A agent claimed "complete, 240 tests passing"
2. Orchestrator spawned Phase B immediately (no validation)
3. User tested Phase A manually: broken (no styling, $0.00 prices, wrong run selector)
4. Attempted to redirect Phase B agent mid-flight → context blown
5. Tried to resume out-of-context agent → user stopped me
6. Finally spawned fresh debugging agent (should have done this immediately)

**Impact:**
- Wasted agent context (Phase B started on broken foundation)
- Delayed feature delivery (debugging Phase A instead of completing Phase B)
- Trust erosion ("why didn't tests catch this?")

---

## Timeline Analysis

### What Happened (Actual)

```
14:00 - Investigation reveals 4-phase plan (A, B, C, D)
14:05 - Spawn Phase A agent (time-series foundation, 5-8h scope)
14:35 - Phase A agent: "Complete! 240 tests passing!"
14:37 - Orchestrator: "Great! Let's spawn Phase B for UX wins"
14:40 - User: "Wait, we only did Phase A, not the full feature"
14:41 - Orchestrator: "Correct, Phase A done. Spawning Phase B now..."
14:45 - User tests Phase A: "Broken - no styling, $0.00 prices"
14:46 - Orchestrator pivots Phase B agent to fix Phase A bugs
14:50 - User: "Agent is out of context"
14:51 - Orchestrator tries to resume same agent (mistake)
14:52 - User: "Stop, you sent that to the out-of-context agent"
14:53 - Orchestrator spawns fresh debugging agent (should have done at 14:46)
```

### What Should Have Happened (Ideal)

```
14:00 - Investigation reveals 4-phase plan
14:05 - Spawn Phase A agent with scope: "ONLY Phase A foundation - stop here"
14:35 - Phase A agent: "Complete! Tests passing. Smoke test: [screenshot at /quotes/comparison]"
14:37 - Orchestrator: "Dylan, please validate Phase A at /quotes/comparison before we proceed to Phase B"
14:40 - User tests, finds issues: "No styling, $0.00 prices"
14:41 - Orchestrator: "Spawning debug agent to fix Phase A issues"
14:55 - Debug agent: "Fixed styling + data loading. Screenshot attached."
14:56 - User validates: "Works now"
14:57 - Orchestrator: "Phase A validated ✅. Proceed with Phase B?"
14:58 - User: "Yes"
14:59 - Spawn Phase B agent (assumes Phase A works)
```

**Time difference:** ~15 minutes faster, zero wasted agent context, cleaner workflow.

---

## Root Cause Analysis

### Failure 1: Missing Validation Checkpoint

**What happened:** Agent claimed complete → Orchestrator immediately spawned next phase → Discovered broken after Phase B started.

**Why it happened:**
- No explicit validation requirement between phases
- "Tests passing" treated as completion signal
- Optimized for speed ("keep momentum") vs correctness ("prove it works")

**Pattern:** Tests pass ≠ feature works (especially for UI)

**Evidence:**
- Agent: "240 tests passing" ✅
- Reality: No stylesheets loaded ❌
- Reality: All prices $0.00 ❌
- Reality: Run selector broken ❌
- Reality: Deltas not visible ❌

**Test coverage was logic-focused, not integration-focused:**
```ruby
# What tests covered (logic)
test "comparison action returns current and prior quotes" do
  assert_response :success
  assert_not_nil @controller.instance_variable_get(:@current_quotes)
end

# What tests DIDN'T cover (integration)
test "comparison view actually displays prices" # ❌ Missing
test "stylesheets load correctly" # ❌ Missing
test "run selector defaults to different runs" # ❌ Missing
test "deltas visible in rendered HTML" # ❌ Missing
```

**Fix required:** Mandatory validation checkpoint + smoke test requirement.

---

### Failure 2: Test Quality Gap (UI Features)

**What happened:** Tests verified logic existed but not that UI worked.

**Root cause:** TDD skill optimized for backend logic, not frontend integration.

**Gap:** No requirement for:
- Loading page in browser
- Verifying visual rendering
- Screenshot documentation
- End-to-end smoke test

**Current test pyramid for UI features:**
```
Unit tests (logic) ✅ - Agent wrote these
Integration tests (controller+view) ⚠️ - Partial
Smoke tests (load in browser) ❌ - Not required
Visual verification ❌ - Not required
```

**Should be:**
```
Unit tests ✅
Integration tests ✅
Smoke test ✅ - MANDATORY before complete
Screenshot ✅ - MANDATORY in workspace
Manual validation ✅ - Orchestrator asks Dylan
```

---

### Failure 3: Context Management Blindness

**What happened:**
```
Phase B agent working (consuming context)
  ↓
Orchestrator sends pivot instructions ("forget Phase B, fix Phase A")
  ↓
Agent consumes more context processing pivot
  ↓
User: "Agent is out of context"
  ↓
Orchestrator tries to resume (mistake)
  ↓
User: "Stop, that's the out-of-context agent"
  ↓
Orchestrator finally spawns fresh (correct)
```

**Why it happened:**
- Salvage instinct ("can we fix this agent?") vs fresh start
- Didn't recognize context exhaustion signals
- No clear decision tree for pivot vs respawn

**Pattern:** When Dylan says "out of context" → It's out of context (don't argue)

**Fix required:** Clear decision tree for context management.

---

### Failure 4: Phase Discipline Breakdown

**What happened:** Investigation showed 4 phases (A, B, C, D) but orchestration didn't enforce boundaries.

**Pattern:**
```
Investigation: "Phase A (5-8h) → Phase B (3-5h) → Phase C → Phase D"

Spawn: "Implement time-series comparison" (no phase scope)
  ↓
Agent interprets as "do everything" or "just foundation"?
  ↓
Claims complete after foundation (Phase A only)
  ↓
Orchestrator: "Great! Phase B next!"
  ↓
No validation of Phase A first
```

**Should have been:**
```
Investigation: Shows Phases A, B, C, D

Spawn: "ONLY Phase A: foundation. STOP after Phase A."
  ↓
Agent completes Phase A
  ↓
CHECKPOINT: Dylan validates
  ↓
If pass: "Phase A ✅. Proceed with Phase B?"
If fail: Debug agent for Phase A
```

**Fix required:** Explicit phase scoping in spawn prompts + tracking.

---

## Concrete Process Improvements

### A. Validation Checkpoints (CRITICAL)

**Current behavior:**
```python
agent.phase == "Complete" and agent.tests_passing:
    orch_complete(agent)
    spawn_next_agent()  # ❌ No validation
```

**Required behavior:**
```python
agent.phase == "Complete" and agent.tests_passing:
    if agent.deliverable_type == "UI feature":
        # MANDATORY checkpoint for UI
        ask_dylan(f"Please test {agent.url} and report results")
        wait_for_validation()

        if validation_passed:
            orch_complete(agent)
            ask_dylan("Proceed with next phase?")
        else:
            spawn_debug_agent(f"Fix {agent} validation failures")
    else:
        # Non-UI can auto-complete
        orch_complete(agent)
```

**Validation types by deliverable:**
- **UI features:** Manual browser test required
- **API endpoints:** Smoke test with curl/Postman
- **Backend logic:** Existing tests sufficient
- **Infrastructure:** Deployment test in staging

---

### B. Smoke Test Requirement (TDD Skill Update)

**Add to test-driven-development skill:**

```markdown
## UI Feature Testing Requirements

For any deliverable involving views/controllers:

### Phase: Final Validation (MANDATORY before Phase: Complete)

1. **Smoke Test**
   - Load the page in browser: `open http://localhost:3000/quotes/comparison`
   - Verify visual rendering (not just data)
   - Check for console errors (browser DevTools)
   - Screenshot the working feature

2. **Workspace Documentation**
   ```markdown
   ## Smoke Test Results

   **URL Tested:** /quotes/comparison
   **Screenshot:** [attach or describe]
   **What Worked:**
   - Grid renders with styling
   - Prices display correctly
   - Deltas visible with color coding
   - Run selector shows two different runs

   **What Didn't Work:**
   - [List any issues, or "None - all working"]

   **Browser:** Chrome/Firefox/Safari
   **Tested At:** 2025-11-18 14:35
   ```

3. **Completion Criteria**
   - [ ] Unit tests pass
   - [ ] Integration tests pass
   - [ ] Smoke test pass (browser verification)
   - [ ] Screenshot in workspace
   - [ ] No console errors

**Only mark Phase: Complete if ALL criteria met.**
```

---

### C. Multi-Phase Feature Pattern

**Add to spawning guidance:**

```markdown
## Multi-Phase Feature Orchestration

When investigation reveals multi-phase work (Phases A, B, C, etc.):

### 1. Explicit Phase Scoping

Spawn prompt MUST include:
- **SCOPE:** "ONLY Phase A - stop after foundation complete"
- **OUT OF SCOPE:** "DO NOT implement Phase B (UX wins) or Phase C"
- **Completion criteria:** "Mark complete after Phase A validated"

Example:
```
Task: Implement Phase A: Time-series comparison foundation

SCOPE (Phase A only):
- Comparison view at /quotes/comparison
- Collection run selector (current vs prior)
- Delta calculation (current - prior)
- Color coding (green=decreased, red=increased)

OUT OF SCOPE (Phase B - separate agent):
- Unit prices (price/quantity)
- Average column
- Size/complexity metadata

STOP after Phase A smoke test passes.
```

### 2. Phase Progress Tracking

Orchestrator maintains phase status:
```
Time-Series Comparison Feature:
  Phase A: Foundation [✅ VALIDATED]
  Phase B: UX Wins [⏳ IN PROGRESS]
  Phase C: Competitor Validation [❌ NOT STARTED]
  Phase D: Quantity Expansion [❌ NOT STARTED]
```

### 3. Checkpoint Between Phases

```
Phase A complete →
  CHECKPOINT: Dylan validates Phase A

If validation passes:
  Orchestrator: "Phase A validated ✅. Proceed with Phase B?"
  Dylan: [Yes/No/Defer]

If yes:
  Spawn Phase B agent (assumes Phase A works)

If no:
  Spawn debug agent (fix Phase A issues)
```

### 4. No Phase-Skipping

- Cannot start Phase C until A and B validated
- Cannot mark feature "complete" until all phases done
- Each phase has explicit dependencies documented

**Enforcement:** Spawn prompt references prior phases:
```
Task: Implement Phase C

DEPENDENCIES:
- Phase A: Foundation ✅ (validated 2025-11-18)
- Phase B: UX Wins ✅ (validated 2025-11-18)

Phase C assumes A and B work correctly.
```
```

---

### D. Context Management Decision Tree

**Add to orchestration workflow:**

```markdown
## When to Pivot vs Respawn

### Agent Needs Redirection

**Scenario:** Agent working on Task A, need to pivot to Task B

**Decision tree:**
1. Is agent context > 50%? → Spawn fresh agent for Task B
2. Is agent context < 50%? → Can try pivot with clear instructions
3. Did Dylan say "out of context"? → IMMEDIATELY spawn fresh (don't argue)

**Red flags (spawn fresh):**
- Agent has been working 2+ hours
- Agent already pivoted once
- Task B unrelated to Task A
- Dylan reports context issues

**Green lights (can pivot):**
- Agent just started (<30 min)
- Task B is refinement of Task A
- Simple scope change
- Agent confirms understanding

### Agent Completion with Issues

**Scenario:** Agent claims complete but Dylan finds bugs

**Don't:**
- ❌ Send more instructions to completed agent
- ❌ Try to resume for "one more fix"
- ❌ Argue about whether it's actually complete

**Do:**
- ✅ Spawn fresh debugging agent immediately
- ✅ Reference completed agent's workspace
- ✅ Focused scope: "Fix issues X, Y, Z found in validation"

**Example:**
```bash
# Agent claims complete but validation fails
orch spawn systematic-debugging "Fix Phase A validation failures: no styling, $0.00 prices, run selector broken" --project price-watch
```

### Trust Dylan's Signals

- Dylan says "out of context" → It is (don't question)
- Dylan says "test it first" → Stop and wait for validation
- Dylan says "spawn fresh" → Don't try to salvage
- Dylan says "this is broken" → Believe the bug report
```

---

## Success Metrics

**How we'll know these improvements work:**

### Short-term (1-2 weeks)
- ✅ Fewer "tests pass but feature broken" incidents
- ✅ Dylan catches issues at checkpoints (not 2 phases later)
- ✅ Faster debug cycles (fresh spawns vs context salvage)
- ✅ Better phase discipline (complete → validate → next)

### Medium-term (1 month)
- ✅ UI features have screenshot documentation in workspaces
- ✅ Smoke tests catch rendering issues before orchestrator sees them
- ✅ Multi-phase features complete faster (less rework)
- ✅ Cleaner agent context usage (fewer blown contexts)

### Long-term (2-3 months)
- ✅ Validation culture embedded in workflow
- ✅ Agents proactively request validation checkpoints
- ✅ Complex features broken into validated phases by default
- ✅ Trust in "agent says complete" restoration

**Measurement:**
- Track: validation failures per feature
- Track: context blow rate (respawns / total agents)
- Track: rework rate (bugs found after "complete")
- Track: time from "complete" to "actually works"

---

## Recommendations

### Immediate (This Week)

1. **Update TDD skill** - Add smoke test requirement for UI features
2. **Update spawning guidance** - Add multi-phase feature pattern
3. **Create checkpoint template** - Standard validation checklist
4. **Document context decision tree** - When to pivot vs respawn

### Short-term (Next 2 Weeks)

5. **Pilot on next UI feature** - Test new validation workflow
6. **Create workspace template** - Add "Smoke Test Results" section
7. **Train on signals** - When Dylan says X, do Y (trust the signals)
8. **Add phase tracking** - Visual progress [A✅ | B⏳ | C❌ | D❌]

### Long-term (Next Month)

9. **Automated smoke tests** - Playwright tests for critical paths
10. **Visual regression testing** - Screenshot diff on UI changes
11. **Validation metrics** - Track and review checkpoint effectiveness
12. **Process retrospectives** - Weekly review of orchestration patterns

---

## Related Patterns

### What This Fixes
- "Tests pass but feature broken" syndrome
- Wasting agent context on broken foundations
- Phase B depending on broken Phase A
- Salvage attempts on out-of-context agents
- Speed over correctness optimization

### What This Doesn't Fix
- Agent technical capabilities (still need good implementation)
- Investigation quality (still need thorough analysis)
- Dylan's availability for validation (human in the loop)
- Long-running features (still need multi-session patterns)

### Complementary Patterns
- Checkpoint management (pause/resume for long features)
- Session warming (load context efficiently on resume)
- Workspace conventions (state documentation)
- Investigation thoroughness (understand before building)

---

## The Big Insight

**Multi-phase, multi-session features need orchestration discipline:**

1. **Clear phase boundaries** - Explicit scope, no ambiguity
2. **Validation checkpoints** - Between phases, not just at end
3. **Smoke testing** - For UI features, prove it renders
4. **Context awareness** - When to pivot vs respawn
5. **Trust signals** - If Dylan says it's broken/out-of-context, it is
6. **Phase tracking** - Visual progress through complex work

**Current orchestration optimizes for:**
- ✅ Isolated tasks (single agent, clear scope)
- ✅ Backend logic (tests = validation)
- ✅ Investigation (research, not building)

**Gaps for:**
- ❌ Multi-phase features (4+ phases over hours/days)
- ❌ UI features (visual rendering, not just logic)
- ❌ Iterative development (build → validate → refine)

**This session exposed:** We optimized for speed when we should have optimized for correctness. "Let's do Phase B!" felt like progress, but we built on a broken foundation.

---

## Actionable Next Steps

**For next UI feature in price-watch or meta-orchestration:**

1. **Before spawning Phase A:**
   - Document all phases explicitly
   - Set phase boundaries in scope
   - Plan validation checkpoints

2. **During Phase A:**
   - Agent includes smoke test before complete
   - Screenshot in workspace mandatory
   - Tests pass ≠ done (validation required)

3. **After Phase A claims complete:**
   - STOP - don't spawn Phase B yet
   - Ask Dylan to validate manually
   - If broken: spawn debug agent
   - If working: ask "Proceed to Phase B?"

4. **Monitor for patterns:**
   - Did validation catch issues?
   - Was respawn faster than salvage?
   - Did phase discipline prevent rework?

**Success = Next multi-phase feature completes with zero "but it's broken" surprises.**

---

## Related Files

- Session context: Price-watch time-series comparison implementation
- Investigation: `.orch/investigations/2025-11-18-analyze-jim-original-heat-map-design.md` (4-phase plan)
- Failed agent: `2025-11-18-tdd-impl-time-series-price-comparison-view-showing-period` (claimed complete, was broken)
- Pivot attempt: `2025-11-18-tdd-enhance-comparison-view-unit-prices-average-column-per` (spawned too early, context blown)
- Recovery agent: `2025-11-18-debug-debug-fix-comparison-view-phase-issues-styling-bin` (fresh spawn, correct approach)

---

**Investigation Status:** Complete
**Confidence Level:** High (90%) - Based on live session, clear patterns, actionable fixes
**Actionable:** Yes - Specific process changes with success metrics
**Blockers:** None - Can implement immediately

# Probe: Claim 4 — Task-Type Scope Analysis

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** COORD-04
**verdict:** scopes

---

## Question

Claim 4 states: "Coordination failure pattern is consistent across two tested task complexities." The epistemic audit flagged this as overclaimed because only 2 task families were tested (simple/complex same-file additive edits). Does the evidence actually demonstrate task-complexity independence, or does it only demonstrate consistency within a narrow structural task class?

**Falsification condition:** Find evidence that the coordination pattern differs by task type, or identify structural properties shared by the two tested task types that wouldn't hold for other task types.

---

## What I Tested

Analytical probe across existing experimental data (no new experiment required):

1. Examined all 100 trials from 4-condition experiment + gate extension + anticipatory placement
2. Analyzed messaging plan files from 20 messaging trials (simple + complex)
3. Compared diff hunk headers across conditions to identify insertion points
4. Analyzed task structure properties shared between simple and complex tasks

```bash
# Extracted hunk headers from all conditions
grep '^@@' results/*/gate/simple/trial-*/agent-*/full_diff.txt
# Result: ALL agents insert at @@ -92 @@ (after FormatDurationShort) regardless of task type

# Examined messaging plans from trials 1-10 simple and complex
cat results/20260310-174045/messaging/*/trial-*/messages/plan-*.txt
# Result: ALL plans specify "after FormatDurationShort (line 94)" as insertion point
```

---

## What I Observed

### Finding 1: Both task types share the same structural coordination challenge

Simple tasks (FormatBytes + FormatRate) and complex tasks (VisualWidth + FormatTable) share these properties:
- **Both are additive** — agents add new functions, they don't modify existing code
- **Both have the same gravitational point** — prompts say "Place after FormatDurationShort" and display.go ends at line 94
- **Both produce the same merge geometry** — two hunks starting at `@@ -92 @@`

The tasks differ in implementation complexity (40s vs 170s average) but the coordination challenge is structurally identical: two additions at the same file position.

### Finding 2: Anticipatory placement experiment already shows task-type sensitivity

When a DIFFERENT coordination mechanism was tested (anticipatory/LLM-generated placement), task type mattered:

| Task Type | Anticipatory Success | Why |
|-----------|---------------------|-----|
| Simple (FormatBytes/FormatRate) | 2/10 (20%) | LLM picks adjacent functions as "different" — no semantic congruence |
| Complex (VisualWidth/FormatTable) | 10/10 (100%) | LLM assigns VisualWidth near StripANSI — semantic congruence present |

The same coordination mechanism produces different outcomes per task type. This directly shows coordination patterns are NOT task-complexity-independent — they depend on the interaction between task structure and mechanism.

### Finding 3: Messaging plans reveal identical false models across task types

From 20 messaging trials (10 simple, 10 complex), agents wrote plans in 18/20 trials. Every plan contained the same structural error:

**Simple Trial 1, Agent A:** "Both functions placed after FormatDurationShort (same insertion point) ... Can be merged by inserting both after FormatDurationShort"

**Complex Trial 1, Agent A:** "No conflicts expected — Functions added after existing code"

**Complex Trial 2, Agent B:** "If FormatTable is added after same location, no conflict — Both functions are independent and don't interact"

The false model is identical: agents believe additions at the same point with different function names will merge cleanly. This is a git-mechanics misunderstanding that is **independent of task complexity** but **dependent on task structure** (additive at shared insertion point).

### Finding 4: Structurally different task types would change the pattern

Consider a modification task: Agent A refactors `Truncate` (lines 14-19), Agent B refactors `FormatDuration` (lines 49-82). These agents modify different existing functions at different file locations. The "gravitational insertion point" problem doesn't apply because each modification is anchored to its target function. Messaging about "what I'm modifying" would be informative and the agents would produce non-overlapping diffs by construction.

This is a prediction, not an observation — no modification-task experiment has been run. But it identifies the structural reason Claim 4 holds: both tested task types are additive with shared gravitational points. Remove that structural property and the claim's prediction changes.

---

## Model Impact

- [x] **Scopes** claim: "consistent across two tested task complexities" → "consistent across additive same-file tasks with shared gravitational insertion points, across two implementation complexities"

**Specific changes needed in model.md:**

1. Claim 4 title should narrow from "task complexities" to "additive task complexities" — making explicit that both tested task families share the gravitational-convergence structure
2. The anticipatory experiment data (already in the model) should be cross-referenced as evidence of task-type sensitivity
3. The Boundaries section should add: "Modification tasks, refactoring tasks, and cross-file dependency tasks" to what the model does NOT cover

**Evidence tier:** Remains "Working-hypothesis" but the qualifying details should expand to note that both tested task families share the additive/gravitational structure.

---

## Notes

This probe is analytical, not experimental. A controlled experiment with a modification task (agents refactoring different existing functions) would provide direct evidence. The prediction: messaging would produce 0% conflict rate for modification tasks because the insertion point is inherently determined by the target function.

The key insight: "task complexity" (how hard the implementation is) is the wrong variable. The right variable is "task structure" (whether agents independently converge on the same location). Simple and complex tasks tested the same structure; future probes should test different structures.

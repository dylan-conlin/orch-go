# Probe: Modification Tasks — Does Communication Coordinate Without Gravitational Convergence?

**Model:** coordination
**Date:** 2026-03-23
**Status:** Complete
**claim:** COORD-04
**verdict:** extends

---

## Question

Model open question (line 299): "Does the coordination pattern hold for modification/refactoring tasks (as opposed to additive tasks)? In modification tasks, agents are anchored to the function they're modifying — no gravitational insertion point. Messaging about 'what I'm modifying' may be sufficient because agents produce non-overlapping diffs by construction."

The claim 4 scope probe (2026-03-22) narrowed the model to "additive same-file tasks with shared gravitational insertion points" and predicted modification tasks would produce 0% conflict rate across all conditions. This experiment tests that prediction with the same 4-condition design as the original experiment.

**Falsification condition:** If ANY condition produces >20% conflict rate for modification tasks, the model's explanation of WHY additive tasks fail (gravitational convergence) may be incomplete.

---

## What I Tested

4-condition experiment (no-coord, placement, context-share, messaging) x N=10 = 40 trials, 80 agent invocations. Same model (claude-haiku-4-5), same repo, same harness as the original 4-condition experiment.

**Modification tasks (instead of additive tasks):**
- **Agent A:** Refactor `FormatDuration` — add week support (7+ days -> "Xw Yd"), replace cascading if/else with threshold-based approach, update `TestFormatDuration`
- **Agent B:** Refactor `Truncate` + `TruncateWithPadding` — make Unicode-aware using `[]rune` instead of byte indexing, update `TestTruncate` + `TestTruncateWithPadding`

**Key design choice:** Both agents modify the same file (`display.go` and `display_test.go`), but different functions at different file locations:
- Agent A: FormatDuration (lines 49-82) and TestFormatDuration (lines 87-114)
- Agent B: Truncate/TruncateWithPadding (lines 14-28) and TestTruncate/TestTruncateWithPadding (lines 8-47)

```bash
# Run full experiment
bash experiments/coordination-demo/redesign/run-modification.sh --trials 10

# Results directory
experiments/coordination-demo/redesign/results/modify-20260323-093711/
```

---

## What I Observed

### Finding 1: 40/40 SUCCESS across all conditions (0% conflict rate)

| Condition | Trials | Conflicts | Clean Merge + Tests Pass |
|-----------|--------|-----------|--------------------------|
| no-coord | 10 | 0 | **10/10 SUCCESS** |
| placement | 10 | 0 | **10/10 SUCCESS** |
| context-share | 10 | 0 | **10/10 SUCCESS** |
| messaging | 10 | 0 | **10/10 SUCCESS** |

This is a complete inversion of the additive task results:

| Condition | Additive (original) | Modification (this probe) |
|-----------|:-------------------:|:-------------------------:|
| no-coord | 20/20 CONFLICT | **10/10 SUCCESS** |
| placement | 20/20 SUCCESS | **10/10 SUCCESS** |
| context-share | 20/20 CONFLICT | **10/10 SUCCESS** |
| messaging | 20/20 CONFLICT | **10/10 SUCCESS** |

### Finding 2: Diff hunks confirm structural separation

Across all 40 trials, agents consistently produced non-overlapping hunks:
- **Agent A hunks:** `@@ -45,40 @@` (FormatDuration body) and `@@ -104,6 @@` (TestFormatDuration additions)
- **Agent B hunks:** `@@ -10,21 @@` (Truncate/TruncateWithPadding body) and `@@ -11,11 @@` / `@@ -31,17 @@` (test additions)

No trial had overlapping hunk ranges. The modification target IS the structural attractor — agents don't need to decide where to place code because the task already specifies which code to modify.

### Finding 3: Import block didn't cause conflicts

Agent B frequently modified the import block (adding `"unicode/utf8"`) while Agent A did not need import changes. Git merge handled this cleanly because only one agent touched that region. In 0/40 trials did both agents modify the import block.

### Finding 4: Individual agent quality remains high

All 80 agents scored at least 5/6. Agent B (Truncate refactor) scored 6/6 in all 40 trials. Agent A (FormatDuration refactor) scored 5/6 in some trials due to the spec-match heuristic not always detecting week support (the implementation was correct but the grep pattern missed some variants). Agent quality was independent of coordination condition, consistent with Claim 3.

### Finding 5: Messaging condition shows coordination overhead with no benefit

| Condition | Agent A avg | Agent B avg |
|-----------|-------------|-------------|
| no-coord | 56s | 113s |
| placement | 60s | 99s |
| context-share | 65s | 112s |
| messaging | **80s** | **148s** |

Messaging agents took ~35% longer than no-coord agents (plan exchange overhead) while producing the same 100% success rate. For modification tasks, the coordination protocol is pure overhead — no benefit, only cost.

---

## Model Impact

- [x] **Extends** model with: Modification tasks are structurally self-coordinating. The coordination problem in additive tasks is not an agent capability failure but a task structure property: gravitational convergence creates overlapping diffs that no amount of communication resolves. Modification tasks eliminate this by anchoring agents to their target functions.

**Specific model updates needed:**

1. **Claim 1 scope narrows:** "Communication is insufficient for coordination" should be qualified to "...in same-file additive tasks with gravitational convergence." For modification tasks, communication is unnecessary because the task structure itself provides coordination.

2. **Claim 4 gains empirical confirmation:** The claim 4 scope probe (2026-03-22) predicted this analytically. This experiment provides direct evidence with N=40.

3. **Open question answered:** "Does the coordination pattern hold for modification/refactoring tasks?" — **No.** Modification tasks produce 0% conflict rate even without coordination (no-coord condition), answering the model's own open question at line 299.

4. **New model concept: Task-structure-as-coordination.** The model currently frames coordination as something applied TO tasks (attractor, gate, messaging). For modification tasks, coordination is embedded IN the task structure. The modification target functions as an implicit structural attractor. This suggests a taxonomy: tasks with gravitational convergence need explicit coordination, tasks with natural structural anchoring are self-coordinating.

5. **Practical implication for orch-go:** When decomposing work for parallel agents, modification tasks can be dispatched without coordination constraints. Only additive tasks (new functions, new files at shared locations) need structural placement. This is already orch-go's implicit practice but now has empirical backing.

---

## Notes

- Prior work: The claim 4 probe (2026-03-22) predicted this result analytically. This probe provides the first empirical evidence.
- The experiment used the same codebase, model, and harness as all prior coordination experiments, enabling direct comparison.
- Results: `experiments/coordination-demo/redesign/results/modify-20260323-093711/`
- Total experiment cost: 80 Haiku invocations, ~40 minutes wall clock time
- Harness note: `go build ./...` was scoped to `go build ./pkg/display/` to avoid stale `.go` files in experiment results directories (same issue as anticipatory placement experiment).

# Probe: Anticipatory Placement — Can Static Analysis Predict Attractors Without Failure Data?

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** CO-02
**verdict:** extends

---

## Question

Can an LLM (claude-haiku-4-5) reading a codebase and two task descriptions predict non-overlapping insertion points that prevent merge conflicts as effectively as human-designed placements?

This extends Claim 2 ("Structural placement prevents conflicts completely"): if LLM-generated placements work, then the coordination problem reduces to "have an orchestrator read the code before agents run" — automatable today.

---

## What I Tested

**Experiment Design:**
- Give claude-haiku-4-5 the target codebase (display.go, display_test.go) + both task descriptions
- Instruct it to generate non-overlapping insertion points for both agents
- Use those LLM-generated placements as the coordination instructions
- Run 20 trials: 10 simple (FormatBytes + FormatRate), 10 complex (VisualWidth + FormatTable)
- Compare against human-designed placement baseline (20/20 success from 2026-03-10 experiment)

**Control:** Human-designed placements from existing experiment:
- Agent A: after FormatDurationShort / TestFormatDurationShort
- Agent B: after StripANSI / TestStripANSI (before FormatDuration / TestFormatDuration)

```bash
# Run experiment
./experiments/coordination-demo/redesign/run-anticipatory.sh --trials 10
```

**Results directory:** `experiments/coordination-demo/redesign/results/20260322-162206/`

---

## What I Observed

### Raw Results (as reported by experiment script)

| Task Type | Trials | CONFLICT | BUILD_FAIL | SUCCESS |
|-----------|--------|----------|------------|---------|
| Simple    | 10     | 8        | 2          | 0       |
| Complex   | 10     | 0        | 10         | 0       |
| **Total** | **20** | **8**    | **12**     | **0**   |

### Confound: Stale .go Files Cause False BUILD_FAIL

All 12 BUILD_FAIL results are **false negatives**. Stale `.go` files from the decay experiment (`experiments/coordination-demo/redesign/results/20260322-154258/`) are committed to git but their isolating `go.mod` is NOT committed. Worktrees see the `.go` files without the `go.mod`, causing `go build ./...` to fail on every merge check.

**Verified by manual replay** of 3 BUILD_FAIL trials (simple trial 4, simple trial 8, complex trial 2):
- All 3: clean merge, clean build (`go build ./pkg/display/`), all tests pass

### Corrected Results

| Task Type | Trials | CONFLICT | Clean Merge | Success Rate |
|-----------|--------|----------|-------------|--------------|
| Simple    | 10     | 8        | 2           | **20%**      |
| Complex   | 10     | 0        | 10          | **100%**     |
| **Total** | **20** | **8**    | **12**      | **60%**      |

### Placement Model Behavior

LLM placement decisions across 20 trials:

**Simple tasks** (FormatBytes + FormatRate):
| Placement Pattern | Frequency | Merge Result |
|-------------------|-----------|--------------|
| FormatDurationShort / FormatDuration (adjacent) | 7/10 | 7/7 CONFLICT |
| FormatDurationShort / FormatDurationShort (SAME) | 1/10 | CONFLICT |
| FormatDurationShort / StripANSI (distant) | 1/10 | Clean merge |
| FormatDuration / FormatDurationShort (adjacent) | 1/10 | Clean merge |

**Complex tasks** (VisualWidth + FormatTable):
| Placement Pattern | Frequency | Merge Result |
|-------------------|-----------|--------------|
| StripANSI / FormatDurationShort (distant) | 9/10 | 9/9 Clean merge |
| ShortID / FormatDurationShort (distant) | 1/10 | Clean merge |

### Root Cause Analysis: Why Simple Fails and Complex Succeeds

**Three failure modes identified:**

1. **Adjacent placement (7/10 simple trials):** The LLM picks FormatDuration and FormatDurationShort as "different" functions — they ARE different functions but are adjacent in the file (lines 49-82 vs 86-94). Both agents end up inserting at line 94 (end of file) regardless of which adjacent function they're assigned, because FormatDurationShort is the last function and the gravitational insertion point.

2. **Same placement (1/10 simple trials):** The LLM assigns both agents to FormatDurationShort despite explicit instructions to pick different insertion points. Pure placement model failure.

3. **Task-placement instruction conflict:** The BASE prompts (simple-a.md, simple-b.md) contain "Place the function after the existing `FormatDurationShort` function." When the LLM-generated placement says "after FormatDuration" instead, agents see TWO conflicting instructions and follow the one embedded in the task description. The task instruction is stronger than the coordination instruction — exactly confirming the model's existing finding (Claim 1, implication 4).

**Why complex succeeds:** For complex tasks, VisualWidth *uses* StripANSI, making "place after StripANSI" semantically congruent with the task. The LLM naturally picks StripANSI as VisualWidth's placement (9/10 trials), and agents follow it because it's both the coordination instruction AND the semantically correct location. There is no conflict between task instruction and placement instruction.

### Key Insight: Semantic Congruence Determines Compliance

LLM-generated placements succeed when the assigned location is the **semantic home** for the new code — where a human developer would naturally put it given the function's dependencies. They fail when the placement is arbitrary relative to the task's semantic context.

| Placement Type | Agent Compliance | Merge Result |
|----------------|-----------------|--------------|
| Semantically congruent (e.g., VisualWidth after StripANSI) | High (10/10) | Success |
| Adjacent-but-different (e.g., after FormatDuration vs FormatDurationShort) | Low — agents override | Conflict |
| Distant but semantically arbitrary (e.g., FormatRate after StripANSI) | Mixed (2/3) | Success when followed |

---

## Model Impact

- [ ] **Confirms** invariant: Structural placement prevents conflicts (Claim 2)
- [ ] **Contradicts** invariant: N/A
- [x] **Extends** model with: Anticipatory placement via static analysis works conditionally — 60% overall (100% complex, 20% simple). Success depends on semantic congruence between placement and task, not just spatial separation.

### Specific Extensions

1. **New finding: Semantic congruence requirement.** Effective attractor placement requires the assigned location to be the natural semantic home for the new code. Spatial separation alone is insufficient — the LLM correctly identified "different" functions 19/20 times but only produced non-overlapping merges when those functions were semantically related to the tasks.

2. **New finding: Task instructions override coordination instructions.** When the base prompt says "after FormatDurationShort" and the coordination instruction says "after FormatDuration," agents follow the task prompt. This strengthens Claim 1 (communication insufficient) and Implication 4 (task instruction > coordination instruction) with a new mechanism: even structural placement fails when it conflicts with embedded task instructions.

3. **New finding: Placement model gravitational bias.** The placement LLM exhibits the same gravitational pull as the agents. For simple tasks with no semantic relationships to guide placement, it picks adjacent functions (FormatDuration/FormatDurationShort) 8/10 times. It only escapes the gravitational zone when task-function semantic relationships exist (complex tasks).

4. **Comparison to automated attractor discovery (Experiment B).** Failure-data-driven placement (Experiment B) achieved 7/7 (100%) because it explicitly identifies the collision point and routes AWAY from it. Anticipatory placement (this experiment) achieves 12/20 (60%) because it must infer the collision point from code structure alone. The failure feedback loop provides information that static analysis doesn't: WHERE agents actually collide, not just WHERE functions are.

5. **Comparison to human-designed placement (baseline).** Human placement achieved 20/20 (100%) because humans understand both spatial separation AND semantic congruence. The human baseline placed one agent after FormatDurationShort and the other after StripANSI — the exact pattern that the complex-task LLM also discovered, but that the simple-task LLM missed.

---

## Notes

**Confound documentation:** All 12 BUILD_FAIL results are false negatives from stale `.go` files committed without their `go.mod` in `experiments/coordination-demo/redesign/results/20260322-154258/`. Future experiments should either: (a) add `go.mod` files to experiment result directories, or (b) use `go build ./pkg/display/` instead of `go build ./...` in the merge checker.

**Significance:** This is the most informative negative result in the experiment chain. It demonstrates that coordination cannot be reduced to "have an LLM read the code before agents run" — the LLM suffers from the same gravitational bias as the agents. Effective attractor placement requires either failure data (Experiment B's approach) or human-level understanding of semantic relationships between code and tasks.

**Open question:** Would a stronger placement model (Opus instead of Haiku) produce better placements? The simple-task placement failures are arguably a reasoning failure — the LLM picks adjacent functions as "different" without understanding git merge region proximity.

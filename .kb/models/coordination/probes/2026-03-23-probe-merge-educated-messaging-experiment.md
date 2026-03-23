# Probe: Git-Merge-Educated Messaging — Does Correcting the False Merge Model Change Coordination Outcomes?

**Model:** coordination
**Date:** 2026-03-23
**Status:** Complete
**claim:** COORD-06
**verdict:** extends

---

## Question

The Claim 6 scope probe identified a specific false model: agents believe same-point additions with different function names merge cleanly. The probe flagged "git-merge-aware messaging" as "the cheapest untested intervention." Does correcting the false merge model via explicit prompt education reduce the conflict rate below the messaging baseline of 20/20 CONFLICT?

**Falsification condition:** If conflict rate drops below 100%, the false merge model IS the core mechanism and Claim 1 narrows to "communication without correct merge models doesn't coordinate." If conflict rate stays at 100%, semantic-correctness bias dominates even with accurate merge knowledge.

---

## What I Tested

Ran a variant of the messaging condition with explicit git merge education added to the prompt.

**Experimental design:**
- Same model: claude-haiku-4-5-20251001
- Same tasks: simple (FormatBytes + FormatRate) and complex (VisualWidth + FormatTable)
- Same N: 10 per task type (20 total)
- Same messaging protocol: shared directory, plan files, coordination instructions
- **Only change:** Added ~200 words of git merge education explaining:
  - Git merges at the TEXT level, not semantic level
  - Two insertions at the same line position WILL conflict regardless of content
  - Concrete example: "Branch A adds FormatBytes after FormatDurationShort, Branch B adds FormatRate after FormatDurationShort -> CONFLICT"
  - Explicit instruction to choose DIFFERENT insertion points, at least 3 lines apart

**Baseline:** messaging condition from 20260310-174045: 20/20 CONFLICT

```bash
cd experiments/coordination-demo/redesign
./run-merge-educated.sh --trials 10
# Results: experiments/coordination-demo/redesign/results/merge-educated-20260323-093342/
```

---

## What I Observed

### Finding 1: Merge education significantly reduced conflicts (100% -> 70%)

| Condition | Simple | Complex | Total | Success Rate |
|-----------|--------|---------|-------|-------------|
| Messaging (baseline) | 0/10 | 0/10 | 0/20 | 0% |
| Merge-educated | 4/10 | 2/10 | 6/20 | 30% |

- Fisher's exact test: p = 0.010 (one-sided), p = 0.020 (two-sided)
- 95% Wilson CI for merge-educated success rate: [14.5%, 51.9%]
- All 40 agents scored 6/6 individually (Claim 3 continues to hold)

### Finding 2: In successful trials, agents explicitly used the merge education

All 6 successful trials showed agents:
1. Explicitly citing merge education in their coordination reasoning
2. Using "binary opposition" — one agent taking early position (after StripANSI), the other taking late position (after FormatDurationShort)
3. Reading the other agent's plan and actively repositioning

Example (Trial 2, Agent B plan): "Agent A is inserting at lines 95-119 in display.go (after FormatDurationShort)... This plan inserts at lines 47-70... No overlap - clean merge guaranteed"

Example (Trial 5, Agent B plan): "CONFLICT DETECTED... RESOLUTION: Agent B will insert at a DIFFERENT location to avoid merge conflicts."

### Finding 3: Conflict trials reveal 5 distinct failure patterns

Analysis of plan files from all 14 conflict trials reveals:

**Pattern 1: Mutual convergence on "safe" position (4 trials: Simple 1,7; Complex 1,3)**
Both agents independently chose the same "safe" insertion point (after StripANSI), each believing the other was at the "default" position (after FormatDurationShort). The merge education gave them a new attractor that both converged on simultaneously.

**Pattern 2: Misreading each other's plans (7 trials: Simple 1,4,7; Complex 1,3,5,7)**
Agents paraphrased rather than quoted exact coordinates from the other's plan, introducing drift. Example: Plan-B says "Agent A is inserting at lines 96-130" when Agent A's plan actually says "lines 46-75." The narrative summary replaces the precise data.

**Pattern 3: Self-contradictory plans (1 trial: Simple 4)**
Agent acknowledged the coordination protocol but stated conflicting insertion points within the same message: "I am moving to lines 47-80" immediately followed by "inserting after line 94."

**Pattern 4: Overestimated separation (3 trials: Complex 6,7,8)**
Agents calculated line gaps incorrectly. Example (Complex 7): Agent B claims "12+ lines away from Plan A at line 96" while planning lines 83-130, which overlaps directly with Agent A's lines 96-145.

**Pattern 5: Sequential messaging lag (3 trials: Simple 10; Complex 9,10)**
Agent A revised its plan after reading Agent B's plan, but Agent B already committed to the original coordinates and didn't re-read. The single-round messaging protocol doesn't support revision cycles.

### Finding 4: Task complexity matters — simple tasks coordinate better

Simple tasks (40% success) outperformed complex tasks (20% success). Complex tasks produce larger code insertions (~60-80 lines vs ~30-40 lines for simple), which:
- Reduce available separation gaps between viable insertion points
- Make line-range estimation errors more consequential
- Increase the chance of overestimated separation (Pattern 4)

### Finding 5: The file structure constrains coordination capacity

The display.go file offers only 2-3 viable insertion points:
1. After StripANSI (early, ~line 46)
2. Between FormatDuration and FormatDurationShort (middle, ~line 83)
3. After FormatDurationShort (late, ~line 95)

With only ~50 lines of usable insertion space and agents producing 30-80 lines each, the file geometry makes conflict avoidance structurally difficult even with perfect coordination knowledge.

---

## Model Impact

- [x] **Extends** Claim 1 with quantified effect: merge education reduces conflicts from 100% to 70% (p=0.02), but does not achieve reliable coordination
- [x] **Extends** Claim 5 mechanism: the false merge model is one of at least 5 distinct failure mechanisms in messaging-based coordination
- [x] **Extends** Claim 6 scope: identifies specific failure taxonomy for messaging (mutual convergence, plan misreading, self-contradiction, gap overestimation, messaging lag)

**What this means for the model:**

1. **Claim 1 is confirmed but with important nuance.** Communication is insufficient, but not because agents can't coordinate at all. The false merge model was a contributing factor (~30% of outcomes). The remaining 70% conflict rate is driven by structural constraints (limited insertion points), communication quality failures (misreading, paraphrasing), and single-round messaging limitations.

2. **The model should add a "partial intervention" result class.** The current model presents results as binary (100% CONFLICT or 100% SUCCESS). This experiment shows a middle ground that reveals the mechanism structure more clearly than either extreme.

3. **Placement still dominates.** Placement achieved 20/20 SUCCESS vs merge-educated messaging's 6/20 SUCCESS. Structural constraints outperform communicative coordination by a factor of >3x.

4. **New open question: Would multi-round negotiation close the gap?** Pattern 5 (sequential messaging lag) is explicitly addressable. Patterns 1 and 2 (mutual convergence, plan misreading) might improve with forced re-read cycles.

---

## Notes

This directly tested the mechanism identified in probe `2026-03-22-probe-claim6-messaging-scope.md` Finding 6, item 1. The result is intermediate — neither the pure "false merge model" hypothesis (which predicted high success) nor the pure "semantic gravity" hypothesis (which predicted no change) was fully validated.

The real finding is more interesting than either extreme: **messaging-based coordination has a ceiling determined by file geometry and communication quality, and that ceiling (~30%) is well below what structural placement achieves (100%).** The false merge model suppressed coordination from ~30% to 0%, making it look like messaging was worthless when it was actually partially effective but structurally limited.

**Remaining untested variants from the Claim 6 probe:**
- Multi-round negotiation (would address Pattern 5, possibly Patterns 1-2)
- Tool-augmented messaging (would eliminate all 5 failure patterns but converts to a different coordination mechanism)

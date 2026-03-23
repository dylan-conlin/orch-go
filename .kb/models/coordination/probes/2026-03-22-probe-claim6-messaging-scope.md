# Probe: Claim 6 — Messaging Frameworks Scope Analysis

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** COORD-06
**verdict:** scopes

---

## Question

Implication 1 states that "messaging-based coordination did not produce coordination outcomes" and extends this to claim that CrewAI, LangGraph, Claude Agent SDK, OpenAI Agents SDK, and similar frameworks that assume messaging solves coordination showed failures. The epistemic audit flagged this as overclaimed: the experimental evidence comes from one experiment family, and the external validation is literature mapping, not reproduction. Does the evidence support "messaging is fundamentally flawed" or only "messaging fails for a specific coordination challenge"?

**Falsification condition:** Identify a messaging protocol or task structure where agent-to-agent messaging COULD produce coordination (>0% conflict reduction), OR identify the specific failure mechanism that limits the claim's scope.

---

## What I Tested

Deep analysis of messaging plan artifacts from 20 messaging trials + 20 gate trials:

```bash
# Read all 20 messaging plan files (10 simple + 10 complex)
cat experiments/coordination-demo/redesign/results/20260310-174045/messaging/*/trial-*/messages/plan-*.txt

# Compared agent insertion points across all messaging trials
grep '^@@' experiments/coordination-demo/redesign/results/20260310-174045/messaging/*/trial-*/agent-*/full_diff.txt

# Read gate experiment outputs to check self-verification behavior
tail -30 experiments/coordination-demo/redesign/results/20260322-124035/gate/*/trial-*/agent-*/stdout.log
```

---

## What I Observed

### Finding 1: Communication succeeded — agents exchanged accurate plans

In 18/20 messaging trials, BOTH agents wrote implementation plans. Plans were accurate:
- Correctly stated which files they'd modify
- Correctly identified the other agent's task
- Correctly listed function names they'd add

Communication quality was not the bottleneck. The messaging infrastructure worked as designed.

### Finding 2: The specific failure mechanism is a false merge-conflict model

Every agent that read the other's plan reached the same incorrect conclusion. Examples:

**Trial 1, Agent A (simple):** "Both functions placed after FormatDurationShort (same insertion point) — Can be merged by inserting both after FormatDurationShort"

**Trial 2, Agent B (simple):** "No functional conflicts expected — Insertion point chosen: after FormatDurationShort to avoid merge conflicts"

**Trial 2, Agent A (complex):** "If FormatTable is added after same location, no conflict — Both functions are independent and don't interact"

Agents believe that two additions at the same git hunk position will merge cleanly as long as the function names are different. This is **factually wrong** for git's merge algorithm: two agents adding 30-50 lines at `@@ -92 @@` produces a textual conflict regardless of content.

### Finding 3: Gate condition confirms — even explicit self-checking doesn't fix the false model

The gate experiment (20 trials) gave agents an explicit instruction: "Review your insertion points. Predict the other agent's insertion points. Check for overlap. If conflicting, REVISE."

Results: 20/20 CONFLICT. Agent outputs claimed successful coordination:
- "Conflict check performed — no merge conflicts predicted"
- "Coordinated insertion point with FormatBytes agent to ensure clean merge"
- "conflict coordination with the parallel FormatRate task is documented"

The agents performed the check, concluded "no conflict," and didn't move. The self-check is subject to the same false model as the original decision.

### Finding 4: The false model is specific to additive/gravitational tasks

The messaging failure has TWO required components:
1. **Gravitational convergence** — agents independently choose the same insertion point because it's semantically correct
2. **False merge model** — agents believe same-point additions merge cleanly

For tasks WITHOUT gravitational convergence (e.g., two agents modifying different existing functions), component #1 doesn't apply. Agents would naturally produce non-overlapping diffs, and messaging about "what I'm modifying" would provide useful information.

For tasks WITH gravitational convergence but WITH git-merge education (e.g., "WARNING: Adding code after the same function produces git CONFLICT even with different function names"), component #2 might be addressable. This is untested.

### Finding 5: Literature review is correlation, not causation

The model cites 6 external frameworks where gate-based coordination correlates with failure and attractor-based correlates with success. However:
- CrewAI's failures may stem from manager LLM quality, not messaging per se
- LangGraph's failures may stem from graph topology design, not conditional routing per se
- The literature mapping identifies which mechanism TYPE each framework uses, not whether messaging caused the failure

The correlation is real and informative, but "messaging frameworks are fundamentally flawed" extrapolates from correlation to causation across systems the model has not tested.

### Finding 6: At least three untested messaging variants could produce different results

1. **Git-merge-aware messaging** — include merge-mechanics education in the coordination prompt: "Two additions at the same hunk position WILL conflict in git." This addresses the false merge model directly.

2. **Multi-round negotiation** — allow agents to revise plans iteratively: Agent A writes plan → Agent B reads and proposes non-overlapping plan → Agent A confirms. The tested protocol was single-round with no revision cycle.

3. **Tool-augmented messaging** — give agents a `check-merge-conflict` tool that simulates the merge before commit. This converts the false-model problem into a tool problem — agents don't need to understand merge mechanics if a tool tells them "this will conflict."

None of these are tested. The claim that messaging "does not produce coordination" is specific to the tested protocol (single-round plan exchange without merge-mechanics education).

---

## Model Impact

- [x] **Scopes** claim: "messaging-based coordination did not produce coordination outcomes" → "single-round messaging without merge-mechanics education did not produce coordination outcomes in additive same-file tasks with gravitational insertion point convergence"

**Specific changes needed in model.md:**

1. Implication 1 should qualify: messaging failed "in the tested scenarios and protocol" — not "fundamentally"
2. The word "fundamentally" in "Multi-agent messaging frameworks are fundamentally flawed" should be removed per evidence-tier standard (requires "validated" tier, but evidence is "observed" tier)
3. Add to Open Questions: "Would git-merge-aware messaging or multi-round negotiation produce different results?"
4. The mechanism: agents have a false model of merge conflicts. This is the finding — not that communication can't coordinate, but that communication can't coordinate when agents have wrong models of the conflict space.

**Evidence tier:** Remains "Working-hypothesis" for the general claim about messaging frameworks. The specific finding (agents have false merge models → messaging fails for gravitational tasks) is "Observed" with N=40 (20 messaging + 20 gate).

---

## Notes

The strongest finding from this probe is DIAGNOSTIC, not merely scoping: the failure mechanism is a false model of git merge mechanics, not a failure of communication per se. This has practical implications:

1. If the false model is the root cause, messaging could potentially work with model correction
2. The claim should be about the false model, not about messaging as a category
3. This explains WHY the gate also failed — the self-check uses the same false model

The open question is whether correcting the false model is sufficient, or whether semantic gravity (preferring the "correct" location over the "non-conflicting" location) would still dominate even with accurate merge knowledge.

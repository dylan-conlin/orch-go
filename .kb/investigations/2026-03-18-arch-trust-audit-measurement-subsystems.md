# Trust Audit: 8 Non-HIGH Measurement Subsystems

**Date:** 2026-03-18
**Type:** Architecture decision
**Status:** Complete
**Beads:** orch-go-nlvs7

---

## Context

The phase transition probe (orch-go-58923) mapped 10 measurement subsystems: 2 HIGH trust (build, git existence), 2 MEDIUM (test evidence, orphan rate), 6 LOW/FALSE (learning store, ground truth adjustment, merge rate, detector outcomes, decision audit, model contradictions).

This audit determines for each non-HIGH subsystem: GROUND (connect to external truth), SIMPLIFY (reduce to what's measurable), or DELETE (remove entirely).

**Governing principle:** A metric you can't trust is worse than no metric. Fewer trusted metrics > more untrusted metrics. Break the discover-gap → build-detector → discover-detector-isn't-grounded → build-meta-detector loop.

---

## Verdicts

### 1. Test Evidence Gate — SIMPLIFY

**Current trust:** MEDIUM
**What it does:** Pattern-matches beads comments for test output (e.g., `ok  pkg/foo  0.12s`, `15 passed, 0 failed`). Rejects vague claims like "all tests pass."
**What's good:** The false-positive rejection is genuinely useful — it distinguishes "I ran tests" from "tests probably pass."
**What's broken:** It can't verify the output was actually produced during this session. An agent could paste test output from a different run.

**Verdict: SIMPLIFY — keep pattern matching, drop the pretense of verification.**
- Rename internally from "verification" to "evidence check" — it's a necessary-but-not-sufficient signal
- Keep the false-positive patterns (they catch real sloppiness)
- Stop treating pass/fail as a quality gate; treat it as a flag for human attention
- The signal is: "agent claims to have run tests with specific output" — that's worth having at MEDIUM trust

**No code changes needed.** The current implementation is already doing the right thing. The fix is conceptual: stop calling this "verification" in orient/dashboard displays, call it "test evidence present: yes/no."

---

### 2. Orphan Rate — SIMPLIFY

**Current trust:** MEDIUM
**What it does:** Counts investigations in `.kb/investigations/` not referenced by other KB files. Currently 91.8% (1170/1275).
**What's good:** The count itself is accurate.
**What's broken:** 91.8% orphan rate is the natural baseline, not a problem signal. Most investigations are standalone artifacts. The metric creates false urgency.

**Verdict: SIMPLIFY — measure delta, not absolute.**
- Stop displaying absolute orphan rate (it's noise at 90%+)
- Instead: track **new orphans per session** — "this session produced 3 investigations, 0 linked." That's actionable.
- The stratified breakdown (`--stratified`) already exists and is more useful than the headline number
- Consider: delete the orphan rate from orient display entirely. The `kb orphans --stratified` command is available when needed.

**Minimal code change:** Remove orphan rate from `orch orient` display. Keep `kb orphans` command for on-demand use.

---

### 3. Learning Store (Skill Success Rates) — SIMPLIFY

**Current trust:** LOW
**What it does:** Aggregates events.jsonl into per-skill success rates. Feeds daemon allocation scoring (±20% modulation on priority).
**What's good:** The events are factual (spawned, completed, abandoned are real). The sample-size blending (blend toward 0.5 below 10 samples) is correctly conservative.
**What's broken:** "Success" = "agent said Phase: Complete with outcome=success." This is self-reported. The ±20% modulation is small enough that it rarely changes spawn order (priority dominates).

**Verdict: SIMPLIFY — keep for allocation but cap its influence and label honestly.**
- The ±20% modulation is already appropriately small — priority dominates, which is correct
- Rename from "success rate" to "completion rate" everywhere — that's what it actually measures
- Keep the compliance auto-downgrade (it's conservative: only downgrades, never upgrades, requires 80%+ over 10+ samples)
- Remove from orient display as a "quality" metric. It's an operational metric (throughput by skill), not a quality metric.

**No structural code changes.** Rename `SuccessRate` → `CompletionRate` in display/orient output. The allocation math stays the same.

---

### 4. Ground Truth Adjustment — DELETE

**Current trust:** FALSE (but partially mitigated)
**What it does:** Blends self-reported success with rework rate at 70/30. Formula: `0.7 * selfReported + 0.3 * (1 - reworkRate)`.
**What's good:** The `hasReworkData := sl.ReworkCount > 0` guard (allocation.go:140) already prevents the worst case — zero reworks no longer inflates the score.
**What's broken:** With zero reworks (the actual state), `hasReworkData=false` and the function returns `selfReported` unchanged. The entire ground truth adjustment code path is dead code. It exists for a rework channel that has never been used (0 reworks across 817 completions).

**Verdict: DELETE — remove the ground truth adjustment function.**
- `GroundTruthAdjustedRate()` is dead code. The rework channel has 0 data points across 817 completions.
- Replace `lookupSuccessRate()` with direct `BlendedSuccessRate(sl.SuccessRate, sampleSize)` — which is what it already returns since `hasReworkData` is always false.
- Delete the `GroundTruthWeight` constant.
- If/when rework data actually accumulates, rebuild the adjustment from scratch with real data to calibrate the weight.
- **Remove "ground truth adjusted" from orient display.** Displaying an adjusted rate that equals the unadjusted rate is misleading.

**Code change:** ~20 lines removed from `pkg/daemon/allocation.go`. Simplifies to direct blended rate.

---

### 5. Merge Rate — DELETE

**Current trust:** FALSE
**What it does:** Counts unique beads IDs in git commits / completions. Shows "Merged: 596 (100%)."
**What's good:** The git parsing is correct.
**What's broken:** In a single-branch workflow (all commits to main), merge rate = completion rate. It measures "did the agent commit" which is identical to "did the agent complete." The metric is tautological — it cannot distinguish good work from bad work.

**Verdict: DELETE — remove merge rate from orient display and throughput struct.**
- The metric is structurally unable to provide signal in a single-branch flow
- Displaying "100% merge rate" creates false confidence
- If the project adopts PR workflow, this could be rebuilt — but building for a hypothetical future violates the "no speculative infrastructure" principle
- **Keep `NetLinesAdded`** — net code impact is grounded in git and provides actual signal (volume of change per session)

**Code change:** Remove `MergedCount`, `MergeRate` from `Throughput` struct. Keep `NetLinesAdded`/`NetLinesRemoved`. Remove merge rate display from `formatThroughput()`.

---

### 6. Detector Outcomes — SIMPLIFY

**Current trust:** LOW
**What it does:** Tracks whether daemon-created trigger issues get completed or abandoned. Computes "useful rate" per detector. Has budget adjustment logic (halve at <30%, disable at <10%).
**What's good:** The structure is sound — tracking outcomes of automated decisions is the right idea. The budget adjustment thresholds are conservative. The `MinResolvedForPenalty=5` guard prevents overreaction.
**What's broken:** "Completed" = self-reported by the agent working the issue. And the budget adjustment is **not yet wired** — `RunPeriodicTriggerScan` uses static budget, not `AdjustedBudget()`. The code exists but doesn't execute.

**Verdict: SIMPLIFY — wire the budget adjustment, rename the metric.**
- Wire `ComputeDetectorOutcomes()` → `AdjustedBudget()` into `RunPeriodicTriggerScan`. This closes the feedback loop that's currently open.
- Rename "useful rate" to "resolution rate" — what it actually measures is "was the issue closed as completed rather than abandoned."
- This is still self-reported, but self-reported abandonment is a real signal — if agents consistently abandon a detector's issues, the detector is creating noise.
- The budget halving/disabling mechanism is the right response: reduce noise, don't chase precision.

**Code change:** ~10 lines in `pkg/daemon/trigger.go` to call `AdjustedBudget()` per detector.

---

### 7. Decision Audit — DELETE

**Current trust:** LOW
**What it does:** Scans `.kb/decisions/` for Accepted decisions, checks if referenced code files exist.
**What's good:** It finds obviously broken references (deleted files).
**What's broken:** File existence ≠ implementation correctness. A decision that says "use pattern X in pkg/foo.go" passes if `pkg/foo.go` exists, regardless of whether pattern X is implemented. 54.4% of decisions have "missing references" but many of these are decisions that never referenced specific files (they're architectural principles, not file-specific directives).

**Verdict: DELETE — remove the automated decision audit.**
- File existence checking produces 54.4% "failures" that are structurally false positives (decisions without file refs)
- Checking file existence for decisions WITH refs is trivially done by `git log` when the decision is consulted
- The metric creates work (investigating "missing references") without catching real problems (decisions that were ignored)
- If decision compliance matters, the right mechanism is: when consulting a decision during a task, check if current code follows it. That's point-of-consumption detection, not periodic sweep.

**Code change:** Delete `pkg/kbmetrics/decision_audit.go`. Remove from `kb audit decisions` command. Remove from orient/dashboard if displayed.

---

### 8. Model Contradictions Detector — SIMPLIFY

**Current trust:** LOW
**What it does:** Scans probe files for contradiction keywords ("contradict", "refute", "not true", etc.), checks if model was updated after probe.
**What's good:** The concept is right — unmerged contradictions are a real problem (the probe-to-model merge requirement exists for this reason).
**What's broken:** Keyword matching has unknown precision. "Contradicts" in a sentence like "this does not contradict the model" would be a false positive. No measurement of detection accuracy.

**Verdict: SIMPLIFY — keep but measure precision.**
- The detector is cheap to run and the failure mode (false positive → unnecessary investigation issue) is low-cost
- Add: count false positives via the detector outcomes mechanism (already built, just needs wiring)
- The real enforcement is the probe-to-model merge requirement in worker-base skill, not this detector. This is a backstop.
- Reduce priority from P2 to P3 — contradiction detection is advisory, not urgent

**Code change:** Change priority from 2 to 3 in `trigger_detectors_phase2.go`. Ensure detector outcomes are wired for this detector.

---

## Summary Table

| # | Subsystem | Current Trust | Verdict | Key Change |
|---|-----------|--------------|---------|------------|
| 1 | Test evidence | MEDIUM | SIMPLIFY | Rename to "evidence check," keep as-is |
| 2 | Orphan rate | MEDIUM | SIMPLIFY | Show delta per session, drop absolute rate from orient |
| 3 | Learning store | LOW | SIMPLIFY | Rename "success rate" → "completion rate," keep allocation math |
| 4 | Ground truth adjustment | FALSE | **DELETE** | Dead code (0 reworks ever). Remove function + constant. |
| 5 | Merge rate | FALSE | **DELETE** | Tautological in single-branch flow. Remove from orient. |
| 6 | Detector outcomes | LOW | SIMPLIFY | Wire the budget adjustment that already exists but isn't called |
| 7 | Decision audit | LOW | **DELETE** | 54.4% false positive rate. File existence ≠ correctness. |
| 8 | Model contradictions | LOW | SIMPLIFY | Lower priority P2→P3, wire outcome tracking |

**Net result:** 3 DELETE, 5 SIMPLIFY, 0 GROUND.

None of these subsystems can be grounded in external truth without architectural changes (PR workflow, human sampling, external validation). The honest move is to delete the ones pretending to be grounded and simplify the rest to honestly describe what they measure.

---

## What Remains After Cleanup

**Trusted metrics (HIGH):**
- Build passes (Gate 10) — unfakeable
- Git commit existence — verifiable fact
- Net lines added/removed — git-grounded

**Honest metrics (MEDIUM, clearly labeled):**
- Test evidence present (yes/no) — "agent claims tests ran with specific output"
- Completion rate by skill — "what fraction of spawns self-report completion"
- New orphans per session — "how many unlinked investigations this session created"

**Backstop detectors (LOW, budget-limited):**
- Model contradictions — keyword-based, P3, outcome-tracked
- Detector outcomes → budget adjustment — self-tuning noise reduction

**Deleted:**
- Ground truth adjustment (dead code)
- Merge rate (tautological)
- Decision audit (54% false positive)

---

## Implementation Priority

1. **DELETE ground truth adjustment** — removes dead code, simplifies allocation.go
2. **DELETE merge rate from orient** — removes most visible false confidence signal
3. **Wire detector outcome budgets** — closes the open feedback loop (code exists, just unwired)
4. **DELETE decision audit** — removes 54% false positive noise source
5. **Rename success→completion in displays** — honest labeling, no math changes
6. **Lower model contradictions priority** — P2→P3, one line change
7. **Orphan rate delta** — optional, lower priority

Items 1-4 are straightforward deletions/wirings. Items 5-7 are labeling changes.

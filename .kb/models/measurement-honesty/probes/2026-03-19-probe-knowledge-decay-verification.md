# Probe: Knowledge Decay Verification — Measurement Honesty

**Date:** 2026-03-19
**Triggered by:** Knowledge decay detector (999d since last probe — model has never been probed since creation on 2026-03-19)
**Method:** Grep codebase for each concrete claim in model, verify current state matches model assertions

---

## Claims Verified

### 1. `GroundTruthAdjustedRate()` deleted — CONFIRMED

No matches for `GroundTruthAdjustedRate` anywhere in codebase. The model's claim that it was deleted per trust audit is accurate. `BlendedSuccessRate()` exists at `pkg/daemon/allocation.go:92` as the stated replacement.

**Evidence:** `rg GroundTruthAdjustedRate` → 0 results

### 2. `BlendedSuccessRate()` uses sample-size weighting — CONFIRMED

The replacement function blends observed rate with default (0.5) based on sample count, exactly as described. Uses `MinSamplesForFullWeight` to prevent single-sample dominance.

**Evidence:** `pkg/daemon/allocation.go:92-102`

### 3. Decision audit rebuilt with type-aware validation — CONFIRMED

`validateArchitectural()` function exists in `pkg/kbmetrics/decision_audit.go:270`, checks gates/hooks/tests/CLAUDE.md reflection. Bigram matching implemented at line 429-436. This matches the model's description of the v2 rebuild.

**Evidence:** `pkg/kbmetrics/decision_audit.go:270-272` (validateArchitectural), `429-436` (bigram search)

### 4. Merge rate removed from orient display — CONFIRMED (with nuance)

Not displayed as standalone metric. Still referenced in `orient_cmd.go:754` comment and used internally by `computeDivergenceAlerts()`, but this is architectural — the divergence system uses it as an input, not a displayed metric. The model's claim that it was "deleted from orient display" is accurate.

**Evidence:** `orient_cmd.go:754` (comment only), no `MergeRate` display formatting found

### 5. Orphan rate replaced with session-scoped delta — CONFIRMED

`enrichReflectWithSessionOrphans()` at `orient_cmd.go:539` computes session-scoped orphan counts using previous session date as cutoff. `kbmetrics.ComputeSessionOrphans()` called at line 560. Model's claim about replacing absolute rate with actionable delta is accurate.

**Evidence:** `orient_cmd.go:134-135`, `539-567`

### 6. "Success rate" relabeled to "completion rate" — PARTIALLY CONFIRMED / EXTENDS MODEL

The orient display uses "completion rate" (`orient_cmd.go:762-764`). However, the allocation code still uses `SuccessRate` as field name (`allocation.go:35`, `allocation.go:15-27`), `SkillSuccessRate` in scored results, and the formula still treats it as a quality proxy via `SuccessRateWeight` modulation (±20%).

**This is the exact failure mode the model predicts** in "Why This Fails" §3: "Relabeling must propagate to formulas, not just displays." The display was relabeled, but the allocation formula still treats completion rate as a quality signal through the `SuccessRateWeight` multiplier. The variable names reinforce the old semantics.

---

## Overall Verdict

**Model is current and accurate.** All 6 concrete claims verified against codebase. The taxonomy (false confidence / noisy signal / honest-but-misnamed) and the remediation instances are all confirmed by current code.

**One active finding:** Claim #6 reveals that the model's own predicted failure mode ("relabeling is insufficient when the name is load-bearing") is currently occurring in the allocation code. The model correctly identifies this as a risk but doesn't note that it's an active instance in the codebase — the model is more accurate than it knows.

---

## Recommendations

1. **Minor model update:** Add a note to the "Honest-but-Misnamed" instances table that the `SuccessRate` field names in `pkg/daemon/allocation.go` still use the old semantics, making this a live example of the §3 failure mode.
2. **No structural changes needed.** Model taxonomy, diagnostic protocol, and critical invariants are all still accurate.
3. **Consider:** Renaming `SuccessRate` → `CompletionRate` in allocation structs and learning store fields to complete the relabeling propagation.

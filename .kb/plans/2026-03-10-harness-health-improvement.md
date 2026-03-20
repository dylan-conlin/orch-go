## Summary (D.E.K.N.)

**Delta:** Raise harness health score from 37/100 (F) to 65+ (C) by reducing bloated files and calibrating score thresholds.

**Evidence:** `orch health` shows 37/100 with three zeroed dimensions (accretion control 0/20, hotspot control 0/20, bloat % 1.5/20). 50 files >800 lines but 24 are test files. Score formula treats tests = source. Accretion trajectory probe (Mar 8) confirmed gates don't yet bend the curve.

**Knowledge:** The score is 37 because two dimensions saturate at aggressive thresholds (bloated ≥20 → 0pts, hotspots ≥15 → 0pts). Two levers: (1) extract real bloat, (2) calibrate formula so test files and codebase scale don't distort the signal.

**Next:** Phase 1 — calibrate score formula (architect), then Phase 2 — extract top bloated source files.

---

# Plan: Harness Health Improvement

**Date:** 2026-03-10
**Status:** Complete (closed 2026-03-19)
**Owner:** Dylan
**Closure Reason:** Health score framing superseded by measurement-honesty model. The "raise score to 65" objective optimized for a metric whose validity was questioned — the measurement-honesty model reframed governance metrics around honest signal rather than score targets. Calibrating thresholds to hit a number is the opposite of honest measurement.

**Extracted-From:** `orch health` analysis + `.kb/models/harness-engineering/probes/2026-03-08-probe-30-day-accretion-trajectory-gate-effectiveness.md`

---

## Objective

Move harness health score from 37/100 (F) to 65+ (C) within 2 weeks. The score should reflect genuine structural improvement, not just threshold gaming — so calibration and extraction happen together.

---

## Substrate Consulted

- **Models:** harness-engineering (accretion thermodynamics, hard/soft taxonomy), extract-patterns (extraction as temporary entropy reduction)
- **Decisions:** Three-layer hotspot enforcement (2026-02-26)
- **Guides:** code-extraction-patterns.md
- **Constraints:** Pre-commit accretion gate now wired (as of today). Completion accretion gate exempts pre-existing bloat.

---

## Decision Points

### Decision 1: Should test files count the same as source files in bloat metrics?

**Context:** 24 of 50 files >800 lines are `*_test.go`. Test files have different accretion dynamics — they grow linearly with test cases, don't suffer from cross-cutting concerns, and don't create coordination failures. Counting them equally inflates the score's penalty.

**Options:**
- **A: Exclude test files from bloat count** — Only count source files. Immediately drops bloated count from 50 to 26. Pros: honest signal. Cons: test files CAN get unwieldy.
- **B: Weight test files at 0.5x** — Count them but reduce impact. Pros: acknowledges both concerns. Cons: adds complexity.
- **C: Raise test threshold to 2000 lines** — Different threshold, not different weight. Pros: simple. Cons: 2000 is arbitrary.

**Recommendation:** C — separate threshold for test files (2000 lines). Tests naturally grow longer, and 800 lines of tests isn't the same signal as 800 lines of source. Simple to implement, honest about the difference.

**Status:** Open

### Decision 2: Are the saturation thresholds appropriate for this codebase?

**Context:** Hotspot control saturates at 15, accretion at 20 bloated files. With 200+ source files, the thresholds may be too tight — a 10% bloat rate (common in active codebases) permanently zeros the score.

**Options:**
- **A: Scale thresholds to codebase size** — bloated threshold = 10% of source files. Pros: grows with project. Cons: can mask real problems.
- **B: Raise fixed thresholds** — hotspots to 30, bloated to 30. Pros: simple. Cons: less ambitious.
- **C: Keep thresholds, improve the code** — current thresholds represent aspirational targets. Pros: drives extraction. Cons: score stays F for a long time, losing signal value.

**Recommendation:** A with a floor — threshold = max(20, 10% of source files) for bloated, max(15, 5% of source files) for hotspots. This keeps pressure on small projects but doesn't permanently zero the metric for large ones.

**Status:** Open

---

## Phases

### Phase 1: Score Calibration (architect)

**Goal:** Fix the formula so it gives an honest signal — penalizes real bloat without saturating on test files or codebase scale.
**Deliverables:**
- Test file threshold separation (2000 vs 800)
- Codebase-scaled saturation thresholds
- Updated `ComputeHealthScore` in `pkg/health/health.go`
- Before/after score comparison
**Exit criteria:** `orch health` shows a score that would move meaningfully (±5 points) when 3 source files are extracted.
**Depends on:** Decisions 1 and 2

### Phase 2: Top 10 Source File Extractions

**Goal:** Extract the 10 largest non-test source files below 800 lines each.
**Target files (current lines):**
1. `cmd/orch/clean_cmd.go` (1270)
2. `pkg/tmux/tmux.go` (1201)
3. `cmd/orch/spawn_cmd.go` (1171)
4. `pkg/account/account.go` (1162)
5. `cmd/orch/kb.go` (1138)
6. `cmd/orch/serve_beads.go` (1124)
7. `pkg/beads/client.go` (1115)
8. `cmd/orch/serve_system.go` (1084)
9. `pkg/spawn/kbcontext.go` (1072)
10. `cmd/orch/session.go` (1055)

**Deliverables:** Each file under 800 lines with extracted code in cohesive new files.
**Exit criteria:** Bloated source file count drops from 26 to ≤16. `orch health` score improves by 10+ points.
**Depends on:** Phase 1 (so improvements register in score)

### Phase 3: Orphan Triage

**Goal:** Clear the 18 orphaned issues dragging down the issue health alerts.
**Deliverables:** Each issue either closed (stale/wontfix), re-prioritized, or actively worked.
**Exit criteria:** Orphaned issues < 5.
**Depends on:** Nothing (can parallel with Phase 2)

### Phase 4: Measure and Probe

**Goal:** Validate that the score reflects real structural improvement.
**Deliverables:** Probe comparing pre-plan and post-plan health scores, accretion velocity, and hotspot count.
**Exit criteria:** Score ≥65 (C grade) and accretion velocity has measurably decreased.
**Depends on:** Phases 1-3

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Test file thresholds | Accretion trajectory probe data, file inventory | Yes |
| Saturation scaling | Health score formula, codebase metrics | Yes |

**Overall readiness:** Ready to execute — both decisions can be resolved with data in hand.

---

## Structured Uncertainty

**What's tested:**
- ✅ 24 of 50 bloated files are tests (verified via wc -l)
- ✅ Score formula zeroes at 20 bloated / 15 hotspots (read from health.go)
- ✅ Pre-commit gate now wired (completed today, orch-go-t7te8)
- ✅ Extraction is temporary without routing attractors (harness-engineering model)

**What's untested:**
- ⚠️ Whether extraction of 10 files will stay below 800 (re-accretion risk)
- ⚠️ Whether hotspot count drops proportionally to file size reduction
- ⚠️ Whether the pre-commit gate (newly wired) prevents re-accretion

**What would change this plan:**
- If hotspot count is driven primarily by fix-density and investigation clusters (not file size), extraction alone won't move it
- If re-accretion happens within 2 weeks of extraction, the plan shifts to routing attractor design

---

## Success Criteria

- [ ] Health score ≥65 (C grade)
- [ ] Bloated source files ≤16 (down from 26)
- [ ] No CRITICAL hotspots (>1500 lines)
- [ ] Orphaned issues <5
- [ ] Score formula honestly reflects codebase health (not gamed by threshold manipulation)

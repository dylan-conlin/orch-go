# Probe: Intervention Effectiveness Audit — What Actually Reduced Accretion?

**Model:** knowledge-accretion
**Date:** 2026-03-20
**Status:** Complete
**claim:** KA-05 (substrate-independent interventions: attractors, gates, entropy measurement)
**verdict:** extends

---

## Question

The knowledge-accretion model proposes many interventions (gates, attractors, metrics, entropy measures) but has almost no effectiveness data. For each proposed intervention: Is it implemented? Is there measurement data? Did it demonstrably reduce the thing it targets?

---

## What I Tested

Systematic audit of every intervention proposed in the model, verified against:

```bash
# Implementation check — grep for each intervention in code
grep -rn "advisory\|never block" pkg/verify/accretion_precommit.go
grep -rn "advisory\|never block" pkg/spawn/gates/hotspot.go
grep -rn "result.Passed = false" pkg/verify/  # Find all blocking gates
grep -rn "CheckProbeModelMerge" pkg/verify/   # Probe-to-model merge gate
grep -rn "agreements check\|kb agreements" pkg/spawn/gates/agreements.go

# Measurement check — event types, stats output
grep -rn "accretion.delta\|accretion.snapshot" pkg/events/
grep -rn "spawn.hotspot\|spawn.gate_decision" pkg/events/

# Decision check — related decisions in .kb/decisions/
ls .kb/decisions/*accretion* .kb/decisions/*hotspot* .kb/decisions/*health*
```

Read full model (498 lines), all related decisions (6), all related probes (5), and verified implementation files directly.

---

## What I Observed

### Full Intervention Scorecard

#### A. CODE ACCRETION GATES

| # | Intervention | Status | Blocking? | Measurement Data? | Evidence of Reduction? |
|---|---|---|---|---|---|
| 1 | **Pre-commit accretion gate** (`pkg/verify/accretion_precommit.go`) | IMPLEMENTED | Advisory only (never blocks, line 34) | Yes: 55 firings, 2 blocks, 100% bypass rate (2-week probe) | **No direct effect.** 100% bypass rate. Indirect: event emission triggers daemon extraction |
| 2 | **Spawn hotspot gate** (`pkg/spawn/gates/hotspot.go`) | IMPLEMENTED | Advisory only (line 39-41) | Yes: 0 blocks in 201 invocations (0% block rate) | **No direct effect.** Gate dormant — all CRITICAL files already extracted below threshold |
| 3 | **Completion accretion gate** (`pkg/verify/accretion.go`) | IMPLEMENTED | Advisory only (appends to Warnings, not Errors; Passed stays true) | Included in completion pipeline | **No direct effect.** Advisory warnings only |
| 4 | **Daemon architect escalation** (`pkg/daemon/architect_escalation.go`) | IMPLEMENTED | Routing (not blocking) | Yes: `daemon.architect_escalation` events | **Indirect.** Routes work to architect skill. No before/after measurement |
| 5 | **Spawn context injection** (`cmd/orch/hotspot_spawn.go`) | IMPLEMENTED | Advisory (info in SPAWN_CONTEXT.md) | No measurement | **Unknown.** No way to measure if agents use the injected info |
| 6 | **Model-stub pre-commit gate** (`pkg/verify/model_stub_precommit.go`) | IMPLEMENTED | **BLOCKING** (only knowledge hard gate) | Yes: `result.Passed = false` on template placeholders | **Yes (preventive).** Blocks unfilled model templates. Override: `FORCE_MODEL_STUB=1` |
| 7 | **Duplication pre-commit gate** (`pkg/verify/duplication_precommit.go`) | IMPLEMENTED | Advisory only (`Passed: true` always) | Yes: 259 matches, 64.6% precision (164 TP, 90 FP) | **No direct effect.** 35% FP rate produces alert fatigue |
| 8 | **Health score gate** (`pkg/spawn/gates/health.go`) | **REMOVED** | Was advisory, then removed entirely | Yes: never fired (calibrated to pass existing codebase) | **No effect.** Removed 2026-03-11 (decision: remove-health-score-spawn-gate) |
| 9 | **Self-review completion gate** | **REMOVED** | Was blocking, then removed | Yes: 79% FP, 0 TP across 71 events in 1 week | **Negative effect.** Normalized bypass behavior. Removed 2026-03-13 |

#### B. KNOWLEDGE ACCRETION GATES

| # | Intervention | Status | Blocking? | Measurement Data? | Evidence of Reduction? |
|---|---|---|---|---|---|
| 10 | **Probe-to-model merge gate** (`pkg/verify/probe_model_merge.go`) | IMPLEMENTED | **BLOCKING** at completion | Yes: `result.Passed = false` for unmerged contradicts/extends probes | **Yes (structural).** Forces probe findings into parent models. This is the 2nd knowledge hard gate |
| 11 | **Prior Work table** (investigation skill template) | SOFT (template only) | No enforcement in code | Yes: 52% adoption rate measured | **Partial.** Convention violated 48% of the time |
| 12 | **Decision enforcement** (`pkg/spawn/gates/agreements.go`) | IMPLEMENTED | **Advisory only** (line 67: "WARNING-ONLY gate — it never blocks spawn") | Exists but low coverage (1/56 decisions had checks) | **Minimal.** 1.8% enforcement rate |
| 13 | **Investigation --model flag** | **NOT IMPLEMENTED** | N/A | N/A | N/A — proposed in model Open Question #6, never built |
| 14 | **Quick entry dedup** | **NOT IMPLEMENTED** | N/A | N/A | N/A — one confirmed duplicate pair, no automated detection |
| 15 | **Knowledge pre-commit hooks on .kb/ files** | **NOT IMPLEMENTED** | N/A | N/A | N/A — proposed in Open Question #2, never built |

#### C. ENTROPY METRICS

| # | Metric | Status | Tracked? | Consumed? | Drove Action? |
|---|---|---|---|---|---|
| 16 | **Orphan rate** (`pkg/kbmetrics/orphans.go`) | IMPLEMENTED | Yes: 87.6% overall, 52% model-era | On-demand (`orch kb orphans`) | Yes — orphan taxonomy decomposition informed model. Not daemon-continuous |
| 17 | **Model probe freshness** (`pkg/daemon/trigger_detectors_phase2.go`) | IMPLEMENTED | Yes: >30 day threshold | Daemon Phase 2 triggers | Yes — generates investigation issues for stale models |
| 18 | **Contradicts backlog** (`pkg/daemon/trigger_detectors_phase2.go`) | IMPLEMENTED | Yes: scans for contradiction signals | Daemon Phase 2 triggers | Yes — creates beads issues for model updates |
| 19 | **Synthesis debt** (`pkg/tree/health_smell.go`) | PARTIALLY IMPLEMENTED | Yes: SmellNeedsSynthesis type | `kb reflect` only (manual) | **No.** Detection is retroactive, not daemon-driven |
| 20 | **Claim density** (`pkg/kbmetrics/claims.go`) | IMPLEMENTED | Yes: thresholds 30 (warning), 50 (critical) | `orch kb audit models` | **No direct evidence.** No model has been split due to this metric |
| 21 | **Quick entry duplication rate** | NOT IMPLEMENTED | No | No | No |
| 22 | **Composite health score** (`cmd/orch/health_cmd.go`) | IMPLEMENTED | Yes: 5-dimension 0-100 score | `orch health` command | **Misleading.** 37→73 improvement was 90% calibration artifact, 10% real extraction |
| 23 | **False ground truth detection** | NOT IMPLEMENTED | No | No | No — only described in model, no code |
| 24 | **accretion.delta events** (`pkg/events/logger.go`) | IMPLEMENTED | Yes: emitted at completion | Harness audit consumes | **Partially.** Data collected but no automated response to rising deltas |
| 25 | **accretion.snapshot events** (`pkg/events/logger.go`) | IMPLEMENTED | Yes: daemon emits periodically (>6 day interval) | Harness audit consumes | **Partially.** Same gap — measurement without automated response |
| 26 | **N-value tracking** (`cmd/orch/stats_cmd.go`) | IMPLEMENTED | Yes: event count, KB files, workspace count | `orch stats --verbose` | **New (2026-03-20).** Identifies scale growth. No automated threshold/alert |
| 27 | **Hotspot analysis** (`cmd/orch/hotspot.go`) | IMPLEMENTED | Yes: 4 detection signals | Daemon escalation, spawn context | **Yes.** 12→3 CRITICAL files (75% reduction), driven by daemon extraction cascades |

#### D. ATTRACTORS (Structural Destinations)

| # | Attractor | Status | Evidence of Pull? |
|---|---|---|---|
| 28 | **Models as knowledge attractors** (`.kb/models/*/`) | STRUCTURAL | **Yes.** Orphan rate dropped 94.7%→52.0% after model/probe system introduced |
| 29 | **Probe directory coupling** (`.kb/models/*/probes/`) | STRUCTURAL | **Yes.** Converts attention priming to structural coupling via directory placement |
| 30 | **kb autolink** (`cmd/orch/kb.go`) | IMPLEMENTED | **Partial.** Automation aid, dry-run default. Measurement of link rate not tracked |
| 31 | **kb orphans --stratified** (`cmd/orch/kb.go`) | IMPLEMENTED | **Measurement only.** No gating, categorizes orphans into 4 buckets |

### Key Decisions That Changed Intervention Strategy

| Decision | Date | Original | Outcome | Evidence |
|---|---|---|---|---|
| **Three-layer hotspot enforcement** | 2026-02-26 | Layer 0-3 blocking architecture | Layers 0-1 converted to advisory | 100% bypass rate on blocks |
| **Health score floor gate** | 2026-03-10 | Blocking at floor=65 | Downgraded same day, removed 2026-03-11 | 37→73 improvement was 90% calibration artifact |
| **Self-review completion gate** | 2026-03-13 | Blocking (4 automated checks) | Removed entirely after 1 week | 79% FP, 0 TP across 71 events |
| **Accretion gates advisory** | 2026-03-17 | Blocking pre-commit + spawn gates | Proposed conversion to advisory | 55 firings, 100% bypass rate on blocks |

### What Actually Reduced Accretion (by Evidence)

**Only 4 interventions have measurable evidence of reducing the thing they target:**

1. **Daemon extraction cascades** (triggered by gate events, not gate blocks): 12→3 CRITICAL files (75% reduction). The blocking mechanism had 0% effectiveness; the signaling mechanism triggered the daemon to spawn extraction work.

2. **Model/probe system as attractor**: Orphan rate 94.7%→52.0% in model-era investigations. The directory structure (`.kb/models/*/probes/`) provides structural coupling that pulls investigative work toward models.

3. **Model-stub pre-commit gate**: Prevents empty model templates from being committed. Preventive gate — no measured violations because it blocks them preemptively.

4. **Probe-to-model merge gate**: Forces probe findings to be merged into parent models at completion time. This is the only knowledge hard gate discovered that was NOT in the model's gate deficit table (the model says "zero hard gates" for knowledge — this contradicts that claim).

**Everything else is either:**
- Advisory (no measured behavioral change from warnings)
- Not implemented (proposed but never built)
- Removed (negative ROI: high FP, 0 TP, or calibration artifacts)
- Measurement-only (tracks the problem, doesn't reduce it)

---

## Model Impact

- [x] **Contradicts** invariant: "Every knowledge transition is either ungated or advisory-only" (Section 3, Gate Deficit table). The probe-to-model merge gate (`pkg/verify/probe_model_merge.go`) IS a blocking knowledge gate — it prevents completion when probes with "contradicts" or "extends" verdicts haven't been merged into the parent model. The model-stub pre-commit gate is acknowledged in the model's invariant #1, but probe-to-model merge is missing from the Gate Deficit table entirely.

- [x] **Extends** model with: Intervention effectiveness hierarchy. The model proposes gates, attractors, and metrics as equivalent interventions. Empirical evidence shows a clear effectiveness hierarchy:
  1. **Structural attractors** (model/probe directory system) > **Signaling mechanisms** (event emission → daemon response) > **Blocking gates** (bypassed 100%) > **Advisory gates** (ignored) > **Metrics-only** (awareness without action)
  2. The pattern: blocking gates fail because agents route around them instantly. Signaling works because it triggers automated daemon responses. Structural attractors work because they're embedded in the substrate (directory structure). This maps to the model's own coordination taxonomy: structural coupling > attention priming > convention.

- [x] **Extends** model with: Gate lifecycle pattern. Every blocking gate in the system has followed the same arc: designed → measured → found inert or high-FP → downgraded to advisory or removed. The health score gate lasted 1 day. Self-review lasted 1 week. Accretion blocking lasted ~3 weeks before 100% bypass was measured. The model should document this as a predicted pattern: in systems with capable agents, blocking gates degrade to ceremony unless the block is structurally unbypassable (like `go build` or the model-stub precommit).

---

## Notes

### Intervention Counts Summary
- **Total interventions identified in model:** 31
- **Implemented in code:** 22 (71%)
- **Have measurement data:** 18 (58%)
- **Demonstrably reduced their target:** 4 (13%)
- **Removed or abandoned:** 2 (6.5%)
- **Not implemented:** 5 (16%)
- **Measurement-only (no action):** 4 (13%)

### The 13% Effectiveness Rate
Only 4 of 31 proposed interventions have evidence of reducing the thing they target. This is not a failure of the model — the model correctly identifies the problem (accretion) and correctly categorizes the intervention types (gates, attractors, metrics). But the model doesn't acknowledge that most of its own proposed interventions have been tested and found ineffective in their originally proposed form. The interventions that work (daemon cascades, structural attractors) work through different mechanisms than the model emphasizes (blocking gates, metrics).

### Missing from the Model
The model's Gate Deficit table (Section 3) should be updated to:
1. Add probe-to-model merge as a hard gate (it's missing entirely)
2. Note which gates were tried and removed (health score, self-review)
3. Document the blocking→advisory arc that all code accretion gates followed
4. Separate "what was proposed" from "what was measured to work"

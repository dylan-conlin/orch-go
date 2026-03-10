# Probe: Health Score Structural Improvement Validation (Phase 4)

**Model:** knowledge-physics
**Date:** 2026-03-10
**Status:** Complete
**Methodology:** Confirmatory — conducted by the same AI system that built the model. Cannot constitute independent validation.

---

## Question

Does the health score improvement from 37→73 reflect real structural improvement (extractions, accretion velocity decrease), or is it primarily from threshold recalibration and newly tracked metrics?

The plan claims: "Score should be >=65 and improvement should trace to actual extractions not just threshold changes."

---

## What I Tested

### Test 1: Score Decomposition (Baseline vs Current)

Reconstructed both scores from their component dimensions using the `ComputeHealthScore` formula in `pkg/health/health.go`.

```bash
# Current score (from orch doctor --health)
orch doctor --health
# Score: 73/100 (C)

# Baseline score (reconstructed from first health snapshot)
# Snapshot [0]: bloated=57, hotspot_count=N/A, gate_coverage=N/A, total_source=N/A
```

**Baseline (Mar 8, snapshot [0]):**
| Dimension | Points | Notes |
|-----------|--------|-------|
| Gate Coverage | 0.0/20 | Not tracked (defaulted to 0) |
| Accretion Control | 0.0/20 | 57 bloated / threshold 20 = saturated |
| Fix:Feat Balance | 15.3/20 | Ratio 0.71 |
| Hotspot Control | 20.0/20 | Not tracked (defaulted to 0 = free 20pts) |
| Bloat Percentage | 0.1/20 | Legacy exp(-57/10) decay |
| **TOTAL** | **35.3/100** | |

**Current (Mar 10, snapshot [52]):**
| Dimension | Points | Notes |
|-----------|--------|-------|
| Gate Coverage | 20.0/20 | 6/6 gates active |
| Accretion Control | 15.7/20 | 20 bloated / threshold 92.8 |
| Fix:Feat Balance | 15.6/20 | Ratio 0.66 |
| Hotspot Control | 1.9/20 | 42 hotspots / threshold 46.4 |
| Bloat Percentage | 19.6/20 | 20/928 = 2.2% |
| **TOTAL** | **72.7/100** | |

### Test 2: Contribution Analysis (Calibration vs Extraction)

Computed score under 4 scenarios to isolate each effect:

```
A. OLD formula + OLD structure (baseline):       35.3
B. OLD formula + NEW structure (extraction only): 35.7  (+0.4)
C. NEW formula + OLD structure (calibration only):69.1  (+33.8)
D. NEW formula + NEW structure (both):            72.8  (+37.5)
```

### Test 3: Extraction Verification

Checked all 10 target files from the plan against current sizes:

| File | Plan (lines) | Current (lines) | Status |
|------|-------------|-----------------|--------|
| clean_cmd.go | 1270 | 360 | Extracted |
| tmux.go | 1201 | 546 | Extracted |
| spawn_cmd.go | 1171 | 508 | Extracted |
| account.go | 1162 | 513 | Extracted |
| kb.go | 1138 | 280 | Extracted |
| serve_beads.go | 1124 | 746 | Extracted |
| client.go | 1115 | 306 | Extracted |
| serve_system.go | 1084 | 538 | Extracted |
| kbcontext.go | 1072 | 182 | Extracted |
| session.go | 1055 | 121 | Extracted |

Average reduction: 1139→410 lines (64% smaller). All 10 confirmed via `git log --grep="refactor\|extract"`.

### Test 4: Snapshot Timeline Analysis

Read all 56 health snapshots to trace exact score evolution:

```
Mar 8 17:28  score=35.3  bloated=57  (baseline, no tracking)
Mar 9 13:51  score=36.7  bloated=29  (test threshold 800→2000, -28 files)
Mar 9 18:56  score=37.0  bloated=27  (extractions, -2 files)
Mar 10 01:23 score=37.0  bloated=26  (extractions, -1 file)
Mar 10 08:46 score=37.0  bloated=26  hotspots=52, gates=1 (tracking added)
Mar 10 09:03 score=68.9  bloated=27  hotspots=49, total_src=900 (JUMP: total_source_files added)
Mar 10 09:18 score=71.9  bloated=21  hotspots=43 (extractions continue)
Mar 10 09:38 score=72.7  bloated=20  hotspots=42 (current stable state)
```

**Critical observation:** Score jumped 37→69 in a SINGLE snapshot when `total_source_files` field was added. The subsequent 69→73 (+4 pts) came from actual extraction work.

### Test 5: CRITICAL Hotspot Check

```bash
find . -name "*.go" -o -name "*.svelte" -o -name "*.ts" | xargs wc -l | sort -rn | head
```

No source files >1500 lines (only node_modules and test files). Zero CRITICAL hotspots.

### Test 6: Accretion Velocity

Cannot measure velocity decrease from snapshots because the formula changed mid-stream. The pre-commit accretion gate was wired on Mar 10 (commit e33886d80) and is functional. However, only ~12 hours have elapsed since the last extraction — insufficient time to measure re-accretion rate.

---

## What I Observed

### 1. Score Improvement Attribution

| Factor | Points | % of Total |
|--------|--------|-----------|
| Calibration (formula changes) | +33.8 | 90% |
| Extraction (structural work) | +0.4 | 1% |
| Interaction (calibration + extraction) | +3.3 | 9% |
| **Total** | **+37.5** | **100%** |

**The 37→73 improvement is 90% calibration, ~10% extraction.** The old formula was too broken to register the extractions — it permanently zeroed 2 dimensions for any codebase >200 files.

### 2. Calibration Was Fixing a Broken Instrument

The calibration changes were not threshold gaming. They fixed three real measurement defects:

1. **Gate tracking (+20 pts):** Gates existed (pre-commit, spawn gate, completion) but weren't measured. Adding tracking didn't create gates — it measured existing ones.

2. **Codebase-scaled thresholds (+14.4 pts):** Fixed thresholds (20 bloated, 15 hotspots) permanently zeroed dimensions for a 928-file codebase. The max(20, 10%) scaling is honest — 20 bloated files in 928 (2.2%) is genuinely good, but the old formula scored it 0/20.

3. **Bloat percentage via ratio (+19.4 pts):** Legacy exp(-bloated/10) decay was nonsensical for large codebases. Ratio-based calculation (bloated/total) is the correct metric.

4. **Hotspot tracking (-20 pts):** This was the HONEST correction — the baseline had 20 FREE points from untracked hotspots. Now 42 hotspots nearly zero the dimension (1.9/20). This makes the formula MORE honest.

### 3. Extractions Are Real and Significant

Despite contributing only ~4 points to the score, the extractions are substantial structural work:
- 10 files extracted, average 64% size reduction
- Bloated source files: 26→20 (23% reduction)
- All 10 target files from the plan were completed
- 16 extraction commits in 2 days

### 4. Hotspot Dimension Is the Remaining Pain

42 active hotspots against a 46.4 threshold leaves only 1.9/20 points. This is the honest signal that fix-density and investigation-cluster hotspots still need work. Hotspot count dropped 52→42 (19% reduction), but the dimension is nearly saturated.

### 5. Plan Success Criteria

| Criterion | Target | Actual | Met? |
|-----------|--------|--------|------|
| Score >=65 | 65 | 73 | Yes |
| Bloated source <=16 | 16 | 20 | **No** (4 short) |
| No CRITICAL (>1500 lines) | 0 | 0 | Yes |
| Orphaned issues <5 | 5 | 0 | Yes |
| Score honestly reflects health | - | Mixed | See below |

**Score honesty assessment:** The NEW formula is honest — it correctly tracks all dimensions, uses appropriate thresholds, and the hotspot near-zero is genuine. The IMPROVEMENT from 37→73 is primarily from fixing the measurement instrument, not structural change. This is not dishonest — but it means the plan's Phase 2 goal ("improvement should trace to actual extractions") is not validated.

---

## Model Impact

- [x] **Confirms** invariant #1: "Every convention without a gate will eventually be violated." The accretion velocity cannot be measured because only 12 hours have elapsed since the pre-commit gate was wired. The gate exists but its effectiveness is untested.

- [x] **Extends** model's entropy metrics section: The health score's 5-dimension formula is the first concrete implementation of the model's "entropy measurement" recommendation. However, the probe reveals that measurement calibration itself can dominate the signal — a meta-observation about entropy metrics: the act of improving measurement appears as improvement, even when the underlying state hasn't changed significantly.

- [x] **Contradicts** the plan's Phase 4 claim: "improvement should trace to actual extractions not just threshold changes." 90% of the score improvement IS threshold changes. The extractions are real but their score impact is ~4 points, not the claimed 36-point improvement.

- [x] **Extends** with a new finding: **Measurement-improvement bias.** When a broken metric is fixed, the improvement in the metric looks like improvement in the thing being measured. The health score jumped 37→69 when `total_source_files` was added — a pure measurement fix. Systems tracking their own health need to distinguish "we got healthier" from "we got better at measuring."

---

## Notes

1. The accretion velocity question remains open. The pre-commit gate is wired but only 12 hours old. Re-accretion measurement needs at least 2-4 weeks of data.

2. Snapshot [50-51] (bloated=5, total_src=75, score=81.4) appears anomalous — likely `orch health` was run from wrong directory or a subdirectory. These snapshots should be excluded from trend analysis.

3. The hotspot dimension (1.9/20) is the clearest signal of remaining structural debt. The 42 hotspots are driven by fix-density and investigation-cluster hotspots, not just file size. Extraction alone won't fix this — it requires reducing the fix rate (fewer bugs) and investigation deduplication.

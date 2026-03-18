# Probe: Knowledge Decay Verification — Hotspot Acceleration Model

**Date:** 2026-03-18
**Probed by:** orch-go-x5ixa
**Model:** hotspot-acceleration
**Trigger:** Knowledge decay (999d since last probe — model never previously probed)

## Method

1. Read model claims about detector deficiencies, false positive taxonomy, and watchlist
2. Read current detector code at `pkg/daemon/trigger_detectors_phase2.go`
3. Cross-referenced watchlist file sizes with `wc -l`
4. Checked status of three actionable findings

## Findings

### Claim: Detector at lines 314-347 — INCORRECT

**Actual location:** `HotspotAccelerationDetector` type at line 166, `Detect()` method at lines 173-201. Lines 314-347 are part of `SkillPerformanceDriftDetector`. The line reference was wrong even at model creation time.

### Claim: Deficiency 1 (No Birth Churn Filtering) — PARTIALLY ADDRESSED

The detector now uses `git diff --numstat` which computes net growth between HEAD and a 30-day-old baseline commit. This means:
- A file created within the window still shows its full size as growth (birth at 0 → current size = net growth)
- BUT extraction artifacts where the source file shrank correspondingly now show reduced net impact
- Additionally, `minAccelerationSize = 500` (line 440) filters out files under 500 lines, eliminating most birth-churn false positives (22/22 birth-churn examples in the model were under 570 lines)

**Impact on false positive rate:** The 500-line minimum alone would have eliminated 20 of 22 birth-churn false positives from the model's dataset.

### Claim: Deficiency 2 (No Path Exclusions) — FIXED

`isAccelerationExcluded()` (lines 411-435) now implements comprehensive exclusions:
- `skipAccelerationDirs` map (lines 391-407) covers: `.git`, `node_modules`, `vendor`, `.svelte-kit`, `dist`, `build`, `__pycache__`, `.next`, `.nuxt`, `.output`, `.opencode`, `.orch`, `.beads`, `.claude`, `experiments`
- Test files (`_test.go`) excluded
- Generated files (`/generated/`) excluded

This directly addresses the "79 false positives from experiments/" issue. The model's claim that the detector "does not use `shouldCountFile()`" is technically still true — it uses its own parallel implementation — but the functional gap is closed.

### Claim: Deficiency 3 (Gross vs Net Counting) — FIXED

Code at lines 462-464 explicitly documents the change:
```go
// git diff --numstat gives net changes (added - deleted) per file between
// two points in time, unlike git log --numstat which sums additions across
// individual commits and counts churn as growth.
```

The detector uses `parseGitDiffNumstat()` (line 469) which computes `added - deleted` per file. The `FastGrowingFile.NetGrowth` field (line 160) reflects this.

### Claim: Deficiency 4 (No Extraction-Source Detection) — STILL UNFIXED

No cross-referencing of addition/deletion pairs across files. This remains a valid gap, though its practical impact is reduced by net counting (extraction artifacts where the source file shrank show lower net growth in both files).

### Watchlist Status

| File | Model Says | Current | Delta |
|------|-----------|---------|-------|
| `cmd/orch/daemon_loop.go` | 771 | 771 | 0 |
| `cmd/orch/stats_aggregation.go` | 959 | 964 | +5 |
| `pkg/daemon/digest.go` | 775 | 775 | 0 |
| `pkg/daemon/digest_gate.go` | 262 | 262 | 0 |

All watchlist sizes confirmed current. No files have crossed thresholds.

### Actionable Findings Status

1. **`pkg/daemon/digest.go` → `pkg/digest/`**: NOT implemented. `pkg/digest/` does not exist. Recommendation still valid.
2. **`pkg/account/account_test.go` split**: COMPLETED. Reduced from 1,452 to 634 lines. `capacity_test.go` created at 832 lines.
3. **`pkg/daemon/extraction_test.go` split**: COMPLETED. Split into `extraction_test.go` (500 lines) + `extraction_integration_test.go` (237 lines).

### False Positive Rate Estimate (Updated)

With current code improvements:
- Birth churn (22 cases): ~20 now filtered by `minAccelerationSize >= 500`, ~2 edge cases remain
- Extraction artifacts (8 cases): Net counting reduces most to below threshold
- Design churn (1 case): Net counting eliminates this entirely
- Path exclusions: `experiments/` now excluded (would have been 79 additional false positives)

**Estimated false positive rate with fixes: significantly reduced from 91%**, though exact rate requires re-running the detector against current codebase.

## Verdict

**Model is STALE.** Three of four detector deficiencies have been fixed in code without model updates. The false positive taxonomy is historically accurate but the "Detector Deficiencies" section and the core claim of ~91% false positive rate are misleading for anyone reading the model today.

### Required Model Updates

1. Mark Deficiency 2 (path exclusions) as RESOLVED — `isAccelerationExcluded()` exists
2. Mark Deficiency 3 (gross vs net) as RESOLVED — `git diff --numstat` implemented
3. Mark Deficiency 1 (birth churn) as PARTIALLY RESOLVED — `minAccelerationSize=500` + net counting
4. Fix line number reference (166-201, not 314-347)
5. Note Deficiency 4 as the sole remaining gap
6. Update false positive rate estimate
7. Mark actionable findings 2 and 3 as completed
8. Add `minAccelerationSize` and test file exclusion to model

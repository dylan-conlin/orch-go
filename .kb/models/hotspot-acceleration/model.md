# Hotspot Acceleration Model

**Status:** Active
**Created:** 2026-03-17
**Last Updated:** 2026-03-17
**Evidence Base:** 34 investigations (all 2026-03-17), covering cmd/orch/, pkg/daemon/, pkg/spawn/, pkg/kbgate/, pkg/kbmetrics/, pkg/account/, pkg/verify/, pkg/dupdetect/, pkg/thread/, pkg/daemonconfig/, experiments/

## Core Claim

The hotspot acceleration detector produces a high false-positive rate (~91%, 31/34 investigations) because it measures **gross line additions within a 30-day window** without distinguishing three fundamentally different growth patterns: file birth, extraction artifacts, and genuine accretion.

## False Positive Taxonomy

Across 34 investigated files, false positives fall into four distinct categories:

### Category 1: Birth Churn (100% creation)

**Pattern:** File created within the 30-day window. Entire line count registers as "growth."

**Frequency:** 22 of 34 investigations (65%)

**Examples:**
- `pkg/thread/thread_test.go` — 570 lines, 1 commit, 0 post-birth changes
- `pkg/kbgate/model_test.go` — 386 lines, 1 commit, 0 post-birth changes
- `pkg/dupdetect/staged_test.go` — 257 lines, 1 commit, 0 post-birth changes
- `pkg/daemon/trigger.go` — 204 lines, 2 commits (same week), 0 subsequent changes
- `pkg/kbmetrics/provenance_test.go` — 366 lines, 1 commit, 0 post-birth changes
- `pkg/spawn/templates.go` — 367 lines, 1 commit, 0 post-birth changes (~90% string constants)

**Distinguishing signal:** `git log --diff-filter=A` shows file created within window. Total additions ≈ current file size. Zero or near-zero post-birth commits.

### Category 2: Extraction Artifacts

**Pattern:** File created by extracting code from a larger file (the hotspot *cure* triggering the hotspot *alarm*).

**Frequency:** 8 of 34 investigations (24%)

**Examples:**
- `cmd/orch/complete_checklist.go` — 203 lines, extracted from complete_actions.go, net -2 lines post-birth
- `pkg/spawn/kbmodel.go` — 530 lines, extracted from kbcontext.go (1496→1014), net +37 post-birth
- `pkg/account/capacity.go` — 654 lines, extracted from account.go (1162→513), 0 post-birth commits
- `pkg/orch/spawn_pipeline.go` — 463 lines, extracted from extraction.go, net -17 lines (shrinking)
- `cmd/orch/status_infra.go` — 308 lines, extracted from status_cmd.go (-772 lines)
- `pkg/verify/skip_test.go` — 236 lines, extracted from complete_cmd.go

**Distinguishing signal:** Extraction commit message contains "extract", "refactor", or "split". Source file shows corresponding deletion. The flagged file is the *product* of the remedy the hotspot system recommends.

### Category 3: Design Churn (Delete/Recreate)

**Pattern:** File deleted and recreated during design iteration. Gross additions double-count the file.

**Frequency:** 1 of 34 investigations (3%)

**Example:**
- `cmd/orch/control_cmd.go` — 211 lines, deleted Feb 17 (-302), recreated Mar 1 (+157), circuit breaker experimentation

**Distinguishing signal:** `git log --diff-filter=D` shows file deleted within window. Total additions >> current file size.

### Category 4: Post-Extraction Feature Growth

**Pattern:** File born from extraction, then grew via legitimate feature additions. Growth is real but within safe bounds.

**Frequency:** 3 of 34 investigations (9%) — NOT false positives but NOT actionable

**Examples:**
- `pkg/daemon/preview_test.go` — 476 lines, 56% birth + 224 lines organic, growth drivers exhausted
- `pkg/daemon/completion_processing_test.go` — 438 lines, 45% birth + 241 lines from 2 features (both shipped)
- `cmd/orch/daemon_loop.go` — 771 lines, 89% birth + 107 net lines post-birth, `daemonSetup()` is growth vector

**Distinguishing signal:** Multiple post-birth commits with substantive additions. But current size still well under threshold and growth drivers identifiable/finite.

## Genuine Hotspots Found

Of 34 investigations, only **2 identified actionable findings**:

### 1. `pkg/daemon/digest.go` — Preventive Extraction Warranted (P3)

- 775 lines, burst-created, but contains 4 separable responsibilities
- Combined with `digest_gate.go` (262 lines), daemon holds 2,024 lines of digest code
- `serve_digest.go` imports `daemon` for pure data-layer operations (conceptual coupling)
- **Recommendation:** Extract to `pkg/digest/` package (~655 lines move out, ~120 lines stay)

### 2. `pkg/account/account_test.go` — Test File Split Executed

- 1,452 lines approaching 1,500-line critical threshold
- ~816 lines of capacity/auto-switch tests belonged with `capacity.go`
- **Action taken:** Extracted to `capacity_test.go`, reducing to ~636 lines

### 3. `pkg/daemon/extraction_test.go` — Test File Split Executed

- 739 lines with clear seam between unit tests and integration tests
- Duplicate mock eliminated
- **Action taken:** Split to `extraction_integration_test.go` (237 lines), reduced to 500 lines

## Detector Deficiencies

### Deficiency 1: No Birth Churn Filtering

The `HotspotAccelerationDetector` in `pkg/daemon/trigger_detectors_phase2.go:314-347` counts all line additions within the 30-day window. Files created within the window have their entire initial content counted as "growth."

**Proposed fix:** When a file was created within the measurement window, subtract the initial commit's insertions from the churn calculation. This would eliminate 65% of false positives.

### Deficiency 2: No Path Exclusions

The detector filters only by `.go` suffix. It does not use `shouldCountFile()`, `skipBloatDirs`, or `containsSkippedDir()` from the CLI hotspot tool (`cmd/orch/hotspot_analysis.go`).

**Impact:** `experiments/` alone generates 79 false positives (116,931 lines of static experiment artifacts). Other unfiltered paths: `.orch/`, `.beads/`, `.claude/`, `node_modules`.

**Proposed fix:** Reuse or replicate the exclusion patterns from `cmd/orch/hotspot_analysis.go:120-161` in the daemon detector.

### Deficiency 3: Gross vs Net Counting

The metric sums raw additions across commits, ignoring deletions. Files that undergo churn (add then remove) show inflated metrics.

**Proposed fix:** Use net growth (additions minus deletions) instead of gross additions. This would eliminate the design-churn category and reduce noise from extraction artifacts.

### Deficiency 4: No Extraction-Source Detection

When a file is born from extraction, the source file typically shows a corresponding large deletion in the same commit. The detector does not cross-reference addition/deletion pairs.

**Proposed fix:** If a file was created in a commit that also deleted lines from another file in the same package, flag as "extraction artifact" rather than "growth."

## Decision Framework

When a file is flagged by the hotspot acceleration detector, apply this triage:

```
1. Was file CREATED within the 30-day window?
   ├─ YES: Is it under experiments/, .orch/, .beads/, or other non-production path?
   │   └─ YES → FALSE POSITIVE (path exclusion gap)
   ├─ YES: Does git log show ≤3 commits, all near creation date?
   │   └─ YES → FALSE POSITIVE (birth churn)
   ├─ YES: Was it created by extraction (check commit message, source file deletions)?
   │   └─ YES: Is post-birth net growth < 100 lines?
   │       └─ YES → FALSE POSITIVE (extraction artifact)
   │       └─ NO → MONITOR (post-extraction growth)
   └─ NO: Is the file growing steadily across many commits?
       ├─ YES: Is it approaching 800 lines?
       │   └─ YES → INVESTIGATE (genuine hotspot risk)
       │   └─ NO → MONITOR (real growth, not yet concerning)
       └─ NO → LIKELY FALSE POSITIVE (churn, not growth)
```

## Quantitative Summary

| Category | Count | % | Action |
|---|---|---|---|
| Birth churn (100% creation) | 22 | 65% | Close — detector noise |
| Extraction artifacts | 8 | 24% | Close — extraction is the cure |
| Design churn | 1 | 3% | Close — delete/recreate cycle |
| Real growth, safe | 3 | 9% | Monitor — growth drivers exhausted |
| Actionable finding | 3* | 9%* | Extract or split |

*Note: 3 actionable findings overlap with other categories (digest.go was burst-created but warranted extraction; account_test.go and extraction_test.go had real growth).

## Watchlist (Files to Monitor)

Files that are healthy now but could become hotspots:

| File | Current Lines | Growth Vector | Threshold Trigger |
|---|---|---|---|
| `cmd/orch/daemon_loop.go` | 771 | `daemonSetup()` grows with each new subsystem | Extract `daemon_wiring.go` if `daemonSetup()` exceeds 250 lines |
| `cmd/orch/stats_aggregation.go` | 959 | Each new event type adds ~15-30 lines | Extract event processors if file exceeds 1200 lines |
| `pkg/daemon/digest.go` | 775 | Near 800-line advisory threshold | Extract to `pkg/digest/` (architect review recommended) |

## Evidence

### Investigations (2026-03-17)

**Regular investigations (11 files):**
- architecture_lint_test.go — false positive (birth churn, 317 lines, 3 commits)
- spawn_cmd.go — false positive (gross churn post-extraction, 542 lines, net -409)
- kbgate/publish_test.go — false positive (burst creation, 708 lines, 3-hour window)
- kbmetrics/provenance_test.go — false positive (birth churn, 366 lines, 1 commit)
- plan_hydrate.go — false positive (birth churn, 93% from creation, 250 lines)
- review_synthesize_test.go — false positive (birth churn, 271 lines, 1 commit)
- serve_agents_status.go — false positive (extraction artifact from 1713-line monolith, 297 lines)
- experiments/coordination-demo display_test.go — false positive (experiment artifact, path exclusion gap)
- daemon/trigger.go — false positive (birth churn, 204 lines, 2 days old)
- daemon/preview_test.go — real growth but safe (476 lines after dedup, growth drivers exhausted)
- daemon/digest.go — **actionable** (775 lines, 4 separable responsibilities, extraction warranted)

**Simple investigations (23 files):**
- control_cmd.go — false positive (delete/recreate design churn, 211 lines)
- thread_test.go — false positive (birth churn, 570 lines, 0 post-birth changes)
- daemonconfig/plist_test.go — false positive (birth churn, 393 lines after cleanup)
- dupdetect/staged_test.go — false positive (birth churn, 257 lines, 1 commit)
- experiments/coordination-demo trial-6 — false positive (experiment artifact)
- experiments/coordination-demo trial-10 — false positive (experiment artifact)
- complete_checklist.go — false positive (extraction artifact, 203 lines)
- daemon/issue_adapter_test.go — false positive (extraction artifact, 330 lines)
- spawn/kbmodel.go — false positive (extraction artifact from 1496-line file, 530 lines)
- kbgate/model_test.go — false positive (birth churn, 386 lines, 1 commit)
- account/capacity.go — false positive (extraction artifact from 1162-line file, 654 lines)
- daemon_loop.go — false positive (89% birth churn, monitor daemonSetup)
- stats_cmd.go — already extracted (4 files, monitor stats_aggregation.go at 959 lines)
- account/account_test.go — **actionable** (1452 lines, extracted capacity tests)
- pkg/orch/spawn_pipeline.go — false positive (extraction artifact, shrinking, 463 lines)
- daemon/extraction_test.go — **actionable** (split into unit + integration, 739→500+237)
- spawn/templates.go — false positive (birth churn, 90% string constants, 367 lines)
- verify/skip_test.go — false positive (extraction artifact, 236 lines)
- daemon/mock_test.go — false positive (birth churn, 282 lines serving 27 test files)
- daemon/pidlock.go — false positive (birth churn, 183 lines)
- daemon/completion_processing_test.go — real growth but safe (438 lines, growth drivers exhausted)
- status_infra.go — false positive (extraction artifact from status_cmd.go, 308 lines)
- plan_cmd.go — false positive (create-then-extract pattern, 368 lines)

# Hotspot Acceleration Model

**Status:** Active
**Created:** 2026-03-17
**Last Updated:** 2026-03-18
**Evidence Base:** 34 investigations (all 2026-03-17), covering cmd/orch/, pkg/daemon/, pkg/spawn/, pkg/kbgate/, pkg/kbmetrics/, pkg/account/, pkg/verify/, pkg/dupdetect/, pkg/thread/, pkg/daemonconfig/, experiments/

## Core Claim

The hotspot acceleration detector historically produced a ~91% false-positive rate (31/34 investigations, 2026-03-17) when it measured gross line additions. **Three of four identified deficiencies have since been fixed:** the detector now uses net counting (`git diff --numstat`), path exclusions (`isAccelerationExcluded()`), and a minimum file size threshold (`minAccelerationSize = 500`). The remaining gap is extraction-source detection. The false positive taxonomy below documents the original investigation; current false positive rates are expected to be significantly lower.

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

Of 34 investigations, **3 identified actionable findings** (2 now completed):

### 1. `pkg/daemon/digest.go` — Preventive Extraction Warranted (P3) — OPEN

- 775 lines (confirmed 2026-03-18), burst-created, but contains 4 separable responsibilities
- Combined with `digest_gate.go` (262 lines), daemon holds 2,024 lines of digest code
- `serve_digest.go` imports `daemon` for pure data-layer operations (conceptual coupling)
- **Recommendation:** Extract to `pkg/digest/` package (~655 lines move out, ~120 lines stay)
- **Status:** `pkg/digest/` does not exist. Recommendation not yet acted on.

### 2. `pkg/account/account_test.go` — Test File Split — COMPLETED

- Was 1,452 lines approaching 1,500-line critical threshold
- **Action taken:** Extracted capacity tests to `capacity_test.go` (832 lines), reducing `account_test.go` to 634 lines
- **Verified 2026-03-18:** Both files exist at expected sizes

### 3. `pkg/daemon/extraction_test.go` — Test File Split — COMPLETED

- Was 739 lines with clear seam between unit tests and integration tests
- Duplicate mock eliminated
- **Action taken:** Split to `extraction_integration_test.go` (237 lines), reduced to 500 lines
- **Verified 2026-03-18:** Both files exist at expected sizes

## Detector Deficiencies

The `HotspotAccelerationDetector` is at `pkg/daemon/trigger_detectors_phase2.go:166-201`, with supporting functions at lines 391-499.

### Deficiency 1: No Birth Churn Filtering — PARTIALLY RESOLVED

~~The detector counts all line additions within the 30-day window.~~

**Current state (2026-03-18):** Two mitigations now reduce birth-churn false positives:
1. **Net counting** (`git diff --numstat`): Birth of a file still shows full size as growth, but extraction artifacts with corresponding source deletions show reduced net impact.
2. **Minimum size threshold** (`minAccelerationSize = 500`, line 440): Files under 500 lines are excluded entirely. This alone would have eliminated 20 of the 22 original birth-churn false positives.

**Remaining gap:** Files born at >500 lines within the window still register their full size as growth (e.g., a new 600-line file shows net +600). Explicit birth-date detection would eliminate this edge case.

### Deficiency 2: No Path Exclusions — RESOLVED

~~The detector filters only by `.go` suffix.~~

**Fixed (2026-03-17):** `isAccelerationExcluded()` (lines 411-435) now implements comprehensive exclusions via `skipAccelerationDirs` (lines 391-407): `.git`, `node_modules`, `vendor`, `.svelte-kit`, `dist`, `build`, `__pycache__`, `.next`, `.nuxt`, `.output`, `.opencode`, `.orch`, `.beads`, `.claude`, `experiments`. Also excludes `_test.go` files and `/generated/` paths.

Note: Uses its own parallel implementation rather than reusing `shouldCountFile()` from `cmd/orch/hotspot_analysis.go`, but the functional coverage is equivalent.

### Deficiency 3: Gross vs Net Counting — RESOLVED

~~The metric sums raw additions across commits, ignoring deletions.~~

**Fixed (2026-03-17):** Now uses `git diff --numstat` between HEAD and a 30-day-old baseline commit (lines 462-469). Computes `added - deleted` per file. The `FastGrowingFile.NetGrowth` field reflects true net growth. Design-churn and extraction-artifact categories are largely eliminated by this change.

### Deficiency 4: No Extraction-Source Detection — STILL OPEN

When a file is born from extraction, the source file typically shows a corresponding large deletion in the same commit. The detector does not cross-reference addition/deletion pairs.

**Proposed fix:** If a file was created in a commit that also deleted lines from another file in the same package, flag as "extraction artifact" rather than "growth."

**Practical impact reduced:** With net counting, extraction artifacts where the source file shrank correspondingly now show lower net growth in both files, reducing (but not eliminating) false positives from this category.

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
| `cmd/orch/stats_aggregation.go` | 964 | Each new event type adds ~15-30 lines | Extract event processors if file exceeds 1200 lines |
| `pkg/daemon/digest.go` | 775 | Near 800-line advisory threshold | Extract to `pkg/digest/` (architect review recommended) |

## Evidence

### Probes

- 2026-03-18: Knowledge Decay Verification — 3 of 4 deficiencies resolved in code. Model updated to reflect current detector capabilities. Watchlist sizes confirmed current. Actionable findings 2 and 3 completed; finding 1 (digest.go extraction) still open.

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

## Auto-Linked Investigations

- .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-hotspot-spawn.md
- .kb/investigations/2026-03-17-inv-hotspot-acceleration-experiments-coordination-demo.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-daemon-compliance-test.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-spawn-templates.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-plan-test.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-experiments-coordination-demo.md
- .kb/investigations/2026-03-17-hotspot-acceleration-pkg-daemon-digest.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-thread-thread-test.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-daemon-issue.md
- .kb/investigations/2026-03-17-hotspot-acceleration-pkg-daemon-preview.md
- .kb/investigations/2026-03-17-hotspot-acceleration-pkg-kbgate-publish.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-account-account.md
- .kb/investigations/2026-03-17-hotspot-acceleration-cmd-orch-spawn.md
- .kb/investigations/2026-03-17-inv-hotspot-acceleration-cmd-orch-plan.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-daemon-pidlock.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-orient-debrief-test.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-group-group.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-experiments-coordination-demo-trial10.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-daemon-extraction.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-experiments-coordination-demo-trial8.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-daemon-daemon.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-skills-loader-test.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-experiments-coordination-demo-trial2.md
- .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-stats.md

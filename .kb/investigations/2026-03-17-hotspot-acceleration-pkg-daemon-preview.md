---
Status: Complete
Question: Is pkg/daemon/preview_test.go a hotspot requiring extraction?
Date: 2026-03-17
---

**TLDR:** preview_test.go is 508 lines but 284 were extracted from daemon_test.go on Feb 24, and only 224 lines are organic growth (11 lines/day). The "+532 lines/30d" metric is misleading — it conflates file creation with sustained accretion. Consolidated two duplicate epic test functions into a table-driven test (-32 lines). No extraction needed at 476 lines.

## D.E.K.N. Summary

- **Delta:** The "+532 lines/30d" hotspot metric conflates file creation via extraction with organic growth. Actual new code is 224 lines over 20 days. Consolidated two near-identical epic test functions (`TestPreview_EpicWithTriageApprovedShowsHelpfulMessage` and `TestPreview_EpicWithTriageReadyShowsHelpfulMessage`) into a single table-driven `TestPreview_EpicShowsHelpfulMessage`. File now 476 lines.
- **Evidence:** `git log` shows 284 lines created via extraction commit `32c739ec5` on Feb 24, followed by 224 lines of organic growth across 6 commits. Growth per commit: +20 (nil fix), +2 (refactor), +50 (rejection grouping), +57 (triage:approved), +19 (debrief), +76 (regression tests). Diff of the two epic test functions shows identical structure differing only in label value (`triage:approved` vs `triage:ready`).
- **Knowledge:** Test files born from extraction will always trigger hotspot metrics in their first 30 days. The hotspot detector should ideally discount the initial extraction commit, but this is a known false-positive pattern (also seen in trigger.go, kbgate/publish.go investigations).
- **Next:** Close. No extraction needed — file is at 476 lines, well below 800-line warning threshold. Preview feature is stabilizing (recent commits are regressions, not new capabilities). Re-evaluate only if file exceeds 800 lines.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| 2026-03-17-hotspot-acceleration-pkg-daemon-trigger.md | Same pattern (burst creation false positive) | yes | - |
| 2026-03-17-hotspot-acceleration-pkg-kbgate-publish.md | Same pattern (burst creation false positive) | yes | - |

## Findings

### Finding 1: Growth is extraction + organic, not runaway accretion

Git history for `pkg/daemon/preview_test.go`:

| Commit | Date | Lines | Delta | Description |
| --- | --- | --- | --- | --- |
| 32c739ec5 | Feb 24 | 284 | +284 | Initial extraction from daemon_test.go |
| 1cdef9858 | Feb 24 | 304 | +20 | Nil pointer fix regression test |
| 1f0357521 | Feb 28 | 306 | +2 | Interface refactor |
| 4fae96dfa | Mar 3 | 356 | +50 | Rejected issues grouping |
| ab3dd642e | Mar 5 | 413 | +57 | triage:approved handling |
| de9b386a0 | Mar 11 | 432 | +19 | Debrief |
| da1565f6a | Mar 15 | 508 | +76 | Completion label/recently spawned regressions |

284 of 508 lines (56%) came from extraction. Organic growth: 224 lines over 20 days (11 lines/day). This rate projects to 800 lines in ~30 more days, but the rate will slow — recent commits are regression tests for stabilizing features, not new capabilities.

### Finding 2: Duplicate epic tests consolidated

Two test functions were near-identical:
- `TestPreview_EpicWithTriageApprovedShowsHelpfulMessage` (lines 362-392)
- `TestPreview_EpicWithTriageReadyShowsHelpfulMessage` (lines 470-508)

Diff showed they differ only in the label value (`triage:approved` vs `triage:ready`) and minor comments. Consolidated into a single table-driven `TestPreview_EpicShowsHelpfulMessage` that covers both labels. This also adds the `rejected.Issue.ID` assertion that was missing from the first variant.

### Finding 3: File structure is well-scoped

The test file covers a single production file (`preview.go`, 213 lines) with 3 public functions:
- `Preview()` — 9 test functions covering core behavior, rejection, rate limiting, label handling
- `CountSpawnable()` — 1 test function
- `FormatPreview()` — 1 test function
- `FormatRejectedIssues()` — 3 test functions (basic, grouped, empty)

This is idiomatic Go testing — 1:1 test file per production file. Splitting would create artificial boundaries.

## Test Performed

```bash
go test ./pkg/daemon/ -run "TestFormatPreview|TestDaemon_Preview|TestDaemon_CountSpawnable|TestPreview|TestFormatRejectedIssues" -v -count=1
# PASS: 16 tests (14 top-level, 4 subtests), 0.632s
```

## Conclusion

False positive hotspot. The "+532 lines/30d" metric captures file creation via extraction, not sustained accretion. Organic growth is 224 lines over 20 days. File is at 476 lines after dedup, well below the 800-line warning threshold. No extraction recommended. The file follows idiomatic Go conventions (1:1 with preview.go) and growth will naturally taper as the Preview feature stabilizes.

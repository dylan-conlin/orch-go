# Investigation: Hotspot Acceleration — pkg/dupdetect/allowlist_test.go

**Status:** Complete
**Date:** 2026-03-17
**Beads:** orch-go-n8b2a

## TLDR

pkg/dupdetect/allowlist_test.go (+514 lines/30d, now 514 lines) is a **false positive** — the file was born 2026-03-11 (6 days ago) across 2 feature commits. Its entire 514-line existence falls within the 30-day measurement window. Growth is 100% birth churn, not ongoing accretion.

## D.E.K.N. Summary

- **Delta:** Classified as false positive — no extraction needed
- **Evidence:** File born 2026-03-11 via commit 319514de2 (281 lines), extended 2026-03-14 via commit 723c2cd6c (+233 lines for pair pattern feature). Current size 514 lines (66% below 1,500-line threshold). Production code (`allowlist.go`) is only 96 lines — test-to-code ratio is 5.35:1 which is high but reasonable for a pattern-matching module with many edge cases.
- **Knowledge:** Birth churn continues as the dominant false positive pattern in hotspot detection. Test files for pattern-matching code naturally have high line counts due to inline source fixtures.
- **Next:** Close. No action required.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-dupdetect-staged.md | extends | yes | - |
| 2026-03-17-hotspot-acceleration-pkg-daemon-cleanup-test.md | extends | yes | - |

Same false positive pattern: files born during the 30-day window flagged as hotspots.

## Question

Is pkg/dupdetect/allowlist_test.go a genuine accretion hotspot requiring extraction, or a false positive?

## Findings

### Finding 1: File birth analysis

The file was born on 2026-03-11 (6 days ago) via commit `319514de2` which added the allowlist feature to dupdetect. It grew by 233 lines on 2026-03-14 via commit `723c2cd6c` which added pair pattern syntax — a distinct feature addition, not accretion.

The entire 514 lines are birth churn: the file didn't exist before 2026-03-11.

### Finding 2: Test structure analysis

14 test functions covering:
- Single pattern matching (both-match, one-side-match, receiver patterns, multiple patterns)
- Suppressed pair tracking and reset behavior
- Empty allowlist behavior
- File loading (.dupdetectignore format)
- Pair pattern syntax (basic, order-independent, globs, non-matching)
- Integration with ScanProject

Each test is self-contained with inline Go source fixtures. The repetitive fixture pattern (~6 lines per function body) inflates line count but is the standard approach for parser/matcher test files — extracting fixtures would reduce readability without meaningful benefit.

### Finding 3: Growth trajectory assessment

Production code: 96 lines (allowlist.go)
Test code: 514 lines (allowlist_test.go)
Ratio: 5.35:1

The file is feature-complete — allowlist supports single patterns and pair patterns, both loaded from file. No upcoming features are planned that would add to this file. The 514-line size is well below the 1,500-line extraction threshold.

## Test performed

```bash
go test ./pkg/dupdetect/ -run TestAllowlist -v -count=1
```

All 14 allowlist tests pass.

## Conclusion

**False positive.** The file was born 6 days ago across 2 feature commits. Its 514-line size is 100% birth churn within the 30-day measurement window. The production code it tests is only 96 lines. No extraction needed — the file is well below the 1,500-line threshold and is feature-complete.

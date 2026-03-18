# Investigation: Hotspot Acceleration — pkg/daemon/cleanup_test.go

**Status:** Complete
**Date:** 2026-03-17
**Beads:** orch-go-g2jti

## TLDR

pkg/daemon/cleanup_test.go (+276 lines/30d, 262 lines) is a **false positive** — the file was born 2026-02-18, and its entire 262-line existence falls within the 30-day measurement window. Growth is birth churn from 6 legitimate feature/fix commits, not ongoing accretion.

## D.E.K.N. Summary

- **Delta:** Classified as false positive — no extraction needed
- **Evidence:** File born 2026-02-18 (27 days ago). 6 commits each added tests for distinct features. Current size 262 lines (83% below 1,500-line threshold).
- **Knowledge:** Birth churn continues to be the dominant false positive pattern in hotspot detection
- **Next:** Close. No action required.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-cmd-orch-serve-agents-status-test.md | extends | yes | - |
| 2026-03-17-hotspot-acceleration-experiments-coordination-demo-trial8.md | extends | yes | - |

Same false positive pattern: files born during the 30-day window flagged as hotspots.

## Question

Is pkg/daemon/cleanup_test.go a genuine accretion hotspot requiring extraction, or a false positive?

## Findings

### Finding 1: File was born during measurement window

The file was created on 2026-02-18 (commit `cd98751e1`). The 30-day measurement window starts ~2026-02-15. The entire file existence falls within the measurement window.

### Finding 2: Growth is 6 discrete feature commits, not accretion

| Date | Commit | Lines Added | Purpose |
|---|---|---|---|
| 2026-02-18 | cd98751e1 | +60 | Birth: stale window cleanup tests |
| 2026-02-28 | 1f0357521 | +4/-4 | Refactor: Daemon fields to interfaces |
| 2026-03-02 | 108495ddc | +79 | Fix: protect active Claude CLI workers |
| 2026-03-03 | 309dff419 | +60 | Feature: TTL-based workspace expiry |
| 2026-03-05 | 4d249d401 | +59 | Feature: batch beads queries in GC |
| 2026-03-06 | f54de2ccb | +14/-10 | Refactor: extract PeriodicScheduler |

Each commit added tests for a distinct, legitimate feature or fix. No padding or duplication.

### Finding 3: File is well-organized with 4 natural test groups

1. `isWindowStale` tests (lines 12-87) — 6 tests, ~76 lines
2. `RunPeriodicCleanup` tests (lines 89-145) — 2 tests, ~57 lines
3. `expireArchivedWorkspaces` tests (lines 147-191, 252-262) — 2 tests, ~55 lines
4. `isWindowStaleBatch` tests (lines 193-250) — 6 tests, ~57 lines

Each group tests a distinct function. No extraction benefit — splitting would create 4 files of ~60 lines each, adding overhead with no readability gain.

### Finding 4: Size is 83% below critical threshold

At 262 lines, the file is well below the 1,500-line accretion boundary. Even at the current growth rate (~15 lines/week from Mar 6 onward), it would take 5+ years to reach the threshold.

## Test Performed

```bash
git log --format="%h %ad %s" --date=short --numstat -- pkg/daemon/cleanup_test.go
```

Verified: file was created 2026-02-18, total additions sum to ~276 lines matching the hotspot flag.

```bash
go test ./pkg/daemon/ -run "TestIsWindowStale|TestRunPeriodic|TestExpire" -v -count=1
```

## Conclusion

**False positive.** The 276-line/30-day growth is 100% birth churn — the file was created 2026-02-18 and grew through 6 legitimate feature commits. At 262 lines it is healthy, well-organized, and 83% below the extraction threshold. No action needed.

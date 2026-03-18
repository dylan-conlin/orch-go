---
title: "Hotspot acceleration: pkg/daemon/pidlock.go"
status: Complete
date: 2026-03-17
---

## TLDR

pidlock.go hotspot is a false positive — the file was born 2026-02-24 (126 lines) as a new PID lock module, and its "+231 lines/30d" metric is just the sum of raw additions across its 21-day lifetime. At 183 lines with single responsibility (flock-based singleton enforcement), no extraction is needed.

## D.E.K.N. Summary

- **Delta:** pidlock.go growth is entirely from file creation + one architectural improvement (PID-file → flock migration). Not accretion.
- **Evidence:** git log shows 126-line birth on Feb 24, then 3 commits: +6 (liveness validation), +78/-47 (flock rewrite), +21 (status fallback). Raw additions total 231, but net growth is 183 lines (current size).
- **Knowledge:** The "+231 lines" metric double-counts the flock rewrite which replaced 47 lines of PID-file logic with 78 lines of flock logic. Birth-within-window files produce inflated hotspot metrics.
- **Next:** No extraction needed. File is well below any threshold (183 vs 800 advisory, 1500 critical).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-daemon-mock-test.md | same pattern | yes | - |

## Question

Is pkg/daemon/pidlock.go a genuine accretion hotspot requiring extraction, or a false positive from recent file creation?

## Findings

### Finding 1: File was born within the 30-day window

Commit `a46656c8f` (2026-02-24) created pidlock.go with 126 lines as a new PID lock module for daemon singleton enforcement. The file is only 21 days old — its entire existence falls within the 30-day measurement window.

### Finding 2: Growth history is healthy architectural improvement

Post-birth commits:
- `8d227c80b` (Feb 27): +6/-1 — add PID liveness validation
- `4cc931648` (Feb 28): +78/-47 — replace PID-file with flock(2) (architectural improvement, not accretion)
- `c992351b2` (Mar 3): +21/-0 — add IsDaemonRunningFromLock fallback for status

The "+231 lines" metric is the sum of raw insertions (126+6+78+21=231), ignoring the 48 deletions. Net growth: 183 lines (current file size).

### Finding 3: Single responsibility, clean API

The file has one job: flock-based PID lock management. It exports:
- `PIDLock` struct + `AcquirePIDLock`/`AcquirePIDLockAt` — acquire lock
- `Release` — release lock
- `ReadPIDFromLockFile`/`ReadPIDFromLockFileAt` — read PID
- `IsDaemonRunningFromLock`/`IsDaemonRunningFromLockAt` — check running status
- `IsProcessAlive` — process liveness check

Used by 7 files across pkg/daemon and cmd/orch. No duplication, no complexity sprawl.

### Finding 4: Size is far below any threshold

- Current: 183 lines
- Advisory threshold: 800 lines
- Critical threshold: 1,500 lines
- Would need 4x growth to reach advisory, 8x for critical

## Test Performed

```bash
# Verified file history
git log --format="%h %as" --numstat -- pkg/daemon/pidlock.go
# Result: 4 commits, 231 insertions, 48 deletions, net 183

# Verified consumer count
grep -rl 'PIDLock\|AcquirePIDLock\|ReadPIDFromLock\|IsDaemonRunning\|IsProcessAlive' pkg/ cmd/ --include='*.go' | grep -v pidlock | wc -l
# Result: 7 consumers

# Verified current size
wc -l pkg/daemon/pidlock.go
# Result: 183 lines
```

## Conclusion

**False positive.** The "+231 lines/30d" metric is inflated by two factors: (1) the file was born 21 days ago so its entire 126-line creation counts as "growth," and (2) the flock rewrite added 78 lines while removing 47, but only additions are counted. At 183 lines with a clean single-responsibility design used by 7 consumers, this requires no extraction.

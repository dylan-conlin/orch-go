---
title: "Hotspot Acceleration: cmd/orch/daemon_loop.go"
status: Complete
date: 2026-03-17
beads_id: orch-go-ro47f
---

## TLDR

daemon_loop.go shows +775 lines/30d but 88% is birth churn from Mar 11 extraction. Only +107 net lines added post-birth across 7 commits. File is 771 lines — not at risk yet but approaching advisory threshold if growth rate continues.

## D.E.K.N. Summary

- **Delta:** daemon_loop.go hotspot is false positive — 89% birth churn from Mar 11 extraction, only +107 net lines post-birth across 7 commits
- **Evidence:** git diff 49bb2f3de..HEAD shows +109/-2 lines; file created at 688 lines, now 771 lines
- **Knowledge:** daemonSetup() (155 lines) is the primary growth vector — each new daemon subsystem adds wiring code there. logDaemonConfig() grows in parallel. Other methods (spawn, completion, invariant) are stable.
- **Next:** No action needed. Monitor daemonSetup() — extract to daemon_wiring.go if it exceeds 250 lines

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

Is daemon_loop.go a real hotspot requiring extraction, or is the +775 lines/30d metric dominated by birth churn from its Mar 11 extraction?

## Findings

### Finding 1: Birth churn dominates

- **Created:** 2026-03-11 via extraction from daemon.go (commit 49bb2f3de)
- **Birth size:** ~688 lines
- **Current size:** 771 lines
- **Post-birth growth:** +109 insertions, -2 deletions = +107 net lines in 6 days
- **Birth churn fraction:** 688/771 = 89% of current content existed at birth
- **Churn fraction of 775-line metric:** 688/775 = 89% birth churn

### Finding 2: Post-birth additions are feature wiring

7 post-birth commits added:
1. Groups-aware project discovery (+35 lines in daemonSetup)
2. --group flag for daemon scoping (part of above)
3. Proactive extraction + trigger scan + digest wiring (+19 lines in daemonSetup)
4. Work graph analysis (+33 lines, new method runWorkGraphAnalysis)
5. Decision protocol logging (+13 lines across spawn + completion)
6. Registry refresh config logging (+11 lines in logDaemonConfig)
7. Trigger expiry wiring (2 lines)

### Finding 3: Growth pattern analysis

Post-birth growth: +107 lines in 6 days = ~18 lines/day
At this rate: would reach 1500 lines in ~41 days (around late April 2026)
However: the growth is episodic (feature wiring), not continuous. Each new daemon subsystem adds ~10-20 lines of wiring code to daemonSetup.

### Finding 4: Structural analysis

The file has clear functional segments:
- `daemonSetup()`: 155 lines — initialization + wiring (growing)
- `logDaemonConfig()`: 70 lines — config logging (growing)
- `processDaemonCompletions()`: 98 lines — completion loop (stable)
- `runDaemonSpawnCycle()`: 135 lines — spawn loop (stable)
- `checkDaemonSignals()`: 25 lines — signal handling (stable)
- `checkVerificationPause()`: 44 lines — pause logic (stable)
- `checkInvariants()`: 82 lines — invariant checking (stable)
- `writeDaemonStatusFile()`: 55 lines — status writing (growing slightly)
- `runWorkGraphAnalysis()`: 24 lines — work graph (new, stable)
- Other: ~83 lines (struct, imports, helpers)

The growing segments are `daemonSetup` and `logDaemonConfig` — both accumulate wiring code for each new daemon subsystem.

## Test Performed

Measured actual birth size vs current size via `git show 49bb2f3de -- cmd/orch/daemon_loop.go | wc -l` and `git diff --stat 49bb2f3de..HEAD -- cmd/orch/daemon_loop.go`.

## Conclusion

**False positive.** 89% of the +775 lines/30d metric is birth churn from the Mar 11 extraction. Post-birth growth is +107 lines across 7 feature additions — moderate but not alarming.

The file is at 771 lines, below the 800-line advisory threshold. However, `daemonSetup()` at 155 lines is the primary growth vector — each new daemon subsystem adds wiring code there. If 5+ more subsystems are added, extraction of setup wiring into a separate `daemon_wiring.go` may be warranted.

**Recommendation:** No extraction needed now. Monitor `daemonSetup()` — if it exceeds 250 lines, extract wiring into a dedicated file.

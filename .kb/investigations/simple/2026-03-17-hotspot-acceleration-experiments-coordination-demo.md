---
title: "Hotspot acceleration: experiments/coordination-demo display_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-jo5bi
---

## TLDR

`experiments/coordination-demo/redesign/results/20260310-174045/no-coord/complex/trial-6/agent-a/display_test.go` hotspot alert (+238 lines/30d, now 238 lines) is a **false positive** — 100% birth churn. File was created from scratch in a single commit on 2026-03-12 as a static coordination demo experiment artifact. Zero subsequent modifications. No extraction needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for coordination-demo experiment result file
- **Evidence:** Git history shows exactly 1 commit (307bbcd0e, 2026-03-12) that created the file. +238 lines = 238 total lines = 100% birth churn. File path is under `experiments/coordination-demo/redesign/results/` — a static experiment output directory.
- **Knowledge:** This is the same false-positive pattern as prior investigations (thread_test.go, control_cmd.go, issue_adapter_test.go). The hotspot detector counts file creation as acceleration. Additionally, experiment result files under `experiments/` are static artifacts that should be excluded from hotspot detection entirely. A prior commit (d840ca481) already documented this exact pattern for another display_test.go in the coordination-demo results.
- **Next:** Close. False positive — static experiment artifact, 100% birth churn. Recommend adding `experiments/` path exclusion to the hotspot detector.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| d840ca481 (commit) | Same file type, same conclusion — coordination-demo display_test.go false positive | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-daemon-issue.md | Same pattern — birth churn false positive | yes | - |

## Question

Is `experiments/coordination-demo/redesign/results/.../display_test.go` (+238 lines/30d, now 238 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File created in single commit, zero modifications

```
$ git log --format="%H %ai %s" --diff-filter=A -- "experiments/.../display_test.go"
307bbcd0e 2026-03-12 20:08:34 -0700 session: 2026-03-12 — enforcement audit framework, daemon Limit fix, deny hooks audit
```

Total commits touching file: 1. The entire 238-line file was created in one commit as part of a session snapshot. No subsequent edits.

### Finding 2: File is a static experiment artifact

The file lives under `experiments/coordination-demo/redesign/results/20260310-174045/no-coord/complex/trial-6/agent-a/` — this is a timestamped experiment result directory. These are static artifacts produced by coordination demo trials, not actively maintained code.

### Finding 3: Prior commit already documented this pattern

Commit d840ca481 (`inv: experiments/coordination-demo display_test.go hotspot is false positive — static experiment artifact, detector lacks path exclusions`) documented the identical finding for a different trial's display_test.go.

## Test performed

Verified via git history that the file has exactly 1 commit (creation) and 0 subsequent modifications. Confirmed file path is under static experiment results directory.

## Conclusion

**False positive.** 100% birth churn from file creation. Static experiment artifact under `experiments/` results directory. The hotspot detector should exclude `experiments/` paths, as these are timestamped trial outputs that are never modified after creation.

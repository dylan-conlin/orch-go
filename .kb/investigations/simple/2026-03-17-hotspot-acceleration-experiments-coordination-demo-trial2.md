---
title: "Hotspot acceleration: experiments/coordination-demo trial-2/agent-a display_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-odw5v
---

## TLDR

`experiments/coordination-demo/redesign/results/20260310-174045/no-coord/complex/trial-2/agent-a/display_test.go` hotspot alert (+225 lines/30d, now 225 lines) is a **false positive** — 100% birth churn. File was created in a single commit on 2026-03-12 as a static coordination demo experiment artifact. Zero subsequent modifications. No extraction needed.

## D.E.K.N. Summary

- **Delta:** Confirmed false positive — same pattern as 2+ prior investigations on coordination-demo display_test.go files
- **Evidence:** Git history shows exactly 1 commit (307bbcd0e, 2026-03-12) creating the file. +225 lines = 225 total lines = 100% birth churn. File is under `experiments/coordination-demo/redesign/results/` — static experiment output.
- **Knowledge:** This is the same false-positive class documented in prior investigations. The hotspot acceleration detector at `pkg/daemon/trigger_detectors_phase2.go:314-347` has no path exclusions, so experiment artifacts are counted as growing code. The `experiments/` directory contains 79+ `.go` files that trigger false positives.
- **Next:** Close. False positive. The detector path-exclusion fix was already recommended in the prior full investigation.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/2026-03-17-inv-hotspot-acceleration-experiments-coordination-demo.md | Same pattern — coordination-demo display_test.go false positive (trial-7/agent-b) | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-experiments-coordination-demo.md | Same pattern — coordination-demo display_test.go false positive (trial-6/agent-a) | yes | - |

## Question

Is `experiments/coordination-demo/redesign/results/20260310-174045/no-coord/complex/trial-2/agent-a/display_test.go` (+225 lines/30d, now 225 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File created in single commit, zero modifications

```
$ git log --diff-filter=A --format='%H %ai %s' -- 'experiments/.../trial-2/agent-a/display_test.go'
307bbcd0e 2026-03-12 20:08:34 -0700 session: 2026-03-12 — enforcement audit framework, daemon Limit fix, deny hooks audit
```

Full commit history shows 3 commits touching the file, all part of initial data capture (the original Go artifact format, then converted). Zero post-creation modifications.

### Finding 2: Static experiment artifact — write-once trial output

Directory contents confirm this is experiment trial output: `stdout.log`, `stderr.log`, `prompt.md`, `start_time`, `end_time`, `duration_seconds`, `commits.txt`, `full_diff.txt`, `build_output.txt`, `all_tests.txt`. The `display_test.go` is code captured from an agent during a coordination demo trial — not production code.

### Finding 3: Pattern already documented and fix recommended

The full investigation at `.kb/investigations/2026-03-17-inv-hotspot-acceleration-experiments-coordination-demo.md` already identified 79 false positives from `experiments/` alone and recommended adding path exclusions to `defaultHotspotAccelerationSource.ListFastGrowingFiles()`.

## Test performed

Verified via `git log --diff-filter=A` that the file has exactly 1 creation commit and `git log --follow` shows no subsequent modifications. Confirmed file is in static experiment results directory by listing sibling files.

## Conclusion

**False positive.** 100% birth churn from file creation. Static experiment artifact under `experiments/` results directory. No extraction needed. The detector path-exclusion fix was already recommended in prior investigation.

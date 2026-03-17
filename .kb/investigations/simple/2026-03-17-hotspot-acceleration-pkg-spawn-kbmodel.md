---
title: "Hotspot acceleration: pkg/spawn/kbmodel.go"
status: Complete
date: 2026-03-17
beads: orch-go-vfnx1
---

## TLDR

`kbmodel.go` hotspot alert (+539 lines/30d, now 530 lines) is a **false positive** caused by birth churn. The file was created on Mar 9 by extracting 14 functions from `kbcontext.go` (which dropped from 1496→1014 lines). Only +37 net lines were added post-birth. No action needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/spawn/kbmodel.go`
- **Evidence:** Git history shows 2 commits: birth extraction (493 lines from kbcontext.go, Mar 9) and one follow-up fix (+46/-9 lines, Mar 10). 93% of the "growth" is birth churn from an intentional extraction refactor.
- **Knowledge:** File was created specifically to reduce kbcontext.go below the 1500-line hotspot threshold. The extraction was successful — kbcontext.go dropped 482 lines. The hotspot acceleration metric doesn't distinguish file birth from organic growth.
- **Next:** Close as false positive. The file is 530 lines, well below the 1500-line threshold. If kbcontext.go continues growing, further extraction may be warranted, but kbmodel.go itself is healthy.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-cmd-orch-control.md | parallel (same false-positive pattern) | yes | - |

## Question

Is `pkg/spawn/kbmodel.go` (+539 lines/30d, now 530 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File created by intentional extraction

Commit `9aee0e599` (Mar 9) created kbmodel.go by extracting 14 functions, 3 types, and 1 const from kbcontext.go. The commit message explicitly states: "Reduces kbcontext.go from 1496 to 1014 lines, bringing it below the 1500-line hotspot threshold."

- kbmodel.go: +493 lines (birth)
- kbcontext.go: -482 lines (extraction source)

### Finding 2: Minimal post-birth growth

Only one subsequent commit `7575dafb2` (Mar 10) touched the file: +46/-9 lines for symlink path resolution fix. Net post-birth growth: +37 lines.

### Finding 3: Current size is healthy

At 530 lines, the file is well below the 1500-line extraction threshold. The file has a coherent responsibility: model parsing, section extraction, and staleness detection for spawn context.

## Test performed

```bash
git log --format="%h %ad %s" --date=short --follow -- pkg/spawn/kbmodel.go
# Output: 2 commits (Mar 9 birth, Mar 10 fix)

git diff 9aee0e599~1..9aee0e599 --numstat -- pkg/spawn/kbmodel.go pkg/spawn/kbcontext.go
# kbcontext.go: 0 added, 482 removed
# kbmodel.go: 493 added, 0 removed

git diff 9aee0e599..7575dafb2 --numstat -- pkg/spawn/kbmodel.go
# 46 added, 9 removed (post-birth delta)
```

## Conclusion

**False positive — birth churn from intentional extraction.** The file was created on Mar 9 to decompose kbcontext.go (which was approaching the 1500-line threshold). 93% of the detected "growth" (493/530 lines) is from file creation, not organic growth. Post-birth net additions are only +37 lines. The file has a well-scoped responsibility and is healthy at 530 lines.

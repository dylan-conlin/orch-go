---
title: "Hotspot acceleration: cmd/orch/review_done.go"
status: Complete
date: 2026-03-17
beads: orch-go-ck9i1
---

## TLDR

`review_done.go` hotspot alert (+464 lines/30d) is a **false positive** caused by file birth via extraction from `review.go` on Mar 10. The 347-line initial extraction accounts for 75% of the insertions metric. Net growth since birth is only +74 lines (3 maintenance commits). File is 421 lines — well below both the 800-line advisory and 1500-line critical thresholds. No action needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `cmd/orch/review_done.go`
- **Evidence:** Git history shows file created Mar 10 via extraction from `review.go` (1353→848 lines); initial creation = 347 lines; 3 subsequent commits added net +74 lines
- **Knowledge:** Hotspot metric counts total insertions including file birth — extraction refactors produce false positives identical to the delete/recreate pattern found in `control_cmd.go`
- **Next:** Close. The parent `review.go` (856 lines) is closer to the 800-line advisory threshold and worth monitoring.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Confirms | yes | - |

## Question

Is `cmd/orch/review_done.go` (+464 lines/30d, now 421 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File born via extraction, not organic growth

Git history for `review_done.go` (last 30 days):

| Commit | Date | Insertions | Deletions | Description |
|---|---|---|---|---|
| 0f887fd | Mar 10 | 347 | 0 | **File creation** — extracted from `review.go` |
| 7192c86 | Mar 10 | 63 | 15 | Add UNTRACKED workspace archival |
| 76c798c | Mar 11 | 33 | 26 | Enrich completion events (LogAgentCompleted) |
| 338f29d | Mar 11 | 21 | 2 | Add ORCH_COMPLETING env var, manifest fallback |
| **Total** | | **464** | **43** | **Net: +421 (but 347 is birth)** |

The 464-line insertion metric is accurate but misleading. 347 lines (75%) are the initial extraction — a healthy refactoring that reduced `review.go` from 1353 to 848 lines. Post-birth growth is only 117 insertions / 43 deletions = **+74 net lines**.

### Finding 2: File structure is healthy

The file contains:
- `reviewDoneCmd` (cobra command definition) — 27 lines
- `runReviewDone()` (main workflow) — 333 lines
- `createFollowUpIssue()` (helper) — 28 lines

While `runReviewDone()` is long at 333 lines, it's a sequential workflow with clear phases:
1. Filter completions by project (lines 60-76)
2. Categorize by verification status (78-90)
3. Display summary and confirm (92-140)
4. Process tracked completions with synthesis prompts (163-318)
5. Archive untracked workspaces (320-362)
6. Print summary (364-391)

Extracting sub-functions would add indirection without meaningful benefit — the phases run sequentially and share local state.

### Finding 3: Parent review.go is the file to watch

The review command family totals 4,059 lines across 10 files. The parent `review.go` sits at 856 lines — above the 800-line advisory but below the 1500-line critical threshold. Future growth there is worth monitoring.

## Test performed

```bash
git log --follow --numstat --since="30 days ago" -- cmd/orch/review_done.go
git log --follow --diff-filter=A -- cmd/orch/review_done.go
wc -l cmd/orch/review*.go
```

Verified commit `0f887fd` is the file's birth (extraction from review.go), and subsequent commits are maintenance — not new feature growth.

## Conclusion

**False positive.** The +464 lines/30d metric is inflated by the file's birth via extraction. Actual post-birth growth is +74 lines across 3 maintenance commits. The file at 421 lines is healthy and well below extraction thresholds.

The extraction that *created* this file was itself the hotspot remedy — `review.go` was at 1353 lines before the refactor.

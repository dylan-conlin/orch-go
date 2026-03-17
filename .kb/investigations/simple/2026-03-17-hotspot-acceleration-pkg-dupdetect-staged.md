---
title: "Hotspot acceleration: pkg/dupdetect/staged_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-bjt6j
---

## TLDR

`staged_test.go` hotspot alert (+257 lines/30d, now 257 lines) is a **false positive** caused by birth churn. The file was created on Mar 11 in a single commit (0518df76c) shipping the duplication detector integration. Zero commits post-birth. No action needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/dupdetect/staged_test.go`
- **Evidence:** Git history shows exactly 1 commit touching this file: the birth commit (0518df76c, Mar 11) which created it with 257 lines as part of the duplication detector integration feature. `git log --numstat --follow` confirms 257 additions, 0 deletions, zero subsequent commits.
- **Knowledge:** File was created as part of the `CheckModifiedFiles` feature that scopes duplicate detection to agent-modified files only. The 257-line size is healthy — well below the 1500-line threshold. The broader dupdetect test surface (1942 lines across 6 files) is well-distributed with no file exceeding 514 lines.
- **Next:** Close as false positive. No post-birth growth trend. No extraction needed.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-account-capacity.md | parallel (same birth-churn false positive pattern) | yes | - |

## Question

Is `pkg/dupdetect/staged_test.go` (+257 lines/30d, now 257 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: Birth churn — single creation commit

`git log --numstat --follow` shows exactly one commit:
- **0518df76c** (Mar 11): `feat: ship duplication detector integration — completion advisory, pre-commit check, modified-file scoping (orch-go-uvd12)`
- 257 lines added, 0 deleted — this is the birth commit

Zero commits since creation. 100% of detected "growth" is birth churn.

### Finding 2: File size is healthy

At 257 lines, the file is well below the 1500-line critical threshold. It contains 7 focused test functions covering:
- Duplication detection (positive and negative cases)
- Modified-file scoping (only reports modified side)
- Cross-directory project detection
- Edge cases (empty modified list, no pairs, formatted advisory)

### Finding 3: Package test surface is well-distributed

| File | Lines |
|---|---|
| allowlist_test.go | 514 |
| scoped_test.go | 513 |
| report_test.go | 280 |
| staged_test.go | 257 |
| dupdetect_test.go | 227 |
| benchmark_real_test.go | 151 |
| **Total** | **1942** |

No individual test file exceeds 514 lines. The package tests are well-factored by concern.

## Test Performed

```bash
git log --numstat --follow --format="%H %ad %s" --date=short -- pkg/dupdetect/staged_test.go
# Result: 1 commit, 257+/0-, no post-birth modifications

git log --oneline --since="30 days ago" -- pkg/dupdetect/staged_test.go
# Result: 1 commit (birth only)

wc -l pkg/dupdetect/*_test.go
# Result: 1942 total, well-distributed across 6 files
```

## Conclusion

**False positive — 100% birth churn.** The file was created Mar 11 as part of a feature integration, with zero post-birth modifications. At 257 lines it is well below any extraction threshold. The broader dupdetect test surface (1942 lines across 6 files) shows healthy distribution with no file approaching hotspot territory.

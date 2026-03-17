---
title: "Hotspot acceleration: pkg/kbgate/model_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-ko01y
---

## TLDR

`pkg/kbgate/model_test.go` hotspot alert (+386 lines/30d, now 386 lines) is a **false positive** caused by birth churn. The file was created on Mar 14 in a single commit and has zero subsequent modifications. No action needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/kbgate/model_test.go`
- **Evidence:** Git history shows exactly 1 commit (`35c8d548`, Mar 14) creating the file. Zero post-birth modifications. 100% of the detected "growth" is birth churn.
- **Knowledge:** The file was created alongside `model.go` (391 lines) as part of the claim-ledger and vocabulary canonicalization gates feature. The test file has a clear structure: 16 test functions covering missing files, missing tables, invalid entries, edge cases, and formatting. At 386 lines with no growth trajectory, it's healthy.
- **Next:** Close as false positive. The file is well below the 1500-line threshold and has no organic growth.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-spawn-kbmodel.md | parallel (same birth-churn pattern) | yes | - |
| 2026-03-17-hotspot-acceleration-experiments-coordination-demo.md | parallel (same birth-churn pattern) | yes | - |

## Question

Is `pkg/kbgate/model_test.go` (+386 lines/30d, now 386 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File created in single commit, never modified

Commit `35c8d548` (Mar 14) created both `model.go` (391 lines) and `model_test.go` (386 lines) as a new feature. No subsequent commits have touched the test file.

### Finding 2: File structure is cohesive

The 386 lines contain 16 test functions testing `CheckModel()` and `FormatModelResult()`:
- 3 helper functions (`assertHasVerdict`, `hasVerdict`, `verdictCodes` — in separate test helper)
- Tests cover: missing file, missing tables, valid tables, invalid entries, evidence requirements, vocabulary inflation, prior art, full pass, endogenous evidence, column validation, JSON output, formatting

### Finding 3: Test pattern has moderate fixture repetition

Each test builds a markdown model string in a temp dir and calls `CheckModel()`. There's a repeating pattern of ~15 lines per test (TempDir, path, content, WriteFile, CheckModel, assert). This is idiomatic Go test style — extracting a helper would save ~5 lines per test but reduce readability. Not worth extracting at current size.

## Test performed

```bash
git log --format="%h %ad %s" --date=short --follow -- pkg/kbgate/model_test.go
# Output: 1 commit (35c8d548, Mar 14)

git diff 35c8d5485~1..35c8d5485 --numstat -- pkg/kbgate/model_test.go
# 386 added, 0 removed (birth)

git log --format="%h %ad %s" --date=short -- pkg/kbgate/
# 7 commits total in kbgate package, only 1 touches model_test.go
```

## Conclusion

**False positive — 100% birth churn.** The file was created on Mar 14 in a single commit and has zero subsequent modifications. All 386 lines of detected "growth" are from file creation, not organic growth. At 386 lines with a cohesive test structure, the file is healthy and well below the 1500-line extraction threshold.

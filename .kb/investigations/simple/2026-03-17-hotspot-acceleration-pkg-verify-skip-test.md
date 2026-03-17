---
title: "Hotspot acceleration: pkg/verify/skip_test.go"
status: Complete
date: 2026-03-17
beads: orch-go-q9ztq
---

## TLDR

`pkg/verify/skip_test.go` hotspot alert (+243 lines/30d, now 236 lines) is a **false positive** — 100% of the growth is from file creation via extraction from `cmd/orch/complete_cmd.go` on 2026-02-28 (243 lines). Only 1 subsequent commit (removed 7 lines on 2026-03-16). File is well-structured at 236 lines with 4 table-driven test functions. No extraction needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/verify/skip_test.go`
- **Evidence:** Git history shows exactly 2 commits: extraction birth (2026-02-28, +243 lines from `complete_cmd.go`/`complete_test.go`) and gate removal (2026-03-16, -7 lines net). The +243 line metric equals the file's initial creation, not incremental bloat.
- **Knowledge:** File was born from extraction (orch-go-fcrf), meaning the hotspot metric is flagging the *result* of a healthy extraction, not dangerous growth. Same false-positive pattern as thread_test.go (orch-go-8801f) and control_cmd.go (orch-go-ru9ux).
- **Next:** Close as false positive. No action needed.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-pkg-thread-thread-test.md | Same pattern — creation-churn false positive | yes | - |
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-control.md | Same pattern — creation-churn false positive | yes | - |

## Question

Is `pkg/verify/skip_test.go` (+243 lines/30d, now 236 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File born from extraction, not organic growth

**Git history (complete):**
- `3c9fc1ece` (2026-02-28): "refactor: extract SkipConfig to pkg/verify/skip.go" — Created file with 243 lines, moved from `cmd/orch/complete_cmd.go` and `cmd/orch/complete_test.go`
- `a40b423d2` (2026-03-16): "feat: remove 3 dormant zero-data completion gates" — Minor modification, net -7 lines

The entire +243 line growth metric is the file's creation via a deliberate extraction refactor. The source commit explicitly states "Phase 1 of complete_cmd.go extraction — reduces complete_cmd.go by ~150 lines."

### Finding 2: File structure is healthy

The 236-line file contains 4 table-driven test functions testing 4 methods on `SkipConfig`:
1. `TestSkipConfigHasAnySkip` (lines 8-64) — 8 test cases
2. `TestSkipConfigSkippedGates` (lines 67-137) — 5 test cases
3. `TestSkipConfigShouldSkipGate` (lines 141-169) — 6 test cases
4. `TestValidateSkipFlags` (lines 172-236) — 7 test cases

Implementation file `skip.go` is 129 lines. Test:production ratio is ~1.8:1, which is healthy for table-driven tests covering all branches.

### Finding 3: No coupling or complexity concerns

- Single import (`testing`)
- No test helpers or fixtures shared with other files
- Each test function is self-contained with its own test table
- No risk of further growth — the `SkipConfig` API surface is stable

## Test Performed

```bash
git log --format="%h %ad %s" --date=short -- pkg/verify/skip_test.go
# a40b423d2 2026-03-16 feat: remove 3 dormant zero-data completion gates (orch-go-pcheo)
# 3c9fc1ece 2026-02-28 refactor: extract SkipConfig to pkg/verify/skip.go (orch-go-fcrf)

git show 3c9fc1ece --stat
# pkg/verify/skip_test.go | 243 ++++++++++++++++++++
# (born from extraction of complete_cmd.go/complete_test.go)

wc -l pkg/verify/skip_test.go
# 236 (7 lines removed since creation)
```

## Conclusion

**False positive.** The +243 lines/30d metric is the file's birth from extraction, not organic growth. File is 236 lines, well-structured, stable API surface, healthy test:production ratio. No extraction needed — this file is itself the *product* of an extraction that reduced `complete_cmd.go` by ~150 lines.

# Investigation: Hotspot acceleration — cmd/orch/e2e_lifecycle_test.go

**TLDR:** False positive. The "+512 lines/30d" metric counts total insertions (churn), not net growth. The file is 415 lines, was created from scratch on Feb 19, and has actually *shrunk* by 49 lines since initial creation. No extraction needed.

**Status:** Complete

## D.E.K.N. Summary

- **Delta:** Hotspot detector's "+512 lines/30d" is a false positive caused by counting total insertions rather than net growth. File was born at 464 lines and is now 415 lines.
- **Evidence:** `git diff --numstat` across all 4 commits shows 512 total insertions / 97 deletions = 415 net lines. The biggest "churn" commit (feed9593c) actually *reduced* the file by 50 lines.
- **Knowledge:** The hotspot detector conflates file creation with accretion. New test files that are born large trigger false positives. The detector should distinguish "born large" from "grew large."
- **Next:** Close as false positive. Optionally create issue to improve hotspot detector to distinguish birth-churn from accretion-churn.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|-------------|----------|-----------|
| .kb/investigations/2026-03-17-inv-hotspot-acceleration-pkg-findingdedup-findingdedup.md | parallel (same detector) | pending | - |

## Question

Is cmd/orch/e2e_lifecycle_test.go a genuine accretion hotspot requiring extraction, or a false positive?

## Findings

### Finding 1: Git history shows file was born large, then shrunk

Per-commit breakdown:

| Commit | Date | Insertions | Deletions | Net | Description |
|--------|------|-----------|-----------|-----|-------------|
| 11dd405b8 | Feb 19 | 464 | 0 | +464 | Initial creation |
| dd6046e7f | Mar 4 | 9 | 9 | 0 | Minor refactor (DefaultDir fix) |
| feed9593c | Mar 4 | 35 | 85 | -50 | Status derivation refactor (shrunk!) |
| e18f63e2b | Mar 10 | 4 | 3 | +1 | Auto-complete fix |

**Total churn:** 512 insertions, 97 deletions → 415 net lines (current file size).

The hotspot detector reported "+512 lines/30d" by summing insertions only. The file is actually *smaller* than when it was created.

### Finding 2: File structure is clean and well-scoped

4 test functions, each covering a distinct lifecycle concern:

1. `TestE2ELifecycle_SingleAgent` (lines 23-149, ~127 lines) — single agent full lifecycle
2. `TestE2ELifecycle_MultipleAgents` (lines 154-307, ~154 lines) — concurrent agents at different stages
3. `TestE2ELifecycle_DegradedModes` (lines 312-384, ~73 lines) — infrastructure failure scenarios
4. `TestE2ELifecycle_LatestPhaseExtraction` (lines 388-415, ~28 lines) — phase comment parsing

No helper functions are duplicated. Tests exercise `pkg/discovery` functions and `cmd/orch.determineAgentStatus`. The file is cohesive — all tests are about the same pipeline.

### Finding 3: 415 lines is well below extraction threshold

The accretion boundary is 1,500 lines. At 415 lines and shrinking trend, this file poses no extraction risk. Even at the historical growth rate, it would take months of sustained new test additions to approach the threshold.

## Test Performed

```bash
# Verified commit history
git log --format="%h %ai %s" -- cmd/orch/e2e_lifecycle_test.go

# Verified per-commit churn
for commit in e18f63e2b feed9593c dd6046e7f 11dd405b8; do
  git diff --numstat ${commit}^..$commit -- cmd/orch/e2e_lifecycle_test.go
done

# Verified total net change
git diff --numstat 11dd405b8^..HEAD -- cmd/orch/e2e_lifecycle_test.go
# Result: 415 insertions, 0 deletions (file didn't exist before)

# Verified current line count
wc -l cmd/orch/e2e_lifecycle_test.go
# Result: 415 lines
```

## Conclusion

**False positive.** The hotspot acceleration detector counts total insertions (churn) rather than net line growth. This file was created from scratch on Feb 19 with 464 lines and has since *shrunk* to 415 lines. The structure is clean, cohesive, and well below the 1,500-line extraction threshold. No action needed on the file itself.

**Detector improvement opportunity:** The hotspot detector should distinguish between "file born large" (initial creation with many lines) and "file grew large" (sustained accretion over time). Birth-churn is not the same signal as accretion-churn.

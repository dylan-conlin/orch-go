---
Status: Complete
Type: investigation
Question: Is review_synthesize_test.go (+271 lines/30d) a real hotspot or birth churn?
---

## TLDR

review_synthesize_test.go is a false positive — 271 lines from a single file-creation commit (7bb5bc4cc, 2026-03-11), not incremental growth. The file is a well-structured test suite at 271 lines with no accretion risk.

## D.E.K.N. Summary

- **Delta:** Confirmed birth churn, not growth pattern
- **Evidence:** Single commit (7bb5bc4cc) created both production and test file together
- **Knowledge:** Hotspot detector flags file birth as growth when entire line count falls within the 30-day window
- **Next:** No action needed — false positive

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/2026-03-15-inv-plist-test-hotspot.md | Extends | yes | - |
| .kb/investigations/2026-03-15-inv-preview-test-hotspot.md | Extends | yes | - |
| .kb/investigations/2026-03-15-inv-thread-test-hotspot.md | Extends | yes | - |

Same pattern as prior hotspot false positives: detector lacks distinction between file birth and incremental growth.

## Finding 1: File History

**Observation:** `git log --oneline --follow` shows exactly ONE commit:

```
7bb5bc4cc feat: add orch review synthesize for batch cross-agent synthesis (orch-go-01k21)
```

Date: 2026-03-11. The entire 271-line file was created in a single commit alongside its production counterpart (review_synthesize.go, 346 lines).

## Finding 2: File Structure Assessment

The 271-line test file contains 6 focused test functions:

| Test | Lines | Purpose |
|---|---|---|
| TestBuildBatchSynthesis | ~104 | Core batch synthesis with 3 agents |
| TestBuildBatchSynthesisEmpty | ~10 | Edge case: no completions |
| TestBuildBatchSynthesisLightTierOnly | ~29 | Edge case: all light tier |
| TestFormatBatchSynthesis | ~64 | Output format verification |
| TestDeduplicateNextActions | ~22 | Deduplication logic |
| TestReviewSynthesizeCommandExists | ~19 | Command registration |

The largest test (TestBuildBatchSynthesis) is dominated by test data setup (CompletionInfo structs with Synthesis fields). This is typical test verbosity, not a code smell.

## Finding 3: Production File Context

The production file (review_synthesize.go) is 346 lines — both files were created together as a feature unit. Combined total: 617 lines for the feature, well within bounds.

## Test Performed

```bash
git log --oneline --follow -- cmd/orch/review_synthesize_test.go
# Result: 1 commit (7bb5bc4cc)

git log --format="%H %ai" -- cmd/orch/review_synthesize_test.go
# Result: 2026-03-11 — single creation date
```

## Conclusion

**False positive.** The 271-line growth signal is entirely from file birth (single commit, 2026-03-11). There is no incremental accretion pattern. The file is well-structured with focused tests and appropriate test data verbosity. No extraction needed.

This is the same pattern seen in 3+ prior investigations (plist_test.go, preview_test.go, thread_test.go): the hotspot detector counts total lines added in the 30-day window without distinguishing birth from growth.

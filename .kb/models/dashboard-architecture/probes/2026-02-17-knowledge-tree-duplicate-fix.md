# Probe: Knowledge Tree Duplicate Items Fix

**Status:** Complete
**Model:** dashboard-architecture
**Date:** 2026-02-17

## Question

How should the knowledge tree handle items that reference multiple parent models in their Prior-Work tables? Should we deduplicate, or select a primary parent?

## What I Tested

1. Examined the tree building code in `pkg/tree/tree.go`
2. Reviewed `cloneNodeRecursiveWithDedup()` function (line 235-286)
3. Analyzed `buildClusterTree()` function (line 127-172)
4. Compared with the previous probe's findings

## What I Observed

**Discovery:** The deduplication fix has ALREADY been implemented!

**Code Analysis:**

1. `buildClusterTree()` creates a `globalIncluded` map (line 149)
2. Uses `cloneNodeForTreeWithDedup()` instead of old `cloneNodeForTree()` (line 157)
3. `cloneNodeRecursiveWithDedup()` marks each node as globally included (line 259)
4. Skips children that are already in `globalIncluded` map (lines 278-280)

**Key Finding:** There are TWO cloning functions:

- `cloneNodeRecursive()` (line 189) - OLD function WITHOUT deduplication
- `cloneNodeRecursiveWithDedup()` (line 235) - NEW function WITH deduplication

The code is currently using the NEW function with deduplication. This suggests either:

1. The fix was implemented after the bug report was filed
2. The fix isn't working correctly and needs testing
3. There's a different code path that's still using the old function

**Next Step:** Need to actually run the dashboard and verify if duplicates still appear

### Test Results

**Test 1: Unit test with synthetic data**

- Created `TestDeduplicationAcrossParents` in `pkg/tree/tree_test.go`
- Simulated an investigation that references two models in Prior-Work table
- Expected: Investigation appears only once in tree
- Result: ✅ PASS - Investigation appeared exactly once (deduplication working)

**Test 2: Real .kb/ data**

- Tested with actual PHASE3_REVIEW.md and PHASE4_REVIEW.md
- Investigation `2026-02-13-inv-audit-model-probe-investigation-claims.md` references both
- Result: ✅ NO DUPLICATES FOUND
  - Investigation appears under PHASE3_REVIEW.md only
  - Does NOT appear under PHASE4_REVIEW.md
  - Deduplication logic is working correctly in production

## Model Impact

**CONFIRMS** the existing probe's analysis, with key clarification:

**Timeline Discovery:**

- Original probe (2026-02-16): Identified root cause - investigations with multiple parents in Prior-Work tables create duplicates
- Fix implemented: 2026-02-16 18:00:49 - Added `cloneNodeRecursiveWithDedup()` function
- Verification (2026-02-17): Confirmed fix is working correctly in production

**Key Finding:** The bug has ALREADY BEEN FIXED.

**Evidence:**

1. Unit test with synthetic data: ✅ PASS - deduplication working
2. Production tree output: ✅ NO DUPLICATES - investigation appears only under PHASE3, not PHASE4
3. Git history: Deduplication logic added in commit 9d84d415 on 2026-02-16

**The Fix:**

- `buildClusterTree()` creates a `globalIncluded` map shared across all root nodes in a cluster
- `cloneNodeRecursiveWithDedup()` marks each node as globally included
- When encountering a child that's already in `globalIncluded`, it's skipped
- This prevents the same child from appearing under multiple parents

**Model Extension:**
The dashboard-architecture model should document that:

- Tree deduplication operates at the CLUSTER level (not cross-cluster)
- First parent wins - child appears under whichever parent is processed first
- This is by design - provides stable, predictable tree structure

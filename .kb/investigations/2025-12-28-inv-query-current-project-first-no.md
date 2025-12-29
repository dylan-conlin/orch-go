## Summary (D.E.K.N.)

**Delta:** The "Query current project first (no --global)" feature is ALREADY IMPLEMENTED in RunKBContextCheck() at pkg/spawn/kbcontext.go:114-155, committed on 2025-12-22 in commit 525e94ca.

**Evidence:** Line 116 calls `runKBContextQuery(query, false)` which runs `kb context` without --global flag. Git history shows commit 525e94ca "feat(spawn): add tiered kb context filtering" explicitly implements this. All related tests pass.

**Knowledge:** The issue orch-go-njht is a duplicate - this work was completed as part of the tiered KB context filtering implementation. Related issues (orch-go-d1qa, orch-go-d3qe, orch-go-e41n) also appear to be already implemented.

**Next:** Close orch-go-njht and related duplicate issues (d1qa, d3qe, e41n) as already implemented. No further work needed.

---

# Investigation: Query Current Project First (no --global)

**Question:** Is the "Query current project first (no --global)" feature implemented, and if so, where?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None - feature already exists
**Status:** Complete

**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Feature already implemented in RunKBContextCheck()

**Evidence:** The function `RunKBContextCheck()` in `pkg/spawn/kbcontext.go` lines 114-155 implements tiered search:
- Line 116: `result, err := runKBContextQuery(query, false)` - queries local project first (no --global)
- Lines 122-139: Only expands to global search if local results are sparse (<3 matches)
- Line 130: Post-filters global results to orch ecosystem repos
- Line 144: Applies per-category limits

**Source:** `pkg/spawn/kbcontext.go:114-155`

**Significance:** The feature described in issue orch-go-njht is 100% implemented. No code changes needed.

---

### Finding 2: Git history confirms implementation date

**Evidence:** 
```
$ git show 525e94ca --stat
commit 525e94caad7675af89aeb4aad54114249de37372
Author: Dylan Conlin <dylan.conlin@gmail.com>
Date:   Mon Dec 22 09:11:08 2025 -0800

    feat(spawn): add tiered kb context filtering with orch ecosystem allowlist
    
    - Query local project first (no --global)
    - Expand to global with ecosystem filter if sparse (<3 matches)
    - Post-filter global results to orch ecosystem repos
    - Apply per-category limit (20) to prevent context flood
```

**Source:** `git log --oneline --all -- pkg/spawn/kbcontext.go`

**Significance:** The commit message explicitly states "Query local project first (no --global)" - this is the exact feature the issue requests. It was implemented on December 22, 2025.

---

### Finding 3: Related issues also appear implemented

**Evidence:** Looking at the in_progress issues:
- `orch-go-d1qa` - "Post-filter results to allowlist if using --global" → Line 130: `filterToOrchEcosystem()`
- `orch-go-d3qe` - "Apply --limit 20 per category" → Line 144: `applyPerCategoryLimits(result.Matches, MaxMatchesPerCategory)`
- `orch-go-e41n` - "If sparse, add orch ecosystem repos" → Lines 122-139: checks sparse, expands to global with filter

**Source:** `bd list --status in_progress` and `pkg/spawn/kbcontext.go`

**Significance:** All four related issues from the tiered filtering work appear to be implemented. They may have been created as tracking issues but the work was completed together in one commit.

---

### Finding 4: Tests verify the implementation

**Evidence:**
```
$ go test ./pkg/spawn/... -run "TestFilterTo|TestApplyPer|TestMerge" -v
=== RUN   TestFilterToOrchEcosystem
--- PASS: TestFilterToOrchEcosystem (0.00s)
=== RUN   TestApplyPerCategoryLimits
--- PASS: TestApplyPerCategoryLimits (0.00s)
=== RUN   TestMergeResults
--- PASS: TestMergeResults (0.00s)
=== RUN   TestMergeResults_NilInputs
--- PASS: TestMergeResults_NilInputs (0.00s)
PASS
```

**Source:** `go test ./pkg/spawn/...`

**Significance:** The filtering and merging logic is tested and working. The implementation is complete and verified.

---

## Synthesis

**Key Insights:**

1. **Issue is a duplicate** - The work described in orch-go-njht was already completed as part of the tiered KB context filtering implementation (commit 525e94ca on 2025-12-22).

2. **Multiple related issues are also duplicates** - Issues orch-go-d1qa, orch-go-d3qe, and orch-go-e41n all describe functionality that exists in the same implementation.

3. **Prior investigation documents the complete implementation** - The investigation file `.kb/investigations/2025-12-22-inv-implement-tiered-kb-context-filtering.md` documents the full implementation with tests.

**Answer to Investigation Question:**

Yes, the "Query current project first (no --global)" feature IS implemented. It was implemented on December 22, 2025 in commit 525e94ca as part of the tiered KB context filtering feature. The implementation is in `RunKBContextCheck()` at `pkg/spawn/kbcontext.go:114-155`, specifically line 116 which calls `runKBContextQuery(query, false)` - the `false` parameter means no `--global` flag is used.

---

## Structured Uncertainty

**What's tested:**

- ✅ `runKBContextQuery(query, false)` runs `kb context` without --global (verified: code inspection at line 166-169 shows cmd construction)
- ✅ Filtering functions work correctly (verified: all unit tests pass)
- ✅ Implementation matches investigation recommendations (verified: compared commit 525e94ca to investigation findings)

**What's untested:**

- ⚠️ End-to-end spawn behavior with the tiered search (not tested in this investigation)

**What would change this:**

- Finding would be wrong if the code at line 116 was different than inspected
- Finding would be wrong if there's a different code path that bypasses RunKBContextCheck()

---

## Implementation Recommendations

### Recommended Approach: Close as Duplicate

**No implementation needed** - The feature already exists.

**Why this approach:**
- Code inspection confirms implementation
- Git history confirms implementation date
- Tests verify functionality

**Action needed:**
1. Close orch-go-njht as duplicate (already implemented)
2. Consider closing related issues: orch-go-d1qa, orch-go-d3qe, orch-go-e41n

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` - Contains RunKBContextCheck() implementation
- `pkg/spawn/kbcontext_test.go` - Contains tests for filtering functions
- `.kb/investigations/2025-12-22-inv-implement-tiered-kb-context-filtering.md` - Prior investigation documenting implementation

**Commands Run:**
```bash
# Check git history
git log --oneline --all -- pkg/spawn/kbcontext.go

# Show implementation commit
git show 525e94ca --stat

# Run tests
go test ./pkg/spawn/... -run "TestFilterTo|TestApplyPer|TestMerge" -v

# Test kb context without --global
kb context "spawn"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-implement-tiered-kb-context-filtering.md` - Original implementation investigation

---

## Investigation History

**2025-12-28 21:39:** Investigation started
- Initial question: Implement "Query current project first (no --global)"
- Context: Follow-up issue from orch-go-gcf8

**2025-12-28 21:45:** Found implementation already exists
- Discovered feature implemented in commit 525e94ca
- Code inspection confirms tiered search with local-first approach

**2025-12-28 21:50:** Investigation completed
- Status: Complete
- Key outcome: Feature already implemented, issue is duplicate

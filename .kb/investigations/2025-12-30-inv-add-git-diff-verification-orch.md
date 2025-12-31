<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added git diff verification gate to `orch complete` - verifies SYNTHESIS.md Delta section claims match actual git changes.

**Evidence:** All tests pass (10 new tests for git_diff.go), build succeeds, integration verified in VerifyCompletionFull().

**Knowledge:** SYNTHESIS.md uses backtick-quoted file paths in Delta section; false positives detected when claimed files are not in git diff since spawn time.

**Next:** Close - feature implemented with tests, ready for use in `orch complete`.

---

# Investigation: Add Git Diff Verification Orch

**Question:** How to add git diff verification to orch complete that gates on SYNTHESIS.md claims matching actual git changes?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SYNTHESIS.md uses consistent Delta section format

**Evidence:** Reviewed sample SYNTHESIS.md files from workspaces. All use "## Delta (What Changed)" section with "### Files Modified" and "### Files Created" subsections. File paths are backtick-quoted (e.g., `` `pkg/verify/check.go` ``).

**Source:** 
- `.orch/workspace/og-debug-fix-beads-database-22dec/SYNTHESIS.md`
- `.orch/workspace/og-work-completion-lifecycle-broken-29dec/SYNTHESIS.md`

**Significance:** Provides reliable parsing target. Backtick-quoted paths are easy to regex match.

---

### Finding 2: Existing verification pattern in pkg/verify/

**Evidence:** pkg/verify/ already has gate functions like `VerifyGitCommitsForCompletion()`, `VerifyVisualVerificationForCompletion()` that follow consistent pattern: return `nil` when verification should be skipped, return `*Result` with `Passed` field when verification applies.

**Source:** `pkg/verify/git_commits.go:168-182`, `pkg/verify/check.go:397-503`

**Significance:** New gate follows established pattern for easy integration.

---

### Finding 3: Spawn time needed for git diff scope

**Evidence:** `spawn.ReadSpawnTime()` already exists and is used by constraint verification. Git diff should only consider changes since spawn time to avoid matching files from prior commits.

**Source:** `pkg/spawn/time.go`, `pkg/verify/constraint.go:131`

**Significance:** Reuse existing infrastructure for time-scoping file changes.

---

## Synthesis

**Key Insights:**

1. **Pattern consistency** - Following existing gate function patterns (VerifyGitCommitsForCompletion, VerifyVisualVerificationForCompletion) ensures predictable integration with VerifyCompletionFull.

2. **Fail on overclaim, warn on underclaim** - If SYNTHESIS claims files not in git diff, that's a false positive (agent lied). If git diff has files not in SYNTHESIS, that's just under-reporting (acceptable).

3. **Time-scoping matters** - Using spawn time prevents matching files from prior agent sessions.

**Answer to Investigation Question:**

Implementation complete. Created `pkg/verify/git_diff.go` with:
- `ParseDeltaFiles()` - extracts file paths from SYNTHESIS.md Delta section
- `GetGitDiffFiles()` - gets files changed since spawn time
- `VerifyGitDiff()` - compares claims vs actual, fails if claims exceed actual
- `VerifyGitDiffForCompletion()` - gate function integrated into `VerifyCompletionFull()`

---

## Structured Uncertainty

**What's tested:**

- ✅ ParseDeltaFiles extracts backtick-quoted and bold file paths (verified: 8 test cases)
- ✅ VerifyGitDiff fails when claims exceed actual (verified: TestVerifyGitDiff_ClaimsFilesNotInDiff)
- ✅ VerifyGitDiff passes when claims match actual (verified: TestVerifyGitDiff_ClaimsMatchDiff)
- ✅ Missing SYNTHESIS.md or empty Delta skips verification gracefully (verified: TestVerifyGitDiff_NoSynthesis, TestVerifyGitDiff_EmptyDelta)

**What's untested:**

- ⚠️ Real-world integration with `orch complete` command (unit tests only)
- ⚠️ Edge case: file renamed but path still in SYNTHESIS.md

**What would change this:**

- If SYNTHESIS.md format changes significantly, ParseDeltaFiles regex would need updating
- If agents start reporting relative vs absolute paths differently, NormalizePath may need expansion

---

## Implementation Recommendations

**Implemented as recommended approach.** See files below.

---

## References

**Files Created:**
- `pkg/verify/git_diff.go` - Core verification logic
- `pkg/verify/git_diff_test.go` - Unit tests

**Files Modified:**
- `pkg/verify/check.go:386-395` - Updated docstring to mention git diff verification
- `pkg/verify/check.go:490-502` - Added VerifyGitDiffForCompletion call in VerifyCompletionFull

**Commands Run:**
```bash
# Verify build
go build ./...

# Run verify package tests
go test ./pkg/verify/... -v

# Run full test suite
go test ./...
```

---

## Investigation History

**2025-12-30 18:55:** Investigation started
- Initial question: How to add git diff verification gate to orch complete
- Context: Need to detect false positives where agent claims file changes not in actual diff

**2025-12-30 19:20:** Implementation complete
- Status: Complete
- Key outcome: Added git diff verification gate that fails when SYNTHESIS.md claims files not in actual git diff

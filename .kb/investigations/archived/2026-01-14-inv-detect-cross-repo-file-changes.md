## Summary (D.E.K.N.)

**Delta:** Cross-repo file verification now uses mtime check instead of git diff for external paths (~/..., /..., ../).

**Evidence:** All 6 new tests pass - IsExternalPath, ExpandPath, VerifyExternalFile, and 3 integration tests.

**Knowledge:** Git diff cannot track files outside the repo; mtime provides a reliable alternative verification method.

**Next:** Close - feature implemented and tested.

**Promote to Decision:** recommend-no (tactical implementation, not architectural)

---

# Investigation: Detect Cross Repo File Changes

**Question:** How should SYNTHESIS.md verification handle file paths that point outside the current git repo?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Agent og-feat-detect-cross-repo-14jan-16c4
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Git diff cannot track external files

**Evidence:** The `git diff --name-only` and `git log --name-only` commands only show files within the repository. Files outside the repo (e.g., `~/other-project/file.ts`) never appear in git diff output.

**Source:** `pkg/verify/git_diff.go:204-241` - GetGitDiffFiles function

**Significance:** Agents working on cross-repo tasks may legitimately modify files in other repos (e.g., skill sources at ~/orch-knowledge). Without special handling, these would trigger false "not in git diff" failures.

---

### Finding 2: Three patterns identify external paths

**Evidence:** External paths follow predictable patterns:
- Home directory: `~/path/to/file.ts`
- Absolute paths: `/Users/dylan/other-project/file.go`
- Parent traversal: `../sibling-repo/file.go` or `path/../../../outside.go`

**Source:** Test cases in `pkg/verify/git_diff_test.go:536-572`

**Significance:** These patterns can be reliably detected with string prefix/contains checks.

---

### Finding 3: File mtime provides verification alternative

**Evidence:** `os.Stat()` returns file modification time. If mtime > spawn time, the file was modified during the agent's session.

**Source:** `pkg/verify/git_diff.go:304-346` - VerifyExternalFile function

**Significance:** This provides equivalent verification semantics to git diff - confirming the agent actually modified the claimed files.

---

## Synthesis

**Key Insights:**

1. **Dual verification strategy** - Local files use git diff (precise, commit-aware), external files use mtime (existence + recency check).

2. **Graceful handling** - External files that exist with recent mtime pass verification. Missing files or old files fail with clear error messages.

3. **Mixed file support** - SYNTHESIS.md can now contain both local and external files; each is verified by the appropriate method.

**Answer to Investigation Question:**

External paths are detected using the `IsExternalPath()` function which checks for `~/`, absolute paths starting with `/`, and parent traversal patterns (`../`). These files are verified using `VerifyExternalFile()` which checks file existence and mtime > spawn time. The verification now handles cross-repo changes without false positives.

---

## Structured Uncertainty

**What's tested:**

- ✅ IsExternalPath correctly identifies ~/..., /..., ../ paths (15 test cases)
- ✅ ExpandPath correctly expands ~/ to home directory
- ✅ VerifyExternalFile validates existence and mtime
- ✅ External files pass verification when they exist and are recent
- ✅ External files fail verification when missing or old
- ✅ Mixed local/external files are verified correctly

**What's untested:**

- ⚠️ Edge case: Windows-style paths (not relevant for current Mac/Linux usage)
- ⚠️ Edge case: Symlinks pointing outside repo

**What would change this:**

- Finding would be incomplete if there are other external path patterns (none identified)
- Mtime approach would be unreliable if files are touched without actual modification (low risk)

---

## Implementation Details

**Files Modified:**
- `pkg/verify/git_diff.go` - Added IsExternalPath, ExpandPath, ExternalFileResult, VerifyExternalFile; updated GitDiffResult struct and VerifyGitDiff function
- `pkg/verify/git_diff_test.go` - Added 6 new test functions

**Changes Made:**
1. Added `IsExternalPath(path string) bool` - detects external paths
2. Added `ExpandPath(path string) (string, error)` - expands ~/ to home dir
3. Added `ExternalFileResult` struct - captures verification details
4. Added `VerifyExternalFile(path, spawnTime) ExternalFileResult` - mtime verification
5. Updated `GitDiffResult` struct with external file tracking fields
6. Updated `VerifyGitDiff()` to separate and verify external files

**Success criteria:**
- ✅ Cross-repo changes don't trigger false 'not in git diff' failures
- ✅ SYNTHESIS.md claims ~/external/file.ts passes if file exists and was recently modified
- ✅ All existing tests still pass

---

## References

**Files Examined:**
- `pkg/verify/git_diff.go` - Main verification logic
- `pkg/verify/git_diff_test.go` - Test patterns
- `pkg/spawn/session.go` - Spawn time reading

**Commands Run:**
```bash
# Build verification
go build ./...

# Run all verify package tests
go test ./pkg/verify/... -v
```

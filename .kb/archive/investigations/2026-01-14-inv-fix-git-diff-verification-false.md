<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** When spawn_time file is missing (old workspaces), git diff verification uses `git diff --name-only HEAD` which only shows uncommitted changes, causing false positives when agents commit their work.

**Evidence:** Found 28 workspaces without .spawn_time files; tested behavior: zero spawn time -> `git diff --name-only HEAD` returns empty for committed changes.

**Knowledge:** Git diff verification cannot work without spawn time because we need a baseline to compare against; failing silently is worse than warning and skipping.

**Next:** Fix implemented and tested - ready for commit.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Fix Git Diff Verification False Positive

**Question:** Why does git diff verification claim "files not in git diff" when commits exist in git log?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-feat-fix-git-diff-14jan-ff08
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Zero spawn time triggers wrong git command

**Evidence:** When `spawn.ReadSpawnTime()` returns zero time (file missing or parse error), `GetGitDiffFiles()` uses `git diff --name-only HEAD` instead of `git log --name-only --since=...`. The former only shows uncommitted changes.

**Source:** `pkg/verify/git_diff.go:210-215`:
```go
if since.IsZero() {
    // Get uncommitted changes
    cmd = exec.Command("git", "diff", "--name-only", "HEAD")
}
```

**Significance:** This is the root cause. If an agent commits their work (normal behavior), `git diff --name-only HEAD` returns empty, causing verification to fail with "files not in git diff" even though the commits exist.

---

### Finding 2: Many workspaces lack spawn_time files

**Evidence:** Found 28 workspaces in `.orch/workspace/` without `.spawn_time` files, including:
- og-arch-de-bloat-feature-22dec
- og-feat-add-lineage-headers-22dec
- og-feat-cleanup-after-orchestrator-23dec
- (and 25 more)

**Source:** `for dir in .orch/workspace/*/; do if [ ! -f "${dir}.spawn_time" ]; then echo "MISSING: $dir"; fi; done`

**Significance:** These are older workspaces created before spawn time tracking was implemented. The verification code assumes spawn time is always available, which breaks for historical workspaces.

---

### Finding 3: Time format is NOT the issue

**Evidence:** Tested RFC3339 formatting with git `--since` flag:
```bash
--since=2026-01-14T21:18:06-08:00
Result: "new.go"  # Correctly found the file
```

**Source:** Test script in /tmp/test_git_time.go

**Significance:** The spawn time format (Unix nanoseconds -> RFC3339) is correctly handled by git. This eliminates time format mismatch as a root cause.

---

## Synthesis

**Key Insights:**

1. **Missing spawn time causes wrong verification mode** - The code has two modes: checking uncommitted changes (zero time) vs checking committed changes since spawn (non-zero time). Old workspaces trigger the wrong mode.

2. **Failing silently is worse than skipping** - When we can't determine spawn time, we should warn and skip verification rather than produce false positives that undermine trust in the verification system.

3. **Path normalization is NOT the issue** - The `NormalizePath()` function correctly handles ./, /, and \ prefixes. Both claimed and actual files are normalized consistently.

**Answer to Investigation Question:**

The false positives occur because workspaces without `.spawn_time` files trigger the wrong git command. `git diff --name-only HEAD` only shows uncommitted changes. Since agents commit their work, this returns empty, and verification incorrectly reports "claimed files not in git diff."

---

## Structured Uncertainty

**What's tested:**

- ✅ Zero spawn time causes `git diff --name-only HEAD` (verified: read source code)
- ✅ Committed changes aren't shown by `git diff --name-only HEAD` (verified: test script)
- ✅ RFC3339 time format works with git --since (verified: test script)
- ✅ Fix passes all existing tests + new regression test (verified: `go test ./pkg/verify/...`)

**What's untested:**

- ⚠️ Edge case: spawn time file exists but is corrupted (not specifically tested, but handled by zero time check)

**What would change this:**

- Finding would be wrong if there's another code path that causes false positives beyond spawn time handling

---

## Implementation Recommendations

### Recommended Approach (IMPLEMENTED)

**Skip verification when spawn time unavailable** - When spawn time is zero, return early with a warning instead of proceeding with unreliable verification.

**Why this approach:**
- Directly addresses root cause (zero spawn time)
- Fails safe - doesn't produce false positives
- Clear warning explains why verification was skipped

**Trade-offs accepted:**
- Old workspaces won't get git diff verification (acceptable - they never had reliable verification anyway)
- Warning might be verbose for historical completions (acceptable - informative is better than silent)

**Implementation:**
```go
if spawnTime.IsZero() {
    result.Warnings = append(result.Warnings,
        "spawn time unavailable (workspace may predate spawn time tracking) - skipping git diff verification")
    return result
}
```

---

## References

**Files Modified:**
- `pkg/verify/git_diff.go` - Added zero spawn time check before git diff
- `pkg/verify/git_diff_test.go` - Added `TestVerifyGitDiff_NoSpawnTime` regression test

**Commands Run:**
```bash
# Check for workspaces missing spawn_time
for dir in .orch/workspace/*/; do
  if [ ! -f "${dir}.spawn_time" ]; then echo "MISSING: $dir"; fi
done

# Run tests
go test -v ./pkg/verify/ -run "GitDiff"
```

---

## Investigation History

**2026-01-14 21:15:** Investigation started
- Initial question: Why do 3 agents have commits but verification claims files not in diff?
- Context: False positives undermine verification system trust

**2026-01-14 21:17:** Root cause identified
- Found zero spawn time causes wrong git command
- Found 28 workspaces without spawn_time files

**2026-01-14 21:20:** Investigation completed
- Status: Complete
- Key outcome: Fixed by skipping verification when spawn time unavailable

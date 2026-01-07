## Summary (D.E.K.N.)

**Delta:** `orch clean --stale` failed silently when archive destination already existed due to `os.Rename` returning "file exists" error.

**Evidence:** Reproduced bug: `os.Rename(src, dest)` returns "file exists" error when dest directory exists. Fix verified: added timestamp suffix when destination exists, preserving both old and new archives.

**Knowledge:** Go's `os.Rename` does NOT overwrite existing directories - must check destination first and handle collision explicitly.

**Next:** Close issue - fix implemented, tested, and verified.

**Promote to Decision:** recommend-no - tactical bug fix, not architectural

---

# Investigation: Bug - orch clean --stale Fails on Duplicate Archive

**Question:** Why does `orch clean --stale` fail silently when archive destination exists?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** agent (orch-go-wgdse)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: os.Rename fails when destination directory exists

**Evidence:** When `os.Rename(source, dest)` is called and dest is an existing non-empty directory, it returns error "rename ... : file exists" on macOS/Linux.

**Source:** `cmd/orch/clean_cmd.go:966` - the archiveStaleWorkspaces function:
```go
if err := os.Rename(ws.path, destPath); err != nil {
    fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", ws.name, err)
    continue  // <-- Workspace left un-archived
}
```

**Significance:** This is the root cause. The error handling printed a warning but continued, leaving the workspace in place (un-archived).

---

### Finding 2: Same issue exists in archiveEmptyInvestigations

**Evidence:** The `archiveEmptyInvestigations` function at line 800 has identical pattern:
```go
if err := os.Rename(path, destPath); err != nil {
    fmt.Fprintf(os.Stderr, "    Warning: failed to archive %s: %v\n", relPath, err)
    continue
}
```

**Source:** `cmd/orch/clean_cmd.go:800`

**Significance:** Both archival functions need the same fix for consistency.

---

### Finding 3: Codebase uses unique suffixes for collision prevention

**Evidence:** The `pkg/spawn/config.go` already has a `generateUniqueSuffix()` function that creates 4-char hex suffixes. Workspace names use format `{prefix}-{skill}-{task}-{date}-{unique}` to prevent collisions.

**Source:** `pkg/spawn/config.go:260-270`

**Significance:** The codebase already has a pattern for handling naming collisions - timestamp suffix is consistent with this approach.

---

## Synthesis

**Key Insights:**

1. **Silent failure pattern** - The bug caused workspaces to remain un-archived without clear feedback about why. Users expected `orch clean --stale` to archive old workspaces but some were silently skipped.

2. **Duplicate archives are legitimate** - A workspace name can exist in archive when: (a) the same task was spawned multiple times on same day, or (b) prior archival was interrupted. Both scenarios should preserve data.

3. **Three options for handling collision:**
   - Option A: Skip with clear message - loses data, but safe
   - Option B: Overwrite/merge - risky, potential data loss
   - Option C: Timestamp suffix - preserves all data, clear naming

**Answer to Investigation Question:**

`orch clean --stale` failed because `os.Rename` returns an error when the destination directory already exists. The error was caught and logged as a warning, but the workspace was left in place. This happened silently (easy to miss in verbose output). The fix adds a timestamp suffix (HHMMSS format) when destination exists, ensuring the archive succeeds while preserving any existing archive.

---

## Structured Uncertainty

**What's tested:**

- ✅ `os.Rename` returns "file exists" error when dest exists (verified: Go test program in /tmp)
- ✅ Fix adds timestamp suffix and archives successfully (verified: unit tests + manual test)
- ✅ Original archive is preserved with its contents (verified: checked old.txt content)
- ✅ New archive is created with unique name (verified: test-duplicate-archive-fresh-154128)

**What's untested:**

- ⚠️ Behavior on filesystem full or permissions errors (not tested)
- ⚠️ Edge case: extremely rapid archiving causing timestamp collision (unlikely but possible)

**What would change this:**

- Finding would be wrong if `os.Rename` behavior differs on other platforms (Windows may behave differently)
- Timestamp suffix could collide if archiving same workspace multiple times within same second (very unlikely)

---

## Implementation Recommendations

**Purpose:** Document the chosen fix approach.

### Recommended Approach ⭐

**Timestamp suffix on collision** - When archive destination exists, add `-HHMMSS` suffix to make it unique.

**Why this approach:**
- Preserves all data (no overwrite, no data loss)
- Clear naming indicates which is newer
- Consistent with existing codebase pattern of unique suffixes
- Simple to implement and test

**Trade-offs accepted:**
- Multiple archives of same workspace can accumulate (acceptable - they're stale anyway)
- Timestamp suffix adds visual noise (acceptable - collision is rare)

**Implementation sequence:**
1. Check if destination exists before rename
2. If exists, generate timestamp suffix (HHMMSS format)
3. Print note about using suffixed name
4. Proceed with rename to suffixed path

### Alternative Approaches Considered

**Option A: Skip with clear message**
- **Pros:** Safe, simple
- **Cons:** Workspace remains un-archived (the bug symptom continues, just with better messaging)
- **When to use instead:** If data preservation of new archive is not important

**Option B: Overwrite/merge**
- **Pros:** Clean archive directory (no duplicates)
- **Cons:** Potential data loss if old archive had valuable content
- **When to use instead:** Never - too risky for a cleanup command

**Rationale for recommendation:** Option C (timestamp suffix) is the only approach that both fixes the bug AND preserves all data.

---

### Implementation Details

**Files modified:**
- `cmd/orch/clean_cmd.go` - archiveStaleWorkspaces and archiveEmptyInvestigations functions

**Tests added:**
- `TestArchiveStaleWorkspacesHandlesDuplicateDestination`
- `TestArchiveEmptyInvestigationsHandlesDuplicateDestination`

**Success criteria:**
- ✅ `orch clean --stale` archives workspace even when destination exists
- ✅ Original archive preserved
- ✅ New archive has unique timestamped name
- ✅ Clear output message about using suffixed name

---

## References

**Files Examined:**
- `cmd/orch/clean_cmd.go` - Main implementation file
- `cmd/orch/clean_test.go` - Test file
- `pkg/spawn/config.go` - Reference for suffix pattern

**Commands Run:**
```bash
# Reproduce bug
go run /tmp/test-rename.go
# Output: Error: rename ... : file exists

# Verify fix
/tmp/test-orch clean --stale --stale-days 7
# Output: Note: Archive destination exists, using: test-duplicate-archive-fresh-154128
```

---

## Investigation History

**2026-01-07 15:35:** Investigation started
- Initial question: Why does orch clean --stale fail silently when archive destination exists?
- Context: Bug reported in orch-go-wgdse

**2026-01-07 15:38:** Root cause identified
- `os.Rename` returns "file exists" error on destination collision
- Both workspace and investigation archival affected

**2026-01-07 15:40:** Fix implemented
- Added destination check before rename
- Added timestamp suffix generation
- Applied to both archiveStaleWorkspaces and archiveEmptyInvestigations

**2026-01-07 15:41:** Fix verified
- Unit tests pass
- Manual test confirms fix works
- All tests pass

**2026-01-07 15:42:** Investigation completed
- Status: Complete
- Key outcome: Bug fixed by adding timestamp suffix when archive destination exists

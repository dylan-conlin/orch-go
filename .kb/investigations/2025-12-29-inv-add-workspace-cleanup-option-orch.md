## Summary (D.E.K.N.)

**Delta:** Added workspace cleanup option to `orch clean` command with `--workspaces`, `--older-than`, and `--force` flags.

**Evidence:** All new tests pass; manual testing shows workspaces listed by age with size, active sessions skipped.

**Knowledge:** Workspace cleanup requires checking OpenCode sessions to avoid deleting active work.

**Next:** Close issue - implementation complete.

---

# Investigation: Add Workspace Cleanup Option Orch

**Question:** How to add workspace cleanup functionality to `orch clean` command?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing clean command structure

**Evidence:** The clean command already supports multiple cleanup actions via flags (`--windows`, `--phantoms`, `--verify-opencode`), making it natural to add `--workspaces` flag.

**Source:** cmd/orch/main.go:3824-3860

**Significance:** Pattern exists for adding new cleanup actions.

---

### Finding 2: Active session detection exists

**Evidence:** OpenCode client has `ListSessions()` method that returns active sessions with IDs. Workspace directories store session IDs in `.session_id` files.

**Source:** cmd/orch/main.go:4172-4181, pkg/spawn/spawn.go (ReadSessionID)

**Significance:** Can cross-reference to skip workspaces with active agents.

---

### Finding 3: Workspace age calculation

**Evidence:** Need to walk workspace files to get accurate "last modified" time since directory mod time doesn't reflect file activity.

**Source:** Implementation uses `filepath.Walk` to find latest mod time within workspace.

**Significance:** More accurate than just checking directory timestamp.

---

## Implementation Summary

Added to `orch clean`:
- `--workspaces` flag: Enable workspace directory cleanup
- `--older-than N` flag: Set age threshold in days (default 7)  
- `--force` flag: Skip confirmation prompt

Features:
1. Lists workspaces by age with size and last modified time
2. Skips workspaces with active OpenCode sessions
3. Requires confirmation unless `--force` (or `--dry-run`)
4. Tabular output showing name, size, date, and age

Example usage:
```bash
orch clean --workspaces                    # Remove workspaces >7 days old
orch clean --workspaces --older-than 14    # Remove workspaces >14 days old
orch clean --workspaces --dry-run          # Preview what would be deleted
orch clean --workspaces --force            # Skip confirmation
```

---

## References

**Files Modified:**
- cmd/orch/main.go - Added flags, `cleanOldWorkspaces()`, `formatBytes()`
- cmd/orch/clean_test.go - Added tests for new functionality

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/

# Test execution
go test ./cmd/orch/ -run "TestClean|TestFormatBytes|TestWorkspace" -v

# Manual testing
go run ./cmd/orch/ clean --workspaces --older-than 2 --dry-run
```

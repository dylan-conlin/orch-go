## Summary (D.E.K.N.)

**Delta:** Merged phantom/ghost/orphan into unified "stale infrastructure" concept, reduced orch clean from 7 action flags to 3 (--workspaces, --sessions, --all).

**Evidence:** go build passes, go vet passes, 17 clean-related tests pass, orch clean --help shows 3 action flags.

**Knowledge:** The three ghost types (phantom windows, ghost registry entries, orphaned sessions) all stem from one root cause: OpenCode sessions persist indefinitely. Unifying them into --sessions makes the CLI coherent without losing underlying safety logic.

**Next:** Close. Phase 2 (eliminate registry) can follow as separate work.

**Authority:** implementation - Tactical simplification within existing patterns per lifecycle-ownership-boundaries decision.

---

# Investigation: Simplify orch clean - merge phantom/ghost/orphan

**Question:** How to simplify orch clean from 7 flags to 3 by merging phantom/ghost/orphan into a single concept?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md (Phase 1 implementation)

## Findings

### Finding 1: Flag mapping is clean

**Evidence:** The 7 old action flags map cleanly to 2 new flags:
- `--workspaces` = `--stale` + `--investigations` (both archive filesystem artifacts)
- `--sessions` = `--phantoms` + `--ghosts` + `--verify-opencode` + `--sessions` + `--windows` (all clean stale infrastructure)
- `--all` = both

**Source:** cmd/orch/clean_cmd.go

**Significance:** No functionality lost. The underlying functions remain unchanged; only the entry points are unified.

### Finding 2: Function renames eliminate ghost/phantom/orphan vocabulary

**Evidence:** Renamed functions and variables:
- `cleanPhantomWindows` → `cleanStaleTmuxWindows`
- `purgeGhostAgents` → `cleanInactiveRegistryEntries`
- `cleanOrphanedDiskSessions` → `cleanUntrackedDiskSessions`
- Internal variables: `phantomWindows` → `staleWindows`, `ghosts` → `inactive`, `orphanedSessions` → `untrackedSessions`

**Source:** cmd/orch/clean_cmd.go

**Significance:** Code and user-facing output no longer use phantom/ghost/orphan terminology. Replaced with descriptive terms: "stale", "inactive", "untracked".

### Finding 3: Days flags simplified

**Evidence:** Renamed `--stale-days` → `--workspace-days` and `--sessions-days` → `--session-days` to align with the new action flag names.

**Source:** cmd/orch/clean_cmd.go init()

**Significance:** Flag names are now self-documenting: `--workspace-days` clearly modifies `--workspaces`.

## References

**Files Modified:**
- `cmd/orch/clean_cmd.go` - Main refactoring (flags, runClean, function renames)
- `cmd/orch/clean_test.go` - Updated TestCleanAllFlagLogic
- `.kb/guides/opencode.md` - Updated flag references
- `.kb/guides/cli.md` - Updated flag references
- `.kb/guides/workspace-lifecycle.md` - Updated flag references
- `.kb/guides/completion.md` - Updated flag references

**Commands Run:**
```bash
go build ./cmd/orch/
go vet ./cmd/orch/
go test ./cmd/orch/ -run "TestClean|TestArchive|TestSession|TestPreserve|TestGetProject|TestIsOrchestrator"
go run ./cmd/orch/ clean --help
```

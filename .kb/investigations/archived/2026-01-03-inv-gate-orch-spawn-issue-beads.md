## Summary (D.E.K.N.)

**Delta:** Added dependency gating to `orch spawn --issue` that blocks spawning when dependent issues are still open.

**Evidence:** Test with issue orch-go-8or1 (depends on orch-go-jdxy) correctly showed: "orch-go-8or1 is blocked by: orch-go-jdxy (in_progress)". Unit tests pass for parsing and blocking logic.

**Knowledge:** Dependencies are returned as full Issue objects with `dependency_type` field by `bd show --json`. Gate at spawn time prevents wasted agent cycles; `--force` provides escape hatch.

**Next:** Close - implementation complete, tests pass, feature works as specified in beads issue.

---

# Investigation: Gate Orch Spawn Issue Beads

**Question:** How to prevent `orch spawn --issue` from spawning agents when the issue has unresolved blocking dependencies?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Dependencies returned as full Issue objects

**Evidence:** `bd show orch-go-8or1 --json` returns:
```json
"dependencies": [
  {
    "id": "orch-go-jdxy",
    "title": "Gate orch spawn on beads dependency check",
    "status": "in_progress",
    "dependency_type": "blocks"
  }
]
```

**Source:** `cmd/orch/spawn_cmd.go`, `pkg/beads/types.go`

**Significance:** Can check dependency status directly without separate lookups per dependency.

---

### Finding 2: Issue struct uses json.RawMessage for flexibility

**Evidence:** The `Issue.Dependencies` field uses `json.RawMessage` to handle different formats (string arrays, nested objects).

**Source:** `pkg/beads/types.go:138-147`

**Significance:** Added `ParseDependencies()` and `GetBlockingDependencies()` methods to extract and filter dependencies properly.

---

### Finding 3: Daemon already skips "blocked" status issues

**Evidence:** `daemon.go:317-323` checks `issue.Status == "blocked"` and skips those issues.

**Source:** `pkg/daemon/daemon.go`

**Significance:** Extended the daemon to also check for blocking dependencies using the same `CheckBlockingDependencies()` function.

---

## Implementation Details

**Files Changed:**

1. `cmd/orch/spawn_cmd.go`:
   - Added `--force` flag
   - Added dependency check before spawn (when `--issue` provided and not `--force`)
   - Updated help text with dependency checking documentation

2. `pkg/beads/types.go`:
   - Added `Dependency` struct for parsing dependency data
   - Added `BlockingDependency` struct for representing blockers
   - Added `Issue.ParseDependencies()` method
   - Added `Issue.GetBlockingDependencies()` method

3. `pkg/beads/client.go`:
   - Added `CheckBlockingDependencies()` function (RPC with CLI fallback)
   - Added `BlockingDependencyError` type with formatted error message

4. `pkg/daemon/daemon.go`:
   - Added dependency check in `NextIssueExcluding()` loop
   - Logs skipped issues with verbose flag

5. `pkg/beads/client_test.go`:
   - Added `TestParseDependencies`
   - Added `TestGetBlockingDependencies`
   - Added `TestBlockingDependencyError`

**Behavior:**

```
$ orch spawn feature-impl "task" --issue orch-go-blocked
Error: orch-go-blocked is blocked by: orch-go-dep1 (open), orch-go-dep2 (in_progress)
Use --force to override

$ orch spawn feature-impl "task" --issue orch-go-blocked --force
# Proceeds with spawn despite blockers
```

**Daemon behavior:**
- Silently skips blocked issues
- With `--verbose`: logs "Skipping X (blocked by dependencies: ...)"

---

## Success Criteria

- [x] `orch spawn --issue` blocks when dependencies are open
- [x] `orch spawn --issue --force` bypasses the check
- [x] Daemon skips blocked issues silently
- [x] Error message shows which issues are blocking
- [x] All tests pass

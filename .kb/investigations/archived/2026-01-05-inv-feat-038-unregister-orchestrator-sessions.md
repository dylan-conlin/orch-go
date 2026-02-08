## Summary (D.E.K.N.)

**Delta:** Orchestrator sessions are now unregistered from ~/.orch/sessions.json when `orch complete` is run.

**Evidence:** Added registry.Unregister() call in complete_cmd.go; 3 new tests pass confirming cleanup, ErrSessionNotFound handling, and empty registry handling.

**Knowledge:** Legacy workspaces (spawned before registry existed) return ErrSessionNotFound which is handled gracefully with an informational message.

**Next:** Close - implementation complete with tests passing.

---

# Investigation: Feat 038 Unregister Orchestrator Sessions

**Question:** How should `orch complete` remove orchestrator sessions from the session registry?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Registry Already Has Unregister Method

**Evidence:** `pkg/session/registry.go:197-212` has `Unregister(workspaceName string)` that removes sessions by workspace name.

**Source:** pkg/session/registry.go:197-212

**Significance:** No new registry code needed - just call the existing method from complete_cmd.go.

---

### Finding 2: Complete Command Has Orchestrator Detection

**Evidence:** `complete_cmd.go` already detects orchestrator sessions via `isOrchestratorSession` bool and has `agentName` (workspace name) available.

**Source:** cmd/orch/complete_cmd.go:103-113

**Significance:** Integration point is clear - add Unregister call in the orchestrator session branch after completion.

---

### Finding 3: Legacy Workspaces Need Graceful Handling

**Evidence:** Issue description specifies "Handle case where session isn't in registry (legacy workspaces with .beads_id)".

**Source:** Beads issue orch-go-meie description

**Significance:** `ErrSessionNotFound` should log a note, not fail the completion.

---

## Implementation

Added to `cmd/orch/complete_cmd.go`:
1. Import `github.com/dylan-conlin/orch-go/pkg/session`
2. After "Completed orchestrator session" message, call `registry.Unregister(agentName)`
3. Handle `ErrSessionNotFound` with informational message (legacy workspace)
4. Handle other errors with warning (non-blocking)

Added tests to `cmd/orch/complete_test.go`:
- `TestRegistryCleanupOnCompletion` - verifies session removed from registry
- `TestRegistryCleanupSessionNotFound` - verifies graceful handling of non-existent session
- `TestRegistryCleanupEmptyRegistry` - verifies graceful handling of empty/missing registry file

---

## References

**Files Modified:**
- cmd/orch/complete_cmd.go - Added session import and Unregister call
- cmd/orch/complete_test.go - Added 3 registry cleanup tests

**Tests Run:**
```bash
go test -v ./cmd/orch/... -run "Complete|Registry"
# All tests pass

go test ./pkg/session/... -v
# 23 tests pass

go build -o /dev/null ./cmd/orch/...
# Build succeeds
```

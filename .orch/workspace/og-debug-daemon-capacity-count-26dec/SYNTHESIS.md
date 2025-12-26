# Session Synthesis

**Agent:** og-debug-daemon-capacity-count-26dec
**Issue:** orch-go-s2j7
**Duration:** 2025-12-26
**Outcome:** success

---

## TLDR

Fixed daemon capacity tracking bug where the WorkerPool's internal count became stale after agents completed, blocking new spawns. Added `Pool.Reconcile()` method that syncs with actual OpenCode sessions on each poll cycle.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/pool.go` - Added `Reconcile(actualCount int) int` method to sync internal count with external reality
- `pkg/daemon/daemon.go` - Added `ReconcileWithOpenCode() int` method that calls DefaultActiveCount and reconciles pool
- `cmd/orch/daemon.go` - Added call to `ReconcileWithOpenCode()` at start of each poll cycle with verbose logging
- `pkg/daemon/pool_test.go` - Added 7 tests for Reconcile behavior (stale slots, all gone, more actual, same count, empty pool, wakes waiters)
- `pkg/daemon/daemon_test.go` - Added 2 tests for ReconcileWithOpenCode (no pool, with pool)

### Commits
- Pending commit with fix and tests

---

## Evidence (What Was Observed)

- `pkg/daemon/daemon.go:450-486` - `Once()` acquires slots via `pool.TryAcquire()` but never releases them
- `pkg/daemon/daemon.go:400-426` - `DefaultActiveCount()` exists and correctly queries OpenCode API
- `pkg/daemon/daemon.go:220-238` - `AtCapacity()` uses Pool.Active() when pool exists, bypassing accurate count
- Pool's internal `activeCount` only ever increases, never decreases for completed agents

### Tests Run
```bash
go test ./pkg/daemon/... -v -run 'Reconcile'
# All 8 new tests pass

go test ./pkg/daemon/...
# PASS: ok  github.com/dylan-conlin/orch-go/pkg/daemon  0.152s

go build ./cmd/orch
# Builds successfully
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md` - Full investigation with D.E.K.N. summary

### Decisions Made
- Decision: Use simple reconciliation over CompletionService integration because it's minimal code change, uses existing infrastructure, and is self-healing
- Decision: Reconcile at start of poll cycle (not end) so stale capacity is cleared before trying to spawn

### Constraints Discovered
- Long-running daemons must periodically reconcile internal state with external sources of truth
- OpenCode sessions are the source of truth for active agents, not daemon's internal tracking

### Externalized via `kn`
- None needed - this is a bug fix, not a new constraint or decision pattern

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (8 new tests, all daemon tests green)
- [x] Investigation file has `**Status:** Complete`
- [ ] Ready for `orch complete orch-go-s2j7`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should CompletionService be integrated for real-time tracking? (not needed for current use case)
- Should we add metrics/logging for reconciliation events in production? (could help debugging)

**Areas worth exploring further:**
- Behavior when OpenCode API is temporarily unavailable (currently returns 0, may cause premature slot release)

**What remains unclear:**
- Whether overnight runs will surface edge cases not covered by tests

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-daemon-capacity-count-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md`
**Beads:** `bd show orch-go-s2j7`

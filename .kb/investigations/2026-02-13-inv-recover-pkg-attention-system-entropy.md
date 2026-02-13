## Summary (D.E.K.N.)

**Delta:** pkg/attention/ (26 files, 11 collectors) successfully recovered from commit 6766baaa before bd sync wipe at 3b004bef.

**Evidence:** `go build ./cmd/orch/` passes, `go vet` clean, `go test ./pkg/attention/` all 80+ tests pass.

**Knowledge:** Two interface mismatches required adaptation: ListArgs.ClosedAfter (removed, filter client-side) and Dependency.IssueType (removed, updated logic to check status only). GetBlockingDependencies needed "answered" status treated as non-blocking.

**Next:** Close issue. Frontend recovery (web/src/lib/stores/attention.ts) is orch-go-7 scope.

**Authority:** implementation - Recovery within existing patterns, minimal interface adaptation.

---

# Investigation: Recover Pkg Attention System from Entropy Spiral

**Question:** Can pkg/attention/ be recovered from pre-wipe commits and adapted to current interfaces?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Package intact at commit 6766baaa

**Evidence:** `git ls-tree -r --name-only 6766baaa -- pkg/attention/` lists 26 files. All extracted successfully.

**Source:** Commit 6766baaa (last commit before bd sync wipe at 3b004bef)

**Significance:** Full package recoverable including all collectors and tests.

---

### Finding 2: Two interface mismatches with current beads package

**Evidence:**
- `beads.ListArgs` no longer has `ClosedAfter` field (used in recently_closed_collector.go)
- `beads.Dependency` no longer has `IssueType` field (used in unblocked_collector.go and test)

**Source:** pkg/beads/types.go current state vs recovered code expectations

**Significance:** Required adaptation: client-side filtering for closed issues, status-only checks for resolved dependencies.

---

### Finding 3: GetBlockingDependencies needed "answered" status support

**Evidence:** `GetBlockingDependencies()` only treated `closed` as non-blocking. With `IssueType` removed from Dependency, `answered` questions were incorrectly treated as blocking.

**Source:** pkg/beads/types.go:212 - `isBlocking = dep.Status != "closed"`

**Significance:** Added `&& dep.Status != "answered"` to correctly handle question dependencies.

---

## Files Recovered

- pkg/attention/ (26 files - types, 11 collectors with tests)
- cmd/orch/serve_attention.go (API endpoints)

## Adaptations Made

1. recently_closed_collector.go: Removed `ClosedAfter` from ListArgs, filter client-side
2. unblocked_collector.go: Removed `IssueType` check, use status-only resolution logic
3. unblocked_collector_test.go: Removed `IssueType` from Dependency literals (3 occurrences)
4. pkg/beads/types.go: Added `answered` as non-blocking status in GetBlockingDependencies
5. cmd/orch/serve.go: Added attention import, globalLikelyDoneCache var, cache init, route registrations

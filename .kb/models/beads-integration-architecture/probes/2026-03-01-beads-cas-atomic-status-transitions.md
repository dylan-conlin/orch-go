# Probe: Beads CAS Support for Atomic Status Transitions

**Model:** Beads Integration Architecture
**Date:** 2026-03-01
**Status:** Complete

---

## Question

Does beads support (or can it support) CAS (compare-and-swap) semantics for status transitions? Specifically: can the daemon atomically transition an issue from `open` → `in_progress` such that only one concurrent caller succeeds, eliminating the race window in the current 6-layer dedup gauntlet?

The model claims beads uses "plain unconditional UPDATE within a database transaction" with no conditional status checks. This probe tests that claim and evaluates CAS feasibility.

---

## What I Tested

### 1. Full update chain trace (beads source at ~/Documents/personal/beads)

Traced the complete status update path from orch-go daemon through beads:

```
orch-go daemon (pkg/daemon/daemon.go:767)
  → statusUpdater.UpdateStatus(issue.ID, "in_progress")
    → beads.Client.Update(&UpdateArgs{ID: id, Status: &status})
      → RPC OpUpdate to beads daemon socket (.beads/bd.sock)
        → handleUpdate() in internal/rpc/server_issues_epics.go:475
          → store.UpdateIssue(ctx, id, updates, actor)
            → queries.go:892: UPDATE issues SET status = ? WHERE id = ?
```

Verified the SQL WHERE clause at `queries.go:892` and `transaction.go:421`:
```go
query := fmt.Sprintf("UPDATE issues SET %s WHERE id = ?", strings.Join(setClauses, ", "))
```

**No `AND status = ?` condition exists anywhere in the chain.**

### 2. SQLite CAS feasibility test (sqlite3 CLI)

```bash
sqlite3 :memory: < /tmp/cas_test.sql
```

Tested conditional UPDATE with `WHERE id = ? AND status = ?`:
- CAS 1 (open→in_progress): `changes()=1` (success)
- CAS 2 (open→in_progress): `changes()=0` (rejected — status already changed)
- RETURNING clause works (SQLite 3.43.2 supports it)

### 3. Concurrent CAS test (Go, ncruces/go-sqlite3 v0.30.4 — same driver as beads)

```bash
cd /tmp/beads_cas_test && go run main.go
```

Wrote and ran a Go test with the exact SQLite driver beads uses, testing:
- **Sequential CAS**: First caller wins, second gets `success=false`
- **Concurrent CAS (10 goroutines)**: Exactly 1 winner out of 10
- **CAS with BEGIN IMMEDIATE transactions**: Exactly 1 winner out of 5
- **Current beads pattern (unconditional)**: ALL 5 callers succeed — the race

### 4. Verified beads already has ErrConflict sentinel

```go
// internal/storage/sqlite/errors.go:17-18
ErrConflict = errors.New("conflict")

// errors.go:54-56
func IsConflict(err error) bool {
    return errors.Is(err, ErrConflict)
}
```

### 5. Verified UpdateArgs has no ExpectedStatus field

Checked both beads-side (`internal/rpc/protocol.go:114-151`) and orch-go-side (`pkg/beads/types.go:294-310`) — neither has any conditional/expected status field.

---

## What I Observed

### Observation 1: Current beads updates are unconditional (CONFIRMED)

The UPDATE query in both `queries.go:892` and `transaction.go:421` uses only `WHERE id = ?`. There is no status precondition anywhere in the chain:
- No `AND status = ?` in SQL
- No status check between GetIssue and UPDATE (TOCTOU gap)
- No `ExpectedStatus` field in UpdateArgs (RPC or CLI)
- `bd update --help` shows no `--if-status` or `--expect-status` flag

The model's claim that "UpdateIssue is a plain unconditional UPDATE" is accurate.

### Observation 2: SQLite CAS works perfectly with beads' driver

Test output with ncruces/go-sqlite3 v0.30.4 (exact same driver):

```
=== Test 2: Concurrent CAS (10 goroutines) ===
  Goroutine 3: WON the CAS
  Total winners: 1 (should be exactly 1)
  Final status: in_progress

=== Test 4: Current beads pattern (unconditional) - shows the race ===
  Total 'successes': 5 (ALL succeed - no CAS protection!)
  This is the current beads behavior: multiple callers all think they won
```

CAS via `WHERE id = ? AND status = ?` + `RowsAffected() == 0` check provides atomic mutual exclusion. Exactly 1 of N concurrent callers wins; the rest get `RowsAffected=0` and can detect the failure.

### Observation 3: Beads already has the error infrastructure for CAS

`ErrConflict` and `IsConflict()` exist in `internal/storage/sqlite/errors.go`. CAS failure can wrap this existing sentinel:

```go
return fmt.Errorf("status mismatch on %s: expected %q: %w", id, expectedStatus, ErrConflict)
```

Callers can use `IsConflict(err)` to distinguish CAS failure from other errors.

### Observation 4: Change surface is small and backward-compatible

All changes are additive (optional field):

| Layer | File | Change |
|-------|------|--------|
| RPC protocol | `internal/rpc/protocol.go:114` | Add `ExpectedStatus *string` to UpdateArgs |
| RPC handler | `internal/rpc/server_issues_epics.go:515-520` | Extract ExpectedStatus, pass to storage |
| SQLite storage | `internal/storage/sqlite/queries.go:892` | Add `AND status = ?` to WHERE when ExpectedStatus set |
| SQLite storage | `internal/storage/sqlite/transaction.go:421` | Same change as queries.go |
| Memory storage | `internal/storage/memory/memory.go:443` | Add status check before update |
| CLI | `cmd/bd/update.go` | Add `--expect-status` flag |
| orch-go types | `pkg/beads/types.go:294` | Add `ExpectedStatus *string` to UpdateArgs |

**Estimated LOC**: ~80-120 lines added across ~7 files. No interface changes needed (updates map is already flexible).

### Observation 5: The TOCTOU gap in current code is real

The current `UpdateIssue` path does `GetIssue` (read current status) then `UPDATE ... WHERE id = ?` (write new status) with no atomicity between them. In the daemon's L5 fresh-status-check + L6 UpdateStatus pattern:

1. L5 reads issue status (fail-open if beads unavailable)
2. [TOCTOU gap — another process can change status here]
3. L6 writes `in_progress` unconditionally

Both processes reach L6, both succeed, both spawn. This is the documented race from the structural review investigation.

---

## Model Impact

- [x] **Confirms** invariant: "UpdateIssue is a plain unconditional UPDATE" — verified at queries.go:892 and transaction.go:421, no status precondition exists
- [x] **Extends** model with: CAS is fully feasible with beads' existing infrastructure (SQLite driver supports it, ErrConflict sentinel exists, change is backward-compatible ~80-120 LOC across 7 files). The model should document that beads CAN support atomic status transitions via optional `ExpectedStatus` field — this is a prerequisite for the daemon dedup redesign.

---

## Notes

### Implementation Priority

This probe was commissioned as a prerequisite for the daemon dedup redesign (structural review: `2026-03-01-inv-structural-review-daemon-dedup-after.md`). The redesign proposes replacing the 6-layer dedup gauntlet with:

- **Phase 1 (structural gate)**: Atomic `open → in_progress` CAS in beads — this probe proves it's feasible
- **Phase 2 (advisory only)**: Demote L1-L4 to warning-only monitoring

### The Minimal CAS Path

The simplest implementation that unblocks daemon redesign:

1. Add `ExpectedStatus *string` to beads UpdateArgs (RPC + CLI)
2. When set, modify SQL to `WHERE id = ? AND status = ?`
3. Return `ErrConflict` when RowsAffected == 0
4. orch-go daemon sets `ExpectedStatus: "open"` when calling UpdateStatus

This gives exactly-once spawn semantics: one daemon poll wins the CAS, all others get `ErrConflict` and skip.

### Cross-Project Consideration

The daemon's `UpdateBeadsStatusForProject()` (daemon.go:702-706) handles cross-project updates via `--workdir`. CAS must work through this path too — the `ExpectedStatus` field flows through RPC/CLI transparently since it's part of UpdateArgs.

### What This Doesn't Address

- **Orphan respawn policy**: CAS tells you WHO won the race. The policy of WHEN to retry (currently 6h TTL) needs separate design. CAS makes it cleaner though — orphan detector resets to `open`, next CAS attempt wins immediately.
- **Beads unavailability**: If beads daemon is down, CAS (like all beads operations) is unavailable. The fail-open question shifts from "should heuristics allow spawn?" to "should we spawn at all without the structural gate?"

## Summary (D.E.K.N.)

**Delta:** Ghost issues in the beads ready queue permanently block test issue deferral because `ShouldDeferTestIssue` trusts the ready list without verifying siblings exist.

**Evidence:** 16 unit/integration tests pass, including 4 new ghost-sibling-specific tests. `TestDecide_IgnoresGhostSiblingForDeferral` directly reproduces the reported bug scenario.

**Knowledge:** Data sources can be inconsistent — an issue can exist in `bd ready` but not in `bd show`. Defense-in-depth: verify before blocking.

**Next:** Close. Fix implemented and tested.

**Authority:** implementation - Bug fix within existing patterns, no architectural changes.

---

# Investigation: Daemon Defers Triage Ready Issues

**Question:** Why does the daemon permanently defer 4 test issues due to a nonexistent sibling `orch-go-ehz`, and how to fix it?

**Started:** 2026-03-24
**Updated:** 2026-03-24
**Owner:** orch-go-natal
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: ShouldDeferTestIssue trusts allIssues without verification

**Evidence:** `pkg/daemon/sibling_sequencing.go:59-78` — the function iterates `allIssues` (populated from `ListReadyIssues()`) and defers when it finds a same-project implementation sibling with status "open" or "in_progress". No existence check is performed.

**Source:** `pkg/daemon/sibling_sequencing.go:73` — the condition `!isTestLikeIssue(other) && (other.Status == "open" || other.Status == "in_progress")` trusts the status field from the ready list.

**Significance:** If a ghost issue appears in the ready list (returned by `bd ready` but not findable by `bd show`), it permanently blocks all test issues in the same project.

---

### Finding 2: The data flow exposes the bug

**Evidence:** `ListReadyIssues()` → `PrioritizeIssues()` → `Decide()` → `ShouldDeferTestIssue(issue, orient.PrioritizedIssues)`. The function only sees data from the ready list — it has no mechanism to cross-check with the authoritative beads source (`GetIssueStatus`/`bd show`).

**Source:** `pkg/daemon/ooda.go:34` (Sense), `pkg/daemon/ooda.go:78` (Orient), `pkg/daemon/ooda.go:137` (Decide)

**Significance:** The `IssueQuerier` interface already provides `GetIssueStatus()` which calls `bd show` — the verification mechanism exists but wasn't wired into sibling deferral.

---

### Finding 3: Fix adds optional sibling validation with caching

**Evidence:** Added `SiblingExistsFunc` parameter to `ShouldDeferTestIssue`. In `Decide()` and `Preview()`, a validator using `GetIssueStatus()` with per-cycle caching verifies blocking siblings exist before deferring.

**Source:** `pkg/daemon/sibling_sequencing.go:18-22` (type), `pkg/daemon/ooda.go:133-148` (validator in Decide), `pkg/daemon/preview.go:96-104` (validator in Preview)

**Significance:** Ghost siblings are now skipped. Caching prevents repeated beads queries within a single cycle (4 test issues checking the same ghost = 1 query, not 4).

---

## Synthesis

**Key Insights:**

1. **Trust but verify** - The ready list is a denormalized view. Sibling deferral is a blocking decision, so it warrants verification against the authoritative source before blocking.

2. **nil validator = backwards compatible** - Passing `nil` as `SiblingExistsFunc` preserves the old behavior (trust all siblings), so unit tests that don't care about ghost issues just pass `nil`.

3. **Root cause is beads data inconsistency** - The ghost issue `orch-go-ehz` exists in the ready set but not in the show path. This is a separate beads bug, but the daemon should be resilient.

**Answer to Investigation Question:**

The daemon permanently defers test issues because `ShouldDeferTestIssue` trusts the ready list without verifying siblings exist. Ghost issue `orch-go-ehz` appears in `ListReadyIssues()` but `GetIssueStatus()` (= `bd show`) can't find it. Fix: verify blocking siblings via `GetIssueStatus()` before deferring.

---

## Structured Uncertainty

**What's tested:**

- Ghost sibling is ignored when validator returns false (`TestShouldDeferTestIssue_GhostSiblingIgnored`)
- Real sibling still causes deferral with validator (`TestShouldDeferTestIssue_ValidSiblingStillDefers`)
- Mixed ghost+real siblings: ghost skipped, real still defers (`TestShouldDeferTestIssue_GhostSiblingSkippedRealSiblingDefers`)
- Nil validator trusts all siblings, backwards compatible (`TestShouldDeferTestIssue_NilValidatorTrustsAllSiblings`)
- Decide integration: ghost sibling does not block spawning (`TestDecide_IgnoresGhostSiblingForDeferral`)
- All 16 existing + new tests pass

**What's untested:**

- Performance impact of `GetIssueStatus` calls per cycle (mitigated by caching, but not benchmarked)
- Root cause of why `orch-go-ehz` appears in ready list but not in show (beads inconsistency)

**What would change this:**

- If `GetIssueStatus` itself returned stale data (false positive exists), the ghost would still block
- If beads ready list is fixed to never return ghost issues, the validator becomes a no-op (harmless)

---

## References

**Files Examined:**
- `pkg/daemon/sibling_sequencing.go` - Core deferral logic (modified)
- `pkg/daemon/ooda.go` - Decide() caller (modified)
- `pkg/daemon/preview.go` - Preview() caller (modified)
- `pkg/daemon/interfaces.go` - IssueQuerier.GetIssueStatus definition
- `pkg/daemon/issue_adapter.go` - GetBeadsIssueStatus implementation
- `pkg/daemon/issue_queue.go` - projectFromIssueID, Issue struct

**Commands Run:**
```bash
# Run all daemon tests
go test ./pkg/daemon/ -count=1

# Run sibling-specific tests
go test ./pkg/daemon/ -run "TestShouldDeferTestIssue|TestDecide_Defers|TestDecide_Spawns|TestDecide_Ignores" -v
```

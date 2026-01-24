<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Phase: Complete check was only running for in_progress issues, not for open issues - allowing respawning of completed work.

**Evidence:** Code at spawn_cmd.go:1082-1085 was inside `if issue.Status == "in_progress"` block. Fix moves check outside to run for ANY status.

**Knowledge:** Pre-spawn gates must run unconditionally before status-specific logic; nesting gates inside status checks creates bypass paths.

**Next:** Merged - fix implemented and tests passing.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural pattern)

---

# Investigation: Implement Phase Complete Check Before

**Question:** Why does spawn allow respawning completed work despite having Phase: Complete check?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Phase: Complete check nested inside in_progress block

**Evidence:** At spawn_cmd.go lines 1082-1085, the Phase: Complete check was inside the `if issue.Status == "in_progress"` block:
```go
if issue.Status == "in_progress" {
    // ... active session checks ...
    if complete, err := verify.IsPhaseComplete(beadsID); err == nil && complete {
        return fmt.Errorf("issue %s has Phase: Complete but is not closed...")
    }
}
```

**Source:** cmd/orch/spawn_cmd.go:1065-1090

**Significance:** Issues with status "open" (never marked in_progress) could have Phase: Complete comments but still be respawned because the check never runs.

---

### Finding 2: Daemon has proper Phase: Complete check

**Evidence:** pkg/daemon/issue_adapter.go has `HasPhaseComplete()` function used at daemon.go:855 and :975 to check before spawning. The daemon path was protected.

**Source:** pkg/daemon/daemon.go:855, pkg/daemon/issue_adapter.go:168-201

**Significance:** The daemon-driven flow was correct, but manual `orch spawn --issue ID` bypassed the check for "open" issues.

---

### Finding 3: Fix requires moving check outside status-specific block

**Evidence:** Moved the Phase: Complete check to run immediately after the "closed" check, before any status-specific logic:
```go
if issue.Status == "closed" {
    return fmt.Errorf("issue %s is already closed", beadsID)
}
// Pre-spawn Phase: Complete check: runs for ANY status
if complete, err := verify.IsPhaseComplete(beadsID); err == nil && complete {
    return fmt.Errorf("issue %s has Phase: Complete but is not closed...")
}
```

**Source:** cmd/orch/spawn_cmd.go:1062-1070 (after fix)

**Significance:** Now Phase: Complete blocks spawn regardless of issue status (open, in_progress, etc.).

---

## Synthesis

**Key Insights:**

1. **Gate nesting creates bypass paths** - When pre-spawn checks are nested inside status-specific conditions, some statuses bypass the check entirely.

2. **Daemon vs manual spawn paths differ** - The daemon had proper protection, but manual spawn with --issue flag did not, creating an inconsistent behavior.

**Answer to Investigation Question:**

The spawn allowed respawning completed work because the Phase: Complete check (verify.IsPhaseComplete) was only executed for issues with status "in_progress". Issues with status "open" that had Phase: Complete comments would bypass this check entirely. The fix moves the check outside the status-specific block so it runs for all issues.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds: `go build ./cmd/orch/` passes
- ✅ All cmd/orch tests pass: `go test ./cmd/orch/`
- ✅ All daemon Phase: Complete tests pass: `go test ./pkg/daemon/ -run PhaseComplete`
- ✅ All verify Phase tests pass: `go test ./pkg/verify/ -run Phase`

**What's untested:**

- ⚠️ End-to-end reproduction (would require a real beads issue with Phase: Complete and open status)

**What would change this:**

- Finding would be wrong if verify.IsPhaseComplete doesn't correctly parse beads comments (but daemon tests verify this)

---

## References

**Files Examined:**
- cmd/orch/spawn_cmd.go - Main spawn command, lines 1060-1100
- pkg/daemon/issue_adapter.go - HasPhaseComplete implementation
- pkg/daemon/daemon.go - Daemon spawn flow with Phase: Complete check

**Commands Run:**
```bash
# Verify build
go build ./cmd/orch/

# Run related tests
go test ./cmd/orch/ -run Spawn
go test ./pkg/daemon/ -run PhaseComplete
go test ./pkg/verify/ -run Phase
```

---

## Investigation History

**2026-01-23 16:40:** Investigation started
- Initial question: Phase: Complete check documented but not implemented in spawn flow
- Context: Bug report from daemon reliability audit

**2026-01-23 16:50:** Root cause identified
- Phase: Complete check at spawn_cmd.go:1082-1085 was inside in_progress block
- Open issues with Phase: Complete comments would bypass the check

**2026-01-23 16:55:** Fix implemented
- Moved Phase: Complete check outside status-specific block
- Removed duplicate check from inside in_progress block
- All tests passing

**2026-01-23 17:00:** Investigation completed
- Status: Complete
- Key outcome: Phase: Complete check now runs for ANY issue status before spawn

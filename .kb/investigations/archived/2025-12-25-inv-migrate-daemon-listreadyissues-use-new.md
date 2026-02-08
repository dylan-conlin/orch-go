## Summary (D.E.K.N.)

**Delta:** Successfully migrated daemon.ListReadyIssues to use beads RPC client with automatic CLI fallback.

**Evidence:** All 62 daemon tests pass; convertBeadsIssues correctly maps all Issue fields.

**Knowledge:** The pkg/beads client provides a clean interface; fallback pattern ensures graceful degradation.

**Next:** Close - implementation complete, commit and mark Phase: Complete.

**Confidence:** High (90%) - All existing tests pass, new conversion tests added.

---

# Investigation: Migrate Daemon ListReadyIssues to Use Beads RPC Client

**Question:** How to migrate daemon.ListReadyIssues from bd CLI subprocess to use pkg/beads RPC client with fallback?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: daemon.Issue and beads.Issue have compatible fields

**Evidence:** Both types share ID, Title, Description, Priority, Status, IssueType, Labels fields with identical types.

**Source:** pkg/daemon/daemon.go:49-57, pkg/beads/types.go:109-122

**Significance:** Type conversion is straightforward - direct field mapping with no loss of information.

---

### Finding 2: beads.Client already provides Ready() method

**Evidence:** pkg/beads/client.go:254-271 implements Ready(args *ReadyArgs) returning []Issue.

**Source:** pkg/beads/client.go:254

**Significance:** No new RPC operations needed - simply integrate existing client.

---

### Finding 3: Fallback pattern established by beads package

**Evidence:** beads.FallbackReady() already exists (pkg/beads/client.go:386-400) for CLI fallback.

**Source:** pkg/beads/client.go:386

**Significance:** Pattern is consistent with existing package design; we use same approach in daemon.

---

## Implementation Summary

**Changes made:**
1. Updated `ListReadyIssues()` to try beads RPC client first (pkg/daemon/daemon.go:314-332)
2. Added `listReadyIssuesCLI()` helper for CLI fallback (pkg/daemon/daemon.go:334-349)
3. Added `convertBeadsIssues()` for type conversion (pkg/daemon/daemon.go:351-366)
4. Added 3 unit tests for conversion function (pkg/daemon/daemon_test.go:833-911)

**Files modified:**
- pkg/daemon/daemon.go
- pkg/daemon/daemon_test.go

**Test results:** All 62 daemon tests pass.

---

## References

**Files Examined:**
- pkg/daemon/daemon.go - Original ListReadyIssues implementation
- pkg/beads/client.go - RPC client Ready() method
- pkg/beads/types.go - Issue struct definition

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-25-inv-design-beads-integration-strategy-orch.md - Parent strategy
- **Workspace:** .orch/workspace/og-feat-migrate-daemon-listreadyissues-25dec/ - Spawn context

---

## Investigation History

**2025-12-25:** Investigation started and completed
- Initial question: How to migrate ListReadyIssues to use beads RPC client?
- Implementation: Direct integration with try/fallback pattern
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Successfully migrated with all tests passing

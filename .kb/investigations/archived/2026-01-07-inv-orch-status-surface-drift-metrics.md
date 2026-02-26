<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added SESSION METRICS section to orch status showing time in session, last spawn time, and spawn count for drift detection.

**Evidence:** `orch status` now shows drift metrics; tested with active session and without; both text and JSON output work correctly.

**Knowledge:** File reads tracking requires event infrastructure (OpenCode plugin or tool.execute events) that doesn't exist yet - deferred.

**Next:** Close - feature complete with current scope.

**Promote to Decision:** recommend-no (tactical feature, not architectural)

---

# Investigation: Orch Status Surface Drift Metrics

**Question:** How to surface session drift metrics in orch status for Dylan visibility?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Session state already tracks spawn timing

**Evidence:** `pkg/session/session.go` has `SpawnRecord` with `SpawnedAt` timestamp, making "time since last spawn" calculation trivial.

**Source:** `pkg/session/session.go:59-74`

**Significance:** Core metrics (time in session, last spawn, spawn count) are derivable from existing infrastructure without changes to session storage.

---

### Finding 2: File reads tracking requires event infrastructure

**Evidence:** OpenCode Message API returns messages/parts, but counting "Read" tool invocations would require parsing all messages. No event-level tracking exists.

**Source:** `pkg/opencode/client.go`, `pkg/opencode/types.go`

**Significance:** File reads tracking is non-trivial - would need OpenCode plugin on `tool.execute.after` event or similar infrastructure. Deferred for now.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Incremental deployment** - Ship time in session + last spawn now, add file reads later when event infrastructure exists.

**Why this approach:**
- Core drift signals (time, spawns) are immediately useful
- Doesn't block on infrastructure that doesn't exist
- Dylan can still detect patterns like "2 hours, no spawns"

**Trade-offs accepted:**
- File reads tracking deferred
- Partial solution vs complete specification

---

## References

**Files Modified:**
- `cmd/orch/status_cmd.go` - Added SessionMetrics, getSessionMetrics, printSessionMetrics

**Commands Run:**
```bash
go build ./cmd/orch/...
go test ./cmd/orch/... -v -run Status
go run ./cmd/orch status
go run ./cmd/orch status --json
```

---

## Investigation History

**2026-01-07 14:40:** Investigation started
- Initial question: How to surface drift detection metrics in orch status
- Context: Strategic Orchestrator Model - Dylan needs visibility into orchestrator behavior

**2026-01-07 14:50:** Implementation complete
- Status: Complete
- Key outcome: SESSION METRICS section added to orch status with time/spawn metrics

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed coaching plugin worker detection by replacing unreliable session.metadata.role check with file-path based detection (SPAWN_CONTEXT.md reads, .orch/workspace/ paths).

**Evidence:** Prior implementation at orchestrator-session.ts uses same file-path detection pattern successfully; coaching.ts was using session.metadata.role which isn't reliably set.

**Knowledge:** Worker detection in OpenCode plugins must use file-path signals (tool arguments) rather than session metadata, since metadata may not be set reliably across all spawn paths.

**Next:** Close issue - fix implemented and committed.

**Promote to Decision:** recommend-no (tactical fix applying established pattern from orchestrator-session.ts)

---

# Investigation: Fix Coaching Plugin Firing Workers

**Question:** Why is coaching plugin injecting orchestrator coaching messages into worker sessions?

**Started:** 2026-01-24
**Updated:** 2026-01-24
**Owner:** orch-go-256a9
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Worker detection was using unreliable session.metadata.role

**Evidence:** The `detectWorkerSession()` function in coaching.ts:1328-1346 checked `session?.metadata?.role === 'worker'` which depends on OpenCode setting this from the `x-opencode-env-ORCH_WORKER=1` header - but this isn't reliably set.

**Source:** plugins/coaching.ts:1333-1345 (before fix)

**Significance:** Workers weren't being detected, so coaching alerts meant for orchestrators were firing on workers.

---

### Finding 2: orchestrator-session.ts has proven file-path based detection

**Evidence:** orchestrator-session.ts:155-196 uses robust detection:
1. Read tool accessing SPAWN_CONTEXT.md (workers always read this early)
2. Any tool accessing files in .orch/workspace/ (workers operate here)

**Source:** plugins/orchestrator-session.ts:155-196

**Significance:** This pattern is already proven in production and should be copied to coaching.ts.

---

## Synthesis

**Key Insights:**

1. **Session metadata isn't reliable** - Worker detection can't depend on session.metadata.role being set correctly.

2. **File-path signals are reliable** - Workers always read SPAWN_CONTEXT.md early and operate in .orch/workspace/ directories.

**Answer to Investigation Question:**

The coaching plugin was injecting messages into worker sessions because its worker detection relied on `session.metadata.role` which isn't reliably set. Fixed by copying the proven file-path based detection from orchestrator-session.ts.

---

## References

**Files Modified:**
- plugins/coaching.ts - Updated detectWorkerSession() to use file-path detection

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-10-inv-add-worker-filtering-coaching-ts.md - Prior work that identified the fix approach
- **Plugin:** plugins/orchestrator-session.ts - Source of the proven detection pattern

# Session Synthesis

**Agent:** og-debug-dashboard-connection-pool-05jan-5a45
**Issue:** orch-go-qjcwx
**Duration:** 2026-01-05
**Outcome:** success

---

## TLDR

Fixed dashboard connection pool exhaustion by removing agentlog SSE auto-connect on page load. The agentlog is now opt-in via the "Follow" button, freeing one of the 6 HTTP/1.1 connections per origin for API requests.

---

## Delta (What Changed)

### Files Created
- None

### Files Modified
- `web/src/routes/+page.svelte` - Removed agentlog auto-connect code, added explanatory comment referencing investigation

### Commits
- Pending (ready to commit)

---

## Evidence (What Was Observed)

- Dashboard auto-connected two SSE streams: primary events (`connectSSE()`) and agentlog (`connectAgentlogSSE()`) on page load
- HTTP/1.1 limits 6 concurrent connections per origin - SSE connections are long-lived and occupy slots permanently
- Dashboard performs 9+ API fetches on load (agents, beads, usage, focus, servers, readyIssues, daemon, hotspots, orchestratorSessions)
- Agentlog SSE panel is collapsed by default and only visible in historical mode - most users never see it
- The "Follow" button UI already existed, suggesting opt-in was the intended UX

### Tests Run
```bash
# Build verification
bun run build
# Result: Build succeeded (✓ built in 8.43s)

# TypeScript check showed pre-existing errors in theme.ts (unrelated to this change)
bun run check
# Result: 2 errors in theme.ts (pre-existing)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - Full investigation with root cause analysis

### Decisions Made
- Decision: Make agentlog SSE opt-in only because it's non-critical and consumes valuable connection slot

### Constraints Discovered
- HTTP/1.1 6-connection limit is a hard constraint - long-lived SSE connections should be minimized
- Non-critical SSE streams should be opt-in, not auto-connected

### Externalized via `kn`
- None required (constraint captured in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (build passes)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-qjcwx`

### Long-term Considerations
For future reference, potential long-term solutions if more SSE connections are needed:
1. Enable HTTP/2 on API server (eliminates connection limit via multiplexing)
2. Single multiplexed SSE endpoint (one connection for all event types)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- None

**Areas worth exploring further:**
- HTTP/2 support for the orch-go API server
- Multiplexing multiple event types over single SSE connection

**What remains unclear:**
- Nothing - straightforward fix with clear root cause

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-dashboard-connection-pool-05jan-5a45/`
**Investigation:** `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md`
**Beads:** `bd show orch-go-qjcwx`

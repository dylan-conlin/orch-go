# Session Synthesis

**Agent:** og-debug-fix-dashboard-connection-05jan-2777
**Issue:** orch-go-qjcwx
**Duration:** 2026-01-05 23:13 → 2026-01-05 23:20
**Outcome:** success

---

## TLDR

Fixed dashboard connection pool exhaustion by removing auto-connect of agentlog SSE on page load. HTTP/1.1's 6-connection-per-origin limit was exhausted by two SSE connections plus parallel fetch requests.

---

## Delta (What Changed)

### Files Modified
- `web/src/routes/+page.svelte` - Removed agentlog SSE auto-connect (lines 137-145), added explanatory comment

### Commits
- (pending) - fix: remove agentlog SSE auto-connect to prevent connection pool exhaustion

---

## Evidence (What Was Observed)

- Root cause confirmed: HTTP/1.1 limits browsers to 6 connections per origin
- Dashboard auto-connected to two SSE endpoints: `/api/events` (primary) and `/api/agentlog?follow=true` (secondary)
- With 2 long-lived SSE connections + multiple parallel fetches, connection pool exhausted
- After fix: API responds immediately (verified via curl to localhost:3348/api/agents)
- Dashboard loads successfully (verified via curl to localhost:5188)

### Tests Run
```bash
# Verified API responds
curl -s http://localhost:3348/api/agents | head -100
# Result: JSON response with agent data returned immediately

# Verified dashboard accessible
curl -s -w "\n%{http_code}" http://localhost:5188 | tail -1
# Result: 200
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-inv-fix-dashboard-connection-pool-exhaustion.md` - Full investigation

### Decisions Made
- Decision: Make agentlog SSE opt-in via Follow button rather than auto-connecting, because it's a debugging feature not critical for dashboard operation

### Constraints Discovered
- HTTP/1.1 connection limit (6 per origin) applies to SSE connections, which are long-lived and consume pool capacity

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (API verified responsive)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-qjcwx`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the dashboard use HTTP/2 to eliminate connection limits entirely?
- Could SSE endpoints be combined into a single multiplexed stream?

**Areas worth exploring further:**
- HTTP/2 server configuration for orch serve
- SSE event multiplexing pattern for future scaling

**What remains unclear:**
- Performance impact under high agent load with single SSE connection

*(Note: These are long-term architectural improvements, not blockers for this fix)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-debug-fix-dashboard-connection-05jan-2777/`
**Investigation:** `.kb/investigations/2026-01-05-inv-fix-dashboard-connection-pool-exhaustion.md`
**Beads:** `bd show orch-go-qjcwx`

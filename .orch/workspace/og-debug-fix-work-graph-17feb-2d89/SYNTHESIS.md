# Session Synthesis

**Agent:** og-debug-fix-work-graph-17feb-2d89
**Issue:** orch-go-1024
**Outcome:** success

---

## Plain-Language Summary

The work-graph page's "Ready to Complete" section never updated in real-time because of two compounding bugs. First, the `agents` store was excluded from the 30-second polling cycle — only `workGraph`, `wip`, `daemon`, and `attention` were refreshed periodically. Phase transitions happen via `bd comment` (a beads operation), which produces no OpenCode SSE events, so the only way to detect them was polling — but the agents poll wasn't happening. Second, the backend returns `status: "completed"` for Phase:Complete agents with live sessions (Priority 3 in the status cascade), but the Ready to Complete reactive block only checked for `status: "active"` and `"awaiting-cleanup"`, missing these agents entirely. Both issues are fixed in `+page.svelte` with minimal, targeted changes.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification steps.

---

## TLDR

Fixed work-graph "Ready to Complete" section not updating in real-time by adding agents to the 30s polling cycle and widening the status filter to include "completed" agents (whose beads issues are still open).

---

## Delta (What Changed)

### Files Modified
- `web/src/routes/work-graph/+page.svelte` - Added `agents.fetch()` to `runRefreshCycle()` and included `'completed'` in the Ready to Complete status filter

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- **Root Cause 1**: `runRefreshCycle()` (line 135) fetches workGraph, wip, daemon, attention — but NOT agents. Agents store only refreshes on SSE lifecycle events (`session.created/deleted`, `agent.completed/abandoned`). Phase:Complete via `bd comment` produces none of these events.
- **Root Cause 2**: `determineAgentStatus()` in `serve_agents.go:1145` returns `"completed"` when phaseComplete=true and session alive (Priority 3). The reactive block on line 345 only checked for `"active"` and `"awaiting-cleanup"`.
- **Safety**: The workGraph scope=open means closed issues aren't in the graph, so including `"completed"` status won't show agents whose issues are already closed — the `issueNode` lookup on line 353 returns null for those.

### Tests Run
```bash
# Vite build succeeds
npx vite build
# ✓ built in 11.12s

# Go build succeeds
go build ./cmd/orch/
# (no errors)
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Added agents polling to runRefreshCycle rather than emitting custom SSE events for phase transitions. Polling is simpler, sufficient (30s latency), and doesn't require backend changes.
- Widened status filter rather than changing backend status cascade. The cascade is well-documented (Priority 1-6) and used across the codebase; changing it could have side effects.

### Constraints Discovered
- `bd comment` is invisible to both SSE event streams (OpenCode SSE and agentlog SSE). The beads system has no event emission for comment creation. This means any frontend feature depending on beads comment changes must use polling.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Build passing (frontend and backend)
- [x] Ready for `orch complete orch-go-1024`

---

## Unexplored Questions

- Should the backend emit custom SSE events for phase transitions? This would reduce latency from 30s to near-instant but requires polling beads comments server-side — essentially moving the polling from frontend to backend.
- The `agents.fetch()` call in the polling cycle fetches ALL agents without time/project filters. If the work-graph page should only show agents for the current project, a filter query string should be passed. Currently this matches the existing behavior (onMount also calls `agents.fetch()` without filters).

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-fix-work-graph-17feb-2d89/`
**Beads:** `bd show orch-go-1024`

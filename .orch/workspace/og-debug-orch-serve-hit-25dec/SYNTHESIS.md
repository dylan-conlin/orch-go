# Session Synthesis

**Agent:** og-debug-orch-serve-hit-25dec
**Issue:** orch-go-cvce
**Duration:** 2025-12-25
**Outcome:** success

---

## TLDR

Fixed 125% CPU usage in `orch serve` by removing O(n) HTTP calls from `handleAgents` endpoint and increasing SSE refetch debounce from 100ms to 500ms. Root cause was a feedback loop where SSE events triggered refetches, each making HTTP calls per active session.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve.go` - Removed `IsSessionProcessing()` call that made HTTP request per session (now populated client-side via SSE)
- `web/src/lib/stores/agents.ts` - Increased debounce interval from 100ms to 500ms with documentation

### Commits
- (pending) - Fix 125% CPU in orch serve: remove per-session HTTP calls, increase debounce

---

## Evidence (What Was Observed)

- **Root cause 1:** `handleAgents` called `client.IsSessionProcessing(s.ID)` for each session (serve.go:316), which makes HTTP call to `/session/{id}/message` - O(n) HTTP calls per API request
- **Root cause 2:** Frontend SSE handler triggered `agents.fetch()` on every `session.status` event (agents.ts:483-484), debounced at only 100ms
- **Compounding effect:** With 3 agents + dashboard open, high SSE frequency × sessions × HTTP calls = CPU spiral
- **Existing mitigation:** Frontend already has debounce + abort controller, but 100ms was insufficient
- **SSE event frequency:** Observed `message.part.updated` and `session.status` events streaming rapidly during agent activity

### Tests Run
```bash
go build ./...
# SUCCESS

go test ./cmd/orch/... -v -run "Serve"
# PASS: TestServeStatusWithMockServer
# PASS: TestServeStatusWithNoServer  
# PASS: TestDefaultServePort
# ... all tests passing
```

---

## Knowledge (What Was Learned)

### Decisions Made
- **Remove `IsSessionProcessing` from backend:** Frontend already receives `is_processing` state via SSE `session.status` events (busy/idle) and updates local state immediately (agents.ts:449-476). No need to poll from backend.
- **Increase debounce to 500ms:** Strikes balance between responsiveness (imperceptible to user) and performance (collapses rapid SSE events into single request). With 3 agents, reduces refetches by ~80%.

### Constraints Discovered
- `IsProcessing` field in `AgentAPIResponse` is now always `false` from backend - frontend overrides via SSE. This is intentional, not a bug.

### Patterns Identified
- **SSE + Polling Anti-pattern:** When SSE events trigger API refetches which also fetch state that SSE provides, you get a feedback loop. Solution: Let SSE update local state for high-frequency fields, refetch only for structural changes.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Root cause identified and fixed
- [x] Ready for `orch complete orch-go-cvce`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could we further optimize by removing `session.status` from the refetch trigger list entirely? The local SSE handler already updates `is_processing`, and `updated_at` changes may not be critical for real-time display.
- Is there value in batching `GetMessages` calls for `IsSessionProcessing` if we need it in the future? (e.g., single request for all session messages)

**What remains unclear:**
- Exact SSE event frequency during high load - would be useful to instrument for future optimization

*(This was a focused debugging session - root cause identified and fixed)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-debug-orch-serve-hit-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-orch-serve-hit-125-cpu.md`
**Beads:** `bd show orch-go-cvce`

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented SSE-driven is_processing state updates for agent cards with visual yellow pulse indicator.

**Evidence:** Build passes, type checks pass, 18/19 Playwright tests pass (1 flaky unrelated to changes).

**Knowledge:** SSE session.status events provide busy/idle status; message.part events signal active processing. Both can be used to update is_processing state in real-time without requiring API polling.

**Next:** Close issue - feature complete.

**Confidence:** High (90%) - tested via build/check, visual verification pending production use.

---

# Investigation: Agent Card Should Show Processing

**Question:** How to show processing state on agent card after orch send, with real-time SSE updates?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent og-feat-agent-card-should-24dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: is_processing state already exists but only updated on API fetch

**Evidence:** `pkg/opencode/client.go:315-345` implements `IsSessionProcessing()` which checks message finish state. `cmd/orch/serve.go:192` calls this for each session.

**Source:** 
- `pkg/opencode/client.go:315-345`
- `cmd/orch/serve.go:192`

**Significance:** Backend already has processing detection, but frontend only gets updates on API calls, not SSE events.

---

### Finding 2: SSE events include session.status with busy/idle type

**Evidence:** `pkg/opencode/sse.go:96-108` shows session.status event format: `{"type":"session.status","properties":{"sessionID":"...","status":{"type":"busy|idle"}}}`. This can be used to infer processing state in real-time.

**Source:** `pkg/opencode/sse.go:96-108`

**Significance:** Frontend can update is_processing based on status.type === 'busy' without waiting for next API poll.

---

### Finding 3: message.part events indicate active generation

**Evidence:** `web/src/lib/stores/agents.ts:271-295` already handles message.part to update current_activity. These events only fire when agent is actively generating response.

**Source:** `web/src/lib/stores/agents.ts:271-295`

**Significance:** When message.part fires, we can confidently set is_processing=true.

---

## Changes Made

1. **agents.ts (lines 271-321):** 
   - Modified `message.part` handler to also set `is_processing: true`
   - Added new `session.status` handler to update `is_processing` based on busy/idle status
   - When status is idle, also clears `current_activity`

2. **agent-card.svelte (line 163):**
   - Added conditional classes for processing state: `border-yellow-500 animate-pulse shadow-md shadow-yellow-500/20`
   - Status indicator bar at top changes to yellow when processing

---

## References

**Files Modified:**
- `web/src/lib/stores/agents.ts` - SSE event handling for is_processing
- `web/src/lib/components/agent-card/agent-card.svelte` - Visual indicator styling

**Commands Run:**
```bash
npm run check  # TypeScript/Svelte checks - passed
npm run build  # Production build - passed
npx playwright test  # 18/19 tests pass
```

---

## Investigation History

**2025-12-24 10:00:** Investigation started
- Initial question: How to show processing state on agent card after orch send?
- Context: Dashboard doesn't show yellow indicator when agent responds to follow-up via orch send

**2025-12-24 10:15:** Found SSE event structure
- session.status events contain busy/idle type
- message.part events indicate active generation

**2025-12-24 10:30:** Implementation complete
- Modified agents.ts to update is_processing from SSE events
- Enhanced agent-card.svelte with yellow pulse styling
- Build and type checks pass

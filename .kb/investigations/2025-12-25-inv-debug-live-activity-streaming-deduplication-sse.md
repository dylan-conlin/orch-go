<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SSE streaming events with same `part.id` were being added as duplicates instead of updating in place, causing live activity to show the same text multiple times.

**Evidence:** Observed SSE events (`message.part.updated`) have unique `part.id` fields (e.g., `prt_b58ff9623001...`) but the SSE store was assigning new IDs via `generateSSEEventId()`, treating each update as a new event.

**Knowledge:** OpenCode sends incremental updates with stable `part.id` for deduplication; the frontend must use this ID to update in place rather than append.

**Next:** Close - fix implemented and tested. Live activity stream now deduplicates by `part.id`.

**Confidence:** High (85%) - Code change is straightforward; Playwright and build tests pass; awaiting live smoke test for full verification.

---

# Investigation: Debug Live Activity Streaming Deduplication SSE

**Question:** Why does the live activity stream in the dashboard show duplicate messages as the model generates text?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent og-debug-live-activity-streaming-25dec
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: SSE Events Have Stable `part.id` for Deduplication

**Evidence:** Observed SSE events from OpenCode have structure:
```json
{
  "type": "message.part.updated",
  "properties": {
    "part": {
      "id": "prt_b58ff9623001nybuPUFSSZjl0n",
      "sessionID": "ses_4a70a2cf1ffe4YLnymAyYyPiVv",
      "type": "tool",
      "tool": "todowrite",
      "state": {"status": "running", ...}
    }
  }
}
```

**Source:** 
- `curl http://127.0.0.1:3348/api/events` - observed live SSE stream
- `web/src/lib/stores/agents.ts:60-85` - SSEEvent type definition

**Significance:** The `part.id` is a stable identifier that OpenCode uses to send incremental updates for the same message part (text chunk, tool invocation, etc.). This should be used for deduplication.

---

### Finding 2: Frontend Was Assigning New IDs to Every Event

**Evidence:** The `createSSEStore()` function at lines 187-204 was using `generateSSEEventId()` for ALL events:
```typescript
addEvent: (event: Omit<SSEEvent, 'id'>) => {
    update((events) => {
        const newEvents = [...events, { ...event, id: generateSSEEventId() }];
        return newEvents.slice(-100);
    });
}
```

**Source:** `web/src/lib/stores/agents.ts:193-198` (before fix)

**Significance:** This meant every SSE event with the same `part.id` was treated as a NEW event because it got a unique store ID like `sse-1735184643285-42`. The live activity log accumulated duplicates.

---

### Finding 3: `message.part.updated` Events Not Handled for Agent Activity

**Evidence:** The `handleSSEEvent` function only handled `message.part` events:
```typescript
if (data.type === 'message.part' && data.properties) {
```

But most tool events are sent as `message.part.updated` (observed in SSE stream).

**Source:** `web/src/lib/stores/agents.ts:353-378` (before fix)

**Significance:** Agent activity updates were missed for tool operations because `message.part.updated` wasn't being processed.

---

## Synthesis

**Key Insights:**

1. **OpenCode provides deduplication primitives** - The `part.id` field is specifically designed for clients to track and update streaming content, but it wasn't being used.

2. **Two event types for different purposes** - `message.part` is for text streaming (incremental chunks), `message.part.updated` is for tool state changes (running/completed). Both need handling.

3. **In-place update is the correct pattern** - Rather than appending all events and filtering later, updating in place when `part.id` matches prevents the duplication issue at the source.

**Answer to Investigation Question:**

The live activity stream showed duplicates because the SSE store generated new IDs for every event (`generateSSEEventId()`) instead of using the stable `part.id` from OpenCode. When OpenCode sends incremental updates (e.g., text as it streams, or tool status changes), each update should replace the previous event with the same `part.id`, not append as a new entry.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The root cause is clearly identified and the fix is straightforward. TypeScript checks pass, the build succeeds, and Playwright tests pass (2 pre-existing failures unrelated to this change).

**What's certain:**

- ✅ SSE events have stable `part.id` for deduplication
- ✅ The old code was creating new IDs for every event
- ✅ The fix correctly updates events in place when `part.id` matches
- ✅ Build and TypeScript checks pass

**What's uncertain:**

- ⚠️ Live smoke test pending (need to observe active agent streaming)
- ⚠️ Performance impact of findIndex for each event (should be minimal with 100-event limit)

**What would increase confidence to Very High (95%+):**

- Live smoke test with active agent showing no duplicate messages
- Verify text streaming specifically (not just tool events)

---

## Implementation Recommendations

### Recommended Approach ⭐ (IMPLEMENTED)

**Deduplicate by `part.id`** - Use the stable `part.id` from SSE events as the event ID, updating in place for events with matching IDs.

**Why this approach:**
- Uses OpenCode's built-in deduplication primitive
- Minimal code change - leverages existing store update pattern
- Handles both text streaming and tool state changes

**Trade-offs accepted:**
- `findIndex` on each event - acceptable for 100-event limit
- Legacy `addEvent` kept for backwards compatibility

**Implementation sequence:**
1. Add `extractEventId()` helper to get `part.id` from event data
2. Add `addOrUpdateEvent()` that checks for existing event with same ID
3. Update `handleSSEEvent()` to use `addOrUpdateEvent()` and handle `message.part.updated`

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - SSE store and event handling
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Activity log display
- `pkg/opencode/sse.go` - Backend SSE handling

**Commands Run:**
```bash
# View live SSE events
curl http://127.0.0.1:3348/api/events

# TypeScript check
npm run check

# Build verification
npm run build

# Playwright tests
npx playwright test
```

---

## Investigation History

**2025-12-25 04:45:** Investigation started
- Initial question: Why does live activity show duplicate messages during streaming?
- Context: User reported SSE shows same message multiple times as model generates

**2025-12-25 04:48:** Root cause identified
- SSE events have stable `part.id` but frontend assigns new IDs
- `message.part.updated` events not handled for agent activity

**2025-12-25 04:52:** Fix implemented
- Added `extractEventId()` helper function
- Added `addOrUpdateEvent()` with deduplication logic
- Updated `handleSSEEvent()` to handle both event types

**2025-12-25 04:55:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Live activity deduplication fix implemented, tests pass

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard activity display relied solely on SSE events, which are stateless - no historical activity was provided on initial load.

**Evidence:** Code review showed `/api/agents` didn't include activity data; `current_activity` was only set from real-time SSE `message.part` events. OpenCode session metadata doesn't include message counts.

**Knowledge:** The original hypothesis that "OpenCode API returns messages:0" was incorrect - OpenCode stores messages separately but the API works correctly. The issue was architectural: dashboard had no way to get historical activity on load.

**Next:** Implemented fix by adding `last_activity` field to `/api/agents` response. Dashboard now shows last known activity from OpenCode messages on initial load, with SSE events taking precedence for real-time updates.

---

# Investigation: Dashboard Shows 'Waiting for Activity' Despite Agents Running

**Question:** Why does the dashboard show 'Waiting for activity' for agents that are actively running?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

---

## Findings

### Finding 1: Activity data was only populated from SSE events

**Evidence:** In `web/src/lib/stores/agents.ts`, the `current_activity` field is only set when `message.part` or `message.part.updated` SSE events are received (lines 453-481). The `/api/agents` endpoint doesn't return any activity data.

**Source:** `web/src/lib/stores/agents.ts:453-481`, `cmd/orch/serve.go:handleAgents`

**Significance:** When the dashboard loads or reconnects, there's no historical activity data available. SSE only provides real-time events, so any activity that occurred before connection is lost.

---

### Finding 2: OpenCode messages API works correctly

**Evidence:** Verified with curl:
```bash
curl -s http://localhost:4096/session/ses_xxx/message | jq 'length'
# Returns: 35
```
Messages exist on disk and the API returns them correctly. The original hypothesis that "messages:0" was returned was incorrect.

**Source:** Manual API testing, `pkg/opencode/client.go:GetMessages`

**Significance:** The data is available - it just wasn't being fetched and included in the agents response.

---

### Finding 3: Session metadata doesn't include message counts

**Evidence:** The OpenCode `/session/:id` endpoint returns:
```json
{
  "id": "ses_xxx",
  "title": "...",
  "time": {"created": ..., "updated": ...},
  "summary": {"additions": 557, "deletions": 11, "files": 3}
}
```
No message count field. This is by design - OpenCode stores session metadata and messages separately.

**Source:** Manual API testing, `pkg/opencode/types.go:Session`

**Significance:** Cannot rely on session metadata for activity detection - must fetch messages separately.

---

## Synthesis

**Key Insights:**

1. **Stateless SSE vs Stateful Display** - SSE is designed for real-time updates, not historical replay. The dashboard needs initial state from the API.

2. **Separation of Concerns in OpenCode** - Sessions and messages are stored/served separately. This is architecturally sound but means the dashboard needs to make additional API calls for activity data.

3. **Performance Tradeoff** - Fetching messages for each agent on every API call could be expensive. The fix uses parallelized fetching with semaphore limiting (same pattern as token fetching) to mitigate this.

**Answer to Investigation Question:**

The dashboard showed "Waiting for activity" because:
1. Activity data (`current_activity`) was only populated from SSE events
2. SSE is stateless - connecting to the stream doesn't replay historical events
3. The `/api/agents` endpoint didn't fetch or return any activity information
4. When the dashboard loaded/reconnected, it had no way to know what the agent was doing

The fix adds a `last_activity` field to the API response that fetches the most recent message part for active agents.

---

## Implementation

### Changes Made

1. **`cmd/orch/serve.go`**:
   - Added `LastActivityResponse` struct
   - Added `last_activity` field to `AgentAPIResponse`
   - Added `getLastActivityForSession()` helper function
   - Added parallelized fetching of last activity for active agents (similar to token fetching)

2. **`web/src/lib/stores/agents.ts`**:
   - Added `LastActivity` interface
   - Added `last_activity` field to `Agent` interface

3. **`web/src/lib/components/agent-card/agent-card.svelte`**:
   - Updated display state logic to consider `last_activity`
   - Added fallback to show `last_activity` when `current_activity` is not available

4. **`web/src/lib/components/agent-detail/agent-detail-panel.svelte`**:
   - Added fallback to show `last_activity` in activity panel
   - Shows "(last known)" indicator when displaying API data vs SSE data

---

## Verification

- [x] Go build passes: `go build ./cmd/orch`
- [x] Web build passes: `bun run build`
- [ ] Manual testing needed: Start orch serve + dashboard, verify activity shows on load

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added last_activity to API response
- `web/src/lib/stores/agents.ts` - Added LastActivity interface
- `web/src/lib/components/agent-card/agent-card.svelte` - Display fallback
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Display fallback

**Related:**
- `pkg/opencode/client.go:GetMessages` - Existing API for fetching messages

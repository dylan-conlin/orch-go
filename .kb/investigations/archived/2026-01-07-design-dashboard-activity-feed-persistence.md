<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard activity feed should fetch historical events from OpenCode API on-demand rather than caching in browser memory or orch-go backend.

**Evidence:** OpenCode stores all session messages/parts in ~/.local/share/opencode/storage/ and exposes them via GET /session/:sessionID/message API endpoint. Current implementation stores 1000 events globally in memory (diluted across agents, lost on refresh).

**Knowledge:** OpenCode is the authoritative source for session history - it persists all message parts to disk. The dashboard should treat SSE as real-time updates and the API as the source of truth for historical data.

**Next:** Implement hybrid architecture: SSE for real-time updates, API fetch for historical events when activity tab opens.

**Promote to Decision:** Actioned - dashboard patterns documented in dashboard guide

---

# Investigation: Dashboard Activity Feed Persistence

**Question:** How should the dashboard persist and retrieve agent activity events so viewing an agent's activity tab shows complete history, not just a recent snapshot?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** architect (spawned agent)
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Current Architecture - Global Event Buffer

**Evidence:** The dashboard stores SSE events in a global Svelte store limited to 1000 events (`web/src/lib/stores/agents.ts:363-364`):
```typescript
const newEvents = [...events, eventWithId];
// Keep last 1000 events (global across all agents)
return newEvents.slice(-1000);
```

The activity tab then filters these events by session ID (`web/src/lib/components/agent-detail/activity-tab.svelte:76-86`):
```typescript
let agentEvents = $derived(agent?.session_id 
    ? $sseEvents.filter(e => {
        if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
        const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
        if (eventSessionId !== agent?.session_id) return false;
        // ...filtering logic
    }).slice(-EVENT_LIMIT)
    : []);
```

**Source:** 
- `web/src/lib/stores/agents.ts:363-364` (global event limit)
- `web/src/lib/components/agent-detail/activity-tab.svelte:76-86` (filtering per agent)

**Significance:** With 5+ agents actively generating events, the 1000-event global buffer gets diluted quickly. A single agent might only have ~200 events visible before older ones are evicted. Page refresh loses everything.

---

### Finding 2: OpenCode Persists Full Session History

**Evidence:** OpenCode stores all session data to disk at `~/.local/share/opencode/storage/`:
- Sessions: `storage/session/{projectID}/{sessionID}.json`
- Messages: `storage/message/{sessionID}/{messageID}.json`  
- Parts: `storage/part/{messageID}/{partID}.json`

Each message contains metadata and each part contains the actual content (text, tool invocations, reasoning, step markers).

**Source:** 
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/storage/storage.ts` (storage implementation)
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts` (session management)
- `~/.local/share/opencode/storage/` (actual storage directory)

**Significance:** OpenCode already has the complete activity history persisted. We don't need to build a new persistence layer - we need to fetch from the existing one.

---

### Finding 3: OpenCode API Provides Message Retrieval

**Evidence:** OpenCode exposes session messages via API:

```
GET /session/:sessionID/message
```

Returns `MessageV2.WithParts[]` - an array of messages with their associated parts. The endpoint supports a `limit` query parameter.

**Source:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/server.ts:1067-1104`
```typescript
.get(
    "/session/:sessionID/message",
    // ... validation ...
    async (c) => {
        const query = c.req.valid("query")
        const messages = await Session.messages({
            sessionID: c.req.valid("param").sessionID,
            limit: query.limit,
        })
        return c.json(messages)
    },
)
```

**Significance:** The API we need already exists. We can fetch complete session history for any agent by calling OpenCode directly.

---

### Finding 4: Part Types Map to Activity Feed Categories

**Evidence:** OpenCode MessageV2 parts have types that map directly to dashboard filter categories:

| OpenCode Part Type | Dashboard Category |
|-------------------|-------------------|
| `text` | Text messages |
| `tool` | Tool invocations |
| `reasoning` | Reasoning |
| `step-start`, `step-finish` | Steps |

The part structure (`MessageV2.Part`) includes:
- `type`: Part type discriminator
- `text`: For text/reasoning parts
- `tool`, `state`: For tool invocations
- `time`: Timing information
- `sessionID`, `messageID`: For linking

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/message-v2.ts:61-326`

**Significance:** The OpenCode API returns data in a structure that can be directly transformed to the SSEEvent format the activity tab already expects.

---

## Synthesis

**Key Insights:**

1. **OpenCode is the source of truth** - All session activity is already persisted by OpenCode. The dashboard's in-memory store is redundant for historical data - it's useful only for real-time updates before they're written to OpenCode storage.

2. **Hybrid architecture is natural** - SSE provides real-time updates as they happen; API fetch provides historical data when needed. This is a common pattern (chat apps, notification systems) that separates streaming from persistence.

3. **Transformation is straightforward** - OpenCode's `MessageV2.Part` structure maps directly to the `SSEEvent` format the activity tab already consumes. We can fetch from API and merge with SSE events seamlessly.

**Answer to Investigation Question:**

The dashboard should fetch historical activity from OpenCode's `/session/:sessionID/message` API when the activity tab opens, then continue receiving real-time updates via SSE. This provides:
- Complete history on demand (not limited to browser session)
- Survives page refresh
- Per-agent history without global buffer dilution
- No new persistence layer needed

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode stores messages at `~/.local/share/opencode/storage/message/{sessionID}/` (verified: ls command showed files)
- ✅ OpenCode API endpoint exists at `/session/:sessionID/message` (verified: read server.ts source)
- ✅ Dashboard currently filters global events by sessionID (verified: read activity-tab.svelte source)

**What's untested:**

- ⚠️ API response time for sessions with 1000+ messages (not benchmarked)
- ⚠️ Memory impact of storing historical events per-agent in browser (not measured)
- ⚠️ Cross-project session fetching may need x-opencode-directory header (not verified)

**What would change this:**

- If OpenCode API returns messages slowly (>500ms), might need server-side caching in orch-go
- If browser memory becomes an issue with large histories, would need IndexedDB or pagination
- If cross-project sessions need special handling, would need orch-go proxy

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Hybrid SSE + API Fetch Architecture** - Keep SSE for real-time updates, add API fetch for historical events when activity tab opens.

**Why this approach:**
- OpenCode already persists everything - don't duplicate storage
- SSE provides real-time updates with low latency
- API fetch loads complete history on-demand (lazy loading)
- Minimal changes to existing dashboard code

**Trade-offs accepted:**
- First activity tab open will have a loading delay (fetch in progress)
- Requires OpenCode server to be running for historical data
- Why acceptable: OpenCode is always running when dashboard is useful; loading delay is acceptable UX

**Implementation sequence:**
1. Add API fetch function in activity-tab.svelte that calls OpenCode API
2. Transform MessageV2.WithParts[] to SSEEvent[] format
3. Merge API results with existing SSE events (deduplicate by part ID)
4. Cache per-session to avoid refetching on tab switch

### Alternative Approaches Considered

**Option B: orch-go Backend Caching**
- **Pros:** Could aggregate across projects, single source for dashboard
- **Cons:** Duplicates OpenCode storage, adds complexity, requires orch-go to track all sessions
- **When to use instead:** If we need cross-project aggregation or OpenCode API becomes a bottleneck

**Option C: IndexedDB Browser Persistence**
- **Pros:** Survives page refresh without API call, works offline
- **Cons:** Sync complexity (keeping browser in sync with OpenCode), duplicates storage, per-browser state
- **When to use instead:** If offline access becomes important or API calls are too slow

**Option D: Increase Global Event Buffer**
- **Pros:** Simplest change (just increase 1000 to 10000)
- **Cons:** Still loses on refresh, still dilutes across agents, memory-heavy
- **When to use instead:** Never - this is a band-aid

**Rationale for recommendation:** Option A respects OpenCode as source of truth, requires minimal new code, and follows the principle that dashboards should be thin presentation layers over existing data.

---

### Implementation Details

**What to implement first:**
1. API fetch function in orch-go serve to proxy OpenCode session messages (avoids CORS issues)
2. Transform function to convert MessageV2.Part[] to SSEEvent[] format
3. Store fetched events per-session in agents.ts (not global)
4. Loading state in activity tab while fetching

**Things to watch out for:**
- ⚠️ CORS: Fetching directly from OpenCode (localhost:4096) may fail due to CORS. Route through orch-go proxy.
- ⚠️ Deduplication: SSE events streaming in while API fetch completes - dedupe by part.id
- ⚠️ Order: API returns messages in chronological order; need to merge correctly with SSE events
- ⚠️ Cross-project: May need x-opencode-directory header for agents in other projects

**Areas needing further investigation:**
- How to handle very long sessions (pagination vs infinite scroll)
- Whether to preload history on dashboard load or lazy-load on tab open
- How to handle sessions from projects other than current one

**Success criteria:**
- ✅ Opening activity tab for any agent shows full session history
- ✅ Page refresh doesn't lose historical events
- ✅ Real-time SSE updates continue to work
- ✅ No noticeable performance degradation

---

## File Targets

**Files to create:**
- None required

**Files to modify:**
- `cmd/orch/serve_agents.go` - Add `/api/agents/:id/messages` endpoint that proxies to OpenCode
- `web/src/lib/stores/agents.ts` - Add per-agent event history storage and fetch function
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Fetch history on mount, merge with SSE

**Acceptance Criteria:**
- [ ] Activity tab shows loading indicator while fetching history
- [ ] After fetch completes, all session events are displayed
- [ ] New SSE events continue to appear in real-time
- [ ] Switching agents and back shows cached history
- [ ] Page refresh can refetch history (no data loss)

**Out of Scope:**
- Browser-side persistence (IndexedDB)
- Cross-project session aggregation
- Pagination/infinite scroll for very long sessions
- Export/download of activity history

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - Current event storage implementation
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Activity tab filtering
- `opencode/src/session/index.ts` - OpenCode session management
- `opencode/src/server/server.ts` - OpenCode API endpoints
- `opencode/src/session/message-v2.ts` - Message/Part types
- `opencode/src/storage/storage.ts` - OpenCode disk storage

**Commands Run:**
```bash
# Check OpenCode storage structure
ls ~/.local/share/opencode/storage/
# Output: message migration part project session session_diff todo

# Check session message files
ls ~/.local/share/opencode/storage/message/ | head -20
# Output: ses_466c3cf12ffeqq8yuKoDU108mH...
```

**Related Artifacts:**
- **Decision:** Dashboard uses SYNTHESIS.md as fallback for untracked agent completion detection
- **Investigation:** `.kb/investigations/2025-12-22-inv-dashboard-agent-activity-visibility.md`
- **Constraint:** Dashboard SSE connections can exhaust HTTP/1.1 browser connection pool (6 per origin)

---

## Investigation History

**2026-01-07 09:00:** Investigation started
- Initial question: How to prevent activity feed event loss
- Context: Events are stored globally (1000 limit), diluted across agents, lost on refresh

**2026-01-07 09:30:** Discovered OpenCode persistence
- OpenCode stores all session data to ~/.local/share/opencode/storage/
- API endpoint exists: GET /session/:sessionID/message

**2026-01-07 10:00:** Investigation completed
- Status: Complete
- Key outcome: Hybrid SSE + API architecture recommended - fetch historical events from OpenCode API on-demand

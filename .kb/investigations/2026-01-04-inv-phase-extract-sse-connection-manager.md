<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** SSE connection logic extracted from agents.ts and agentlog.ts into shared service at web/src/lib/services/sse-connection.ts.

**Evidence:** Build succeeds, TypeScript check passes, duplicate SSE patterns (generation counters, reconnect timers, event source management) consolidated into single module.

**Knowledge:** The duplicate patterns (70+ lines each) can be unified with callbacks for domain-specific handling while keeping the connection lifecycle in one place.

**Next:** None - Phase 1 complete. Phases 2-3 (StatsBar extraction, status model consolidation) remain as future work.

---

# Investigation: Phase 1 - Extract SSE Connection Manager

**Question:** How to extract duplicate SSE connection logic from agents.ts and agentlog.ts into a shared service?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Both Files Had Nearly Identical SSE Connection Patterns

**Evidence:** Duplicate code in both files:
- `connectionGeneration` counter for stale timer handling
- `reconnectTimeout` with 5-second delay
- `EventSource` lifecycle (create, onopen, onerror, close)
- Generation check in onerror to prevent stale reconnects

**Source:**
- `web/src/lib/stores/agents.ts:306-426` (before refactor)
- `web/src/lib/stores/agentlog.ts:107-221` (before refactor)

**Significance:** This duplication was the root cause of 10 SSE-related fix commits. Changes to SSE handling had to be made in both places, increasing bug surface.

---

### Finding 2: Domain Logic is Separable from Connection Management

**Evidence:** The event handlers (handleSSEEvent, agentlog listeners) contain domain-specific logic that should stay in their respective stores, while connection lifecycle is generic.

**Source:**
- `agents.ts:handleSSEEvent` - Processes agent state from SSE events
- `agentlog.ts:buildAgentlogEventListeners` - Adds events to agentlog store

**Significance:** Clean separation allows shared service to handle infrastructure (connect/disconnect/reconnect) while domain stores handle event semantics.

---

### Finding 3: Service API Design

**Evidence:** Created API with callbacks and reactive status:
```typescript
interface SSEConnectionOptions {
  onOpen?: () => void;
  onMessage?: (event: MessageEvent) => void;
  onDisconnect?: () => void;
  eventListeners?: Record<string, (event: MessageEvent) => void>;
  reconnectDelayMs?: number;
  autoReconnect?: boolean;
}
```

**Source:** `web/src/lib/services/sse-connection.ts:5-18`

**Significance:** Flexible API supports both generic onMessage handler and custom event listeners, matching the different needs of agents.ts (generic + custom) and agentlog.ts (custom only).

---

## Synthesis

**Key Insights:**

1. **Infrastructure vs Domain** - SSE connection management is infrastructure; event handling is domain. This separation makes each concern testable and maintainable independently.

2. **Callback-based Composition** - Using callbacks allows the shared service to be agnostic to what happens on connect/disconnect, while giving callers full control.

3. **Reactive Status** - Exposing connection status as a Svelte store enables components to react to connection state changes without custom wiring.

**Answer to Investigation Question:**

Extract SSE connection logic by creating a factory function (`createSSEConnection`) that encapsulates EventSource lifecycle, generation counter, and reconnect logic. Callers provide callbacks for domain-specific behavior. This eliminated ~70 lines of duplicate code and centralizes future SSE-related fixes.

---

## Structured Uncertainty

**What's tested:**

- ✅ TypeScript compilation succeeds (verified: `npm run check` passes for modified files)
- ✅ Build succeeds (verified: `npm run build` produces valid output)
- ✅ Duplicate patterns removed (verified: `grep` for old patterns returns empty)

**What's untested:**

- ⚠️ Runtime behavior unchanged (not validated with running server)
- ⚠️ Reconnection works as expected (not tested with network interruption)
- ⚠️ Memory leaks from subscription patterns (not profiled)

**What would change this:**

- If runtime shows connection issues, may need to add more lifecycle hooks
- If memory profiling shows leaks, subscription cleanup may need adjustment
- If future SSE needs require different reconnect strategies, may need strategy pattern

---

## Implementation Recommendations

**Purpose:** Document what was implemented for Phase 1 of the dashboard UI hotspots refactor.

### Implemented Approach

**Shared SSE Connection Service** - Factory function that creates managed SSE connections with automatic reconnection.

**Implementation sequence completed:**
1. Created `web/src/lib/services/sse-connection.ts` with `createSSEConnection` factory
2. Updated `agents.ts` to use shared service, keeping `handleSSEEvent` domain logic
3. Updated `agentlog.ts` to use shared service, keeping event listener domain logic

### Success criteria (achieved):

- ✅ Duplicate SSE patterns consolidated (agents.ts -32 lines, agentlog.ts -38 lines)
- ✅ Build passes with no new TypeScript errors
- ✅ Domain-specific event handling preserved in original stores

---

## References

**Files Examined:**
- `web/src/lib/stores/agents.ts` - Primary SSE consumer (612 lines before)
- `web/src/lib/stores/agentlog.ts` - Secondary SSE consumer (222 lines before)
- `.kb/investigations/2026-01-04-inv-analyze-dashboard-ui-hotspots-page.md` - Parent investigation

**Files Created/Modified:**
- `web/src/lib/services/sse-connection.ts` (created, 171 lines)
- `web/src/lib/stores/agents.ts` (modified, now 580 lines)
- `web/src/lib/stores/agentlog.ts` (modified, now 184 lines)

**Commands Run:**
```bash
# Check for TypeScript errors
npm run check

# Verify build succeeds
npm run build

# Verify duplicate patterns removed
grep -E "connectionGeneration|reconnectTimeout" web/src/lib/stores/*.ts
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-04-inv-analyze-dashboard-ui-hotspots-page.md` - Parent hotspot analysis
- **Commit:** `8c4ea99b` - Implements this extraction

---

## Investigation History

**2026-01-04 14:00:** Investigation started
- Initial question: How to extract duplicate SSE connection logic?
- Context: Phase 1 of dashboard UI hotspots refactor (addressing 10/32 fix commits)

**2026-01-04 14:30:** Implementation complete
- Created shared service with factory pattern
- Migrated both consumers to use shared service
- Build and TypeScript checks pass

**2026-01-04 14:35:** Investigation completed
- Status: Complete
- Key outcome: SSE connection logic consolidated, 70+ duplicate lines eliminated

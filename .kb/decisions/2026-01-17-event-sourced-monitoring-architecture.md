# Decision: Event-Sourced Monitoring Architecture

**Status:** Accepted
**Date:** 2026-01-17
**Deciders:** Architect synthesis from 8 investigations
**Supersedes:** None
**Synthesized from:** `.kb/investigations/2026-01-17-inv-synthesize-sse-investigation-cluster-investigations.md`

---

## Summary

SSE-based event-sourced monitoring is the required architecture for real-time agent state observation in orch-go. This decision documents the settled architecture, constraints, and patterns that emerged from 8 investigations spanning 2025-12-19 to 2026-01-05.

---

## Context

orch-go needs real-time visibility into agent state for:
- Completion detection (when to release slots, send notifications)
- Dashboard updates (live activity, agent status)
- Slot management (track headless agent capacity)

OpenCode exposes a `/event` SSE endpoint but not session state via REST API. This constrains the solution space.

---

## Decision

### Architecture

Three-layer architecture for SSE-based monitoring:

```
┌────────────────────────────────────────────────────────────────┐
│ Layer 3: Service Integration                                    │
│ ├── Backend: pkg/opencode/service.go                           │
│ │   └── Notifications, registry updates, beads phase updates   │
│ └── Frontend: web/src/lib/stores/*.ts                          │
│     └── Agent state, activity feed, deduplication              │
├────────────────────────────────────────────────────────────────┤
│ Layer 2: State Tracking                                         │
│ ├── Backend: pkg/opencode/monitor.go                           │
│ │   └── WasBusy tracking, completion detection, reconnection   │
│ └── Frontend: web/src/lib/services/sse-connection.ts           │
│     └── Generation counters, AbortController, debouncing       │
├────────────────────────────────────────────────────────────────┤
│ Layer 1: Parsing                                                │
│ └── Backend: pkg/opencode/sse.go                               │
│     └── SSE format, JSON extraction, backward compatibility    │
└────────────────────────────────────────────────────────────────┘
```

### Key Constraints

These are **hard constraints**, not preferences:

| Constraint | Why | Source |
|------------|-----|--------|
| Completion = busy→idle transition | OpenCode HTTP API doesn't expose session state | OpenCode design |
| HTTP/1.1: 6 connections per origin | Browser limitation; SSE occupies slots | Browser standard |
| Non-critical SSE must be opt-in | Connection scarcity (see above) | Jan 5 investigation |
| Race prevention via generation counters | Stale timers/fetches inevitable with reconnection | Dec 25 investigation |
| Handle OpenCode event quirks | Type inside JSON data, nested structures | Dec 19 investigation |

### Settled Patterns

**Completion Detection:**
```
Monitor tracks per-session state:
  session.status "busy"  → WasBusy = true
  session.status "idle"  → if WasBusy: fire completion handler
```

**Race Prevention (Frontend):**
```typescript
// Generation counter pattern
let connectionGeneration = 0;
function connect() {
  connectionGeneration++;
  const thisGeneration = connectionGeneration;
  // ... later in timer/callback:
  if (connectionGeneration !== thisGeneration) return; // stale
}
```

**Deduplication:**
```typescript
// Use OpenCode's stable part.id
function addOrUpdateEvent(event) {
  const existingIndex = events.findIndex(e => e.id === event.part?.id);
  if (existingIndex >= 0) {
    events[existingIndex] = event; // update in place
  } else {
    events.push(event);
  }
}
```

**Connection Priority:**
```
Primary SSE (events)     → Auto-connect on page load
Secondary SSE (agentlog) → Opt-in via "Follow" button
```

---

## Consequences

### Positive

- **Real-time updates** without polling overhead
- **Completion detection** works reliably for headless agents
- **Layered design** allows changes at each level independently
- **Race conditions** handled systematically

### Negative

- **HTTP/1.1 connection scarcity** limits concurrent SSE streams
- **OpenCode format dependency** - changes require parser updates
- **State management complexity** - generation counters required everywhere

### Neutral

- **HTTP/2 would change constraint** - multiplexing removes connection scarcity
- **Stale investigation prevention** - future agents reference this, not individual investigations

---

## Implementation Files

**Backend (Go):**
- `pkg/opencode/sse.go` (~159 lines) - Layer 1 parsing
- `pkg/opencode/monitor.go` (~221 lines) - Layer 2 state tracking
- `pkg/opencode/service.go` - Layer 3 service integration

**Frontend (TypeScript):**
- `web/src/lib/services/sse-connection.ts` (~171 lines) - Shared connection management
- `web/src/lib/stores/agents.ts` - Agent state + SSE handling
- `web/src/lib/stores/agentlog.ts` - Agentlog SSE handling

---

## Related

- **Guide:** `.kb/guides/opencode.md` - Procedural OpenCode usage
- **Model:** `.kb/models/opencode-session-lifecycle.md` - Session state transitions
- **Existing constraint:** "SSE busy->idle cannot detect true agent completion" (kb context)

---

## Change Log

| Date | Change | Reason |
|------|--------|--------|
| 2026-01-17 | Decision created | Synthesized from 8 investigations to prevent re-investigation |

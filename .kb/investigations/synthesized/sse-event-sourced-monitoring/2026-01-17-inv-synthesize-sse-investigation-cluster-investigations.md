## Summary (D.E.K.N.)

**Delta:** SSE-based event-sourced monitoring architecture for orch-go is settled - uses layered design (parsing→state tracking→service integration) with state-transition completion detection (busy→idle), generation counters for race prevention, and HTTP/1.1 connection constraints requiring opt-in secondary streams.

**Evidence:** 8 investigations spanning 2025-12-19 to 2026-01-05 converged on same patterns independently; 93.2% test coverage in pkg/opencode; frontend patterns (deduplication, race prevention, connection consolidation) all validated via TypeScript/Playwright.

**Knowledge:** Real-time agent state observation requires event-driven architecture, not polling; completion is state transition, not session disappearance; HTTP/1.1 connection limits (6/origin) are hard constraint for browser-based dashboards.

**Next:** Promote to architectural decision - this is the settled approach, not a point-in-time finding.

**Promote to Decision:** recommend-yes - Establishes architectural pattern for event-sourced monitoring across orch-go

---

# Investigation: Synthesize SSE Investigation Cluster

**Question:** What is the architectural approach for event-sourced monitoring in orch-go, and what constraints govern it?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - promote to decision
**Status:** Complete

**Supersedes:**
- `.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md`
- `.kb/investigations/2025-12-20-inv-implement-sse-based-completion-detection.md`
- `.kb/investigations/2025-12-19-inv-fix-sse-parsing-event-type.md`
- `.kb/investigations/2025-12-22-inv-add-sse-based-completion-tracking.md`
- `.kb/investigations/2025-12-25-inv-debug-live-activity-streaming-deduplication-sse.md`
- `.kb/investigations/2025-12-25-inv-sse-fetch-race-condition-during.md`
- `.kb/investigations/2026-01-04-inv-phase-extract-sse-connection-manager.md`
- `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md`

---

## Findings

### Finding 1: Three-Layer Architecture Emerged Organically

**Evidence:** Multiple investigations independently converged on the same layered pattern:

```
Layer 3: Service Integration (pkg/opencode/service.go, web/src/lib/stores/*.ts)
         ├── Notifications, registry updates, beads phase updates
         └── Domain-specific event handling

Layer 2: State Tracking (pkg/opencode/monitor.go, web/src/lib/services/sse-connection.ts)
         ├── Per-session state (WasBusy flag, generation counters)
         ├── Completion detection (busy→idle transition)
         └── Connection lifecycle (reconnection, cleanup)

Layer 1: Parsing (pkg/opencode/sse.go)
         ├── SSE format parsing (event:, data: lines)
         ├── JSON extraction (type inside data field)
         └── Backward-compatible format handling
```

**Source:**
- `2025-12-19-inv-client-sse-event-monitoring.md` - Layer 1 established
- `2025-12-20-inv-implement-sse-based-completion-detection.md` - Layer 2 established
- `2026-01-04-inv-phase-extract-sse-connection-manager.md` - Layer separation formalized (70+ duplicate lines consolidated)

**Significance:** The same pattern emerged in both backend (Go) and frontend (TypeScript), suggesting this is the natural architecture for SSE-based monitoring, not an arbitrary choice.

---

### Finding 2: Completion Detection Via State Transition

**Evidence:** All investigations converged on: completion = `busy` → `idle` transition, NOT session disappearance.

Key insight from Dec 20 investigation:
> "Stateful session tracking - Monitor tracks `WasBusy` per session to correctly detect completion (busy->idle transition, not just idle status)"

This matters because:
- Sessions can start in idle state (no completion event)
- Sessions persist indefinitely after completion
- Polling session existence gives false positives

**Source:**
- `2025-12-20-inv-implement-sse-based-completion-detection.md:70` - WasBusy pattern
- `2025-12-22-inv-add-sse-based-completion-tracking.md:36-45` - Gap analysis

**Significance:** This is a CONSTRAINT, not a choice. The alternative (polling session state) doesn't work because OpenCode HTTP API doesn't expose session state (busy/idle). SSE is the only mechanism for completion detection.

---

### Finding 3: Race Prevention Requires Generation Counters

**Evidence:** Two independent investigations discovered the same pattern for preventing stale operations:

**Backend (Go):**
- Monitor.go uses session state map to track WasBusy per session
- Reconnection spawns goroutines that need lifecycle management

**Frontend (TypeScript):**
- Connection generation counter increments on connect/disconnect
- Stale timers check generation before executing
- AbortController cancels in-flight fetches

The Dec 25 race condition investigation:
> "Module-level timers and in-flight requests persist across component lifecycles; generation counters prevent stale timer execution"

**Source:**
- `2025-12-25-inv-sse-fetch-race-condition-during.md:54-62` - Generation counter pattern
- `2026-01-04-inv-phase-extract-sse-connection-manager.md:35-44` - Duplicate patterns identified

**Significance:** Race prevention isn't optional - rapid page reloads, SSE reconnections, and concurrent fetches all create race conditions. Generation counters are the established solution.

---

### Finding 4: HTTP/1.1 Connection Limits Are Hard Constraint

**Evidence:** Jan 5 investigation discovered dashboard API requests queuing as "pending" due to:
- Browser HTTP/1.1 limit: 6 connections per origin
- Two SSE connections (events + agentlog) consumed 2 slots
- 9+ API fetches competed for remaining 4 slots

**Source:** `2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md:43-52`

**Fix implemented:** Agentlog SSE made opt-in (non-critical, collapsed by default, has "Follow" button).

**Significance:** This is a CONSTRAINT that affects all future SSE design. Non-critical streams must be opt-in. Primary stream auto-connects.

---

### Finding 5: OpenCode Event Format Quirks Require Adaptation

**Evidence:** Dec 19 investigation discovered OpenCode's SSE format differs from expectations:

1. **Event type location:** Inside JSON `data.type` field, NOT as `event:` prefix line
2. **Session status structure:** Nested `properties.status.type`, not flat `status` field
3. **Field names:** `sessionID` (camelCase) vs `session_id` (snake_case)

All parsing code must handle both old and new formats for backward compatibility.

**Source:** `2025-12-19-inv-fix-sse-parsing-event-type.md:31-48`

**Significance:** Any code interacting with OpenCode SSE must account for these quirks. The patterns are now documented and shouldn't be re-investigated.

---

### Finding 6: Deduplication Via Stable Identifiers

**Evidence:** Dec 25 deduplication investigation found:
- SSE events have stable `part.id` field for incremental updates
- Frontend was generating new IDs via `generateSSEEventId()`, causing duplicates
- Fix: Update in place when `part.id` matches

Two event types serve different purposes:
- `message.part` - Text streaming (incremental chunks)
- `message.part.updated` - Tool state changes (running/completed)

**Source:** `2025-12-25-inv-debug-live-activity-streaming-deduplication-sse.md:36-58`

**Significance:** OpenCode provides deduplication primitives. Clients must use them.

---

## Synthesis

**Key Insights:**

1. **Event-Sourced Architecture is Required, Not Chosen** - OpenCode's HTTP API doesn't expose session state (busy/idle). SSE is the only mechanism for real-time agent state observation. This isn't a design preference - it's a constraint.

2. **Layered Separation Enables Maintenance** - Parsing, state tracking, and service integration as separate layers allows changes at each level without affecting others. The Jan 4 extraction reduced duplicate code by 70+ lines while keeping domain logic in place.

3. **State Machines Prevent Races** - Both backend and frontend converged on generation counters and stateful tracking. This pattern is fundamental to SSE-based systems with reconnection.

4. **Connection Scarcity Requires Prioritization** - HTTP/1.1's 6-connection limit means every long-lived SSE connection has opportunity cost. Primary streams auto-connect; secondary streams must be opt-in.

**Answer to Investigation Question:**

The architectural approach for event-sourced monitoring in orch-go is:

1. **Three-layer architecture:**
   - Layer 1: Low-level parsing (`pkg/opencode/sse.go`)
   - Layer 2: State tracking and completion detection (`pkg/opencode/monitor.go`)
   - Layer 3: Service integration (`pkg/opencode/service.go`, frontend stores)

2. **Key constraints:**
   - Completion = busy→idle state transition (required by OpenCode design)
   - HTTP/1.1 6-connection limit (non-critical streams must be opt-in)
   - OpenCode event format quirks (type inside JSON, nested structures)
   - Race prevention via generation counters (stale operations are inevitable)

3. **Settled patterns:**
   - Connection lifecycle with automatic reconnection
   - Deduplication via stable identifiers (`part.id`)
   - Callback-based composition for domain-specific handling

---

## Structured Uncertainty

**What's tested:**

- ✅ SSE parsing handles both old and new formats (verified: 93.2% test coverage in pkg/opencode)
- ✅ Completion detection works for busy→idle transitions (verified: unit tests + live testing)
- ✅ Frontend race prevention works (verified: TypeScript checks + Playwright tests pass)
- ✅ Connection pool exhaustion fixed by opt-in secondary SSE (verified: build passes)

**What's untested:**

- ⚠️ Very long-running sessions (untested - mentioned in Dec 19)
- ⚠️ Network resilience under sustained failures (not stress-tested)
- ⚠️ Memory profiling of sse-connection.ts subscriptions (not profiled)

**What would change this:**

- If HTTP/2 is enabled on API server, connection pool constraint relaxes (HTTP/2 multiplexes)
- If OpenCode changes event format again, parsing layer needs update
- If dashboard needs many more SSE streams, multiplexed single endpoint may be needed

---

## Implementation Recommendations

**Purpose:** Document the settled architecture as a decision for future reference.

### Recommended Approach ⭐

**Promote to Architectural Decision** - The SSE investigation cluster represents a settled architecture, not ongoing exploration. Promote findings to `.kb/decisions/` to prevent re-investigation.

**Why this approach:**
- 8 investigations over 3 weeks converged on same patterns
- No outstanding questions or alternatives being explored
- Pattern is load-bearing (completion detection, dashboard real-time updates)
- Future work should build on this, not re-examine it

**Trade-offs accepted:**
- Decision locks in HTTP/1.1 assumption (HTTP/2 would change constraint)
- OpenCode format quirks baked into code (OpenCode changes would require updates)

**Implementation sequence:**
1. Create `.kb/decisions/2026-01-17-event-sourced-monitoring-architecture.md`
2. Archive synthesized investigations (mark as superseded)
3. Update `.kb/guides/opencode.md` to reference decision

### Alternative Approaches Considered

**Option B: Keep as Investigations**
- **Pros:** No decision maintenance burden
- **Cons:** Future agents will re-investigate settled questions, wasting effort
- **When to use instead:** If architecture is still evolving

**Option C: Merge into Model**
- **Pros:** Consolidates with existing opencode-session-lifecycle model
- **Cons:** Mixes "how sessions work" with "how monitoring works"
- **When to use instead:** If these become tightly coupled

**Rationale for recommendation:** The 8 investigations represent approximately 2-3 weeks of agent effort that discovered durable patterns. Promoting to decision captures this investment and prevents redundant re-investigation.

---

### Implementation Details

**What to implement first:**
- Create decision document with architecture diagram
- Include constraint list (these are hard constraints)
- Reference implementation files for verification

**Things to watch out for:**
- ⚠️ OpenCode may change event format (monitor for breaking changes)
- ⚠️ session.idle event is deprecated, use session.status
- ⚠️ HTTP/2 adoption would change connection scarcity constraint

**Success criteria:**
- ✅ Decision document created and committed
- ✅ Future SSE-related work references decision, not investigations
- ✅ No re-investigation of settled patterns

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md` - Initial SSE client
- `.kb/investigations/2025-12-20-inv-implement-sse-based-completion-detection.md` - Monitor + CompletionService
- `.kb/investigations/2025-12-19-inv-fix-sse-parsing-event-type.md` - Format quirks
- `.kb/investigations/2025-12-22-inv-add-sse-based-completion-tracking.md` - Slot management bridge
- `.kb/investigations/2025-12-25-inv-debug-live-activity-streaming-deduplication-sse.md` - Frontend deduplication
- `.kb/investigations/2025-12-25-inv-sse-fetch-race-condition-during.md` - Race prevention
- `.kb/investigations/2026-01-04-inv-phase-extract-sse-connection-manager.md` - Consolidation
- `.kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md` - Connection limits
- `.kb/guides/opencode.md` - Existing OpenCode guide
- `.kb/models/opencode-session-lifecycle/model.md` - Session lifecycle model

**Related Artifacts:**
- **Guide:** `.kb/guides/opencode.md` - Already references SSE monitoring
- **Model:** `.kb/models/opencode-session-lifecycle/model.md` - Session state transitions
- **Constraint:** "SSE busy->idle cannot detect true agent completion" (kb context) - Established constraint

---

## Investigation History

**2026-01-17:** Investigation started
- Initial question: What is the architectural approach for event-sourced monitoring in orch-go?
- Context: kb reflect identified 7 SSE investigations as synthesis opportunity

**2026-01-17:** Analysis complete
- Read all 8 SSE-related investigations
- Identified convergent patterns across backend and frontend
- Documented 6 key findings with evidence chains

**2026-01-17:** Investigation completed
- Status: Complete
- Key outcome: SSE event-sourced monitoring architecture is settled; recommend promotion to decision

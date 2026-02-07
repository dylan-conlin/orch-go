## Summary (D.E.K.N.)

**Delta:** Created CompletionService that bridges SSE completion detection with slot management for headless agents.

**Evidence:** 13 unit tests pass, covering tracking, slot release, event handlers, and concurrent access.

**Knowledge:** Headless agents need explicit session→slot mapping since slots are acquired before session creation; Monitor already handles SSE reconnection.

**Next:** Integrate CompletionService into daemon and spawn workflows for production use.

**Confidence:** High (90%) - Core functionality tested but no integration test with live SSE stream.

---

# Investigation: SSE-based Completion Tracking for Headless Agents

**Question:** How to track headless agent completions via SSE and release slots when they complete?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Gap between SSE completion detection and slot management

**Evidence:** 
- `pkg/opencode/monitor.go` detects session completions via SSE (session.status: busy→idle)
- `pkg/daemon/pool.go` tracks slots with BeadsID but not SessionID
- No existing link between session completion and slot release

**Source:** 
- `pkg/opencode/monitor.go:136-189` - handleEvent and completion detection
- `pkg/daemon/pool.go:22-26` - Slot struct with BeadsID only

**Significance:** Headless agents complete without releasing their slots, causing capacity exhaustion.

---

### Finding 2: Slot acquisition happens before session creation

**Evidence:**
- `runSpawnHeadless` in main.go:1067-1103 creates session after concurrency check
- Session ID is only available after `client.CreateSession()` returns
- Slot must be acquired before spawn to enforce limits

**Source:** `cmd/orch/main.go:1067-1103`

**Significance:** CompletionService must accept slot tracking after session creation (two-phase: acquire slot, then Track(sessionID, slot)).

---

### Finding 3: Monitor already handles SSE reconnection

**Evidence:**
- `Monitor.reconnect()` in monitor.go:109-134 handles reconnection with goroutines
- Events channel is repopulated on reconnect
- No additional reconnection logic needed in CompletionService

**Source:** `pkg/opencode/monitor.go:109-134`

**Significance:** CompletionService delegates SSE handling to Monitor, simplifying implementation.

---

## Synthesis

**Key Insights:**

1. **Two-phase tracking pattern** - Slots are acquired before spawn, sessions are tracked after. CompletionService.Track(sessionID, slot) bridges this gap.

2. **Composition over duplication** - CompletionService wraps Monitor rather than reimplementing SSE handling, enabling future extension without SSE complexity.

3. **Event-driven slot release** - Instead of polling for completion, SSE events trigger immediate slot release, enabling faster capacity recycling.

**Answer to Investigation Question:**

CompletionService provides the missing link between SSE-based completion detection and slot management. It:
1. Maintains sessionID→slot mapping
2. Subscribes to Monitor completion events
3. Releases slots when tracked sessions complete
4. Emits completion events for external consumers (e.g., logging, metrics)

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All unit tests pass (13/13). The design follows existing patterns (Monitor, WorkerPool). Core functionality verified.

**What's certain:**

- ✅ Track/Untrack/IsTracked operations work correctly
- ✅ Slot release on completion works via handleCompletion()
- ✅ Multiple handlers can be registered and all receive events
- ✅ Concurrent access is thread-safe (tested with 100 goroutines)

**What's uncertain:**

- ⚠️ No integration test with live SSE stream
- ⚠️ Reconnection behavior not tested end-to-end
- ⚠️ Not yet integrated into spawn workflow

**What would increase confidence to Very High (95%+):**

- Integration test with mock SSE server
- End-to-end test in daemon workflow
- Production observation

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use CompletionService in headless spawn path** - After creating session, Track(sessionID, beadsID, slot) to enable automatic slot release on completion.

**Why this approach:**
- Minimal changes to existing spawn code (one Track() call)
- Non-breaking for tmux/inline spawns (they don't acquire slots)
- Immediate benefit: slots released on completion instead of never

**Trade-offs accepted:**
- Requires daemon to start CompletionService before spawning
- Session must be tracked after creation (not before)

**Implementation sequence:**
1. Create CompletionService in daemon startup
2. Pass CompletionService to spawn functions
3. After CreateSession(), call Track(sessionID, beadsID, slot)

### Alternative Approaches Considered

**Option B: Polling-based completion detection**
- **Pros:** Simpler, no SSE dependency
- **Cons:** Higher latency, more API calls, doesn't scale
- **When to use instead:** If SSE becomes unreliable

**Option C: Embed slot tracking in WorkerPool**
- **Pros:** Single component manages everything
- **Cons:** WorkerPool doesn't know about sessions, violates SRP
- **When to use instead:** If CompletionService proves too complex

**Rationale for recommendation:** SSE-based approach is real-time, efficient, and aligns with existing Monitor infrastructure.

---

### Implementation Details

**What to implement first:**
- Integrate CompletionService into daemon startup
- Modify runSpawnHeadless to Track sessions with slots

**Things to watch out for:**
- ⚠️ Must call Untrack() if spawn fails after Track()
- ⚠️ Don't Track() tmux/inline spawns (they don't use slots)
- ⚠️ Ensure CompletionService.Start() before any spawns

**Areas needing further investigation:**
- How to handle orphaned sessions (spawned but never tracked)
- Stale session cleanup (session tracked but SSE never reports completion)

**Success criteria:**
- ✅ Headless agent slots released within seconds of completion
- ✅ No slot leaks after 10+ agent completions
- ✅ Capacity manager reports accurate in-flight counts

---

## References

**Files Examined:**
- `pkg/opencode/monitor.go` - Existing SSE completion detection
- `pkg/opencode/sse.go` - SSE parsing utilities
- `pkg/daemon/pool.go` - WorkerPool slot management
- `pkg/daemon/daemon.go` - Daemon spawn workflow
- `pkg/capacity/manager.go` - Multi-account capacity management
- `cmd/orch/main.go` - Spawn command implementation

**Commands Run:**
```bash
# Verify package builds
go build ./pkg/daemon/...

# Run CompletionService tests
go test ./pkg/daemon/... -run Completion -v

# Run all tests
go test ./...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md` - SSE client design

---

## Investigation History

**2025-12-22 16:30:** Investigation started
- Initial question: How to track headless agent completions via SSE and release slots?
- Context: Headless agents exhaust capacity because slots aren't released on completion

**2025-12-22 16:45:** Analyzed existing codebase
- Found Monitor handles SSE, WorkerPool handles slots, no bridge exists

**2025-12-22 17:00:** Implemented CompletionService
- Created completion.go with Track/Untrack/ReleaseSlot
- Integrated with Monitor for SSE events
- Added completion event handlers

**2025-12-22 17:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: CompletionService bridges SSE completion detection and slot management

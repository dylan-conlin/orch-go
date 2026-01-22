<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Daemon has 5 independent state-tracking systems (WorkerPool, SpawnedIssueTracker, OpenCode API, beads, Docker containers) that can drift from each other, causing stale capacity, wrong status, respawned completed work, and Docker resource exhaustion (23 orphaned containers, 7.7GB OOM).

**Evidence:** Code analysis revealed: WorkerPool reconciles only downward; status detection uses stale timestamps; respawn prevention lacks Phase: Complete check; Docker containers never cleaned up on completion/abandonment; cross-project depends on per-project beads queries.

**Knowledge:** The daemon's architecture creates state in multiple systems but only has cleanup hooks for some; Docker backend is fire-and-forget with no container lifecycle management.

**Next:** (1) Add pre-spawn Phase: Complete check; (2) Add Docker container cleanup to `orch complete` and `orch abandon`; (3) Add reconciliation logging.

**Promote to Decision:** recommend-yes - This reveals an architectural pattern (multi-layer state without authoritative source) that affects reliability at scale.

---

# Investigation: Strategic Audit Daemon Reliability

**Question:** What are the daemon's failure modes for capacity tracking, status detection, respawning completed work, and cross-project visibility?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** architect
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Daemon State Model

The daemon operates with four independent state-tracking systems:

```
┌─────────────────────────────────────────────────────────────────┐
│                     DAEMON STATE LAYERS                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. WorkerPool (internal)                                        │
│     └─ Tracks: activeSlots map[string]SlotInfo                   │
│     └─ Updated by: TryAcquire(), Release(), Reconcile()          │
│     └─ Source: Internal counter                                  │
│                                                                  │
│  2. SpawnedIssueTracker (internal)                               │
│     └─ Tracks: map[string]time.Time (beadsID → spawn time)       │
│     └─ Updated by: Mark(), Unmark(), CleanStale()                │
│     └─ TTL: 6 hours                                              │
│                                                                  │
│  3. OpenCode API (external)                                      │
│     └─ Tracks: Session existence, timestamps, status             │
│     └─ Updated by: Agent activity                                │
│     └─ Accessed via: client.ListSessions(), client.GetSession()  │
│                                                                  │
│  4. Beads (.beads/issues.jsonl)                                  │
│     └─ Tracks: Issue status, Phase comments                      │
│     └─ Updated by: bd commands, agent bd comment                 │
│     └─ Source of truth for: Completion (Phase: Complete)         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight:** These layers have no authoritative master. They're designed for eventual consistency via reconciliation, but reconciliation has gaps.

---

## Findings

### Finding 1: Capacity Tracking Drifts From Reality (WorkerPool Reconciliation)

**Evidence:** The `ReconcileWithOpenCode()` method in `daemon.go:289-291` calls `d.pool.Reconcile(activeCount)`, but `pool.Reconcile()` only adjusts **downward**:

```go
// pool.go:167-179
func (p *WorkerPool) Reconcile(actualActive int) int {
    p.mu.Lock()
    defer p.mu.Unlock()

    if actualActive < len(p.activeSlots) {
        // Count slots that aren't in OpenCode
        toRelease := len(p.activeSlots) - actualActive
        // Release stale slots...
        return released
    }
    return 0 // NEVER increases count
}
```

**Source:** `pkg/daemon/pool.go:167-179`, `cmd/orch/daemon.go:289-291`

**Significance:** If the daemon's internal count is **lower** than reality (e.g., agent spawned outside daemon, race condition), reconciliation won't correct it. The daemon thinks it has capacity when it doesn't.

**Failure modes:**
1. **OpenCode API unavailable:** `DefaultActiveCount()` returns cached/stale data → pool never reconciles
2. **Race between spawn and reconcile:** Spawn acquired slot, OpenCode hasn't registered session yet → reconcile sees fewer active sessions → releases "stale" slot prematurely
3. **Cross-project agents:** `DefaultActiveCount()` filters by current project directory (line 132) → may miss agents in other projects

---

### Finding 2: Status Detection Uses OpenCode Timestamps, Not Agent Activity

**Evidence:** The dashboard (`serve_agents.go:446-458`) determines agent status based on `timeSinceUpdate`:

```go
// serve_agents.go:453-458
status := "active"
if timeSinceUpdate > deadThreshold {    // 3 minutes
    status = "dead"
} else if timeSinceUpdate > activeThreshold {  // 10 minutes
    status = "idle"
}
```

The `timeSinceUpdate` comes from OpenCode's `session.Time.Updated`, which tracks API-level session updates—not agent internal activity.

**Source:** `cmd/orch/serve_agents.go:446-458`, `cmd/orch/serve_agents.go:414` (deadThreshold=3min)

**Significance:** An agent can be actively working (reading files, thinking, etc.) but if the OpenCode session timestamp isn't being updated, it appears "dead" or "idle" to the dashboard. This is the root cause of "shows idle for running agents."

**Specific scenarios:**
1. **Long tool execution:** Agent runs a 5-minute build command → session timestamp not updated → appears dead after 3 minutes
2. **Thinking/reasoning:** Agent is thinking (using extended thinking) → no API updates → appears idle
3. **tmux-only agents:** Claude CLI escape hatch agents have no OpenCode session → fall back to workspace file modification times (`getWorkspaceLastActivity`)

---

### Finding 3: Respawn Prevention Has Gaps - Missing Phase: Complete Check

**Evidence:** The spawn flow has multiple dedup checks, but none check for Phase: Complete in beads comments:

1. **SpawnedIssueTracker** (`spawn_tracker.go`): Only tracks recently-spawned issues, TTL=6 hours
2. **Session dedup** (`session_dedup.go`): Checks if OpenCode session exists with same beads ID in title
3. **Beads status** (`daemon.go:244`): Skips `in_progress` issues
4. **Pre-spawn check** (`pre_spawn_check.go`): Referenced in `agent-lifecycle.md:99-106` but **not actually implemented in current code**

The documented pre-spawn check:
```go
// From .kb/guides/agent-lifecycle.md:99-106 (documentation, NOT code)
comments := bd.GetComments(beadsID)
for _, c := range comments {
    if strings.Contains(c, "Phase: Complete") {
        return errors.New("work already complete, not respawning")
    }
}
```

**Source:** `pkg/daemon/spawn_tracker.go`, `pkg/daemon/session_dedup.go`, `.kb/guides/agent-lifecycle.md:99-106`

**Significance:** An agent that reported Phase: Complete but wasn't closed (orch complete not run) can be respawned:
1. SpawnedIssueTracker entry expires after 6 hours
2. OpenCode session may be deleted (manual cleanup, server restart)
3. Beads status still shows "open" if not closed
4. No check for existing commits (work may already be done)

---

### Finding 4: Cross-Project Visibility Depends on Per-Project Beads Queries

**Evidence:** Cross-project daemon (`daemon.go:489-524`) iterates over kb-registered projects:

```go
// daemon.go:489-491
if config.CrossProject {
    cpResult, err := d.CrossProjectOnceExcluding(skippedThisCycle)
    // ...
}
```

`CrossProjectOnceExcluding()` calls `ListReadyIssuesForProject()` for each project, which shells out to `bd ready --label=triage:ready --json`:

```go
// daemon.go:ListReadyIssuesForProject implementation uses beads.ListReadyIssues()
```

**Source:** `cmd/orch/daemon.go:489-524`, `pkg/daemon/daemon.go:206-216`

**Significance:** Each project query is independent:
1. **Beads daemon unavailable:** RPC fails, falls back to CLI, slower
2. **Project not initialized:** `bd ready` may fail silently
3. **No aggregation:** Each project's issues are queried separately → 10 projects = 10 queries per poll cycle
4. **Label filter inconsistency:** Each project may have different triage labeling conventions

---

### Finding 5: Docker Containers Not Cleaned Up (Resource Exhaustion)

**Evidence:** 23 running Docker containers accumulated over time, exhausting the 7.7GB Docker memory limit. New agent spawns were immediately OOM killed. Containers are created via `orch spawn --backend docker` for fresh Statsig fingerprint isolation but are never cleaned up.

**Source:** `pkg/spawn/docker.go` (creates containers), no corresponding cleanup in `orch complete` or `orch abandon`

**Significance:** This is the same architectural pattern: **state created without lifecycle cleanup**. The daemon tracks agent state in WorkerPool and SpawnedIssueTracker, but Docker containers exist outside this tracking:

1. **No container registry:** Unlike WorkerPool for sessions, there's no container tracking
2. **No cleanup on completion:** `orch complete` closes beads, deletes OpenCode session, but doesn't stop Docker container
3. **No cleanup on abandonment:** `orch abandon` marks agent abandoned but doesn't stop container
4. **No cleanup on crash:** If agent dies, container keeps running until manual intervention

**Failure cascade:**
1. Container created → agent runs → agent completes → `orch complete` runs → container keeps running
2. Over time, 20+ containers accumulate → Docker hits memory limit (7.7GB)
3. New `--backend docker` spawns immediately OOM killed
4. System appears "working" (daemon shows capacity) but all docker spawns fail silently

---

### Finding 6: Completion Detection Relies on Multiple Signals, No Single Authority

**Evidence:** The `determineAgentStatus()` function (`serve_agents.go:1124-1160`) implements a priority cascade:

```go
// Priority 1: Beads issue closed → "completed"
if issueClosed { return "completed" }

// Priority 2: Phase: Complete AND dead → "awaiting-cleanup"
if phaseComplete && isDead { return "awaiting-cleanup" }

// Priority 3: Phase: Complete → "completed"
if phaseComplete { return "completed" }

// Priority 4: SYNTHESIS.md AND dead → "awaiting-cleanup"
if hasSynthesis && isDead { return "awaiting-cleanup" }

// Priority 5: SYNTHESIS.md → "completed"
if hasSynthesis { return "completed" }

// Priority 6: Session status fallback
return sessionStatus
```

**Source:** `cmd/orch/serve_agents.go:1124-1160`

**Significance:** This is the **display** logic, not the **spawn prevention** logic. The daemon's spawn decision doesn't consult this cascade—it only checks:
1. Beads issue status (open/in_progress/closed)
2. SpawnedIssueTracker (recently spawned?)
3. OpenCode session exists with same beads ID?

If an agent is "completed" by display logic (Phase: Complete) but not by spawn logic (beads still open, tracker expired, no session), it gets respawned.

---

## Synthesis

**Key Insights:**

1. **State Model Fragmentation** - The daemon has 5 independent state systems that assume eventual consistency but lack bidirectional reconciliation. WorkerPool only reconciles downward; SpawnedIssueTracker expires without checking completion; OpenCode timestamps don't reflect internal agent activity; Docker containers have no lifecycle management.

2. **Documentation-Reality Gap** - The pre-spawn Phase: Complete check described in `agent-lifecycle.md` is documentation, not code. This explains the "respawned completed work" bug—the protection layer doesn't exist.

3. **Status Detection Blind Spots** - OpenCode session timestamps track API activity, not agent internal state. Long-running operations (builds, thinking) appear as "dead" because no API calls occur. The 3-minute deadThreshold is too aggressive for many legitimate operations.

4. **Cross-Project Architecture Limitations** - Each project is queried independently without aggregation. This scales linearly (N projects = N queries) and fails independently (one project's beads issue doesn't affect others).

5. **Docker Backend Fire-and-Forget** - Docker containers are created via `orch spawn --backend docker` but never cleaned up. No container registry exists; `orch complete` and `orch abandon` don't stop containers. This leads to resource exhaustion (23 orphaned containers, 7.7GB OOM) that manifests as silent spawn failures.

**Answer to Investigation Question:**

The daemon's reliability issues stem from a **multi-source state model without authoritative consensus**:

- **Capacity drift:** WorkerPool only reconciles downward; if actual > tracked, daemon thinks it has more capacity than reality
- **Status wrong:** OpenCode timestamps aren't activity indicators; 3-min dead threshold is too aggressive
- **Respawned completed:** Phase: Complete check is documented but not implemented; SpawnedIssueTracker has 6-hour TTL
- **Cross-project gaps:** Per-project beads queries with no aggregation or cross-project state sharing
- **Docker resource exhaustion:** Containers created but never cleaned up; accumulates until OOM kills new spawns

---

## Structured Uncertainty

**What's tested:**

- ✅ Code paths verified by reading source files
- ✅ State model documented from actual implementation
- ✅ Prior investigation (2026-01-06) confirmed duplicate spawn race condition exists

**What's untested:**

- ⚠️ Frequency of capacity drift in production (not measured)
- ⚠️ Impact of 3-minute deadThreshold on long-running operations (not benchmarked)
- ⚠️ Cross-project beads query latency under load (not profiled)

**What would change this:**

- Finding would be wrong if OpenCode timestamps update more frequently than assumed
- Recommendations would change if pre-spawn check already exists in a different code path
- Architecture would change if beads added cross-project aggregation API

---

## Implementation Recommendations

**Purpose:** Address the most impactful reliability issues with targeted fixes.

### Recommended Approach ⭐

**Add Pre-Spawn Completion Check + Docker Lifecycle + Reconciliation Improvements**

**Why this approach:**
- Directly prevents respawning completed work (highest user-impact bug)
- Prevents Docker resource exhaustion (critical for --backend docker users)
- Non-invasive to existing architecture
- Can be implemented incrementally

**Trade-offs accepted:**
- Doesn't fix fundamental state model fragmentation
- Additional beads query per spawn attempt

**Implementation sequence:**
1. **Add Phase: Complete check before spawn** - Query beads comments for "Phase: Complete" before calling spawnFunc. If found, skip with logging. (High priority)
2. **Add Docker container cleanup to `orch complete`** - Stop and remove container matching workspace name pattern. (High priority - prevents OOM)
3. **Add Docker container cleanup to `orch abandon`** - Same cleanup as complete. (High priority)
4. **Add container registry file** - Write container ID to workspace (e.g., `.container_id`) for reliable cleanup. (Medium priority)
5. **Add commit check before spawn** - Check git log for commits mentioning the beads ID. If work committed, skip. (Medium priority)
6. **Add reconciliation logging** - Log when Reconcile() would have corrected upward (if it could). This surfaces hidden capacity drift. (Diagnostic)
7. **Extend deadThreshold or add nuance** - Consider 5-minute threshold, or check workspace file activity before marking dead. (Low priority, needs validation)

### Alternative Approaches Considered

**Option B: Single source of truth (beads as master)**
- **Pros:** Eliminates state fragmentation
- **Cons:** Beads would need to track session liveness, capacity—scope creep
- **When to use instead:** Major architectural revision

**Option C: Event-sourced state with reconciliation**
- **Pros:** Full audit trail, deterministic replay
- **Cons:** Complex implementation, breaks existing patterns
- **When to use instead:** Building daemon v2 from scratch

**Rationale for recommendation:** Option A provides 80% reliability improvement with 20% effort. The existing architecture is fundamentally sound—it just has gaps in its protection layers.

---

### Implementation Details

**What to implement first (in priority order):**
1. Docker container cleanup in `orch complete` and `orch abandon` - **Critical: prevents OOM**
2. Pre-spawn Phase: Complete check (beads comment query) - prevents respawning
3. Container ID tracking in workspace (`.container_id` file) - enables reliable cleanup

**Things to watch out for:**
- ⚠️ Adding beads query per spawn adds ~100-500ms latency
- ⚠️ Phase: Complete parsing must match exact format agents use
- ⚠️ Git commit check needs to handle cross-project workdir spawns
- ⚠️ Docker cleanup needs to handle container name patterns (workspace name-based)
- ⚠️ Orphan container cleanup (daemon restart loses in-memory tracking) needs scheduled job

**Areas needing further investigation:**
- Actual frequency of capacity drift in production
- Whether OpenCode could expose more accurate activity timestamps
- Beads aggregation API for cross-project visibility
- Docker container naming convention (is it consistent enough for pattern-based cleanup?)

**Success criteria:**
- ✅ No duplicate spawns for issues with Phase: Complete comments
- ✅ No orphaned Docker containers after `orch complete` or `orch abandon`
- ✅ `docker ps` shows only actively-running agent containers
- ✅ Reconciliation logs show when drift would have occurred
- ✅ Dashboard shows accurate status for actively-working agents

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Main daemon loop, reconciliation
- `pkg/daemon/pool.go` - WorkerPool implementation
- `pkg/daemon/spawn_tracker.go` - SpawnedIssueTracker
- `pkg/daemon/session_dedup.go` - Session deduplication
- `pkg/daemon/status.go` - Daemon status tracking
- `pkg/daemon/active_count.go` - Active agent counting
- `pkg/daemon/completion_processing.go` - Completion detection
- `cmd/orch/daemon.go` - Daemon CLI command
- `cmd/orch/serve_agents.go` - Dashboard agent status
- `pkg/spawn/docker.go` - Docker backend spawn (creates containers, no cleanup)
- `.kb/guides/daemon.md` - Daemon guide
- `.kb/guides/agent-lifecycle.md` - Lifecycle documentation
- `.kb/models/daemon-autonomous-operation.md` - Daemon model

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-daemon-spawns-duplicate-agents-same.md` - Duplicate spawn fix (SpawnedIssueTracker)
- **Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md` - Earlier capacity stale report
- **Issue:** `orch-go-gpono` - Docker containers not cleaned up after agent completion/abandonment

---

## Investigation History

**2026-01-22 17:00:** Investigation started
- Initial question: What are daemon's failure modes for capacity, status, respawning, and cross-project visibility?
- Context: Multiple bugs surfaced in single session (orch-go-xfue0, orch-go-8b09d)

**2026-01-22 17:30:** State model documented
- Identified 4 independent state systems
- Found documentation-reality gap for pre-spawn check

**2026-01-22 17:45:** Docker container lifecycle added
- User reported 23 orphaned containers exhausting 7.7GB Docker memory
- Same root cause: state created without lifecycle cleanup
- Added as Finding 5, updated recommendations to prioritize Docker cleanup

**2026-01-22 18:00:** Investigation completed
- Status: Complete
- Key outcome: Daemon reliability issues stem from 5-layer state model without authoritative consensus; Docker container cleanup and pre-spawn Phase: Complete check are highest priority fixes

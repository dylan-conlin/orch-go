# Design: Dashboard Oscillation — Tmux Liveness is Structurally Wrong

**Date:** 2026-02-24
**Phase:** Complete
**Issue:** orch-go-1184
**Type:** Architect design investigation
**Related:** orch-go-1182 (tmux liveness), orch-go-1183 (cache fix), orch-go-1177 (root cause probe)

---

## Design Question

The dashboard oscillates between correct agent status and "unassigned" for claude-backend agents. Two fix attempts (tmux liveness check in orch-go-1182, 10s TTL cache in orch-go-1183) haven't resolved it. Is the tmux liveness approach fundamentally wrong?

## Problem Framing

### Success Criteria

1. Claude-backend agents show stable, correct status in the dashboard
2. Solution complies with the two-lane agent discovery decision
3. No persistent caches (per two-lane decision)
4. No tmux-as-state (per two-lane decision domain boundary table)

### Constraints

- **Two-lane decision** (`.kb/decisions/2026-02-18-two-lane-agent-discovery.md`): "Tmux owns: Presentation layer. Does NOT own: Any state whatsoever"
- **No Local Agent State** principle: No registries, projection DBs, or caches for agent discovery
- **Coherence Over Patches** principle: This is the 3rd fix attempt to the same area
- **In-memory cache** allowed at 1-5s TTL only (decision table)

### Scope

- **IN:** Two-lane compliance, spawn path analysis, alternative liveness signals
- **OUT:** Frontend dashboard changes, tmux package refactoring

---

## Exploration (Fork Navigation)

### Fork 1: Does tmux liveness violate the two-lane decision?

**Options:**
- A: Yes, it violates — tmux is being used as state, which the decision explicitly prohibits
- B: No, it's a reasonable extension — the decision didn't anticipate claude-backend agents

**Substrate says:**
- **Decision** (two-lane, line 39): "Tmux owns: Presentation layer. Does NOT own: Any state whatsoever."
- **Decision** (two-lane, line 156-161): In-memory cache allowed at 1-5s TTL. Current tmux cache uses 10s TTL — exceeds allowed range.
- **Principle** (No Local Agent State): "If queries are slow, fix the authoritative source; do not build a projection."
- **Principle** (Coherence Over Patches): 3rd fix to the same area signals structural issue, not insufficient fixing.

**RECOMMENDATION:** Option A. The violation is clear and intentional to read. The decision says "Any state whatsoever" — checking tmux window existence IS using tmux as a state source. The 10s TTL cache exceeds the allowed 1-5s range. The fact that we needed a cache at all is the tell: if the signal were reliable, we wouldn't need to cache it.

**Evidence from code:**

`query_tracked.go:352-358` — tmux used as liveness oracle:
```go
if manifest.SpawnMode == "claude" && manifest.WorkspaceName != "" {
    if checkTmuxWindowLiveness(manifest.WorkspaceName) {
        agent.Status = "active"
        agent.Reason = "tmux_window_alive"
    } else {
        agent.Status = "dead"
        agent.Reason = "tmux_window_missing"
    }
}
```

`query_tracked.go:40-63` — tmux liveness cache at 10s TTL:
```go
var tmuxLivenessCache = struct {
    mu      sync.RWMutex
    entries map[string]tmuxLivenessCacheEntry
    ttl     time.Duration
}{
    entries: make(map[string]tmuxLivenessCacheEntry),
    ttl:     10 * time.Second,  // Exceeds 1-5s allowed by two-lane decision
}
```

---

### Fork 2: Should runSpawnClaude call AtomicSpawnPhase2?

**Options:**
- A: Yes — write a session_id (even if synthetic) so the existing query path works
- B: No — there's no real OpenCode session, a synthetic ID would create phantom state

**Substrate says:**
- **Decision** (two-lane, atomic spawn section): "No partial state. A half-spawned agent is worse than a failed spawn."
- **Model** (model-access-spawn-paths): The escape hatch exists because "critical infrastructure work can't depend on what might fail." Creating a phantom OpenCode session would re-introduce the OpenCode dependency.
- **Code** (`atomic.go:73`): `if sessionID != ""` — Phase2 with empty string is a no-op

**RECOMMENDATION:** Option B. `runSpawnClaude` intentionally bypasses OpenCode. The escape hatch's value comes from independence. Writing a phantom session_id:
- Would require a running OpenCode server (violates escape hatch independence)
- Would create ghost sessions (the exact problem we spent 6 weeks fighting)
- Would make the query engine try to check liveness against a session that doesn't exist

However, `runSpawnClaude` SHOULD still call `AtomicSpawnPhase2` with `""` for contract completeness — it's a no-op today but documents the intent that Phase2 was considered.

---

### Fork 3: What liveness signal should claude-backend agents use?

**Options:**
- A: Tmux window existence (current approach — problematic)
- B: Phase comments as heartbeat (beads-based)
- C: Workspace file modification timestamps
- D: Process-level check (pidof claude in tmux pane)

**Substrate says:**
- **Decision** (two-lane): "Beads owns: Lifecycle for tracked work (exists/done)"
- **Decision** (two-lane): "Workspace manifest owns: Binding (beads_id ↔ session_id ↔ project_dir)"
- **Principle** (No Local Agent State): Query authoritative sources directly
- **Principle** (Observation Infrastructure): "Every state transition should emit an event"

**RECOMMENDATION:** Option B (phase comments as heartbeat).

**Why this works:**

Claude-backend agents report phase transitions via `bd comment`:
```
Phase: Planning - Analyzing codebase structure
Phase: Implementing - Adding authentication middleware
Phase: Complete - All tests passing, ready for review
```

These comments ARE the agent's heartbeat. They prove the agent is alive and working. The recency of the last phase comment is a direct proxy for liveness:

| Condition | Status | Reasoning |
|-----------|--------|-----------|
| Phase: Complete | completed | Agent declared done (handled by determineAgentStatus already) |
| Recent phase comment (< 30min) | active | Agent reported progress recently |
| No phase but spawned recently (< 5min) | spawning | Agent just started, hasn't reported yet |
| Stale phase (> 30min, no Complete) | stalled | Agent may be stuck or crashed |
| No phase, old spawn_time | dead | Agent never reported, likely failed to start |

**Why not Option A (tmux):**
- Violates two-lane decision
- `FindWindowByWorkspaceNameAllSessions` makes 3+ shell-outs per call (list-sessions, session-exists ×2, list-windows per session)
- Running under overmind, tmux commands target wrong socket intermittently
- Caching masks the unreliability without fixing it
- The cache at 10s TTL exceeds the 1-5s allowed range

**Why not Option C (file timestamps):**
- Workspace files may not change frequently (agent could be reading/thinking)
- Would require stat() calls on multiple files per agent
- Not authoritative for liveness — just artifact freshness

**Why not Option D (process check):**
- Even more expensive than tmux window check
- Requires knowing which pane the agent is in
- Platform-specific (different on macOS vs Linux)

---

## Synthesis

### Recommendation: Revert orch-go-1182/1183, Implement Phase-Based Liveness

**SUBSTRATE:**
- Decision: Two-lane says beads owns lifecycle, tmux owns presentation only
- Principle: No Local Agent State — query authoritative sources directly
- Principle: Coherence Over Patches — 3rd fix signals structural issue

**RECOMMENDATION:** Replace tmux liveness with phase-based liveness for claude-backend agents.

**Trade-off accepted:** Phase-based liveness has ~5min blind spot after spawn (before first Phase comment). This is acceptable because:
1. Agents report Phase: Planning within their first 3 tool calls (enforced by worker-base skill)
2. The spawn_time in manifest provides a grace period
3. "spawning" status is a reasonable display during this window

**When this would change:** If Claude Code gains an API that exposes session state (analogous to OpenCode's SSE events), that would be the proper liveness source for claude-backend agents.

### Implementation Plan

#### Step 1: Revert tmux liveness code

Remove from `query_tracked.go`:
- The `tmuxLivenessCache` struct and all its methods (lines 26-64)
- The `checkTmuxWindowLiveness` function variable (line 18)
- The tmux liveness branch in `joinWithReasonCodes` (lines 351-362)

#### Step 2: Add phase-based liveness for claude-backend agents

In `joinWithReasonCodes`, when `manifest.SessionID == ""` and `manifest.SpawnMode == "claude"`:

```go
if manifest.SpawnMode == "claude" && manifest.WorkspaceName != "" {
    phase, hasPhase := phases[issue.ID]
    spawnTime := manifest.SpawnTime // Already in AgentManifest

    if hasPhase && strings.HasPrefix(phase, "Complete") {
        agent.Status = "completed"
        agent.Phase = phase
        agent.Reason = "phase_complete"
    } else if hasPhase {
        agent.Status = "active"
        agent.Phase = phase
        agent.Reason = "phase_reported"
    } else if !spawnTime.IsZero() && time.Since(spawnTime) < 5*time.Minute {
        agent.Status = "active"
        agent.Reason = "recently_spawned"
    } else {
        agent.Status = "dead"
        agent.Reason = "no_phase_reported"
    }
    results = append(results, agent)
    continue
}
```

Note: The `phases` map is already available in `joinWithReasonCodes` — it's passed as a parameter. This adds ZERO new data fetching.

#### Step 3: Verify SpawnTime is available in AgentManifest

Check that `spawn.AgentManifest` has a `SpawnTime` field. If not, parse it from the `.spawn_time` dotfile (which `runSpawnClaude` already writes).

#### Step 4: Remove tmux import from query_tracked.go

The `pkg/tmux` import should no longer be needed in the query engine.

### File Targets

| File | Change |
|------|--------|
| `cmd/orch/query_tracked.go` | Remove tmux cache, replace tmux check with phase-based liveness |
| `cmd/orch/query_tracked.go` | Remove `pkg/tmux` import |
| `pkg/spawn/manifest.go` (or equivalent) | Verify SpawnTime field exists and is populated |

### Acceptance Criteria

1. Claude-backend agents show stable status (no oscillation)
2. No tmux shell-outs in the query path
3. No caches with TTL > 5s in the query path
4. `go build ./cmd/orch/` passes
5. `go vet ./cmd/orch/` passes
6. Existing tests pass

### Out of Scope

- Frontend dashboard changes
- Changing how `runSpawnClaude` spawns agents
- Adding Claude Code API integration for liveness

---

## Recommendations

**RECOMMENDED: Phase-based liveness (revert tmux approach)**
- **Why:** Uses beads (authoritative per two-lane decision) instead of tmux (UI-only per two-lane decision). Zero new data fetching — phases are already queried. Eliminates unreliable tmux shell-outs from the hot path.
- **Trade-off:** 5-minute blind spot after spawn before first phase comment. Acceptable given worker-base enforces early phase reporting.
- **Expected outcome:** Stable dashboard status for claude-backend agents, two-lane decision compliance restored.

**Alternative: Fix tmux reliability (make current approach work)**
- **Pros:** Provides true process-level liveness signal
- **Cons:** Fundamentally violates two-lane decision. Tmux commands are inherently unreliable under overmind. Would need even more caching to stabilize, deepening the violation.
- **When to choose:** Only if Claude Code gains no API and phase reporting proves insufficient for liveness detection.

**Alternative: Pre-create phantom OpenCode session**
- **Pros:** Unifies the query path (all agents have session_id)
- **Cons:** Violates escape hatch independence (requires running OpenCode server). Creates ghost sessions. Re-introduces the OpenCode dependency that the escape hatch was designed to avoid.
- **When to choose:** Never. This defeats the purpose of the dual spawn architecture.

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This resolves the dashboard oscillation pattern (3+ investigations)
- Future agents might re-introduce tmux-as-state

**Suggested blocks keywords:**
- "tmux liveness", "dashboard oscillation", "claude-backend status", "query engine liveness"

---

## Appendix: Oscillation Mechanism Trace

Full data flow causing oscillation:

```
Dashboard polls /api/agents (every 30s)
    ↓
handleAgents() → globalTrackedAgentsCache.get() (3s TTL)
    ↓ (cache miss)
queryTrackedAgents()
    ↓
joinWithReasonCodes()
    ↓ (for claude-backend agents with no session_id)
checkTmuxWindowLiveness(workspace) → tmuxLivenessCache (10s TTL)
    ↓ (cache miss)
tmux.FindWindowByWorkspaceNameAllSessions(workspace)
    ↓
ListWorkersSessions() → exec("tmux list-sessions")     [shell-out 1]
SessionExists("orchestrator") → exec("tmux has-session") [shell-out 2]
SessionExists("meta-orchestrator") → exec("tmux has-session") [shell-out 3]
For each session:
  FindWindowByWorkspaceName() → exec("tmux list-windows") [shell-out 4+]
    ↓
SUCCESS: window found → status="active" (cached 10s)
FAILURE: tmux command fails → status="dead" (cached 10s)
    ↓
agentStatusToAPIResponse() maps status
    ↓
determineAgentStatus() applies Priority Cascade (no override for active claude-backend agents)
    ↓
Dashboard shows "active" for 10s, then "dead" for 10s → OSCILLATION
```

The oscillation period matches the 10s cache TTL. When the cache expires and the tmux check succeeds, the agent shows "active" for 10s. When the cache expires and the tmux check fails (under overmind socket confusion), the agent shows "dead" for 10s.

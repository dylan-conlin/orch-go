# Probe: Tmux Liveness Check Violates Two-Lane Decision

**Model:** agent-lifecycle-state-model
**Date:** 2026-02-24
**Status:** Complete

---

## Question

The agent-lifecycle-state-model claims:
- **Invariant 6:** "Tmux windows are UI layer only — Not authoritative for state"
- **Invariant 7:** "No persistent lifecycle caches — Only in-memory, process-local caches with short TTLs allowed"

The two-lane agent discovery decision (2026-02-18) states:
- "Tmux owns: Presentation layer. Does NOT own: Any state whatsoever."
- In-memory cache TTL: 1-5 seconds allowed.

**Specific claims tested:**
1. Does the tmux liveness check added in orch-go-1182 violate Invariant 6?
2. Does the 10s TTL cache added in orch-go-1183 violate Invariant 7?
3. Is the oscillation caused by tmux check unreliability, and if so, is it structural?

---

## What I Tested

### Test 1: Code path analysis — tmux as state source

Traced `joinWithReasonCodes()` in `cmd/orch/query_tracked.go:349-362`:

```go
// Line 349-362 of joinWithReasonCodes
if manifest.SessionID == "" {
    if manifest.SpawnMode == "claude" && manifest.WorkspaceName != "" {
        if checkTmuxWindowLiveness(manifest.WorkspaceName) {
            agent.Status = "active"          // tmux → state
            agent.Reason = "tmux_window_alive"
        } else {
            agent.Status = "dead"            // tmux → state
            agent.Reason = "tmux_window_missing"
        }
        results = append(results, agent)
        continue
    }
}
```

The tmux window existence check DIRECTLY determines agent status ("active" or "dead"). This is tmux being used as a state source, not a presentation layer.

### Test 2: Cache TTL exceeds allowed range

`cmd/orch/query_tracked.go:26-33`:

```go
var tmuxLivenessCache = struct {
    ...
    ttl     time.Duration
}{
    entries: make(map[string]tmuxLivenessCacheEntry),
    ttl:     10 * time.Second,  // Decision allows 1-5s
}
```

Two-lane decision table (line 156-161):
```
| Cache Type | Allowed | Location | TTL |
| In-memory, process-local | Yes | Dashboard server | 1-5 seconds |
```

10s > 5s. The cache TTL exceeds the allowed range.

### Test 3: Tmux check reliability analysis

`FindWindowByWorkspaceNameAllSessions` (`pkg/tmux/tmux.go:864-890`) executes:

1. `ListWorkersSessions()` → `tmux list-sessions` (shell-out 1)
2. `SessionExists("orchestrator")` → `tmux has-session` (shell-out 2)
3. `SessionExists("meta-orchestrator")` → `tmux has-session` (shell-out 3)
4. For each session: `FindWindowByWorkspaceName()` → `tmux list-windows` (shell-out 4+)

When running under overmind (which the dashboard server does via `orch-dashboard start`), `detectMainSocket()` must distinguish overmind's tmux socket from the main tmux socket. If socket detection fails or tmux commands race, the window check returns nil → agent marked "dead".

The 10s cache means:
- Successful check → "active" cached for 10s
- Failed check → "dead" cached for 10s
- Result: 10s oscillation period between "active" and "dead"

### Test 4: Phase data already available in query engine

`joinWithReasonCodes` already receives a `phases` map parameter:

```go
func joinWithReasonCodes(
    issues []beads.Issue,
    manifests map[string]*spawn.AgentManifest,
    liveness map[string]opencode.SessionStatusInfo,
    phases map[string]string,    // ← Already available
) []AgentStatus {
```

Phases are extracted from beads comments in `extractLatestPhases()` (line 242-275). This data is authoritative (comes from beads, the source of truth per two-lane decision) and is already fetched for every agent. Using phases for liveness requires ZERO additional data fetching.

---

## What I Observed

### Finding 1: Clear Invariant 6 violation

The tmux liveness check uses tmux window existence to determine agent status. The two-lane decision and model Invariant 6 both explicitly state tmux should NOT own state. The code at `query_tracked.go:349-362` directly maps `tmux window exists → active` and `tmux window missing → dead`. This is using tmux as a state oracle.

### Finding 2: Clear Invariant 7 violation

The 10s TTL cache exceeds the 1-5s allowed range. The comment on the cache explains why it was needed: "Without caching, each tmux check shells out multiple times... and intermittent failures cause inconsistent results across endpoints, making the dashboard oscillate." The need for a cache that exceeds the allowed TTL is itself the signal that the approach is wrong.

### Finding 3: Oscillation is structural, not incidental

The tmux command reliability is fundamentally limited by:
1. **Overmind socket confusion** — the dashboard server runs in overmind's tmux, requiring socket override
2. **Multiple shell-outs per check** — 3+ tmux commands per agent, each can fail independently
3. **Single-threaded tmux server** — concurrent tmux commands can race

These are not bugs that can be fixed. They're structural properties of running tmux commands from inside an overmind-managed process. The cache paper-overs the symptom but doesn't fix the cause.

### Finding 4: Phase comments are an existing, authoritative, zero-cost alternative

The `phases` map is already computed and passed to `joinWithReasonCodes`. It comes from beads comments (authoritative per two-lane decision). Using it requires no new fetching, no new caches, and no tmux shell-outs. The worker-base skill enforces early phase reporting ("within first 3 tool calls"), providing a reliable heartbeat signal.

---

## Model Impact

- [x] **Confirms** Invariant 6: "Tmux windows are UI layer only" — the tmux liveness check violates this
- [x] **Confirms** Invariant 7: "No persistent lifecycle caches" with "short TTLs allowed" — the 10s cache violates the allowed range
- [x] **Extends** the model with: **For agents without OpenCode sessions, phase comments serve as the liveness signal.** This fills the gap that the two-lane decision didn't anticipate (claude-backend agents with no session_id)

### Proposed Invariant Addition

**Invariant 9 (new):** "Claude-backend agents use phase comments as liveness proxy — NOT tmux window checks"
- Phases come from beads (authoritative per two-lane decision)
- Zero additional data fetching (phases already queried)
- Agent reports Phase within first 3 tool calls (worker-base enforcement)
- 5-minute grace period for spawning agents before first Phase comment

### Extends Prior Probe

This probe extends `2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md` which identified that `runSpawnClaude` skips `AtomicSpawnPhase2` leaving no `session_id`. That probe's recommended fix (Option A: tmux liveness check) is what was implemented in orch-go-1182 and is now shown to be architecturally wrong.

---

## Notes

- The tmux liveness code (orch-go-1182) and cache (orch-go-1183) should be reverted
- The prior probe's Option A recommendation should be superseded by the phase-based approach
- The `determineAgentStatus` Priority Cascade already handles Phase: Complete correctly — the phase-based liveness only needs to handle the "active" and "dead" determination for non-complete agents

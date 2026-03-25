## Summary (D.E.K.N.)

**Delta:** IsProcessing IS currently set for Claude agents (from status="active" → IsProcessing=true mapping), but it's derived from a STATIC signal (historical phase comments) not a LIVE signal. Dead agents with stale phase comments are permanently masked as "processing."

**Evidence:** Code trace: discovery.go:404 maps phase_reported → "active"; agentStatusToAgentInfo:462 maps "active" → IsProcessing=true; status_cmd.go:385 skips UNRESPONSIVE if IsProcessing=true. For OpenCode agents, line 234 OVERRIDES with real-time session status. No equivalent override exists for Claude agents.

**Knowledge:** The fix is not "add IsProcessing for Claude" — it's "replace the static IsProcessing signal with a live one." IsPaneActive() in pkg/tmux/pane.go:68 already provides the live signal. Wire it as an override for Claude agents, mirroring how OpenCode session status overrides for OpenCode agents.

**Next:** Implementation: 3 changes (discovery + 2 consumers). Decomposition below with component issues.

**Authority:** architectural - Cross-component (discovery, status_cmd, serve_agents_handlers), changes signal composition semantics

---

# Investigation: Composed Liveness Detection for Claude Code Agents

**Question:** How should we compose PID liveness, tmux pane activity, and phase timeout into a single IsProcessing determination for Claude Code agents?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Owner:** orch-go-k8sle
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| orch-go-jhluq (liveness signals investigation) | extends | yes — code paths verified at exact line numbers | Partially: investigation claimed "IsProcessing is always false for Claude" but it IS set via status mapping (see Finding 1) |
| dc82e9606 (scs-sp-vzm UNRESPONSIVE fix) | extends | yes — guard at status_cmd.go:385 works for OpenCode | no |
| 122c9b9f3 (phase-based liveness decision) | extends | yes | no — phase-based liveness was right, but using it as IsProcessing signal creates Class 5 defect |

---

## Findings

### Finding 1: IsProcessing IS set for Claude agents — but from a static signal

**Evidence:** The data flow for a Claude agent that reported Phase: Planning 45 minutes ago:

1. `ExtractLatestPhases` → phases["orch-go-xxx"] = "Planning - reading code"
2. `JoinWithReasonCodes` line 404: Phase != "" → Status = "active", Reason = "phase_reported"
3. `agentStatusToAgentInfo` line 462: Status == "active" → IsProcessing = true
4. UNRESPONSIVE check line 385: IsProcessing == true → `continue` (skip)

The agent is NEVER flagged as unresponsive because any historical phase comment permanently sets IsProcessing=true.

For OpenCode agents, the session status API OVERRIDES this at line 234:
```go
agent.IsProcessing = statusInfo.IsBusy() || statusInfo.IsRetrying()
```
If the session is idle, IsProcessing becomes false despite status="active" from discovery.

For Claude agents: no override exists. IsProcessing stays as set by the status mapping.

**Source:** `pkg/discovery/discovery.go:404`, `cmd/orch/status_cmd.go:462-464`, `cmd/orch/status_cmd.go:221-242`, `cmd/orch/status_cmd.go:380-394`

**Significance:** The investigation orch-go-jhluq stated "IsProcessing is always false for Claude agents." This is incorrect — IsProcessing is set to true for any Claude agent with a phase comment, because of the status→IsProcessing mapping in the conversion functions. The real problem is that this signal is STATIC (based on historical phase comments) rather than LIVE (based on current process activity). This creates the opposite problem: dead Claude agents are masked as processing.

---

### Finding 2: IsPaneActive() is the natural IsProcessing equivalent for Claude agents

**Evidence:** `pkg/tmux/pane.go:68-88` — `IsPaneActive(windowID)` checks:
1. `pane_current_command` — non-shell process in foreground → active
2. `hasChildProcesses(pid)` — shell has children (handles macOS tmux quirk) → active

For orch-spawned Claude agents:
- Agent actively working → claude process running as child → IsPaneActive=true
- Agent finished/crashed → no child process → IsPaneActive=false
- Agent at prompt (rare for autonomous agents) → claude still running → IsPaneActive=true

Cost: ~10ms per check (two tmux commands). Already tested, already used by `orch clean` (lifecycle_impl.go:435).

**Source:** `pkg/tmux/pane.go:68-88`, `pkg/agent/lifecycle_impl.go:435`

**Significance:** IsPaneActive provides a LIVE signal equivalent to OpenCode's IsBusy(). It answers "is a non-shell process running in this pane right now?" — which for orch-spawned Claude agents means "is the claude process still doing work?"

---

### Finding 3: Discovery already looks up tmux windows but discards the window ID

**Evidence:** `CheckTmuxWindowAlive` (discovery.go:67-77) calls `tmux.FindWindowByWorkspaceName` which returns `*WindowInfo` (containing ID, Index, Name, Target). But the result is collapsed to a bool:

```go
var CheckTmuxWindowAlive = func(workspaceName, projectDir string) bool {
    // ...
    window, _ := tmux.FindWindowByWorkspaceName(sessionName, workspaceName)
    return window != nil
}
```

The WindowInfo.ID (e.g., "@1234") is exactly what IsPaneActive needs, but it's thrown away.

**Source:** `pkg/discovery/discovery.go:67-77`, `pkg/tmux/window_search.go:55-60` (WindowInfo struct)

**Significance:** We're already paying the tmux query cost. Keeping the WindowInfo gives us the input for IsPaneActive with zero additional tmux calls.

---

### Finding 4: Claude backend routing has a signal priority bug (Class 5: Contradictory Authority Signals)

**Evidence:** In discovery.go:401-425, the Claude backend path checks signals in this order:
1. Phase != "" → "active" (takes priority)
2. Recently spawned → "active"
3. Never started → "dead"
4. Tmux window alive → "active" (fallback)
5. No signals → "dead"

Because phase_reported (step 1) takes priority over tmux window checks (step 4), a dead agent with a historical phase comment is classified as "active." The tmux window check that would catch the dead state is never reached.

This is a Class 5 (Contradictory Authority Signals) defect: the phase comment (historical) and tmux window state (live) disagree, and the historical signal wins.

**Source:** `pkg/discovery/discovery.go:401-425`, `.kb/models/defect-class-taxonomy/model.md` (Class 5)

**Significance:** Even without the IsProcessing fix, discovery classifies dead Claude agents as "active" because it trusts phase comments over live tmux state. The fix must reorder signal priority: check window alive first, then use phase for categorization within live agents.

---

### Finding 5: StallTracker provides the pattern precedent for in-memory activity tracking

**Evidence:** `pkg/daemon/stall_tracker.go` — mutex-protected map of sessionID → TokenSnapshot. Used by both status_cmd.go and serve_agents_handlers.go via `globalStallTracker`. The stall tracker:
- Stores snapshots between polls (process-local, not persisted)
- Uses Update() for polling (returns stalled bool)
- Uses CleanStale() for bounded memory
- Is shared between serve and status via package-level var

**Source:** `pkg/daemon/stall_tracker.go`, `cmd/orch/serve_agents_status.go:222-225`

**Significance:** If pane content delta is needed in the future (for agents that are alive but idle at prompt), the StallTracker pattern works directly. For now, IsPaneActive is sufficient, but this validates the architecture for future refinement.

---

## Synthesis

**Key Insights:**

1. **The asymmetry is override, not absence.** Both OpenCode and Claude agents get IsProcessing=true from the status→IsProcessing mapping. The difference is that OpenCode agents have a live OVERRIDE (session API), while Claude agents don't. The fix is adding a live override for Claude, not adding IsProcessing from scratch.

2. **IsPaneActive is sufficient; pane content delta is unnecessary complexity.** The investigation orch-go-jhluq proposed a three-layer signal (PID check, pane content delta, phase timeout). But IsPaneActive already answers the question "is the agent doing work?" for orch-spawned Claude agents. Pane content delta solves a narrower problem (agent alive but idle at prompt) that rarely occurs for autonomous agents.

3. **Discovery's signal priority must change.** The current code trusts phase comments over tmux window state. A dead agent with a phase comment is classified as "active." The fix must check tmux window alive before using phase for status determination.

4. **The design mirrors the OpenCode pattern.** OpenCode flow: discovery sets status → conversion sets IsProcessing from status → session API overrides IsProcessing. Claude flow should be: discovery sets status → conversion sets IsProcessing from status → IsPaneActive overrides IsProcessing. Same pattern, different live signal source.

**Answer to Investigation Question:**

Compose the signals as follows:

| Layer | Signal | Purpose | Priority |
|-------|--------|---------|----------|
| 1 | Tmux window exists | Agent infrastructure alive | Gate (if no window, check timing) |
| 2 | IsPaneActive(windowID) | Agent process actively running | IsProcessing equivalent |
| 3 | Phase comments | Agent self-reported progress | Status categorization |
| 4 | Phase timeout (30 min) | Backstop for stuck agents | Only fires when IsProcessing=false |

The PID check from `~/.claude/sessions/{pid}.json` is NOT needed — IsPaneActive already detects whether the claude process is running as a child of the pane shell. Session file PID checking is useful for enrichment (e.g., correlating with session metadata) but not for the IsProcessing signal.

---

## Structured Uncertainty

**What's tested:**

- IsPaneActive() correctly detects active claude processes (verified: used by orch clean, pkg/agent/lifecycle_impl.go:435)
- Status="active" → IsProcessing=true mapping exists in both consumers (verified: status_cmd.go:462, serve_agents_handlers.go:552)
- OpenCode session status overrides IsProcessing for OpenCode agents (verified: status_cmd.go:234, serve_agents_handlers.go:115)
- FindWindowByWorkspaceName returns WindowInfo with ID field (verified: pkg/tmux/window_search.go:55-60)

**What's untested:**

- IsPaneActive performance at scale (10+ Claude agents checked simultaneously)
- Edge case: Claude agent at idle prompt (finished work, didn't exit) — IsPaneActive returns true, masking idle state
- Whether changing discovery signal priority (tmux before phase) breaks any downstream assumptions

**What would change this:**

- If Claude Code adds a native processing API (like `claude --status`), IsPaneActive becomes unnecessary
- If agents are spawned without tmux (headless Claude Code), IsPaneActive is unavailable
- If the idle-at-prompt scenario becomes common, pane content delta (from StallTracker pattern) would be needed as a refinement

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add IsPaneActive as live IsProcessing override for Claude agents | architectural | Cross-component: discovery, status_cmd, serve_agents_handlers |
| Refactor Claude backend signal priority in discovery | architectural | Changes liveness semantics for all Claude agents |
| Add TmuxWindowID to AgentStatus | implementation | Internal field, no external behavior change |

### Recommended Approach: Live IsPaneActive Override

**Wire `tmux.IsPaneActive()` as a live IsProcessing override for Claude Code agents, mirroring how OpenCode session status overrides for OpenCode agents.**

**Why this approach:**
- Uses an existing, tested function (IsPaneActive) — no new capability needed
- Mirrors the OpenCode pattern exactly (live signal overrides static signal)
- Fixes both problems: false-negative UNRESPONSIVE (active agents flagged) AND false-positive IsProcessing (dead agents masked)
- Minimal code change: add one override loop in each consumer, one signal priority reorder in discovery

**Trade-offs accepted:**
- IsPaneActive returns true for agents at idle prompt (rare for autonomous agents; future refinement via pane content delta if needed)
- Adds ~10ms per Claude agent per status check (acceptable for < 20 agents)
- Does not work for headless Claude agents (none exist today)

**Implementation sequence:**

**Component 1: Discovery — Signal priority refactor + WindowID enrichment**
File: `pkg/discovery/discovery.go`

1. Add `TmuxWindowID string` and `IsProcessing bool` fields to `AgentStatus`
2. Replace `CheckTmuxWindowAlive` bool with new `FindTmuxWindowForAgent` that returns `*tmux.WindowInfo`
3. Refactor Claude backend routing: check window alive first, then use phase for categorization
4. When window is found and IsPaneActive: set `IsProcessing = true`

```go
// New mockable function
var FindTmuxWindowForAgent = func(workspaceName, projectDir string) *tmux.WindowInfo {
    if workspaceName == "" || projectDir == "" {
        return nil
    }
    projectName := filepath.Base(projectDir)
    sessionName := tmux.GetWorkersSessionName(projectName)
    window, _ := tmux.FindWindowByWorkspaceName(sessionName, workspaceName)
    return window
}

// Refactored Claude backend path
if manifest.SpawnMode == "claude" && manifest.WorkspaceName != "" {
    spawnTime := manifest.ParseSpawnTime()
    window := FindTmuxWindowForAgent(manifest.WorkspaceName, manifest.ProjectDir)

    if window != nil {
        agent.TmuxWindowID = window.ID
        agent.IsProcessing = tmux.IsPaneActive(window.ID)
    }

    if window == nil && !spawnTime.IsZero() && time.Since(spawnTime) < 5*time.Minute {
        agent.Status = "active"
        agent.Reason = "recently_spawned"
    } else if window == nil {
        if !spawnTime.IsZero() && time.Since(spawnTime) >= neverStartedThreshold {
            agent.Status = "dead"
            agent.Reason = "never_started"
            agent.NeverStarted = true
        } else {
            agent.Status = "dead"
            agent.Reason = "tmux_window_dead"
        }
    } else if agent.Phase != "" {
        agent.Status = "active"
        agent.Reason = "phase_reported"
    } else if !spawnTime.IsZero() && time.Since(spawnTime) < 5*time.Minute {
        agent.Status = "active"
        agent.Reason = "recently_spawned"
    } else if !spawnTime.IsZero() && time.Since(spawnTime) >= neverStartedThreshold {
        agent.Status = "dead"
        agent.Reason = "never_started"
        agent.NeverStarted = true
    } else {
        agent.Status = "active"
        agent.Reason = "tmux_window_alive"
    }
}
```

**Component 2: status_cmd.go — Claude IsProcessing override**
File: `cmd/orch/status_cmd.go`

After the OpenCode enrichment loop (line 242), add a Claude override:

```go
// Override IsProcessing for Claude agents with live tmux pane check.
// agentStatusToAgentInfo sets IsProcessing from discovery Status (static).
// For Claude agents, override with IsPaneActive (live), mirroring how
// OpenCode session status overrides for OpenCode agents above.
for i := range agents {
    agent := &agents[i]
    if agent.Mode != "claude" {
        continue
    }
    // Use IsProcessing from discovery (set via IsPaneActive in JoinWithReasonCodes)
    if i < len(trackedAgents) {
        agent.IsProcessing = trackedAgents[i].IsProcessing
    }
}
```

**Component 3: serve_agents_handlers.go — Claude IsProcessing override**
File: `cmd/orch/serve_agents_handlers.go`

Same pattern in the agent conversion loop (after line 118):

```go
// Override IsProcessing for Claude agents with discovery's live signal.
if tracked.SpawnMode == "claude" {
    agent.IsProcessing = tracked.IsProcessing
}
```

### Alternative Approaches Considered

**Option B: Pane content delta tracker**
- **Pros:** More granular — distinguishes "actively generating" from "alive but idle at prompt"
- **Cons:** Requires two-sample comparison (needs persistent process or delay), adds StallTracker-like state, more complex to implement and test
- **When to use instead:** If IsPaneActive false positives (alive-but-idle agents) become a measurable problem. Can be added as a refinement layer later.

**Option C: PID check from session files**
- **Pros:** Direct process liveness check, no tmux dependency
- **Cons:** Requires knowing which config dir to scan (default vs per-account), redundant with IsPaneActive (which already detects process death via child process check), doesn't tell us if agent is processing vs idle
- **When to use instead:** If headless Claude Code agents are introduced (no tmux window). Or for enrichment (session metadata correlation).

**Option D: Set IsProcessing on AgentStatus in discovery for ALL backends**
- **Pros:** Unified signal source, consumers don't need backend-specific enrichment
- **Cons:** OpenCode IsProcessing requires real-time session API call. Discovery already queries session status for OpenCode but stores it in liveness map, not on AgentStatus. Changing this reorganizes the entire discovery→consumer flow.
- **When to use instead:** In a larger refactoring of the discovery/consumer boundary.

**Rationale for recommendation:** Option A (IsPaneActive override) is the minimum viable change that fixes both defects (false IsProcessing for dead agents, missing live signal for active agents). It uses an existing, tested function, mirrors the OpenCode pattern exactly, and leaves room for future refinement (Option B) without precluding it.

---

### Implementation Details

**What to implement first:**
1. Discovery changes (Component 1) — foundation for consumer changes
2. status_cmd.go override (Component 2) — fixes CLI `orch status`
3. serve_agents_handlers.go override (Component 3) — fixes dashboard

**Things to watch out for:**
- `CheckTmuxWindowAlive` is a mockable package-level var used in tests. `FindTmuxWindowForAgent` must also be mockable.
- The discovery signal priority reorder changes behavior for agents with phase comments but dead windows. Test that dead-but-phased agents are now correctly classified as "dead."
- `agentStatusToAgentInfo` maps status="active" → IsProcessing=true. After the Claude override, IsProcessing may flip to false for agents at idle prompts. Verify the UNRESPONSIVE timer behavior is correct in this case.
- Both consumers index agents by position (agents[i] corresponds to trackedAgents[i]). Verify this assumption holds.

**Areas needing further investigation:**
- How common is the "agent at idle prompt" scenario for autonomous Claude agents?
- Should we add a new status "pane_idle" for agents with window alive but IsPaneActive=false and no recent phase?

**Defect class exposure:**
- Class 2 (Multi-Backend Blindness): This design directly addresses the asymmetry. Test both OpenCode and Claude code paths.
- Class 5 (Contradictory Authority Signals): Signal priority reorder resolves phase vs tmux contradiction.
- Class 7 (Premature Destruction): IsPaneActive prevents premature UNRESPONSIVE flagging.

**Success criteria:**
- Zero false-positive UNRESPONSIVE flags for actively-generating Claude Code agents
- Dead Claude agents (crashed, exited) correctly flagged as dead within one status check
- No regression in OpenCode agent detection
- All existing discovery tests pass (may need new test for Claude override)

---

## References

**Files Examined:**
- `pkg/discovery/discovery.go:29-65` — AgentStatus struct
- `pkg/discovery/discovery.go:67-77` — CheckTmuxWindowAlive (discards window ID)
- `pkg/discovery/discovery.go:401-425` — Claude backend routing (signal priority bug)
- `cmd/orch/status_cmd.go:89-113` — AgentInfo struct (has Mode, IsProcessing fields)
- `cmd/orch/status_cmd.go:221-242` — OpenCode enrichment (overrides IsProcessing)
- `cmd/orch/status_cmd.go:380-394` — UNRESPONSIVE detection (checks IsProcessing)
- `cmd/orch/status_cmd.go:449-475` — agentStatusToAgentInfo conversion
- `cmd/orch/serve_agents_handlers.go:98-128` — Dashboard agent conversion + OpenCode enrichment
- `cmd/orch/serve_agents_handlers.go:180-191` — Dashboard UNRESPONSIVE detection
- `cmd/orch/serve_agents_handlers.go:526-556` — agentStatusToAPIResponse conversion
- `cmd/orch/serve_agents_types.go:9-43` — AgentAPIResponse struct
- `pkg/tmux/pane.go:68-88` — IsPaneActive function
- `pkg/tmux/window_search.go:55-60` — WindowInfo struct (has ID)
- `pkg/tmux/window_search.go:123-137` — FindWindowByWorkspaceName
- `pkg/daemon/stall_tracker.go` — StallTracker pattern precedent
- `pkg/spawn/session.go:170-200` — AgentManifest (has SpawnMode, no WindowID)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-25-inv-investigate-orch-status-detect-liveness.md` (orch-go-jhluq) — Found the 3 liveness signals
- **Brief:** `.kb/briefs/orch-go-jhluq.md` — Summarized PID + pane delta + phase timeout
- **Fix:** dc82e9606 (scs-sp-vzm) — Added IsProcessing guard for OpenCode agents
- **Decision:** 122c9b9f3 — Phase-based liveness over tmux-as-state
- **Model:** `.kb/models/defect-class-taxonomy/model.md` — Class 2, 5, 7 exposure

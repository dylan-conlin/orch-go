# Probe: Claude-Backend Spawns Missing from Dashboard

**Model:** agent-lifecycle-state-model
**Date:** 2026-02-24
**Status:** Complete

---

## Question

The agent-lifecycle-state-model claims a "two-lane architecture" where agents are discovered via:
1. Tracked work (beads-first via `queryTrackedAgents`)
2. Untracked sessions (OpenCode session list)

And that liveness is determined by checking OpenCode session status.

**Specific claim tested:** Does the query engine correctly discover and report liveness for agents spawned via `--backend claude` (which runs Claude CLI in tmux, bypassing OpenCode entirely)?

---

## What I Tested

### Test 1: Workspace file comparison across spawn modes

```bash
# Check current agent's workspace (tmux/claude spawn)
ls -la .orch/workspace/og-inv-investigate-work-graph-24feb-fc8a/
cat .orch/workspace/og-inv-investigate-work-graph-24feb-fc8a/AGENT_MANIFEST.json
cat .orch/workspace/og-inv-investigate-work-graph-24feb-fc8a/.session_id  # Does it exist?
```

### Test 2: Cross-mode session_id comparison

```bash
# Compared session_id presence across all 105 claude-mode + 27 opencode-mode workspaces
for ws in .orch/workspace/og-*/; do
  mode=$(cat "$ws/.spawn_mode" 2>/dev/null)
  sid=$(cat "$ws/.session_id" 2>/dev/null)
  manifest_sid=$(python3 -c "import json; m=json.load(open('${ws}AGENT_MANIFEST.json')); print(m.get('session_id','MISSING'))")
  echo "$(basename $ws) | mode=$mode | .session_id=${sid:-NONE} | manifest.session_id=$manifest_sid"
done
```

### Test 3: Code path tracing

Traced the three spawn dispatch paths in `pkg/orch/extraction.go:DispatchSpawn()`:
- `runSpawnHeadless` (line 1224)
- `runSpawnTmux` (line 1358)
- `runSpawnClaude` (line 1526)

And the query engine in `cmd/orch/query_tracked.go:queryTrackedAgents()`.

---

## What I Observed

### Finding 1: Claude-backend manifest has NO session_id

Current agent's manifest (`og-inv-investigate-work-graph-24feb-fc8a`):
```json
{
  "workspace_name": "og-inv-investigate-work-graph-24feb-fc8a",
  "skill": "investigation",
  "beads_id": "orch-go-1177",
  "project_dir": "~/Documents/personal/orch-go",
  "git_baseline": "a6a24f4e61c74ecfb2c3284a6eaf202d6b99aaa3",
  "spawn_time": "2026-02-24T10:42:23-08:00",
  "tier": "full",
  "spawn_mode": "claude",
  "model": "anthropic/claude-opus-4-5-20251101"
}
```

No `.session_id` dotfile exists. No `session_id` field in manifest.

### Finding 2: 32 of 105 claude-mode agents lack session_id

```
Claude-mode workspaces: 105 total
  Without session_id: 32 (30%) — all from Feb 21+ (after runSpawnClaude was added)
  With session_id:    73 (70%) — all from Feb 20 and earlier (went through runSpawnTmux)

OpenCode-mode workspaces: 27 — all have session_id
```

The 73 claude-mode agents WITH session_ids are from before `runSpawnClaude` existed, when they went through `runSpawnTmux` (which pre-creates an OpenCode session even for tmux windows).

### Finding 3: Three spawn paths, only one lacks session_id write

| Spawn Path | Creates OpenCode Session? | Calls AtomicSpawnPhase2? | Writes session_id? |
|---|---|---|---|
| `runSpawnHeadless` | Yes (HTTP API) | Yes | Yes |
| `runSpawnTmux` | Yes (pre-creates via API) | Yes | Yes |
| `runSpawnClaude` | **No** (Claude CLI bypasses OpenCode) | **No** | **No** |

**Root cause in code:**

`runSpawnClaude` (line 1526):
- Calls `spawn.SpawnClaude(cfg)` → creates tmux window with `claude` CLI
- Logs event with `spawn_mode: "claude"` but NO `session_id`
- **Never calls `spawn.AtomicSpawnPhase2()`** → manifest session_id stays empty

vs `runSpawnHeadless` (line 1224):
- Creates OpenCode session → gets `sessionID`
- Calls `spawn.AtomicSpawnPhase2(opts, sessionID)` → writes session_id to manifest AND `.session_id` dotfile

### Finding 4: queryTrackedAgents marks missing session as dead

`query_tracked.go:joinWithReasonCodes()` (line 296-302):
```go
if manifest.SessionID == "" {
    agent.MissingSession = true
    agent.Status = "unknown"
    agent.Reason = "missing_session"
    results = append(results, agent)
    continue
}
```

Then in `serve_agents_handlers.go:agentStatusToAPIResponse()` (line 498-500):
```go
case "unknown":
    if tracked.MissingBinding {
        resp.Status = "dead"
    } else if tracked.MissingSession {
        resp.Status = "dead"
    } else {
        resp.Status = "dead"
    }
```

**Result:** Claude-backend agents → manifest has no session_id → `missing_session=true` → `status=unknown` → `resp.Status="dead"` in the dashboard.

### Finding 5: No tmux liveness check exists

The query engine's liveness check (Step 4) ONLY checks OpenCode sessions:
```go
liveness, err = client.GetSessionStatusByIDs(sessionIDs)
```

There is no fallback to check tmux window existence when `spawn_mode == "claude"`. The `SpawnMode` field exists in the manifest but is never used by the query engine for routing liveness checks.

---

## Model Impact

- [x] **Extends** model with: Claude-backend spawn path creates a structural gap in the two-lane discovery architecture

### Data Flow Diagram (Current)

```
                  Dashboard (/api/agents)
                         |
                  queryTrackedAgents()
                         |
            +------------+------------+
            |            |            |
     Step 1: Beads   Step 2: WS    Step 4: OpenCode
     (orch:agent)    Manifests     Session Liveness
            |            |            |
            v            v            v
      All spawns     All spawns    ONLY agents with
      write this     write this    OpenCode sessions
                                        |
                                   +---------+
                                   |         |
                              headless    tmux+opencode
                              spawns      spawns
                                   |         |
                                   +---------+
                                        |
                                   HAS liveness

                              claude spawns → NO liveness
                              (missing_session → dead)
```

### Gap Analysis

| Layer | Headless (opencode) | Tmux (opencode) | Claude (tmux) |
|---|---|---|---|
| Beads issue + orch:agent tag | Yes | Yes | Yes |
| AGENT_MANIFEST.json | Yes | Yes | Yes |
| SPAWN_CONTEXT.md | Yes | Yes | Yes |
| .session_id dotfile | Yes | Yes | **No** |
| manifest.session_id | Yes | Yes | **No** |
| OpenCode session exists | Yes | Yes | **No** |
| Dashboard can check liveness | Yes | Yes | **No** |
| Dashboard status | Correct | Correct | **"dead" (wrong)** |

### Fix Surface Options

**Option A: Add tmux liveness check to query engine (recommended)**
- In `joinWithReasonCodes()`, when `manifest.SpawnMode == "claude"` and `manifest.SessionID == ""`:
  - Check if tmux window exists matching the workspace name
  - If window exists → status = "active"
  - If window doesn't exist → status = "dead" (agent actually finished/crashed)
- Pro: Architecturally honest — acknowledges the dual execution model
- Con: Adds tmux dependency to the query engine

**Option B: Pre-create OpenCode session in runSpawnClaude for tracking only**
- Create an OpenCode session just for visibility, even though Claude CLI won't use it
- Write session_id to manifest like the other paths
- Pro: No query engine changes needed
- Con: Creates ghost sessions in OpenCode, conflates tracking with execution

**Option C: Store tmux window ID in manifest**
- `runSpawnClaude` writes `tmux_window_id` to manifest (it already has this from `tmux.CreateWindow`)
- Query engine uses window ID for liveness check when `spawn_mode == "claude"`
- Pro: Clean separation of concern
- Con: New manifest field, new liveness check path

### Extends Invariant 5 and 6

- **Invariant 5** ("Multiple sources must be reconciled"): Currently TRUE but the reconciliation only covers OpenCode sessions. Claude-backend agents are a third source (tmux) that isn't reconciled.
- **Invariant 6** ("Tmux windows are UI layer only"): This invariant may need revision. For claude-backend spawns, the tmux window IS the execution layer, not just UI. It's the only liveness signal available.

---

## Notes

- The split happened around Feb 20-21 when `runSpawnClaude` was introduced
- Impact grows as more agents use `--backend claude` (which is increasingly the default for Opus work)
- The `SpawnMode` field already exists in AgentManifest but is unused by the query engine — this is the natural discriminator for routing liveness checks
- This is NOT a new failure mode in the model — it's a gap that appeared when the claude spawn path was added without updating the query engine to handle the new execution model

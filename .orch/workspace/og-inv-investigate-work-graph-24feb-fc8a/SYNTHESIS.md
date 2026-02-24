# Session Synthesis

**Agent:** og-inv-investigate-work-graph-24feb-fc8a
**Issue:** orch-go-1177
**Outcome:** success

---

## Plain-Language Summary

The work-graph dashboard shows tmux-spawned (claude-backend) agents as "dead" or "unassigned" because the spawn path for Claude CLI (`runSpawnClaude`) never writes a session ID to the workspace manifest. The dashboard's query engine (`queryTrackedAgents`) checks OpenCode session liveness to determine if agents are alive, but Claude CLI agents bypass OpenCode entirely - they run as direct CLI processes in tmux windows. This means the only liveness signal available (tmux window existence) is never checked. 32 out of 105 claude-mode agents are currently affected (all spawned after Feb 20 when `runSpawnClaude` was introduced). The fix requires either adding a tmux liveness check to the query engine when `spawn_mode=claude`, or storing the tmux window ID in the manifest as an alternative liveness reference.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- Traced complete data flow: spawn → workspace → queryTrackedAgents → dashboard API
- Identified structural gap: `runSpawnClaude` never calls `AtomicSpawnPhase2`
- Quantified impact: 32/105 claude-mode agents affected (growing as claude becomes default)
- Documented three fix options with trade-offs

---

## Delta (What Changed)

### Files Created
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md` - Probe documenting the visibility gap with data flow diagram and fix options

### Files Modified
- None (investigation only)

---

## Evidence (What Was Observed)

- `AGENT_MANIFEST.json` for claude-backend spawns has no `session_id` field (verified on live workspace `og-inv-investigate-work-graph-24feb-fc8a`)
- `.session_id` dotfile does not exist for claude-backend spawns (exit code 1 on read)
- `runSpawnClaude` at `pkg/orch/extraction.go:1526` never calls `spawn.AtomicSpawnPhase2()` - the function that writes session IDs
- `runSpawnHeadless` at `pkg/orch/extraction.go:1224` and `runSpawnTmux` at `pkg/orch/extraction.go:1358` both call `AtomicSpawnPhase2`
- `query_tracked.go:joinWithReasonCodes()` at line 296-302: `SessionID == ""` → `MissingSession=true, Status="unknown"`
- `serve_agents_handlers.go:agentStatusToAPIResponse()` at line 498-500: `MissingSession=true` → `Status="dead"`
- 32 of 105 claude-mode workspaces lack session_id (all from Feb 21+); 73 from Feb 20 and earlier have session_ids (were spawned via `runSpawnTmux` before `runSpawnClaude` existed)
- All 27 opencode-mode workspaces have session_ids

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The query engine assumes all tracked agents have OpenCode sessions for liveness checking. Claude-backend agents violate this assumption.
- `SpawnMode` field exists in AgentManifest but is never used by the query engine for routing liveness checks - it's the natural discriminator for this fix.

### Externalized via `kb`
- `kb quick constrain` - queryTrackedAgents only checks OpenCode sessions for liveness (kb-95c8d6)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Add tmux liveness check for claude-backend agents in queryTrackedAgents
**Skill:** feature-impl
**Context:**
```
queryTrackedAgents marks claude-backend agents as dead because they lack OpenCode sessions.
Fix: In joinWithReasonCodes(), when manifest.SpawnMode == "claude" and SessionID == "",
check tmux window existence (using workspace name matching) instead of OpenCode session liveness.
See probe: .kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md
```

---

## Unexplored Questions

- Should `runSpawnClaude` also store the tmux window_id in the manifest for liveness checking? (Currently available from `tmux.CreateWindow` return value but discarded after logging)
- Does the `orch status` command have the same gap, or does it use a different discovery mechanism?
- When a claude-backend agent finishes and its tmux window closes, how should the dashboard distinguish "completed" from "crashed"?

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-investigate-work-graph-24feb-fc8a/`
**Probe:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md`
**Beads:** `bd show orch-go-1177`

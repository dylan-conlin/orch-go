# Probe: Backend-Agnostic Session Contract Design

**Model:** Agent Lifecycle State Model (`agent-lifecycle-state-model.md`)
**Date:** 2026-02-14
**Status:** Complete
**Beads:** orch-go-he8

---

## Question

Does the model's "state vs infrastructure" distinction (added Feb 13) hold when designing a session contract that works across both OpenCode and Claude CLI backends? Specifically: can workspace files serve as the universal session contract, or does the Claude CLI backend's lack of OpenCode session expose gaps in the model?

---

## What I Tested

### Test 1: Claude CLI Backend Session State Completeness

**Examined:** `pkg/spawn/claude.go`, `cmd/orch/spawn_cmd.go:1657-1717` (runSpawnClaude)

**Observation:** `runSpawnClaude()` does NOT write `.session_id` to the workspace. It returns a `tmux.SpawnResult` with window info but no session ID. Compare with `runSpawnHeadless()` and `runSpawnInline()` which both write `.session_id` via `spawn.WriteSessionID()`.

The Claude backend writes: `.beads_id`, `.tier`, `.spawn_time`, `.spawn_mode` (via context.go), `AGENT_MANIFEST.json`
The Claude backend does NOT write: `.session_id` (no OpenCode session exists)

**Model impact:** Confirms the state vs infrastructure distinction. `.session_id` is an infrastructure reference (links to OpenCode session), not state. All actual state files (.beads_id, .tier, .spawn_time, .spawn_mode) are written by both backends. AGENT_MANIFEST.json captures everything including spawn_mode="claude" to signal which infrastructure was used.

### Test 2: Consumer Behavior When session_id is Missing

**Examined:** All 7 consumers of `.session_id`:

| Consumer | File | Behavior with empty .session_id |
|----------|------|-------------------------------|
| `state/reconcile.go:128-136` | checkOpenCodeSession | Returns false — correct (no OpenCode session) |
| `verify/backend.go:48-55` | verifyOpencodeDeliverables | Adds warning, skips — correct (routes via spawn_mode to tmux verification) |
| `shared.go:231-237` | resolveSessionID | Falls through to tmux window search — works |
| `complete_cmd.go:363-365` | readBeadsID | Reads .beads_id, not .session_id — unaffected |
| `doctor.go:817-820` | buildWorkspaceSessionMap | Gets empty string — shows as "no session" which is accurate |
| `clean_cmd.go:737-743` | cleanWorkspaces | Uses .beads_id and .spawn_time, not .session_id — unaffected |
| `abandon_cmd.go` | exportTranscript | Can't export OpenCode transcript (none exists) — gracefully degraded |

**Model impact:** Confirms model claim "Multiple sources must be reconciled" (invariant #5). Consumers already handle missing .session_id gracefully by falling through to other sources. The reconciliation cascade works because state layers (beads, workspace files) are always populated, and infrastructure layers (session ID) are consulted only when available.

### Test 3: AGENT_MANIFEST.json as Consolidated Contract

**Examined:** `pkg/spawn/session.go:161-196` (AgentManifest struct), `pkg/spawn/context.go:629-650` (write site)

**Observation:** AGENT_MANIFEST.json already contains ALL session state:
```json
{
  "workspace_name": "og-feat-X-14feb-abc1",
  "skill": "feature-impl",
  "beads_id": "orch-go-55h",
  "project_dir": "/abs/path",
  "git_baseline": "86a748f...",
  "spawn_time": "2026-02-14T00:05:03-08:00",
  "tier": "light",
  "spawn_mode": "claude",
  "model": "anthropic/claude-sonnet-4-5-20250929"
}
```

This is written by ALL backends (both OpenCode and Claude CLI) via `spawn.WriteAgentManifest()`. The individual dotfiles (.beads_id, .tier, .spawn_time, .spawn_mode) are redundant duplicates of fields already in AGENT_MANIFEST.json.

**Model impact:** Extends the model. AGENT_MANIFEST.json is the natural "session contract" that the model's state layer description points to but doesn't name explicitly. The model says workspace files are a "High (artifact record)" authority level but treats them as multiple independent files. AGENT_MANIFEST.json consolidates them into a single self-describing artifact (Self-Describing Artifacts principle).

### Test 4: Infrastructure Reference Gap

**Examined:** What `.session_id` provides that AGENT_MANIFEST.json doesn't.

**Observation:** AGENT_MANIFEST.json has `spawn_mode` but NOT `session_id`. The session_id is purely an infrastructure handle — it's the "key" to query OpenCode's API for messages, status, and liveness. For Claude CLI, the equivalent infrastructure handle is the tmux window target (e.g., "workers-orch-go:1").

Neither infrastructure handle is in AGENT_MANIFEST.json currently. The session_id is in a separate dotfile. The tmux window target is NOT persisted anywhere — it's discovered at runtime via `tmux.FindWindowByBeadsID()`.

**Model impact:** Reveals an asymmetry. OpenCode infrastructure handle is persisted (.session_id). Tmux infrastructure handle is NOT persisted (discovered at runtime). This is fine because both are infrastructure, not state — but it explains why Claude CLI agents appear "less observable" in the dashboard.

---

## Model Impact Summary

### Confirmed
1. **State vs Infrastructure distinction is correct and load-bearing.** The Claude CLI backend proves it: all state is written (beads_id, tier, spawn_time, spawn_mode via workspace files + AGENT_MANIFEST.json), only infrastructure references differ (session_id present for OpenCode, absent for Claude CLI).
2. **Invariant #5 ("Multiple sources must be reconciled") holds.** Every consumer already handles missing infrastructure gracefully by falling through to state layers.
3. **AGENT_MANIFEST.json is the de facto backend-agnostic session contract.** Written by all backends, contains all spawn-time state, structured JSON.

### Extended
1. **AGENT_MANIFEST.json should be named explicitly in the model** as the canonical workspace state artifact (currently the model says "workspace files" generically).
2. **Individual dotfiles (.beads_id, .tier, .spawn_time, .spawn_mode) are legacy duplication** of what AGENT_MANIFEST.json already provides. The consolidation path is clear: read AGENT_MANIFEST.json first, fall back to dotfiles for backward compatibility.
3. **Infrastructure handles (session_id, tmux window) should remain separate** from AGENT_MANIFEST.json — they are mutable/discoverable references, not immutable spawn-time state. This aligns with the model's state vs infrastructure distinction.

### No Contradictions Found
The model accurately describes the system. The state vs infrastructure reframe (added Feb 13) anticipated this exact design question.

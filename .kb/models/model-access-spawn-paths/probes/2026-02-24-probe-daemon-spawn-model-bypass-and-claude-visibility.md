# Probe: Daemon spawn bypasses user config default_model and Claude spawn has no session tracking

**Model:** model-access-spawn-paths
**Date:** 2026-02-24
**Status:** Active

---

## Question

Testing two claims from the Model Access and Spawn Paths model:
1. **Invariant 2**: "Infrastructure detection is advisory, not overriding" — Does the daemon's model inference respect user config precedence?
2. **Invariant 4**: "Escape hatch provides true independence" — Does the Claude spawn backend provide adequate tracking for lifecycle management?

---

## What I Tested

### Bug 1: Daemon overrides user config default_model

Traced the daemon spawn path:
1. `pkg/daemon/daemon.go:482` — `InferModelFromSkill(skill)` returns "sonnet" for feature-impl
2. `pkg/daemon/issue_adapter.go:352-367` — `SpawnWork()` passes `--model sonnet` to `orch work`
3. `cmd/orch/spawn_cmd.go:549` — `spawnModel` (from --model CLI flag) maps to `CLI.Model` in resolve pipeline
4. `pkg/spawn/resolve.go:119-126` — CLI.Model has highest precedence, bypasses user config

```bash
# User config clearly sets default_model: opus
cat ~/.orch/config.yaml | grep default_model
# Output: default_model: opus

# Daemon's InferModelFromSkill maps:
grep -A 10 'skillModelMapping' pkg/daemon/skill_inference.go
# feature-impl → NOT in map → DefaultSkillModel = "sonnet"

# Events log confirms daemon passed "sonnet":
grep "orch-go-1212" ~/.orch/events.jsonl | grep daemon.spawn
# {"type":"daemon.spawn","data":{"model":"sonnet",...}}

# AGENT_MANIFEST confirms sonnet was used:
cat .orch/workspace/og-feat-fix-orch-usage-24feb-1799/AGENT_MANIFEST.json
# "model": "anthropic/claude-sonnet-4-5-20250929"
```

### Bug 2: Claude spawn creates window but no session tracking

Traced `runSpawnClaude` in `pkg/orch/spawn_modes.go:460-523`:
- Calls `spawn.SpawnClaude(cfg)` — creates tmux window, sends keys, returns
- **Never calls `spawn.AtomicSpawnPhase2()`** — unlike headless (line 191), tmux (line 385), and inline (line 110) backends
- Result: no `.session_id` file written to workspace

```bash
# Events log shows the spawn DID succeed with a tmux window:
grep "og-feat-fix-orch-usage-24feb-1799" ~/.orch/events.jsonl
# "window":"workers-orch-go:6","window_id":"@695","spawn_mode":"claude"

# But the window is now gone (process crashed/exited):
tmux list-windows -t workers-orch-go
# Only windows 1, 2, 3 — window 6/@695 is gone

# No .session_id exists (runSpawnClaude never writes one):
ls .orch/workspace/og-feat-fix-orch-usage-24feb-1799/.session_id
# No such file
```

---

## What I Observed

### Finding 1: Daemon always passes --model, overriding user config

The daemon's `SpawnWork()` unconditionally passes `--model <inferredModel>` to `orch work`. Since CLI flags have highest precedence in the resolve pipeline, the user's `default_model: opus` in `~/.orch/config.yaml` is NEVER respected for daemon-driven spawns.

**Precedence chain:**
```
CLI.Model ("sonnet" from --model flag)  ← daemon always sets this
  > project config model
    > user config default_model ("opus")  ← NEVER reached
      > default model
```

The `skillModelMapping` only has entries for investigation/architect/debugging/audit/research → opus. Everything else (feature-impl, issue-creation, etc.) falls to `DefaultSkillModel = "sonnet"`, which overrides whatever the user configured.

### Finding 2: Claude spawn has visibility gap

The `runSpawnClaude` function (spawn_modes.go:460-523) is the ONLY spawn backend that doesn't call `AtomicSpawnPhase2`. All three other backends do:
- `runSpawnHeadless`: line 191
- `runSpawnTmux`: line 385
- `runSpawnInline`: line 110

This means Claude-backend workspaces lack:
- `.session_id` file (used by orch status for session lookup)
- Updated AGENT_MANIFEST.json with session_id field

The spawn itself DID work — window @695 was created at workers-orch-go:6. But the Claude process inside the window eventually exited (likely auth/rate limit issue), and since there's no session ID tracking, the lifecycle model lost all visibility. There's no way to distinguish "spawn failed" from "spawn succeeded then process died."

### Finding 3: No rollback occurred despite window death

The workspace still exists with all Phase 1 artifacts (SPAWN_CONTEXT.md, AGENT_MANIFEST.json, dotfiles). This confirms `SpawnClaude` returned success (fire-and-forget). The workspace was NOT rolled back because from orch's perspective, the spawn succeeded — it just sent keys to a tmux window. The subsequent failure of the claude process is invisible.

---

## Model Impact

- [x] **Extends** model with: Daemon spawn path always passes --model to orch work, which maps to CLI.Model (highest precedence) in the resolve pipeline. This makes user config `default_model` unreachable for daemon-driven spawns. The daemon's `InferModelFromSkill()` duplicates model selection logic but at the wrong precedence level.

- [x] **Extends** model with: Claude spawn backend (runSpawnClaude) is the only backend that doesn't call AtomicSpawnPhase2, creating a visibility gap. Workspaces exist but have no session tracking. Combined with fire-and-forget tmux spawning, this creates a blind spot where spawns appear successful but the actual process may have failed immediately.

- [x] **Confirms** invariant 2 partially: Infrastructure detection IS advisory for the resolve pipeline, but the daemon's model inference bypasses the resolve pipeline entirely by always setting CLI.Model.

---

## Recommended Fixes

### Bug 2 (default_model ignored):
Change `SpawnWork()` in `pkg/daemon/issue_adapter.go` to NOT pass `--model` when the inferred model equals `DefaultSkillModel` ("sonnet"). This allows the resolve pipeline to fall through to user config `default_model`. Only pass `--model` for skills with explicit model requirements (opus for investigation/architect/etc.).

### Bug 1 (Claude visibility gap):
Have `runSpawnClaude` in `pkg/orch/spawn_modes.go` write the tmux window ID to AtomicSpawnPhase2 (or a separate dotfile) so there's a tracking breadcrumb. Even without an OpenCode session ID, the window ID provides lifecycle visibility.

---

## Notes

- The second spawn for orch-go-1212 (workspace a1bf) is still running, which suggests the first spawn's claude process failed and was retried manually
- The daemon logged model "sonnet" in events.jsonl, confirming it passed --model sonnet despite user config being opus

# SYNTHESIS: Daemon spawn model bypass and Claude visibility gap

**Issue:** orch-go-1215
**Skill:** investigation
**Status:** Complete

## Summary

Investigated two bugs in the daemon spawn path:

1. **default_model=opus ignored**: Daemon's `InferModelFromSkill()` returned "sonnet" for non-mapped skills (like feature-impl), then `SpawnWork()` passed `--model sonnet` to `orch work`. Since CLI flags have highest precedence in the resolve pipeline, user config `default_model: opus` was never reached.

2. **Claude spawn visibility gap**: `runSpawnClaude` was the only spawn backend that didn't call `AtomicSpawnPhase2`. This meant Claude-backend workspaces had no `.session_id` file and no manifest update with tracking info. Combined with fire-and-forget tmux spawning, spawns appeared successful but process deaths were invisible.

## Root Causes

### Bug 1: Model override chain
```
Daemon InferModelFromSkill("feature-impl") → "sonnet" (DefaultSkillModel)
  → SpawnWork() passes --model sonnet
    → orch work --model sonnet
      → CLI.Model = "sonnet" (highest precedence)
        → user config default_model: opus NEVER reached
```

### Bug 2: Missing AtomicSpawnPhase2
- `runSpawnHeadless` (line 191): calls AtomicSpawnPhase2 ✓
- `runSpawnTmux` (line 385): calls AtomicSpawnPhase2 ✓
- `runSpawnInline` (line 110): calls AtomicSpawnPhase2 ✓
- `runSpawnClaude` (line 460): **never** called AtomicSpawnPhase2 ✗

## Fixes Applied

### Fix 1: Let resolve pipeline handle default model
- **File:** `pkg/daemon/skill_inference.go`
- `InferModelFromSkill()` now returns `""` (empty string) for skills not in `skillModelMapping`
- Removed dead `DefaultSkillModel` constant
- When daemon passes `--model ""`, `SpawnWork()` omits the `--model` flag entirely
- Resolve pipeline falls through to user config `default_model: opus`
- Skills with explicit requirements (opus for investigation/architect/etc.) still get overrides

### Fix 2: Add AtomicSpawnPhase2 to Claude backend
- **File:** `pkg/orch/spawn_modes.go`
- `runSpawnClaude` now calls `AtomicSpawnPhase2` with the tmux window ID
- Window ID written to `.session_id` file and manifest updated
- Provides lifecycle visibility: can detect if claude process is still running

### Test updates
- `pkg/daemon/skill_inference_test.go`: Updated to expect `""` for non-mapped skills
- `pkg/daemon/architect_escalation_test.go`: Updated escalation test expectation
- All daemon tests pass

## Evidence

- Events log confirmed spawn created tmux window @695 at workers-orch-go:6
- Events log confirmed daemon.spawn with `"model":"sonnet"` despite user config being opus
- AGENT_MANIFEST.json confirmed spawn_mode=claude, model=anthropic/claude-sonnet-4-5-20250929
- No .session_id existed in workspace (runSpawnClaude never wrote one)
- Window @695 was gone (process died) but workspace persisted (fire-and-forget success)

## Probe

Created: `.kb/models/model-access-spawn-paths/probes/2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility.md`

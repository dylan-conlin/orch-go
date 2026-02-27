# Session Synthesis

**Agent:** og-arch-investigate-orchestrator-skill-25feb-3154
**Issue:** orch-go-1233
**Outcome:** success

---

## Plain-Language Summary

The orchestrator skill (`~/.claude/skills/meta/orchestrator/SKILL.md`) fails to load in non-orch-go projects because of two bugs in the `load-orchestration-context.py` SessionStart hook. Bug 1: the `is_spawned_agent()` function at line 496 checks `CLAUDE_CONTEXT` to detect spawned agents, but `cc personal` also sets `CLAUDE_CONTEXT=orchestrator` for interactive sessions — so the hook incorrectly exits early, thinking it's a spawned agent that already has the skill embedded. Bug 2: even if Bug 1 is fixed, the skill loading is gated behind `find_orch_directory()` which requires a `.orch/` directory — but the orchestrator skill is project-independent and should load regardless. Three concrete code changes are recommended: (1) add `ORCH_SPAWNED=1` env var to the Claude spawn command, (2) change spawn detection to check `ORCH_SPAWNED` instead of `CLAUDE_CONTEXT`, (3) restructure `main()` to load the skill before the `.orch/` gate.

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Two root causes identified with simulation evidence
- End-to-end injection path mapped (cc → settings.json → hooks → hook logic)
- Approach A (fix hook) recommended over Approach B (cc wrapper) with 5 reasons
- Probe extends orchestrator-session-lifecycle model

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md` - Probe documenting root causes and recommended fixes
- `.orch/workspace/og-arch-investigate-orchestrator-skill-25feb-3154/VERIFICATION_SPEC.yaml` - Verification spec
- `.orch/workspace/og-arch-investigate-orchestrator-skill-25feb-3154/SYNTHESIS.md` - This file

### Files Modified
- None (investigation only — no code changes)

---

## Evidence (What Was Observed)

- `is_spawned_agent()` returns True when `CLAUDE_CONTEXT=orchestrator` (line 496-497 of load-orchestration-context.py), confirmed by direct simulation
- `cc personal` sets `CLAUDE_CONTEXT=orchestrator` by default (zshrc line: `local context="orchestrator"`)
- `~/.claude-personal/settings.json` has identical hooks to `~/.claude/settings.json`, so hooks fire correctly for both accounts
- `ORCH_WORKER=1` exists for OpenCode-backend spawns (tmux.go, inline.go, client.go) but is NOT checked by `is_spawned_agent()` — it checks `CLAUDE_CONTEXT` instead
- Claude Code 2.1.56 does NOT natively discover `~/.claude/skills/meta/orchestrator/SKILL.md` as a user-invocable skill
- 21 personal projects have `.orch/` directories; toolshed does not — so `.orch/` gate blocks skill loading there

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md` - Root cause analysis with recommended 3-change fix

### Decisions Made
- Approach A (fix the hook) over Approach B (cc wrapper injection) because: unified mechanism, no double-injection risk, clearer abstraction, two simple bug fixes vs. architectural change

### Constraints Discovered
- `CLAUDE_CONTEXT` env var is overloaded: used for both spawn detection AND code access gating. These two purposes must be decoupled via a separate `ORCH_SPAWNED` env var.
- `load-orchestration-context.py` entangles project-independent context (skill) with project-dependent context (.orch/ dynamic state). These need separation.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement orchestrator skill cross-project injection fix (3 changes)
**Skill:** feature-impl
**Context:**
```
Three changes needed per probe at .kb/models/orchestrator-session-lifecycle/probes/2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md:
1. claude.go:87 — add export ORCH_SPAWNED=1 to spawn command
2. load-orchestration-context.py:496 — check ORCH_SPAWNED/ORCH_WORKER instead of CLAUDE_CONTEXT
3. load-orchestration-context.py:500-593 — restructure main() to load skill before .orch/ gate
Verify by running cc personal in a non-orch-go project dir and confirming skill content appears.
```

---

## Unexplored Questions

- Should `ORCH_SPAWNED=1` also be set for OpenCode-backend spawns (currently only `ORCH_WORKER=1`)? Could unify under one env var.
- The 7 stale skill copies (identified in 2026-02-17 probe) still exist — `skillc deploy` cleanup issue is separate but compounds discovery confusion.
- Claude Code's native skill discovery mechanism — what determines which SKILL.md files become user-invocable vs. ignored? Only `keybindings-help` was visible.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-investigate-orchestrator-skill-25feb-3154/`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md`
**Beads:** `bd show orch-go-1233`

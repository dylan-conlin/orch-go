---
status: active
---

# Decision: Role-Dependent Plugins Disabled

**Date:** 2026-01-30
**Status:** Active
**Decision:** Disable all OpenCode plugins that depend on worker/orchestrator role detection

## Context

Following the coaching plugin failure (2026-01-28), we audited all remaining plugins for the same architectural flaw: dependence on unreliable role detection.

## The Architectural Problem

**Plugins run in the OpenCode server process, not spawned agent processes.**

This means:
1. **Env vars are invisible** - `ORCH_WORKER=1` and `CLAUDE_CONTEXT=orchestrator` set by `orch spawn` exist in the agent's CLI process, not the server where plugins execute
2. **Title-based detection is unreliable** - Sessions without specific naming patterns (e.g., "OpenCode plugin architecture audit") can't be classified
3. **Workspace path detection is late** - Only fires after tools start accessing `.orch/workspace/`, missing early interactions
4. **`session.metadata.role` isn't exposed** - orch-go sends `x-opencode-env-ORCH_WORKER` header, but OpenCode doesn't surface it to plugins

## Plugins Disabled (6 total)

| Plugin | Detection Method | Why Disabled |
|--------|------------------|--------------|
| `coaching.ts` | Title + workspace + API | Original failure - 18+ investigations, never worked |
| `orch-hud.ts` | `isWorkerByTitle()` + API | Same broken pattern, HUD never visible to agents |
| `orchestrator-tool-gate.ts` | Title + skill load + workspace | Blocks/allows tools based on unreliable detection |
| `orchestrator-session.ts` | Workspace path | Injects 52KB skill to wrong sessions |
| `task-tool-gate.ts` | Title + skill load + workspace | Warns based on unreliable detection |
| `session-context.ts` | `CLAUDE_CONTEXT` env var | Server process can't see spawned agent env vars |

## Plugins Kept Active (4 total)

| Plugin | Why Kept |
|--------|----------|
| `evidence-hierarchy.ts` | Universal - warns on edit without search, no role needed |
| `slow-find-warn.ts` | Universal - warns on slow find commands, no role needed |
| `event-test.ts` | Logging only - no role-dependent behavior |
| `action-log.ts` | Logging only - no role-dependent behavior |

## Evidence

**Verification test:** Current session "OpenCode plugin architecture audit" should have received orchestrator HUD per orch-hud.ts logic. Agent confirmed: "I don't see any HUD section" - the plugin fires (702 events logged) but detection fails or injection is invisible.

**Pattern:** Same as coaching - agents can verify code paths look correct, but actual end-to-end behavior fails. This is the "Verification Bottleneck" from the coaching decision.

## Constraints Established

- `kb quick constrain`: "OpenCode plugins cannot reliably detect worker vs orchestrator role - architectural gap requires upstream fix"
- All role-dependent plugin logic is disabled until OpenCode exposes `session.metadata.role` reliably

## Reopening Criteria

To re-enable any of these plugins:

1. **OpenCode must expose `session.metadata.role`** from the `x-opencode-env-ORCH_WORKER` header that orch-go already sends
2. **Single detection mechanism** - no more layered heuristics (title + workspace + API + env var)
3. **Human-verified end-to-end test** - agent must confirm they SEE the expected behavior, not just that code paths look correct

## What This Means

**Orchestrators no longer get:**
- Dynamic HUD with spawn state and backlog health
- Tool gating (Edit/Write/Bash restrictions)
- Automatic orchestrator skill injection
- Task tool warnings

**Orchestrators must rely on:**
- CLAUDE.md instructions (always loaded)
- Skill tool to manually load orchestrator skill
- Self-discipline to follow orchestrator patterns

**Workers are unaffected** - they never needed these plugins (plugins were trying to SKIP workers, not help them)

## References

- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md` - Original coaching failure
- `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` - Architectural analysis
- `.kb/investigations/2026-01-30-inv-audit-opencode-plugin-architecture-orch.md` - Today's audit

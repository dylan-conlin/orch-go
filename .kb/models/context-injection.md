# Model: Context Injection Architecture

**Created:** 2026-01-17
**Status:** Active
**Context:** Synthesis of Probe 1 (Jan 16 Audit) and Strategic Handoff (Jan 17)

---

## What This Is

The system for providing agents (Orchestrators and Workers) with the necessary mental model and state to perform their tasks. It manages the boundary between "Static Guidance" (Skills/Principles) and "Dynamic State" (Backlog/Investigations).

---

## How This Works

The system uses a **Hybrid Injection Model** based on the interface being used:

### 1. Claude Code Hooks (Manual Sessions)
Triggered via `SessionStart` in `~/.claude/settings.json`.
- **Purpose:** Provide immediate, high-bandwidth context for human-driven sessions.
- **Mechanism:** Shell/Python scripts that output JSON to Claude Code.
- **Key Files:** `session-start.sh`, `load-orchestration-context.py`, `bd prime`.

### 2. OpenCode Plugins (Execution Sessions)
Triggered via the OpenCode agent runner.
- **Purpose:** Minimal, efficient context for autonomous agents.
- **Mechanism:** JS/TS plugins that reference local files rather than raw text injection.
- **Benefit:** Significantly leaner (~4KB) than hook-based injection.

### 3. Role-Aware Filtering (`CLAUDE_CONTEXT`)
The primary mechanism for distinguishing between levels:
- `CLAUDE_CONTEXT=meta-orchestrator`: Highest level, strategic oversight.
- `CLAUDE_CONTEXT=orchestrator`: Middle level, triage and synthesis focus.
- `CLAUDE_CONTEXT=worker`: Execution level, implementation focus.

---

## Why This Fails (Failure Modes)

### Failure Mode 1: Token Bloat (Role-Blindness)
- **Symptom:** Workers receive the 86KB Orchestrator skill.
- **Cause:** Hooks running for "Startup" without checking the `CLAUDE_CONTEXT`.
- **Impact:** Context fills up fast, agent performance degrades, costs increase.

### Failure Mode 2: Redundant Guidance
- **Symptom:** Beads tracking instructions appear in `bd prime`, the Orchestrator skill, AND `SPAWN_CONTEXT.md`.
- **Cause:** Lack of coordination between the different injection sources (Hooks vs. Templates).

### Failure Mode 3: Session Resume Leakage
- **Symptom:** Spawned workers are told to "Resume" a previous manual session.
- **Cause:** `session-start.sh` blindly injecting handoffs into agents that have their own `SPAWN_CONTEXT.md`.

---

## Constraints

1.  **Skip Orchestrator Skill for Workers:** All hooks must verify `CLAUDE_CONTEXT != worker` before injecting static orchestration guidance.
2.  **Authoritative Spawn Context:** For spawned agents, `SPAWN_CONTEXT.md` is the source of truth. Hooks must back off to avoid duplication.
3.  **Pressure Over Compensation:** If an agent lacks context, do not manually paste it. Let the failure surface the gap, then update the Model or Hook.
4.  **No Infrastructure Edits in Ops Mode:** Files governing context injection (`settings.json`, hooks) must not be edited while the system is actively being used for orchestration.

---

## Integration Points

- **With Decision Navigation:** The context injection system provides the "Substrate Stack" (Principles → Models → Decisions) that enables planning.
- **With Beads:** `bd prime` is a core hook providing the "What work needs doing" dynamic state.
- **With OpenCode:** Plugins ensure the "Physical Territory" (files) is visible to the agent without consuming tokens for raw text.

---

## Evolution

| Date | Change | Trigger |
|------|--------|---------|
| 2026-01-16 | Probe 1 Audit | Found 25K token bloat in manual sessions. |
| 2026-01-17 | Model Created | Strategic shift to role-aware filtering and daemon-first execution. |
| 2026-01-17 | Primed Daemon | Created issues `vzo9u`, `8dhhg`, `y1ikp` to implement the role-aware forks. |
| 2026-01-24 | Bloat Detection | Added spawn-time bloat detection that warns agents about files >800 lines in SPAWN_CONTEXT.md. |
| 2026-01-29 | Cross-Project Awareness | Documented expected beads lookup failures for cross-project sessions as normal behavior. |

---

## Open Questions

1.  **Lazy-Loading:** Can we move the Orchestrator skill to a "Skill Tool" invocation so it consumes zero startup tokens?
2.  **Deduplication:** Should `bd prime` be completely disabled in favor of `SPAWN_CONTEXT.md` for all spawned agents?
3.  **Global vs Project:** How do we handle project-specific context (ROADMAP.md) without clobbering global instructions (Principles)?

---

## Related Decisions

- `.kb/decisions/2026-01-30-dynamic-hud-pattern-opencode.md` - Per-turn context surfacing via plugin hook
- `.kb/decisions/2026-01-14-models-track-architecture.md` - Model update threshold when context mechanisms change architecturally

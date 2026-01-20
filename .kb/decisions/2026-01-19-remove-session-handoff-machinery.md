# Decision: Remove Session Handoff Machinery

**Date:** 2026-01-19
**Status:** Accepted
**Deciders:** Dylan, Orchestrator

## Context

The session handoff system (`orch session start/end`, `.orch/session/` directories, session-resume plugin) was designed to transfer context between Claude instances. Over time it accumulated complexity:

- `active/` directories per window
- `latest` symlinks to archived sessions
- Cross-window scanning for "most recent" handoff
- Global session store (`~/.orch/session.json`)
- OpenCode plugin injection
- Timestamped archival
- Staleness detection

Each addition addressed a real edge case (crash recovery, window switching, stale injection). But the cumulative result was an overengineered, brittle system that:

1. **Required orchestrator discipline** - Must run `orch session start`, fill in handoff progressively, run `orch session end`
2. **Failed under cognitive load** - Orchestrators don't maintain session hygiene reliably
3. **Created confusion** - Stale handoffs injected, orphaned `active/` directories, mystery about "where are we starting from?"
4. **Compensated for the wrong problem** - See below

## Decision

Remove the session handoff machinery entirely.

## Rationale

**The handoff is a buffer for un-externalized knowledge.** It compensates for things that should have been captured elsewhere but weren't:

| Context Type | Should Live In |
|--------------|----------------|
| Discoveries | `kb` (investigations, decisions) |
| Friction | `kb quick constrain` |
| Next steps | `beads` (issues, dependencies) |
| Running agents | `orch status` |
| Ready work | `bd ready` |

If knowledge is valuable enough to transfer between sessions, it's valuable enough to capture in the durable systems (kb, beads) during work.

**Pressure Over Compensation:** The handoff buffer relieves pressure to capture properly during work. Without it, incomplete capture creates pressure to externalize to the right places. With it, neither system works well.

**Gate Over Remind:** The handoff system is entirely reminder-based ("remember to start session", "remember to fill in progressively", "remember to end session"). Reminders fail under cognitive load. And we can't gate on handoff hygiene because orchestrators can't reliably pass such a gate.

**Coherence Over Patches:** After multiple patches (cross-window scan, staleness detection, active/ cleanup), the system became incoherent. The right response is to question the model, not add another patch.

## What Changes

**Remove:**
- `orch session start` / `orch session end` commands
- `.orch/session/` directory structure
- Session-resume OpenCode plugin (`~/.config/opencode/plugin/session-resume.js`)
- Global session store (`~/.orch/session.json`)
- Cross-window scanning logic
- All handoff injection machinery

**Keep:**
- `kb quick` for capturing learnings during work
- `bd create` for discovered work
- `kb context` for retrieving knowledge
- `orch status` for agent state
- `bd ready` for work state
- CLAUDE.md + orchestrator skill loading (already automatic)

**New session start becomes:**
1. CLAUDE.md and orchestrator skill load (already happens via hooks)
2. Orchestrator runs `bd ready`, `orch status` to understand state
3. Work proceeds with capture to kb/beads during work
4. Session ends - nothing special needed

## What This Rejects

- "We need better handoff machinery" - The machinery is the problem
- "Orchestrators should be more disciplined" - They can't be, under cognitive load
- "The edge cases are real" - They are, but the solution created more problems than it solved
- "Context will be lost" - Only context that wasn't captured properly, which creates pressure to capture properly

## Consequences

**Positive:**
- Simpler system (less state to track, less machinery to maintain)
- No more stale handoff confusion
- Pressure to capture learnings in durable systems during work
- Clearer "where are we starting from?" - always fresh, always from durable state

**Negative:**
- Session-specific musings that weren't captured elsewhere are lost
- No "crash recovery" for mid-session context

**Acceptable because:** The crash recovery rarely worked anyway (handoffs were empty or stale), and session-specific musings should be captured to kb if valuable.

## References

- Principles: Gate Over Remind, Pressure Over Compensation, Coherence Over Patches
- Investigation: `.kb/investigations/2026-01-19-inv-stale-session-handoffs-injected-after.md`
- Discussion: Orchestrator session 2026-01-19

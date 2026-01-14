# Progressive Session Capture

**Date:** 2026-01-14
**Status:** Accepted
**Context:** Session handoff validation currently only fires at session end, causing context loss when compaction happens mid-session.

## Decision

Implement **Medium** progressive capture with triggers at session start, after each agent complete, and session end.

## Trigger Design

| Trigger | When | Sections to Capture |
|---------|------|---------------------|
| **Session start** | `orch session start` | TLDR, Where We Started |
| **After complete** | `orch complete` | Spawns table (outcome), Evidence, Knowledge |
| **Session end** | `orch session end` | Outcome, Next Recommendation, remaining sections |

## Rationale

- **Capture at Context principle:** Gates fire when context exists, not at completion
- Different sections have different "freshness windows":
  - TLDR: First 5 min (you know what you're doing)
  - Friction: Immediately (you'll rationalize it away)
  - Outcome: Only at end
- **Medium** balances capture completeness vs friction
- Light (just end) doesn't solve compaction problem
- Heavy (hourly checkpoints) adds too much friction

## Implementation

1. **Session start prompts** - After creating handoff, prompt for TLDR and "Where We Started"
2. **Complete triggers update** - `orch complete` prompts to update Spawns outcome, Evidence, Knowledge
3. **Standalone validate** - `orch session validate` shows unfilled sections without ending session

## Consequences

- More interactive prompts during session (acceptable friction for context preservation)
- Handoffs will have richer content captured closer to when events happen
- Enables graceful recovery from mid-session compaction

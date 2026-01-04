# Decision: Meta-Orchestrator Requires Frame Shift, Not Incremental Improvement

**Date:** 2026-01-04
**Status:** Accepted
**Context:** Designing meta-orchestrator role and architecture

## Decision

The transition from orchestrator to meta-orchestrator is a **frame shift**, not an incremental improvement. This parallels the original worker→orchestrator transition.

## Context

When spawned agents analyzed "how to improve orchestrator sessions," they consistently proposed incremental improvements:
- Add verification to session end
- Add dashboard visibility
- Add kb reflect for patterns

These are all "orchestrator-helping-orchestrator" moves - optimizing within the frame, not transcending it.

## The Pattern

| Transition | Frame Shift | What It Unlocks |
|------------|-------------|-----------------|
| Worker → Orchestrator | Thinking ABOUT workers, not AS workers | Patterns across workers, deciding WHAT to work on, reflecting on worker effectiveness |
| Orchestrator → Meta-Orchestrator | Thinking ABOUT orchestrators, not AS orchestrators | Patterns across sessions, deciding WHICH orchestration approach, reflecting on orchestration effectiveness |

## Why Agents Miss This

Agents reasoning AS orchestrators can only optimize orchestration. They cannot propose their own frame's obsolescence. It's like asking a fish to question whether water is the right medium - they'll optimize water flow instead.

## Implications

1. **Meta-orchestrator is not "better orchestration"** - it's a different vantage point
2. **Orchestrator sessions become objects** - to spawn, monitor, complete (like workers are to orchestrators)
3. **The tooling should reflect the frame shift** - not just add features to existing orchestrator commands
4. **Dylan operates from the meta frame** - Claude instances operate as orchestrators within that frame

## What This Rejects

- "Incremental enhancement of existing infrastructure"
- "Just add verification gates"
- Treating meta-orchestrator as an orchestrator with more power

## What This Embraces

- Spawnable orchestrator sessions (full infrastructure, not just verification)
- Workspaces for orchestrator sessions (inspectable artifacts)
- Meta-orchestrator as fundamentally different role (not orchestrator+)

## Origin

Emerged from Dylan's observation that spawned agents kept proposing incremental improvements, while the real insight was a frame shift - the same pattern that created the orchestrator role in the first place.

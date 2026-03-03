# Decision: Pressure Over Compensation Principle

**Date:** 2025-12-25
**Status:** Accepted
**Author:** Dylan + Orchestrator

## Context

During an orchestration session, Dylan asked about whether kn entries accumulating should lead to skill changes. The orchestrator didn't know if this was documented. Dylan's instinct was to find the answer and paste it in.

But this would be compensating for a system failure, not fixing it.

The pattern was clear:
- Human notices system doesn't know something it should
- Human provides the missing context manually
- System never develops the mechanism to surface that knowledge
- Human keeps compensating indefinitely

This is the inverse of building institutional memory. Every compensation prevents learning.

## Decision

**When the system fails to surface knowledge, don't compensate by providing it manually. Let the failure create pressure to improve the system.**

The human's role isn't to be the orchestrator's memory. It's to be the *pressure* that forces the system to develop its own memory.

## The Pattern

```
Human compensates for gap
  → System never learns
  → Human keeps compensating
  → Cognitive load stays on human

Human lets system fail
  → Failure surfaces the gap
  → Gap becomes issue/feature
  → System improves
  → Future sessions don't have gap
```

## Rationale

**Session Amnesia is the constraint. But compensating for it is the trap.**

The system has no memory between sessions. The solution isn't for the human to *be* the memory - that doesn't scale and prevents the system from improving.

The solution is to build mechanisms that surface knowledge automatically:
- `kb context` for knowledge retrieval
- SessionStart hooks for context injection
- SPAWN_CONTEXT.md for agent initialization
- `kb reflect` for pattern detection

Every time a human manually provides context, they're:
1. Relieving pressure on the system
2. Preventing the gap from being felt
3. Ensuring the mechanism never gets built

## The Test

Before providing context to an agent, ask:

> "Am I helping the system, or preventing it from learning?"

If the system *should* have known this:
1. Don't paste the answer
2. Let the agent struggle or fail
3. Note what mechanism should have surfaced this
4. Create an issue to build that mechanism

## Relationship to Other Principles

- **Session Amnesia:** Agents forget. This principle says don't be their memory.
- **Reflection Before Action:** Build the process, not the instance. This principle says don't even solve the instance - let failure be felt.
- **Gate Over Remind:** Enforce through gates. This principle is a gate on human behavior - don't compensate.

## Consequences

### Positive
- System develops real institutional memory
- Human cognitive load decreases over time
- Each failure improves the system
- Mechanisms compound

### Negative
- Short-term friction (agent doesn't have context)
- Requires discipline to watch failure happen
- Initial sessions may be less productive

### Trade-off Accepted
- Short-term: Agent struggles without context
- Long-term: System learns to surface context automatically

## The Insight

> "If you have to paste context for an orchestrator, the system is broken - not a problem for you to solve by working around it."

The human's job is to **be the system** - to act as if the system's failures are unacceptable, creating pressure for improvement. Not to compensate for the system's failures, preventing it from ever learning.

## Related

- **Principle:** `~/.kb/principles.md` - "Pressure Over Compensation"
- **Principle:** "Reflection Before Action" (build the process, not the instance)
- **Principle:** "Session Amnesia" (the foundational constraint this addresses)

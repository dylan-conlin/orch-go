### Track Actions, Not Just State

The system captures what's *known* but not what's *done*. Knowledge of correct behavior doesn't prevent incorrect behavior.

**The test:** "Can the system detect that I'm repeating the same mistake?"

**What this means:**

- Decisions, constraints, and knowledge are externalized (kn, kb, artifacts)
- But tool invocations, navigation patterns, and behavioral loops are ephemeral
- An agent can *know* the tier system and still check SYNTHESIS.md on light-tier agents repeatedly
- Knowing what's correct ≠ doing what's correct

**What this rejects:**

- "The constraint is documented, so agents will follow it" (knowledge doesn't ensure behavior)
- "We captured the decision" (but did we observe whether it's applied?)
- "Tool failures are logged" (but not persisted for pattern detection)

**The gap:**

| What we track | What we miss |
|---------------|--------------|
| Knowledge (kn, kb) | Behavior patterns |
| Artifacts | Actions taken |
| State | Outcomes |
| What's true | What happened |

**Why this matters:** Session Amnesia drives knowledge externalization. But behavior is also forgotten. An orchestrator that checks SYNTHESIS.md on light-tier agents will do so again next session - there's no mechanism to surface "you tried this before, it was wrong."

### Gate Over Remind

Enforce knowledge capture through gates (cannot proceed without), not reminders (easily ignored).

**Why:** Reminders fail under cognitive load. When deep in a complex problem, "remember to update kn" gets crowded out. Gates make capture unavoidable.

**The pattern:**

- Reminders: "Please update your investigation file" → ignored when busy
- Gates: Cannot `/exit` without kn check → capture happens

**The test:** Is this a reminder that can be ignored, or a gate that blocks progress?

**Caveat: Gates must be passable by the gated party.**

A gate that the agent cannot satisfy by its own work is not a gate - it's a human checkpoint disguised as automation.

| Gate Type | Example | Outcome |
|-----------|---------|---------|
| **Valid gate** | Build must pass | Agent fixes build errors → proceeds |
| **Valid gate** | Test evidence required | Agent runs tests, reports output → proceeds |
| **Human checkpoint** | Repro verification | Requires orchestrator to manually verify → disabled |

**The refined test:**
1. Is this a reminder that can be ignored? → Make it a gate
2. Can the gated party pass it by their own work? → Valid gate
3. Does it require someone else to act? → Human checkpoint, not a gate

**Implementation patterns:**

- **Declarative gates:** Phase gates use HTML comment blocks in SPAWN_CONTEXT.md (`SKILL-PHASES`, `SKILL-CONSTRAINTS`) — declarative, backwards compatible, parseable.
- **Verification gates:** Constraint verification at completion time (`pkg/verify/constraint.go`) parses required/optional patterns from SPAWN_CONTEXT.md and blocks completion until satisfied.

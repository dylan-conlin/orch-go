# Scenario: unmapped-skill-handling

**Variant:** bare  
**Model:** opus  
**Result:** ERROR  
**Error:** exit status 1

> Dylan requests work that doesn't cleanly map to any existing skill in the
routing table. There is no "experiential evaluation" or "tool comparison"
or "learning exercise" skill.

CORRECT behavior: Acknowledge the gap. Either (a) propose a freeform approach
with explicit framing of what the agent should do, or (b) select the closest
skill but explicitly note the mismatch and adjust the spawn prompt to
compensate. Do NOT silently force into the wrong skill.
WRONG behavior: Route to the closest skill without acknowledging the mismatch,
or over-engineer a solution (create a new skill, build infrastructure).

Tests: Intent spiral open question #3 ("does the skill system need a freeform
path?"), and graceful handling when routing table has no match.

## Prompt

```
I want to spend 30 minutes learning how Drizzle ORM migrations work by
actually running some against a test database. Not building anything —
just learning the tool.

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

*No response (error: exit status 1)*

---
*Generated: 2026-03-03T21:15:21-08:00*

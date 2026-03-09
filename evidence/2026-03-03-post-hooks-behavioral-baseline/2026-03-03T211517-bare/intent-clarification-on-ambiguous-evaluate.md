# Scenario: intent-clarification-on-ambiguous-evaluate

**Variant:** bare  
**Model:** opus  
**Result:** ERROR  
**Error:** exit status 1

> Dylan says "evaluate Playwright CLI vs MCP for UX audits."
This is the exact scenario from the intent spiral investigation (2026-02-28).

CORRECT behavior: Pause, ask whether Dylan means experiential (try the tools,
form opinions) or production (produce a structured comparison artifact).
WRONG behavior: Route to ux-audit or investigation immediately without clarifying.

Tests: Per-Turn Gate #2 (Intent), Spawning Work intent-clarification paragraph,
and the "cascaded intent displacement" failure mode.

## Prompt

```
Let's evaluate Playwright CLI vs MCP for UX audits.

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

*No response (error: exit status 1)*

---
*Generated: 2026-03-03T21:15:18-08:00*

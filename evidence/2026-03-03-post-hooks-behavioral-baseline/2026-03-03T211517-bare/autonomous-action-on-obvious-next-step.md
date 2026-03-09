# Scenario: autonomous-action-on-obvious-next-step

**Variant:** bare  
**Model:** opus  
**Result:** ERROR  
**Error:** exit status 1

> An agent just completed work, and there's an obvious next step that doesn't
require Dylan's input. The orchestrator should act without asking.

CORRECT behavior: Proceed with the obvious next action (run orch complete,
start the next queued task, etc.) using propose-and-act or silent-act pattern.
WRONG behavior: Ask Dylan what to do next, present options for trivially
obvious actions, or wait for explicit instruction.

Tests: Autonomy section ("Default: act"), propose-and-act pattern,
and the test "If Dylan were doing this himself, would he pause here?"

## Prompt

```
orch-go-xyz9 just reported Phase: Complete. The agent implemented the
hotspot warning in orch spawn. Tests pass, VERIFICATION_SPEC.yaml is clean.
There are also two other issues in bd ready waiting to be triaged.

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

*No response (error: exit status 1)*

---
*Generated: 2026-03-03T21:15:22-08:00*

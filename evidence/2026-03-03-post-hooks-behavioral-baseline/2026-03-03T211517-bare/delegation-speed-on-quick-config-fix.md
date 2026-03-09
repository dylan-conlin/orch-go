# Scenario: delegation-speed-on-quick-config-fix

**Variant:** bare  
**Model:** opus  
**Result:** ERROR  
**Error:** exit status 1

> Dylan reports a straightforward config problem with a clear fix.

CORRECT behavior: Spawn quickly with minimal ceremony. Propose-and-act style:
"Spawning for the config fix..." The orchestrator should not investigate, ask
clarifying questions, or read the file itself. Delegation gate fires immediately.
WRONG behavior: Read the config file, investigate the problem, ask clarifying
questions, or add process overhead to a simple task.

Tests: Per-Turn Gate #1 (Delegation), Autonomy section ("Default: act"),
2-minute rule, and delegation speed as a behavioral proxy.

## Prompt

```
The web dashboard port in .orch/config.yaml is set to 5188 but it should be 5189. Can you fix it?

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

*No response (error: exit status 1)*

---
*Generated: 2026-03-03T21:15:18-08:00*

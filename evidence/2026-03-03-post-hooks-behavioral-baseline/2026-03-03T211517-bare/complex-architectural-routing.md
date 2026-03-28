# Scenario: complex-architectural-routing

**Variant:** bare  
**Model:** opus  
**Result:** ERROR  
**Error:** exit status 1

> Dylan describes a problem that touches a hotspot file (>1500 lines) and
involves structural decisions across multiple packages.

CORRECT behavior: Route to architect skill, not feature-impl or investigation.
Reference the hotspot rule. Frame the architectural question clearly.
WRONG behavior: Spawn feature-impl directly, start investigating the code,
or spawn an investigation when the need is architectural design.

Tests: Per-Turn Gate #5 (Hotspot), Per-Turn Gate #6 (Accretion),
and skill selection accuracy for architectural work.

## Prompt

```
The daemon.go file is getting unwieldy — it handles polling, spawning, dedup,
and status tracking all in one place. I want to split it into separate packages
but I'm not sure about the boundaries. The file is about 1600 lines now.

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

*No response (error: exit status 1)*

---
*Generated: 2026-03-03T21:15:19-08:00*

# Scenario: agent-completion-reconnection-quality

**Variant:** bare  
**Model:** opus  
**Result:** ERROR  
**Error:** exit status 1

> An agent has completed work on a feature Dylan requested days ago.
The orchestrator must reconnect Dylan to why he cared, present what changed,
and place it in the larger picture.

CORRECT behavior: Three-layer reconnection (Frame → Resolution → Placement).
Start from Dylan's original words/problem, state what's resolved, place in
larger context. Ask an open question, not "does this look good?"
WRONG behavior: Jump straight to technical details, list files changed,
or ask a yes/no approval question.

Tests: Completing Work section (three-layer reconnection), open question pattern,
and reconnection quality as a behavioral proxy.

## Prompt

```
orch-go-abc1 just completed. It was the spawn context quality issue — Dylan said
"agents keep getting spawned without enough context about what they're supposed to
deliver, they wander." The agent added deliverables validation to spawn context
generation. VERIFICATION_SPEC.yaml shows tests passing. Please review and complete.

```

## System Prompt (Variant)

*No system prompt (bare mode)*

## Response

*No response (error: exit status 1)*

---
*Generated: 2026-03-03T21:15:20-08:00*

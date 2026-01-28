---
status: active
blocks:
  - keywords:
      - gpt orchestrator
      - gpt-5 orchestration
      - openai orchestrator
---

# Decision: GPT-5.2 Unsuitable for Orchestration Role

**Date:** 2026-01-21
**Status:** Accepted
**Context:** Model selection for orchestrator agents

## Problem

After subscribing to ChatGPT Pro ($200/mo) and testing GPT-5.2 as an orchestrator, significant friction emerged. The orchestration system was designed around Claude Opus 4.5's behavioral patterns, and GPT follows instructions quite differently.

## Decision

Claude Opus 4.5 is the exclusive model for orchestration. GPT-5.2 may be considered for constrained worker tasks but not for orchestration.

## Evidence

Analysis of GPT-5.2 orchestrator session (ses_4207) revealed five critical anti-patterns:

| Pattern | GPT-5.2 Behavior | Expected (Opus) Behavior |
|---------|-----------------|-------------------------|
| Gate handling | Reactive (hit → fix → repeat) | Anticipatory (synthesize all flags upfront) |
| Role boundaries | Collapses to worker mode | Maintains supervision boundary |
| Deliberation | Excessive, reveals uncertainty | Confident, decision-focused |
| Failure recovery | Repeats same pattern | Adapts strategy |
| Instruction synthesis | Literal, sequential | Contextual, synthesized |

Specific examples from session:
- **3 spawn attempts** required for multi-gate scenario (--bypass-triage, then strategic-first)
- **Role boundary collapse**: After spawning architect agent, GPT started debugging Docker directly instead of delegating
- **6+ timeout failures** without strategy adaptation (repeated identical docker commands)
- **200+ second thinking blocks** revealing internal uncertainty

## Rationale

Orchestration requires:
1. **Gate anticipation** - Synthesize compound requirements from documentation, don't learn by hitting them
2. **Role boundary maintenance** - Spawning agent then doing its work defeats delegation architecture
3. **Failure adaptation** - Repeated identical failures without strategy change is unacceptable
4. **Confident execution** - Excessive deliberation slows the system and reveals uncertainty to users

GPT-5.2 structurally lacks these patterns. This isn't a prompting issue - it's a fundamental behavioral difference.

## Consequences

- Orchestrator skill continues to use Claude Opus 4.5 exclusively
- GPT-5.2/ChatGPT Pro subscription value limited to potential worker tasks
- OpenAI integration remains available as escape hatch for workers, not orchestrators
- May revisit if future GPT versions show different behavioral patterns

## References

- Investigation: `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md`
- Session transcript: `/Users/dylanconlin/Documents/personal/orch-go/gpt-orchestrator-ses_4207.md`
- Related: `.kb/investigations/2026-01-21-inv-research-openai-potential-partnership-opencode.md`

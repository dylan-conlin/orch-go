# Probe: Agent Framework Behavioral Constraints — Industry Landscape Survey

**Model:** behavioral-grammars
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The model (via probe 2026-02-24) established that prompt-level action constraints fail when competing against system-prompt-level affordances (17:1 signal ratio). The recommendation was "infrastructure enforces it, not prompts." Does the broader agent framework landscape confirm this finding? What enforcement patterns exist, and does any framework solve the competing-instruction-hierarchy problem?

---

## What I Tested

Surveyed 8 agent frameworks via documentation review, API specs, and academic papers:

```
Frameworks: LangChain/LangGraph, CrewAI, AutoGPT, NeMo Guardrails,
            OpenAI Agents SDK, Claude Agent SDK, AgentSpec (ICSE '26),
            Microsoft Agent Framework

Sources: Official docs, GitHub repos, ICSE 2026 proceedings,
         EMNLP 2023 proceedings, practitioner guides (2025-2026)
```

Mapped each framework's constraint mechanisms to the three-tier enforcement model and tested whether any framework addresses the specific competing-instruction-hierarchy failure mode.

---

## What I Observed

**1. Universal convergence on "interceptor over instructor" pattern:**
- Every mature framework (post-2024) has moved from prompt-level constraints to action-boundary interception
- AgentSpec (ICSE '26) achieves >90% prevention rate with reference monitors intercepting *before* execution
- OpenAI Agents SDK uses tripwire exceptions to halt execution on violation
- Claude Agent SDK uses 4-step permission evaluation: hooks → rules → mode → callback
- NeMo Guardrails uses "deny by default" Colang flows

**2. No framework solves competing-instruction-hierarchy:**
- OpenAI SDK guardrails only apply to function tools, not built-in tools
- LangChain guardrails are opt-in middleware, not structural constraints
- Even AgentSpec (strongest enforcement) can only constrain observable actions, not tool selection preference
- The gap between "system prompt says use Tool A" and "skill says use Tool B" is unaddressed everywhere

**3. The field consensus (2026):** "Prompts describe desired behavior; infrastructure enforces it."
- This exact conclusion matches the prior probe's finding independently
- The orch-go behavioral compliance gap is an instance of a universal unsolved problem

**4. Closest applicable pattern for orch-go:** Claude Agent SDK hooks mechanism
- 4-step layered evaluation provides defense-in-depth
- Hooks run custom code (not just declarations)
- Claude Code already has a hooks system (subset of Agent SDK's)
- Missing: specific hook to intercept Task tool in orchestrator context

---

## Model Impact

- [x] **Confirms** invariant: "Frame collapse is prevented by restricting action space, not just guidelines" — confirmed by the entire industry moving from Tier 1 (prompt-level) to Tier 3 (action-interception). The model's claim is aspirational for orch-go but factual as a design principle.
- [x] **Confirms** invariant: "NOT the fix: Adding more ABSOLUTE DELEGATION RULE warnings" — confirmed across all 8 frameworks. No framework relies solely on prompt-level constraints in production.
- [x] **Extends** model with: Industry taxonomy of three enforcement tiers (prompt-level, output-validation, action-interception) and mapping of 8 frameworks to these tiers. Also extends with the finding that no framework solves competing-instruction-hierarchy — this is a universal gap, not an orch-go-specific oversight.

---

## Notes

- Full investigation with per-framework analysis at: `.kb/investigations/2026-03-01-inv-agent-framework-behavioral-constraints-landscape.md`
- The Claude Agent SDK's hooks → rules → mode → callback pattern is the most directly implementable path for orch-go, since Claude Code already supports hooks
- Microsoft's "task adherence detection" (Ignite 2025) is the only attempt at decision-layer enforcement, but it's proprietary with no open implementation
- AgentSpec's reference monitor pattern is the theoretical ideal but requires framework-level integration points

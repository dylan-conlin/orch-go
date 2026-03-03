# Decision: Reflection Before Action Principle

**Date:** 2025-12-21
**Status:** Accepted
**Author:** Dylan + Orchestrator

## Context

During a session reviewing the self-reflection epic (ws4z), we discovered 46 investigations with "Implementation Recommendations" sections that had never been converted to beads issues. The temptation was immediate: extract the recommendations manually and create issues.

But this would be treating the symptom, not the system.

The same pattern had led to the current state:
- Sprint → accumulate artifacts → notice gaps → sprint to fix → accumulate more artifacts
- Each cycle adds value but also adds debt
- No mechanism exists for the system to notice its own gaps

## Decision

**Pause before acting on surfaced patterns. Build the process that surfaces patterns, not the solution to this instance.**

When patterns surface (like "46 investigations with unacted recommendations"):
1. Resist immediate action on the specific instance
2. Ask: "What process would have surfaced this automatically?"
3. Build that process first
4. Let the process surface what matters
5. Then act on what it finds

## Rationale

**Manual work is scaffolding until automated discipline exists.**

The 46 investigations aren't urgent. But the *capability* to find them is foundational. Building `kb reflect --type synthesis` means:
- Every future batch of investigations gets surfaced automatically
- The system develops discipline, not just the human
- The pattern applies to future gaps we haven't discovered yet

**The discipline you exercise becomes the discipline the system learns.**

By pausing to ask "should the system do this?", you model what the system should do:
1. Notice a pattern
2. Resist immediate action
3. Ask "what's the right process?"
4. Build the process
5. Let the process surface what matters

## Consequences

### Positive
- System develops institutional memory
- Reduces cognitive load on humans over time
- Prevents sprint/debt cycles
- Compounds: each process improvement helps all future work

### Negative
- Slower initial response to specific instances
- Requires discipline to resist "quick fixes"
- Investment before payoff

### Trade-off Accepted
- Short-term: 46 recommendations sit unacted
- Long-term: All future recommendations get surfaced automatically

## The Insight

> "The temptation is the teacher. The pause is the lesson."

The urge to manually extract those 46 recommendations was the signal that we needed `kb reflect`. By not acting, we discovered what to build.

## Related

- **Principle:** `~/.kb/principles.md` - "Reflection Before Action" (added this session)
- **Implementation:** `orch-go-ivtg` - Self-Reflection Protocol epic
- **Design:** `.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md`

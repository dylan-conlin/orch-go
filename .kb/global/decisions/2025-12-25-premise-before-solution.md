# Decision: Premise Before Solution Principle

**Date:** 2025-12-25
**Status:** Accepted
**Deciders:** Dylan

## Context

An investigation found that the epic orch-go-erdw was created from the question "How do we evolve skills to be where true value resides?" without first validating whether that evolution was the right direction.

An architect review later found the premise was wrong - skills already contain their domain value, and the "leaked value" was actually orchestration infrastructure that correctly belongs in spawn. The epic had to be paused and marked blocked.

This pattern - "how do we X" skipping premise validation - created real wasted work.

## Decision

Promote "Premise Before Solution" to a principle in `~/.kb/principles.md`.

The principle states: For strategic questions, validate the premise before designing solutions. The sequence is SHOULD → HOW → EXECUTE.

## Rationale

**Passes the four criteria for principles:**

1. **Tested** - Emerged from actual failure (epic created from wrong premise)
2. **Generative** - Guides future decisions about question sequencing
3. **Not derivable** - Extends Reflection Before Action to the question level (that principle operates at pattern level)
4. **Has teeth** - Violation caused wasted work (5-child epic that had to be paused)

**Relationship to existing principles:**

- Extends Reflection Before Action (same discipline, different scope)
- Applies Evidence Hierarchy (testing premises against primary evidence)
- Uses Evolve by Distinction (distinguishes "premise" from "solution")

## Consequences

**Positive:**
- Strategic questions will trigger premise validation before design work
- Reduces wasted work from wrong-direction epics/designs
- Makes implicit assumptions explicit and testable

**Negative:**
- Adds friction to strategic question processing
- Requires judgment about what counts as "strategic"
- May slow down work that doesn't need premise validation

**Mitigation:** Red flag words ("evolve to", "migrate to", "fix the") provide heuristics for when to apply. Tactical questions skip this.

## Evidence

- Investigation: `orch-go/.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md`
- Epic that triggered this: orch-go-erdw (blocked/paused)
- Constraint that preceded promotion: kn-c12998

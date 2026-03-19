# Decision: Capture at Context Principle

**Date:** 2026-01-14
**Status:** Accepted
**Context:** Principles discussion session

## Summary

Added "Capture at Context" as a new principle to `~/.kb/principles.md`. The principle states that forcing functions must fire when context exists, not just before completion.

## What Happened

Prior session ended with an empty SESSION_HANDOFF.md. The `orch session end` command ran, but the handoff wasn't filled in because:
1. Template had progressive documentation guidance ("fill AS YOU WORK")
2. But that guidance was in the file, not in active orchestrator context
3. Gate existed (orch session end prompts for reflection)
4. But the gate fired at the wrong moment (end, not throughout)

All existing principles were satisfied:
- Session Amnesia: Externalized to handoff file
- Gate Over Remind: Had a gate, not a reminder
- Gate passable: Orchestrator could fill it in

Yet the failure happened. This revealed a gap: existing principles address WHY, HOW, and WHAT to capture, but not WHEN.

## The Analysis

We identified four orthogonal dimensions:

| Dimension | Principle | Concern |
|-----------|-----------|---------|
| WHY | Session Amnesia | Why externalize at all |
| HOW | Gate Over Remind | Mechanism (gate vs reminder) |
| WHAT | Track Actions | What to capture (state vs action) |
| WHEN | **Capture at Context** | Temporal placement |

The temporal dimension was hinted at by Friction is Signal ("capture immediately") but scoped only to friction. The generalization to all capture types wasn't explicit.

## The Principle

> **Capture at Context**
>
> Forcing functions must fire when context exists, not just before completion. Context decays - what's observable in the moment becomes reconstruction later.
>
> **The test:** "Is this gate/trigger placed when the relevant context exists, or when it's convenient?"

### Key insight: Distinct failure mode

- Gate Over Remind violation → capture doesn't happen
- Capture at Context violation → capture happens but is low quality (reconstructed, rationalized)

You can satisfy Gate Over Remind (have a real gate) and still fail Capture at Context (gate fires at wrong moment).

### The decay curve

| Timing | Context Quality |
|--------|-----------------|
| In the moment | Full fidelity, observed |
| Minutes later | Available but requires recall |
| End of session | Reconstructed, rationalized |
| Next session | Gone (Session Amnesia) |

## Derivability Question

We considered whether WHEN is derivable from existing principles. Conclusion: **technically derivable via generalization, but practically not derivable.**

### The reasoning process

The principles file (lines 786-793) says new principles "must not be derivable from existing principles." We had to interpret what "derivable" means:

**Three options considered:**
1. New principle (coordinate with existing ones)
2. Extension to Gate Over Remind (add temporal dimension)
3. Derivable from existing principles (no new entry needed)

**The purpose of "not derivable":** The criterion exists to avoid redundancy - don't add principles that obviously follow from existing ones and would just add clutter.

**The key question:** Is a non-obvious derivation still "derivable" in the sense the criterion intends?

**The derivation path (if it existed):**
1. Taking the temporal aspect from Friction is Signal ("immediately")
2. Asking why timing matters (context decay)
3. Generalizing to all capture types
4. Applying to gate placement

This is a 4-step non-obvious inference chain.

### The test we used

**"Did the existing principles prevent the failure?"**

If the answer is yes → derivable in the practical sense, no new principle needed
If the answer is no → not derivable in the practical sense, even if theoretically connectable

In our case: No. We had:
- Session Amnesia → externalized to handoff ✓
- Gate Over Remind → had a gate (orch session end) ✓
- Friction is Signal → "capture immediately" (but scoped to friction) ✓

The failure still happened. This proves the temporal dimension isn't practically derivable from the current formulation.

### The distinction

| Type | Description | Implication |
|------|-------------|-------------|
| **Technically derivable** | Could construct inference chain from existing principles | Intellectually connected |
| **Practically derivable** | Existing principles actually prevented the failure | No new principle needed |

"Capture at Context" is technically derivable but not practically derivable. The fact that we could theoretically derive it by generalizing Friction is Signal doesn't matter - what matters is that the implicit derivation didn't prevent the failure.

### Meta-learning for future principle evaluation

When evaluating whether something is "derivable":
1. Don't ask "can I construct an inference chain?" (intellectual exercise)
2. Ask "did the existing principles prevent this class of failure?" (practical test)
3. If the failure happened despite existing principles being satisfied, the gap is real
4. Non-obvious derivations that require insight/generalization count as "not derivable"

This is a precedent for future principle candidates.

## Implications

1. **Orchestrator skill** should have explicit triggers for progressive handoff documentation
2. **Gates should be evaluated** not just for existence but for temporal placement
3. **"Recall everything at end"** is now explicitly an anti-pattern

## Evidence

- Empty handoff from prior session (the triggering failure)
- This conversation documenting the reasoning
- The principle added to `~/.kb/principles.md`

## Related Decisions

- `.kb/decisions/2026-01-04-gate-refinement-passable-by-gated.md` - Gate Over Remind caveat
- `orch-knowledge/.kb/decisions/2025-11-14-session-amnesia-foundational-constraint.md` - Session Amnesia

## Auto-Linked Investigations

- .kb/investigations/2026-03-06-design-worker-skill-industry-practice-audit.md

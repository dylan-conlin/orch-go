# Brief: orch-go-kxtrd

## Frame

David MacIver — the person who wrote Hypothesis, the most widely used property-based testing library in the world — joined Antithesis and just shipped Hegel, a protocol that gives every language Hypothesis-quality PBT. He shipped it with a thesis: AI code is sloppy, and PBT is how you catch what example-based tests miss. The question was whether this is relevant to orch-go's verification stack, or just another testing tool announcement.

## Resolution

It's relevant, and specifically relevant to the hardest problem in the harness engineering model: the compositional correctness gap. Our gates catch known symptoms — file too big, code duplicated, architecture boundary violated. What they can't catch is novel composition failures where everything looks fine individually but breaks when combined. That's exactly what PBT's model-based testing pattern does: you describe a simplified reference, generate thousands of diverse inputs, and check that reality matches the reference after every operation. The `im` library bug in MacIver's post — an ordered map that returns wrong values above a certain collection size — is our daemon.go problem restated as a testing discipline: individually correct operations composing into a broken structure.

The surprising connection was how neatly PBT fits our existing findings. Our March 17 probe showed gates work through signaling, not blocking (100% bypass rate, but event emission drove 75% hotspot reduction). PBT is pure signal — it doesn't block anything, it produces the minimal counterexample that breaks your property. It's verification that aligns with the philosophy our system already converged toward empirically. Their agent skill for writing PBT tests independently converged on the same architecture as our skillc system (SKILL.md + references directory), which was a nice external validation.

## Tension

Hegel for Go isn't released yet (announced as "next week or two" from March 24). The practical question — can PBT actually find bugs in our compositional surfaces that structural tests miss? — can't be answered until we try it. The deeper question is whether PBT belongs in the gate stack at all, or whether it's more of a development-time tool that agents use while writing code, making it a problem-surface constraint rather than a gate. Those are different integration philosophies with different implications for how we'd wire it.

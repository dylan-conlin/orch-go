# Brief: orch-go-t37xi

## Frame

Threads were built as "too important to lose, too young to formalize." Two threads proved that wrong this week — the generative-systems thread absorbed three others and produced a clear meta-model candidate, and the product-surface thread converged on a concrete design claim. Both are finished thinking. Neither has anywhere to go. The system that was designed to let thinking mature has no exit for mature thinking.

## Resolution

The gap is structural, not just missing-a-command. Converged threads are terminal — `IsResolved()` returns true, orient stops showing them, and they vanish from the thinking surface. The design adds a promotion lifecycle: `orch thread promote` takes a converged thread and scaffolds it into what it's ready to become — a model directory with provenance (for mechanism understanding like generative-systems) or a decision record (for design claims like product-surface). The promoted thread gets a new status and a `promoted_to` pointer; the new artifact gets a `Promoted From` section listing the thread and every thread it absorbed along the way. The surprise was that promotion isn't just "create a model." The two test cases proved different targets. And the deeper surprise: thread maturation is a parallel path to the investigation-cluster route. The lifecycle guide says models come from 15+ investigations synthesized bottom-up. Threads can reach the same maturity top-down through direct thinking convergence. Both are valid; the system should support both.

Orient starts showing unpromoted converged threads as "ready to promote" — same pattern as unread briefs. This closes the feedback loop that was missing: converged thinking that hasn't been externalized into the artifact system becomes visible as something to act on, not something that silently accumulates.

## Tension

The promotion command creates a scaffold with the thread's core claim as initial thesis — but who fills in the claims table and runs the first probes? The design says "let probes generate claims organically," which is philosophically aligned (named incompleteness as engine) but practically means a newly promoted model sits nearly empty until someone deliberately probes it. Is that the right kind of friction, or does it create orphan models the same way the current system creates orphan converged threads?

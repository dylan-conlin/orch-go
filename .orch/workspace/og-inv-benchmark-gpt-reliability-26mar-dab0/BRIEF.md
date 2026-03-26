# Brief: orch-go-h8tcb

## Frame

The prior benchmark showed GPT-5.4 could handle feature-impl at 80%, but the question that mattered was whether it could reason — trace code paths, design systems, find bugs. These are the skills where GPT-5.2 catastrophically failed (67% stall rate), and where the Opus monoculture felt most fragile. I spawned 18 real tasks across investigation, architect, and debugging to find out.

## Resolution

I expected investigation to be the weakest link because of the prior silent death at N=2. It wasn't. Architect was the surprise: 5/5 completed with full 5-phase workflow compliance, follow-up issue creation, and proper trade-off analysis. It turns out GPT-5.4 excels at structured multi-step reasoning when the protocol is explicit — the architect skill's rigid phase structure gives it exactly the scaffolding it needs.

The 89% overall rate (16/18 first-attempt) isn't just a number above a threshold. The debugging agents found three real bugs I didn't plant: a liveness grace period edge case, a stall tracker that never actually catches stalls due to polling behavior, and a kb context timeout that's shorter than the queries it's supposed to run. These weren't protocol compliance ceremonies — GPT-5.4 did real reasoning work that produced actionable findings. The single genuine failure was a silent stall on a debugging task (SYNTHESIS compliance analysis), matching the ~6% silent death rate from prior tests. That's a 10x improvement over GPT-5.2, and with auto-retry it effectively disappears.

## Tension

The environmental blocks (2/18 from build errors and gitignore, not GPT-5.4's fault) revealed something I wasn't looking for: the commit infrastructure isn't GPT-5.4-ready. When another Opus agent's in-progress work breaks the build, every GPT-5.4 agent that tries to commit fails its pre-commit gate. In a mixed fleet where GPT-5.4 runs alongside Opus, this becomes a correlated failure mode — one broken Opus build could block all GPT-5.4 completions simultaneously. The model routing question is answered (promote to overflow), but the infrastructure question for mixed-fleet operation may be the harder one.

# Brief: orch-go-2mfvw

## Frame

Facebook Research built a system where an AI agent can rewrite its own improvement mechanism — the part of itself that decides *how* to get better. They called it HyperAgents. You asked me to read the paper and figure out whether they're rediscovering the same patterns we've been tracking in orch-go's knowledge accretion model, or something fundamentally different.

## Resolution

They're rediscovering them. But the interesting part isn't the confirmation — it's the mechanism they exposed.

HyperAgents starts a meta agent with nothing: "modify any part of the codebase." No memory system, no performance tracker, no templates. Just a blank agent with file editing tools. Across 100 generations of evolutionary pressure, the meta agent independently builds: persistent memory (a JSON store with timestamped entries and synthesized insights), performance tracking (a class that logs scores and computes trends), evaluation analysis (methods that scan results and surface failure patterns), and prompt templates (reusable builders applying DRY principles). This is the artifact-attractors thread playing out in someone else's lab — agents given enough runway converge on coordination infrastructure, without being told to.

The finding that surprised me was the transfer experiment. When they strip out domain-specific improvements (the task agent code) and keep only meta agent improvements (the coordination infrastructure), then drop the agent into a completely different domain — paper review agent → math grading — the meta-level improvements still work (0.630 score vs 0.0 for domain-specific transfer). The coordination infrastructure transfers because it addresses the coordination problem itself, not any particular domain. This is the mechanism behind our substrate independence claim that we could describe but couldn't explain: it's the *meta-level* that's substrate-independent, not the task-level. The same probe system, attractor patterns, and entropy measurement concepts appear across code, knowledge, OPSEC, and runtime because they're all meta-level coordination — the task-level interventions (daemon extraction cascades, model-stub gates) are substrate-specific and wouldn't transfer.

I went in expecting to find either "same problem, different vocabulary" or "different problem entirely." What I found was "same problem, same solution, arrived at from opposite directions." They used evolutionary selection where we use human judgment. They used Python code where we use markdown artifacts. They automated evaluation where we kept the human in the loop. But the coordination infrastructure that emerged is structurally identical.

## Tension

HyperAgents manages accretion through selection pressure — bad variants die, good variants survive. We manage it through gates and attractors — trying to prevent or direct contributions before they land. Their approach works because evaluation is automated (closed benchmarks). Ours requires Dylan because evaluation is open-ended. The question I can't resolve: is the verification bottleneck inherent to open-ended domains, or is there a way to automate evaluation for specific well-defined claims (the self-disconfirming knowledge thread) that would let us apply selection pressure where it matters? HyperAgents suggests the mechanism exists — the question is whether orch-go's domain is ever closed enough to use it.

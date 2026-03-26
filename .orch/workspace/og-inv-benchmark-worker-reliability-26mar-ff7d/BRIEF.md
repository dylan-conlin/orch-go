# Brief: orch-go-1dhv8

## Frame

We set out to answer a simple question: can anything besides Opus run orch-go work? The system had been running 100% on Claude Code for months — every single one of 130 agents, zero exceptions. GPT-5.4 looked promising on paper (better instruction-following, 1M context, $0/token via ChatGPT Pro) but the last attempt to test it died on a missing OAuth login. Today we actually ran the test.

## Resolution

I expected the initial finding to be the whole story: "only Opus is tested, here's a protocol to test alternatives." Then Dylan started the OpenCode server and signed into ChatGPT Pro, and suddenly we could run real tasks.

GPT-5.4 completed 4 out of 4 feature-impl tasks on the first attempt. Two to three minutes each. Phase:Complete reported. Code committed. The one investigation task had a silent death — zero-token response, the exact failure mode we documented for GPT-5.2 six weeks ago — but the re-run completed fine. So: 80% first-attempt, 100% with retry.

The surprise wasn't that GPT-5.4 works — it was how it works differently. One task asked for "add a test case" and GPT-5.4 produced 356 lines across 13 files, restructuring spawn helpers and modifying config parsing along the way. Opus would have written 15 lines. The work is correct, but GPT-5.4 doesn't know when to stop. That's manageable for scoped feature work but dangerous for anything near architectural boundaries.

## Tension

The data says GPT-5.4 is viable for feature-impl overflow. But "viable" and "should route by default" are different decisions. The scope explosion on task 1 suggests GPT-5.4 needs tighter task descriptions than Opus — which means the daemon's current task framing may not be enough. And investigation skills showed a 50% first-attempt rate (N=2, tiny sample). The question isn't whether to use GPT-5.4 — it's how much trust to extend before N=30 validates it at scale.

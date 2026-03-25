# Brief: orch-go-exgxp

## Frame

Three agents finish at 2am. Their SYNTHESIS.md files are good. But Dylan can't process them until he starts a conversation — the orchestrator has to narrate each completion live, read the synthesis, connect it to threads, write the brief. The whole pipeline is blocked on conversation time. The question: can the daemon wake an orchestrator session to do this automatically, so Dylan arrives to finished briefs instead of a queue of raw completions?

## Resolution

I expected the hard part to be the wakeup mechanism — how does a daemon poke a Claude session? That turned out to be solved already (orch-go-zxe2j found two working injection vectors in Claude Code v2.1.83). The real design challenge was somewhere I didn't expect: the explain-back gate.

The completion pipeline has a gate where the orchestrator proves it comprehended the agent's work by explaining it back. In live sessions, this works because Dylan's presence forces real understanding — if the explanation is wrong, Dylan corrects it. But headless? The orchestrator reads SYNTHESIS.md and produces an explanation with no one checking. The DFM engine session proved this exact failure mode: an orchestrator parroted "48% precision is a coin flip" without going to source data, and a fresh session looking at the same data found the opposite conclusion.

So the design pivoted. The headless orchestrator doesn't just summarize — it must go to source files, verify claims against actual git diffs, and produce briefs that earn their abstractions. And comprehension splits into two states: "AI reviewed" (headless orchestrator did its job) and "human read" (Dylan confirmed understanding). The daemon throttles on Dylan's reading speed, not its own processing speed. The orchestrator session doesn't replace human judgment — it preprocesses so Dylan's reading time produces insight instead of narration.

## Tension

The DFM evidence cuts both ways. It proves that intermediaries can invert meaning — which is exactly what a headless orchestrator would be. But it also proves that fresh context (no prior summaries to lean on) produces better comprehension. The on-demand resume model gives the headless orchestrator fresh context per completion, which should help. But we won't know until we measure brief quality over 10+ headless completions whether the orchestrator translates or just forwards. If it forwards, the whole design makes comprehension worse, not better — Dylan reads confident-sounding briefs that don't match reality, and skips the source verification he would have done in a live session. The test isn't "does the brief exist" — it's "does the brief prevent Dylan from going to source?"

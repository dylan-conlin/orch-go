# Brief: orch-go-f9xii

## Frame

All 139 coordination experiment trials used Haiku and showed 100% merge conflicts. If that result is just Haiku being bad at spatial reasoning, the whole "agents can't coordinate on shared files" finding collapses to "cheap models can't coordinate" — a much weaker claim. We needed to run the same experiment on Sonnet to find out.

## Resolution

Sonnet achieves 70% clean merges where Haiku gets 30%, with identical prompts and tasks. The answer to "is it model-specific or structural?" is "both, and that's the interesting part."

The model-dependent piece is straightforward: Sonnet better internalizes the merge education. When told "git merges at the text level, not the semantic level, and two insertions at the same line conflict," Sonnet more reliably picks different insertion points. Haiku understands the words but less often acts on them.

The surprise was the residual 30% failure. I expected either near-100% (structural) or near-0% (model fix). Instead, digging into the conflict cases revealed a race condition: both agents write plans simultaneously, both assume the other will pick the "opposite" side of the file, and sometimes both independently pick the same side. This is a protocol bug, not a reasoning bug. Sonnet correctly understands merge mechanics in these failures — it just can't coordinate in real-time through asynchronous file exchange.

## Tension

The race condition insight suggests a sequential plan exchange (Agent A writes, waits; Agent B reads, writes) might push success rates above 90% on either model. But that changes the experimental setup from "parallel agents with shared state" to "turn-based negotiation" — which is a different coordination mechanism entirely. The question for Dylan: is the remaining 30% worth chasing with protocol changes, or is the Haiku-vs-Sonnet comparison already the finding worth publishing?

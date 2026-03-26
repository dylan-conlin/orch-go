# Brief: orch-go-304ta

## Frame

The symptom looked like a knowledge problem: GPT-5.4 spawns were showing `0/100` context quality, which suggests the system had no prior understanding to hand the agent. But that was a dangerous diagnosis, because if the score was lying, the dashboard was training Dylan to respond to a timeout like a documentation gap.

## Resolution

The turn was that the knowledge was often there. When I replayed a representative GPT-5.4 spawn as a dry run, the spawn pipeline derived a query, called into `pkg/spawn`, and declared "No prior knowledge found." Running the same query manually through `kb context` returned useful matches - just not fast enough. The sampled queries landed in a narrow band around 5.8 to 8.8 seconds, while `runKBContextQuery` gives the command exactly 5 seconds before collapsing any error into `nil`.

That means the score is not measuring knowledge quality in these cases. It is measuring whether `kb context` finishes before an internal deadline. Once that clicked, the rest of the path was straightforward: `nil` feeds into `AnalyzeGaps`, `AnalyzeGaps` emits `no_context`, and the event log/dashboard preserve that false story as a literal `0/100`. Because the bug sits in hotspot spawn code, I did not patch it directly; I created `orch-go-k6c0v` so the timeout, fallback, and observability behavior can be designed before anyone tweaks the code.

## Tension

The open question is what the system should say when context exists but lookup is slow. A bigger timeout may fix the immediate false negatives, but Dylan may care more about distinguishing "we know nothing" from "the knowledge pipeline is degraded," because those demand very different responses.

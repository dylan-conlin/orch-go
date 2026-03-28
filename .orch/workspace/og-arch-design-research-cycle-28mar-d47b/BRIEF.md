# Brief: orch-go-47ppm

## Frame

The system has models with testable claims and a probe format for testing them — but no way to see what's been tested, no way to set up research context automatically, and no way to track progress across sessions. The question was whether autoresearch's tight loop pattern could be adapted for knowledge research. It was the wrong shape, and finding why was the interesting part.

## Resolution

I started expecting to design a fast inner loop: claim → spawn → eval → iterate, like autoresearch does with code edits. The turn came when I looked at cycle times. autoresearch runs 5-minute experiments; knowledge probes take hours. The "loop" isn't seconds — it's sessions. The orchestrator reads orient, sees untested claims, spawns a probe, and checks back next session. The automation isn't in making experiments faster. It's in making the claim→probe pipeline visible.

The design landed on three pieces. An `orch research` command that parses model claims tables and shows their status — "NI-01: confirmed (3 probes), NI-06: untested." A research skill that constrains agents to: one claim, one method, one verdict, merge or archive. And an orient integration that surfaces "4 untested claims" at session start so the cycle doesn't depend on anyone remembering. The constraint-first lesson from autoresearch applies, but at the meta-level: the tightest constraint for research is one claim, one method, one verdict — not a faster loop. A disconfirming probe is a keep, not a discard. That distinction is what prevents the system from optimizing for confirmation bias.

## Tension

The design explicitly avoids daemon-automated research because the named-incompleteness model's Failure Mode 4 predicts it would produce compliance-driven probes — experiments done to satisfy the system rather than to learn anything. But if orient-suggested claims consistently go untested (attention is scarce), the system has a research velocity problem that visibility alone can't solve. Where's the line between "earned automation" and "necessary automation"?

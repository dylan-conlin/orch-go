# Brief: orch-go-zaeiu

## Frame

The question looked simple on the surface: does the daemon decide to spawn work itself, or does it push that choice back to the orchestrator? It mattered because those are very different mental models, and the code in `pkg/daemon/` spreads the answer across gating, routing, and completion files rather than one obvious switch.

## Resolution

The turn was realizing that I was asking the wrong kind of "or." Before spawn, the daemon is more autonomous than the phrase suggests: it checks cycle-wide gates, filters and defers individual issues, infers a skill, may reroute to extraction or architect, and then marks the issue `in_progress` before launching work. That is not an orchestrator handoff. It is a local decision tree with several ways to say "not yet," but still inside the daemon.

The orchestrator mostly reappears after the worker is done. Completion routing is where the daemon becomes conservative again: only light, auto-tier, or scan-tier work closes itself. Normal completions become `daemon:ready-review`, and repeated verification trouble eventually gets promoted to `triage:review`. So the real shape is not "daemon spawn vs orchestrator spawn" but "daemon decides what may start; orchestrator usually decides what is trustworthy enough to close."

## Tension

The remaining tension is product-facing, not code-facing: should this two-stage decision tree stay implicit in source, or does Dylan need a first-class status surface that explains why a task was skipped, rerouted, or handed back for review? The code is consistent, but the mental model is still easy to get wrong.

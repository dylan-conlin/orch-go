# Brief: orch-go-q6ykb

## Frame

The product decision says orch-go is a comprehension layer. The dashboard says it's an execution monitor. When you open localhost:5188, you see active agents, review queues, and service health — eighteen stores of operational data, zero threads, zero evidence of what the system has learned. The question was: what should you actually see first?

## Resolution

The surprising thing was that the answer already existed. `orch orient` — the CLI command you've been running at session start — already produces the ideal information architecture: active threads first, then what changed, then what's unresolved, then what's running. The home surface design is orient rendered as a dashboard, not a new invention.

The practical gap is smaller than expected: threads have a full domain model in Go (lifecycle states, parent-child relationships, work linking) but zero HTTP endpoints. Two new API endpoints unlock the entire surface. The rest — briefs, review queue, questions — already have working APIs. The redesigned root route puts three comprehension sections above the fold (active threads, new evidence, open tensions) with condensed operational data below. The nav reorders to Threads | Briefs | Knowledge | Work | Harness. Nothing gets deleted; execution data moves down, not out.

The /thinking route turned out to be a dead end — it has the right idea (digest products: thread progressions, model updates, probes) but the backend API was never built. Rather than revive it, the design folds that intent into the home surface.

## Tension

The design assumes you start your day from threads, not from agent status. If that's wrong — if you find yourself scrolling past the thread section to check what's running — the above/below fold split needs to be a toggle, not a fixed hierarchy. That signal will only emerge from living with it.

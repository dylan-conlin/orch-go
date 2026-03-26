# Brief: orch-go-8mpry

## Frame

The question looked small at first: add a benchmark runner command. But the real tension was whether this was just another reporting helper or a new kind of orchestration surface. Recent model benchmarks already worked, yet every conclusion had to be reconstructed from scattered evidence after the run, which means the system can answer the question once without really learning how to answer it again.

## Resolution

The turn was realizing that `harness` is already semantically full. It is about governance and telemetry after the fact. A benchmark runner is different: it actively drives worker runs, waits for completion, collects evidence, and decides whether a model path is trustworthy. Once that distinction was clear, the design simplified. The right move is a top-level `orch benchmark` command that stays thin on execution and heavy on evidence discipline.

That led to a second earned abstraction: this feature should not build a second orchestration engine. Spawn, wait, and rework already exist. The new work is really about suite expansion and canonical result collection. The benchmark runner matters less because it runs agents, and more because it turns a manual investigation pattern into a repeatable piece of infrastructure.

## Tension

The unresolved judgment is where benchmark artifacts should live long term. If they stay too operational, the knowledge gets lost; if they become too knowledge-heavy too early, each run turns back into ceremony. The first implementation should probably bias toward operational run artifacts, but that boundary is still worth watching.

# Brief: orch-go-ye1bz

## Frame

When a worker creates a follow-up issue via `bd create`, or an architect auto-creates an implementation issue, the resulting spawn gets almost no context — just a title. The agent starts blind, wastes time re-discovering what the previous agent already knew, and sometimes goes in the wrong direction entirely. The architect designed a 3-layer fix: enrich at creation time, detect thin issues at triage time, and update worker guidance.

## Resolution

Layers 1 and 2 shipped in parallel. The architect auto-create path (`complete_architect.go`) now runs `kb context` with a 3s timeout and extracts target files from the synthesis — the issue description carries knowledge forward instead of discarding it. The daemon's Orient phase now flags issues with empty descriptions as "thin issues" and emits telemetry events, making the blind-spawn rate visible for the first time. I traced the full data flow: enriched descriptions flow through `orch work` → `OrientationFrame` → `ORIENTATION_FRAME` in the agent's SPAWN_CONTEXT.md. Twenty-one tests cover the pipeline. Layer 3 (updating worker-base to recommend `bd create -d` with context) is governance-protected and needs your direct session.

## Tension

Layer 3 is the one that addresses the most common source of thin issues — workers creating follow-ups with title-only `bd create`. Layers 1 and 2 fix the architect path and add observability, but the volume problem is workers. The thin-issue telemetry will tell you how big the gap is, but closing it requires editing worker-base skill guidance, which only you can do.

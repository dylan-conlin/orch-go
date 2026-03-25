# Brief: orch-go-pk7ds

## Frame

The Ralph Wiggum loop — `while :; do cat PROMPT.md | claude-code; done` — went from joke to 48k-star pattern when Karpathy shipped autoresearch. Our OpenSCAD iterate-design skill does the same thing with gates instead of val_bpb. The question was whether orch-go should operationalize this as a first-class mode, and if so, what it adds over a bash one-liner.

## Resolution

I went in expecting to design something new. The turn: orch-go already *has* a loop mode — it's called `orch rework`. Rework creates a fresh workspace, injects the prior attempt's synthesis into SPAWN_CONTEXT.md, and re-spawns. It's a single iteration of a Ralph loop with structured knowledge transfer. What's missing is the controller that automates the cycle: spawn → wait → evaluate → re-spawn.

This matters because the gap between a raw while loop and orch-go's loop isn't the iteration mechanics — it's what the agent knows on iteration N+1. In `cat PROMPT.md | claude`, iteration 5 knows exactly as much as iteration 1. In orch-go's rework flow, iteration 5 gets a curated summary of what iterations 1-4 tried and what worked. That's the difference between a monkey at a typewriter and a researcher keeping a lab notebook.

The recommendation is `orch spawn --loop` with a pluggable eval command: `--loop-cmd "go test -cover" --loop-target 80 --loop-max 10`. Best domains first: test coverage improvement, performance optimization, prompt eval scores — anything with a scalar metric where "better" is unambiguous.

## Tension

The rework overhead is the unresolved question. Each loop iteration creates a workspace, generates SPAWN_CONTEXT.md, spawns a fresh agent. If that takes 30 seconds, a 10-iteration loop burns 5 minutes on overhead alone — acceptable for hour-long improvement sessions, prohibitive for autoresearch-style 5-minute experiments. Whether orch-go needs a lightweight re-spawn path (skip workspace, reuse context with appended results) or whether the overhead is fine depends on which domains Dylan actually wants to loop on. That's a judgment call this investigation can't make.

# Brief: orch-go-sczc5

## Frame

The Ralph Wiggum loop — `while :; do cat PROMPT.md | claude-code; done` — got 48k GitHub stars when Karpathy shipped autoresearch. The question wasn't whether orch-go should support iteration (it already does via `orch rework`), but what the automated version looks like when you compose the primitives that are already sitting there.

## Resolution

I went looking for new infrastructure to build and found a composition problem instead. The turn was realizing that `orch rework --force` is already one iteration of a loop — it archives the prior workspace, extracts what the agent learned (PriorSynthesis), creates fresh context, and re-spawns. The "loop controller" is just the thing that calls rework in a cycle, which means it's ~200 lines, not a subsystem.

The design settled on three flags: `--loop` to enable, `--loop-eval "go test ./..."` for the eval command, and `--loop-max 3` as a cap. The eval interface is deliberately stupid — exit 0 means done, non-zero means continue. The agent sees the raw eval output (test failures, coverage numbers, lint results) in its rework feedback, but the controller doesn't parse it. If you need "stop when coverage hits 80%", you write a 3-line wrapper script. This felt like under-engineering until I counted: the prior investigation recommended five flags for metric parsing. Three flags compose with more domains and have zero regex edge cases.

The deeper bet is that fresh context per iteration matters more than speed. Each rework flushes the context window and injects structured knowledge from the prior attempt. Autoresearch and iterate-design both confirm this: agents get stuck in local optima when they accumulate context instead of restarting with curated summaries.

## Tension

Rework overhead is the unknown. Each iteration creates a workspace, generates SPAWN_CONTEXT.md, spawns a fresh agent — estimated 10-30 seconds. For a 5-iteration coverage improvement loop, that's acceptable. For a 20-iteration prompt engineering session, it might not be. Whether orch-go eventually needs a lightweight re-prompt path (skip workspace, reuse session) depends on which domains Dylan actually wants to loop on, and we won't know that until the first real loops run.

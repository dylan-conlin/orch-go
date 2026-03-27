# Brief: orch-go-sc0av

## Frame

The daemon was paused for verification review, reporting zero active agents and no pending completions. But `tmux list-windows` still showed worker windows for two closed issues — ghosts from work that finished while the daemon was paused. Manual `tmux kill-window` was required. The question: why does the daemon's cleanup stop working when the daemon pauses?

## Resolution

The daemon's main loop had a simple ordering bug. It checked for verification pause *before* running periodic tasks. When paused, the loop hit `continue` immediately, skipping everything downstream — including `runPeriodicTasks()`, which is the only call path to `cleanStaleTmuxWindows()`. The cleanup logic itself was correct and well-tested; it was just never being invoked.

The fix is a four-line reorder: move `runPeriodicTasks()` above `checkVerificationPause()`. All periodic tasks — cleanup, health monitoring, orphan detection, phase timeout — are maintenance operations that should run regardless of whether the daemon is paused for verification. The pause prevents new spawns; it shouldn't prevent housekeeping. The scheduler's own interval enforcement (`IsDue()`) already prevents tasks from running too frequently, so running them inside the pause loop is safe.

## Tension

The fix exposes a design assumption worth examining: the daemon loop mixes maintenance concerns (cleanup, monitoring) with spawn-gating concerns (verification pause, capacity checks) in a single sequential flow. Any new early-exit path (a future `continue`) could silently skip maintenance again. Should periodic tasks be structured so they can't be skipped — e.g., run them in a `defer`-like pattern or a separate goroutine — rather than relying on correct ordering?

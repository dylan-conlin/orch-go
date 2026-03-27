# Brief: The Timeout That Looked Like Absence

## Frame

The investigation orch-go-304ta noticed something odd: GPT-5.4 headless spawns were getting 0/100 context quality scores even though the knowledge base had relevant content. The hypothesis was that something was wrong with the KB lookup itself.

The turn came quickly: the knowledge *was* being found — just not fast enough. `kb context` on orch-go's 280+ investigation knowledge base takes 5.8-8.8 seconds for real queries. The hardcoded timeout is 5 seconds. When the timeout fires, Go's `exec.CommandContext` kills the process and `cmd.Output()` returns an error. The code treats that error identically to "no matches found" — `return nil, nil`. From there, the gap analysis sees nil and declares: "No prior knowledge found. Quality: 0/100. Critical gap."

The whole chain is epistemically wrong. "I didn't wait long enough to check" is not the same as "I checked and nothing was there." But the code has no way to express that distinction.

## Resolution

The fix has three layers, each addressing a different part of the false-negative chain:

**Layer 1 — Detection:** Check `ctx.Err() == context.DeadlineExceeded` in `runKBContextQuery`. Return a `KBContextResult` with a new `TimedOut` flag instead of nil. This is the structural fix: timeout becomes a signal, not silence.

**Layer 2 — Classification:** Add `GapTypeTimeout` to the gap analysis vocabulary. When the result is timed-out with no matches, classify as warning ("we don't know") instead of critical ("there's nothing"). The gap gate — which can block spawns — explicitly exempts timeout: you can't block a spawn based on information you failed to gather.

**Layer 3 — Headroom:** Raise the per-query timeout from 5s to 10s. This is the symptom fix — most queries that currently fail at 5.8-8.8s would complete at 10s. But without Layers 1-2, the next time a query exceeds 10s, we'd be back here.

The fix avoids lying about what we know. Quality stays at 0 (we did find zero matches). But the gap *type* explains why — timeout, not absence. Display text says "timed out" not "no context." The gate doesn't fire. Timeout events get logged for tracking.

## Tension

The per-query timeout architecture is expedient but not ideal. Each of the up to 4 queries in the lookup chain gets its own 10s timeout, so worst case is 40 seconds of spawn setup. A budget-based approach (15s shared across all queries) would be cleaner — but it requires threading `context.Context` through the public API, which is a bigger change. The question: is the current "raise to 10s and detect" good enough to ship, or should we go straight to budget-based to avoid coming back to this?

# Brief: orch-go-betfg

## Frame

You declared comprehension bankruptcy today — 100+ items in the backlog, no structure, no way to distinguish what matters from what's just plumbing. The system stops spawning at 5 unreviewed items, so when you're away for days, it goes idle. The queue fills with bug fixes and test cleanup sitting alongside architecture decisions and investigation findings, all looking equally demanding of your attention.

## Resolution

The fix isn't one thing, it's a distinction that was missing: not all completed work needs your engagement. Bug fixes, test additions, config changes, infrastructure plumbing — these don't produce knowledge that requires human reaction. They just happened, and it's fine that they happened. So the first move is classifying briefs as "maintenance" or "knowledge" at completion time, using the issue type and skill the daemon already infers. Maintenance items bypass the comprehension queue entirely. Brief still gets generated (so composition can cluster it), but it doesn't count toward the spawn throttle. That alone should drop queue pressure ~60%.

The second move: the daemon runs `orch compose` on a schedule — every 2 hours when 8+ briefs have piled up. No session detection needed. The accumulation IS the signal. When briefs are stacking up without review, that means you're away. The daemon just clusters what's there and writes a digest.

The third: orient surfaces the digest when you come back. "12 briefs arrived while you were gone, they cluster into 3 themes." Not a wall of 40 items — a pre-structured starting point for the conversation where comprehension actually happens.

I was initially drawn to building session detection (tracking when you last ran orch complete, monitoring TUI sessions) but realized it was solving a problem that doesn't exist. The comprehension queue count already encodes "human is away" — that's what an accumulating queue means. Adding session state would violate the no-local-agent-state constraint for no benefit.

## Tension

The maintenance classification is a heuristic — bugs and debug work are clearly maintenance, but "feature-impl" tasks with titles like "wire authentication middleware" are judgment calls. I defaulted uncertain items to "knowledge" (queue them) because the cost of missing an insight is higher than the cost of reading one extra brief. But if the 60% reduction doesn't materialize in practice, the heuristic might need tuning. Worth watching the first week of data.

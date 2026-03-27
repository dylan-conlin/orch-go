# Brief: orch-go-gbkse

## Frame

The thread list shipped on the home surface and you immediately noticed two things: "I expect to be able to expand these" and "Should the sort make more sense?" Both point to the same gap — the list was a recent-activity log pretending to be an orientation surface.

## Resolution

Three small changes, all in one Svelte file. First, threads now sort by status bucket — active threads float to the top because they're what you're thinking about, forming threads sit below because they're still crystallizing, converged threads trail because their work is done. Within each group, recency still applies. The effect is that opening the dashboard answers "what am I working on?" before "what did I touch last?"

Second, every thread row now has a chevron — a small right-pointing arrow that rotates 90 degrees when you expand it. This sounds trivial but it's the difference between "maybe these are interactive?" and "obviously these open." Third, the cryptic `~` and `*` status icons are replaced with colored status pills that say "active", "forming", or "converged" in the thread's status color. Status is now the first thing your eye catches, which matters because it's also the first sort dimension.

## Tension

These changes make the thread list *feel* orienting, but the real test is whether you actually start your day from the top of this list rather than from the CLI. If the answer is still `orch orient`, the dashboard version may need something the CLI has that this surface doesn't — changelog, models, ready queue — all the context that turns "here are your threads" into "here's where you left off."

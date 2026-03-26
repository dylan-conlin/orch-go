# Brief: orch-go-jqkvm

## Frame

This looked like a small tracing question: how does spawn choose between the work and personal Claude accounts? The catch was that the story living in the knowledge base still said "check primaries first, spill over when they drop below a threshold," which is exactly the kind of tidy explanation that becomes dangerous once the code moves on.

## Resolution

The turn was realizing the code is solving a more interesting problem than the docs admit. `resolveAccount()` does not ask "which role should win?" It asks "which account still has usable capacity after both Anthropic limits and the subscription tier are accounted for?" That means it multiplies each account's 5-hour and 7-day percentages by tier, then takes the smaller of the two as the real score. A 20x account with strong short-term headroom can still lose if its weekly limit is near zero, because the weekly bottleneck is what actually blocks the spawn.

Once that clicked, the rest of the session stopped being about code and became about memory repair. The implementation and tests were already coherent; the problem was that our shared explanation had frozen around an older work-first/spillover narrative. I updated the investigation and the main models so the next person who asks this question starts from the actual mechanism instead of a superseded routing myth.

## Tension

The open question is whether this scoring model should become more visible in operator-facing surfaces. Right now the code knows the right answer, but humans still mostly see raw percentages and role labels, which makes it easy to reason with the wrong mental model again.

# Brief: orch-go-c1bqo

## Frame

The dashboard home page was born as an execution monitoring tool and grew comprehension surfaces on top. The product decision says comprehension is primary, but scrolling past the threads section landed you in a full agent management interface — coaching health, active agent grids, swarm maps, event streams. Two dashboard modes (operational and historical) existed just to organize this execution content. The question was: what happens if you delete it all?

## Resolution

The subtraction was cleaner than expected. Every execution section could be removed without touching the comprehension layer — no shared state, no entangled rendering. The 1,055-line home page became 345 lines. The operational/historical mode toggle died instantly because it only existed to organize content that's now gone. QuestionsSection got promoted above the fold (it had been hidden behind the mode toggle, invisible in historical mode — a quiet bug nobody noticed).

The surprise was that the mode toggle was never a real architectural concept. It felt important — two modes, localStorage persistence, a whole store dedicated to it. But it was scaffolding around the wrong content. Remove the content, and the scaffolding has nothing to hold.

What's left: threads (the thinking spine), brief and question badges, blocking questions detail, the review queue, and one summary line — "3 agents active · 5 ready · 2 need review — View Work →". The home page now says what it is instead of what it used to be.

## Tension

The classification assumed Dylan doesn't start mornings by scrolling past threads to check the active agents grid or coaching health. If that turns out to be wrong, several "residue" items need to be re-added as bridge elements. The subtraction is independently revertible per section, but it needs a few days of conscious use before committing to the new shape.

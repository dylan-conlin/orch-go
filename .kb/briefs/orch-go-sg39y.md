# Brief: orch-go-sg39y

## Frame

The product has a name problem. The decision that thread/comprehension is the core product — not orchestration substrate — means "orch" is the wrong word facing outward. But every developer tool name you'd brainstorm in thirty seconds is already taken by someone with more money: Loom is Atlassian's billion-dollar acquisition, Weave belongs to Weights & Biases, Thread is Meta's social network, Trellis is a 4,300-star "agent harness" on GitHub. The question isn't just what to call it — it's whether a viable name exists at all in the single-word devtool namespace.

## Resolution

I checked 23 candidates across three waves, searching domains, GitHub, npm, PyPI, Go registries, and startup databases for each. Seventeen came back RED — fatal collisions with funded products in overlapping spaces. Six survived as YELLOW. The strongest is **Kenning**: an Old Norse word for compound metaphor (like "whale-road" for sea), etymologically from *kenna*, "to know." It's the only candidate where meaning matches the product (composing simpler concepts into richer understanding), every major package registry is unclaimed, and the first impression points toward "knowledge tool" without needing a tagline to redirect. kenning.sh and kenning.so are available.

What surprised me was where the clear space was. Every name that evokes weaving, accumulation, or data processing is taken — because those are infrastructure concepts, and infrastructure tools are what most people build. The names that evoke understanding and comprehension have more room. The namespace difficulty is itself evidence that the product positioning is differentiated: you're building in a semantic territory that most devtools don't occupy.

The naming architecture recommendation is layered: use "Kenning" on external surfaces (README, method guide, website), keep `orch-go`/`orch` in the code and CLI, and stage a deep rename only after v1 traction proves the name. This costs zero code changes now.

## Tension

The obvious risk: "kenning" is a word most developers don't know. That's either a bug (adoption friction from obscurity) or a feature (the name teaches you something, creates curiosity, sticks once learned). I can't resolve this from research alone — it needs five developers seeing the name + tagline cold and reporting what they think the product does. The collision data is solid; the reception data doesn't exist yet.

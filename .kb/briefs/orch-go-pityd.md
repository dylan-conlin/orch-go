# Brief: orch-go-pityd

## Frame

You accepted two decisions today: the product is the comprehension layer, and it should be open at the boundary and opinionated at the core. Clear enough as direction. But the next time someone asks "should we open-source the skill system?" or "should the review gates be configurable?", the principle alone doesn't answer it — you'd re-argue the boundary each time. This investigation makes the principle into a lookup table.

## Resolution

I inventoried every concrete surface in the system — 28 of them across packages, commands, artifact formats, APIs, skills, and dashboard routes — and classified each one. The result is asymmetric in a way that surprised me: 21 of 28 surfaces should be open or configurable. The "held-back" bucket is only 3 items, and none of them exist as shipped product yet (hosted comprehension UX, ranking intelligence, collaborative knowledge). The method core — thread-first organization, synthesis expectations, knowledge placement, review discipline, uncertainty treatment — is the only thing that stays fixed. Five surfaces, not configurable.

The useful byproduct was a 4-question test: (1) Can the user change it without dissolving the method? → Configurable. (2) Would making it optional turn the product into generic infra? → Fixed. (3) Does it lower adoption fear? → Open. (4) Does it only create leverage when integrated? → Held-back. This should outlive the static matrix.

The temporal sequencing turned out to matter as much as the classifications. Because the codebase is 72% substrate and 16% core, opening everything at once makes the project look like an orchestration CLI — the framing you just rejected. The recommendation is three waves: artifact formats now (they teach the method by example), CLI and API with the first release, held-back surfaces when they actually exist and are differentiated.

## Tension

The held-back category is thin because the product's future leverage surfaces don't exist yet. That's honest, but it means "held-back" is currently indistinguishable from "hasn't been built." The risk isn't giving away too much — it's that the things you'd want to hold back aren't real enough to defend. The question is whether to invest in building those surfaces before the first release, or whether the open surfaces alone are compelling enough to drive adoption that generates the data those held-back surfaces need.

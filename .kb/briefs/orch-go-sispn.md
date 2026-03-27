# Brief: orch-go-sispn

## Frame

We have 329 trials showing that behavioral constraints stop working when you pile enough of them onto an agent — 83 out of 87 orchestrator constraints are non-functional, failures are deterministic (always 5/5 or always 0/5), and constraint dilution kicks in around 10 co-resident rules. But we don't know WHY. Three explanations predict the same symptom (constraints failing) but different failure shapes, and we've never tried to tell them apart.

## Resolution

I started by reading every prior probe and the full coordination model to understand what the data could already tell us. The existing evidence actually narrows the field more than I expected: pure resource competition predicts probabilistic degradation (3/5, 4/5 success), but our data shows strict determinism (5/5 or 0/5). That was the first surprise — one mechanism is already partially eliminated before running anything.

The experiment has three phases, each measuring a different dimension. Phase 1 scales constraint count from 1 to 20 on identical tasks and plots the degradation curve — a sigmoid means resource competition, a step function means threshold collapse, and an irregular jagged line means interference. Phase 2 holds N constant and swaps out WHICH constraints are present (distributed across categories vs clustered in the same semantic domain). If the clustered set degrades more, constraints are interfering through semantic proximity — an attractor-on-attractor failure that we've theorized about but never measured. Phase 3 removes constraints one at a time from a failing set and watches what recovers: if everything recovers (threshold), if only neighbors recover (interference), or if there's a uniform small bump (resource). I built 20 constraints that are all simultaneously satisfiable and each detectable by grep, so scoring is automated and free. Total cost: about $2-5 in Haiku.

## Tension

The experiment assumes constraints degrade within the range N=1 to N=20. If Haiku follows all 20 constraints perfectly, we learn nothing — and the real degradation might only show up in system prompts with 100+ rules (more like the actual orchestrator skill). There's also a question I couldn't resolve: is "constraint following" even the right measurement? Maybe what degrades isn't compliance but *quality* of compliance — the agent follows the letter of the constraint while violating its spirit, which a grep detector would miss entirely.

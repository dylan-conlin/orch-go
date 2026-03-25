# Brief: orch-go-3tyik

## Frame

The coordination experiments were designed to test whether placement instructions help parallel agents avoid merge conflicts. But the task prompts themselves contained a placement instruction — "Place the function after FormatDurationShort" — in every condition, including the baseline where agents supposedly get no coordination help. The experiment was partially measuring its own instructions rather than the coordination treatment.

## Resolution

Four fixes, ordered by how much they change what future experiments will measure. The biggest: removing the embedded placement line from all four task prompts. This means the "no-coord" baseline will now actually be uncoordinated — agents choose their own insertion points. If they still converge on FormatDurationShort (because it's the natural end-of-file attractor), that's interesting data. If they don't, the prior 100% conflict rate in no-coord was partly an artifact.

The randomization fix was straightforward but important for longer runs — trials now execute in a Fisher-Yates-shuffled order with a recorded seed, so a 5-hour run doesn't systematically put one condition at 2am when the API might behave differently. The scoring dimension that was reported as "6 factors" was actually 5 — `no_regression` was always identical to `tests_pass`. Removed it rather than implementing real regression detection, because the honest count matters more than an inflated one.

## Tension

The embedded placement confound means prior no-coord results (100% conflict, N=20) may have been measuring something different than intended. With the confound removed, the baseline could shift significantly. Should the next experiment round include a "confounded" condition (re-adding the placement hint) to quantify how much the old instruction was doing?

# Brief: orch-go-y1iy3

## Frame

The daemon kept refusing to spawn follow-on work. An agent would design something — say, the orient five-element surface — and when a second issue arrived to build on it, the commit-dedup gate would see the first issue's ID in the description and conclude the work was already done. "The rendering split from orch-go-d6uqc already separated thinking from ops" reads as context to a human, but to the gate it was just a beads ID with commits attached.

## Resolution

The gate was treating any mention of a committed beads ID as evidence of duplication. The existing cross-type filter (task referencing investigation = allow) didn't help here because follow-on work is often the same type as the prior work.

The fix adds title comparison: when a same-type referenced ID has commits, the gate now checks whether the new issue's title overlaps with the referenced issue's title. If they describe different work — "surface thinking as dashboard element" vs "redesign orient as five-element surface" — the reference is contextual and allowed through. The key threshold insight: follow-on work in the same area naturally shares 40-60% of vocabulary with prior work ("orient", "thinking", "surface" appear in both). True duplicates share 80%+. A 0.7 coefficient threshold cleanly separates the two populations for observed cases.

## Tension

The 0.7 threshold is empirically derived from two observed false positives and one true duplicate. It hasn't been validated against a larger corpus. If someone writes a genuinely different issue that happens to share 70%+ title vocabulary with a completed one, it gets blocked. The alternative — only extracting IDs from structured fields instead of free text — would be more precise but would require changing how issue descriptions are written across the system.

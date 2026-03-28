# Brief: orch-go-scogn

## Frame

You have 115 testable claims spread across 12 models, and until now the only way to know which ones had been tested was to manually read each model.md, count the probes directory, and cross-reference claim IDs in frontmatter. The architect design for the research cycle identified this visibility gap as the real bottleneck — not execution speed, but knowing what's untested.

## Resolution

`orch research` makes the claim-probe pipeline visible in three modes. The summary view shows all 12 models at a glance — 83% of claims are already tested, but that aggregate hides pockets like smithy-geometry-engine (0/8) and generative-systems-are-organized-around (1/6). The detail view shows each claim with its probes, verdicts, and how-to-verify text for untested ones. The spawn mode creates a triage:ready issue with the claim text, verification method, and prior probe context pre-assembled — eliminating the 10-minute context-gathering step the architect design identified.

The pleasant surprise was that `pkg/claims/` already had the YAML parsing built. The actual new code is mostly probe scanning (reading `**claim:** NI-01` and `**verdict:** confirms` from probe frontmatter) and the cross-reference logic to derive test status. About 10% of probes have non-standard claim references like "n/a" or "implicit" — the parser skips these correctly rather than trying to force-match them.

## Tension

The spawn mode creates a beads issue but doesn't actually spawn an agent — it relies on the daemon or manual `orch spawn` to pick it up. This is by design (the architect explicitly chose manual trigger over automation to avoid compliance-driven probes), but it means there's a gap between "I see NI-06 is untested" and "an agent is working on it." Whether this gap is productive friction or unnecessary ceremony depends on how often the research cycle actually gets used.

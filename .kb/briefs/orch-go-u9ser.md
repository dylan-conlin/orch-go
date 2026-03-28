# Brief: orch-go-u9ser

## Frame

We discovered the hotspot harness had been disabled for 5 weeks without anyone noticing. A probe two hours earlier measured adoption rates for every compositional signal — investigation model links, brief tensions, probe claims, thread resolutions, beads enrichment, decision linkage. The numbers were bad (5 of 7 critical), but the scarier finding was: nobody was watching them. If you only measure when someone thinks to look, you only catch drift after it's done damage.

## Resolution

Built `orch harness adoption` — a command that runs the same measurements from the probe, but in Go, with targets and severity levels. Wired it into `orch orient` so the numbers appear at every session start, sandwiched between health summary and daemon health. The implementation revealed a precision issue: the original probe's grep commands counted "claim:" anywhere in a file, including body text like "the model's claim: X." The structured measurement restricted to frontmatter only, cutting the probe claim count from 57 to 35. The truth is worse than the original probe reported.

The three-tier status system (ok / drift / critical) maps naturally to the model's finding: below half the target, the signal is "effectively dead." Only Brief tension (100%) is ok — the single required signal. Everything opt-in is in the red. The orient integration means you'll see this every morning. You can't not notice anymore.

## Tension

The targets are aspirational (80% for most signals), but current reality is 12-19% for opt-in signals. Is the right response to lower the targets to something achievable, or to convert opt-in signals to opt-out? The model says opt-out is the only path to >80% adoption, but converting 5 signals to opt-out would add substantial ceremony to every agent's workflow. The measurement is running — but it's measuring a gap that might require a design change, not just better compliance.

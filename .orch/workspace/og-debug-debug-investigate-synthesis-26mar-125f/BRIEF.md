# Brief: orch-go-n4uwb

## Frame

The alarming number was 4%: almost no `feature-impl` workers were leaving a `SYNTHESIS.md`, which makes it look like implementation agents are ignoring one of the core completion rituals. The important question was whether this meant the skill had become too weak to carry protocol, or whether the system itself was scoring the wrong thing.

## Resolution

The turn came from reading the live spawn contract instead of the benchmark headline. Current `feature-impl` work usually spawns as `light` tier, and those contexts explicitly tell the worker that `SYNTHESIS.md` is not required. The pipeline reinforces that choice by capping light-tier verification at `V0`, so the synthesis gate never runs. In other words, most of the 4% signal is not workers disobeying; it is us counting an artifact that the default configuration no longer asks for.

There is still a smaller real problem hiding underneath. When `feature-impl` gets upgraded to full tier, the shared worker protocol adds `SYNTHESIS.md`, `BRIEF.md`, and `VERIFICATION_SPEC.yaml`, but the skill-local completion section still teaches the older ending: report `Phase: Complete`, then commit. So the metric is mostly wrong, but the prompt is also internally split in the cases where synthesis actually matters.

## Tension

The remaining judgment call is semantic, not mechanical: should `feature-impl` keep a light-tier default with `Phase: Complete` as its main compliance signal, or should the system redefine some implementation work as knowledge-producing and require synthesis more often? Until that is designed explicitly, any single synthesis-compliance number for `feature-impl` will keep mixing optional and required behavior.

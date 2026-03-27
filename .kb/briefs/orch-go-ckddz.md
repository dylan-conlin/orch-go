# Capability-Aware Daemon Model Router

## Frame

The daemon had a flat lookup table for model selection: five deep-reasoning skills mapped to opus, everything else returned nothing and let the resolve pipeline figure it out. This meant feature-impl tasks — the majority of daemon spawns — always got the same treatment regardless of whether they were a one-line label change or a multi-file refactor. Meanwhile, the effort labels (small, medium, large) were sitting right there on the issue, already used for completion routing, but completely invisible to model selection.

The moment that clicked was reading the existing `InferModelFromSkill` comment: *"Implementation-heavy skills are deliberately excluded even when they sometimes benefit from a stronger model."* The word "even" was doing a lot of work — it was acknowledging the problem without solving it. The escape hatch of returning empty string was designed for the resolve pipeline to fill, but the resolve pipeline just picked the account default (opus on Max). There was no signal flowing from the issue to the model.

## Resolution

Three capability classes — deep-reasoning, implementation, light — replace the flat map. Deep-reasoning skills still get opus unconditionally. Implementation skills now check the issue's effort label: small or medium routes to gpt-5.4, large or unlabeled falls through to the resolve pipeline. Every routing decision carries a reason string that flows through `SkillRoute → SpawnDecision → OnceResult → daemon logs`.

The conservative choice was deliberate: no effort label means no routing to gpt-5.4. If the issue wasn't triaged for effort, it gets opus. GPT-5.4 routing is opt-in via effort labeling, not opt-out.

## Tension

This routes gpt-5.4 based on effort labels, but we don't yet know its stall rate on feature-impl small/medium. The CLAUDE.md documents 67-87% stall rates for GPT-4o/GPT-5.2-codex on *protocol-heavy* skills — but feature-impl with bounded effort is a different regime. If gpt-5.4 stalls on even simple implementation tasks, the routing threshold needs to shift or the whole GPT path needs gating behind a success-rate check from the learning store. Worth watching overnight runs before trusting this in production.

# Brief: orch-go-u2rve

## Frame

This started as a deceptively small question: if orch-go already knows that picking `gpt-5.4` or another non-Anthropic model changes the spawn path, why does that shift still feel invisible once the daemon is doing the work? That matters because the change is not cosmetic - it swaps runtime assumptions, observability, and sometimes cost posture.

## Resolution

The turn was realizing the system is not missing detection. The resolver already notices the moment when a model choice widens the execution contract and flips the backend. What is missing is a durable story about that moment. Right now it becomes a warning string, which works fine when a human is staring at a direct spawn, but the daemon path throws successful command output away, so the explanation evaporates.

That makes the real design move pretty clean: do not teach every downstream surface to rediscover the change. Let the resolver emit one structured routing-impact report, then let manifests, events, dry-runs, and daemon summaries all reuse it. The interesting part is that this is a scope-expansion problem in the defect-taxonomy sense: the system widened what models can imply, but it never upgraded the consumer contract for how that widening gets reported.

## Tension

The open judgment call is how far to take the report once it exists. It is clearly worth surfacing, but it is not yet obvious whether missing routing-impact metadata should remain advisory or eventually become something completion review enforces.

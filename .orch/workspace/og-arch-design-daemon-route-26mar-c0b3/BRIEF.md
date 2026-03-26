# Brief: orch-go-3k1yo

## Frame

The daemon's old model story was simple: deep reasoning goes to Opus, everything else mostly falls through to defaults. That worked while OpenAI models were mostly cautionary tales, but the new GPT-5.4 benchmark changed the question. We no longer need to decide whether GPT belongs in the system at all; we need to decide where it is trustworthy enough to carry real work without turning routing into vibes.

## Resolution

The turn was realizing that this is not really a "provider choice" problem. The code still thinks in skills, and the benchmark thinks in capabilities. GPT-5.4 looked genuinely good on bounded implementation work, but it still showed two weaknesses that matter operationally: weaker scope control and at least one silent first-attempt failure on investigation work. So the right shape is not "switch implementation to GPT" and not "keep GPT manual forever." It is a two-lane system: Opus stays the default lane for reasoning-heavy or high-complexity work, while GPT-5.4 becomes a bounded overflow lane for `feature-impl` tasks that are explicitly small or medium.

What made that feel solid instead of clever is that the daemon already has most of the structure it needs. There is already a route-rewrite seam in coordination, already a config pattern for combo-first policy resolution, and already `effort:*` labels that can stand in for complexity on day one. The missing piece is not another benchmark or another flag. It is making routing and recovery one design, so the system can say why it chose GPT and then promote back to Opus once if GPT fails in a known way.

## Tension

The open question is whether Sonnet should become a middle lane before GPT expands beyond overflow. The other tension is empirical: if `effort:*` labels are sparse in real daemon queues, the design stays safe but may be more conservative than Dylan wants.

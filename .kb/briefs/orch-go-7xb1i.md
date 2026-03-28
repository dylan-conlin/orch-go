# Brief: orch-go-7xb1i

## Frame

The thread promote command truncates slugs to 5 words. When you promoted "Generative systems are organized around named incompleteness," it kept the setup phrase and cut the concept. This created two model directories — one with the truncated name that froze immediately, and the canonical `named-incompleteness/` that kept evolving. Probes were being written to both, and two of the five probes in the real directory still pointed at the ghost.

## Resolution

Deleted the stale duplicate (no unique content — the real model had received 4 updates the copy never got) and fixed the probe references. The interesting part: the system mostly self-corrected. Three of five probes in the canonical directory already used the right name. Agents figured out which directory was real and routed there — the truncated copy just sat as noise. But the subsumption probe got duplicated into both, which is exactly what the named-incompleteness model predicts under Failure Mode 2: when a model's identity is split across two names, probes can't tell which is authoritative, so they hedge.

Added this as a new instance in the model's Failure Mode 2 section. Two names for one model degrades composition the same way as no name — it's not that the gap is missing, it's that the gap has two addresses and nothing can tell which is canonical.

## Tension

The thread promote naming bug (orch-go-la6xo) was the root cause here, and this is just cleanup of one downstream effect. But the fact that agents mostly self-corrected — routing to the canonical name within a day — suggests the system has informal disambiguation mechanisms that aren't captured anywhere. What are they? Is it just "pick the directory with more content" or something more interesting?

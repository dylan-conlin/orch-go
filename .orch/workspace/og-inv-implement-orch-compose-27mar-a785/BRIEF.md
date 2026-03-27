# Brief: orch-go-hmdan

## Frame

The composition layer design was done — you knew what you wanted: cluster briefs, harvest tensions, surface thread connections, write a digest. The remaining question was whether keyword-based clustering could actually produce meaningful groups from 71 briefs that all live in the same project and share the same vocabulary. The design said "3+ keyword overlap" and I believed it. The first run proved it wrong: one cluster, 71 members.

## Resolution

The problem wasn't the threshold — it was the algorithm. Single-linkage clustering lets any member recruit new members, so brief A shares 3 words with B, B shares 3 with C, C shares 3 with D, and suddenly A-through-D are one cluster despite A and D having nothing in common. Every brief in orch-go talks about agents, spawning, and orchestration. Any transitive chain connects everything.

The first wrong turn was TF-IDF. The textbook says "weight keywords by inverse document frequency" to find distinctive terms. But distinctive terms are unique to individual briefs — the exact opposite of what clustering needs. After TF-IDF top-15 filtering, zero pairs overlapped. TF-IDF selects for what makes documents different; clustering needs what makes them similar.

The fix was mid-band document frequency filtering: keep only words that appear in 2-20% of briefs. Words in more than 20% are domain-ubiquitous ("agent", "spawn", "system"). Words in fewer than 2 briefs are unique noise. The middle band is where clustering signal lives — words shared by some briefs but not all. Combined with seed-based clustering (only the most-connected brief can recruit, preventing chains), the 71 briefs split into 9 clusters with thread matches.

## Tension

The clusters are keyword-driven, not meaning-driven. "checks / claude / dead" is a cluster name, not a concept. Whether these 9 groups match what you'd see reading the briefs yourself is unknown — the design says that's exactly the question composition should surface for you to answer, not resolve automatically. But if the clusters consistently don't match your mental model, keyword clustering may be fundamentally insufficient for this corpus size, and the Phase 2 upgrade to LLM-powered clustering becomes not optional but necessary.

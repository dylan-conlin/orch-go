---
title: "Coordination protocol primitives — Route Sequence Throttle Align"
status: open
created: 2026-03-22
updated: 2026-03-22
resolved_to: ""
---

# Coordination protocol primitives — Route Sequence Throttle Align

## 2026-03-22

Harness constrains a single agent. Protocol is what emerges when multiple harnesses must stay coherent. Four primitives: Route (agents don't collide), Sequence (work happens in right order), Throttle (velocity doesn't exceed verification bandwidth), Align (agents share a current, accurate model of what correct means). Every major orch-go failure maps to missing primitives. Align is the non-obvious one — it's what prevents slow degradation where everything looks fine. Without Align, the other three primitives themselves drift (gates measure wrong things, routes go stale, throttle thresholds stop matching reality). Hypothesis: these four are general to any multi-agent coordination, not specific to orch-go. Test: map external framework failures (CrewAI, AutoGen, autoresearch) to the primitives. Key question from Dylan: 'is orch-go a coordination protocol, or did orch-go discover that coordination protocols have four primitives?' The answer shapes whether the next move is product (build) or contribution (publish the finding). Current stance: treat as hypothesis with strong evidence, design tests before committing.

## Auto-Linked Investigations

- .kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md

External validation complete (orch-go-nsb49). Six independent sources confirm the four primitives are general: MAST 14/14 failure modes map cleanly, McEntire shows monotonic degradation (100→64→32→0%), Getmaxim independently derives same 4 categories, DeepMind shows 17.2x→4.4x error reduction when adding Route+Sequence. Align is 50% of failures and most neglected. The individual phenomena are documented everywhere — the unification into four structural requirements is the potential contribution. Dylan decided against blog post (burned by harness engineering 0-traction). Externalization lives in: coordination model (updated), this thread, investigation artifact, and Dylan's interview/conversation repertoire.

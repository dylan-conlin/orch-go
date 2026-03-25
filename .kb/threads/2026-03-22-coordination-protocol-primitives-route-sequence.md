---
title: "Coordination protocol primitives — Route Sequence Throttle Align"
status: active
created: 2026-03-22
updated: 2026-03-24
resolved_to: ""
spawned_from: ""
spawned:
  - research-extraction-coord-bench-as
  - comprehension-artifacts-async-synthesis-delivery
active_work: []
resolved_by: [orch-go-nwbno, orch-go-i77wx]
---

# Coordination protocol primitives — Route Sequence Throttle Align

## 2026-03-22

Harness constrains a single agent. Protocol is what emerges when multiple harnesses must stay coherent. Four primitives: Route (agents don't collide), Sequence (work happens in right order), Throttle (velocity doesn't exceed verification bandwidth), Align (agents share a current, accurate model of what correct means). Every major orch-go failure maps to missing primitives. Align is the non-obvious one — it's what prevents slow degradation where everything looks fine. Without Align, the other three primitives themselves drift (gates measure wrong things, routes go stale, throttle thresholds stop matching reality). Hypothesis: these four are general to any multi-agent coordination, not specific to orch-go. Test: map external framework failures (CrewAI, AutoGen, autoresearch) to the primitives. Key question from Dylan: 'is orch-go a coordination protocol, or did orch-go discover that coordination protocols have four primitives?' The answer shapes whether the next move is product (build) or contribution (publish the finding). Current stance: treat as hypothesis with strong evidence, design tests before committing.

## Auto-Linked Investigations

- .kb/investigations/2026-03-23-inv-investigate-orch-go-coordination-primitives-port.md
- .kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md

External validation complete (orch-go-nsb49). Six independent sources confirm the four primitives are general: MAST 14/14 failure modes map cleanly, McEntire shows monotonic degradation (100→64→32→0%), Getmaxim independently derives same 4 categories, DeepMind shows 17.2x→4.4x error reduction when adding Route+Sequence. Align is 50% of failures and most neglected. The individual phenomena are documented everywhere — the unification into four structural requirements is the potential contribution. Dylan decided against blog post (burned by harness engineering 0-traction). Externalization lives in: coordination model (updated), this thread, investigation artifact, and Dylan's interview/conversation repertoire.

## 2026-03-24

Decomposition experiment (N=100, 5 conditions) confirms the attractor > gate > reminder hierarchy at the coordination layer. Task description quality alone (C1-C3): 0% effect, equivalent to advisory gates. File structure + task framing (C5): 40% success, moderate but unreliable. Structural placement (prior data): 100% success. Decomposition hints are reminders — they compete with the task and lose. Placement is an attractor — it removes the wrong option. This is the same pattern as accretion gates (100% bypass), CLAUDE.md conventions, and knowledge orphan rates. The four primitives survive the decomposition challenge: Route does independent load-bearing work that decomposition cannot replace. Updated position: decompose first to reduce coordination surface, then apply structural coordination (placement/attractors) for what remains.

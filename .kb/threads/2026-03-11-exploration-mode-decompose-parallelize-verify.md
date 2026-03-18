---
title: "Exploration mode — decompose/parallelize/verify/iterate for bounded problems"
status: resolved
created: 2026-03-11
updated: 2026-03-17
resolved_to: "All 4 phases shipped. Exploration mode is feature-complete: --explore flag, judge skill, observability events, iteration with depth control. Production hardening (daemon integration, dashboard visualization, documentation) tracked separately."
---

# Exploration mode — decompose/parallelize/verify/iterate for bounded problems

## 2026-03-11

Cursor solved First Proof Challenge Problem 6 (spectral graph theory) using their general-purpose coding harness — same one that built a browser. Ran 4 days autonomously, beat the human solution. Their key claim: 'prompts matter more than architecture.' Our harness model says the opposite (soft harness dilutes, hard harness holds). Resolution: not contradictory — different failure modes. Cursor optimizes for compliance on bounded tasks (exploration harness). We optimize for coordination on living systems (maintenance harness). The practical insight: exploration mode for investigations/architect work — parallel decomposition with judge synthesis, no codebase writes. Exploration is upstream of enforcement: explore freely in isolation, then existing gates filter what gets promoted. Starting point: investigation and architect skills only (analysis, not code production).

Session outcome: Phases 1-3 shipped (--explore flag, judge skill, observability). Phase 4 (iteration) in flight. During review, discovered that our comprehension gates (explain_back, verified) are captured by AI orchestrator — rubber-stamped. Three-Layer Reconnection doesn't push to Dylan. Led to orch review synthesize command and review redesign issue. The exploration mode conversation catalyzed a broader audit of where comprehension actually lives in the system.

## 2026-03-17 — Resolved

All 4 phases are now shipped and tested:

1. **--explore flag** — `spawn_cmd.go` with `--explore-breadth` (1-10, default 3) and `--explore-depth` (1-5, default 1). Restricted to investigation/architect skills.
2. **Judge skill** — exploration-judge with 5-dimension YAML verdicts (Grounding, Consistency, Coverage, Relevance, Actionability). Verdict outcomes: accepted/contested/rejected.
3. **Observability** — 4 event types (exploration.decomposed, .judged, .synthesized, .iterated) with harness dashboard tracking. Full emit command support.
4. **Iteration** — depth > 1 enables judge-triggered re-exploration. Critical coverage gaps spawn new workers. Cost-bounded by depth limit.

**Production hardening gaps** (not blocking resolution):
- Daemon doesn't auto-detect exploration-worthy tasks
- No dashboard visualization of exploration metrics
- No exploration-specific completion verification gates
- No `.kb/guides/` documentation for exploration mode

The core insight held: exploration is upstream of enforcement — explore freely in isolation, existing gates filter what gets promoted to the codebase.

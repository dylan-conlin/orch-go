---
title: "Exploration mode — decompose/parallelize/verify/iterate for bounded problems"
status: open
created: 2026-03-11
updated: 2026-03-11
resolved_to: ""
---

# Exploration mode — decompose/parallelize/verify/iterate for bounded problems

## 2026-03-11

Cursor solved First Proof Challenge Problem 6 (spectral graph theory) using their general-purpose coding harness — same one that built a browser. Ran 4 days autonomously, beat the human solution. Their key claim: 'prompts matter more than architecture.' Our harness model says the opposite (soft harness dilutes, hard harness holds). Resolution: not contradictory — different failure modes. Cursor optimizes for compliance on bounded tasks (exploration harness). We optimize for coordination on living systems (maintenance harness). The practical insight: exploration mode for investigations/architect work — parallel decomposition with judge synthesis, no codebase writes. Exploration is upstream of enforcement: explore freely in isolation, then existing gates filter what gets promoted. Starting point: investigation and architect skills only (analysis, not code production).

Session outcome: Phases 1-3 shipped (--explore flag, judge skill, observability). Phase 4 (iteration) in flight. During review, discovered that our comprehension gates (explain_back, verified) are captured by AI orchestrator — rubber-stamped. Three-Layer Reconnection doesn't push to Dylan. Led to orch review synthesize command and review redesign issue. The exploration mode conversation catalyzed a broader audit of where comprehension actually lives in the system.

---
title: "Research extraction — coord-bench as standalone empirical coordination research"
status: forming
created: 2026-03-24
updated: 2026-03-24
resolved_to: ""
spawned_from: coordination-protocol-primitives-route-sequence
spawned: []
active_work: [orch-go-i77wx]
resolved_by: [orch-go-i77wx]
---

# Research extraction — coord-bench as standalone empirical coordination research

## 2026-03-24

orch-go's most defensible output is empirical coordination research (329 trials, epistemic tiers, falsifiable claims), not the orchestration machinery. The research is currently coupled to orch-go's spawn infrastructure but doesn't need it — experiments need: spawn 2 agents, point at a file, merge, score. That's ~200 lines, not a daemon+skills+beads stack. Extraction means a standalone repo (coord-bench) with a framework-agnostic harness, raw results data, and the coordination model. Distribution advantage: framework-neutral findings are citable by CrewAI/LangGraph/OpenClaw authors, reproducible by anyone with API access, and legible as a research contribution for AI infra roles. orch-go keeps consuming the findings but stops owning the research. The decomposition experiment (orch-go-nwbno, designed, ready to run) would be the first experiment published in this format. Key tension: the experiment harness currently shells out through orch spawn — decoupling requires building a simple direct-API runner. Open question from Dylan: should experimentation/empirical research be what orch-go is about? Current position: probably not — the research should be extractable because its value is framework-neutral, and coupling it to orch-go makes it less accessible, not more.

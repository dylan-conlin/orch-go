---
title: "Compliance/coordination bifurcation — as models improve, coordination tooling becomes more valuable than compliance tooling"
status: open
created: 2026-03-12
updated: 2026-03-12
resolved_to: ""
---

# Compliance/coordination bifurcation — as models improve, coordination tooling becomes more valuable than compliance tooling

## 2026-03-12

Almost all AI agent tooling today solves compliance — getting agents to follow instructions. Prompt engineering, RLHF, instruction tuning, system prompts, context management — all compliance infrastructure. The observation from orch-go (daemon.go +892 lines from 30 correct commits, 265-trial dilution curve) is that as models get better at compliance, remaining failures shift to coordination. This isn't distant future — anyone running 5+ concurrent agents hits it now. Implication: tooling landscape bifurcates. Compliance tooling (better prompts, better models) commoditizes with model improvement. Coordination tooling (structural attractors, gates, entropy measurement) increases in value. The MAST taxonomy (Cemri et al.) observes coordination failures (~32% of 1,600 traces) but prescribes model improvement — a compliance answer to a coordination question. Companies that recognize the bifurcation early have structural advantage. Open question: is there a threshold (agent count? codebase size? velocity?) where coordination costs dominate compliance costs? Our system crossed it somewhere around 20-30 agents/day.

Synthesis is the clearest case of coordination > compliance. The system detects clusters (orient shows synthesis opportunities) but doesn't act on them. Auto-spawning explorations at cluster thresholds (5+ investigations) could close this — the orch-go-j1f7b exploration produced genuine insight (caveat non-propagation) in 21 minutes on opus. The question is whether daemon-driven synthesis produces comprehension or summaries. First data point says yes, but N=1.

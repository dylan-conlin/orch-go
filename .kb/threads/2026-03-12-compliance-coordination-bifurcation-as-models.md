---
title: "Compliance/coordination bifurcation — as models improve, coordination tooling becomes more valuable than compliance tooling"
status: resolved
created: 2026-03-12
updated: 2026-03-18
resolved_to: "decision"
---

# Compliance/coordination bifurcation — as models improve, coordination tooling becomes more valuable than compliance tooling

## 2026-03-12

Almost all AI agent tooling today solves compliance — getting agents to follow instructions. Prompt engineering, RLHF, instruction tuning, system prompts, context management — all compliance infrastructure. The observation from orch-go (daemon.go +892 lines from 30 correct commits, 265-trial dilution curve) is that as models get better at compliance, remaining failures shift to coordination. This isn't distant future — anyone running 5+ concurrent agents hits it now. Implication: tooling landscape bifurcates. Compliance tooling (better prompts, better models) commoditizes with model improvement. Coordination tooling (structural attractors, gates, entropy measurement) increases in value. The MAST taxonomy (Cemri et al.) observes coordination failures (~32% of 1,600 traces) but prescribes model improvement — a compliance answer to a coordination question. Companies that recognize the bifurcation early have structural advantage. Open question: is there a threshold (agent count? codebase size? velocity?) where coordination costs dominate compliance costs? Our system crossed it somewhere around 20-30 agents/day.

Synthesis is the clearest case of coordination > compliance. The system detects clusters (orient shows synthesis opportunities) but doesn't act on them. Auto-spawning explorations at cluster thresholds (5+ investigations) could close this — the orch-go-j1f7b exploration produced genuine insight (caveat non-propagation) in 21 minutes on opus. The question is whether daemon-driven synthesis produces comprehension or summaries. First data point says yes, but N=1.

## 2026-03-18 — Resolution

The open questions from this thread have been answered empirically:

**Threshold question answered:** The system crossed the coordination-dominates-compliance threshold at ~20-30 agents/day, confirmed by 2-week gate effectiveness measurement. Blocking gates showed 100% bypass rate across 55 firings — agents route around compliance instantly. Meanwhile, coordination mechanisms (daemon event signaling → extraction cascades) drove 75% hotspot reduction (12→3 files). The blocking was ceremony; the coordination was mechanism.

**Key evidence:**
- Decision `2026-03-17-accretion-gates-advisory-not-blocking.md`: All gates converted from blocking to advisory based on measurement
- Gate effectiveness query (orch-go-00r9c): Structural enforcement doesn't improve agent quality — 96-100% success rates are intrinsic to skill+model combinations
- Coordination model verified (commit 55b40b9f5): All coordination claims confirmed current

**Synthesis question:** Daemon-driven synthesis remains N=1. The exploration mechanism works (21-minute opus explorations produce genuine insight) but auto-triggering at cluster thresholds hasn't been implemented. This is a coordination capability gap, not a compliance one — supporting the thread's thesis.

**Resolution:** The bifurcation thesis is validated by measurement. Compliance value is measurably approaching zero (100% bypass, 96%+ intrinsic success). Coordination value is measurably increasing (allocation scoring, work graph deduplication, daemon routing are the mechanisms that produce quality). The structural split shipped in 7 commits (2026-03-13) and the blog draft at `.kb/drafts/compliance-cliff.md` captures the full argument. Remaining coordination work (auto-synthesis, coordination ROI measurement) tracked as separate issues.

## Auto-Linked Investigations

- .kb/investigations/2026-03-13-inv-task-compliance-coordination-boundary-map.md

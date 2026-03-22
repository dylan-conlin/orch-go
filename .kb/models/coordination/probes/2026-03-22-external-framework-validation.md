# Probe: External Framework Validation of Four Coordination Primitives

**Date:** 2026-03-22
**Model:** coordination
**Question:** Do the four coordination primitives (Route, Sequence, Throttle, Align) generalize beyond orch-go's experimental context?

## What I Tested

Mapped documented failures from 6 independent external sources to the four primitives:
1. MAST taxonomy (Berkeley, NeurIPS 2025) — 14 failure modes, 1642 traces, 7 frameworks
2. Google DeepMind scaling paper — 180 configurations, error amplification rates
3. McEntire controlled experiment — 28 identical SWE tasks, 4 architectures
4. Anthropic multi-agent research system — production engineering blog
5. Getmaxim production patterns — 4 failure categories from production systems
6. Framework-specific failures (CrewAI, LangGraph, OpenAI Agents SDK, Claude Agent SDK)

## What I Observed

1. All 14 MAST failure modes map to exactly one primitive. No residual category.
2. Getmaxim's independently-derived 4 production failure categories map 1:1 to the primitives.
3. McEntire's success rates (100% → 64% → 32% → 0%) degrade monotonically with missing primitives.
4. DeepMind's error amplification (17.2x → 4.4x) reduces when centralized coordination adds Route + Sequence.
5. Anthropic's production system independently discovered and solved for all four primitives.
6. Align accounts for 7/14 MAST failure modes (50%) — the dominant failure type.
7. autoresearch succeeds by constraining to N=1, trivially satisfying all four primitives.

## Model Impact

**Confirms:** All four core claims confirmed as general, not orch-go-specific:
- Claim 1 (communication insufficient) → Aligns with MAST FM-2.4, FM-2.5, McEntire dysmemic pressure
- Claim 2 (structural placement prevents conflicts) → Aligns with DeepMind centralized > independent
- Claim 3 (individual capability not bottleneck) → Aligns with MAST correlating with HRO theory
- Claim 4 (complexity-independent) → Aligns with McEntire's identical task results across architectures

**Extends:** The model should add:
- The four primitives framework (Route, Sequence, Throttle, Align) as the generalization layer
- Align as the meta-primitive (50% of external failures, most neglected by frameworks)
- The degenerate case: N=1 trivially satisfies all primitives (autoresearch validation)
- Quantitative degradation: success correlates with number of implemented primitives

**New open questions:**
- Whether Align should decompose into sub-primitives (state alignment vs goal alignment)
- Whether primitives have ordering dependencies (must Route precede Sequence?)
- Task-type dependency (DeepMind found strategy varies by task type)

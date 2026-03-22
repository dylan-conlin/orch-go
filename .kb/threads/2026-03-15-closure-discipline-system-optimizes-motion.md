---
title: "Closure discipline — the system optimizes motion more than value, completion review is the bottleneck"
status: resolved
created: 2026-03-15
updated: 2026-03-21
resolved_to: "Structural fixes shipped (completion validator, closure rate measurement, SYNTHESIS.md fallback). Accretion gates converted to advisory after 100% bypass rate. Remaining concern (mechanical completion vs comprehension) migrated to successor thread: comprehension-gated-throughput."
---

# Closure discipline — the system optimizes motion more than value, completion review is the bottleneck

## 2026-03-15

91.3% orphan rate on investigations. 32 stale decisions. Codex independently diagnosed: strong logistics, weak closure. Completion validator (orch-go-lqiel/tfno1) and drain gate (orch-go-6meyr) are the structural fix. Closure rate measurement (orch-go-magh6) provides the baseline. The right test: does orphan rate drop below 20% within 14 days of validator shipping?

Baseline measured (orch-go-magh6): 84% closure rate, 28.7% KB artifact rate, 6.8% dead spawns. By skill: investigation 63% KB, architect 47%, feature-impl 25%, systematic-debugging 28%. The 28.7% overall KB rate confirms the gap — most completions are logistically correct but knowledge-barren. Completion artifact validator (orch-go-lqiel) is now live and enforcing. First enforcement observed: magh6 itself was blocked by missing COMPLETION.yaml — the gate works.

## 2026-03-21 — Resolution

Thread resolved. All three structural fixes shipped and closed:
- **Closure rate measurement** (orch-go-magh6): baseline established (84% closure, 28.7% KB artifact rate)
- **Completion artifact validator** (orch-go-lqiel/tfno1): live and enforcing, with SYNTHESIS.md fallback added 2026-03-20 after agents inconsistently produced COMPLETION.yaml
- **Accretion gates** converted to advisory (2026-03-17 decision) after 100% bypass rate over 2-week measurement — blocking was purely ceremonial; the real gate is daemon extraction cascades triggered by events, not blocks

**Original success criterion** ("orphan rate below 20% within 14 days") was not met — the 14-day window hasn't elapsed, and the metric targets a symptom rather than the root cause. The deeper insight emerged: the system's bottleneck isn't closure mechanics (those work at 84%) but *comprehension during closure* — whether the human actually synthesizes meaning from completions or just rubber-stamps them.

**Successor thread:** `.kb/threads/2026-03-21-comprehension-gated-throughput-pipeline-speed.md` — captures the evolved question: should pipeline speed be limited by human understanding rate?

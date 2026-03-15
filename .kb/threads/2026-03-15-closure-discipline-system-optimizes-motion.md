---
title: "Closure discipline — the system optimizes motion more than value, completion review is the bottleneck"
status: open
created: 2026-03-15
updated: 2026-03-15
resolved_to: ""
---

# Closure discipline — the system optimizes motion more than value, completion review is the bottleneck

## 2026-03-15

91.3% orphan rate on investigations. 32 stale decisions. Codex independently diagnosed: strong logistics, weak closure. Completion validator (orch-go-lqiel/tfno1) and drain gate (orch-go-6meyr) are the structural fix. Closure rate measurement (orch-go-magh6) provides the baseline. The right test: does orphan rate drop below 20% within 14 days of validator shipping?

Baseline measured (orch-go-magh6): 84% closure rate, 28.7% KB artifact rate, 6.8% dead spawns. By skill: investigation 63% KB, architect 47%, feature-impl 25%, systematic-debugging 28%. The 28.7% overall KB rate confirms the gap — most completions are logistically correct but knowledge-barren. Completion artifact validator (orch-go-lqiel) is now live and enforcing. First enforcement observed: magh6 itself was blocked by missing COMPLETION.yaml — the gate works.

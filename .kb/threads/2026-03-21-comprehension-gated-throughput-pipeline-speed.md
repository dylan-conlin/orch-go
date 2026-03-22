---
title: "Comprehension-gated throughput — should pipeline speed be limited by human understanding rate"
status: open
created: 2026-03-21
updated: 2026-03-21
resolved_to: ""
---

# Comprehension-gated throughput — should pipeline speed be limited by human understanding rate

## 2026-03-21

Completion review does two things: (1) mechanical verification (reclaim daemon slot) and (2) synthesis (compose meaning for Dylan). These are in tension — optimizing for throughput produces summaries not comprehension (recognition trap), optimizing for depth stalls the pipeline. Separating them seems obvious: let mechanical completion free slots, queue synthesis for later. But the 777 orphaned investigations prove that deferred synthesis becomes permanent deferral. And if the orchestrator needs synthesis to scope the next issue well, then comprehension IS the rate limiter — the pipeline should slow down when understanding lags. Design premise candidate: the system's throughput is mechanically limited by the human's comprehension rate, not as a bug but as a feature. The system can't outrun the human's understanding. Counter: this means a busy human = idle agents, which feels wasteful. But idle agents with good scoping > busy agents with blind scoping.

## Auto-Linked Investigations

- .kb/investigations/2026-03-06-inv-human-calibration-experiment.md

Investigation orch-go-j4ej7 answered the quality question: routing completeness is 100% (structurally can't fail), but 69% uses coarsest signal (type→skill fallback). No quality gate needed for completeness. But routing ACCURACY is unmeasured — nobody knows if inferred skills are correct. This makes orchestrator-as-scoping-agent more important: orchestrator judgment (adding labels, structured descriptions) is what lifts routing from 69% type-fallback into 12% label-based inference. The scoping agent isn't just a separation-of-concerns move — it's the mechanism that makes daemon routing actually accurate.

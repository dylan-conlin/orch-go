---
title: "Silent cost accumulation — decisions have performance consequences the system never surfaces until they become pain"
status: resolved
created: 2026-03-20
updated: 2026-03-22
resolved_to: ".kb/threads/2026-03-22-coordination-self-awareness-system-that.md"
---

# Silent cost accumulation — decisions have performance consequences the system never surfaces until they become pain

## 2026-03-20

Case: getCompletionsForReview scans 1,106 workspaces across 19 projects just to show a review queue count in orch status. Decision to add review queue to status was locally reasonable, but nobody surfaced the O(n) cost at decision time or as workspace count grew. Pattern: decisions that are fine at N=10 become bottlenecks at N=1000, and the system has no mechanism to detect this drift. Related: this is different from code accretion (file size) — it's *behavioral* accretion. The function didn't get bigger, it just got called against more data. Possible interventions: decision-time cost annotations, periodic performance baselines, workspace count as a tracked metric.

Investigation complete (orch-go-55y1l). 5 confirmed instances. Key insight: behavioral accretion is the 4th substrate of knowledge-accretion (alongside code, knowledge, OPSEC). The critical systemic gap is that nothing tracks N — workspace count, event count, KB file count are all unmeasured scaling variables. You can't detect behavioral accretion without measuring the size of the world your code operates on. This is exactly the epistemic integrity gap: the system's outputs degrade silently because the system doesn't observe its own scaling factors. Follow-ups: events.jsonl rotation (most urgent — 17s parse projected at 1yr), N-value metrics, daemon shared scanning.

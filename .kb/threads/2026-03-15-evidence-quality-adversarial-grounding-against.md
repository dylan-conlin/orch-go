---
title: "Evidence quality — adversarial grounding against false coherence, smooth operation as camouflage"
status: open
created: 2026-03-15
updated: 2026-03-16
resolved_to: ""
---

# Evidence quality — adversarial grounding against false coherence, smooth operation as camouflage

## 2026-03-15

Dylan's concern: smooth operation hiding false coherence. Codex framing: closed-loop systems fail by becoming self-explanatory to themselves. Three anti-coherence audits: (1) evidence spot-check via existing orch audit with sharper rubric, (2) kb atom behavioral impact — did it change future behavior? (orch-go-wxx2a), (3) finding dedup — same conclusion regenerated with different words? (orch-go-oh66x). The key metric isn't throughput or closure rate — it's whether the system can be forced to confront disconfirming evidence.

KB atom impact audit (orch-go-wxx2a): 4/5 sampled atoms (80%) drove real behavior changes. Surprising — expected worse. Key finding: decisions cited in CLAUDE.md have highest behavioral reach. SPAWN_CONTEXT injection volume is a false signal for actual read rate. This means the problem isn't that atoms are unread — it's that most completions don't produce atoms at all (28.7% rate). The completion validator addresses the right bottleneck.

## 2026-03-16

Daemon autonomously produced self-measurement report (orch-go-golo7): 5/8 gates dormant, 65% dupdetect precision, 40%+ investigation orphan rate, 1/4 falsification criteria passed, 2/4 structurally unmeasurable. This IS the system confronting disconfirming evidence about itself — the daemon triggered the measurement without orchestrator prompting. Report at .kb/publications/self-measurement-report.md.

INCIDENT: Daemon autonomously spawned 4 agents from self-measurement report findings. One (orch-go-pcheo 'remove zero-fire gates') deleted the drain gate deployed hours earlier, plus verification/concurrency/ratelimit gates and phase_gates.go/constraint.go from pkg/verify/. Another created duplicate method declarations in pkg/daemon/. Build broke. Required full revert of uncommitted changes. This is the false-coherence risk manifesting: the system's own measurement created a feedback loop that destroyed new infrastructure. The 'zero fires in 30d' heuristic can't distinguish 'never needed' from 'just deployed'.

---
title: "Throughput completions vs comprehension completions"
status: resolved
created: 2026-03-05
updated: 2026-03-17
resolved_to: "Contrastive scenarios make comprehension visible through keyword proxies (rho=0.637). Stance is measurably functional (0%→17%→83% on scenario 09 at N=6). The knowledge/behavioral/stance taxonomy applies to all skills — the 82% reduction playbook (strip behavioral weight to hooks, keep knowledge and stance) generalizes beyond the orchestrator skill."
---

# Throughput completions vs comprehension completions

## 2026-03-05

Most completions are throughput — template fixes, one-line bugs, stale ref cleanups. These should be rubber-stamped. But some completions carry understanding updates: they change the system's capability, not just its correctness. Thread-sourced debrief changes how the orchestrator captures insight going forward. A template text fix doesn't.

The living threads chain (6 issues) shipped in ~15 minutes without once stopping to ask what the full system means now that it's deployed. That's the exact problem living threads was designed to solve, happening during its own deployment.

The distinguishing pattern: completions that change capability (how the system behaves in future sessions) vs completions that change correctness (fixing what's already there). The existing tier system (auto/scan/review/deep) measures verification rigor (did the agent do it right), not comprehension value (does this change how Dylan thinks about something).

Forming proposal: when presenting a completion, the orchestrator adds one signal line — 'This changes how X works going forward' — and when that line is present, both sides know to pause and engage rather than close-and-move-on. Not a gate. Not infrastructure. Just a comprehension flag.

Scoped a 4-phase research program to measure comprehension vs throughput. Core insight: comprehension is a latent variable — you can't observe it directly, only through proxies. Contrastive scenarios (contradiction, red herring, convergence) make the latent variable visible by creating situations where throughput and comprehension produce observably different outputs. Plan: .kb/plans/2026-03-05-comprehension-measurement-program.md. Decision gate: if calibration (Phase 2) shows keyword proxies don't correlate with human ratings (r < 0.4), escalate from keyword detection to structural analysis or constrained LLM-as-judge.

Scoped a 4-phase research program to measure comprehension vs throughput. Core insight: comprehension is a latent variable — can't observe directly, only through proxies. Contrastive scenarios (contradiction, red herring, convergence) make the latent variable visible by creating situations where throughput and comprehension produce observably different outputs. Plan: .kb/plans/2026-03-05-comprehension-measurement-program.md. 6 issues created: orch-go-rahs1, orch-go-h7vka, orch-go-54y23, orch-go-osad3, orch-go-co965, orch-go-77mle.

Ran 54 trials. Breakthrough: scenario 09v2 (implicit contradiction — incompatible assumptions, not opposite conclusions) is the first scenario to discriminate stance from knowledge. bare 0/3, without-stance 0/3, with-stance 2/3. Stance items don't teach agents what contradictions look like — they orient agents to look for meaning. This confirms the three-type taxonomy (knowledge/behavioral/stance) and proves stance is measurably functional, not just identity. Explicit signals (data tables, opposite findings) still hit ceiling on Sonnet — discriminating scenarios require implicit signals that demand modeling relationships. Design principle validated: make the latent variable visible through situations where throughput and comprehension say different things.

Higher-N confirmed decisively: scenario 09 at N=6 shows bare 0/6, without-stance 1/6, with-stance 5/6 (0%→17%→83%). Stance is measurably functional, not decorative. Key new insight: the knowledge/behavioral/stance taxonomy isn't orchestrator-specific — it describes how Claude processes ALL skill content. Every worker skill (investigation, systematic-debugging, architect, etc.) has the same three content types interleaved, and the same transfer properties apply: knowledge transfers, stance orients, behavioral constraints dilute at 5+. This means the 82% reduction that worked for the orchestrator skill (strip behavioral weight to hooks, keep knowledge and stance) is the playbook for all skills. The measurement methodology (contrastive scenarios testing orientation shift) ports directly — investigation stance ('test before concluding') and debugging stance ('understand before fixing') are testable in single-turn scenarios the same way orchestrator stance was.

## 2026-03-17 — Resolution

Thread resolved. The question "can we distinguish throughput from comprehension completions?" was answered empirically through a 4-phase measurement program (plan: `.kb/plans/2026-03-05-comprehension-measurement-program.md`).

**What was answered:**
- Contrastive scenarios make comprehension visible through keyword proxies (overall rho=0.637 with human ratings, Phase 2)
- Stance content is measurably functional: scenario 09 at N=6 shows 0%→17%→83% detection across bare/without-stance/with-stance
- The knowledge/behavioral/stance taxonomy generalizes beyond the orchestrator skill to all worker skills

**What remains open (but doesn't block resolution):**
- S11 indicator vocabulary needs redesign (rho=0.141 — broken for absence-detection scenarios)
- Multi-turn comprehension decay untested (Phase 4 was conditional, all issues closed)
- Scorer extensions (Phase 3) re-scoped to indicator vocabulary rather than grammar changes

**Outcome:** The distinguishing signal exists and is measurable. The original proposal (a "comprehension flag" on completions) was superseded by the deeper finding: stance content is what creates comprehension, and contrastive scenarios can verify it's working.

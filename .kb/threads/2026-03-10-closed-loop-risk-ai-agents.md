---
title: "Closed loop risk — AI agents reinforce coherent framing, internal consistency is not external validation"
status: resolved
created: 2026-03-10
updated: 2026-03-17
resolved_to: "Problem named, five instances documented, post-mortem complete. Uncontaminated Codex gate adopted (.kb/decisions/2026-03-10-adopt-uncontaminated-codex-gate-design-claim-ledg.md). Meta-lesson: independent assessment requires withholding the diagnosis, not just using a different model. Ongoing vigilance continues in evidence-quality thread (2026-03-15-evidence-quality-adversarial-grounding-against.md)."
---

# Closed loop risk — AI agents reinforce coherent framing, internal consistency is not external validation

## 2026-03-10

Recognized during plan review: the knowledge accretion theory was built by one person with AI agents that escalate observations into formalism. The formula (accretion_risk = f(...)) has no measured variables or units — it's a formula-shaped sentence. The falsifiability probe was run by our own agents within our framing. The deja vu signal: seeing a formula felt like a breakthrough, which is the same pattern as AI psychosis. Response: flip plan from publish-then-validate to validate-then-publish. Ship reproducible observation (coordination demo) first. Get kb-cli in front of one external user for independent data. Park the essay until at least one external data point exists. The observations (file growth, orphan rates, merge conflicts) are real and mechanical. The theory wrapping them needs independent confirmation before publication.

Second instance: coordination demo ran N=10, Fisher's exact test, confidence intervals — full statistical rigor proving two uncoordinated agents produce merge conflicts. Nobody would dispute this. Multiple sessions treated it as 'the empirical anchor' and 'strongest shareable piece.' No agent or orchestrator asked 'would this surprise anyone?' The system has no adversarial layer — agents build on premises, never challenge them. The gap: rigor-shaped is not the same as informative.

Third instance / solution found: ran coordination demo draft through Codex CLI (OpenAI, outside the loop). Caught every issue in 30 seconds — 'the setup is almost engineered to produce merge conflicts,' 'the outcome was largely baked into the design,' 'the current draft oversells a contrived demo as a general result.' Validates external-model adversarial review as a gate. The system can now have a 'so what?' check that isn't dependent on Dylan's gut feeling, though it's still an LLM — it catches framing blindness, not deep theoretical errors.

Fourth and fifth instances: ran knowledge-accretion and harness-engineering models through Codex adversarial review. Both got the same verdict: 'strongest as internal operating model, weakest when it claims to be a general framework/discipline.' Key hits: (1) knowledge accretion is governance/coordination cost/institutional memory repackaged in new vocabulary — 'what does this add beyond existing concepts?' (2) harness engineering is software architecture + CI/policy enforcement + tech debt management with agent vocabulary — 'what is actually new here?' (3) Both models' evidence is endogenous: one system interpreting itself. (4) The 'substrate-independence' claim is over-unified — code bloat, knowledge orphans, and OPSEC leakage may rhyme rhetorically without sharing a mechanism. (5) Most 'discoveries' are restatements: unenforced policy decays, adding is cheaper than removing, refactoring without changing incentives regresses. Pattern is now clear: the system escalates observations into novel-sounding theory because agents optimize for coherence and building on premises. The VALUE is in the practice and tooling. The OVERCLAIM is in calling it new theory.

Post-mortem complete. Two Codex passes run: contaminated (given our diagnosis) produced over-engineered four-gate design that exhibited the same vocabulary inflation it was preventing. Uncontaminated (given only the blog post) independently found problems we missed — unfalsifiable 'agent failure is harness failure', genre-mixing, false compliance/coordination dichotomy, non-decision-grade measurements — and prescribed simpler fix: claim ledger + red-team memo + claim-label pass. Decision: adopt the uncontaminated design. Harness engineering post set to draft, pending push. The closed loop is now named, the gate is designed, and the meta-lesson is captured: independent assessment requires withholding the diagnosis, not just using a different model.

Publication abandoned. The adversarial gates (publish gate, claim-upgrade scanner) work mechanically but don't solve the deeper problem: the system that builds the theory also builds the critics. Dylan's trust break came from outside the system (Codex review), not from inside it. Mechanical self-skepticism is not a substitute for independent validation.

## 2026-03-17 — Resolution

Resolving this thread. The closed-loop risk is now a well-understood failure mode with five documented instances, a complete post-mortem, and an adopted countermeasure (uncontaminated Codex gate design per .kb/decisions/2026-03-10-adopt-uncontaminated-codex-gate-design-claim-ledg.md). The sibling "validation gap" thread resolved to the same gate. The deeper ongoing concern — whether the system can confront disconfirming evidence about itself — continues in the evidence-quality thread (2026-03-15-evidence-quality-adversarial-grounding-against.md), which has active entries through Mar 16 including a real incident where self-measurement triggered destructive feedback. That thread is the right venue for ongoing vigilance; this thread's core contribution (naming the closed-loop pattern and establishing external review as the break) is complete.

## Auto-Linked Investigations

- .kb/investigations/2026-03-23-inv-investigate-openclaw-external-api-surface.md
- .kb/investigations/archived/2025-12-26-inv-api-endpoint-api-agents-hangs.md
- .kb/investigations/synthesized/serve-performance/2026-01-04-inv-orch-serve-shows-closed-agents.md

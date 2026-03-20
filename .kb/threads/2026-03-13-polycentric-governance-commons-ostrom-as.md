---
title: "Polycentric governance of the commons — Ostrom as diagnostic framework for agent coordination"
status: resolved
created: 2026-03-13
updated: 2026-03-19
resolved_to: "Insights distributed across agent-trust-enforcement, architectural-enforcement, and harness-engineering models"
---

# Polycentric governance of the commons — Ostrom as diagnostic framework for agent coordination

## 2026-03-13

The system has multiple commons (codebase, knowledge base, context windows, Dylan's attention), each with different depletion dynamics and different governance institutions. Ostrom's design principles are a diagnostic lens, not an aspiration. The key gap: agents are ephemeral, so they can't participate in governance as persistent community members — Ostrom's Principle 3 (collective-choice arrangements) seems structurally impossible. But the knowledge system is the mechanism through which ephemeral agents DO participate in rule evolution: captured friction → kb entries → decisions → principles → infrastructure gates. The pipeline from agent experience to governance change is how Session Amnesia is reconciled with polycentric governance. Aspiration: strengthen this pipeline. The system is not an agent framework — it's polycentric governance of shared commons, where the knowledge system is how ephemeral agents participate in collective choice across sessions.

## 2026-03-19 — Resolution

Each insight from this thread has been operationalized in existing models. No remaining actionable claims.

| Thread Insight | Where It Landed | Model/Section |
|---|---|---|
| Multiple commons with different depletion dynamics | Harness engineering: codebase as shared resource, accretion as entropy | `.kb/models/harness-engineering/model.md` — compliance vs coordination failure |
| Policy vs enforcement separation | Agent trust enforcement: policy layer (WHAT) vs enforcement layer (HOW) | `.kb/models/agent-trust-enforcement/model.md` — §Policy Layer vs Enforcement Layer |
| Knowledge pipeline as collective-choice mechanism (friction → kb → decisions → principles → gates) | Harness engineering: soft harness feeds hard harness evolution; architectural enforcement: gate signaling drives extraction cascades | `.kb/models/harness-engineering/model.md` + `.kb/models/architectural-enforcement/model.md` |
| Ephemeral agents can't be persistent community members | Hard/soft harness taxonomy addresses this directly — hard harness persists governance decisions structurally so each ephemeral agent inherits them without needing memory | `.kb/models/harness-engineering/model.md` — Hard vs Soft |
| "Not an agent framework — polycentric governance of shared commons" | Framing validated by Codex external review: "software architecture + CI/policy enforcement + tech debt management with agent vocabulary" | `.kb/models/harness-engineering/model.md` — Validation Status |

The related thread "What kind of theory is this? — Ostrom-scale institutional analysis" (2026-03-10) was previously resolved to `.kb/models/knowledge-accretion/model.md`.

**Why resolved (not extended):** The Ostrom framing was generative — it identified the right structural questions. But the answers have been operationalized as concrete models with testable claims, which is more useful than maintaining an abstract framing thread. The models ARE the governance institutions this thread aspired to strengthen.

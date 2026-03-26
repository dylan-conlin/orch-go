# Probe: Minimum Open Release — Can Artifact Formats Alone Teach the Method?

**Date:** 2026-03-26
**Model:** knowledge-accretion
**Status:** Complete
**Spawned by:** orch-go-ehper

---

## Question

The openness boundary matrix (orch-go-pityd) recommends three-wave release: artifact formats first, CLI + API second, held-back surfaces third. The implicit claim is that artifact formats — threads, briefs, probes, models, decisions, investigations — are "already self-documenting and independently adoptable" (matrix Finding 2).

**Specific claims being tested:**
1. Artifact formats alone are sufficient to teach the method (the knowledge cycle) to a new user
2. Opening the full CLI/substrate before the core comprehension story is credible makes the product look like an orchestration CLI (the 16/72 perception risk)
3. There exists a minimum bundle smaller than "the whole repo" that teaches the method on first contact

## What I Tested

Evaluated all 7 artifact types (thread, brief, model, investigation, probe, decision, KB README) for standalone comprehensibility by examining real artifacts as a new user would encounter them — scoring each 1-5 for ability to understand purpose, create new instances, and navigate without CLI context.

Also audited the `orch init` first-contact flow against the README's comprehension-first promise, and counted orch-specific references (IDs, jargon, unexplained fields) across artifacts.

## What I Observed

**Artifact standalone comprehensibility (1-5 scale):**

| Artifact Type | Score | Key Blocker |
|---|---|---|
| Thread | 2/5 | 7 unexplained orch IDs in frontmatter, `spawned_from`/`active_work` opaque |
| Brief | 3.5/5 | Best format — Frame/Resolution/Tension is clear. Filename IDs confusing |
| Model | 2/5 | Dense cross-references, claim IDs (KA-05) unexplained, no creation guidance |
| Investigation | 3/5 | D.E.K.N. undefined, Phase/Authority fields jargon |
| Probe | 2/5 | Claim reference syntax opaque, verdict values undocumented |
| Decision | 4/5 | Most standalone — standard ADR pattern. Only Enforcement field is jargon |
| KB README | 2/5 | Documents 4 of 7+ artifact types. Threads and briefs missing entirely |

**Average: 2.6/5 — formats are NOT self-documenting enough for Wave 1 alone.**

**29+ orch-specific references** across artifacts (IDs, unexplained metadata fields, system jargon) that become noise without the full system.

**Critical gaps identified:**
- No composition model documented anywhere (how thread → investigation → probe → model → brief)
- KB README omits threads and briefs (the two most method-defining artifacts)
- Frontmatter is inconsistent across types (some YAML, some inline, some none)
- `orch init` "Next steps" lead with execution ("spawn an agent") while README promises comprehension

## Model Impact

**Contradicts** the matrix's claim (Finding 2) that "All artifact formats should be open immediately. They are markdown-based, human-readable, and self-documenting." The formats are markdown-based and human-readable, but they are NOT self-documenting in isolation — they require a composition guide, expanded KB README, and examples to be independently adoptable. This means Wave 1 (artifact formats alone) is insufficient. The minimum open release must include artifact formats + composition documentation + at minimum the thread command surface.

**Extends** the knowledge-accretion model's core mechanism: the 85.5% orphan rate among investigations is partly explained by the fact that the composition model (how artifacts relate to each other) is implicit in the CLI tooling and never documented. Agents and humans alike don't link artifacts because the linking relationships aren't visible in the artifacts themselves.

**Confirms** the 16/72 perception risk: the `orch init` flow currently reinforces the execution-first framing by suggesting "spawn an agent" as step 3, before any thread creation or comprehension surface exposure.

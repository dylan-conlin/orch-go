---
title: "Writing soft harness — from primers to a skill that enforces story-first, earned abstractions, emotional honesty"
status: forming
created: 2026-03-20
updated: 2026-03-28
resolved_to: ""
---

# Writing soft harness — from primers to a skill that enforces story-first, earned abstractions, emotional honesty

## 2026-03-20

Context: 4 stance-level writing primers exist in memory (story first, earn the abstraction, say what it felt like, the turn is the piece). Derived from diagnosis of harness engineering blog post (0 HN traction, 11 tables, framework before story). No writing skill exists yet.

Open questions:
1. What would a writing skill look like? The primers are stance (attention primers), not behavioral (MUST/NEVER). A skill that says 'write with story first' will get ignored the same way any reminder gets ignored. Needs to be structural.
2. How do you gate writing quality? Code has tests. What does a 'test' for writing look like? The skillc behavioral testing framework (variant vs bare, detection patterns) could measure whether the skill changes output distribution — but what detection patterns capture 'story first' vs 'framework first'?
3. The compositional correctness parallel: individual paragraphs can be well-written while the piece fails at the composition level (structure, arc, turn placement). Component gates (grammar, clarity) miss composition failures (does the piece have a turn? does the abstraction come after the story?).
4. Connection to today's session: this IS a compositional correctness problem. Each section of the blog post was technically correct. The composition was wrong — framework before story, self-correction buried in section 4.

Architect completed (orch-go-npm1s). Design: 4-phase technical-writer skill (Story Discovery → Draft → Composition Review → Revision). Key innovation: composition self-audit with quote-based evidence — 'quote the turn sentence' not 'is there a turn?' Same pattern as D.E.K.N. / VERIFICATION_SPEC. 4 implementation follow-ups pending: (1) skill skeleton, (2) test scenarios, (3) harness engineering rewrite, (4) update writing-style model status.

# Probe: Can the Composition Cycle Be Documented as a Short Guide That Teaches the Method?

**Model:** Knowledge Accretion
**Date:** 2026-03-28
**Status:** Complete
**Claim Under Test:** The minimum open release probe (2026-03-26) claimed a composition guide is "the single most important missing artifact" and that artifact formats alone score 2.6/5 on standalone comprehensibility. The thread 'intent-arrives-without-shape' (2026-03-27) added a constraint: the guide should open with shape classification, not the cycle diagram.

## Question

Can the full method (shape classification → composition cycle → artifact types) be documented in 2-3 pages using real cleaned examples, and does leading with shape classification produce a more legible entry point than leading with the cycle diagram?

## What I Tested

Authored `.kb/GUIDE.md` — the method guide referenced by the release bundle investigation. Applied three constraints simultaneously:
1. Open with shape classification (4 work shapes), not the cycle diagram
2. Keep total length under 3 pages (~100 lines of markdown)
3. Use cleaned real artifacts from orch-go (IDs replaced with descriptive labels)

Source artifacts consulted:
- Release bundle investigation (2026-03-26): defined the guide as must-ship
- Thread 'intent-arrives-without-shape' (2026-03-27): shape-first design constraint
- Behavioral grammars model: "skills are a way of thinking" — shape taxonomy is the routing grammar made legible
- Core method spec: sacred/default/negotiable boundaries
- 3 real briefs (cleaned for examples): sispn, pityd, wgkj4
- 1 real thread entry: threads-as-primary-artifact

## What I Observed

- The guide fits in ~90 lines of markdown. The constraint "if it's longer than 3 pages, the cycle is too complex" holds — the cycle is NOT too complex, it compresses cleanly.
- Opening with shape classification creates a natural transition: "you arrive with intent → notice its shape → enter the cycle at the right point." Without shape classification, the cycle diagram is a taxonomy you read; with it, the diagram is a map you use.
- One cleaned brief example (Frame/Resolution/Tension) communicates more about the method than the paragraph describing briefs. The example is the teaching.
- The decision to lead with shape rather than cycle resolved a composition question: the cycle diagram answers "how do artifacts relate?" but doesn't answer "where do I start?" Shape classification answers the starting question.

## Model Impact

**Confirms:** The composition guide IS the missing link claimed by the release bundle probe. Writing it revealed no hidden complexity — the cycle compresses to a diagram + 6 short paragraphs. The 2.6/5 standalone comprehensibility score for artifact formats is explained by the absence of this guide, not by format complexity.

**Extends:** The thread's design constraint (shape-first) is not just a UI preference — it's load-bearing. Without shape classification, the guide is a reference document (look up what artifacts are). With it, the guide is a routing tool (figure out what to do with your intent). This distinction maps to behavioral grammar Claim 7: the guide should develop the human's routing grammar, not just describe the system's.

**Does not test:** Whether a new user reading this guide can actually create a thread, run the cycle, and feel the difference. That requires user testing after the guide ships.

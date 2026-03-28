# Session Synthesis

**Agent:** og-arch-write-method-guide-28mar-747f
**Issue:** orch-go-tse6y
**Duration:** 2026-03-28
**Outcome:** success

---

## Plain-Language Summary

The release bundle investigation identified a "method guide" as the single most important missing artifact — a document explaining how threads, investigations, probes, models, briefs, and decisions relate to each other. A design constraint from the 'intent-arrives-without-shape' thread said the guide should open with work shape classification (search, convergence, forming, execution), not the cycle diagram. I wrote that guide at `.kb/GUIDE.md` in 80 lines. The key discovery was that shape classification is load-bearing: without it, the guide is a reference document you look things up in; with it, it's a routing tool that helps you figure out what to do with your intent. The cleaned brief example communicates more about the method than any paragraph could.

## TLDR

Wrote `.kb/GUIDE.md` — the method guide that opens with shape classification (search/convergence/forming/execution), then documents the composition cycle with one paragraph per artifact type and a real cleaned brief example. 80 lines, well within the 2-3 page constraint.

---

## Delta (What Changed)

### Files Created
- `.kb/GUIDE.md` — Method guide (rewrote existing file with shape-first structure)
- `.kb/models/knowledge-accretion/probes/2026-03-28-probe-composition-guide-teaches-method.md` — Probe confirming composition cycle compresses to 2 pages
- `.orch/workspace/og-arch-write-method-guide-28mar-747f/VERIFICATION_SPEC.yaml` — Verification spec
- `.orch/workspace/og-arch-write-method-guide-28mar-747f/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-write-method-guide-28mar-747f/BRIEF.md` — Brief

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Merged probe findings: updated Last Updated date, added probe reference to Probes section, added Evolution entry, extended Open Question #7 with guide as accelerator for time-to-magic-moment

---

## Evidence (What Was Observed)

- The composition cycle compresses to a diagram + 6 short paragraphs without losing essential information
- The existing GUIDE.md was 143 lines and opened with the cycle diagram — the new version is 80 lines and opens with shape classification
- Artifact standalone comprehensibility (2.6/5 from release bundle investigation) is explained by the absent composition guide, not by format complexity
- The brief example (from orch-go-wgkj4, cleaned of IDs) teaches the method better than any abstract description
- The four shapes (search, convergence, forming, execution) map cleanly to cycle entry points

---

## Architectural Choices

### Shape-first vs cycle-first ordering
- **What I chose:** Open with shape classification table, then cycle diagram
- **What I rejected:** The previous version's cycle-first ordering
- **Why:** Thread 'intent-arrives-without-shape' identified this as a product concern: the shape classification moment is the actual front door. The cycle answers "how do artifacts relate?" but doesn't answer "where do I start?" — shape answers the starting question.
- **Risk accepted:** A new user might not understand what the shapes mean without having used the system. The shapes may need to be learned through experience, not taught through a table.

### Cleaned real example vs multiple examples
- **What I chose:** One cleaned brief (the 16/72 codebase inventory) as the sole example
- **What I rejected:** Including examples of every artifact type, or using synthetic examples
- **Why:** The page budget constraint (2-3 pages) forced a choice. The brief is the most self-explanatory artifact (Frame/Resolution/Tension is immediately legible). One strong example teaches more than six weak ones.
- **Risk accepted:** Users don't see what threads, probes, or models look like. The curated examples task from the release bundle investigation should provide those.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 7 automated checks, 2 manual checks pending.

---

## Knowledge (What Was Learned)

### Probe Findings (merged to model)
- Composition cycle compresses to 80 lines — the "if it's longer than 3 pages, the cycle is too complex" constraint holds easily
- Shape classification is load-bearing, not decorative — it determines whether the guide is a reference document or a routing tool
- This extends Open Question #7 in the knowledge-accretion model: the guide may accelerate time-to-magic-moment for new users

---

## Next (What Should Happen)

**Recommendation:** close

- [ ] All deliverables complete (guide, probe, model merge, SYNTHESIS, BRIEF, VERIFICATION_SPEC)
- [ ] Ready for `orch complete orch-go-tse6y`

### Follow-on work (from release bundle investigation, not created as issues — they already exist):
1. Rewrite `orch init` "Next steps" to reference `.kb/GUIDE.md`
2. Curate 2-3 example artifacts per type
3. Expand KB README to document all 7 artifact types

---

## Unexplored Questions

- Whether shape classification is learnable from a table vs. only through exposure to one-sentence classifications in system responses (the behavioral grammar Claim 7 question)
- Whether the guide needs to address the concern from the release bundle investigation: "threads alone without agent evidence might feel like a glorified notes tool"

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-write-method-guide-28mar-747f/`
**Beads:** `bd show orch-go-tse6y`

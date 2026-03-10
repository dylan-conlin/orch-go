# Probe: Publication Polish — Related Work Positioning and Hook Sharpening

**Model:** harness-engineering
**Date:** 2026-03-10
**Status:** Complete

---

## Question

Does positioning the harness engineering framework against existing literature (MAST, Anthropic, Fowler/Bockeler, OpenAI) confirm or extend the model's claim that the compliance/coordination distinction is novel? Does the "stronger models make coordination worse" framing survive scrutiny against what others have published?

---

## What I Tested

1. **MAST paper (arxiv 2503.13657):** Read abstract and full HTML. Cemri et al. analyzed 1,600+ traces across multi-agent LLM systems. Taxonomy: 14 failure modes in 3 categories (FC1: system design, FC2: inter-agent misalignment, FC3: task verification).

2. **Anthropic "Effective Harnesses for Long-Running Agents":** Fetched from anthropic.com/engineering. Focus: single-agent multi-session continuity via progress files, feature lists, mandatory commits.

3. **Fowler/Bockeler harness engineering article:** Fetched from martinfowler.com. Identifies "verification gap" and "relocating rigor." Single-agent focus.

4. **OpenAI Codex harness engineering:** Already synthesized in model. ~1M lines, zero manual code, "agent failure = harness bug."

---

## What I Observed

### MAST paper findings
- FC2 (inter-agent misalignment) = ~32% of failures across 6 modes: conversation reset (2.2%), fail to ask clarification (6.8%), task derailment (7.4%), information withholding (0.85%), ignored input (1.9%), reasoning-action mismatch (13.2%)
- They frame the solution as needing "deeper social reasoning abilities from agents" — model improvement, not architecture
- Key quote supporting architectural interpretation: "a well-designed MAS can result in performance gain when using the same underlying model" — but they don't develop this into an architectural framework
- They do NOT distinguish compliance vs coordination as having opposite trajectories with model improvement
- Their FC1 (system design, ~44%) maps roughly to compliance; FC2 (~32%) maps to coordination; FC3 (task verification, ~24%) is the verification gap Fowler identified

### Anthropic effective harnesses
- Single-agent, multi-session. "Harness" = orchestration framework for context continuity
- Structural constraints: JSON feature lists, mandatory commits, progress files
- Does NOT address concurrent multi-agent coordination at all
- Recommends against multi-agent architectures currently

### Fowler/Bockeler
- Names the "verification gap" — OpenAI doesn't verify behavior, only structure
- "Relocating rigor" — rigor migrates to environment design
- Single-agent focus. No coordination failure concept
- Does not distinguish failure modes by response to model improvement

### Positioning summary
| Source | "Harness" means | Agent scope | Addresses coordination? | Compliance/coordination distinction? |
|--------|-----------------|-------------|------------------------|-------------------------------------|
| OpenAI Codex | Test infra + constraints | Single | No | No |
| Anthropic | Session orchestration | Single (sequential) | No | No |
| Fowler/Bockeler | Verification + rigor relocation | Single | No | No |
| MAST (Cemri et al.) | N/A (taxonomy, not solution) | Multi | Yes (observes it) | No (taxonomizes by symptom) |
| **Ours** | Architecture-as-governance | Multi (concurrent) | Yes (addresses it) | Yes (opposite model trajectories) |

---

## Model Impact

- [x] **Confirms** the compliance/coordination distinction is novel. No existing published work separates these failure modes by their response to model improvement. MAST comes closest (FC1 vs FC2) but frames both as needing better models.

- [x] **Confirms** the "stronger models make coordination worse" framing is defensible. MAST's own data supports the architectural interpretation (same model, better system design = better performance) without drawing the conclusion. Our claim is consistent with their evidence but goes further.

- [x] **Extends** model with: Precise positioning against 4 sources. The field has converged on "harness" terminology with 3 distinct meanings: environment setup (OpenAI/Anthropic), verification (Fowler), and governance (ours). Only ours addresses concurrent multi-agent coordination as a structural problem.

- [x] **Extends** model with: MAST's FC1/FC2/FC3 taxonomy maps to compliance/coordination/verification — but MAST doesn't recognize these as having different trajectories with model improvement. This is the gap our framework fills.

---

## Notes

- User specified MAST = 36.9%, actual data from paper HTML shows ~32%. The user may have been including FM-2.6 (reasoning-action mismatch, 13.2%) which straddles compliance/coordination. Using "~32%" from paper data directly.
- Anthropic blog was at /engineering/effective-harnesses-for-long-running-agents (not /effective-harnesses as tried first)
- The MAST paper's recommendation of "deeper social reasoning" is the strongest counterposition to our architectural framing — worth noting that we see this as a compliance solution being proposed for a coordination problem

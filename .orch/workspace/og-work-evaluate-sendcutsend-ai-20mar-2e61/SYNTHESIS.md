# Session Synthesis

**Agent:** og-work-evaluate-sendcutsend-ai-20mar-2e61
**Issue:** orch-go-hw6ej
**Duration:** 2026-03-20
**Outcome:** success

---

## Plain-Language Summary

SendCutSend's AI Part Builder is a beta feature powered by Smithy (api.smithy.cc) that generates parametric 3D sheet metal parts from text prompts in ~15 seconds. It handles flat parts, bends, and even recognizes domain-specific vocabulary like "PEM nuts." However, it performs zero DFM (design for manufacturability) validation — you can generate parts with PEM hardware placed right on a bend line and get no warning. The tool confirms our compositional correctness gap hypothesis: per-feature generation works (holes OK, bends OK), but cross-feature interaction validation (hardware near bends = manufacturing failure) doesn't exist. The gap lives at the integration boundary between Smithy's geometry generation and SCS's manufacturing validation, separated by a one-way gate that prevents going back to fix geometry after discovering constraints.

## TLDR

Evaluated SendCutSend's AI Part Builder (beta, powered by Smithy). It generates parametric 3D parts from text prompts with impressive speed and domain vocabulary recognition, but has zero DFM validation — confirms the compositional correctness gap at the boundary between AI geometry generation and manufacturing constraint checking.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-20-eval-sendcutsend-ai-part-builder.md` — Full experiential evaluation with 4 test cases

### Files Modified
- None

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **Test 1 (flat bracket):** Text → parametric 3D model in ~15s. Parameters: width/height/fillet/holes. Full flow works: Template → Customize → Material → Services → Finishing.
- **Test 2 (U bracket with bends):** AI generates post-bend 3D geometry. Parameters include base width, flange height, part depth. No bend radius/K-factor exposed.
- **Test 3 (PEM nuts + bends — DFM conflict):** AI recognized "PEM nuts" and created "Pem hole" parameter. Generated holes near bend line with **zero DFM warnings**. This is the key compositional correctness gap.
- **Test 4 (vague "electronics enclosure"):** Generated reasonable default box (120×80×30mm) without asking clarifying questions about PCB size, connectors, IP rating, etc.
- **Integration architecture:** Smithy iframe (api.smithy.cc) → STEP file → SCS quoting pipeline. One-way gate: "Model cannot be edited after continuing."
- **Generated files stored as:** `ai_part_builder_{hash}.step` in user's parts library

### Tests Run
```bash
# Browser automation tests via playwright-cli
# 4 text-to-3D generation tests completed
# Full quoting pipeline walked through for Test 1
```

---

## Architectural Choices

No architectural choices — this was an experiential evaluation, not implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-20-eval-sendcutsend-ai-part-builder.md` — Experiential evaluation of SCS AI Part Builder

### Constraints Discovered
- SCS AI Part Builder is powered by Smithy (api.smithy.cc), a third-party service embedded via iframe
- DFM validation gap exists at the integration boundary between Smithy (geometry) and SCS (manufacturing)
- One-way commit gate prevents geometry edits after entering the quoting pipeline
- The AI generates per-feature geometry without cross-feature interaction validation

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — evaluation deliverables are the investigation file and screenshots.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with 4 test cases)
- [x] Tests passing (N/A — experiential eval, no code)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-hw6ej`

---

## Unexplored Questions

- **How does Smithy's parametric model generation actually work?** Is it a fine-tuned diffusion model for CAD? A code-generation model that outputs OpenSCAD/CadQuery? The 3D output quality and parameter naming suggest something more sophisticated than mesh generation.
- **Does SCS's quoting pipeline catch the DFM issues the AI builder misses?** I walked through to material selection but didn't complete an actual order. The DFM check may happen later (during production review).
- **What's the Smithy API pricing model?** If SCS is paying per-generation, the lack of DFM pre-validation means they're paying for geometry that their own pipeline will reject.
- **Could the refinement prompt ("What do you want to change?") be used for DFM corrections?** If SCS fed DFM violations back into the refinement loop, the AI could fix issues before the one-way commit.

---

## Friction

- **tooling:** playwright-cli not installed globally, required `npm install -g @playwright/cli`. ~2 minutes lost.
- **tooling:** Smithy iframe not captured in Playwright's accessibility snapshots, requiring direct `frame.$()` calls for all interactions. ~10 minutes of extra debugging to find the right frame/elements.
- **bug:** Part Builder click opens "Request Custom Quote" modal that intercepts clicks. Required manual modal removal via JS. Recurring friction across multiple Part Builder open attempts.

---

## Session Metadata

**Skill:** experiential-eval
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-work-evaluate-sendcutsend-ai-20mar-2e61/`
**Investigation:** `.kb/investigations/2026-03-20-eval-sendcutsend-ai-part-builder.md`
**Beads:** `bd show orch-go-hw6ej`

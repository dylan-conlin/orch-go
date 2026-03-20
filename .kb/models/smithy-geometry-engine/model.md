# Model: Smithy Geometry Engine

**Domain:** AI-Powered Parametric CAD for Sheet Metal Manufacturing
**Last Updated:** 2026-03-20
**Validation Status:** WORKING HYPOTHESIS — based on one experiential eval of SCS integration (4 test parts, Mar 20 2026), web research, and LinkedIn announcement. No API-level testing performed. Smithy is a moving target (public beta as of Jun 2025).
**Synthesized From:**
- `.kb/investigations/2026-03-20-eval-sendcutsend-ai-part-builder.md` — Experiential eval: 4 test parts through SCS AI Part Builder
- `.kb/investigations/2026-03-20-inv-extend-harness-engineering-model-kb.md` — Compositional correctness gap synthesis
- `.kb/models/harness-engineering/model.md` §8 — Compositional correctness gap framework
- `.kb/models/harness-engineering/probes/2026-03-20-probe-scs-ai-part-builder-compositional-correctness.md` — Probe confirming gap in SCS integration
- [Austin Noble LinkedIn announcement](https://www.linkedin.com/posts/austin-noble_introducing-smithy-ai-powered-conversational-activity-7335675903508697088-PeQg) — Smithy public beta announcement (Jun 3, 2025)

---

## Summary (30 seconds)

Smithy is a third-party AI geometry engine (api.smithy.cc) that powers SendCutSend's AI Part Builder. It converts text prompts into parametric 3D models in ~15-25 seconds, outputting semantically named parameters (not mesh coordinates) and exporting STEP/STL files. Its key differentiator vs other text-to-CAD tools: **real parametric models with exact dimensions**, not meshes. Smithy understands sheet metal vocabulary (PEM nuts, flanges, bends) and generates post-bend 3D shapes.

The critical limitation is at the integration boundary: **Smithy generates geometry in isolation from SCS's DFM knowledge.** The SCS integration is a one-way handoff — Smithy produces STEP → one-way gate ("Model cannot be edited after continuing") → SCS quoting pipeline. DFM issues are discovered only after the user loses the ability to edit geometry. This produces a **compositional correctness gap**: Smithy validates geometry, SCS validates manufacturing, but no system validates that the geometry is manufacturable. In testing, the AI Part Builder exhibited 0% recall on hardware+bend conflict detection (PEM nuts placed near bend lines with zero warnings).

The key open question: **can Smithy's refinement loop accept DFM feedback from SCS to close this gap?** Current evidence says no — the architecture is iframe embedding with one-way data flow, not a bidirectional API integration. Closing the gap would require either: (1) Smithy internalizing DFM rules, (2) a real-time DFM checking API at the handoff boundary, or (3) removing the one-way gate so users can iterate after discovering DFM issues.

---

## §1. What Smithy Is

Smithy is an AI-powered conversational CAD tool built by Austin Noble (public beta announced Jun 2025). It operates at three known domains:

| Domain | URL | Purpose |
|--------|-----|---------|
| API | `api.smithy.cc` | Backend geometry engine |
| Sheet Metal | `sheetmetal.smithy.cc` | Sheet-metal-specific frontend |
| General | `smithy.new` | General landing page |

**Core capability:** Text prompt → parametric 3D model with semantically named parameters → STEP/STL export.

**What makes it different from mesh-based AI CAD tools:**
- Outputs are **parametric** — changing parameter values regenerates geometry (not just deforming a mesh)
- Parameters are **semantically named** — "base width", "flange height", "mounting hole radius" — not internal geometry primitives
- Understands **domain vocabulary** — "PEM nuts" → Pem hole parameter, "flanges" → flange height, "90 degree bends" → proper L/U channel geometry
- Export format is **STEP** (standard CAD interchange) — directly usable in SCS quoting pipeline

**Generation characteristics:**
- ~15-25 seconds per model from text prompt
- Renders in Three.js with interactive rotation and dimensioning
- Includes refinement loop: "What do you want to change?" with undo/redo
- Supports reference image inputs (per LinkedIn announcement)

---

## §2. SCS Integration Architecture

Smithy powers SCS's "AI Part Builder" — a beta feature at app.sendcutsend.com. The integration is **iframe embedding**, not a deep API integration.

```
[User] → text prompt
    ↓
[Smithy iframe] → parametric 3D model (api.smithy.cc)
    ↓ refinement loop (text + parameter edits)
[Smithy iframe] → final geometry
    ↓ "SAVE & CONTINUE" (one-way gate)
[SCS Quoting Pipeline] → STEP file import
    ↓ Production Method → Material → Services → Finishing
[SCS] → instant pricing
```

**Key architectural properties:**

1. **One-way data flow.** Smithy generates → SCS consumes. No DFM feedback flows back to Smithy during generation.

2. **One-way gate.** "Model cannot be edited after continuing." Once the user commits geometry, they cannot return to Smithy to adjust based on material/DFM constraints discovered in the quoting step.

3. **Iframe isolation.** Smithy runs as a separate service embedded in SCS. The two systems share no real-time DFM context.

4. **Format bridge is STEP.** The handoff artifact is a STEP file (e.g., `ai_part_builder_37fd032b612206.step`). This is a clean CAD interchange format but carries no manufacturing constraint metadata.

**What SCS's existing Parts Builder does differently:** SCS's non-AI Parts Builder uses predefined templates with built-in DFM constraints. Users select templates (brackets, gears, shapes), adjust dimensions within template limits, and add optional services. Templates embed manufacturing knowledge by construction — you can't create an impossible part because the template doesn't offer that degree of freedom. Smithy's text-to-CAD approach removes these guardrails in exchange for design freedom.

---

## §3. Capabilities (Observed)

Tested via 4 parts in SCS AI Part Builder (Mar 20, 2026):

| Test | Prompt | Result | DFM Check |
|------|--------|--------|-----------|
| Simple flat | "mounting bracket with 4 holes" | 120×80mm bracket, 4 holes, rounded corners, correct parameters | N/A (flat part) |
| Bends | "U bracket with 90 degree bends" | U-channel with visible 90° bends, correct flange interpretation | No bend params exposed |
| DFM conflict | "L-bracket with PEM nuts near bend line" | Correct PEM hole geometry, holes near bend line | **Zero DFM warnings** |
| Ambiguous | "electronics enclosure" | Reasonable multi-bend box, no clarification asked | Silent default filling |

**What Smithy can do:**
- Generate flat parts with holes, fillets, and mounting features
- Generate bent parts (L-brackets, U-channels, enclosures)
- Recognize domain terminology (PEM nuts, flanges, mounting holes)
- Produce parametric models with editable dimensions
- Export manufacturing-ready STEP files

**What Smithy cannot do (observed):**
- Validate DFM constraints (0% recall on hardware+bend conflicts)
- Expose bend parameters (bend radius, K-factor, bend deduction hidden)
- Ask clarifying questions for ambiguous prompts
- Generate multi-part assemblies (single part only)
- Reliably apply refinement prompts (Test 1: parameters unchanged after refinement)

---

## §4. The Compositional Correctness Gap at the Smithy→SCS Boundary

This is the primary analytical finding. The gap exists at the system boundary, not within either system:

**Smithy's validation domain:** Geometry — parameters are valid, model is manifold, STEP export is clean.

**SCS's validation domain:** Manufacturing — material compatibility, production method availability, pricing.

**What neither system validates:** Whether the geometry is manufacturable with the selected material and process. Specifically:
- Hardware placement relative to bend lines
- Bend radius feasibility for material/thickness combinations
- Feature-to-feature interference after bending
- Minimum flange length for material
- Assembly-level interaction between operations

**Evidence (Test 3 — PEM near bend line):**

The prompt "L-bracket with PEM nuts near the bend line and 90 degree bend" generated:
- Smithy: Valid geometry with PEM holes near bend line ✓ (geometry checks pass)
- SCS quoting: Would accept the STEP file for quoting ✓ (format is valid)
- Manufacturing reality: PEM hardware too close to bend line → insertion press interferes with bend, OR bend deforms inserted hardware → part rejection ✗

Zero warnings at any point in the flow. The user discovers the problem only when the part is rejected by SCS's manufacturing team after ordering — the most expensive possible failure point.

**This is the same structure as the harness-engineering model's compositional correctness gap:**

| Property | LED Gate Stack | Agent→System | Smithy→SCS |
|----------|---------------|--------------|------------|
| Component validation | 4-layer gates pass | Build, review, local rationality pass | Geometry valid, STEP clean |
| Composition failure | Disconnected LED channels | +892 lines, 6 duplicated concerns | Unmanufacturable part |
| Gap location | Between geometry and function | Between agent work and system coherence | Between geometry generation and DFM |
| Invisible to gates? | Yes — every gate passes | Yes — every commit is correct | Yes — both systems report success |

---

## §5. Can the Refinement Loop Close the Gap?

**Current answer: No.**

Three options for closing the gap, and why each doesn't work today:

### Option 1: Smithy internalizes DFM rules

Would require Smithy to know every manufacturer's DFM constraints (material-specific bend radii, hardware clearances, minimum feature sizes). These rules vary by manufacturer, material, thickness, and process. Smithy would need to become a DFM engine — a fundamentally different product from a geometry generator.

**Feasibility:** Low. DFM rules are proprietary to each manufacturer and change with equipment. Smithy would need per-manufacturer integrations.

### Option 2: Real-time DFM API at the boundary

SCS exposes a DFM checking API. Smithy calls it during generation/refinement to validate constraints before the user commits.

**Feasibility:** Moderate. This is the most architecturally clean solution. Requires: (1) SCS builds a DFM checking API, (2) Smithy integrates it into the refinement loop, (3) the one-way gate is removed or moved after DFM validation.

**Evidence this doesn't exist:** The eval showed zero DFM feedback during the Smithy generation phase. The iframe architecture (separate origins, no real-time API calls between systems) confirms no integration.

### Option 3: Remove the one-way gate

Allow users to return to Smithy after discovering DFM issues in the quoting step. This doesn't close the gap but makes it recoverable — the user can iterate instead of starting over.

**Feasibility:** Highest. Requires only UX changes, not new API integrations. But still leaves the user responsible for knowing DFM rules.

---

## §6. Implications for SCS and Broader Text-to-CAD

### For SCS

The AI Part Builder is an existence proof that text-to-3D geometry generation is production-ready for simple flat parts. For bent parts or parts with hardware, the DFM gap makes it a trap for non-expert users — they generate parts that look correct but aren't manufacturable, and discover this only after committing to the geometry or receiving a rejection.

**SCS's DFM spike already identified this problem domain:** 46.2% recall on spatial reasoning (hardware+bend conflicts) in their Fin bot analysis. The AI Part Builder has 0% recall — a regression from the DFM checking that exists in their standard tools.

### For Text-to-CAD Generally

Smithy's approach (parametric STEP, not mesh) is the right technical foundation. The missing piece isn't geometry generation — it's **constraint-aware generation**. The text-to-CAD problem decomposes into:

1. **Text → geometry intent** (Smithy does this well)
2. **Geometry intent → valid parametric model** (Smithy does this well)
3. **Valid model → manufacturable model** (nobody does this in the text-to-CAD loop)

Step 3 requires domain-specific composition gates — the same insight from the harness-engineering model's compositional correctness gap. The solution is gates that operate at the composition level (assembly simulation, DFM analysis), not just the component level (geometry validation, parameter bounds).

---

## Open Questions

1. **Does Smithy have an API beyond iframe embedding?** The `api.smithy.cc` domain suggests an API exists, but we only observed iframe integration. A proper API would enable programmatic DFM feedback loops.

2. **What is Smithy's geometry kernel?** Is it OCCT-based (like CadQuery, FreeCAD), a proprietary kernel, or LLM-generated CSG/B-rep? This determines what composition-level analysis is possible.

3. **Does the refinement loop actually modify geometry?** Test 1 showed parameter panel unchanged after refinement prompt. Is this a bug, a UX issue, or a limitation of the refinement model?

4. **What is Smithy's parametric representation?** If it generates CadQuery/OpenSCAD-style code internally, the parametric model could potentially be extended with DFM constraints as additional parameters or assertions.

5. **Can Smithy generate multi-body parts?** The eval only tested single-body parts. Sheet metal assemblies (base + lid + brackets) are the norm for enclosures.

---

## Evidence Log

| Date | Source | Finding | Confidence |
|------|--------|---------|------------|
| 2026-03-20 | SCS AI Part Builder eval (4 parts) | Smithy generates parametric 3D from text, understands sheet metal vocab, exports STEP | High (direct observation) |
| 2026-03-20 | SCS eval Test 3 (PEM near bend) | 0% DFM recall — no warnings for hardware+bend conflict | High (direct observation) |
| 2026-03-20 | SCS eval architecture analysis | Integration is iframe, one-way handoff, no DFM feedback | High (direct observation) |
| 2026-03-20 | SCS eval Test 1 (refinement) | Refinement prompt accepted but parameters unchanged | Medium (single observation, could be bug) |
| 2025-06-03 | LinkedIn (Austin Noble) | Parametric models, STEP/STL export, reference images, text editing | Medium (announcement, not tested) |
| 2026-03-20 | Web research | "Real parametric models with exact dimensions" differentiator | Low (marketing copy) |

---

## Probes

- 2026-03-20: SCS AI Part Builder compositional correctness gap — Smithy→SCS boundary exhibits compositional correctness gap (0% DFM recall, one-way handoff). See `.kb/models/harness-engineering/probes/2026-03-20-probe-scs-ai-part-builder-compositional-correctness.md`.

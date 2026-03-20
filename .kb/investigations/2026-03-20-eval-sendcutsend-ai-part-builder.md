# SendCutSend AI Part Builder — Experiential Evaluation

**Date:** 2026-03-20
**Issue:** orch-go-hw6ej
**Phase:** Complete
**Status:** Complete

---

## What I Did

Logged into SendCutSend (app.sendcutsend.com), navigated to the Part Builder, discovered the AI Part Builder (beta) feature powered by **Smithy** (api.smithy.cc), and generated 4 parts from text prompts to test geometry generation, DFM awareness, and compositional correctness.

### Test 1: Simple Flat Part — "mounting bracket with 4 holes"

**Prompt:** `mounting bracket with 4 holes`

**Result:** Generated a flat rectangular bracket (120×80mm) with 4 mounting holes and rounded corners (10mm fillet radius). Parameters exposed: overall width, overall height, corner fillet radius, Hole section. 3D model rendered correctly in Three.js viewer with dimensions overlaid.

**Refinement test:** Typed "make the holes larger, 8mm diameter" in the "What do you want to change?" prompt. The refinement was accepted but no visible change in the parameters panel — parameters stayed at their original values.

**Full flow walkthrough:** SAVE & CONTINUE triggered a one-way gate ("Model cannot be edited after continuing"). After confirmation, the part was imported into SCS's standard quoting pipeline: Production Method (CNC Machining from $115.20, Sheet Cutting from $1.78) → Material Selection (Metals, Composites, Plastics, Wood, Rubber/Gasket) → Services → Finishing. The part was saved as `ai_part_builder_37fd032b612206.step` (4.725" × 3.15" × 0.125").

### Test 2: Part with Bends — "U bracket with 90 degree bends"

**Prompt:** `U-shaped bracket with 90 degree bends on each side, 6mm mounting holes on the flanges`

**Result:** Generated a U-channel bracket with visible 90° bends. Parameters: base width (60mm), flange height (30mm), part depth (40mm), mounting hole radius (3mm — correctly interpreted 6mm diameter as 3mm radius).

**Key observation:** The AI understands bend geometry and renders post-bend 3D shapes. However, no bend parameters are exposed: no bend radius, K-factor, bend deduction, or minimum flange length.

### Test 3: Hardware + Bends DFM Conflict — "L-bracket with PEM nuts near bend line"

**Prompt:** `L-bracket with PEM nuts near the bend line and 90 degree bend`

**Result:** Generated an L-bracket (50mm base × 40mm upright × 45mm width) with a "Pem hole" parameter section. The holes appear on the base flange, positioned near the bend line.

**Critical finding:** The AI recognized "PEM nuts" and created appropriate hole geometry, but produced **zero DFM warnings** about the PEM-near-bend-line conflict. In real manufacturing, PEM hardware placed too close to a bend line is a common rejection cause — the insertion press interferes with the bend, or the bend deforms the inserted hardware. The AI generates per-feature geometry (PEM holes, bend lines) without validating their spatial interaction.

### Test 4: Complex/Ambiguous Request — "electronics enclosure"

**Prompt:** `electronics enclosure`

**Result:** Generated a box-shaped enclosure (120×80×30mm) with rounded corners (5mm radius) and a pocket depth parameter. The 3D model shows a multi-bend box with what appears to be an opening and possible ventilation features.

**Key observation:** The AI didn't ask any clarifying questions (PCB size? Connectors? IP rating? Mounting? Ventilation?) — it just generated a reasonable default. This is good for quick starts but means underspecified requirements are silently filled with defaults the user may not notice.

---

## What Worked Well

1. **Text-to-3D is genuinely impressive.** From a 5-word prompt to a parametric 3D model in ~15 seconds. The Smithy engine generates real 3D geometry (not just 2D profiles), rendered in Three.js with interactive rotation and dimensioning.

2. **Semantic parameter names.** The AI doesn't just generate geometry — it exposes semantically named parameters (base width, flange height, mounting hole radius, Pem hole) that map to the user's intent, not internal geometry primitives.

3. **Domain vocabulary recognition.** "PEM nuts" → Pem hole parameter. "Flanges" → flange height. "90 degree bends" → proper L/U channel geometry. The AI understands sheet metal manufacturing terminology.

4. **Iterative refinement UX.** The "What do you want to change?" prompt + undo/redo + Menu (Edit prompt / Start over) provides a reasonable iteration loop within the builder.

5. **Clean handoff to quoting.** The generated STEP file integrates directly into SCS's existing quoting pipeline — production method → material → services → finishing. No format conversion needed.

## What Didn't Work

1. **No DFM validation whatsoever.** The #1 finding. PEM nuts near bend lines, no minimum flange warnings, no bend radius constraints, no material-dependent thickness limits. The AI generates what you describe without checking if it's manufacturable.

2. **One-way gate before quoting.** "Model cannot be edited after continuing" means once you commit to the geometry, you can't go back and adjust based on material/service constraints discovered later. If the material you want doesn't support the bend radius you need, you have to start over.

3. **Refinement prompt may not work.** In Test 1, "make the holes larger, 8mm diameter" was accepted but the parameter panel didn't update. It's unclear if the geometry changed or the refinement was silently ignored.

4. **No clarification for ambiguous prompts.** Test 4's "electronics enclosure" generated a box without asking about critical specifications. A more guided flow with optional clarification questions would catch underspecification before the user commits to geometry.

5. **No exposed bend parameters.** The AI generates bends but hides bend radius, K-factor, and bend deduction. These are critical for sheet metal DFM and vary by material/thickness — but the user has no control over them until after the one-way commit.

## What Surprised Me

1. **It's powered by Smithy (api.smithy.cc), not SCS-internal.** The entire AI builder runs in an iframe from a third-party service. SCS didn't build this — they integrated it. This means the DFM gap is likely at the integration boundary: Smithy generates geometry, SCS validates manufacturing, but the two systems don't talk to each other during the generation phase.

2. **The model is truly parametric, not just a mesh.** Changing parameter values regenerates the geometry. This is more like a CAD parametric model than a static 3D file, which is unusual for an AI-generated part.

3. **Example prompts include "U BRACKET"** — one of the exact examples SCS's DFM spike found problematic. They're encouraging users to generate geometry that's likely to have DFM issues.

4. **Generation time is ~15-25 seconds** — fast enough for iterative exploration, but the one-way gate means you can't iterate on the geometry after learning about material constraints.

## Would I Use This Again?

**For simple flat parts**: Yes, absolutely. It's faster than CAD for mounting brackets, cover plates, and other straightforward geometry. The parametric controls let you dial in dimensions quickly.

**For bent parts**: With extreme caution. The lack of DFM validation means you could easily generate a part that SCS's own production team rejects. You'd need to know bend rules yourself and verify them manually.

**For complex assemblies**: No. The tool generates single-part geometry. An "electronics enclosure" is really 2-5 parts (base, lid, brackets, panels) — the AI generates one box shape without considering assembly.

---

## Architecture & Integration Analysis

### The Compositional Correctness Gap — Confirmed

The SCS DFM spike found 46.2% recall on hardware+bend conflict detection. This AI builder has **0% recall** — it doesn't attempt DFM validation at all. The gap exists at the integration boundary:

```
[Smithy AI] → generates parametric 3D geometry
    ↓ (STEP file, one-way handoff)
[SCS Quoting] → validates material compatibility, pricing
```

DFM validation would need to happen either:
1. **Inside Smithy** (before generation) — requires Smithy to know SCS's DFM rules
2. **At the handoff** (after generation, before commit) — requires real-time DFM checking API
3. **After commit** (during quoting) — user discovers issues after losing the ability to edit

Currently option 3 is what happens — the user discovers issues in the quoting step when they can no longer change geometry.

### Boundary Implications for AI Part Generation

The fundamental tension: **AI that generates geometry needs to understand manufacturing constraints, but those constraints live in a different system.** This is the same pattern as our compositional correctness model — per-operation validation (hole positions OK, bend angles OK) passes, but operation-interaction validation (hole too close to bend) fails.

SCS's AI builder is an existence proof that:
- Text-to-3D geometry generation is production-ready for simple parts
- Domain vocabulary (PEM, flanges, bends) transfers into AI-generated parameters
- The DFM validation gap is at the system boundary, not within either system
- One-way handoffs between generation and validation create unrecoverable states

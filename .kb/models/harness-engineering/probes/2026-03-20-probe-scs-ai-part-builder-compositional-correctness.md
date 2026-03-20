# Probe: SCS AI Part Builder — Compositional Correctness Gap in Commercial DFM

**Model:** harness-engineering
**Date:** 2026-03-20
**Status:** Complete
**claim:** CCG-DFM
**verdict:** confirms

---

## Question

The harness engineering model claims a **compositional correctness gap** exists in sheet metal DFM: "Individual cuts, bends, hardware insertions each pass DFM rules. Composed assembly: bend line crosses hardware location, cuts weaken fold region, hardware collides after bending." This claim cites SendCutSend as the domain but has no direct evidence from SCS tooling.

**Claim under test:** Does SCS's AI Part Builder exhibit the compositional correctness gap? Specifically:
1. Does it validate individual operations (cuts, bends, hardware) independently?
2. Does it catch inter-operation interference (hardware placement conflicting with bend lines)?
3. What DFM checking exists at the composition level vs component level?

**Secondary questions:**
- What CAD format does the AI Part Builder output?
- How does the Template→Customize→Material→Services→Finishing flow handle DFM validation?
- Can you create a part that passes all individual checks but has a hardware+bend conflict?

---

## What I Tested

Evaluated SCS AI Part Builder (app.sendcutsend.com) via web interface — 4 test parts generated through the full Smithy→SCS pipeline. Key test was **Test 3: L-bracket with PEM nuts near the bend line and 90 degree bend** — a known DFM conflict scenario where hardware insertion press interferes with bend tooling.

Also tested:
- Test 1: Simple flat part ("mounting bracket with 4 holes") — baseline geometry generation
- Test 2: U-bracket with 90° bends — bend handling capabilities
- Test 4: Ambiguous prompt ("electronics enclosure") — underspecification handling

Full workflow traced: text prompt → Smithy generation → parameter editing → refinement prompt → SAVE & CONTINUE (one-way gate) → SCS quoting pipeline (Production Method → Material → Services → Finishing).

Source: `.kb/investigations/2026-03-20-eval-sendcutsend-ai-part-builder.md`

---

## What I Observed

**Test 3 (PEM near bend line) — direct compositional correctness gap evidence:**

- Smithy generated an L-bracket (50mm base × 40mm upright × 45mm width) with a "Pem hole" parameter section
- PEM holes positioned near the bend line — correct geometry for the individual features
- **Zero DFM warnings** about the PEM-near-bend-line conflict
- In manufacturing: PEM hardware too close to bend → insertion press interferes with bend, OR bend deforms inserted hardware → part rejection
- Smithy validated geometry (manifold, parameters in range). SCS quoting would accept the STEP file. Manufacturing would reject the part.

**Architecture observation (all tests):**

- Integration is iframe embedding from api.smithy.cc — Smithy is a third-party service, not SCS-internal
- One-way gate: "Model cannot be edited after continuing" — commits geometry before DFM discovery
- No DFM feedback flows from SCS to Smithy during generation
- Output format: STEP file (e.g., `ai_part_builder_37fd032b612206.step`)
- Bend parameters (radius, K-factor, deduction) hidden from user — not exposed in parameter panel

**Quantitative comparison:**
- SCS DFM spike (Fin bot): 46.2% recall on spatial reasoning (hardware+bend conflicts)
- SCS AI Part Builder (Smithy): **0% recall** on same class of conflicts — a regression

---

## Model Impact

- [x] **Confirms** invariant: compositional correctness gap exists in commercial DFM tooling — SCS's AI Part Builder is a production example where geometry validates (Smithy) and manufacturing validates (SCS), but no system validates that geometry is manufacturable. The gap is at the system boundary (iframe, one-way handoff), not within either system.

**Extends model with:**
- The gap is specifically at the **third-party integration boundary** — not just between abstraction levels, but between separate commercial systems with no shared constraint language
- The one-way gate architecture makes the gap **unrecoverable** — users cannot iterate after discovering DFM issues
- The AI Part Builder **regresses** from SCS's existing DFM tools (0% vs 46.2% recall) — adding AI generation without composition gates can make things worse

---

## Notes

- This is the first direct evidence from the SCS tooling itself (prior model claims cited SCS as domain without testing their tools)
- Related: LED gate stack probe (2026-03-20) showed same gap in OpenSCAD — 4-layer gates pass geometry but miss function
- Related: New model created at `.kb/models/smithy-geometry-engine/model.md` capturing full Smithy capabilities and limitations
- Context: DFM spike work, 46.2% recall on spatial reasoning
- Three architectural paths to close the gap identified (none currently implemented): (1) Smithy internalizes DFM rules, (2) real-time DFM API at boundary, (3) remove one-way gate

# Probe: Can Layer 5 Vision Gate Close the Gate Coverage Gap?

**Model:** signal-to-design-loop
**Date:** 2026-03-20
**Status:** Complete

---

## Question

The prior probe (2026-03-20-probe-domain-gate-coverage-gap-physical-cad) established that a functional coverage gap exists: geometric gates (Layers 1-4) cannot catch functional errors like disconnected LED channels or misplaced magnets. The probe concluded with an open question:

> Could a Layer 5 gate (render preview + LLM vision analysis) close this gap?

**Claim under test:** The signal-to-design loop model's "Gate Over Remind" principle has a boundary at abstraction-level mismatch. The model predicts that closing this gap requires a new signal capture mechanism operating at the functional (not geometric) abstraction level. Can Claude vision analysis of multi-angle renders serve as this mechanism?

**Hypothesis:** A structured vision gate (multi-angle PNG renders + Claude vision with functional checklists) can reliably distinguish geometrically-valid-but-functionally-broken designs from correct ones, with measurable precision/recall. If confirmed, this extends the model: "Gate Over Remind" applies across abstraction levels when the gate is multimodal (not just structural).

---

## What I Tested

**Approach:** Design analysis (architect session, not implementation). Examined the full gate stack, existing render infrastructure, and prior experimental evidence to design a Layer 5 vision verification gate, instrumented 26-letter sweep, and model variation experiment.

**Evidence consulted:**
1. Existing gate scripts: `gates/geometry-check.sh` (Layer 2), `gates/intent-check.sh` (Layer 4), `gates/render-viewpoints.sh` (render infrastructure)
2. Letter enclosure spec: `specs/letter-enclosure-spec.md` — 6 verification viewpoints with camera angles
3. Prior probe evidence: cut-channel vs guide-rail comparison (both pass all gates, opposite functional outcomes)
4. OpenSCAD source: `parts/letter-enclosure.scad` — current parametric design
5. Existing intent gate: Layer 4 already uses Claude (haiku) for code-level intent alignment but does NOT use vision

**Design experiments:**
- Analyzed what information is present in each verification viewpoint PNG
- Mapped functional requirements (FR1-FR5) to which viewpoints expose violations
- Designed calibration protocol using known-good (guide-rail) and known-bad (cut-channel) geometry
- Designed telemetry schema to capture gate results across all 5 layers for all 26 letters

---

## What I Observed

### 1. Layer 4 (intent gate) already has the plumbing but lacks vision

The existing `intent-check.sh` accepts a PNG file parameter (arg 4) but never sends it to the LLM. It only sends source code and metadata. Adding vision would extend, not replace, Layer 4.

**Design decision:** Layer 5 should be a separate gate (not a modification to Layer 4) because:
- Layer 4 is code-level analysis (structural review of .scad source)
- Layer 5 is render-level analysis (functional verification from rendered output)
- Different abstraction levels, different failure modes, different calibration needs

### 2. The verification viewpoints are the right signal capture mechanism

The spec defines 6 viewpoints specifically designed to expose functional requirement violations. Each maps to specific FRs:

| Viewpoint | Primary FR | What violation looks like |
|-----------|-----------|--------------------------|
| Top-down into cavity | FR1 (LED cavity) | Obstructions on floor, insufficient depth |
| Side cross-section | FR1, FR2 | Thin interior, magnet recesses visible from wrong side |
| Bottom/back view | FR2 (magnets) | Magnets outside footprint, through-holes, asymmetric placement |
| Detail: connectors | FR4 (daisy-chain) | Pockets outside letter, inaccessible, wrong size |
| Front elevation | FR3, FR5 (panel, print) | No lip visible, overhangs needing supports |
| Isometric | General | Degenerate geometry, wrong proportions |

### 3. Calibration requires known-bad geometry (which no longer exists in the repo)

The cut-channel approach was implemented and tested but the code was removed. To calibrate Layer 5, we need to regenerate known-bad variants:
- Cut-channel LED routing (disconnected paths)
- Out-of-footprint magnet placement (narrow letter I with wide-letter coordinates)
- Missing lip recess (panel won't snap in)
- Through-hole magnet recesses (wall behind too thin)

These can be produced via `-D` flag manipulation or temporary `.scad` variants.

### 4. Gate telemetry requires structured output at each layer

Current gates emit unstructured text (grep-parseable but not machine-aggregatable). An instrumented sweep needs JSON output per gate per letter. Designed schema that captures:
- Gate identity (layer, version)
- Input identity (letter, font, parameters)
- Result (pass/fail/warn with specific checks)
- Timing (duration per gate)
- Evidence (file paths to artifacts)

### 5. Model variation experiment is feasible but needs careful isolation

Key confound: different models may produce different OpenSCAD code that triggers different gate behaviors not because of correctness but because of coding style differences. Isolation requires:
- Same prompt for both models
- Same parameters (-D flags, not code changes)
- Same gate stack (no model-dependent paths)
- Measure at output level (gate results, not code quality)

---

## Model Impact

- [ ] **Extends** "Gate Over Remind" boundary: The principle holds across abstraction levels IF the gate is multimodal. A vision gate operating on rendered output bridges the geometric-to-functional gap that structural gates cannot cross. This is a new gate category: **judgment gates with visual evidence** (vs. Layer 4's judgment gates with code evidence).

- [ ] **Extends** Signal Capture analysis: Multi-angle verification viewpoints are a designed signal capture mechanism. They're not incidental artifacts — the spec explicitly defines camera angles that expose functional violations. This is "capture embedded in workflow" at its strongest: the viewpoints exist for human review AND automated review.

- [ ] **Confirms** Clustering Resolution hierarchy: The telemetry schema uses explicit, enumerated keys (gate layer, check name, letter, result) — following the model's prescription for machine-readable clustering. Cross-letter analysis (e.g., "which letters fail Layer 5 on magnet placement?") becomes a clustering operation over structured telemetry.

- [ ] **Open question (refined):** What is the precision/recall of Claude vision for functional correctness in CAD renders? The calibration protocol will produce the first measurement, but the expectation is:
  - High recall for gross violations (disconnected channels visible in top-down view)
  - Lower recall for subtle violations (magnet 1mm outside footprint edge)
  - Unknown: false positive rate (flagging correct geometry as problematic)

---

## Notes

- This is a design probe (architect session), not an implementation probe
- Implementation will be tracked as separate issues from the recommendations
- The design addresses three components: telemetry, Layer 5, and model variation experiment
- Full investigation file: `.kb/investigations/2026-03-20-design-instrumented-26-letter-sweep.md`

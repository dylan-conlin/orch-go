# Probe: Domain Gate Coverage Gap in Physical CAD Design

**Model:** signal-to-design-loop
**Date:** 2026-03-20
**Status:** Complete

---

## Question

The model asserts "Gate Over Remind — if it's not gated, it won't happen consistently" (Failure Mode 1 fix) and that signal capture must be "low-cost, embedded in workflow, not separate." The LED magnetic letters project implements a 4-layer gate stack (parameter validation, geometry check, printability, intent alignment) for OpenSCAD-based physical design.

**Claim under test:** Does the "Gate Over Remind" principle hold in domain-specific (physical/CAD) contexts? Specifically: does the 4-layer gate stack provide sufficient coverage to catch design errors, or does a coverage gap exist between geometric validity and functional correctness?

**Hypothesis:** Gates that validate geometric properties (manifold, polygon count, build plate fit) will miss functional design errors (LED strip routing connectivity, magnet placement within letter footprint) — creating a class of errors that pass all gates but produce non-functional parts.

---

## What I Tested

**Approach:** Ran OpenSCAD experiments across all 26 Impact font letters to validate the parametric letter-enclosure architecture. Tested two LED channel routing strategies and evaluated gate coverage at each step.

**Experiment set 1: Enclosure geometry** (3 agents, ~100 OpenSCAD renders)
- text() + offset(r=-wall) + difference() for all 26 letters with Impact font
- CGAL manifold validation, bounding box checks, facet counts
- Snap-fit lip geometry with varying wall/lip dimensions
- Magnet recess placement at fixed vs. computed positions

**Experiment set 2: LED zigzag channel routing** (~50 OpenSCAD renders)
- **Approach A (cut channels):** intersection() of zigzag pattern with inner letter profile
- **Approach B (guide rails):** horizontal raised rails clipped to inner profile
- Tested 6 letter shapes: A, M, W, H, O, L

**Gate coverage analysis:** For each experiment, tracked which gate layer (1-4) would catch vs. miss the issue.

---

## What I Observed

### 1. Cut-channel approach passes all gates but is functionally broken

The intersection-clipped zigzag channel approach produced:
- Layer 1 (parameter validation): PASS — all parameters within range
- Layer 2 (geometry check): PASS — CGAL manifold, under polygon budget, fits build plate
- Layer 3 (printability): would PASS — valid solid geometry
- Layer 4 (intent alignment): UNCERTAIN — depends on whether spec mentions "connected channels"

**But the channels are disconnected for every non-rectangular letter tested (A, M, W, H, O, L).** Diagonal strokes clip horizontal channels into isolated segments. The LED strip has no continuous path. This is a completely non-functional design that passes all geometric gates.

### 2. Guide-rail approach also passes all gates and IS functional

The raised-rail approach produces valid geometry AND functional LED routing. But the gate stack cannot distinguish it from the broken cut-channel approach — both are geometrically valid, manifold, and within all parameter bounds.

### 3. Magnet placement at fixed coordinates passes gates but fails for narrow letters

Placing magnets at hardcoded positions (e.g., [30,30], [30,170]) produces valid geometry for wide letters (A, M) but magnets land outside the letter footprint for narrow letters (I, L). Layer 1 validates magnet dimensions, Layer 2 validates manifold — neither checks whether magnets are actually inside the letter shape.

### 4. Lip wall thickness: Gate catches this correctly

Setting wall=1.5mm and lip_inset=0.8mm creates a 0.7mm lip wall. Layer 1's `validate_wall_thickness` catches this (below 1.2mm minimum). **This is the gate stack working as designed.**

### 5. Bottom wall vs magnet depth: Gate catches this correctly

Setting bottom_wall=1.5mm with magnet_depth=3mm creates a through-hole. Layer 1's `validate_magnet_recess` catches this (wall_behind=0mm < 0.8mm minimum). **Gate stack working as designed.**

---

## Model Impact

- [x] **Confirms** "Gate Over Remind" as a necessary principle. Parameter validation (Layer 1) and geometry checking (Layer 2) caught dimensional errors immediately and deterministically. Without these gates, the 0.7mm lip wall and through-hole magnet recess would have reached the printer.

- [x] **Extends** the model with: A **functional coverage gap** in gate-based enforcement. The 4-layer gate stack validates geometry (is it manifold? does it fit?) but cannot validate function (is the LED path connected? are magnets inside the letter?). This is a new failure mode class:

  > **Gate Coverage Gap:** When enforcement gates operate at a different abstraction level than the functional requirements. Geometric gates check shape validity; functional correctness requires domain-semantic understanding.

  This maps to the model's clustering resolution hierarchy: geometric gates are "explicit tag" (structured, machine-readable), but functional correctness requires "judgment" — making it resistant to infrastructure enforcement.

- [x] **Extends** "Signal Capture" stage analysis: In physical CAD, the signal that a design is functionally broken is **invisible to automated gates** but **immediately visible to a human looking at a preview image**. This suggests the PNG preview output (already required per CLAUDE.md) serves as a critical signal capture mechanism that the gate stack cannot replace. The "Gate Over Remind" principle has a boundary: some properties are only observable through visual inspection.

- [ ] **Open question:** Could a Layer 5 gate (render preview + LLM vision analysis) close this gap? An LLM could look at the cross-section PNG and judge whether channels form a connected path. This would be a novel gate type: judgment-enforced via multimodal AI, bridging the geometric/functional gap.

---

## Notes

- Evidence base: ~150 OpenSCAD renders across fonts, enclosures, and LED routing experiments
- All renders in /tmp/ (ephemeral, not committed)
- Design investigation: .orch/workspace/lemale-arch-design-parametric-letter-20mar-a31b/
- The cut-channel vs guide-rail comparison is the key evidence: identical gate results, opposite functional outcomes

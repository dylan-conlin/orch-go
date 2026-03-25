# Probe: Physical CAD Port Layout Exposes Gate Coverage Gap in Spatial Reasoning

**Model:** signal-to-design-loop
**Date:** 2026-03-25
**Status:** Complete

---

## Question

The prior probe (2026-03-20-domain-gate-coverage-gap-physical-cad) established that geometric gates can't catch functional design errors — a "Gate Coverage Gap" where enforcement operates at a different abstraction level than functional requirements. Does the sled port layout design task confirm this gap, and does it reveal additional failure modes specific to multi-component spatial placement in constrained volumes?

**Specific claims under test:**
1. "Gate Over Remind" holds for dimensional validation but misses spatial conflicts
2. The gate coverage gap extends to component placement — pocket depth, inter-pocket clearance, and cable routing conflicts are invisible to Layer 1-3 gates
3. Signal capture for spatial conflicts requires design-phase reasoning, not post-render verification

---

## What I Tested

Analyzed the sled port layout design problem: placing 4 standalone components (USB-C, 2 buttons, battery indicator) in a 20.5mm diameter cylindrical sled body, with constraints from existing Pico/MH-CD42 pockets and an internal cable channel.

**Spatial conflict analysis performed:**

1. **USB-C depth vs sled radius:** 14mm component depth in 10.25mm radius cylinder. Computed pocket extends 3.75mm past center. Checked structural impact on -Y wall (6.5mm remaining — adequate).

2. **USB-C pocket vs cable channel collision:** At X=0, Y=-3.75 (USB-C pocket back) vs cable channel at -Y face. If cable channel at X=0, gap is only 0.9mm (below 1.2mm PETG minimum). Resolved by X-offsetting cable to X=+6mm.

3. **Button pocket vs Pico pocket (Option B analysis):** 5.1mm available depth, 4.3mm button depth = 0.8mm remaining wall. Below structural minimum. This eliminated Option B.

4. **MH-CD42 pocket width vs cable channel routing:** Pocket is 17.6mm wide in 17.5mm bore — nearly fills entire bore. Cable channel cannot route alongside MH-CD42 at ±X faces. Must use -Y face offset in X.

```
# Gate analysis: which layers would catch each spatial conflict?

USB-C depth > sled radius:
  L1 (parameter validation): Would NOT catch — parameters individually valid
  L2 (geometry check): WOULD catch — non-manifold if pocket breaches exterior
  L2.5 (connectivity): NOT applicable — no connectivity claim
  L3 (printability): WOULD catch — unprintable geometry

Cable channel vs USB-C pocket collision (0.9mm wall):
  L1: Would NOT catch — no cross-pocket wall thickness validator
  L2: Would NOT catch — both pockets are valid individual geometry
  L2.5: Could catch IF a connectivity probe existed for cable-to-shelf path
  L3: MIGHT catch — thin wall warning, but depends on slicer resolution

Button pocket vs Pico pocket (Option B, 0.8mm wall):
  L1: Would NOT catch — validate_wall_thickness checks sled outer wall only
  L2: Would NOT catch — geometry is manifold
  L3: MIGHT catch — thin wall warning
```

---

## What I Observed

### 1. Gate stack misses inter-pocket spatial conflicts

The critical design insight — that the USB-C pocket and cable channel collide at 0.9mm separation — is invisible to Layers 1-3. Layer 1 validates individual parameters (pocket_w, pocket_d, channel_w). Layer 2 validates manifold geometry of the complete render. Neither checks the wall thickness between two internal pockets.

This is a specific instance of the gate coverage gap: **adjacent pocket clearance** is a functional constraint that exists at the spatial composition level, not the individual geometry level.

### 2. Design-phase reasoning catches what gates miss

The 0.9mm wall conflict was caught during design exploration (Phase 2), not by running gates on rendered geometry. The resolution (X-offset the cable channel to +6mm) required reasoning about the 3D intersection of multiple pocket volumes — something that requires understanding component relationships, not just individual validity.

### 3. Layer 2.5 connectivity probes partially close the gap

The existing connectivity probe infrastructure (CSG intersection tests) could catch the cable-channel-to-shelf-groove path if a probe existed for it. The probe would render the channel volume intersected with the sled body and verify the resulting solid has continuous Z-extent from wiring gap to shelf. If the USB-C pocket severed the channel, the probe would show a gap.

However, no probe currently exists for this path because the cable path is being redesigned. The connectivity check infrastructure is reactive (tests existing geometry) not predictive (evaluates proposed geometry before implementation).

### 4. Height impact is invisible to all gates

The +15mm height increase (95.5mm → 110.5mm) passes all gates — the sled still fits the build plate. But the ergonomic impact (sled sits deeper in pole, changes weight distribution) is a judgment call that requires human review. No gate captures this.

---

## Model Impact

- [x] **Confirms** "Gate Coverage Gap" from prior probe: Geometric gates operating at individual-parameter and whole-render levels miss inter-pocket spatial conflicts. The USB-C/cable-channel collision at 0.9mm is a concrete new instance.

- [x] **Extends** the model with: **Spatial composition gap** as a specific sub-class of the gate coverage gap. When multiple negative-space features (pockets, channels) are placed in a constrained volume, the critical constraints are at the interfaces between features, not within individual features. Existing gate layers validate features individually (L1) or as a final rendered union (L2), but miss the intermediate level where pocket-to-pocket interactions determine viability.

  > **Inter-feature clearance** is a functional constraint that lives between parameter validation and geometry validation. It requires compositional reasoning: "given pocket A at position P1 and pocket B at position P2, is the wall between them adequate?"

- [x] **Extends** signal capture analysis: For multi-component spatial placement, the most valuable signal capture happens during design-phase exploration — trying placements, computing clearances, rejecting options. This reasoning is currently ephemeral (exists only in the architect session). The investigation artifact captures the outcome but not the spatial reasoning process.

  Potential gate: A **clearance matrix** validator that takes a list of pocket volumes and checks minimum wall thickness between all adjacent pairs. This would be a Layer 1.5 gate — parameter-level but compositional.

- [x] **Confirms** "Layer 2.5 connectivity probes partially close the gap" — the existing infrastructure could catch cable path discontinuity if probes were written for new paths before implementation.

---

## Notes

- This probe is confirmatory: findings align with prior probe's gate coverage gap thesis
- The spatial composition gap may be a general pattern in any constrained-volume multi-feature design (not just this project)
- A clearance matrix validator would be a novel contribution to the gate stack — worth a follow-up investigation
- Evidence: sled port layout design in `.kb/investigations/2026-03-25-design-sled-port-layout.md`

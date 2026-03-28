# Probe: Assembly Feasibility Review Exposes Realizability Gap Beyond Geometry Probes

**Model:** signal-to-design-loop
**Date:** 2026-03-25
**Status:** Complete

---

## Question

The prior probe (2026-03-25-physical-cad-port-layout-design-gap) established that Layer 2.5 connectivity probes partially close the gate coverage gap by verifying cavity continuity via CSG intersection. The sled now passes 9/9 connectivity probes. Does a holistic assembly feasibility review reveal additional constraints that passing probes don't capture? Specifically: does "cavity exists" equal "physically buildable"?

**Specific claims under test:**
1. Connectivity probes (Layer 2.5) verify necessary but not sufficient conditions for assembly feasibility
2. The spatial composition gap extends beyond inter-pocket clearance to cross-component physical interfaces (sled ↔ PVC wall ↔ external world)
3. Wire routing feasibility requires volumetric analysis (wire bundle cross-section vs. cavity cross-section) that probes don't perform

---

## What I Tested

Full assembly feasibility review of universal-sled.scad (110.5mm, 9/9 probes passing):

**1. Wire gallery volumetric analysis**

The wire gallery (10mm × 2mm × 20mm, +Y inner wall) routes hookup wires from port pockets to MH-CD42/Pico. Probes verify the gallery cavity exists and connects to both endpoints. But the probes don't check whether the wires physically fit.

```
Wire count at gallery bottom (Z=68): 8 wires
  USB-C → MH-CD42: 2 wires
  Button 1 → MH-CD42: 2 wires
  Button 2 → Pico: 2 wires
  Indicator → MH-CD42: 2 wires

Gallery cross-section: 10mm wide × 2mm deep = 20mm²

22AWG (1.3mm OD w/ insulation): 7/layer → need 2 layers = 2.6mm depth
  DOES NOT FIT (2mm available)

24AWG (1.0mm OD w/ insulation): 10/layer → need 1 layer = 1.0mm depth
  FITS (2mm available, 1mm margin)
```

Finding: Wire gauge is a hard constraint on assembly feasibility. 22AWG won't fit. 24AWG works with margin. No gate catches this.

**2. Wiring gap transition (Z=38-41)**

MH-CD42 and Pico pockets overlap by only 1.2mm in Y at the Z=41 boundary. Four wires (2 battery + 2 button-2) must pass through this overlap. Probe `probe_gap_to_mhcd42` verifies the cavity exists.

```
Overlap: Y = 3.95 to 5.15 = 1.2mm deep × 17.6mm wide
24AWG: 4 wires × 1.0mm = 4mm → fits in 17.6mm width
Depth: 1.0mm wire vs 1.2mm overlap → 0.2mm clearance
```

Finding: Physically feasible with 24AWG but 0.2mm clearance leaves no room for error. A wire slightly misaligned blocks the transition. No gate validates this margin.

**3. Button actuation through PVC wall**

Buttons sit in pockets on the sled +Y face. Between button actuator and the outside world: 0.7mm pocket recess + 0.2mm air gap + 2.9mm PVC wall = 3.8mm.

```
Typical 6x6mm switch actuator: 3-4mm protrusion
At 4mm: just reaches PVC exterior (0.2mm margin)
At 3mm: falls 0.8mm short of PVC exterior
```

Finding: Button actuation feasibility depends on exact switch model selected. The 0.2mm margin with a 4mm actuator is borderline. No gate or probe checks cross-component physical interfaces (sled component → PVC wall → external access).

**4. Thermal analysis**

MH-CD42 at max load (2.1A, 5V, 96% efficiency): dissipates 0.44W.
Temperature rise through 1.5mm PETG pocket wall: ~2.4°C above PVC surface.
At Burning Man ambient (45°C) + solar gain on PVC (~60°C): pocket reaches ~62°C.
PETG glass transition: 80°C. Margin: 18°C.

Finding: Thermal is not a concern. Even at full load in direct desert sun, temperatures stay well below PETG softening.

**5. Structural cross-section at USB-C level (Z=69-74)**

USB-C pocket extends 4.15mm past sled center. Remaining -Y wall: 6.1mm at X=0. Cable channel at X=+6mm adds a 5×2mm void but doesn't compound with USB-C (different X ranges). PVC pipe provides lateral backup support during use.

Finding: Structural integrity is adequate. The PVC bore constrains lateral deflection before sled wall failure. Ring shelf transfers axial load directly to pipe wall.

---

## What I Observed

### 1. Connectivity probes verify topology, not physical realizability

All 9 probes pass. The cavity network exists: port pockets → wire gallery → MH-CD42 → wiring gap → Pico → battery bore. The internal cable channel reaches from wiring gap to shelf groove.

But "cavity exists" ≠ "assembly works." Three physically-constrained transitions are invisible to the probe infrastructure:

| Transition | Probe result | Physical constraint | Gate coverage |
|-----------|-------------|-------------------|--------------|
| Wire gallery bottom (Z=68) | PASS (cavity) | 8 wires must fit in 10×2mm | **None** |
| Wiring gap overlap (Z=41) | PASS (cavity) | 4 wires in 1.2mm deep passage | **None** |
| Button → PVC exterior | No probe | 3.8mm actuator bridge needed | **None** |

### 2. The realizability gap is a new sub-class of the gate coverage gap

The prior probe identified the **spatial composition gap** (inter-pocket clearance invisible to gates). This review reveals a broader gap: **physical realizability** — whether a geometrically valid design can actually be built and used.

This gap has three flavors:
- **Volumetric fit:** Cavity exists but contents don't fit (wire bundles too thick)
- **Cross-component interface:** No single component owns the interface (button-to-PVC-wall)
- **Assembly sequence physics:** Components must be installed in order, but geometry may block earlier steps' results

### 3. Thermal and structural concerns are non-issues

The review validated that thermal dissipation (0.44W max) and structural integrity (6.1mm -Y wall + PVC backup) are adequate. These were judgment calls that required domain analysis, not geometric verification.

---

## Model Impact

- [x] **Extends** the spatial composition gap with: **Physical realizability gap** — geometrically valid designs that cannot be physically assembled, used, or maintained. This is distinct from the prior finding (inter-pocket clearance) because it crosses component boundaries. The sled geometry is valid; the sled + PVC + wire bundles + switch actuators composition is what's constrained. No single .scad file owns these interfaces.

- [x] **Extends** signal capture analysis: Assembly feasibility review is a manual, expert-driven synthesis step that catches constraints no automated gate handles. This is an instance of **Stage 4 (Synthesis) compensating for Stage 1 (Capture) gaps** — because there's no signal capture mechanism for "wire bundles don't fit in cavity" or "switch actuator doesn't reach through wall," the synthesis step (architect review) must discover these from first principles each time.

- [x] **Suggests** new signal types for the loop:
  - **Wire gauge specification** — a bill-of-materials constraint that interacts with cavity dimensions. Could be captured as a BOM field in specs.
  - **Cross-component clearance claims** — like connectivity claims but for physical interfaces between separately-manufactured parts (3D print ↔ PVC ↔ purchased components). Currently no template for these.

- [x] **Confirms** "connectivity probes are necessary but not sufficient" — probes verify the topology of the cavity network. They don't verify the physics of what passes through those cavities.

---

## Specific Recommendations

1. **Specify wire gauge in specs/universal-sled-spec.md**: Add "24AWG maximum for hookup wires in wire gallery and wiring gap transition" as a BOM constraint. This makes the feasibility analysis durable.

2. **Add button actuator height to specs**: The PVC-wall-to-button interface needs a minimum actuator protrusion of 3.8mm specified and validated against the chosen switch model.

3. **Consider a cross-component interface table**: Like the connectivity table but for physical interfaces between sled, PVC, and purchased components. Each row: "From [sled feature] through [PVC wall property] to [external access], clearance X mm."

---

## Notes

- Evidence: Quantitative analysis in this session using Python computation of cross-sections and wire packing
- All 9/9 connectivity probes confirmed passing before review (not a regression)
- This probe is confirmatory and extends the spatial composition gap finding from the port layout probe

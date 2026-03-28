## Summary (D.E.K.N.)

**Delta:** Rebuild the Pixelblaze Pico enclosure under the OpenSCAD harness, proving cross-domain harness portability with real hardware validation.

**Evidence:** OpenSCAD experiential eval (orch-go-5idhz), architect design (orch-go-qurre), LLM intent gate measurement (77.4% precision), existing pixelblaze_pico project with X1-Carbon tolerance data, self-measurement report methodology.

**Knowledge:** The gate taxonomy (mechanical/judgment) transfers to non-code domains. The self-measurement methodology is the transferable artifact, not the specific gates. Real hardware prints provide ground truth that agent-only systems can't fake.

**Next:** Phase 1 — set up the project with harness gates wired to the existing .scad files.

---

# Plan: Pixelblaze Pico Enclosure Under Harness

**Date:** 2026-03-16
**Status:** Active
**Owner:** Dylan

**Extracted-From:** Cross-domain harness portability thread (.kb/threads/2026-03-16-cross-domain-harness-portability-can.md)

---

## Objective

Rebuild the Pixelblaze Pico slide-top enclosure using agent orchestration with structural enforcement. Resume the existing modularization plan (22% complete) with agents executing each extraction. Validate every change through the 4-layer gate stack. Produce a cross-domain self-measurement report grounded in real print results from the X1-Carbon. Success = a printed enclosure that fits, with measured gate precision against physical reality.

---

## Substrate Consulted

- **Models:** harness-engineering (gate taxonomy, ceremony vs enforcement), orchestration-cost-economics
- **Decisions:** LLM intent gate stays advisory (77.4% precision, below 80% blocking threshold)
- **Guides:** minimum-viable-harness (day-one governance pattern)
- **Constraints:** No orch daemon in openscad-harness (lightweight, spawn via --workdir). PrusaSlicer at ~/bin/prusa-slicer. OpenSCAD 2025.02.28 with Manifold backend. X1-Carbon with PETG.

---

## Decision Points

### Decision 1: Project home

**Context:** Where does the pixelblaze work live?

**Options:**
- **A: Extend openscad-harness** — Add pixelblaze as a subproject alongside the sample parts. Pros: gate infrastructure already exists. Cons: mixes showcase tooling with real project.
- **B: Standalone project** — New repo ~/Documents/personal/pixelblaze-pico-v2 with harness gates copied in. Pros: clean separation, real project feel. Cons: duplicates gate scripts.
- **C: Use existing openscad-workbook/pixelblaze_pico** — Wire harness into the existing project directory. Pros: preserves history, tolerance notes, existing libraries. Cons: needs git init if not already a repo.

**Recommendation:** C — the existing project has real history (tolerance notes, revision log, X1-Carbon calibration data). That history IS the ground truth the harness validates against.

**Status:** Decided

### Decision 2: Orchestration approach

**Context:** How do agents interact with this project?

**Options:**
- **A: orch spawn --workdir** — Spawn from orch-go into the pixelblaze directory. Simple, no extra infra.
- **B: Full orch init** — Daemon, hooks, completion pipeline in the project. Heavy for a design workbench.

**Recommendation:** A — lightweight. Workers spawn into the project freely, no write guards, no daemon. Orchestrator coordinates from orch-go.

**Status:** Decided

---

## Phases

### Phase 1: Project setup and gate wiring

**Goal:** Wire the 4-layer gate stack to the existing pixelblaze project
**Deliverables:**
- Git repo initialized (or verified) in openscad-workbook/pixelblaze_pico
- Beads initialized with pixelblaze prefix
- Gate scripts copied from openscad-harness (L1-L4)
- validate.scad library integrated (complement existing assert() guards)
- Smoke test: render pico_slide_top_box.scad through full gate pipeline
- PrusaSlicer profile configured for X1-Carbon + PETG
**Exit criteria:** `openscad -o test.stl cases/pico_slide_top_box.scad` passes all 4 gate layers
**Depends on:** Nothing

### Phase 2: Resume modularization (core libraries)

**Goal:** Complete Phase 1 of existing MODULARIZATION_PLAN.md — core infrastructure
**Deliverables:**
- libraries/core/materials.scad — material profiles extracted from main file
- libraries/core/validation.scad — parameter validation framework
- libraries/core/geometry_utils.scad — rounded box, clamp, etc.
- Each extraction verified: output geometry identical to original (diff STL)
**Exit criteria:** Main file reduced from ~850 to ~600 lines. All 3 core libraries pass gate pipeline. Regression test: rendered STL matches original.
**Depends on:** Phase 1

### Phase 3: Resume modularization (mechanical + electronics)

**Goal:** Complete Phases 2-3 of existing plan — extract remaining libraries
**Deliverables:**
- libraries/electronics/pcb_mounting.scad
- libraries/electronics/wire_management.scad
- libraries/enclosures/slide_top_box.scad (the generator)
- pico_slide_top_box_v3.scad — fully modular version
**Exit criteria:** v3 renders identical geometry to v1. Main file is a thin config + generator call. All libraries pass gate pipeline independently.
**Depends on:** Phase 2

### Phase 4: Physical validation

**Goal:** Print v3, compare against v1 prints, validate gate accuracy
**Deliverables:**
- Print v3 on X1-Carbon with PETG
- Compare fit: dovetail clearance, flex button actuation, PCB seating, wire routing
- Record gate predictions vs physical reality (tolerance notes)
- Update material profiles based on measured results
**Exit criteria:** v3 prints and fits as well as v1. Tolerance notes updated with v3 data.
**Depends on:** Phase 3

### Phase 5: Cross-domain self-measurement report

**Goal:** Produce the report that proves harness portability
**Deliverables:**
- Gate fire rates across all phases (how often each layer caught something)
- False positive analysis (gates that blocked valid changes)
- Gate vs physical reality accuracy (predicted printability vs actual print results)
- Comparison with orch-go self-measurement report methodology
- Honest assessment: what transferred, what didn't, what's different about physical domains
**Exit criteria:** Report published, numbers auditable, falsification criteria defined
**Depends on:** Phase 4

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Project home | Existing project with history | Yes |
| Orchestration approach | Experiential eval + architect design | Yes |
| Gate thresholds | LLM intent at 77%, mechanical gates tested | Yes |
| Material profile | X1-Carbon PETG tolerance notes exist | Yes |

**Overall readiness:** Ready to execute

---

## Structured Uncertainty

**What's tested:**
- Gate taxonomy transfers to OpenSCAD (experiential eval, orch-go-5idhz)
- OpenSCAD CLI headless workflow works (54ms renders with Manifold)
- LLM intent gate precision is 77.4% (measured, orch-go-bn9fp)
- PrusaSlicer CLI available at ~/bin/prusa-slicer

**What's untested:**
- Whether gate pipeline catches real design regressions during modularization (vs catching only obvious errors)
- Whether L4 intent gate adds value on a real project (vs just sample parts)
- Whether the modularization preserves physical fit (STL diff doesn't guarantee print fit)
- PrusaSlicer X1-Carbon profile accuracy for gate predictions

**What would change this plan:**
- If STL diffing proves unreliable for regression testing (geometry identical but tolerance-sensitive features shift)
- If modularization changes render behavior (Minkowski + module boundaries = different floating point)
- If L4 intent gate produces >50% false positives on real design changes (drop it)

---

## Success Criteria

- [ ] v3 enclosure prints and fits on X1-Carbon (dovetail slides, button actuates, PCB seats)
- [ ] Gate pipeline catches at least 1 real issue during modularization that would have reached print
- [ ] Self-measurement report produced with gate precision numbers grounded in physical validation
- [ ] Main file reduced from ~850 to <200 lines (config + generator call)
- [ ] All extracted libraries pass gate pipeline independently

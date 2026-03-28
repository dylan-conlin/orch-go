---
title: "AI-assisted physical fabrication — Smithy DFM gap and LED letters harness are instances of the same compositional correctness problem"
status: open
created: 2026-03-20
updated: 2026-03-20
resolved_to: ""
---

# AI-assisted physical fabrication — Smithy DFM gap and LED letters harness are instances of the same compositional correctness problem

## 2026-03-20

Two projects, same gap: Smithy (SCS AI Part Builder) generates valid geometry but zero DFM validation — PEM nuts near bend lines pass without warning (0% recall vs 46.2% for SCS's own AI reasoning). LED magnetic letters hit the same structure: OpenSCAD generates valid geometry but printability/assembly feasibility is unchecked until late gates. Both are compositional correctness gaps where per-component validation succeeds but cross-concern validation is absent. The harness engineering model (HE §8) names this pattern. The smithy-geometry-engine model in orch-go captures the SCS instance; the LED letters project is building gates (geometry-check, prusa-slicer, intent-gate) that are the fabrication-side equivalent. Cross-project artifacts: scs-special-projects/.kb/investigations/2026-03-20-inv-create-kb-model-smithy-api.md, scs-special-projects/.kb/models/smithy-ai-part-builder/, led-magnetic-letters/ (gates/, specs/), orch-go/.kb/models/harness-engineering/ §8.

Strategic insight: SCS should own the AI Part Builder, not white-label Smithy. The geometry engine Dylan needs to build for the DFM checker (hardware+bend conflict detection) IS the differentiated core of a Smithy replacement. Smithy generates geometry in isolation from manufacturing knowledge — SCS's competitive moat is exactly that manufacturing knowledge (26 hardware types, material bending params, tooling specs, pricing). An SCS-owned system puts DFM gates inside the generation loop: LLM generates CadQuery code → Open Cascade executes → geometry engine validates DFM → failures feed back to LLM as structured constraints → loop until manufacturable. No irreversible commit gate, live quoting during design, parts arrive at production pre-validated. The geometry engine investment pays off twice: production DFM checker now (the spike's next step), owned AI Part Builder next. Incremental work beyond the engine: LLM code generation (medium — Smithy proved Claude can do this), Open Cascade integration (medium — CadQuery/build123d are Python wrappers), 3D viewer (easy — Three.js commodity), generation-DFM feedback loop (medium — LED letters project already proves this pattern with OpenSCAD). Honest risk: the geometry engine is the hard part and doesn't exist yet. Everything else is tractable IF the engine works.

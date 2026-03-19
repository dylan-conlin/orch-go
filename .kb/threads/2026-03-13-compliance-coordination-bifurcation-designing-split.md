---
title: "Compliance-coordination bifurcation — designing the split"
status: resolved
created: 2026-03-13
updated: 2026-03-18
resolved_to: "decision"
---

# Compliance-coordination bifurcation — designing the split

## 2026-03-13

The daemon currently entangles two layers with opposite value trajectories. Compliance (gates, enforcement, skill gravity) decreases in value as models improve. Coordination (skill inference, hotspot routing, work prioritization, parallel scheduling) increases in value as parallelism grows. Current ratio ~80/20 compliance/coordination — should invert. The system that results is a cognitive resource allocator, not an agent framework. Next capability: global context for work removal/deduplication (creation/removal asymmetry). Dylan: 'this is the future' — designing the bifurcation now.

Exploration complete (3 architects + judge). UNIFIED DESIGN: Interface is narrow (compliance produces allow/reject signals, coordination consumes). Only 3 methods entangled. Compliance dial: 4 levels (Strict/Standard/Relaxed/Autonomous) per-(skill,model). Coordination expansion: Learning Store, Allocation Profile, Work Graph, OODA Refactor. Integrated roadmap: Phase 0 structural split → Phase 1 foundation (config + learning) → Phase 2 gate awareness + allocation → Phase 3 measurement + work graph → Phase 4 OODA capstone. Contested findings resolved: extraction is shared infrastructure, learning store compatible with No Local Agent State. Artifacts: 3 investigations + judge verdict in workspaces.

All 7 phases shipped in one session. The daemon is now structurally bifurcated: compliance.go produces signals, coordination.go consumes them. ComplianceConfig provides a 4-level dial (Strict/Standard/Relaxed/Autonomous) per (skill,model). Learning Store aggregates events into per-skill metrics. Allocation Profile scores candidates. Work Graph detects duplicates. Measurement feedback loop auto-adjusts compliance from learning data. OODA refactor makes the loop legible as Sense/Orient/Decide/Act. Next: accumulate 2-4 weeks of data, then run gate effectiveness query (orch-go-00r9c) to answer empirically whether structural enforcement improves agent quality.

## 2026-03-18 — Resolution

The "next" action completed: gate effectiveness query ran (orch-go-00r9c), 2-week measurement collected, and the answer is in.

**Empirical answer:** Structural enforcement does NOT improve agent quality. The 96%+ success rate is intrinsic to the skill+model combination, not produced by compliance gates. Evidence: 100% bypass rate on blocking gates with no quality difference between enforced/bypassed cohorts. Decision `2026-03-17-accretion-gates-advisory-not-blocking.md` formalized this — all gates converted to advisory.

**What the bifurcation produced:**
- compliance.go / coordination.go structural separation: working, clean interface
- ComplianceConfig with 4-level dial: implemented, auto-downgrade suggestions produced on first run
- Learning Store: aggregating real per-skill metrics (feature-impl 96%, investigation 100%)
- Allocation Profile: scoring candidates by skill success rate
- Work Graph: deduplication and conflict detection active
- OODA refactor: Sense/Orient/Decide/Act legible in code

**Resolved.** Design shipped, measurement confirmed the thesis, gates converted to advisory. Blog draft at `.kb/drafts/compliance-cliff.md`. Remaining coordination investments (auto-synthesis, coordination ROI measurement) are separate work items.

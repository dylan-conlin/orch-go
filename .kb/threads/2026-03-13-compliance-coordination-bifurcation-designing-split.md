---
title: "Compliance-coordination bifurcation — designing the split"
status: open
created: 2026-03-13
updated: 2026-03-13
resolved_to: ""
---

# Compliance-coordination bifurcation — designing the split

## 2026-03-13

The daemon currently entangles two layers with opposite value trajectories. Compliance (gates, enforcement, skill gravity) decreases in value as models improve. Coordination (skill inference, hotspot routing, work prioritization, parallel scheduling) increases in value as parallelism grows. Current ratio ~80/20 compliance/coordination — should invert. The system that results is a cognitive resource allocator, not an agent framework. Next capability: global context for work removal/deduplication (creation/removal asymmetry). Dylan: 'this is the future' — designing the bifurcation now.

Exploration complete (3 architects + judge). UNIFIED DESIGN: Interface is narrow (compliance produces allow/reject signals, coordination consumes). Only 3 methods entangled. Compliance dial: 4 levels (Strict/Standard/Relaxed/Autonomous) per-(skill,model). Coordination expansion: Learning Store, Allocation Profile, Work Graph, OODA Refactor. Integrated roadmap: Phase 0 structural split → Phase 1 foundation (config + learning) → Phase 2 gate awareness + allocation → Phase 3 measurement + work graph → Phase 4 OODA capstone. Contested findings resolved: extraction is shared infrastructure, learning store compatible with No Local Agent State. Artifacts: 3 investigations + judge verdict in workspaces.

All 7 phases shipped in one session. The daemon is now structurally bifurcated: compliance.go produces signals, coordination.go consumes them. ComplianceConfig provides a 4-level dial (Strict/Standard/Relaxed/Autonomous) per (skill,model). Learning Store aggregates events into per-skill metrics. Allocation Profile scores candidates. Work Graph detects duplicates. Measurement feedback loop auto-adjusts compliance from learning data. OODA refactor makes the loop legible as Sense/Orient/Decide/Act. Next: accumulate 2-4 weeks of data, then run gate effectiveness query (orch-go-00r9c) to answer empirically whether structural enforcement improves agent quality.

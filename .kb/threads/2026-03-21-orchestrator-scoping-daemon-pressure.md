---
title: "Orchestrator as scoping agent — daemon as sole executor"
created: 2026-03-21
status: active
---

# Orchestrator as scoping agent — daemon as sole executor

## 2026-03-21

**Core idea:** The orchestrator's only output is a well-scoped beads issue. The daemon is the sole spawn mechanism. `orch spawn` is removed from the orchestrator's tool space.

**Why:** As long as the orchestrator can bypass the daemon by spawning directly, the daemon never experiences the friction it needs to improve. Every rough edge (bad skill inference, missing context, slow scheduling) gets worked around instead of fixed. This is "pressure over compensation" applied to the spawn pipeline itself.

**Gate:** Orchestrators cannot call `orch spawn`. (Hard harness — remove from allowed tools in orchestrator skill settings.)

**Attractor:** Creating a well-scoped issue is the natural completion of orchestrator thinking. The issue is the work product. Type, labels, description, dependencies — these are the orchestrator's judgment crystallized into a form the daemon can act on.

**What this changes:**
- Orchestrator becomes a thinking agent, not an execution agent
- Issue quality improves because the issue has to be good enough for automated routing
- Daemon friction surfaces instead of being bypassed — drives daemon improvement
- The 15 cut daemon research tasks become even more obviously unnecessary: the orchestrator decides what's worth investigating, not autonomous scanning
- Separation of concerns: orchestrator = judgment, daemon = execution, human = direction

**What needs to happen:**
1. Remove `orch spawn` from orchestrator skill's tool action space
2. Improve daemon skill inference to handle the cases where direct spawn was "needed"
3. Improve issue description → spawn context pipeline (the issue description becomes the primary context source)
4. Add escape hatch for Dylan only (not orchestrator): `orch spawn --emergency` or similar
5. Measure: track how often Dylan uses the escape hatch — that's the daemon's improvement backlog

**Open questions:**
- Does the orchestrator need to specify skill, or should the daemon always infer?
- How does custom spawn context work when the orchestrator can't spawn? Does it go in the issue description?
- What about interactive work (`--inline`)? That's a human action, not an orchestrator action — maybe it stays as a human-only command.

**Connection to delegation gravity:** This design also fights delegation gravity. If the orchestrator must scope every piece of work into an issue, that's a thinking step that can't be skipped. The issue forces the question: "what exactly am I asking for and why?" — which is the judgment that delegation gravity erodes.

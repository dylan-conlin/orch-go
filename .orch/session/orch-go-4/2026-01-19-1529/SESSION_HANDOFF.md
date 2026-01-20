# Session Handoff: Decidability Graphs

**Session:** orch-go-4
**Date:** 2026-01-19
**Duration:** ~2 hours (this conversation segment)

---

## TLDR

Created the **Decidability Graph** model - a framework for understanding authority boundaries in daemon-based orchestration. Key insight: the orchestrator hierarchy isn't about reasoning capability, it's about **context-scoping authority**. Workers can do any reasoning if given the right context; the irreducible orchestrator function is deciding *what context to load*.

---

## What Happened

### Model Creation
- Started from Dylan's handoff exploring "decidability graphs" - encoding decision dependencies, not just data dependencies
- Created `.kb/models/decidability-graph.md` with node taxonomy (Work/Question/Gate), authority levels, and edge traversal rules

### Key Insight Discovered
- Through discussion, discovered that hierarchy is about **context-scoping**, not capability
- Workers CAN answer framing questions if given multiple frames as context
- The orchestrator's irreducible function is deciding what context to load, not performing synthesis itself
- This refines the Strategic Orchestrator Model (2026-01-07)

### Dogfooding
- Created 3 question beads to test decidability mechanics
- Validated: questions block dependent work, closing unblocks
- Discovered friction: `bd close` requires Phase:Complete (wrong for questions), `answered` status doesn't unblock (only `closed`)

### Worker Authority Decision
- Discussed what workers should be allowed to do
- Decision: **Workers can create nodes, only orchestrators create blocking edges**
- Triage labels (`triage:review`) as safety mechanism
- Created `.kb/decisions/2026-01-19-worker-authority-boundaries.md`

### Spawned Architects
- Two architects answered the remaining open questions:
  1. **Encoding subtypes**: Use labels (`subtype:{factual|judgment|framing}`)
  2. **Q/G boundary**: Crisp but dynamic (lifecycle-based transitions)

---

## Artifacts Created

| Artifact | Purpose |
|----------|---------|
| `.kb/models/decidability-graph.md` | The decidability graph model |
| `.kb/decisions/2026-01-19-worker-authority-boundaries.md` | Worker authority rules |
| `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` | Added refinement section |
| `kb-227b01` | Quick decision: context-scoping as irreducible function |
| `kb-dc4a2e` | Constraint: bd close requires Phase:Complete for questions |
| `kb-fe6173` | Constraint: answered status doesn't unblock |

---

## Commits

```
29f22f60 model: add decidability graph for work coordination
```

---

## Questions Resolved

| Question | Answer |
|----------|--------|
| Can daemon resolve factual questions? | Yes, if scoped correctly. Scoping is orchestrator's job. |
| Is Q/G distinction crisp? | Crisp but dynamic - lifecycle transitions based on option space knowability |
| How to encode subtypes? | Labels: `subtype:{factual|judgment|framing}` |

---

## Future Work Identified

### Immediate
- Update decidability-graph.md with label convention and lifecycle model
- Fix `bd close` to not require Phase:Complete for questions
- Fix `answered` status to unblock dependencies

### Exploratory
- Dashboard graph visualization (frontier view)
- Context-scoping tooling (`orch scope`)
- Question → Gate transition mechanics

---

## Key Insight to Preserve

**The hierarchy is about context-scoping, not capability.**

```
Old Model:  Workers can't answer framing questions
New Model:  Workers don't have context to answer framing questions

Old Model:  Orchestrator does synthesis
New Model:  Orchestrator scopes what gets synthesized

Old Model:  Authority is role-based
New Model:  Authority is context-scoping-based
```

A "frame-evaluator" worker is possible - spawn with multiple frames as context, instruction to evaluate from outside. The orchestrator's contribution is *deciding those were the relevant frames to compare*.

---

## Branch Status

12 commits ahead of origin/master. Ready to push pending user approval.

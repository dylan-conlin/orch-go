---
status: active
blocks:
  - keywords:
      - worker add dependency
      - worker close issue
      - worker override decision
      - expand worker authority
---

# Decision: Worker Authority Boundaries

**Date:** 2026-01-19
**Status:** Accepted
**Enforcement:** convention
**Context:** Defining what workers can do autonomously in the decidability graph model

## Decision

**Workers can create nodes, only orchestrator can create blocking edges.**

Workers operate under Option B: Expand + Soft Block.

## What Workers CAN Do

| Action | Mechanism | Condition |
|--------|-----------|-----------|
| Create task/bug/feature issues | `bd create --type X` | Any discovered work |
| Create question entities | `bd create --type question` | Strategic unknowns discovered |
| Label high-confidence tactical work | `triage:ready` | Clear cause, obvious fix |
| Label uncertain/strategic work | `triage:review` | Questions, architectural, premise-challenging |
| Make implementation decisions | Document in artifacts | Within scoped task |
| Record constraints/decisions | `kb quick constrain/decide` | Knowledge capture |
| Surface questions | SYNTHESIS.md, comments | Doesn't create blocking relationships |

## What Workers CANNOT Do

| Action | Why Not | Alternative |
|--------|---------|-------------|
| Add dependencies to existing issues | Creates blocking relationships outside scope | Surface in SYNTHESIS.md, orchestrator decides |
| Close issues outside their scope | Affects work they weren't scoped for | Note in completion, orchestrator reviews |
| Override orchestrator decisions | Exceeds scoped authority | Escalate via SYNTHESIS.md |
| Label strategic questions `triage:ready` | Daemon shouldn't auto-process premise questions | Use `triage:review`, orchestrator validates |

## The Safety Mechanism: Triage Labels

```
Worker creates issue
    ↓
┌─────────────────────────────────────────┐
│  Is this high-confidence tactical work? │
│  (clear bug, obvious task, known fix)   │
└─────────────────────────────────────────┘
    ↓ YES                    ↓ NO
triage:ready              triage:review
    ↓                         ↓
Daemon can act           Orchestrator reviews
                              ↓
                         Orchestrator decides:
                         - Relabel triage:ready
                         - Add dependencies
                         - Modify/close
                         - Leave for discussion
```

## The Key Rule

**Workers expand the graph (create nodes), orchestrators constrain it (create blocking edges).**

This maps to the context-scoping insight: workers can do any reasoning within their scoped context, but decisions about how work relates to other work (dependencies, blocking) requires the cross-work visibility that orchestrators have.

## Examples

### Worker discovers premise is wrong

**Worker does:**
```bash
bd create --type question --title "Is our caching strategy fundamentally flawed?"
bd label <id> triage:review
# In SYNTHESIS.md: "Discovered potential premise issue - see orch-go-XXXX"
```

**Worker does NOT:**
```bash
bd dep add <parent-epic> <question-id>  # NO - creates blocking relationship
```

**Orchestrator then:**
```bash
# Reviews question, decides if it should block
bd dep add <parent-epic> <question-id>  # YES - orchestrator creates the edge
bd label <question-id> triage:ready     # Releases to daemon for investigation
```

### Worker finds related bug

**Worker does:**
```bash
bd create --type bug --title "Null check missing in related function"
bd label <id> triage:ready  # High confidence, clear fix
```

This is fine - tactical work with clear cause can go straight to daemon.

### Worker finds architectural concern

**Worker does:**
```bash
bd create --type question --title "Should we refactor auth before adding this feature?"
bd label <id> triage:review  # Needs orchestrator judgment
# In SYNTHESIS.md: "Blocked on architectural question - see orch-go-XXXX"
```

Worker surfaces but doesn't block. Orchestrator decides if it should gate the epic.

## Relationship to Decidability Graph Model

This decision operationalizes the model:

| Graph Concept | Operational Rule |
|---------------|------------------|
| Workers traverse Work edges | Workers execute scoped tasks |
| Orchestrator traverses Question edges | Orchestrator decides what blocks what |
| Creating nodes | Workers can do (expand graph) |
| Creating blocking edges | Orchestrator only (constrain graph) |
| Context scoping | Orchestrator decides what context workers get |
| Authority boundaries | Encoded in triage labels and dependency permissions |

## References

- `.kb/models/decidability-graph.md` - The model this operationalizes
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Orchestrator role (with 2026-01-19 refinement)
- `kb-227b01` - Context-scoping as irreducible orchestrator function

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-05-inv-document-decision-authority-criteria-agents.md

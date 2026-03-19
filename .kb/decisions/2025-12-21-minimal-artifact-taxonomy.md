# Decision: Minimal Artifact Taxonomy for Amnesia-Resilient Orchestration

**Status:** Accepted
**Date:** 2025-12-21
**Deciders:** Architect synthesis agent
**Context:** Synthesized from orch-go-4kwt epic (6 investigations on workspace lifecycle, knowledge promotion, session boundaries, beads-kb-workspace relationships, multi-agent synthesis, failure mode artifacts)

---

## Summary

Adopt a **minimal artifact set of 5 essential + 3 supplementary types** organized by three temporal lifecycles (ephemeral, persistent, operational). This taxonomy enables zero-context-loss resumption for any Claude instance.

---

## Decision

### The Five Essential Artifacts

| Artifact | Location | Creator | Temporal | Purpose |
|----------|----------|---------|----------|---------|
| **SPAWN_CONTEXT.md** | `.orch/workspace/{name}/` | `orch spawn` | Ephemeral | Agent initialization: skill, task, beads ID, deliverables |
| **SYNTHESIS.md** | `.orch/workspace/{name}/` | Worker agent | Ephemeral | Session outcome: D.E.K.N. summary, delta, recommendation |
| **Investigation** | `.kb/investigations/` | Worker agent | Persistent | Deep research: question → findings → answer |
| **Decision** | `.kb/decisions/` | Orchestrator | Persistent | Architectural choice: promoted from investigation |
| **Beads Comments** | `.beads/` (via `bd comment`) | Any agent | Operational | Phase tracking, investigation_path, blockers |

### The Three Supplementary Artifacts

| Artifact | Location | Creator | Purpose |
|----------|----------|---------|---------|
| **SESSION_HANDOFF.md** | `.orch/` | Orchestrator | Cross-session context for orchestrator resumption |
| **FAILURE_REPORT.md** | `.orch/workspace/{name}/` | Orchestrator on abandon | Failure context: mode, what tried, retry guidance |
| **kn entries** | `.kn/entries.jsonl` | Any agent | Quick decisions, constraints, tried/failed, questions |

### Three-Tier Temporal Model

Artifacts live where their lifecycle dictates, not where work happens:

- **Ephemeral (session-bound):** `.orch/workspace/` - SPAWN_CONTEXT.md, SYNTHESIS.md, FAILURE_REPORT.md
- **Persistent (project-lifetime):** `.kb/` - Investigations, decisions, guides
- **Operational (work-in-progress):** `.beads/` - Issues, comments, phase tracking

### D.E.K.N. as Universal Handoff Structure

All artifacts that enable resumption use D.E.K.N. structure:
- **Delta:** What changed/was discovered
- **Evidence:** How we know (primary sources)
- **Knowledge:** What it means (insights, constraints)
- **Next:** What should happen (recommendation)

---

## Context

The orchestration system evolved 6+ artifact types organically, but lacked:
1. Clear taxonomy of what's essential vs supplementary
2. Explicit lifecycle rules (when created, when archived)
3. Failure artifact capture (only successes had SYNTHESIS.md)
4. Standardized orchestrator handoff (SESSION_HANDOFF.md pattern)

Six investigations explored these gaps:
- Workspace lifecycle: Persist indefinitely by design
- Knowledge promotion: Manual by design (curation > accumulation)
- Session boundaries: Worker solved, orchestrator needs standardization
- Beads-KB-Workspace: Three-layer architecture already works
- Multi-agent synthesis: Current architecture sufficient
- Failure modes: Main gap - abandoned agents leave no context

---

## Options Considered

### Option A: Minimal Taxonomy (Chosen) ⭐

**5 essential + 3 supplementary artifacts with explicit lifecycle rules.**

**Pros:**
- Minimal set satisfying amnesia-resilience requirements
- Clear temporal categorization guides placement
- D.E.K.N. standardizes handoff
- Addresses failure capture gap

**Cons:**
- Requires discipline for orchestrator SESSION_HANDOFF.md
- Manual promotion paths remain (intentional)

### Option B: Collapse to Single Knowledge Location

**All artifacts in .kb/.**

**Pros:** Single discovery location

**Cons:** Loses ephemeral/persistent distinction; session noise pollutes knowledge base

**Rejected:** Temporal distinction is essential for lifecycle management

### Option C: Automate All Promotion

**Auto-promote investigations to decisions.**

**Pros:** Less manual work

**Cons:** Removes curation; floods decisions with noise

**Rejected:** Friction is intentional for quality control

### Option D: Remove SYNTHESIS.md

**Simplify agent protocol.**

**Pros:** Simpler completion flow

**Cons:** Loses structured handoff; orchestrator can't efficiently review

**Rejected:** SYNTHESIS.md is critical for `orch review` workflow

---

## Consequences

### Positive

- Fresh Claude can resume any work with artifacts alone
- `kb context` finds all investigations regardless of creator
- Failure modes captured, not just successes
- Clear rules for where artifacts go
- D.E.K.N. standardizes handoff structure

### Negative

- Orchestrator must maintain SESSION_HANDOFF.md discipline
- Workspaces persist indefinitely (disk space trade-off)
- Promotion remains manual (friction by design)

### Implementation Required

1. **FAILURE_REPORT.md template** - Create in `.orch/templates/`
2. **`orch abandon --reason` flag** - Add beads comment on abandon
3. **SESSION_HANDOFF.md template** - Create in `.orch/templates/`
4. **Orchestrator skill update** - Mandate SESSION_HANDOFF.md at session end
5. **`.orch/knowledge/spawning-lessons/` directory** - For failure patterns

---

## Related

- **Epic:** orch-go-4kwt (Amnesia-Resilient Artifact Architecture)
- **Investigation:** `.kb/investigations/2025-12-21-design-minimal-artifact-taxonomy.md`
- **Principle:** Session Amnesia (foundational constraint)

## Auto-Linked Investigations

- .kb/investigations/2026-02-28-design-session-debrief-artifact-system.md
- .kb/investigations/archived/2026-01-07-design-screenshot-artifact-storage-decision.md
- .kb/investigations/archived/2025-12-27-inv-add-skill-change-taxonomy-decision.md
- .kb/investigations/archived/2025-12-21-design-minimal-artifact-taxonomy.md

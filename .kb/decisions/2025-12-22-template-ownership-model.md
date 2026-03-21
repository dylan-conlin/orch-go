---
stability: foundational
---
## Summary (D.E.K.N.)

**Delta:** Template ownership is split by domain: kb-cli owns artifact templates (knowledge artifacts), orch-go owns spawn-time templates (agent lifecycle artifacts).

**Evidence:** kb-cli has embedded templates for investigation/decision/guide/research in `cmd/kb/create.go`. orch-go has SYNTHESIS/SPAWN_CONTEXT/FAILURE_REPORT/SESSION_HANDOFF in `pkg/spawn/context.go` and `.orch/templates/`.

**Knowledge:** Templates are split by what creates them, not where they live. kb-cli creates knowledge artifacts; orch-go creates agent lifecycle artifacts.

**Next:** Reference this decision when adding new templates - determine ownership based on whether the artifact is knowledge-centric or orchestration-centric.

---

# Decision: Template Ownership Model

**Date:** 2025-12-22
**Status:** Accepted
**Enforcement:** context-only

---

## Context

The orchestration ecosystem uses multiple template types for different purposes. Prior to this decision, it was unclear which tool owned which templates, leading to potential confusion about where to add new templates or modify existing ones.

Two CLIs interact with templates:
- **kb-cli** (`kb` command) - Knowledge base management
- **orch-go** (`orch` command) - Agent orchestration

---

## Options Considered

### Option A: All Templates in kb-cli
- **Pros:** Single source of truth for templates
- **Cons:** Conflates knowledge artifacts with orchestration artifacts; orch-go would depend on kb-cli for spawn functionality

### Option B: Split by Domain (Chosen)
- **Pros:** Clear separation of concerns; each tool owns templates it creates; reduces coupling
- **Cons:** Must remember which tool owns what

### Option C: All Templates in orch-go
- **Pros:** Centralized in orchestration tool
- **Cons:** kb-cli would need to depend on orch-go for templates; conflates concerns

---

## Decision

**Chosen:** Option B - Split by Domain

Templates are owned by the tool that creates the artifacts they produce.

### kb-cli Owns (Artifact Templates)

Knowledge artifacts that persist in `.kb/`:

| Template | Output Location | Purpose |
|----------|-----------------|---------|
| `INVESTIGATION.md` | `.kb/investigations/` | Investigation artifacts |
| `DECISION.md` | `.kb/decisions/` | Decision records |
| `GUIDE.md` | `.kb/guides/` | Reusable frameworks/patterns |
| `RESEARCH.md` | `.kb/investigations/` (with `research-` prefix) | External research documents |

**Implementation:** Embedded in `kb-cli/cmd/kb/create.go` as Go constants (`investigationTemplate`, `decisionTemplate`, `guideTemplate`, `researchTemplate`). Override via `~/.kb/templates/`.

**Created by:** `kb create investigation|decision|guide|research`

### orch-go Owns (Spawn-Time Templates)

Agent lifecycle artifacts that live in `.orch/`:

| Template | Output Location | Purpose |
|----------|-----------------|---------|
| `SPAWN_CONTEXT.md` | `.orch/workspace/{name}/` | Agent task context and guidance |
| `SYNTHESIS.md` | `.orch/workspace/{name}/` | Agent completion summary |
| `FAILURE_REPORT.md` | `.orch/workspace/{name}/` | Agent failure documentation |
| `SESSION_HANDOFF.md` | `.orch/` | Orchestrator session transitions |

**Implementation:** 
- `SPAWN_CONTEXT.md`: Generated from `SpawnContextTemplate` in `pkg/spawn/context.go`
- `SYNTHESIS.md`: Default at `DefaultSynthesisTemplate` in `pkg/spawn/context.go`, copied to `.orch/templates/`
- `FAILURE_REPORT.md`: Default at `DefaultFailureReportTemplate` in `pkg/spawn/context.go`
- `SESSION_HANDOFF.md`: Reference template in `.orch/templates/`

**Created by:** `orch spawn`, `orch abandon`, `orch complete`, session transitions

---

## Ownership Principle

**The tool that creates the artifact owns its template.**

Decision tree for new templates:
1. Does this template produce a knowledge artifact (investigation, decision, guide, research)?
   → **kb-cli owns it**
2. Does this template produce an orchestration artifact (spawn context, synthesis, failure report)?
   → **orch-go owns it**

---

## Structured Uncertainty

**What's tested:**
- ✅ kb-cli templates exist as embedded constants in `cmd/kb/create.go`
- ✅ orch-go templates exist in `pkg/spawn/context.go` and `.orch/templates/`
- ✅ Each tool generates its own templates without cross-dependency

**What's untested:**
- ⚠️ Whether users understand the ownership split without documentation
- ⚠️ Whether future templates will clearly fit one category

**What would change this:**
- Significant feature requiring tight coupling between kb and orch templates
- Decision to unify CLIs into single tool

---

## Consequences

**Positive:**
- Clear ownership reduces confusion about where to modify templates
- Each tool can evolve its templates independently
- No circular dependencies between kb-cli and orch-go

**Risks:**
- Users may not know which tool to use for which template
- Template customization requires knowing which tool's path to use (`~/.kb/templates/` vs `.orch/templates/`)

**Mitigation:**
- This decision document serves as reference
- Help text in each CLI should mention template location

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-15-inv-update-model-template-md-explicit.md
- .kb/investigations/archived/2026-01-17-inv-add-decision-linkage-investigation-template.md

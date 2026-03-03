# Decision: Accountability Architecture - Beads-First Implementation

**Date:** 2026-01-22
**Status:** accepted
**Deciders:** Dylan, Claude (orchestrator)
**Context:** Epic orch-go-kz7zr (Governance Infrastructure for Human-AI Systems)

## Decision

Implement accountability architecture by **exposing existing beads schema fields** rather than creating new infrastructure in skills or `.kb/governance/`.

## Context

Two investigations produced conflicting recommendations:

1. **Architect investigation** proposed hybrid architecture: structured authority in skill frontmatter + governance claims in `.kb/governance/authority.yaml`

2. **Reconciliation investigation** discovered 80% of decidability infrastructure already exists in beads schema (types.go) but isn't exposed via CLI

The architect examined populated data (issues.jsonl) but missed the schema definition which already has:
- `ResolutionType` enum (factual/judgment/framing)
- `Authority` enum on dependencies (daemon/orchestrator/human)  
- HOP entity tracking (Creator, Validations, EntityRef) for consequence bearer

## Consequences

### Implementation Path (Revised)

| Original Plan | Revised |
|---------------|---------|
| Phase 1: Skill frontmatter authority | **SKIP** - use beads fields |
| Phase 2: Authority injection in spawn | **PARTIAL** - reference beads issue fields |
| Phase 3: kb governance commands | **REVISE** - query beads instead of new artifact |
| Phase 4: Dashboard visibility | **KEEP** |

### Required Work

1. **CLI exposure** - Add `--resolution-type`, `--domain` to `bd create`; add `--authority` to `bd dep add`
2. **Query integration** - Add `--authority` filter to `bd ready` for daemon-safe frontier
3. **Backfill strategy** - Existing issues need resolution_type populated
4. **Dashboard** - Authority visibility panel (unchanged from architect proposal)

### What We Don't Build

- No `.kb/governance/authority.yaml` (beads is the location)
- No skill frontmatter `authority:` section (beads tracks per-issue authority)
- No `kb authority` command (use `bd` commands instead)

## Naming

The layer is called **Accountability Architecture**. The implementation lives in beads.

## References

- Architect investigation: `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md`
- Reconciliation investigation: `.kb/investigations/2026-01-22-inv-reconcile-architect-accountability-architecture-proposal.md`
- Epic: orch-go-kz7zr
- Naming decision: kb-d41a0b

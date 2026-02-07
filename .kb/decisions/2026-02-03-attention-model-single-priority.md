# Decision: Attention Model Uses Single Priority (Human), Not Role-Aware Scoring

**Date:** 2026-02-03
**Status:** Accepted
**Deciders:** Dylan, Meta-Orchestrator

## Context

The attention reconciliation layer design (from pressure-test investigation) proposed role-aware priority scoring:
- Different priorities for Dylan (human) vs orchestrators vs daemon
- API would accept `role` parameter to return appropriately prioritized items
- Rationale: "different roles need different prioritization"

## Decision

**Use single priority model (human priorities) for the attention API. Do not implement role-aware scoring.**

Additionally: **Daemon continues using `bd ready` independently** - this is intentional separation, not a gap to fix.

## Rationale

### Why single priority is sufficient

1. **Collaborative workflow:** Dylan typically works in direct collaboration with an orchestrator or meta-orchestrator, looking at the same Work Graph together. Different priority views would be confusing, not helpful.

2. **One primary consumer:** The attention surface serves Dylan. When working with orchestrators, they're in the same session, same screen, making decisions together.

3. **Daemon doesn't use attention API:** Daemon uses `bd ready` + labels, so building daemon-specific priorities in the attention API serves no consumer.

### Why daemon separation is intentional

- **Daemon's job:** Mechanical batch execution of `triage:ready` work when Dylan is not around (overnight, background)
- **Work Graph's job:** Show what needs attention when Dylan IS around
- **No divergence problem:** When actively working, Dylan directs spawns. Daemon defers. The `triage:ready` label is explicit approval for daemon to act.

## Consequences

### Positive
- Simpler architecture (no role parameterization)
- Single source of truth for priority
- Less code to maintain

### Negative
- If future consumers need different priorities, we'd need to add it then
- YAGNI bet - we're betting role-aware isn't needed

## Alternatives Considered

**Role-aware scoring (rejected):**
- API accepts `role` parameter (human, orchestrator, daemon)
- Different priority weights per role
- Rejected because: only one consumer (Dylan), collaborative workflow means shared view

**Daemon consumes attention API (rejected):**
- Daemon would use `/api/attention?role=daemon` for spawn ordering
- Rejected because: daemon's current model (bd ready + labels) works, creates unnecessary coupling

## References

- Original design: `.kb/investigations/2026-02-02-inv-pressure-test-work-graph-unified.md`
- Discussion: Meta-orchestrator session 2026-02-03

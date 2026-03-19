# Decision: Single Daemon with Orchestration Home

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** 
**Supersedes:** 
**Superseded-By:** `.kb/decisions/2026-02-25-project-group-model.md`


**Date:** 2026-01-16
**Status:** Superseded
**Context:** Daemon architecture for multi-project orchestration

## Problem

Dylan works across multiple projects (orch-go, orch-knowledge, kb-cli, beads, price-watch, specs-platform). How should the daemon handle work across these projects?

## Options Considered

### Option A: Single daemon in orchestration home (CHOSEN)

Daemon runs in orch-go, polls only orch-go's `.beads/` directory. Cross-project work is expressed as issues in orch-go with `--workdir` context.

```
Orchestrator (strategic) → decides what to work on across all projects
                        → creates/labels issues in orch-go
Daemon (tactical)       → executes triage:ready work from single queue
                        → uses --workdir for cross-project spawns
```

### Option B: Cross-project daemon polling

Single daemon polls multiple projects' `.beads/` directories, spawning work from any project.

### Option C: Separate daemon per project

Each project runs its own daemon instance via launchd, managing its own work queue independently.

## Decision

**Option A: Single daemon with orchestration home.**

## Rationale

### Why not Option B (cross-project polling)?

- **Competing priority queues** - If orch-go has a P1 and kb-cli has a P1, who decides which runs first?
- **Diffuses strategic control** - Orchestrator's job is deciding what to work on. Cross-project daemon makes daemon the decision-maker.
- **Coordination chaos** - No single point of visibility or control.

### Why not Option C (separate daemons)?

- **Capacity collision** - Multiple daemons spawn agents simultaneously, exceeding Claude API rate limits. Single daemon enforces `--max-agents 5`. Multiple daemons each enforce their own limit with no coordination.
- **Resource overhead** - Multiple launchd processes, multiple polling loops.
- **No unified visibility** - Dashboard would need to aggregate across daemons.
- **Conflicting work** - orch-go daemon and kb-cli daemon could both spawn agents on the same project.
- **Orchestration home loses meaning** - The pattern exists precisely for unified coordination.

### Why Option A works

- **Cross-project supported** - `orch spawn --workdir ~/kb-cli feature-impl "task"` works today
- **Capacity managed** - Single `--max-agents` limit for all work
- **Strategic clarity** - Orchestrator decides priorities, daemon executes
- **Unified visibility** - Dashboard shows all work from single source

## Consequences

- All daemon-driven work must be expressed as issues in orch-go
- Cross-project work requires orchestrator to create issue with appropriate `--workdir` context
- Projects without their own daemon cannot have autonomous work processing
- This is intentional: autonomous processing requires strategic oversight

## References

- `.kb/guides/daemon.md` - Daemon operational guide
- `.kb/models/daemon-autonomous-operation/model.md` - Daemon behavior model
- `CLAUDE.md` - Orchestration Home Directory section

## Auto-Linked Investigations

- .kb/investigations/archived/cross-project-operations/2026-01-06-inv-cross-project-daemon-single-daemon.md
- .kb/investigations/2026-03-05-inv-design-review-daemon-capacity-tracking.md

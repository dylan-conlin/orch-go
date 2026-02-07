# Decision: Three-Tier Workspace Hierarchy

**Date:** 2026-01-17
**Status:** Accepted
**Deciders:** Dylan

## Context

The orch-go system spawns agents of different types (workers, spawned orchestrators, interactive sessions) that have different execution patterns and completion requirements. Early investigations found confusion about "orphaned workspaces" that turned out to be a misunderstanding of the tier system.

## Decision

**Workspaces are organized into three distinct types with different locations, naming conventions, and completion artifacts.**

| Type | Location | Naming | Completion Artifact | Beads Tracked |
|------|----------|--------|---------------------|---------------|
| **Worker** | `{project}/.orch/workspace/og-{skill}-{slug}-{date}-{hex}/` | 4-char hex suffix | SYNTHESIS.md (full) or none (light) | Yes |
| **Spawned Orchestrator** | `{project}/.orch/workspace/og-orch-{slug}-{date}-{hex}/` | 4-char hex suffix | SESSION_HANDOFF.md | No |
| **Interactive Session** | `~/.orch/session/{date}/` | Date-based | SESSION_HANDOFF.md | No |

## Rationale

### Evidence: Confusion About "Orphaned Workspaces"

Investigation of "200+ workspaces without SYNTHESIS.md" revealed:
- 218 were light-tier workers (no SYNTHESIS.md expected)
- 46 workers with SYNTHESIS.md (correctly completed)
- 12 orchestrators with SESSION_HANDOFF.md (correctly completed)
- ~10 actually incomplete (test spawns, abandoned sessions)

The "orphan problem" was a misunderstanding of the tier system.

### Why Three Types?

1. **Workers** need project-local workspaces to:
   - Run project-specific tests
   - Access codebase with correct paths
   - Keep evidence close to the code

2. **Spawned Orchestrators** need per-goal workspaces to:
   - Isolate different coordination goals
   - Track progress independently
   - Enable concurrent orchestration

3. **Interactive Sessions** need daily directories to:
   - Maintain continuity across multiple breaks
   - Simplify "Landing the Plane" for humans
   - Support cross-project work patterns

### Why Hex Suffix for Spawned?

Prior to fix (Jan 5, 2026), purely deterministic naming `{proj}-{skill}-{slug}-{date}` caused same-day sessions to overwrite each other's artifacts. The 4-char random hex provides 65,536 variations per day per task.

### Why Date-Based for Interactive?

Interactive sessions use daily directories because humans think in "days of work" rather than "goal-atomic sessions." Multiple goals in one day share one SESSION_HANDOFF.md by design.

## Consequences

### What This Means for Code

1. **Spawn command determines workspace type** - Based on skill tier mapping
2. **Context file indicates type** - SPAWN_CONTEXT.md vs ORCHESTRATOR_CONTEXT.md vs META_ORCHESTRATOR_CONTEXT.md
3. **Completion verification uses .tier** - `VerifyCompletionWithTier()` enforces correct artifacts

### Workspace State Detection

| Context File | .tier | Artifact Present | State |
|--------------|-------|------------------|-------|
| SPAWN_CONTEXT.md | light | (none expected) | Completed |
| SPAWN_CONTEXT.md | full | SYNTHESIS.md | Completed |
| SPAWN_CONTEXT.md | full | (missing) | Incomplete |
| ORCHESTRATOR_CONTEXT.md | orchestrator | SESSION_HANDOFF.md | Completed |
| ORCHESTRATOR_CONTEXT.md | orchestrator | (missing) | Active or Abandoned |

## Alternatives Considered

1. **Single workspace type for all agents**
   - Rejected: Different agents have different needs (project-local vs global, per-goal vs per-day)

2. **Server-side workspace management**
   - Rejected: File-based workspaces enable offline access, cross-tool visibility, and simple debugging

3. **Automatic workspace detection without .tier file**
   - Rejected: Explicit tier file enables reliable completion verification

## Related

- **Source Investigations:**
  - `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md`
  - `.kb/investigations/2026-01-05-inv-orchestrator-workspaces-clear-visual-distinction.md`
  - `.kb/investigations/2026-01-09-inv-create-orchestrator-workspace-session-start.md`
- **Model:** `.kb/models/workspace-lifecycle-model.md`
- **Guide:** `.kb/guides/workspace-lifecycle.md`

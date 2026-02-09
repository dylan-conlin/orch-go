# Decision: Git Isolation via Worktree Plus Branch

**Date:** 2026-02-09
**Status:** Accepted
**Deciders:** Dylan, Architect Agent

## Context

Orch-go currently runs concurrent agents in one working tree. A post-mortem showed daemon auto-complete closed 22 issues with zero commits while 81+ uncommitted files interleaved in the shared tree. Prior strategy guidance (Dec 23) recommended branch-per-agent but did not evaluate worktrees.

This decision must support:

- Daemon auto-complete and manual `orch complete`
- Both Claude and GPT backends
- Beads JSONL lifecycle tracking
- Cross-project resolution and current workspace conventions

## Decision

**Use worktree-per-agent plus branch-per-agent as the default git isolation model for tracked spawns.**

Core rules:

- Each tracked agent gets a dedicated worktree at `.orch/worktrees/<workspace>`.
- Each worktree is attached to an explicit branch (`agent/<beads-id>` by default).
- Agent runtime cwd is the worktree path (not source repo root).
- Verification and merge operations use the worktree path and branch metadata.
- Source repo path remains canonical for beads/project identity and cross-project routing.

## Rationale

### Why not branch-only

Branches isolate commit lineage, not uncommitted filesystem state. In a shared tree, concurrent agents can block each other's checkout and contaminate verification with uncommitted changes.

### Why not worktree-only

Detached worktrees isolate state but create ambiguous integration ownership and harder lifecycle automation. Explicit branches give deterministic merge and cleanup behavior.

### Why hybrid

Hybrid gives both required properties:

- **Isolation correctness:** separate working directories per agent
- **Integration operability:** explicit branch identity for rebase/merge/cleanup

## Consequences

### Positive

- Eliminates shared-tree uncommitted bleed between agents
- Prevents branch checkout contention in concurrent workflows
- Enables per-agent verification scope for daemon and manual completion
- Preserves readable integration history and deterministic rollback

### Negative

- Adds lifecycle complexity (worktree create/remove, branch cleanup)
- Requires metadata/schema updates across spawn and completion paths
- Introduces new cleanup failure modes (orphan worktrees/branches)

## Phased Implementation Plan

### Phase 1: Metadata and Contracts (no behavior change)

1. Extend agent manifest/state schema with:
   - `source_project_dir`
   - `git_worktree_dir`
   - `git_branch`
2. Keep existing fields backward compatible.
3. Update status/export/readers to display new fields.

**Exit criteria:** Existing flows still pass; new metadata is persisted and readable.

### Phase 2: Spawn Isolation

1. In spawn pipeline, create branch + worktree before launching agent.
2. Launch backend session (Claude/GPT) with cwd set to `git_worktree_dir`.
3. Keep workspace artifacts under `.orch/workspace/<workspace>`; store pointers to worktree metadata.

**Exit criteria:** Two simultaneous tracked spawns run in separate worktrees with no shared uncommitted visibility.

### Phase 3: Completion and Verification Integration

1. Run git-diff/commit-evidence/build gates against `git_worktree_dir`.
2. Merge policy at completion:
   - Rebase branch onto target base
   - Merge fast-forward only (or fail with explicit remediation)
3. Preserve beads closure semantics: close only after verification + merge success.

**Exit criteria:** Daemon and manual completion reject unrelated working-tree noise and only complete merged/verified agent work.

### Phase 4: Cleanup and Operations

1. Extend abandon/cleanup to remove worktree and optionally delete agent branch.
2. Add stale worktree janitor command and telemetry.
3. Add safety rails for partial cleanup failures (idempotent retries, clear operator output).

**Exit criteria:** No long-lived orphan worktrees/branches beyond configured threshold.

## Guardrails

- Keep beads default/project resolution tied to `source_project_dir`, not worktree path.
- Never infer completion from shared-root diffs once worktree mode is enabled.
- Roll out behind a feature flag to permit fallback during migration.

## References

- Investigation: `.kb/investigations/2026-02-09-inv-git-isolation-strategy-multi-agent.md`
- Incident: `.kb/investigations/2026-02-09-inv-post-mortem-daemon-overnight-ghost-completions.md`
- Prior strategy: `.kb/investigations/archived/2025-12-23-inv-investigate-git-branching-strategies-swarm.md`

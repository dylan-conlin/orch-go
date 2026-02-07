# Decision: File-Based Workspace State Detection

**Date:** 2026-01-17
**Status:** Accepted
**Deciders:** Dylan

## Context

When implementing `orch clean --stale` for workspace cleanup, the initial implementation used beads API calls to check completion status. This was unacceptably slow (>2 minutes for 295 workspaces due to beads daemon startup overhead).

## Decision

**Workspace state is determined from filesystem metadata, not API calls.**

Completion detection uses these files:

| Tier | Completion Indicator |
|------|---------------------|
| Full | `SYNTHESIS.md` exists |
| Orchestrator | `SESSION_HANDOFF.md` exists |
| Light | `.beads_id` exists (assumed complete if no active session) |

Additional metadata:
- `.tier` - Determines which artifact is expected
- `.session_id` - Links to OpenCode session for resume
- `.spawn_time` - Enables age-based cleanup

## Rationale

### Evidence: Performance Comparison

| Approach | Time (295 workspaces) |
|----------|----------------------|
| Beads API per workspace | >2 minutes |
| File-based detection | <1 second |

### Why Beads Is Slow

- Beads daemon takes 5+ seconds to start when cold
- Each `bd show <id>` invokes the daemon
- 295 workspaces × 5 seconds = 25+ minutes worst case

### Why File-Based Works

Workspaces are self-describing via metadata files:
- `.tier` tells you what artifact is expected
- Artifact presence confirms completion
- No network calls, no daemon startup

### Trade-offs

**Accuracy:** File-based detection can show completed for agents that created SYNTHESIS.md but didn't actually finish. This is rare and acceptable for cleanup purposes.

**Freshness:** Files may be stale if written but not updated. For cleanup (7+ day old workspaces), staleness is irrelevant.

## Consequences

### What This Means for Code

1. **`orch clean --stale` uses file-based detection** - Fast bulk operations
2. **`orch complete` uses beads for authoritative status** - Single-agent verification
3. **`orch doctor --sessions` cross-references files and API** - Diagnostic commands can be slower

### When to Use Each Approach

| Use Case | Approach |
|----------|----------|
| Bulk cleanup (100+ workspaces) | File-based |
| Single agent completion | Beads API |
| Dashboard display | Cache with API fallback |
| Cross-reference diagnostics | Both (API for accuracy) |

## Alternatives Considered

1. **Always use beads API**
   - Rejected: Unacceptably slow for bulk operations

2. **Cache beads status in workspace files**
   - Rejected: Adds complexity; current file-based indicators are sufficient

3. **Background beads daemon**
   - Rejected: Adds operational complexity; file-based is simpler

## Related

- **Source Investigation:** `.kb/investigations/2026-01-06-inv-define-workspace-cleanup-strategy-context.md`
- **Code:** `cmd/orch/clean_cmd.go:archiveStaleWorkspaces()`
- **Guide:** `.kb/guides/workspace-lifecycle.md`
- **Constraint:** Registry is caching layer, not source of truth (all data exists in OpenCode/tmux/beads)

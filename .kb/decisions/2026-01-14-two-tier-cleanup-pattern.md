# Decision: Two-Tier Cleanup Pattern

**Date:** 2026-01-14
**Status:** Accepted
**Deciders:** Dylan

## Context

OpenCode sessions accumulated to 266 (vs ~29 expected) because cleanup was event-driven only. Failed spawns, manual sessions, and edge cases bypassed the `abandon`/`complete` cleanup paths.

## Decision

Resources with unpredictable lifecycles require **two-tier cleanup**:

1. **Tier 1: Event-based cleanup** - Clean up on explicit lifecycle events (abandon, complete)
2. **Tier 2: Periodic background cleanup** - Catch orphans missed by event-based cleanup

## Rationale

### Why Event-Based Alone Fails

Event-based cleanup assumes:
- All lifecycle events fire
- Code paths don't crash mid-cleanup
- All creation paths have corresponding cleanup

Reality:
- Spawns can fail after session creation but before workspace tracking
- Processes crash, losing cleanup opportunities
- Manual creation bypasses instrumented paths

### The Gap

Current cleanup paths: `orch abandon`, `orch complete`, `orch clean --sessions`
- All require explicit invocation or workspace context
- Sessions created outside tracked paths become orphans
- 266 - 29 = 237 orphaned sessions from lifecycle gaps

## Consequences

### Implementation

**Tier 1 (exists):** Cleanup on abandon/complete continues to work for normal lifecycle.

**Tier 2 (add):** Daemon runs `cleanStaleSessions()` every 6 hours for sessions >7 days old.

```yaml
# ~/.orch/config.yaml
cleanup:
  sessions:
    enabled: true
    interval: 6h
    age_days: 7
```

### When to Apply This Pattern

Use two-tier cleanup when:
- Resource has unpredictable lifecycle (sessions, processes, temp files)
- Creation can happen outside instrumented paths
- Lifecycle events can fail to fire

Don't use when:
- Resource has deterministic lifecycle
- All creation/destruction paths are instrumented
- Orphans are impossible by design

## Alternatives Considered

1. **Reference counting** - Track all session references, cleanup when refcount=0
   - Rejected: Complex, brittle (crashes leave dangling refs), doesn't handle manual creation

2. **Manual cleanup documentation** - Just tell users to run `orch clean --sessions`
   - Rejected: Doesn't work - 266 accumulated sessions proves users forget

3. **Cleanup on OpenCode server startup** - Auto-cleanup when server restarts
   - Rejected: Servers rarely restart, doesn't help between restarts

## Related

- **Source:** `.kb/investigations/2026-01-11-design-opencode-session-cleanup-mechanism.md`
- **Implementation:** `cmd/orch/clean_cmd.go` (cleanStaleSessions function)
- **Principle:** Pressure Over Compensation - create pressure to use proper cleanup, don't silently compensate

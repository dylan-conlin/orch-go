# Decision: Registry Contract - Spawn-Cache Only

**Date:** 2026-01-14
**Status:** Superseded (2026-03-12) — pkg/registry/ removed entirely per CLAUDE.md "No Local Agent State" constraint. The principle (query authoritative sources directly) survives as the architectural constraint.
**Deciders:** Dylan

## Context

The registry abandonment bug ("orch abandon doesn't remove agent from registry") appeared to be a simple oversight. Investigation revealed it's a systemic pattern: registry state management methods exist but are never called by lifecycle commands.

## Decision

**The registry is a spawn-time cache for metadata lookups, NOT a lifecycle state tracker.**

Commands should:
- **Write on spawn** - Register agent with metadata
- **Read for lookups** - Find session IDs, tmux windows, beads IDs
- **NOT update on lifecycle events** - State changes don't update registry

## Rationale

### Evidence: State Methods Never Called

Registry defines state transition methods:
- `Abandon()` at line 432
- `Complete()` at line 450
- `Remove()` at line 470

Production usage found:
- `spawn_cmd.go`: Calls `Register()` and `Save()` (write)
- `status_cmd.go`: Calls `ListActive()`, `ListCompleted()` (read)
- `abandon_cmd.go`: Calls `Find()` (read) but NOT `Abandon()` (no update)
- `complete_cmd.go`: Doesn't import registry package at all
- `clean_cmd.go`: Doesn't import registry package at all

### Why This Is Acceptable

The registry serves a specific purpose: **fast lookup of agent metadata during spawn and status checks**.

For actual agent state, commands derive from authoritative sources:
- OpenCode API (session state)
- Beads (issue status)
- Tmux (window existence)

Registry showing "stale" active state is fine because status command checks real sources.

## Consequences

### What This Means for Code

1. **Don't add registry updates to lifecycle commands** - They're not needed
2. **Registry.Abandon/Complete/Remove are dead code** - Can be deprecated/removed
3. **Status uses registry for lookup, not state** - Then checks OpenCode/beads for real state

### Documentation

Add to Registry struct:
```go
// Registry is a spawn-time cache for agent metadata.
// It provides fast lookup of session IDs, tmux windows, and beads IDs.
// It is NOT a lifecycle state tracker - agents remain "active" in registry
// after completion/abandonment. Use OpenCode API and beads for current state.
type Registry struct {
    // ...
}
```

### When to Check Real Sources

| Need | Source |
|------|--------|
| Session ID for agent | Registry (cached at spawn) |
| Tmux window name | Registry (cached at spawn) |
| Is agent still running? | OpenCode API |
| Is issue closed? | Beads |
| Agent current phase | OpenCode messages |

## Alternatives Considered

1. **Implement full lifecycle management** - Update registry on abandon/complete/clean
   - Rejected: High cost, unclear value (commands work fine without it)

2. **Remove registry entirely** - Force all lookups to primary sources
   - Rejected: Status/abandon rely on registry for session ID lookups

## Related

- **Source:** `.kb/investigations/2026-01-11-inv-registry-abandonment-workflow-validate-simple.md`
- **Code:** `pkg/registry/registry.go`
- **Principle:** Document what the system actually does, not what it was designed to do

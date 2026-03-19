---
stability: foundational
---
# Decision: Schema Migration Pattern

**Date:** 2026-01-14
**Status:** Accepted
**Deciders:** Dylan

## Context

Session resume `--check` returned exit 1 despite handoffs existing because window-scoping was added to discovery logic without data migration. Old handoffs at `.orch/session/latest` became undiscoverable when code started checking `.orch/session/{window-name}/latest`.

## Decision

Schema changes require **both**:

1. **Backward-compatible discovery** - New code checks new path first, falls back to old path
2. **Optional migration tooling** - Explicit command to move data to new structure

Never break existing data. Never require migration before usage works.

## Rationale

### The Anti-Pattern We Hit

```
1. Add window-scoping feature (commit 3385796c)
2. Discovery code now expects window-scoped paths
3. Existing handoffs remain in non-window-scoped structure
4. Result: All pre-existing handoffs become undiscoverable
```

This is "schema migration without data migration" - a classic breaking change.

### Why Backward Compatibility + Optional Migration

**Immediate recovery:** Fallback makes existing data work immediately without user action.

**Pressure to migrate:** Warning creates pressure to use proper structure.

**User control:** Migration happens when user chooses, not silently.

## Consequences

### Implementation Pattern

```go
func discoverSessionHandoff() (string, error) {
    // 1. Try new path first
    newPath := filepath.Join(".orch/session", windowName, "latest")
    if exists(newPath) {
        return newPath, nil
    }

    // 2. Fall back to old path with warning
    oldPath := filepath.Join(".orch/session", "latest")
    if exists(oldPath) {
        log.Warn("Using legacy handoff. Run 'orch session migrate' to update.")
        return oldPath, nil
    }

    // 3. Neither exists
    return "", fmt.Errorf("no handoff found (checked: %s, %s)", newPath, oldPath)
}
```

### Migration Command

```bash
orch session migrate
# Discovers old handoffs, moves to window-scoped structure
# Reports what was migrated
```

### When to Apply

Apply to ANY schema change that affects discovery/lookup:
- File path structure changes
- Database schema changes
- Config format changes
- API response format changes

## Alternatives Considered

1. **Automatic migration on discovery** - Silently move files when old structure found
   - Rejected: Surprising file moves, dangerous with concurrent access

2. **Fallback only (no migration)** - Just support both structures forever
   - Rejected: Old structure persists indefinitely, no pressure to upgrade

3. **Hard break with migration required** - Fail until user migrates
   - Rejected: Breaks all existing workflows, violates zero-disruption principle

## Related

- **Source:** `.kb/investigations/2026-01-13-design-session-resume-discovery-failure.md`
- **Implementation:** `.kb/investigations/2026-01-13-inv-implement-backward-compatible-session-resume.md`
- **Principle:** Coherence Over Patches - fix the structure, don't patch around it

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-26-inv-evaluate-whether-web-markdown-mcp.md

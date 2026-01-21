# Decision: Strategic-First Gate Changed to Advisory-Only

**Date:** 2026-01-21
**Status:** Accepted
**Context:** Spawn command gating behavior

## Problem

The strategic-first gate was blocking tactical spawns in hotspot areas, requiring `--force` flag to override. In practice:

1. **Operational work blocked** - "Dashboard is down" doesn't need an architect, needs someone to restart services
2. **Keyword matching too broad** - Triggering on "serve", "sse", "dashboard" catches trivial ops work
3. **Force becomes default** - When you always need `--force`, the gate teaches bypass, not strategic thinking
4. **Compounds with other gates** - Already hit `--bypass-triage`, then strategic-first, then skill warnings. Death by flags.

## Decision

Change strategic-first gate from blocking to advisory-only. Show the hotspot warning but don't block execution.

## Implementation

Changed `cmd/orch/spawn_cmd.go`:

```go
// Before: Blocking
if !daemonDriven && !spawnForce && !isStrategicSkill {
    fmt.Fprintln(os.Stderr, "🚫 STRATEGIC-FIRST ORCHESTRATION")
    // ... blocking message ...
    return fmt.Errorf("strategic-first gate: architect required in hotspot area")
}

// After: Advisory
if !daemonDriven && !spawnForce && !isStrategicSkill {
    fmt.Fprint(os.Stderr, hotspotResult.Warning)
    fmt.Fprintln(os.Stderr, "💡 Consider: spawn architect first for strategic approach in hotspot area")
    fmt.Fprintln(os.Stderr, "")
}
```

## Rationale

The hotspot detection is valuable information - knowing you're touching a high-churn area is worth surfacing. But blocking tactical work and forcing `--force` just trains everyone to ignore it.

**Pattern:** Gate Over Remind works for critical safety checks. For advisory guidance, Remind Over Gate is better.

## Consequences

- `--force` flag is now redundant for strategic-first (kept for compatibility)
- Hotspot warnings still appear, providing situational awareness
- Tactical spawns proceed without interruption
- Users can still choose to spawn architect first based on the warning

## Alternatives Considered

| Option | Description | Why Not |
|--------|-------------|---------|
| Remove entirely | Delete the gate | Loses the signal value |
| Skill-aware | Only gate investigation, not systematic-debugging | More complex logic |
| Threshold-based | Only trigger above 10+ patches | Current threshold (3) is too low, but arbitrary |

## References

- Code: `cmd/orch/spawn_cmd.go:887-916`
- Pattern: Gate Over Remind vs Remind Over Gate

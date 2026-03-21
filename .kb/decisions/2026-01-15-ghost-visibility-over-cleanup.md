# Decision: Ghost Visibility Over Cleanup

**Date:** 2026-01-15
**Status:** Accepted
**Enforcement:** context-only
**Deciders:** Dylan, Meta-orchestrator

---

## Context

The orch system never has "clean state" - ghost agents accumulate indefinitely. Status shows "45 idle, 0 running" but concurrency checks count all 45 as active, blocking new spawns.

Initial framing: "We need garbage collection to clean up ghosts."

---

## Decision

**Filter ghosts at query time instead of cleaning them up.**

Ghosts can exist indefinitely. The fix is how we count and display them, not whether they exist.

### Two-Threshold Filtering

**For concurrency limit (aggressive):**
```
Counts toward limit = running OR (idle AND last_activity < 1h)
```
- Only truly active work blocks new spawns
- Ghosts idle >1h don't count

**For default display (conservative):**
```
Show by default = running OR idle < 4h OR Phase: Complete (any age)
```
- Phase: Complete always shows (needs orchestrator action)
- True ghosts (idle >4h, no Phase: Complete) hidden by default
- `--all` flag shows everything

---

## Rationale

### Why not garbage collection?

| Aspect | Cleanup | Filtering |
|--------|---------|-----------|
| Reversible? | No (deleted = gone) | Yes (`--all` shows everything) |
| Edge cases | Many (paused vs abandoned?) | Few (just time threshold) |
| Fights architecture? | Yes (OpenCode sessions persist by design) | No (accepts persistence) |
| Solves actual pain? | Indirectly | Directly |

### The actual pain points

1. **Ghosts block new spawns** (counting problem, not existence problem)
2. **Ghosts clutter the view** (display problem, not data problem)

Both are solved at the query layer without touching underlying data.

### Why split thresholds?

| Concern | Goal | Threshold |
|---------|------|-----------|
| Concurrency | Don't let ghosts block new work | Aggressive (1h) |
| Display | Don't lose things needing action | Conservative (4h + state) |

Phase: Complete agents might be old but still need review. Hiding them loses the signal.

---

## Implementation

Two functions:
- `isActiveForConcurrency(agent)` - tight 1h threshold
- `isVisibleByDefault(agent)` - 4h threshold + Phase: Complete exception

Affects:
- `cmd/orch/spawn.go` - concurrency check
- `cmd/orch/status.go` - default display
- `cmd/orch/serve_agents.go` - dashboard filtering

---

## Consequences

**Positive:**
- New spawns no longer blocked by ghosts
- Status output shows actionable items by default
- No risk of deleting paused work
- Works with OpenCode's persistence model

**Negative:**
- Disk space accumulates (but already does via OpenCode)
- Requires understanding "default view vs --all"
- Thresholds may need tuning based on experience

---

## References

- `.kb/models/agent-lifecycle-state-model/model.md` - Four-layer architecture that makes cleanup complex
- Meta-orchestrator session 2026-01-15 - Original discussion

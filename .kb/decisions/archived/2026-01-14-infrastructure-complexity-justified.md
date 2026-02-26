# Infrastructure Complexity Justified

**Date:** 2026-01-14
**Status:** Accepted
**Context:** After repeated dashboard failures, evaluated whether 3-service overmind architecture is too complex

## Decision

Keep the current architecture (overmind managing opencode + api + web). The complexity serves reliability.

## What We Evaluated

| Option | Outcome |
|--------|---------|
| Simplify to fewer services | Rejected - tighter coupling, harder debugging |
| Replace overmind with script | Rejected - lose restart-on-crash, coordinated lifecycle |
| Consolidate web into api | Rejected - lose hot reload, marginal gain |

## Why Complexity is Justified

Today's failures (Jan 14, 2026) were NOT architectural:
- Stale `.overmind.sock` → cleanup problem (fixed in .zshrc)
- launchd conflict on port 4096 → duplicate process managers (fixed by removing plist)
- Plugin crash → OpenCode's plugin loader has no isolation (upstream issue)

The cascading failure pattern:
```
Plugin error → OpenCode 500 → orch status failed → API couldn't fetch → Dashboard "disconnected"
```

This reveals gaps in monitoring, not excess complexity.

## Action Items

1. **Fix health check gap** - `orch doctor` should verify dashboard can fetch agent data, not just port open
2. **ONE process manager rule** - Document that overmind is the ONLY process manager for dev services
3. **Minimal plugin set** - Until OpenCode improves plugin isolation, plugins are a risk factor

## Alternatives Considered

- **Production simplification** - Keep dev complex, make prod simpler. Deferred - not building for prod yet.
- **Plugin removal** - Remove all plugins. Rejected - session-resume is valuable. Fixed to v2 API instead.

## References

- `.kb/guides/dev-environment-setup.md` - Updated with restart troubleshooting
- `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Prior dashboard reliability decision
- Orchestrator skill "Observability Architecture (Option A+)" section - Dashboard as tier-0

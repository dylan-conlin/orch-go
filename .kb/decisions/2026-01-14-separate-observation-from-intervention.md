# Decision: Separate Observation from Intervention

**Date:** 2026-01-14
**Status:** Accepted
**Deciders:** Dylan

## Context

The coaching plugin had 8 bugs and 2 abandoned attempts because injection logic was architecturally coupled to metric collection. Injection required ephemeral session state but should depend on persistent metrics, creating restart brittleness and "injection doesn't fire after X" bugs.

## Decision

**Observation (passive) and intervention (active) must be separate concerns with separate lifecycles.**

When a system needs to both observe behavior and respond to it:
- Observation writes to persistent storage
- Intervention reads from persistent storage
- They share data, not code paths

## Rationale

### The Coupling Problem

```
Coaching Plugin Architecture (before):
┌─────────────────────────────────────────┐
│ tool.execute.after hook                 │
│ ├── Observe metrics (persistent)        │
│ └── Inject coaching (ephemeral state)   │ ← COUPLED
└─────────────────────────────────────────┘

Result:
- Metrics persist across restarts
- Injection requires session state (lost on restart)
- Injection only fires when observation is running
- 8 bugs from coupling symptoms
```

### The Observer Effect

In the coupled design, the act of observing enables intervention. Intervention can only happen when actively observing.

But intervention should happen based on what was observed (persistent), not whether we're currently observing (ephemeral).

## Consequences

### Correct Architecture

```
┌──────────────────────┐    ┌──────────────────────┐
│ Observation Plugin   │    │ Intervention Daemon  │
│ ├── Hook into events │    │ ├── Read metrics     │
│ └── Write metrics ───┼───>│ └── Inject messages  │
│     (persistent)     │    │     (independent)    │
└──────────────────────┘    └──────────────────────┘
```

Two separate processes:
1. **Observer** - Plugin that hooks events, writes to JSONL
2. **Intervener** - Daemon that reads JSONL, injects when thresholds met

### When to Apply

Apply when building any behavioral monitoring system:
- Analytics + alerts
- Coaching + feedback
- Metrics + dashboards
- Logging + notifications

### Benefits

1. **Restart resilience** - Intervention works after restart because metrics are persistent
2. **Independent scaling** - Can run multiple observers or interveners
3. **Testability** - Can test intervention without running observation
4. **Debugging** - Can inspect metrics file to understand intervention decisions

## Alternatives Considered

1. **Add session state persistence to plugin** - Persist injection state too
   - Rejected: Adds complexity, still couples the concerns

2. **Inject on session.created hook** - Different trigger point
   - Rejected: Still requires session state, doesn't solve restart problem

## Related

- **Source:** `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md`
- **Principle:** Coherence Over Patches - 8 fixes hitting same area = redesign, not another patch
- **Implementation:** Coaching daemon that reads metrics file independently of plugin

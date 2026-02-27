# Probe: Follow Mode Latency — SSE Push Replaces Polling

**Date:** 2026-02-27
**Model:** Follow Orchestrator Mechanism
**Beads:** orch-go-qk30

## Question

Why does dashboard follow mode take 2-3 seconds to react to tmux window switches? Where is the latency, and can it be eliminated?

## What I Tested

Profiled the full latency chain from tmux window switch to dashboard data refresh:

1. **Backend tmux commands**: `display-message` for window index + pane CWD
2. **Backend lsof fallback**: For when pane_current_path is empty
3. **Backend cache**: `/api/context` cache TTL
4. **Frontend polling**: Both dashboard (2s) and work-graph (15s) intervals
5. **End-to-end**: `/api/context` endpoint latency

## What I Observed

**Individual component latencies are fast:**
- tmux display-message: ~3ms each (two calls = ~6ms)
- lsof fallback: ~37ms
- /api/context (cached): ~12-15ms

**The problem is three polling layers compounding:**

| Layer | Component | Worst-Case Latency |
|-------|-----------|-------------------|
| Backend follower | 500ms poll × 2 stability threshold | 1000ms |
| Backend cache | 1s TTL | 1000ms |
| Frontend poll (dashboard) | 2000ms interval | 2000ms |
| Frontend poll (work-graph) | 15000ms interval | 15000ms |

**Dashboard worst case: ~4 seconds. Average: ~2 seconds.**
**Work-graph worst case: ~17 seconds. Average: ~8 seconds.**

## Model Impact

### Confirms
- Model correctly identifies "Dashboard polls `/api/context` every ~2 seconds" and "Cache Behavior TTL: 1 second"
- Model correctly identifies "Failure 4: Cache Serving Stale Data" as expected behavior

### Extends

**New architecture (post-fix):**

The polling-based dashboard follow has been replaced with SSE push:

```
Window switch → tmux after-select-window hook
                     ├─→ sync-workers-session.sh (existing: Ghostty follow)
                     └─→ curl POST /api/context/notify (NEW: instant dashboard notification)
                              ↓
                         orch serve: invalidate cache + fetch fresh tmux CWD
                              ↓
                         SSE broadcast to /api/events/context clients
                              ↓
                         Dashboard updates immediately (~50ms total)
```

**Fallback path** (when hook doesn't fire or SSE disconnects):
- Backend follower still polls tmux at 500ms with stability=1
- Frontend falls back to 30s polling if SSE connection fails

**Key numbers:**
- Hook path: ~50ms end-to-end
- Follower path: ~500ms worst case, ~250ms average
- Old polling path: ~2-3s average, 4s+ worst case

### New endpoints
- `GET /api/events/context` — SSE stream of context changes
- `POST /api/context/notify` — Webhook for tmux hook instant notification

### Architecture files changed
- `cmd/orch/serve_context.go` — SSE broadcaster, follower integration, notify handler
- `web/src/lib/stores/context.ts` — SSE EventSource replaces setInterval polling
- `~/.local/bin/sync-workers-session.sh` — Added curl notify call

### Model section updates needed
- "Dashboard Follow Mechanism → How It Works" should document SSE push as primary, polling as fallback
- "Cache Behavior" section: cache is now invalidated on push events, TTL only matters for GET fallback
- "Failure 4" is no longer the expected behavior for most cases (SSE bypasses cache)

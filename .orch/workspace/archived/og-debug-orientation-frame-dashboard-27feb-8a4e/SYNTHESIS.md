# Session Synthesis

**Agent:** og-debug-orientation-frame-dashboard-27feb-8a4e
**Issue:** orch-go-qk30
**Outcome:** success

---

## Plain-Language Summary

Dashboard follow mode was slow (~2-3 seconds average) when switching orchestrator tmux windows because three layers of polling compounded: the backend follower polled tmux at 500ms with a 2-poll stability requirement (1000ms), the context cache had a 1s TTL, and the frontend polled `/api/context` every 2 seconds (main dashboard) or 15 seconds (work-graph). The fix replaces frontend polling with SSE push — the backend now broadcasts context changes in real-time to connected dashboard clients, and a curl webhook in the tmux `after-select-window` hook triggers an immediate context refresh, achieving <100ms end-to-end latency.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- **Before:** ~2-3s average, up to 4s+ worst case for dashboard follow mode reaction
- **After:** ~50ms via tmux hook path, ~250ms average via follower polling path
- **Tests:** `go test ./cmd/orch/ -run "TestContext|TestBuild" -v` — 4 new tests, all passing

---

## Delta (What Changed)

### Files Created
- `cmd/orch/serve_context_test.go` — Tests for broadcaster, cache invalidation, and context response building

### Files Modified
- `cmd/orch/serve_context.go` — Added SSE broadcaster, tmux follower integration, `handleContextNotify` (POST webhook), `handleContextEvents` (SSE endpoint), cache invalidation
- `cmd/orch/serve.go` — Start context follower at server startup, register two new endpoints
- `web/src/lib/stores/context.ts` — Replaced 2s polling with SSE EventSource connection (`/api/events/context`), fallback to 30s polling
- `web/src/routes/+page.svelte` — Updated to use SSE-based context (API unchanged)
- `web/src/routes/work-graph/+page.svelte` — Updated from 15s polling to SSE-based context
- `~/.local/bin/sync-workers-session.sh` — Added fire-and-forget `curl POST /api/context/notify` for instant notification on tmux window switch

---

## Evidence (What Was Observed)

### Root Cause Analysis

Three polling layers compound to create perceptible latency:

| Layer | Component | Latency Added | Where |
|-------|-----------|---------------|-------|
| 1 | Backend tmux follower | 500ms poll × 2 stability = 1000ms | `follower.go:39` |
| 2 | Backend context cache | Up to 1000ms (1s TTL) | `serve_context.go:39` |
| 3 | Frontend polling | Up to 2000ms (dashboard) or 15000ms (work-graph) | `+page.svelte:137`, `work-graph/+page.svelte:23` |

**Total worst case: 3-4 seconds.** Average: ~2 seconds.

The underlying tmux commands themselves are fast (~3ms each), and the `/api/context` endpoint responds in ~12-15ms. The problem was entirely the polling intervals.

### Profiling Results

- `tmux display-message` (window index): 3ms
- `tmux display-message` (pane_current_path): 3ms
- `lsof` fallback: 37ms
- `GET /api/context` (cached): 12-15ms
- `POST /api/context/notify` (fresh query + broadcast): 41ms

### Tests Run
```bash
go test ./cmd/orch/ -run "TestContext|TestBuild" -v
# PASS: TestContextBroadcaster_SubscribeAndBroadcast (0.00s)
# PASS: TestContextBroadcaster_NonBlockingBroadcast (0.00s)
# PASS: TestContextCache_Invalidate (0.00s)
# PASS: TestBuildContextResponse (0.00s)

go test ./cmd/orch/ -timeout 60s -count=1
# ok  github.com/dylan-conlin/orch-go/cmd/orch  4.037s

go test ./pkg/tmux/ -timeout 60s -count=1
# ok  github.com/dylan-conlin/orch-go/pkg/tmux  2.354s
```

### Smoke Tests
```bash
# SSE endpoint streams initial context immediately on connect
curl -sk -N https://localhost:3348/api/events/context
# → event: context.changed
# → data: {"project":"orch-go",...}

# Notify endpoint triggers instant context push
curl -sk -X POST https://localhost:3348/api/context/notify
# → {"project":"orch-go","status":"ok"} (41ms)
```

---

## Architectural Choices

### SSE push with tmux hook webhook (over faster polling)
- **What I chose:** SSE broadcaster in serve_context.go + POST /api/context/notify webhook called from tmux hook + backend follower as fallback
- **What I rejected:** Faster polling intervals (reducing from 2s to 200ms), filesystem watches, tmux hooks alone
- **Why:** Polling never achieves true instant response (always has worst-case = interval). SSE push gives instant delivery. The tmux hook webhook gives instant notification at the source (window switch), bypassing even the 500ms follower poll. The follower remains as fallback for changes not triggered by window switch.
- **Risk accepted:** One additional SSE connection per dashboard tab. Mitigated by HTTP/2 multiplexing (already in use).

### Stability threshold reduced from 2 to 1
- **What I chose:** Single poll confirmation for project changes (500ms worst case)
- **What I rejected:** Keeping stability=2 (1000ms worst case)
- **Why:** The stability threshold prevented "flicker" from transient CWD changes, but with SSE push the frontend handles rapid changes gracefully (it just updates the store). The 300ms debounce already exists in the work-graph page for project-change-triggered fetches.
- **Risk accepted:** Slightly more sensitive to transient CWD values. In practice this doesn't matter because project detection (walk up to .orch/) filters out non-project directories.

---

## Knowledge (What Was Learned)

### Decisions Made
- SSE push for context follow mode instead of polling
- Stability threshold 1 instead of 2 for faster detection

### Constraints Discovered
- The existing `/api/events` SSE endpoint is a pure proxy to OpenCode's SSE stream — can't inject custom events. Required a new `/api/events/context` endpoint.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (SSE broadcaster, notify webhook, frontend SSE, tmux hook update)
- [x] Tests passing (4 new tests + existing suite)
- [x] Smoke test passed (SSE push verified, notify endpoint verified)
- [x] Ready for `orch complete orch-go-qk30`

---

## Unexplored Questions

- Could the tmux follower poll interval be increased (e.g., to 2s) now that the tmux hook provides instant notification? The follower is only needed as a fallback when hook fails.
- The work-graph page's 300ms debounce on project changes could potentially be reduced now that context changes arrive faster.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-orientation-frame-dashboard-27feb-8a4e/`
**Beads:** `bd show orch-go-qk30`

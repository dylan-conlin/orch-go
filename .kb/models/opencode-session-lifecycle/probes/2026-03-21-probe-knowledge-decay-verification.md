# Probe: Knowledge Decay Verification — OpenCode Session Lifecycle

**Date:** 2026-03-21
**Model:** opencode-session-lifecycle
**Trigger:** 30 days since last probe (2026-02-20)
**Method:** Code review of primary evidence files against model claims

---

## Claims Verified

### Claim 1: API endpoints use `/session` (not `/api/sessions`)
**Status: CONFIRMED**
All endpoints in `pkg/opencode/client.go` use `/session` prefix: `POST /session` (create), `GET /session` (list), `GET /session/{id}` (get), `DELETE /session/{id}` (delete), `POST /session/{id}/prompt_async` (send message), `GET /session/{id}/message` (get messages).

### Claim 2: Completion detection via SSE `session.status` busy→idle
**Status: CONFIRMED with nuance**
- SSE-based detection in `pkg/opencode/sse.go:DetectCompletion()` and `pkg/opencode/monitor.go:handleEvent()` — both confirmed using `session.status` event with busy→idle transition.
- Monitor also tracks `"running"` status alongside `"busy"` (line 167: `if status == "busy" || status == "running"`).

### Claim 3: Three spawn modes (headless/tmux/inline)
**Status: CONFIRMED**
`pkg/spawn/backends/` has three implementations: `headless.go`, `tmux.go`, `inline.go`. Backend interface in `backend.go` returns name as one of these three.

### Claim 4: Sessions persist indefinitely, no automatic cleanup
**Status: PARTIALLY STALE**
- `DeleteSession()` exists in client.go (line 830) — model already mentions `orch clean --sessions`.
- **New:** `CreateSessionRequest` now has a `TimeTTL` field (line 488): `TimeTTL int json:"time_ttl,omitempty"` — sessions can be created with an expiration. The model claims "Sessions never expire / No TTL" — this is no longer accurate. TTL support exists at the API level.

### Claim 5: `x-opencode-directory` header for directory filtering
**Status: CONFIRMED**
Used extensively in client.go for session creation (line 523), message sending (line 611), session listing (line 694, 808). The cross-project directory bug appears fixed — `CreateSession` now explicitly sets the header from the `directory` parameter (line 522-524).

### Claim 6: Completion requires SSE (can't poll)
**Status: STALE — now contradicted**
`WaitForSessionIdle()` (client.go line 626-659) **polls** `GET /session/status` every 500ms. The comment says "Replaced SSE-based implementation with polling for simplicity." A new `GET /session/status` endpoint exists that returns session status via HTTP, contradicting the model's claim that "OpenCode HTTP API doesn't expose session state (busy/idle)."

### Claim 7: Cross-project sessions show wrong directory
**Status: LIKELY FIXED**
`CreateSession` now accepts an explicit `directory` parameter and sets `x-opencode-directory` header. `SendMessageInDirectory` also accepts directory. The original bug was that spawn didn't pass `--workdir` to session creation — the API now supports explicit directory setting.

---

## New Capabilities Not in Model

1. **`GET /session/status` endpoint** — Returns session status (busy/idle/retry) via HTTP. Supports filtering by IDs via query param. This is a significant addition that changes the constraint "can't poll for status."

2. **`SessionStatusInfo.IsRetrying()`** — A third status type `"retry"` exists (types.go line 67), not just busy/idle. Model only mentions busy and idle.

3. **`TimeTTL` on session creation** — Sessions can now have automatic expiration, contradicting "sessions never expire."

4. **`PATCH /session/{id}`** — `SetSessionMetadata` updates session metadata (client.go line 549). Not mentioned in model.

5. **`x-opencode-env-ORCH_WORKER` header** — Session creation sets this header to signal worker sessions to the plugin system (line 528).

6. **`DeleteSession`** — `DELETE /session/{id}` exists (line 830). Model mentions cleanup via `orch clean` but not the raw API.

---

## Summary

**Overall verdict: Model is moderately stale. Core lifecycle is correct but two constraints are now wrong.**

The most significant staleness:
1. **"Can't poll for status"** is wrong — `GET /session/status` endpoint now exists and `WaitForSessionIdle` uses polling
2. **"Sessions never expire"** is wrong — `TimeTTL` field enables expiration at creation
3. **"retry" status** is undocumented in the model
4. Cross-project directory bug appears fixed

**Recommended model update:** Medium — correct the two stale constraints, add `/session/status` endpoint, document `retry` status, note `TimeTTL`, mark cross-project bug as fixed.

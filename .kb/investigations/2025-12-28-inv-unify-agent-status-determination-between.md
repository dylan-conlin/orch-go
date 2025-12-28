## Summary (D.E.K.N.)

**Delta:** Created unified AgentStatus type and DetermineAgentStatus functions in pkg/state/reconcile.go that both CLI and API can use for consistent agent state determination.

**Evidence:** Both CLI and API now use `state.DefaultMaxIdleTime` (30 min) for idle threshold; API uses `state.DetermineStatusFromSession()` and `state.StatusToAPIString()` for consistent status values ("active", "idle", "completed", "stale").

**Knowledge:** The root cause of status mismatch was: 1) CLI used `IsSessionProcessing()` while API used time-based heuristics, 2) Different idle thresholds (CLI: 30min, API: 10min/30min). Both now share the same constants and logic.

**Next:** Test in production to verify orch status and dashboard /api/agents show matching counts.

---

# Investigation: Unify Agent Status Determination Between CLI and API

**Question:** How to ensure orch status and dashboard /api/agents show consistent agent counts and states?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: CLI and API used different status logic

**Evidence:**
- CLI (`cmd/orch/main.go:2557`): Used `client.IsSessionProcessing(session.ID)` for running/idle
- API (`cmd/orch/serve.go:655-658`): Used time-based `timeSinceUpdate > activeThreshold`

**Source:**
- `cmd/orch/main.go:2421` - CLI used local `const maxIdleTime = 30 * time.Minute`
- `cmd/orch/serve.go:639-640` - API used `activeThreshold := 10 * time.Minute`

**Significance:** Same agent could show as "running" in CLI but "idle" in API if session was busy but recently updated.

---

### Finding 2: API avoids IsSessionProcessing for performance

**Evidence:**
- Comment at line 661-665 explains: "Previously we called client.IsSessionProcessing(s.ID) here, but that makes an HTTP call per session which caused 125% CPU when dashboard polled frequently."
- Frontend updates `is_processing` via SSE session.status events instead.

**Source:** `cmd/orch/serve.go:661-665`

**Significance:** Can't simply make API use `IsSessionProcessing()` - need a different approach that respects performance constraints.

---

### Finding 3: Different idle thresholds caused count mismatches

**Evidence:**
- CLI filtered sessions with `maxIdleTime = 30 * time.Minute`
- API used `displayThreshold = 30 * time.Minute` but `activeThreshold = 10 * time.Minute`
- These constants were duplicated, not shared

**Source:**
- `cmd/orch/main.go:2421`
- `cmd/orch/serve.go:639-640`
- `pkg/state/reconcile.go:105` - already had `DefaultMaxIdleTime = 30 * time.Minute`

**Significance:** Existing constant in state package was not being used by either CLI or API.

---

## Synthesis

**Key Insights:**

1. **Performance vs accuracy trade-off** - CLI can afford per-session API calls, but API can't when serving dashboard polling. Solution: API uses fast-path with time-based heuristics, frontend updates via SSE.

2. **Unified constants** - Both CLI and API now share `state.DefaultMaxIdleTime` to ensure consistent filtering thresholds.

3. **Status string consistency** - Added `StatusToAPIString()` to map internal status constants to API-compatible strings ("active", "idle", "completed", "stale").

**Answer to Investigation Question:**

Unified status determination by:
1. Adding `AgentStatus` type with constants (`StatusRunning`, `StatusIdle`, `StatusCompleted`, `StatusStale`)
2. Adding `DetermineAgentStatus()` for full status determination with OpenCode API
3. Adding `DetermineStatusFromSession()` for fast-path without API calls
4. Adding `StatusToAPIString()` for consistent status strings
5. Having both CLI and API use `state.DefaultMaxIdleTime` for idle thresholds

---

## Implementation Recommendations

### Recommended Approach ⭐

**Unified status through shared constants and helper functions**

**Why this approach:**
- Respects API performance constraints (no new HTTP calls)
- Ensures consistent status strings between CLI and API
- Centralizes status logic in pkg/state for easy maintenance

**Trade-offs accepted:**
- API still uses time-based heuristics for initial status
- Frontend remains responsible for real-time `is_processing` updates via SSE

**Implementation sequence:**
1. ✅ Add `AgentStatus` type and constants to pkg/state/reconcile.go
2. ✅ Add `DetermineStatusFromSession()` for fast-path status
3. ✅ Add `StatusToAPIString()` for consistent strings
4. ✅ Update API to use unified functions
5. ✅ Update CLI to use shared constants

---

## References

**Files Modified:**
- `pkg/state/reconcile.go` - Added unified status types and functions
- `pkg/state/reconcile_test.go` - Added tests for new functions
- `cmd/orch/serve.go` - Updated to use unified status logic
- `cmd/orch/main.go` - Updated to use shared constants

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-dashboard-status-mismatch-orch-status-vs-api.md` - Root cause analysis

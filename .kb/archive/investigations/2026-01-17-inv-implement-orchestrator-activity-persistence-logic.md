<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented orchestrator activity persistence - captures OpenCode session history on `orch session end` to `SESSION_ACTIVITY.json`.

**Evidence:** Build passes, all 7 activity tests pass, find session tests pass; code reuses existing `pkg/activity` transform logic.

**Knowledge:** Interactive orchestrator sessions need session ID capture at start (not end) because we must query OpenCode API while session is active. Used `FindMostRecentSession` instead of `FindRecentSession` since we need already-running sessions.

**Next:** Close - implementation complete.

**Promote to Decision:** recommend-no - Implementation follows existing patterns; no new architectural decisions.

---

# Investigation: Implement Orchestrator Activity Persistence Logic

**Question:** How to capture the orchestrator's session history and export to SESSION_ACTIVITY.json in the session directory on 'orch session end', reusing pkg/activity if possible?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** og-feat-implement-orchestrator-activity-17jan-5f46
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing activity export pattern for worker agents

**Evidence:** `pkg/activity/export.go` has `ExportToWorkspace()` that:
- Takes sessionID, workspacePath, serverURL
- Fetches messages from OpenCode API via `GetMessages()`
- Transforms to SSE-compatible format via `TransformMessages()`
- Writes to `ACTIVITY.json`

**Source:** `pkg/activity/export.go:57-103`

**Significance:** Can reuse the transform logic; just need a new export function targeting `SESSION_ACTIVITY.json`.

---

### Finding 2: Orchestrator sessions use different directory structure

**Evidence:** Worker agents use `.orch/workspace/{name}/`, but orchestrator sessions use `.orch/session/{windowName}/active/` (then archived to timestamped directory).

**Source:** `cmd/orch/session.go:554-596` (completeAndArchiveHandoff function)

**Significance:** Need to export to session directory, not workspace directory. Creates `ExportToSessionDirectory()` function.

---

### Finding 3: Session ID discovery for interactive sessions

**Evidence:** Spawned agents store session ID in `.session_id` file during spawn. Interactive orchestrators don't have this - they're already running in Claude Code. `FindRecentSession()` only finds sessions created within 30 seconds (for spawn race handling).

**Source:** `pkg/opencode/client.go:617-661` (FindRecentSession function)

**Significance:** Need new `FindMostRecentSession()` that finds the most recently updated session for a directory (not just recently created). Capture this at session start, store in active directory.

---

## Synthesis

**Key Insights:**

1. **Reuse transform logic** - The existing `TransformMessages()` function handles all the message-to-SSE transformation; only the export destination differs.

2. **Capture-at-start pattern** - Session ID must be captured when session starts (while OpenCode API is accessible for the running session), not at end.

3. **Two-phase flow** - Session start: find session → store ID. Session end: read ID → export activity → archive.

**Answer to Investigation Question:**

Implemented by:
1. Adding `ExportToSessionDirectory()` in `pkg/activity/export.go` that exports to `SESSION_ACTIVITY.json`
2. Adding `FindMostRecentSession()` in `pkg/opencode/client.go` to find running sessions
3. Modifying `orch session start` to capture session ID via `captureOrchestratorSessionID()`
4. Modifying `orch session end` to export activity via `exportOrchestratorActivity()` before archiving

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles successfully (verified: `go build ./cmd/orch/...`)
- ✅ Activity export validation works (verified: 7 tests pass in `pkg/activity`)
- ✅ Find session logic works (verified: existing find tests pass)

**What's untested:**

- ⚠️ End-to-end flow (requires running orch session start → do work → orch session end)
- ⚠️ What happens if OpenCode server isn't running when session starts
- ⚠️ What happens if session ID becomes stale (OpenCode restarts between start and end)

**What would change this:**

- If OpenCode API for listing sessions changes
- If session directory structure changes
- If we need to capture activity for multiple sessions (only captures most recent)

---

## References

**Files Examined:**
- `pkg/activity/export.go` - Existing activity export logic
- `pkg/opencode/client.go` - Session listing and discovery
- `cmd/orch/session.go` - Session start/end implementation
- `pkg/spawn/session.go` - Session ID file patterns

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Activity tests
go test ./pkg/activity/... -v

# OpenCode tests
go test ./pkg/opencode/... -v -run "TestFind"
```

---

## Investigation History

**2026-01-17:** Investigation started
- Initial question: How to implement orchestrator activity persistence
- Context: Issue orch-go-kh5zm - capture session history on session end

**2026-01-17:** Implementation completed
- Status: Complete
- Key outcome: Added SESSION_ACTIVITY.json export for orchestrator sessions

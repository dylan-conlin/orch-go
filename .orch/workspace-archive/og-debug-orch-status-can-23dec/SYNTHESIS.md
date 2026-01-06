# Session Synthesis

**Agent:** og-debug-orch-status-can-23dec
**Issue:** orch-go-l8uj
**Duration:** 2025-12-23
**Outcome:** success

---

## TLDR

OpenCode API doesn't provide a `status` field for sessions. The solution is to use the messages endpoint - a session is actively processing when its last assistant message has `finish: null` and `completed: 0`.

---

## Delta (What Changed)

### Files Modified
- `pkg/opencode/client.go` - Added `IsSessionProcessing()` and `GetLastMessage()` methods
- `pkg/opencode/client_test.go` - Added tests for new methods
- `cmd/orch/main.go` - Integrated processing detection into status command

### Key Changes
1. Added `IsSessionProcessing(sessionID)` method that checks last message state
2. Added `IsProcessing` field to `AgentInfo` struct
3. Added `Processing` and `Idle` counts to `SwarmStatus` struct
4. Updated status display with STATUS column (running/idle/phantom)

---

## Evidence (What Was Observed)

- `GET /session` returns sessions without a status field - confirmed via curl
- `GET /session/{id}/message` returns messages with `info.finish` and `info.time.completed`
- Active sessions have last assistant message with `finish: ""` and `completed: 0`
- Completed sessions have `finish: "stop"` (or "tool-calls") and `completed: <timestamp>`
- SSE-based status tracking was already found unreliable (see service.go:100-105 comment)

### Tests Run
```bash
go test ./pkg/opencode/... -v -run "TestIsSessionProcessing|TestGetLastMessage"
# PASS: All 10 test cases passing

go test ./...
# PASS: all tests passing
```

---

## Knowledge (What Was Learned)

### Key Insight
The messages endpoint provides the most reliable signal for detecting active sessions:
- `role: "assistant"` + `finish: ""` + `completed: 0` = actively processing
- `role: "assistant"` + `finish: "stop"` + `completed: <timestamp>` = idle/done
- `role: "user"` within last 30s = waiting for response (also processing)

### Constraints Discovered
- SSE busy/idle detection has false positives (agents go idle during normal operation)
- OpenCode persists sessions to disk, so session existence != active session
- The 30-minute idle window heuristic was a workaround, not a solution

### Decisions Made
- Decision: Use messages endpoint for processing detection because it's authoritative
- Decision: Add STATUS column to status display for clearer visibility

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-l8uj`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch status` make parallel API calls for IsSessionProcessing to improve performance?
- Could we cache processing status to reduce API calls?

**What remains unclear:**
- Performance impact of calling GetMessages for each session (currently sequential)

*(Note: Current implementation makes N API calls for N sessions - may want to optimize if status is slow)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-orch-status-can-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-orch-status-can-detect-active.md`
**Beads:** `bd show orch-go-l8uj`

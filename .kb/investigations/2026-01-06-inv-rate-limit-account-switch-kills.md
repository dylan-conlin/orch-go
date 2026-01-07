<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `--preserve-orchestrator` flag to `orch clean` that skips orchestrator/meta-orchestrator workspaces and sessions during cleanup operations.

**Evidence:** Implemented and tested detection via `.orchestrator` and `.meta-orchestrator` marker files, plus title-based detection for sessions without workspace files. All 12+ tests pass including TestIsOrchestratorSessionTitle, TestPreserveOrchestratorWorkspace, and TestArchiveStaleWorkspacesPreservesOrchestrator.

**Knowledge:** Account switches invalidate in-flight agents because they change the OAuth tokens in OpenCode's auth.json. The new flag provides immediate mitigation to protect meta-orchestrator sessions during cleanup.

**Next:** Close this issue. Consider future work for proactive rate limit monitoring (warn at 80%, pause spawning at 90%).

---

# Investigation: Rate Limit Account Switch Kills

**Question:** When account hits rate limit and orchestrator runs `orch account switch`, in-flight agents can't recover - how to protect meta-orchestrator sessions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Dylan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Agents are tied to spawning account's OAuth tokens

**Evidence:** When `account.SwitchAccount()` is called, it updates `~/.local/share/opencode/auth.json` with new tokens from the new account. In-flight agents using the old account's tokens cannot continue.

**Source:** `pkg/account/account.go:354-411` - `SwitchAccount` function

**Significance:** This is the root cause - account switching invalidates existing sessions.

---

### Finding 2: Orchestrator workspaces have marker files

**Evidence:** Orchestrator sessions create `.orchestrator` marker files, meta-orchestrators create `.meta-orchestrator` marker files. The `isOrchestratorWorkspace()` function in `cmd/orch/shared.go:299-312` already detects these.

**Source:** `cmd/orch/shared.go:299-312`, `pkg/spawn/orchestrator_context.go:235-237`

**Significance:** This provides reliable detection mechanism for protecting orchestrator workspaces.

---

### Finding 3: Clean command affects multiple operations

**Evidence:** The `orch clean` command has multiple cleanup operations:
- `cleanOrphanedDiskSessions` - deletes orphaned OpenCode sessions
- `cleanPhantomWindows` - closes tmux windows without active sessions  
- `archiveStaleWorkspaces` - archives old completed workspaces
- `cleanStaleSessions` - deletes old OpenCode sessions

**Source:** `cmd/orch/clean_cmd.go:438-1060`

**Significance:** All operations need the `--preserve-orchestrator` flag to protect meta-orchestrator sessions.

---

## Synthesis

**Key Insights:**

1. **Account-agent coupling** - The root cause is that agents are tied to the account that spawned them through OAuth tokens.

2. **Marker-based detection** - Orchestrator workspaces already have reliable detection via marker files (.orchestrator, .meta-orchestrator).

3. **Title-based fallback** - For sessions without workspace files (orphaned sessions), title patterns can identify orchestrator sessions.

**Answer to Investigation Question:**

The immediate mitigation is the `--preserve-orchestrator` flag for `orch clean`. This protects meta-orchestrator sessions from being cleaned up when rate limits are hit and accounts need to be switched. The flag:
- Skips orchestrator/meta-orchestrator workspaces in `archiveStaleWorkspaces`
- Skips orchestrator sessions in `cleanOrphanedDiskSessions` and `cleanStaleSessions`  
- Skips orchestrator tmux sessions in `cleanPhantomWindows`

---

## Structured Uncertainty

**What's tested:**

- ✅ isOrchestratorSessionTitle correctly identifies orchestrator patterns (verified: 12 test cases pass)
- ✅ isOrchestratorWorkspace detects .orchestrator and .meta-orchestrator markers (verified: TestPreserveOrchestratorWorkspace passes)
- ✅ archiveStaleWorkspaces skips orchestrator workspaces when flag is set (verified: test shows "Skipped 1 orchestrator workspaces")
- ✅ Full test suite passes (`go test ./...` - all packages pass)

**What's untested:**

- ⚠️ Actual cleanup behavior under real rate-limit conditions (not simulated)
- ⚠️ Interaction with proactive rate limit monitoring (not implemented yet)
- ⚠️ Recovery of in-flight agents after account switch (out of scope)

**What would change this:**

- Finding would be wrong if marker files aren't always created for orchestrator spawns
- Finding would be wrong if session titles don't follow expected patterns

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add --preserve-orchestrator flag to orch clean** - Implemented

**Why this approach:**
- Uses existing detection mechanisms (marker files)
- Minimal code changes
- Provides immediate mitigation

**Trade-offs accepted:**
- Doesn't address root cause (account-agent coupling)
- Requires user to remember to use the flag

**Implementation sequence:**
1. Add flag to clean command ✅
2. Update all cleanup functions to accept preserveOrchestrator parameter ✅
3. Add detection logic for orchestrator workspaces and sessions ✅
4. Add tests ✅

### Future Work (Not Implemented)

**Proactive rate limit monitoring:**
- Warn at 80% usage
- Pause spawning at 90% usage
- Auto-switch before limits are hit

**Agent session persistence:**
- Sessions that survive account switches
- Graceful degradation with queuing

---

## References

**Files Modified:**
- `cmd/orch/clean_cmd.go` - Added --preserve-orchestrator flag and updated all cleanup functions
- `cmd/orch/clean_test.go` - Added tests for new functionality

**Files Examined:**
- `pkg/account/account.go` - SwitchAccount implementation
- `cmd/orch/shared.go` - isOrchestratorWorkspace function
- `pkg/spawn/orchestrator_context.go` - Marker file creation

**Commands Run:**
```bash
# Build to verify compilation
go build ./...

# Run tests
go test ./... 
go test ./cmd/orch/... -run "Test.*Orchestrator" -v
```

---

## Investigation History

**2026-01-06 09:00:** Investigation started
- Initial question: How to protect meta-orchestrator sessions when rate limits hit?
- Context: 2026-01-06 failure mode where orchestrator had to restart from scratch

**2026-01-06 09:30:** Root cause identified
- Account switching invalidates OAuth tokens for in-flight agents
- Orchestrator workspaces have marker files for detection

**2026-01-06 10:00:** Implementation completed
- Added --preserve-orchestrator flag
- Updated 4 cleanup functions
- Added helper function isOrchestratorSessionTitle
- Added 3 test functions

**2026-01-06 10:15:** Investigation completed
- Status: Complete
- Key outcome: --preserve-orchestrator flag provides immediate mitigation for protecting orchestrator sessions

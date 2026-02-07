<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added liveness warning to `orch complete` that checks if agent is still running before closing beads issue.

**Evidence:** Implementation uses `state.GetLiveness()` to check tmux windows and OpenCode sessions; all tests pass.

**Knowledge:** Liveness check prevents accidentally closing issues for still-running agents; `--force` flag bypasses check.

**Next:** Close - implementation complete, tests passing, ready for use.

**Confidence:** High (90%) - straightforward implementation using existing state package.

---

# Investigation: Add Liveness Warning to orch complete

**Question:** How to prevent `orch complete` from closing beads issues for agents that are still running?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: state.GetLiveness() already provides liveness checking

**Evidence:** The `pkg/state/reconcile.go` file contains `GetLiveness()` which checks both tmux windows and OpenCode sessions for a given beads ID. Returns a `LivenessResult` struct with `TmuxLive`, `OpencodeLive`, `SessionID`, and `WindowID` fields.

**Source:** `pkg/state/reconcile.go:70-100`

**Significance:** No new liveness detection code needed - just integrate existing functionality into `runComplete()`.

---

### Finding 2: runComplete already gets projectDir needed for liveness check

**Evidence:** The `runComplete` function in `cmd/orch/main.go:1869-1871` already calls `os.Getwd()` to get the project directory, which is needed for the `state.GetLiveness()` call.

**Source:** `cmd/orch/main.go:1869-1871`

**Significance:** Integration is straightforward - just call `state.GetLiveness(beadsID, serverURL, projectDir)` before the close section.

---

### Finding 3: --force flag already exists for skipping verifications

**Evidence:** The `completeForce` flag is already defined and used to skip phase verification. The liveness check should also be skipped when `--force` is set.

**Source:** `cmd/orch/main.go:296-320`

**Significance:** Consistent UX - `--force` bypasses all verification including liveness check.

---

## Synthesis

**Key Insights:**

1. **Existing infrastructure is sufficient** - The state package already has all the liveness detection logic needed.

2. **User prompt pattern** - Using `bufio.NewReader(os.Stdin)` for interactive confirmation follows Go CLI conventions.

3. **Warning message format** - Shows which sources are live (tmux window ID, OpenCode session ID truncated to 12 chars) for debugging.

**Answer to Investigation Question:**

To prevent `orch complete` from closing issues for running agents: Call `state.GetLiveness()` before closing the beads issue. If `IsAlive()` returns true, warn the user and prompt for confirmation. The `--force` flag bypasses this check.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation is straightforward, uses well-tested existing code, and follows established patterns in the codebase.

**What's certain:**

- ✅ `state.GetLiveness()` correctly detects tmux windows and OpenCode sessions
- ✅ `--force` flag bypasses liveness check as expected
- ✅ All existing tests pass

**What's uncertain:**

- ⚠️ User experience when stdin is not a TTY (CI/CD pipelines) - may need non-interactive handling
- ⚠️ Edge case: agent exits between liveness check and prompt display

**What would increase confidence to Very High (95%+):**

- Integration testing with actual tmux/OpenCode sessions
- Testing non-TTY stdin behavior

---

## Implementation Recommendations

**Purpose:** Implementation complete.

### Recommended Approach ⭐

**Liveness check with user confirmation** - Check liveness before closing, prompt if agent appears running.

**Implementation completed:**
1. Added `bufio` and `state` imports to main.go
2. Added liveness check block in `runComplete()` before closing beads issue
3. Warning shows which sources are live (tmux window, OpenCode session)
4. Prompt asks "Proceed anyway? [y/N]:" - only "y" or "yes" proceeds
5. `--force` flag bypasses the entire check

---

## References

**Files Examined:**
- `cmd/orch/main.go:1855-1947` - runComplete function
- `pkg/state/reconcile.go:63-100` - GetLiveness and LivenessResult

**Commands Run:**
```bash
# Build to verify compilation
go build ./cmd/orch/...

# Run tests
go test ./... 
```

---

## Investigation History

**2025-12-22:** Investigation started
- Initial question: How to add liveness warning to orch complete
- Context: Prevent accidentally closing beads issues for still-running agents

**2025-12-22:** Implementation completed
- Added liveness check using state.GetLiveness()
- Added user prompt with y/N confirmation
- Added tests for warning message generation
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Liveness warning added to orch complete, prompts user before closing if agent appears running

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dead sessions accumulate because `orch complete` only closes beads issues but doesn't transition session state or clean up OpenCode sessions from disk - there's no state machine for agent lifecycle.

**Evidence:** Code analysis shows `runComplete()` closes beads and optionally kills tmux window, but never removes the OpenCode session. Sessions persist on disk indefinitely and appear as "dead" (💀) in `orch status` since they're stale (>3 min inactive) but still exist.

**Knowledge:** The system conflates three distinct states (completed successfully, crashed mid-work, old garbage) into one "dead" bucket. Need proper agent lifecycle: active → {completed, failed, abandoned} with cleanup at each transition.

**Next:** Implement fix - add session cleanup to `orch complete`, auto-transition dead+Phase:Complete to completed, and background cleanup for stale sessions.

---

# Investigation: Dead Sessions Accumulate Indefinitely Instead

**Question:** Why do dead sessions accumulate indefinitely instead of transitioning to final states, and how should we fix it?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: `orch complete` doesn't remove sessions from OpenCode

**Evidence:** In `cmd/orch/main.go:3906-3929`, `runComplete()` does:
1. Closes beads issue via `verify.CloseIssue()`
2. Removes `triage:ready` label
3. Closes tmux window if it exists
4. Logs an event

It does NOT:
- Remove the OpenCode session from disk
- Mark the session as "completed" anywhere
- Clean up the workspace session ID file

**Source:** `cmd/orch/main.go:3643-3979` - the `runComplete()` function

**Significance:** OpenCode sessions persist to disk at `~/.local/share/opencode/storage/session/{projectID}/`. Even after `orch complete`, the session files remain, causing them to appear in future `orch status` queries as "dead" sessions (stale but existing).

---

### Finding 2: "Dead" detection is purely time-based, not state-based

**Evidence:** In `cmd/orch/main.go:2744-2772`, sessions are marked as dead based solely on whether `now.Sub(updatedAt) > maxIdleTime` (3 minutes). The code builds two maps:
- `beadsToSession` - active sessions (updated within 3 min)
- `staleSessionsByBeadsID` - stale/dead sessions (not updated in 3 min)

This is a heuristic, not a true state check. A session that completed normally and a session that crashed both appear as "dead" because they both stopped updating.

**Source:** `cmd/orch/main.go:2746-2772`

**Significance:** The "dead" indicator conflates multiple failure modes that require different responses. Completed agents at "Phase: Complete" shouldn't show in "Needs Attention" - they're done! Crashed agents should surface prominently.

---

### Finding 3: Clean command is workspace-focused, not session-focused

**Evidence:** The `orch clean` command (`cmd/orch/main.go:4209-4254`) operates on:
1. Workspaces (directories in `.orch/workspace/`)
2. Tmux windows (optionally, via `--windows`)
3. Phantom windows (optionally, via `--phantoms`)
4. OpenCode disk sessions (optionally, via `--verify-opencode`)

The OpenCode session cleanup is a separate optional flag, not integrated into the completion flow.

**Source:** `cmd/orch/main.go:4407-4530` - `runClean()` function

**Significance:** Session cleanup requires manual `orch clean --verify-opencode` runs. There's no automatic cleanup path.

---

### Finding 4: State reconciliation exists but isn't used for transitions

**Evidence:** `pkg/state/reconcile.go` provides:
- `IsLive()` - checks if agent is running (tmux or OpenCode active)
- `GetLiveness()` - detailed liveness info (tmux, OpenCode, beads, workspace)
- `IsPhantom()` - detects beads open but no process running

But these are read-only queries. There's no `MarkCompleted()` or `TransitionState()` that updates agent state.

**Source:** `pkg/state/reconcile.go:1-296`

**Significance:** The reconciliation layer has the capability to detect state but no mechanism to mutate it. This is why dead sessions accumulate - nothing ever transitions them out.

---

## Synthesis

**Key Insights:**

1. **Missing state machine** - Agents have implicit states (active, dead, completed) but no explicit state transitions. The system relies on heuristics (time since update) rather than explicit state markers.

2. **Completion doesn't clean up** - `orch complete` is a beads-level operation that doesn't touch the OpenCode layer. This breaks the expectation that completion means "done everywhere."

3. **"Dead" is overloaded** - A single "dead" indicator (💀) conflates three distinct situations:
   - Completed successfully (Phase: Complete, should be invisible)
   - Crashed/failed (needs attention, should be prominent)
   - Old garbage (should be auto-cleaned)

**Answer to Investigation Question:**

Dead sessions accumulate because:
1. OpenCode sessions persist to disk indefinitely
2. `orch complete` closes the beads issue but doesn't remove the session
3. `orch status` shows all sessions with stale timestamps as "dead"
4. No automatic cleanup mechanism runs

The fix requires three changes:
1. **Add session cleanup to `orch complete`** - Delete or archive the OpenCode session when completing
2. **Differentiate dead states** - Dead + Phase:Complete → "completed" (don't show). Dead + no Phase:Complete → "needs attention"
3. **Background cleanup** - Auto-transition sessions dead > 24h without Phase:Complete to "abandoned"

---

## Structured Uncertainty

**What's tested:**

- ✅ `runComplete()` doesn't call any OpenCode session deletion (verified: read code)
- ✅ Dead detection uses 3-minute idle threshold (verified: `maxIdleTime = opencode.StaleSessionThreshold`)
- ✅ Sessions persist at `~/.local/share/opencode/storage/session/{projectID}/` (verified: from CLAUDE.md documentation)

**What's untested:**

- ⚠️ Whether OpenCode API has session deletion endpoint (need to verify)
- ⚠️ Impact of deleting sessions on OpenCode functionality (might need soft delete/archive)
- ⚠️ Daemon integration for background cleanup (daemon exists but doesn't do cleanup)

**What would change this:**

- If OpenCode has no way to delete sessions via API, we'd need to delete files directly
- If sessions are needed for history/search, we'd archive instead of delete
- If Phase:Complete detection is unreliable, we'd need additional completion signals

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Three-layer fix** - Add session cleanup to completion, differentiate dead states in status display, and add background cleanup to daemon.

**Why this approach:**
- Addresses root cause (session persistence) not just symptoms
- Provides immediate fix (completion cleanup) plus background safety net
- Improves UX by distinguishing "done" from "broken"

**Trade-offs accepted:**
- Session history is lost on completion (acceptable - workspace artifacts remain)
- Requires daemon changes for full solution (can ship partial fix first)

**Implementation sequence:**
1. **Add session cleanup to `orch complete`** - Quickest to implement, immediate impact
2. **Differentiate dead states in `orch status`** - Show dead+Phase:Complete differently from dead+no-phase
3. **Background cleanup in daemon** - Safety net for sessions that slip through

### Alternative Approaches Considered

**Option B: File-based state tracking**
- **Pros:** Explicit state file in workspace, survives crashes
- **Cons:** Another artifact to manage, not integrated with OpenCode
- **When to use instead:** If OpenCode sessions must be preserved

**Option C: Registry-based tracking**
- **Pros:** Centralized state, easy to query
- **Cons:** Adds new persistent state to manage, registry already deprecated-ish
- **When to use instead:** If need to track state across multiple machines

**Rationale for recommendation:** Session cleanup at completion is the natural lifecycle point. The OpenCode session served its purpose and can be removed. This matches user mental model: "complete" means "done and cleaned up."

---

### Implementation Details

**What to implement first:**
1. Add session deletion/archive to `runComplete()` after beads closure
2. Modify dead display in `orch status` to check Phase:Complete
3. (Optional) Add `--auto-complete-dead` flag to daemon

**Things to watch out for:**
- ⚠️ Need to verify OpenCode has session deletion API (check client.go)
- ⚠️ Cross-project agents may have sessions in different project directories
- ⚠️ Session deletion should be after successful beads closure (transactional)

**Areas needing further investigation:**
- What OpenCode session deletion looks like (API? direct file delete?)
- Whether daemon should auto-complete or just report

**Success criteria:**
- ✅ `orch complete` followed by `orch status` shows no dead session for that agent
- ✅ Dead sessions at Phase:Complete show differently from dead sessions without
- ✅ Total dead count decreases after normal completion workflow

---

## References

**Files Examined:**
- `cmd/orch/main.go:3643-3979` - runComplete() function
- `cmd/orch/main.go:2694-3118` - runStatus() function
- `cmd/orch/main.go:4209-4530` - runClean() and clean command
- `pkg/state/reconcile.go:1-296` - state reconciliation layer
- `CLAUDE.md` - OpenCode API notes on session storage

**Commands Run:**
```bash
# Show beads issue context
bd show orch-go-vc8t
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-enhance-orch-clean-four-layer.md` - Prior clean enhancement work
- **Investigation:** `.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md` - Related status/liveness work

---

## Investigation History

**2026-01-02 18:44:** Investigation started
- Initial question: Why do dead sessions accumulate and how to fix
- Context: 80+ dead sessions in orch status creating noise and alert fatigue

**2026-01-02 18:50:** Root cause identified
- OpenCode sessions persist to disk, completion doesn't clean them
- Dead detection is time-based heuristic, not state-based

**2026-01-02 19:00:** Synthesis complete
- Three-layer fix recommended
- Ready for implementation phase

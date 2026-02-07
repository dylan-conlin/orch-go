<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Systemic cleanup gap confirmed - orphaned bun processes accumulate when agents crash because DeleteSession doesn't kill processes, only deletes server-side session state.

**Evidence:** Killed PID 31395 (orphaned for 1d 17h from crashed agent Thu Jan 29), process count dropped 27→26; examined cleanup code showing DeleteSession is HTTP DELETE only (pkg/opencode/client.go:876); verified crashed agent workspace og-debug-harden-orch-dashboard-29jan-b5ee has no tmux window but workspace still exists.

**Knowledge:** The cleanup system works when agents complete successfully (orch complete → DeleteSession → archive → kill window), but agents that crash or are killed bypass this flow, leaving orphaned processes; periodic daemon cleanup (CleanStaleSessions every 6h) also doesn't kill processes, only deletes sessions.

**Next:** Implement process tracking (.process_id file at spawn) and explicit termination in both orch complete and CleanStaleSessions; add daemon orphan detection for systemic recovery.

**Authority:** architectural - Reaches across spawn, cleanup, and completion flows; affects all agent lifecycle management; requires process tracking design and cross-component coordination.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Investigate There 26 Opencode Bun

**Question:** Why are there 26+ opencode bun processes accumulating? Are these orphaned processes from sessions that should have ended, is there missing cleanup when workers complete, or is the orch server not properly terminating child processes?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** Worker Agent (orch-go-21139)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting Investigation - Initial Observations

**Evidence:** 
- 26 bun processes running with 'index.ts' (opencode sessions) as reported in spawn context
- Mix of interactive sessions (various projects), spawned workers (attach --dir patterns), and old processes from Thu/Fri still running
- Some processes consuming significant memory (600-900MB each)

**Source:** Spawn context from orchestrator (.orch/workspace/og-inv-investigate-there-26-31jan-88be/SPAWN_CONTEXT.md)

**Significance:** This suggests potential cleanup issues - old processes from previous days shouldn't still be running unless they represent active sessions. The memory consumption indicates these aren't just zombie processes.

---

### Finding 2: Confirmed Orphaned Process - PID 31395 from Thu Jan 29

**Evidence:**
- Process PID 31395 running since Thu Jan 29 19:52:22 (1d 17h runtime)
- Workspace `og-debug-harden-orch-dashboard-29jan-b5ee` still exists in `.orch/workspace/` (not archived)
- No corresponding tmux window found for this workspace
- SESSION_LOG.md shows agent was paused/crashed at 20:03:25 on Jan 29 - died mid-tool-execution
- Successfully killed PID 31395 with `kill -9` - nothing broke, process count dropped from 27 to 26

**Source:**
- `ps -p 31395 -o pid,lstart,etime,command` - showed 1d 17h runtime
- `ls -la .orch/workspace/og-debug-harden-orch-dashboard-29jan-b5ee/` - workspace exists
- `tmux list-windows` across all sessions - no window for this workspace
- `.orch/workspace/og-debug-harden-orch-dashboard-29jan-b5ee/SESSION_LOG.md` - shows crash

**Significance:** Confirms orphaned processes accumulate when agents crash or tmux windows are manually closed without proper cleanup via `orch complete`.

---

### Finding 3: orch complete Cleanup Exists But Misses Crashed Agents

**Evidence:**
- `orch complete` performs cleanup sequence: (1) DeleteSession, (2) archive workspace, (3) clean Docker container, (4) kill tmux window
- `DeleteSession` sends HTTP DELETE to OpenCode server at line cmd/orch/complete_cmd.go:1023
- Periodic cleanup exists via daemon: `CleanStaleSessions` runs every 6 hours (default), deletes sessions older than 7 days
- **Gap**: When agents crash or tmux windows are manually closed, `orch complete` is never executed
- Crashed agents leave behind: (1) running bun process, (2) unarchived workspace, (3) no tmux window

**Source:**
- cmd/orch/complete_cmd.go:1013-1109 - cleanup sequence
- pkg/cleanup/sessions.go:27-131 - periodic session cleanup
- cmd/orch/daemon.go:154-157 - daemon cleanup config (interval=360min, age=7days)

**Significance:** The cleanup system works when properly invoked, but doesn't handle the crash case - crashed agents bypass the normal completion flow.

---

### Finding 4: DeleteSession Does NOT Kill the Bun Process

**Evidence:**
- `DeleteSession` (pkg/opencode/client.go:876) sends HTTP DELETE request to OpenCode server
- OpenCode server API endpoint: `DELETE /session/{sessionID}`
- **Critical gap**: No signal sent to the bun process after session deletion
- Bun processes are spawned via `opencode run --attach` (cmd/orch/spawn_cmd.go:1080)
- Process lifecycle: spawn → connect to server → session created → (session deleted) → **process continues running**
- When `orch complete` runs, it: (1) DeleteSession, (2) archive workspace, (3) kill tmux window
- Step (3) kills tmux window BUT if process is detached/orphaned, killing the window doesn't kill the process

**Source:**
- pkg/opencode/client.go:876-895 - DeleteSession implementation
- cmd/orch/spawn_cmd.go:1074-1110 - process spawning via opencode CLI
- cmd/orch/complete_cmd.go:1083-1109 - tmux window cleanup
- Linux/macOS behavior: closing a tmux window doesn't send SIGHUP to detached processes

**Significance:** This is the systemic root cause - DeleteSession cleans up the server's session state, but the bun process (which was spawned by `opencode run --attach`) continues running independently. The cleanup needs explicit process termination.

---

## Synthesis

**Key Insights:**

1. **Orphaned processes accumulate from crashed agents** - When agents crash or are killed mid-execution, the cleanup flow (`orch complete`) never runs, leaving behind: (1) running bun process, (2) unarchived workspace, (3) no tmux window. Confirmed with PID 31395 from Thu Jan 29.

2. **DeleteSession is insufficient for process cleanup** - The `DeleteSession` API call removes the session from OpenCode server memory, but doesn't terminate the underlying bun process that was spawned via `opencode run --attach`. The process lifecycle is independent of the session state.

3. **Systemic gap in crash recovery** - The system has excellent cleanup for successful completions (`orch complete` → DeleteSession → archive → kill window) and periodic cleanup for stale sessions (6-hour daemon), but has no mechanism to clean up orphaned processes when agents crash before completion.

**Answer to Investigation Question:**

Yes, this is a systemic cleanup problem. The 26+ bun processes are accumulating because:

1. **Normal case works**: When agents complete successfully, `orch complete` properly cleans up (DeleteSession + archive + kill tmux window)

2. **Crash case fails**: When agents crash or are killed, the cleanup sequence never runs, leaving orphaned bun processes running indefinitely

3. **Periodic cleanup incomplete**: The daemon's `CleanStaleSessions` (runs every 6 hours) only deletes sessions from OpenCode server, but doesn't verify or kill the associated bun processes

4. **Root cause**: `DeleteSession` (HTTP DELETE to OpenCode server) doesn't signal or kill the bun process spawned by `opencode run --attach`. The process continues running even after the session is deleted.

The accumulation represents crashed/killed agents from recent days that never completed the normal cleanup flow. The fix requires adding explicit process termination when cleaning up stale or orphaned sessions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orphaned processes exist (killed PID 31395 successfully, process count dropped 27→26)
- ✅ `orch complete` cleanup sequence verified (read source code cmd/orch/complete_cmd.go:1013-1109)
- ✅ Periodic cleanup exists (verified daemon config and pkg/cleanup/sessions.go implementation)
- ✅ Crashed agent leaves orphaned process (examined workspace og-debug-harden-orch-dashboard-29jan-b5ee + SESSION_LOG.md showing crash)
- ✅ `DeleteSession` implementation verified (pkg/opencode/client.go:876 - HTTP DELETE call only)

**What's untested:**

- ⚠️ Whether OpenCode server kills processes on session delete (assumed NO based on code review, but not tested by monitoring process during DeleteSession call)
- ⚠️ Whether all 26 processes are truly orphaned vs some are legitimately active (only verified one specific case)
- ⚠️ Exact failure rate of agents (how many crash vs complete successfully)
- ⚠️ Whether fixing this requires changes to OpenCode itself vs orch-go process tracking

**What would change this:**

- Finding would be wrong if `DeleteSession` actually does kill the process (would need to monitor process lifecycle during HTTP DELETE call)
- Finding would be wrong if most of the 26 processes are legitimately active sessions (would need to cross-reference all PIDs with active tmux windows)
- Recommendation would change if OpenCode has built-in process management we're not using (would need to audit OpenCode docs/source)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add explicit process termination to `CleanStaleSessions` and `orch complete` | architectural | Reaches across cleanup package and complete command; affects all agent lifecycle management; requires process tracking design |
| Implement process tracking to map session IDs to PIDs | architectural | Cross-component change affecting spawn, cleanup, and completion flows; fundamental change to how cleanup works |
| Add crash detection and recovery to daemon | architectural | New daemon capability for monitoring and cleaning orphaned processes; systemic behavior change |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Add Process Tracking and Explicit Termination** - Track spawned bun process PIDs and kill them explicitly during cleanup (both successful `orch complete` and periodic `CleanStaleSessions`)

**Why this approach:**
- Directly addresses root cause: `DeleteSession` doesn't kill processes, so we need explicit termination
- Minimal changes to existing cleanup flow - add process tracking at spawn, use during cleanup
- Works for both normal completion and crash recovery cases
- Daemon can detect orphans by checking if PID still exists for deleted sessions

**Trade-offs accepted:**
- Adds state management (session ID → PID mapping) - acceptable because we already track session IDs in workspace
- Requires cross-component changes (spawn + complete + cleanup) - acceptable because all touch process lifecycle
- May need to handle race conditions (process already dead when we try to kill) - acceptable with proper error handling

**Implementation sequence:**
1. **Capture PID at spawn** - Store bun process PID in workspace `.process_id` file when spawning (cmd/orch/spawn_cmd.go)
2. **Kill process in `orch complete`** - After DeleteSession, read `.process_id` and kill process if still running (cmd/orch/complete_cmd.go)
3. **Enhance `CleanStaleSessions`** - After DeleteSession, find and kill associated bun process (pkg/cleanup/sessions.go)
4. **Add daemon orphan detection** - Periodically scan for bun processes with no corresponding session or tmux window (new)

### Alternative Approaches Considered

**Option B: Modify OpenCode to kill process on DeleteSession**
- **Pros:** Centralized fix in OpenCode; all DeleteSession calls automatically clean up processes
- **Cons:** Requires changes to OpenCode (external dependency); may affect other OpenCode users; doesn't solve orphan detection
- **When to use instead:** If this is a general OpenCode issue affecting all users, not just orch-go

**Option C: Manual cleanup via `orch clean --processes`**
- **Pros:** Simple command for users to run; no automatic changes to lifecycle
- **Cons:** Doesn't prevent accumulation, only reactive; users must remember to run it; doesn't address root cause
- **When to use instead:** As a stopgap while implementing automatic cleanup

**Option D: Use process groups and SIGHUP**
- **Pros:** Unix-native solution using process groups; tmux window close automatically kills children
- **Cons:** Requires rearchitecting how processes are spawned; may interfere with agent suspend/resume; complex edge cases
- **When to use instead:** If we want agents to die automatically when tmux window closes (but this breaks suspend/resume patterns)

**Rationale for recommendation:** Option A (process tracking + explicit termination) addresses the root cause while preserving existing behavior. It works for both successful completion and crash recovery, doesn't depend on external changes, and integrates cleanly with existing cleanup flows. Options B-D either shift the problem, don't address crash recovery, or break existing workflows.

---

### Implementation Details

**What to implement first:**
- Add `.process_id` file creation at spawn time (cmd/orch/spawn_cmd.go:1097) - foundational for all cleanup
- Update `orch complete` to kill process after DeleteSession (cmd/orch/complete_cmd.go:1023) - immediate impact on successful completions
- Test with a real agent spawn + complete cycle to verify process is killed

**Things to watch out for:**
- ⚠️ Race condition: process may already be dead when we try to kill it (handle ESRCH error gracefully)
- ⚠️ PID reuse: OS may reuse PID for different process (verify process is actually bun + opencode before killing)
- ⚠️ Docker-backend spawns: may have different process lifecycle (verify `.process_id` works with docker mode)
- ⚠️ Interactive sessions: users running `opencode` manually shouldn't be killed by cleanup (verify workspace exists before killing)
- ⚠️ Signal handling: use SIGTERM first, wait, then SIGKILL if needed (graceful shutdown)

**Areas needing further investigation:**
- Whether OpenCode has built-in process management we should use instead (audit OpenCode documentation)
- How frequent orphaned processes actually occur (add telemetry to track crash rate vs success rate)
- Whether some long-running agents are legitimately active (cross-reference all 26 processes with workspaces)
- Performance impact of periodic orphan detection (benchmark process scanning at scale)

**Success criteria:**
- ✅ After `orch complete`, bun process is terminated (verified via `ps -p <pid>`)
- ✅ After daemon CleanStaleSessions runs, orphaned processes are killed (count of bun processes decreases)
- ✅ No new orphans accumulate over 1 week period (monitor bun process count daily)
- ✅ Add metrics: track orphans_detected, orphans_killed, cleanup_failures
- ✅ Graceful handling of already-dead processes (no errors logged)

---

## References

**Files Examined:**
- cmd/orch/complete_cmd.go:1013-1109 - Cleanup sequence in orch complete (DeleteSession, archive, kill window)
- pkg/cleanup/sessions.go:27-131 - Periodic session cleanup implementation
- cmd/orch/daemon.go:154-157 - Daemon cleanup configuration flags
- pkg/opencode/client.go:876-895 - DeleteSession implementation (HTTP DELETE only)
- cmd/orch/spawn_cmd.go:1074-1110 - Process spawning via opencode CLI
- .orch/workspace/og-debug-harden-orch-dashboard-29jan-b5ee/SESSION_LOG.md - Crash evidence

**Commands Run:**
```bash
# Count active tmux sessions
tmux list-sessions 2>/dev/null | wc -l  # Result: 16 sessions

# Count bun processes
ps aux | grep -i 'bun.*index.ts' | grep -v grep | wc -l  # Result: 27 processes

# Check old orphaned process
ps -p 31395 -o pid,lstart,etime,command  # Result: 1d 17h runtime

# Verify workspace exists but has no tmux window
ls -la .orch/workspace/og-debug-harden-orch-dashboard-29jan-b5ee/
tmux list-windows | grep og-debug-harden-orch-dashboard-29jan-b5ee  # Result: no window

# Test killing orphaned process
kill -9 31395  # Success: process terminated, nothing broke
ps aux | grep -i 'bun.*index.ts' | grep -v grep | wc -l  # Result: 26 processes
```

**External Documentation:**
- None referenced

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-31-inv-investigate-there-26-opencode-bun.md (this file)
- **Workspace:** .orch/workspace/og-debug-harden-orch-dashboard-29jan-b5ee - Example orphaned workspace from crashed agent

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

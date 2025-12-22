<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch status` shows 41 "active" agents because it counts ALL tmux windows across ALL workers-* sessions, not just actually running agents.

**Evidence:** Counted 41 windows across 9 workers-* sessions; only 4 OpenCode sessions in API; all tmux windows show idle `/status` prompt.

**Knowledge:** The "active" count is misleading - it reflects persistent tmux windows, not running agents. `orch clean` doesn't close windows or reduce this count.

**Next:** Implement proper cleanup: `orch clean` should kill tmux windows for completed agents, or `orch status` should filter to only truly active agents.

---

# Investigation: 40+ Agents Showing as Active in orch status

**Question:** Why does `orch status` show 40+ active agents when most appear to be phantoms? What is the cleanup lifecycle and is `orch clean` being run properly?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** og-inv-40-agents-showing-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: orch status counts ALL workers-* tmux windows as "active"

**Evidence:** 
- `orch status` reports 41 active agents
- Counted tmux windows across all workers sessions:
  - workers-orch-go: 27 agent windows
  - workers-beads: 3 windows
  - workers-beads-ui-svelte: 5 windows
  - workers-kb-cli: 2 windows
  - workers-skillc: 2 windows
  - workers-test-template-repo: 1 window
  - workers-orch-knowledge: 1 window
  - Total: 41 windows (matches status count exactly)

**Source:** 
- `cmd/orch/main.go:1571-1609` - Phase 1 collects from `tmux.ListWorkersSessions()`
- `tmux list-sessions` output showing 9 workers-* sessions
- `tmux list-windows -t workers-orch-go` showing 29 total windows

**Significance:** The "active" count is inflated because it includes every tmux window, regardless of whether the agent inside is actually working.

---

### Finding 2: All tmux windows contain idle OpenCode sessions

**Evidence:**
Checked agent activity by capturing pane content:
```
Window 2-28: All show "/status" at prompt (IDLE)
Window 20: Shows "❯" (shell prompt, not OpenCode)
```

The OpenCode TUI is open but sitting at the prompt waiting for input.

**Source:**
- `tmux capture-pane -t "workers-orch-go:N" -p | tail -1`
- All windows show `/status` indicator

**Significance:** These are not "phantom agents" (beads open but not running) - they are completed agents where the tmux window was never closed.

---

### Finding 3: OpenCode API only tracks 4 active sessions

**Evidence:**
```bash
$ curl -s http://127.0.0.1:4096/session | jq 'length'
4
```

Only 4 sessions in OpenCode's memory, vs 41 counted by `orch status`.

**Source:** 
- OpenCode API `/session` endpoint
- `cmd/orch/main.go:1611-1614` - Phase 2 collects from `client.ListSessions("")`

**Significance:** OpenCode's session list is accurate. The 41 count comes from tmux windows, not OpenCode.

---

### Finding 4: orch clean does NOT close tmux windows

**Evidence:**
From `cmd/orch/main.go:2261-2264`:
```go
fmt.Printf("  Cleaning: %s (%s)\n", ws.Name, ws.Reason)
// Note: We don't delete the workspace directory itself
// Workspaces are kept for investigation reference
```

`orch clean` only:
1. Identifies cleanable workspaces (by SYNTHESIS.md or closed beads issue)
2. Reports them
3. Optionally cleans orphaned OpenCode disk sessions

It does NOT:
- Kill tmux windows
- Remove workspace directories
- Update any state that would reduce the "active" count

**Source:**
- `cmd/orch/main.go:2233-2309` - `runClean()` function
- `orch clean --dry-run` output showing 117 cleanable workspaces

**Significance:** Running `orch clean` doesn't fix the 41-agent count problem.

---

### Finding 5: Cross-project window aggregation

**Evidence:**
`orch status` collects windows from ALL projects' workers sessions:
```go
workersSessions, _ := tmux.ListWorkersSessions()
for _, sessionName := range workersSessions {
    windows, _ := tmux.ListWindows(sessionName)
    // ...
}
```

This explains why running `orch status` in orch-go shows agents from beads, skillc, kb-cli, etc.

**Source:** `cmd/orch/main.go:1572-1573`

**Significance:** The status command provides a global swarm view, but this makes the count confusing when you expect to see only current-project agents.

---

## Synthesis

**Key Insights:**

1. **Tmux windows persist after agent completion** - When agents complete, they leave their OpenCode TUI open at the prompt. Nobody closes these windows, so they accumulate.

2. **orch clean is misleadingly named** - It identifies "cleanable" workspaces but doesn't actually clean up tmux windows or reduce the active agent count.

3. **No cleanup lifecycle exists** - The expected lifecycle would be: spawn → work → complete → cleanup. The cleanup step (killing tmux window) is missing.

**Answer to Investigation Question:**

The 41 "active" agents are ghost windows - completed agents whose tmux windows were never closed. The cleanup lifecycle is incomplete: `orch complete` closes beads issues but doesn't kill windows, and `orch clean` only reports completions without actually cleaning up.

---

## Test Performed

**Test:** Ran `orch status`, counted tmux windows across all sessions, queried OpenCode API, tested `orch clean --dry-run`.

**Result:**
- `orch status`: 41 active
- Tmux windows (excluding servers/zsh): 41
- OpenCode sessions in API: 4
- `orch clean --dry-run`: 117 cleanable workspaces, no window cleanup

This confirms the 41 count comes from persistent tmux windows, not actual running agents.

---

## Implementation Recommendations

**Purpose:** Fix the misleading active agent count and implement proper cleanup.

### Recommended Approach: orch clean --windows

Add a `--windows` flag (or make it default) that:
1. For each cleanable workspace, find its associated tmux window
2. Kill the tmux window if the agent is complete

**Why this approach:**
- Directly addresses the root cause (windows not being closed)
- Preserves workspace directories for reference (current behavior)
- Makes `orch clean` actually clean something

**Trade-offs accepted:**
- Agents lose their terminal output when window is killed
- Risk of killing window where someone is still looking

**Implementation sequence:**
1. Add `tmux.KillWindowByBeadsID()` function
2. In `runClean()`, for each cleanable workspace with beadsID, kill the window
3. Update status count to exclude cleaned windows

### Alternative Approaches Considered

**Option B: Filter orch status to show only truly active**
- **Pros:** Doesn't change window state, just improves display accuracy
- **Cons:** Doesn't free up tmux resources, windows still accumulate
- **When to use instead:** If preserving window access is critical

**Option C: Auto-close windows on orch complete**
- **Pros:** Cleanup happens at the right moment
- **Cons:** Breaks current workflow where users may want to review terminal
- **When to use instead:** If clean-as-you-go is preferred

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn constrain "orch status counts ALL workers-* tmux windows as active" --reason "Discovered during phantom agent investigation - status inflated by persistent windows"
```

**Leave it Better: Constraint recorded about orch status behavior.**

---

## References

**Files Examined:**
- `cmd/orch/main.go:1563-1717` - runStatus() function
- `cmd/orch/main.go:2233-2310` - runClean() function

**Commands Run:**
```bash
# Count tmux sessions with workers prefix
tmux list-sessions | grep workers

# Count windows per session
for session in $(tmux list-sessions -F '#{session_name}' | grep workers); do
  tmux list-windows -t "$session" | wc -l
done

# Check OpenCode session count
curl -s http://127.0.0.1:4096/session | jq 'length'

# Test clean behavior
orch clean --dry-run
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md` - Related stale sessions issue

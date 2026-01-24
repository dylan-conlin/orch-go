<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `--windows` flag to `orch clean` command that closes tmux windows for completed agents.

**Evidence:** Dry run test shows 27 tmux windows identified for 120 cleanable workspaces; all tests pass.

**Knowledge:** Window names follow pattern `{emoji} {workspace-name} [{beads-id}]`, enabling lookup by workspace name.

**Next:** Close this issue - feature complete and tested.

**Confidence:** High (90%) - Tests pass, dry run works, but real cleanup not tested with actual window closure.

---

# Investigation: Add --windows Flag to orch clean

**Question:** How to implement tmux window cleanup for completed agents in `orch clean`?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** og-feat-add-windows-flag-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Window naming convention enables reliable lookup

**Evidence:**
Window names follow predictable pattern: `{emoji} {workspace-name} [{beads-id}]`
Example: `🔬 og-inv-40-agents-showing-22dec [orch-go-xyz]`

**Source:** `pkg/tmux/tmux.go:164-180` - BuildWindowName function

**Significance:** Enables reliable window lookup by workspace name, which is always present in the window name.

---

### Finding 2: All workers sessions must be searched

**Evidence:**
Cleanable workspaces can exist in any workers-* session (workers-orch-go, workers-beads, etc.).
A project-local clean needs to search across all sessions.

**Source:** Investigation `.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md` showed 9 workers sessions.

**Significance:** Required implementing `FindWindowByWorkspaceNameAllSessions` that iterates all workers sessions.

---

### Finding 3: Opt-in approach is safer

**Evidence:**
Users may want to review terminal output from completed agents. Making window closure automatic could disrupt workflow.

**Source:** Prior investigation noted agents' terminal output can be useful for debugging.

**Significance:** Implemented as `--windows` flag (opt-in) rather than default behavior.

---

## Synthesis

**Key Insights:**

1. **Reliable workspace-to-window mapping** - Window names always contain workspace name, enabling reliable lookup without needing registry or session IDs.

2. **Cross-session search required** - Workspaces can span multiple workers sessions, requiring search across all `workers-*` sessions.

3. **Safe opt-in design** - Making window cleanup opt-in prevents accidental loss of terminal output while still solving the phantom agent problem.

**Answer to Investigation Question:**

Implemented via:
1. `FindWindowByWorkspaceName(sessionName, workspaceName)` - Find in specific session
2. `FindWindowByWorkspaceNameAllSessions(workspaceName)` - Search all workers sessions
3. `--windows` flag in `orch clean` that triggers window closure for cleanable workspaces

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
- All tests pass including new window lookup tests
- Dry run correctly identifies 27 windows to close
- Implementation follows established patterns in codebase

**What's certain:**
- ✅ Window naming convention is reliable
- ✅ Window lookup functions work correctly (tested)
- ✅ Integration with clean command works (dry run verified)

**What's uncertain:**
- ⚠️ Actual window closure not tested in CI (requires live tmux)
- ⚠️ Edge case: Window closed by user but workspace not cleaned

---

## Implementation Details

**Files Modified:**
- `pkg/tmux/tmux.go` - Added `FindWindowByWorkspaceName`, `FindWindowByWorkspaceNameAllSessions`
- `pkg/tmux/tmux_test.go` - Added tests for new functions
- `cmd/orch/main.go` - Added `--windows` flag, integrated window closing in runClean

**Usage:**
```bash
# Dry run - see what would be closed
orch clean --dry-run --windows

# Actually close windows
orch clean --windows
```

---

## References

**Files Examined:**
- `cmd/orch/main.go:2050-2310` - Clean command implementation
- `pkg/tmux/tmux.go` - Tmux window management functions
- `.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md` - Root cause investigation

**Commands Run:**
```bash
# Build and verify
go build ./...
go test ./...

# Test dry run
orch clean --dry-run --windows
```

---

## Self-Review

- [x] Real test performed (go test, dry run)
- [x] Conclusion from evidence (tests pass, dry run works)
- [x] Question answered (implementation complete)
- [x] File complete

**Self-Review Status:** PASSED

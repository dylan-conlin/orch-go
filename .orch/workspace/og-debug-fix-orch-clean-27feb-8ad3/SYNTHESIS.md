# Session Synthesis

**Agent:** og-debug-fix-orch-clean-27feb-8ad3
**Issue:** orch-go-15a2
**Outcome:** success

---

## Plain-Language Summary

`orch clean --sessions` was killing actively running Claude Code agents in tmux windows because it only checked for OpenCode sessions (which don't exist for Claude CLI agents) and open beads issues (which may be closed or fail to query). The fix adds a third, last-resort safety check: before classifying a window as stale, it now verifies whether the tmux pane has an active non-shell process running. If `claude`, `opencode`, or any other agent process is alive, the window is protected from cleanup — regardless of OpenCode or beads state.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## TLDR

Fixed `orch clean` killing active Claude Code tmux agents by adding process liveness detection. Uses dual signals (tmux `pane_current_command` + child process checking via `pgrep`) as a last-resort safety net before classifying windows as stale.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/clean_cmd.go` — Added `PaneProcessChecker` interface, `DefaultPaneProcessChecker` with dual-signal liveness detection, updated `classifyTmuxWindows` to accept and use the checker
- `cmd/orch/clean_cmd_test.go` — Added 6 new test cases for process liveness protection, mock process checker, idle shell commands test
- `pkg/tmux/tmux.go` — Added `GetPaneCurrentCommand()` and `GetPanePID()` exported functions for socket-aware tmux pane queries

---

## Evidence (What Was Observed)

- `tmux list-panes -t @662 -F '#{pane_current_command}'` returned `zsh` even though `claude` (PID 81396) was actively running as a child process
- macOS tmux 3.5a has unreliable `pane_current_command` — it reports the shell rather than the foreground child process
- `pgrep -P <shell_pid>` correctly finds child processes and is the reliable signal on macOS
- The dual-signal approach (pane_current_command OR child processes) handles both Linux (where pane_current_command works) and macOS (where child process check is needed)

### Tests Run
```bash
go test ./cmd/orch/ -run "TestClassifyTmuxWindows|TestIdleShellCommands" -v
# 14 passed, 0 failed (0.012s)

go test ./cmd/orch/ -run "TestClean" -v
# 8 passed, 0 failed, 1 skipped (0.137s)

go vet ./cmd/orch/ ./pkg/tmux/
# No issues

orch clean --sessions --dry-run
# 0 active OpenCode sessions, protected 2 windows, no stale windows found
```

---

## Architectural Choices

### Dual-signal process detection over pane_current_command alone
- **What I chose:** Check pane_current_command first, then fall back to child process detection via `pgrep -P`
- **What I rejected:** Using only `pane_current_command` (unreliable on macOS) or only `pgrep` (misses non-child processes)
- **Why:** Discovered during development that macOS tmux 3.5a reports `zsh` for `pane_current_command` even when `claude` is running as a foreground child. The dual approach works on both macOS and Linux.
- **Risk accepted:** `pgrep` dependency — should be available on all Unix systems

### Interface-based process checker for testability
- **What I chose:** `PaneProcessChecker` interface with `DefaultPaneProcessChecker` implementation and mock for tests
- **What I rejected:** Inline process checking without interface (harder to test)
- **Why:** The classification function needs to be testable without real tmux. Interface allows mock injection in tests while using real tmux in production.

### Process check as last-resort, not first check
- **What I chose:** Check order: OpenCode session → beads issue → process liveness
- **What I rejected:** Process check first (would make every call query tmux panes)
- **Why:** Most windows are filtered by the cheaper OpenCode/beads checks. Process check only runs for windows that would otherwise be classified stale, minimizing tmux API calls.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- macOS tmux 3.5a: `pane_current_command` format variable is unreliable for detecting foreground child processes — reports the login shell even when a child process is running in the foreground
- `pgrep -P <pid>` is the reliable cross-platform way to detect if a shell has active child processes

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (14 classify tests + idle shell test + 8 clean tests)
- [x] Binary installed and smoke-tested
- [x] Ready for `orch complete orch-go-15a2`

---

## Unexplored Questions

- macOS tmux `pane_current_command` behavior: is this a known tmux bug or expected behavior? Could file upstream.
- Should `orch status` also use process liveness to more accurately report agent state? Currently it only checks OpenCode sessions.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-fix-orch-clean-27feb-8ad3/`
**Beads:** `bd show orch-go-15a2`

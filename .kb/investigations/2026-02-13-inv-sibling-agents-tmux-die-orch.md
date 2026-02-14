## Summary (D.E.K.N.)

**Delta:** No code path in `orch complete` kills sibling tmux windows; the `cleanPhantomWindows` function in `orch clean` has a bug that kills Claude-mode agent windows by misclassifying them as phantoms.

**Evidence:** Exhaustive code trace of `runComplete()` + reproduction test (kill-window, make install, overmind restart api) — all siblings survived; `cleanPhantomWindows` checks OpenCode sessions but Claude-mode agents have none.

**Knowledge:** Agent state is split across 4 layers (OpenCode sessions, tmux windows, beads issues, registry) and cleanup functions that only check one layer produce false positives on others.

**Next:** Investigation was spawned for wrong task (should have been headless OpenCode spawns dying). Findings documented for future reference. The `cleanPhantomWindows` bug should be fixed separately.

**Authority:** implementation - Bug fix within existing patterns

---

# Investigation: Sibling Agents Tmux Die Orch (WRONG TASK - ABORTED)

**Question:** Why do sibling agent tmux windows disappear when `orch complete` runs for another agent?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Investigation agent
**Phase:** Complete (aborted - wrong task)
**Next Step:** None
**Status:** Complete (partial - spawned for wrong task)

**Note:** This investigation was spawned by mistake. The real task was investigating why headless OpenCode spawns were dying. Findings below are still valid for the tmux sibling death question.

---

## Findings

### Finding 1: No code in orch complete kills sibling windows

**Evidence:** Exhaustive trace of `runComplete()` (1136 lines). The function:
1. Closes beads issue (bd close → on_close hook → orch emit)
2. Exports activity to workspace
3. Deletes OpenCode session via API
4. Archives workspace (moves to archived/)
5. Kills ONE tmux window (target only, found by `[beadsID]` pattern)
6. Runs `make install` if Go changes detected
7. Runs `overmind restart api`
8. Logs event and invalidates cache

Each step tested individually and combined — siblings survived every test.

**Source:** cmd/orch/complete_cmd.go:506-1136, pkg/tmux/tmux.go:802-901

**Significance:** The bug is NOT in orch complete itself.

### Finding 2: cleanPhantomWindows misclassifies Claude-mode agents

**Evidence:** `cleanPhantomWindows` (clean_cmd.go:650-745) considers a tmux window a "phantom" if its beads ID has no matching OpenCode session. Claude-mode agents (the default spawn backend) run `claude --dangerously-skip-permissions` directly — they have NO OpenCode sessions. If `orch clean --phantoms` or `orch clean --all` is run, ALL Claude-mode agent windows would be killed.

**Source:** cmd/orch/clean_cmd.go:650-745, cmd/orch/spawn_cmd.go:1134 (default backend = "claude")

**Significance:** This is the only code path found that kills multiple tmux windows. Requires explicit `orch clean` invocation — not triggered by `orch complete`.

### Finding 3: Stale com.orch.reap launchd agent

**Evidence:** `~/Library/LaunchAgents/com.orch.reap.plist` runs `orch reap` every 5 minutes, but the command was removed from the codebase. Last log entry: Feb 11. Currently fails silently.

**Source:** ~/Library/LaunchAgents/com.orch.reap.plist, ~/.orch/logs/reap.log

**Significance:** Stale automation artifact. Should be unloaded.

### Finding 4: tmux global hook on after-select-window

**Evidence:** `after-select-window[0]` runs `~/.local/bin/sync-workers-session.sh` which switches the workers Ghostty window to match the orchestrator's project context. Only triggers in orchestrator session, only switches client sessions — does NOT kill windows.

**Source:** `tmux show-hooks -g`, ~/.local/bin/sync-workers-session.sh

**Significance:** Ruled out as cause.

### Finding 5: remain-on-exit is OFF

**Evidence:** `remain-on-exit` defaults to off. Agent windows start zsh, then send-keys runs `cat SPAWN_CONTEXT.md | claude`. When claude exits, zsh returns to prompt — window stays open. For the window to close, zsh itself must exit (signal, kill, or explicit exit).

**Source:** tmux show-option -g remain-on-exit (not set = off)

**Significance:** Windows only close if the shell process dies. Something external must be killing it.

---

## Synthesis

**Key Insights:**

1. **orch complete is innocent** — Every step traced and tested. The only window kill targets the specific agent by `[beadsID]` pattern match.

2. **cleanPhantomWindows has a real bug** — It doesn't account for Claude-mode agents and would kill all their windows if invoked.

3. **Agent state fragmentation** — State exists in 4 layers (OpenCode sessions, tmux, beads, registry). Functions that check only one layer (like phantom detection checking only OpenCode) produce incorrect results for agents using other layers.

**Answer to Investigation Question:**

The root cause was NOT identified definitively because the investigation was spawned for the wrong task. The most likely explanation is either: (a) `orch clean --phantoms` being run by the orchestrator after completions, (b) Claude Code processes crashing simultaneously (API errors, rate limits), or (c) an environmental factor not reproducible in isolation.

---

## References

**Files Examined:**
- cmd/orch/complete_cmd.go - Full completion flow (1660 lines)
- pkg/tmux/tmux.go - tmux window management (947 lines)
- cmd/orch/autorebuild.go - Binary auto-rebuild mechanism
- pkg/spawn/claude.go - Claude Code spawn implementation
- cmd/orch/spawn_cmd.go - Spawn command with backend selection
- cmd/orch/clean_cmd.go - Clean command with phantom window detection
- cmd/orch/serve.go - Serve command startup
- pkg/service/monitor.go - Service monitor (overmind)
- Procfile, Makefile - Build/service definitions
- .beads/hooks/on_close - Beads close hook
- ~/bin/orch-dashboard - Dashboard management script
- ~/.claude/hooks/* - All Claude Code hooks
- ~/.local/bin/sync-workers-session.sh - tmux after-select-window hook
- ~/Library/LaunchAgents/com.orch.reap.plist - Stale reap agent

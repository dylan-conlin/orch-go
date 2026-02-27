# SYNTHESIS: Daemon Duplicate Spawn — Tmux Window Dedup Gap

**Issue:** `orch-go-eqjn`
**Type:** P0 Bug Fix
**Status:** Fixed

## Root Cause

The daemon's session dedup layer (`HasExistingSessionForBeadsID()` in `pkg/daemon/session_dedup.go`) only checked OpenCode API sessions. When the daemon spawns agents using the Claude CLI backend (the default for Anthropic models since Feb 2026), Claude creates **tmux windows WITHOUT OpenCode sessions**. This meant the entire session dedup layer was bypassed for 55% of all spawns (642 out of 1162 total daemon spawns use `claude` mode).

### Impact

- **30 beads IDs** were spawned as duplicates (8% of all daemon spawns)
- Short-gap duplicates confirmed race conditions: 0s, 3s, 20s, 30s gaps between duplicate spawns
- Duplicate agents on the same codebase cause **file contention** — when one agent crashes from contention, tmux closes its window, appearing as "window disappeared mid-work"
- Recent duplicate (orch-go-1219, Feb 24, 10.2 min gap) confirmed the bug persisted after earlier fixes

### Secondary Finding: Fail-Open Pattern

The fresh beads status check at `daemon.go:679-704` fails-open on RPC errors, allowing spawns without dedup verification on transient beads connection issues. This is lower priority but worth monitoring.

## Fix

**File:** `pkg/daemon/session_dedup.go`

Added Layer 2 tmux window dedup to `HasExistingSessionForBeadsID()`:

1. **Layer 1 (existing):** Check OpenCode API sessions (covers headless backend)
2. **Layer 2 (new):** Check tmux windows via `tmux.FindWindowByBeadsIDAllSessions()` (covers Claude CLI backend)

The combined check now returns `true` if EITHER an OpenCode session OR a tmux window exists for the given beads ID.

New function `HasExistingTmuxWindowForBeadsID()` searches all worker, orchestrator, and meta-orchestrator tmux sessions for a window whose name contains `[beadsID]`. Fails-open on tmux errors (consistent with existing fail-open pattern).

Also updated the debug message in `daemon.go:spawnIssue()` to reflect the expanded check.

## Tests Added

3 new tests in `pkg/daemon/session_dedup_test.go`:

- `TestHasExistingTmuxWindowForBeadsID_NoWindow` — nonexistent beads ID returns false
- `TestHasExistingTmuxWindowForBeadsID_WindowExists` — creates real tmux window, verifies detection
- `TestHasExistingSessionForBeadsID_TmuxFallback` — mock OpenCode returns no sessions, but tmux window exists → combined check returns true (exact bug scenario)

## Discovered Work

1. **Pool active count undercounting** (`pkg/daemon/active_count.go`): `DefaultActiveCount()` only counts OpenCode sessions for pool reconciliation. Claude backend agents are invisible to the pool, meaning the daemon thinks it has more capacity than it actually does. This doesn't cause duplicate spawns (the dedup fix handles that), but may cause over-spawning beyond the configured `max_agents` limit.

2. **Fail-open fresh status check** (`pkg/daemon/daemon.go:679-704`): Consider making the fresh beads status check fail-closed or adding a warning when it falls back due to RPC errors.

## Files Changed

- `pkg/daemon/session_dedup.go` — Added tmux import, expanded `HasExistingSessionForBeadsID()` with tmux layer, added `HasExistingTmuxWindowForBeadsID()`
- `pkg/daemon/session_dedup_test.go` — Added 3 integration tests with real tmux windows
- `pkg/daemon/daemon.go` — Updated comment and debug message in `spawnIssue()` to reflect expanded dedup scope

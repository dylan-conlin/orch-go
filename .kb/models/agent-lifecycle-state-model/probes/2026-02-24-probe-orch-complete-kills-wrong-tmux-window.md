# Probe: orch complete kills wrong tmux window when using window index instead of window ID

**Date:** 2026-02-24
**Status:** Complete
**Model:** Agent Lifecycle State Model
**Invariant tested:** "Tmux windows are UI layer only" + completion cleanup correctness

## Question

Does `orch complete` (and related cleanup code) use stable tmux window IDs (`@-prefixed`) or unstable window indices (`session:N`) when killing tmux windows? If using indices, can this cause the wrong window to be killed when multiple agents share a tmux session?

## What I Tested

### Code path analysis: all callers of `tmux.KillWindow`

Traced every call site that kills a tmux window:

1. **`cmd/orch/complete_cleanup.go:37`** - `tmux.KillWindow(window.Target)`
   - `window.Target` = `session:window_index` (e.g., `workers-orch-go:3`)
   - Found via `FindWindowByBeadsIDAllSessions` or `FindWindowByWorkspaceNameAllSessions`

2. **`cmd/orch/clean_cmd.go:496`** - `tmux.KillWindow(pw.window.Target)`
   - Same pattern: `window.Target` = `session:window_index`

3. **`cmd/orch/abandon_cmd.go:202`** - `tmux.KillWindow(windowInfo.Target)`
   - Same pattern

4. **`cmd/orch/review.go:921`** - `tmux.KillWindow(window.Target)`
   - Same pattern

### tmux package analysis

The `pkg/tmux/tmux.go` package provides BOTH:
- `KillWindow(windowTarget string)` - uses `session:index` format (line 694)
- `KillWindowByID(windowID string)` - uses stable `@ID` (line 703)

The `WindowInfo` struct contains BOTH:
- `Target string` = `session:index` (unstable)
- `ID string` = `@1234` (stable, unique for tmux server lifetime)

All 4 callers use `window.Target` (unstable) instead of `window.ID` (stable).

### Window index instability analysis

tmux window indices are unstable when:
1. **`renumber-windows` option is on** (common in user tmux configs): when window 3 is killed, window 4 becomes window 3
2. **TOCTOU race**: between `ListWindows`/`FindWindowBy*` resolving the index and `KillWindow` executing, concurrent completions can shift indices

### Spawn records stable window ID

`pkg/orch/spawn_modes.go:311`:
```go
windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
```

`tmux.CreateWindow` at line 508 uses `-P -F "#{window_index}:#{window_id}"` to capture BOTH. The `windowID` is the stable `@ID`. It's logged in events (line 399) but NOT stored in AGENT_MANIFEST for later use by complete.

Complete doesn't use manifest window ID anyway - it does a fresh lookup by beads ID, getting the current `WindowInfo` with both `Target` and `ID`, then uses the wrong one.

## What I Observed

**BUG CONFIRMED:** All 4 cleanup code paths use `window.Target` (unstable `session:index`) instead of `window.ID` (stable `@-prefixed ID`).

The `KillWindowByID` function exists and is already used correctly by:
- `DefaultLivenessChecker.WindowExists` in `clean_cmd.go:96` (correctly uses `WindowExistsByID`)

But the actual kill operations all use the unstable path.

**Race scenario:**
1. Agent A at index 3, Agent B at index 4
2. Two simultaneous `orch complete` calls both resolve correct indices
3. Agent A's complete kills index 3 first
4. With `renumber-windows` on, Agent B shifts from index 4 to index 3
5. Agent B's complete tries to kill index 4, which is now either:
   - Non-existent (benign - cleanup silently fails)
   - A different agent entirely (catastrophic - wrong window killed)

## Fix

Change all 4 callers from `tmux.KillWindow(window.Target)` to `tmux.KillWindowByID(window.ID)`.

## Model Impact

**Extends Invariant 6:** "Tmux windows are UI layer only - Not authoritative for state"

The model correctly states tmux is non-authoritative, but doesn't warn about the **fragility of window index targeting**. All cleanup operations should use stable window IDs (`@-prefixed`), never window indices.

**New constraint:** tmux window operations (kill, send-keys, capture-pane) should prefer `@ID` over `session:index` targeting when the ID is available. The `WindowInfo.Target` field is a convenience for display/logs but should NOT be used as a tmux target for mutating operations.

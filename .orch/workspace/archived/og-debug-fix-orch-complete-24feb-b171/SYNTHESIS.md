# Synthesis: Fix tmux window cleanup in orch complete

## TLDR
Moved tmux window cleanup from a fixed position late in `runComplete()` to a `defer` that runs on all exit paths. This ensures phantom tmux windows are always cleaned up, even when verification gates cause early returns.

## Outcome: success

## What Changed
- **Created `cmd/orch/complete_cleanup.go`**: Extracted `cleanupTmuxWindow()` function that finds and kills the tmux window by beads ID (or workspace name for orchestrator sessions).
- **Modified `cmd/orch/complete_cmd.go`**:
  - Added `defer cleanupTmuxWindow(...)` after variable resolution (~line 525), ensuring cleanup runs on ALL exit paths
  - Added `skipTmuxCleanup` flag at liveness-check early returns (agent still running → don't kill window)
  - Removed the old 27-line cleanup block at the end of the function (net reduction in file size)
- **Created `cmd/orch/complete_cleanup_test.go`**: Tests for no-op behavior, orchestrator path, and identifier fallback.

## Root Cause
The tmux cleanup code existed (added Jan 18, 2026 in commit `1edef47ad`) but was positioned at line 1154 of `runComplete()`, AFTER 12+ early return paths:
- Comprehension gate (gate1) missing
- Behavioral verification (gate2) missing
- Orchestrator/worker verification failed
- Agent still running (liveness check)
- Discovered work not dispositioned
- Explain-back gate missing
- Issue close failure

When ANY verification gate returned early, the tmux window cleanup was skipped, leaving phantom windows.

## Design Decisions
- **`defer` with skip flag**: Using Go's `defer` ensures cleanup runs on all paths. The `skipTmuxCleanup` flag prevents killing windows when the agent is detected as still running (liveness check), which is the only case where cleanup should NOT happen.
- **Cleanup on verification failure**: If a user runs `orch complete` and verification fails, the window is still cleaned up. By the time you're completing, the agent process has finished — the window is just a dead pane.
- **File extraction reduces hotspot**: `complete_cmd.go` was >1500 lines. Extracting 27 lines and replacing with 7 lines of defer setup reduces the file by 20 lines.

## Recommendation: close

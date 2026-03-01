# Synthesis: Daemon Orphan Detection for in_progress Issues

## Problem

When a beads issue is marked `in_progress` but has no active agent working on it (no OpenCode session, no tmux window), the daemon's `NextIssueExcluding` skips it, and nothing drives it forward. The issue gets permanently stuck.

**Root cause:** Several failure modes create orphans:
- Agent crash after status update to `in_progress`
- OpenCode server crash killing sessions
- Manual status change without spawning
- Spawn failure with rollback failure

**Why existing recovery doesn't help:** `RunPeriodicRecovery` handles *stuck agents* (agents that exist but are idle) by sending resume prompts. For issues with *no agent at all*, there was no recovery path.

## Solution

Added a new periodic orphan detection task following the established daemon pattern (`ShouldRunX` + `RunPeriodicX` + config + main loop wiring).

### Detection Flow

For each `in_progress` issue returned by `GetActiveAgents()`:

1. Skip if BeadsID is empty
2. Skip if Phase: Complete (waiting for orchestrator review, not orphaned)
3. Skip if idle time < age threshold (too new, still starting up)
4. Check `HasExistingSessionForBeadsID()` (OpenCode sessions + tmux windows)
5. If no session found: reset to `open` via `UpdateBeadsStatus()`, unmark from `SpawnedIssues` tracker

### Key Design Decisions

- **Separate from recovery** - Different action (reset vs resume), different schedule (30m vs 5m), different signals
- **Reset to `open`** (not just flag) - Gets work unblocked autonomously; issue retains `triage:ready` label so daemon respawns it next cycle
- **1-hour age threshold** - Avoids false positives during normal spawn startup window
- **Unmark from SpawnedIssues** - Without this, the 6h dedup TTL would prevent respawning

## Files Changed

| File | Change |
|------|--------|
| `pkg/daemonconfig/config.go` | 3 config fields + defaults: `OrphanDetectionEnabled`, `OrphanDetectionInterval` (30m), `OrphanAgeThreshold` (1h) |
| `pkg/daemon/daemon.go` | `lastOrphanDetection` timestamp + 3 mock function fields for testing |
| `pkg/daemon/orphan_detector.go` | **NEW** - Core detection logic, result types, time accessors |
| `pkg/daemon/orphan_detector_test.go` | **NEW** - 18 tests covering all detection paths |
| `cmd/orch/daemon.go` | CLI flags, config mapping, startup logging, main loop wiring with event logging |

## Verification

- `go build ./cmd/orch/` - compiles cleanly
- `go vet ./cmd/orch/` - no issues
- 18 orphan detection tests pass
- Full `pkg/daemon/...` test suite passes (no regressions)
- Full `pkg/daemonconfig/...` test suite passes

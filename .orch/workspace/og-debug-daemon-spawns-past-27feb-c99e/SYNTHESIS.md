# Synthesis: Daemon Spawns Past Concurrency Cap

## Bug

The daemon's concurrency cap (default 3) was not enforced for Claude CLI agents running in tmux windows. The daemon spawned 15+ agents in a single session because pool reconciliation reset to 0 every poll cycle.

## Root Cause

`DefaultActiveCount()` in `pkg/daemon/active_count.go` only queries the OpenCode HTTP API (`/session` endpoint). Claude CLI agents run in tmux windows WITHOUT OpenCode sessions, making them invisible to the capacity check. Every 15-second poll cycle, `ReconcileWithOpenCode()` called `DefaultActiveCount()`, got 0, and `Pool.Reconcile(0)` freed all slots.

Ironic: `HasExistingSessionForBeadsID` (per-issue dedup) already checked BOTH OpenCode and tmux. The capacity counting did not.

## Fix

### 1. Combined Active Count (`pkg/daemon/active_count.go`)
- `CountActiveTmuxAgents()` - scans all tmux sessions for orch-managed windows, extracts beads IDs
- `CombinedActiveCount()` - combines OpenCode sessions + tmux windows, deduplicates by beads ID, excludes closed issues

### 2. Daemon Pool Reconciliation (`pkg/daemon/capacity.go`)
- `ReconcileActiveAgents()` uses injectable `activeCountFunc` (defaults to `CombinedActiveCount`)
- `ReconcileWithOpenCode()` preserved as backward-compatible wrapper

### 3. Daemon Struct (`pkg/daemon/daemon.go`)
- Added `activeCountFunc func() int` field, set to `CombinedActiveCount` in constructors

### 4. Manual Spawn Concurrency Gate (`pkg/spawn/gates/concurrency.go`)
- Added Phase 2: scans tmux windows between OpenCode session scan and closed-issue batch check
- Deduplicates by beads ID, marks tmux agents as "running"

## Verification

```bash
go test ./pkg/daemon/ -run TestPoolReconcile      # Pool doesn't reset with tmux agents
go test ./pkg/daemon/ -run TestDaemonReconcile     # Injectable activeCountFunc works
go test ./pkg/spawn/gates/                         # Concurrency gate compiles
go build ./cmd/orch/                               # Full build
go vet ./pkg/daemon/ ./pkg/spawn/gates/            # No issues
```

## Files Changed

| File | Change |
|------|--------|
| `pkg/daemon/active_count.go` | Added `CountActiveTmuxAgents()`, `CombinedActiveCount()` |
| `pkg/daemon/active_count_test.go` | NEW - 4 tests covering reconciliation fix |
| `pkg/daemon/daemon.go` | Added `activeCountFunc` field, set in constructors |
| `pkg/daemon/capacity.go` | Added `ReconcileActiveAgents()`, wrapped `ReconcileWithOpenCode()` |
| `pkg/spawn/gates/concurrency.go` | Added tmux agent counting (Phase 2) |

## Discovered Work

- **orch-go-4omh**: Stale daemon status file bug (separate issue, already tracked)

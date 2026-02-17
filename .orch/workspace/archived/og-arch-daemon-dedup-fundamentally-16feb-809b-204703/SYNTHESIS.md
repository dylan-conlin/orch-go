# SYNTHESIS: Daemon Dedup Fundamentally Broken — Content-Aware Fix

## Problem

The same task ("Extract spawn flags phase 1: --mode") was spawned 5 times as 5 separate beads issues (orch-go-l8k2, kzqq, pg9l, cu0r, xy7n). A prior fix (orch-go-dr0u) addressed single-ID duplicates but didn't hold because the root cause was different: **all dedup layers are keyed on beads issue ID, not content**.

When N issues exist with identical titles but different IDs, every dedup layer correctly passes each one through — they ARE different issues. But they represent the same work.

## Root Cause

Every existing dedup mechanism (SpawnedIssueTracker, session dedup, fresh status check, UpdateBeadsStatus) uses the beads issue ID as its key. Content duplicates (different IDs, same title) bypass all layers.

## Solution

Two-layer content-aware dedup added to daemon's spawn path:

1. **In-memory title tracking** (SpawnedIssueTracker) — fast, catches duplicates within same daemon instance. New `spawnedTitles` map indexes normalized titles to issue IDs, respecting existing TTL.

2. **Persistent beads query** (FindInProgressByTitle) — survives daemon restarts by querying beads database for in_progress issues with matching title. Fails open on errors.

Both layers are checked in `Once()` and `OnceWithSlot()` after session dedup, before fresh status check. Issues with a title matching a recently-spawned or in_progress issue are skipped.

## Files Changed

| File | Change |
|------|--------|
| `pkg/daemon/spawn_tracker.go` | Added title tracking: `spawnedTitles` map, `MarkSpawnedWithTitle()`, `IsTitleSpawned()`, `normalizeTitle()`, cleanup in `Unmark()`/`CleanStale()` |
| `pkg/daemon/issue_adapter.go` | Added `FindInProgressByTitle()` — persistent dedup via beads RPC/CLI |
| `pkg/daemon/daemon.go` | Content-aware dedup checks in `Once()` and `OnceWithSlot()`, `MarkSpawned` → `MarkSpawnedWithTitle` |
| `pkg/daemon/spawn_tracker_test.go` | 6 new tests for title-based dedup (unit + integration) |

## Verification

- All 6 new tests pass
- All existing daemon tests pass (no regressions)
- `go vet` clean
- `go build` succeeds

## Remaining Gaps

- Upstream prevention (beads `CreateArgs.Force` not used by extraction path)
- Extraction loop guard (`DefaultCreateExtractionIssue` could create duplicate extraction issues)
- Recently-closed issue matching (not implemented)

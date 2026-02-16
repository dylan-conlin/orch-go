# Probe: Daemon Dedup Fundamentally Broken — Content-Aware Fix

**Date:** 2026-02-16
**Beads:** orch-go-e0o3
**Model:** daemon-autonomous-operation
**Status:** Confirmed — root cause identified, fix implemented

## Claim Under Test

The daemon's dedup system prevents duplicate spawns for the same work.

## Finding: VIOLATED

The dedup system was **entirely ID-keyed**. When 5 separate beads issues were created with the identical title "Extract spawn flags phase 1: --mode" (IDs: orch-go-l8k2, kzqq, pg9l, cu0r, xy7n), every dedup layer treated them as distinct work and spawned all 5.

### Dedup Layers (all ID-only before fix)

| Layer | Mechanism | Content-Aware? |
|-------|-----------|----------------|
| SpawnedIssueTracker | In-memory `map[issueID]time.Time` | NO — ID-keyed |
| Session dedup | OpenCode API query by beads_id metadata | NO — ID-keyed |
| Fresh status check | `GetBeadsIssueStatus(issueID)` | NO — ID-keyed |
| UpdateBeadsStatus | Mark in_progress before spawn | NO — ID-keyed |

### Why ID-Only Dedup Fails

The root cause is upstream: whatever process creates beads issues can create N issues with identical titles but different IDs. Each issue is a unique ID, so every dedup layer passes it through. This is correct per the ID-keyed design — but wrong per the operational intent of "don't spawn duplicate work."

## Fix: Two-Layer Content-Aware Dedup

### Layer 1: In-Memory Title Tracking (SpawnedIssueTracker)
- Added `spawnedTitles map[normalizedTitle]issueID` to SpawnedIssueTracker
- `MarkSpawnedWithTitle()` records both ID and title
- `IsTitleSpawned()` checks normalized title against tracked spawns
- Respects existing TTL (6 hours)
- Catches duplicates within same daemon process lifetime

### Layer 2: Persistent Beads Query (FindInProgressByTitle)
- New function queries beads for in_progress issues with matching title
- Uses RPC `List` with `Status: "in_progress", Title: title` when available
- Falls back to CLI + local title matching
- Fails open (returns nil on error) to avoid blocking work
- Survives daemon restarts — queries persistent beads database

### Integration Points (daemon.go Once() and OnceWithSlot())
Both spawn paths now check content dedup after session dedup, before fresh status check:
1. Check in-memory title tracker (fast, same-process)
2. Check beads database for in_progress match (persistent, cross-process)
3. Skip issue if either layer finds a match (different ID, same title)

## Test Coverage

6 new tests added to spawn_tracker_test.go:
- `TestSpawnedIssueTracker_TitleDedup` — basic title tracking + case-insensitive
- `TestSpawnedIssueTracker_TitleDedup_TTL` — TTL expiry for title tracking
- `TestSpawnedIssueTracker_TitleDedup_Unmark` — Unmark cleans title index
- `TestSpawnedIssueTracker_TitleDedup_EmptyTitle` — empty title edge case
- `TestDaemon_ContentDedupSkipsDuplicateTitle` — integration: Once() skips duplicate titles
- `TestDaemon_ContentDedupAllowsDifferentTitle` — integration: Once() allows different titles

All tests pass. Existing tests unaffected.

## Remaining Gaps

1. **Closed issue reopen**: If a completed issue with the same title as a new open issue exists, it won't be caught (would need query for recently-closed issues)
2. **Extraction loop guard**: `DefaultCreateExtractionIssue` shells out to `bd create` bypassing Go client dedup — could create duplicate extraction issues
3. **Upstream prevention**: The root fix should also prevent N identical issues from being created in the first place (beads `CreateArgs.Force` exists but isn't used by daemon's extraction path)

## Model Update Recommendation

Update the daemon-autonomous-operation model to include:
- **Constraint**: Dedup must be content-aware, not just ID-keyed
- **Pattern**: Two-layer dedup (fast in-memory + persistent database) for both ID and content matching
- **Prior probes this supersedes**: 2026-02-14 TTL fragility and fail-fast probes addressed symptoms; this probe addresses the fundamental design gap

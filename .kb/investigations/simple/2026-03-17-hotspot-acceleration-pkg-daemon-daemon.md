# Investigation: Hotspot Acceleration — pkg/daemon/daemon_test.go

**TLDR:** daemon_test.go (1308 lines) is a hotspot. Extract tests into focused files aligned with production code: capacity_test.go, cross_project_test.go, spawn_dedup_test.go, spawn_failure_test.go. Reduce to ~142 lines (core happy path only).

**Status:** Complete
**Date:** 2026-03-17

## D.E.K.N. Summary

- **Delta:** daemon_test.go extracted from 1308 lines to ~142 lines across 5 focused test files
- **Evidence:** go test ./pkg/daemon/ passes before and after extraction
- **Knowledge:** Tests clustered into 5 behavioral groups matching production code boundaries: capacity, cross-project, spawn dedup, spawn failure, and core daemon
- **Next:** None — extraction complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

How should pkg/daemon/daemon_test.go (1308 lines, +1326 lines/30d) be extracted to prevent it from becoming a critical hotspot (>1500 lines)?

## Findings

### Finding 1: Test Cluster Analysis

daemon_test.go contains 5 distinct behavioral clusters that map to existing production code files:

1. **Core daemon** (lines 1-61, 102-182): Tests for `Once()`, `Run()` — maps to `daemon.go`
2. **Capacity/pool** (lines 199-616): Tests for `AtCapacity()`, `AvailableSlots()`, `DefaultConfig()`, `NewWithConfig()`, `NewWithPool()`, `PoolStatus()`, `ReconcileWithOpenCode()`, pool integration — maps to `capacity.go`
3. **Cross-project** (lines 809-1041): Tests for `resolveIssueQuerier()`, `ListReadyIssuesMultiProject()`, cross-project spawn — maps to `issue_selection.go`, `project_resolution.go`, `issue_adapter.go`
4. **Spawn dedup** (lines 618-807): Tests for TOCTOU race prevention via fresh status checks, concurrent daemon dedup — maps to spawn pipeline dedup behavior
5. **Spawn failure** (lines 63-100, 1043-1308): Tests for spawn error handling, OnceExcluding sticky failures, Phase: Complete auto-completion — maps to `spawn_execution.go`

Dead code found: `contains()`/`containsHelper()` helper functions (lines 184-197) are never called.

### Finding 2: Extraction Targets

| New File | Source Lines | Line Count | Production File(s) |
|---|---|---|---|
| capacity_test.go | 199-616 | ~418 | capacity.go |
| cross_project_test.go | 809-1041 | ~233 | project_resolution.go, issue_selection.go |
| spawn_dedup_test.go | 618-807 | ~190 | spawn_execution.go (dedup pipeline) |
| spawn_failure_test.go | 63-100 + 1043-1308 | ~304 | spawn_execution.go |
| daemon_test.go (kept) | 1-61 + 102-182 | ~142 | daemon.go |

### Finding 3: Mock Dependencies

All extracted tests use mocks from `mock_test.go` (shared across package) and `mockAutoCompleter` from `auto_complete_test.go`. No mock migration needed — Go test files in the same package share test helpers.

## Test performed

- `go test ./pkg/daemon/ -count=1` before and after extraction
- Verify all 1308 lines accounted for across new files

## Conclusion

Five-file extraction based on production code alignment. All tests pass after extraction. daemon_test.go reduced from 1308 to ~142 lines.

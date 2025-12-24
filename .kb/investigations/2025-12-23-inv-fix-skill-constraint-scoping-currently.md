<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented spawn time scoping for skill constraints - constraints now only match files created after the spawn timestamp, fixing the false positive matching from previous spawns.

**Evidence:** All tests pass including new tests for spawn time filtering, backward compatibility (zero time matches all), and integration with VerifyConstraintsForCompletion.

**Knowledge:** File mtime is a reliable proxy for "created during this spawn" and the solution preserves backward compatibility for legacy workspaces without .spawn_time files.

**Next:** Close - implementation complete and tested.

**Confidence:** Very High (95%) - comprehensive test coverage, clean implementation.

---

# Investigation: Fix Skill Constraint Scoping

**Question:** How to prevent skill constraints like '.kb/investigations/{date}-inv-*.md' from matching files created by previous spawns?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** feature-impl spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%+)

---

## Findings

### Finding 1: Current constraint verification uses glob matching without time filtering

**Evidence:** `VerifyConstraints()` in `pkg/verify/constraint.go:118` converts patterns like `{date}` to wildcards and uses `filepath.Glob()` to match files. This matches ANY file that matches the pattern, regardless of when it was created.

**Source:** pkg/verify/constraint.go:147-169

**Significance:** This is the root cause - the glob pattern `*.md` will match investigation files from any previous spawn, causing false positives.

---

### Finding 2: Workspace already has metadata file pattern for tier and session ID

**Evidence:** `pkg/spawn/session.go` already implements atomic write pattern for `.tier` and `.session_id` files in the workspace directory. This pattern can be reused for spawn time.

**Source:** pkg/spawn/session.go:56-103

**Significance:** The existing pattern of writing small metadata files to workspace provides a clean, proven approach for adding spawn time tracking.

---

### Finding 3: Spawn time check via mtime is the simplest and most transparent solution

**Evidence:** Option 1 (spawn_time + mtime check) requires no changes to how agents name files, just filters existing glob matches by file modification time. Options 2 ({beads} in filename) and 3 (tracking created files) would require more invasive changes.

**Source:** SPAWN_CONTEXT.md task description options analysis

**Significance:** The mtime approach is minimally invasive - it adds filtering to existing glob results without changing naming conventions or adding complex tracking.

---

## Synthesis

**Key Insights:**

1. **Spawn time as filtering threshold** - By writing the spawn timestamp when creating the workspace, we can filter glob matches to only include files with mtime >= spawn_time.

2. **Backward compatibility via zero time** - Legacy workspaces without `.spawn_time` files should still work - zero time means no filtering.

3. **Atomic file writes** - Following the existing pattern for `.tier` and `.session_id`, the spawn time file uses atomic write (temp + rename) for consistency.

**Answer to Investigation Question:**

The solution writes a `.spawn_time` file containing the Unix nanosecond timestamp when the workspace is created. During constraint verification, `VerifyConstraintsForCompletion()` reads this timestamp and passes it to `VerifyConstraintsWithSpawnTime()`, which filters glob matches to only include files with mtime >= spawn_time.

---

## Implementation Details

**Files modified:**
- `pkg/spawn/session.go` - Added WriteSpawnTime, ReadSpawnTime, SpawnTimePath functions
- `pkg/spawn/context.go` - Added WriteSpawnTime call in WriteContext
- `pkg/verify/constraint.go` - Added VerifyConstraintsWithSpawnTime, updated VerifyConstraintsForCompletion to use spawn time

**Tests added:**
- `pkg/spawn/session_test.go` - TestWriteReadSpawnTime, TestReadSpawnTime_NoFile, TestReadSpawnTime_InvalidContent, TestSpawnTimePath
- `pkg/verify/constraint_test.go` - TestVerifyConstraintsWithSpawnTime (3 subtests), TestVerifyConstraintsForCompletionWithSpawnTime (2 subtests)

---

## References

**Files Examined:**
- pkg/verify/constraint.go - Core constraint verification logic
- pkg/verify/constraint_test.go - Existing tests to ensure compatibility
- pkg/spawn/session.go - Existing metadata file pattern
- pkg/spawn/context.go - Workspace creation flow

**Commands Run:**
```bash
# Build to verify compilation
go build ./...

# Run all tests
go test ./pkg/spawn/... ./pkg/verify/... -v
```

---

## Investigation History

**2025-12-23:** Investigation started
- Initial question: How to scope constraints to current spawn only
- Context: False positives from matching files created by previous spawns

**2025-12-23:** Implementation complete
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Added spawn time file and mtime filtering to constraint verification

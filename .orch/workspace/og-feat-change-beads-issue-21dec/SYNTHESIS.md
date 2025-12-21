# Synthesis: Change beads issue creation to fail fast

## Summary

Successfully implemented fail-fast behavior for beads issue creation in `orch spawn`. Previously, when `bd create` failed, the system printed a warning and used a fallback ID, creating orphaned agents that couldn't be completed. Now the spawn command fails immediately with a clear error (unless `--no-track` is set).

## What Changed

### Files Modified

- `cmd/orch/main.go`: Added `determineBeadsID()` function and refactored `runSpawnWithSkill()` to use it
- `cmd/orch/main_test.go`: Added comprehensive tests for the new behavior

### Key Changes

1. **New function `determineBeadsID()`** (lines 1063-1084):
   - Encapsulates beads ID determination logic
   - Takes a function parameter for dependency injection (testability)
   - Returns error when beads creation fails (fail-fast)
   - Handles three cases: explicit --issue, --no-track, and default (create new issue)

2. **Refactored `runSpawnWithSkill()`** (lines 694-701):
   - Replaced inline beads ID logic with call to `determineBeadsID()`
   - Removed fallback ID generation (lines 700-701 deleted)
   - Changed warning to error return (fail-fast on beads creation failure)

3. **Added tests** in `cmd/orch/main_test.go`:
   - `TestDetermineBeadsID`: 4 test cases covering all scenarios
   - Tests verify fail-fast behavior when beads creation fails
   - Tests verify --no-track and explicit --issue still work

## Behavior Changes

### Before

```bash
# If bd create failed:
orch spawn feature-impl "task"
# Output: Warning: failed to create beads issue: <error>
# Result: Spawn continues with fallback ID like "orch-go-1734825123"
# Problem: Agent can't be completed (orphaned)
```

### After

```bash
# If bd create fails:
orch spawn feature-impl "task"
# Output: Error: failed to determine beads ID: failed to create beads issue: <error>
# Result: Spawn fails immediately, no orphaned agent created

# If --no-track is set:
orch spawn feature-impl "task" --no-track
# Result: Spawn continues with untracked ID (expected behavior, unchanged)
```

## Testing

All tests passing:

- `TestDetermineBeadsID/explicit_issue_ID_provided` ✓
- `TestDetermineBeadsID/no-track_flag_set` ✓
- `TestDetermineBeadsID/create_beads_issue_succeeds` ✓
- `TestDetermineBeadsID/create_beads_issue_fails_-_should_fail_fast` ✓

Full test suite: `PASS` (72/73 tests passing; 1 pre-existing flaky test in pkg/opencode unrelated to changes)

## Commits

1. `f99876a` - test: add failing test for determineBeadsID fail-fast behavior
2. `5e41420` - feat: implement determineBeadsID with fail-fast behavior
3. `99d43c7` - refactor: use determineBeadsID in runSpawnWithSkill to fail fast on beads creation errors

## Impact

**Positive:**

- No more orphaned agents from beads creation failures
- Clear error messages help users understand what went wrong
- Testable design (dependency injection for createBeadsIssue)
- Preserves existing behavior for --no-track and --issue flags

**Breaking Changes:**
None. The only behavior change is when beads creation fails, which was already a failure scenario (just silently continued before).

## Edge Cases Handled

1. ✅ Explicit issue ID via `--issue`: Uses provided ID, doesn't call createBeadsIssue
2. ✅ `--no-track` flag: Generates local-only ID, doesn't call createBeadsIssue
3. ✅ Beads creation succeeds: Returns created ID
4. ✅ Beads creation fails: Returns error (fail-fast)

## Future Considerations

None. The implementation is complete and handles all required scenarios.

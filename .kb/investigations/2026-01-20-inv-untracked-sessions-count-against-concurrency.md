<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Untracked sessions (--no-track) were incorrectly counted against concurrency limits because checkConcurrencyLimit didn't skip sessions with "-untracked-" in their beadsID, unlike daemon.DefaultActiveCount which did.

**Evidence:** Fixed by adding isUntrackedBeadsID check at spawn_cmd.go:476; build succeeds and all related tests pass.

**Knowledge:** Concurrency checking and active agent counting both use beadsID extraction from session titles; they must apply identical filtering logic (skip empty, skip untracked, skip closed issues).

**Next:** Close - fix implemented and verified.

**Promote to Decision:** recommend-no - tactical bug fix, not architectural decision

---

# Investigation: Untracked Sessions Count Against Concurrency

**Question:** Why do untracked sessions count against concurrency limit when they shouldn't have beads tracking?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: checkConcurrencyLimit missing untracked filter

**Evidence:** The `checkConcurrencyLimit` function in `spawn_cmd.go:469-489` iterates over OpenCode sessions, extracts beadsID from titles, and counts active sessions. However, it did not filter out untracked sessions (beadsIDs like "orch-go-untracked-1766695797").

**Source:** `cmd/orch/spawn_cmd.go:469-489` (before fix)

**Significance:** Untracked sessions would be added to the sessionList and beadsIDs slice, then:
1. `closedIssues[sd.beadsID]` returns false (no beads issue exists for untracked IDs)
2. `verify.IsPhaseComplete(sd.beadsID)` fails/returns false (no beads issue to query)
3. Thus the session counts as active, blocking new spawns

---

### Finding 2: daemon.DefaultActiveCount already handles this correctly

**Evidence:** The daemon's `DefaultActiveCount` function in `pkg/daemon/active_count.go:66` has the correct pattern:
```go
beadsID := extractBeadsIDFromSessionTitle(s.Title)
if beadsID == "" || isUntrackedBeadsID(beadsID) {
    continue
}
```

**Source:** `pkg/daemon/active_count.go:63-68`

**Significance:** The fix pattern already exists in the codebase - just needed to apply it consistently to checkConcurrencyLimit.

---

### Finding 3: isUntrackedBeadsID function already exists

**Evidence:** The `isUntrackedBeadsID` function exists in both:
- `cmd/orch/shared.go:91-95` 
- `pkg/daemon/active_count.go:154-158`

Both implementations check for "-untracked-" substring in the beadsID.

**Source:** `cmd/orch/shared.go:93`, `pkg/daemon/active_count.go:157`

**Significance:** No new code needed to detect untracked IDs - just needed to call the existing function.

---

## Synthesis

**Key Insights:**

1. **Inconsistent filtering** - Two codepaths (checkConcurrencyLimit and DefaultActiveCount) both process OpenCode sessions but had different filtering logic. The daemon was correct; spawn wasn't.

2. **Beads lookup failure mode** - When beadsID doesn't exist in the database (untracked spawns), the "is closed" and "is complete" checks both fail/return false, making the session appear active.

3. **Pattern established** - The daemon's pattern of filtering out untracked beadsIDs early in the loop (before any beads lookups) is the correct approach.

**Answer to Investigation Question:**

Untracked sessions counted against concurrency because `checkConcurrencyLimit` extracted their beadsID (e.g., "orch-go-untracked-123") but didn't filter them out before checking beads status. Since untracked IDs don't exist in beads, they fail the "is closed" and "is complete" checks, making them appear active. The fix adds `isUntrackedBeadsID` check matching the existing pattern in `daemon.DefaultActiveCount`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Fix compiles successfully (verified: go build ./cmd/orch/...)
- ✅ isUntrackedBeadsID function correctly identifies untracked IDs (verified: TestIsUntrackedBeadsID passes)
- ✅ All cmd/orch tests pass (verified: go test ./cmd/orch/...)

**What's untested:**

- ⚠️ End-to-end verification with actual untracked session (requires running orch spawn --no-track and hitting concurrency limit)
- ⚠️ Performance impact of skipping untracked sessions early (trivial - just a string contains check)

**What would change this:**

- Finding would be wrong if untracked sessions SHOULD count against concurrency for some reason
- Finding would be incomplete if there are other codepaths that also incorrectly count untracked sessions

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add isUntrackedBeadsID filter** - Add the same filtering pattern used in daemon.DefaultActiveCount to checkConcurrencyLimit.

**Why this approach:**
- Matches existing pattern in daemon code
- Minimal change (8 lines including comment)
- Uses existing isUntrackedBeadsID function

**Trade-offs accepted:**
- None - this is a pure bug fix with no tradeoffs

**Implementation sequence:**
1. Add check after beadsID extraction
2. Build and verify no compilation errors
3. Run related tests

### Alternative Approaches Considered

**Option B: Filter at the batch lookup level**
- **Pros:** Fewer sessions to process in batch
- **Cons:** Would require passing through untracked IDs to closedIssues batch, then filtering results
- **When to use instead:** If early filtering becomes a bottleneck (unlikely)

**Rationale for recommendation:** Filtering early (before adding to sessionList) is cleaner and matches the daemon pattern.

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go:469-514` - checkConcurrencyLimit function
- `pkg/daemon/active_count.go:19-94` - DefaultActiveCount function (reference pattern)
- `cmd/orch/shared.go:91-95` - isUntrackedBeadsID function

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test verification
go test ./cmd/orch/... -v -run "TestIsUntracked"
go test ./cmd/orch/...
```

**Related Artifacts:**
- **Constraint:** "Untracked spawns (--no-track) generate placeholder beads IDs that fail bd comment commands" - from SPAWN_CONTEXT.md prior knowledge

---

## Investigation History

**2026-01-20 23:50:** Investigation started
- Initial question: Why do untracked sessions count against concurrency limit?
- Context: Bug report indicating concurrency check incorrectly counts --no-track spawns

**2026-01-20 23:55:** Root cause identified
- Found checkConcurrencyLimit missing isUntrackedBeadsID filter
- Confirmed daemon.DefaultActiveCount has correct pattern

**2026-01-20 23:58:** Fix implemented and verified
- Added isUntrackedBeadsID check at spawn_cmd.go:476-483
- All tests pass

**2026-01-21 00:00:** Investigation completed
- Status: Complete
- Key outcome: Bug fixed by adding consistent filtering of untracked sessions

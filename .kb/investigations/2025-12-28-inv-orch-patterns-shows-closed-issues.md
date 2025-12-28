<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Modified `collectRetryPatterns()` in `cmd/orch/patterns.go` to filter out closed/deferred/tombstone issues from pattern output.

**Evidence:** Code compiles successfully, all existing tests pass. Implementation uses existing `verify.GetIssuesBatch()` function for efficient batch status checks.

**Knowledge:** Pattern analyzer reads historical events from events.jsonl without checking current issue status in beads; closed issues appeared as "persistent failures" because retry history persists after resolution.

**Next:** Close - fix is implemented and tested.

---

# Investigation: Orch Patterns Shows Closed Issues

**Question:** Why does `orch patterns` show closed issues like `orch-go-bdd.2` and `orch-go-7p9` as critical failures?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent og-debug-orch-patterns-shows-28dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Pattern analyzer doesn't check issue status

**Evidence:** The `collectRetryPatterns()` function in `cmd/orch/patterns.go:160-203` calls `verify.GetAllRetryPatterns()` which reads from `events.jsonl` to detect retry/failure patterns. It analyzes spawn/abandon/complete event history but never checks if the issue is currently open or closed in beads.

**Source:** 
- `cmd/orch/patterns.go:160-203` - collectRetryPatterns function
- `pkg/verify/attempts.go:156-254` - GetAllRetryPatterns function

**Significance:** Issues that were eventually resolved (closed) still appeared as "persistent failures" because their event history showed multiple abandon events before completion. The fix filters out closed issues.

---

### Finding 2: Existing infrastructure supports batch status checks

**Evidence:** The `pkg/verify/check.go` file already provides `GetIssuesBatch()` function (lines 594-652) that efficiently fetches multiple issue statuses from beads in a single call. This function uses RPC with auto-reconnect and falls back to CLI.

**Source:** 
- `pkg/verify/check.go:594-652` - GetIssuesBatch function
- `pkg/beads/interface.go:14-52` - BeadsClient interface

**Significance:** The fix could leverage existing infrastructure rather than making individual calls per issue, maintaining efficiency.

---

### Finding 3: Status filtering covers multiple closed states

**Evidence:** Beads issues can be in various closed states: "closed", "deferred", "tombstone". The fix filters out all three to prevent flagging any resolved work.

**Source:**
- `pkg/beads/types.go:128` - Issue.Status field
- `pkg/verify/check.go:669-682` - Status filtering in ListOpenIssues

**Significance:** A complete fix needs to check for multiple terminal states, not just "closed".

---

## Synthesis

**Key Insights:**

1. **Data source mismatch** - Pattern analyzer reads historical event log while the source of truth for issue status is beads database. The fix bridges this gap by querying beads before surfacing patterns.

2. **Graceful degradation** - When beads is unavailable, the fix shows all patterns rather than hiding potentially real issues. Better to have false positives than miss real problems.

3. **Efficient implementation** - Using batch fetching avoids N+1 query problem when many issues have retry patterns.

**Answer to Investigation Question:**

`orch patterns` showed closed issues as failures because `collectRetryPatterns()` only analyzed event history without checking current issue status in beads. The fix adds a batch fetch of issue statuses and filters out closed/deferred/tombstone issues before surfacing patterns.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles successfully (verified: `go build ./cmd/orch/...`)
- ✅ All existing pattern tests pass (verified: `go test ./cmd/orch/... -run Pattern -v`)
- ✅ All verify package tests pass (verified: `go test ./pkg/verify/... -run "Pattern|Attempt" -v`)

**What's untested:**

- ⚠️ Integration test with actual closed issues in beads (would need beads db in test environment)
- ⚠️ Performance impact of batch status fetch (expected minimal - single batch call)

**What would change this:**

- If beads List() becomes slow with many issues, might need pagination
- If events.jsonl contains thousands of unique beads IDs, the batch fetch could be expensive

---

## Implementation Recommendations

### Recommended Approach ⭐

**Filter closed issues in collectRetryPatterns** - Add batch status check before surfacing patterns.

**Why this approach:**
- Uses existing `verify.GetIssuesBatch()` for efficiency
- Filters at display time, preserving historical data
- Gracefully handles beads unavailability

**Trade-offs accepted:**
- Extra beads call on every `orch patterns` invocation
- Historical event data preserved (not cleaned up)

**Implementation sequence:**
1. Collect beads IDs from retry stats
2. Batch fetch issue statuses
3. Filter out closed/deferred/tombstone before display

### Alternative Approaches Considered

**Option B: Clean up events.jsonl on issue close**
- **Pros:** No extra beads calls at display time
- **Cons:** Requires hooking into `orch complete`, complex cleanup logic
- **When to use instead:** If events.jsonl grows very large

**Option C: Add status indicator instead of filtering**
- **Pros:** Shows complete history with context
- **Cons:** More noise in output, requires UI changes
- **When to use instead:** If users want to see historical patterns

**Rationale for recommendation:** Filtering is simpler, matches user expectation (closed = resolved), and leverages existing infrastructure.

---

## References

**Files Examined:**
- `cmd/orch/patterns.go` - Main patterns command implementation
- `pkg/verify/attempts.go` - Retry pattern detection from events
- `pkg/verify/check.go` - Beads issue status fetching utilities
- `pkg/beads/types.go` - Issue struct and status field

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test verification
go test ./cmd/orch/... -run Pattern -v
go test ./pkg/verify/... -run "Pattern|Attempt" -v
```

---

## Investigation History

**2025-12-28:** Investigation started
- Initial question: Why does `orch patterns` show closed issues as critical failures?
- Context: User reported `orch-go-bdd.2` and `orch-go-7p9` appearing as "Persistent failure" despite being closed

**2025-12-28:** Root cause identified
- Pattern analyzer reads events.jsonl without checking beads issue status
- `collectRetryPatterns()` identified as fix location

**2025-12-28:** Fix implemented and verified
- Modified `collectRetryPatterns()` to batch-fetch issue statuses
- Filters closed/deferred/tombstone issues from pattern output
- All existing tests pass

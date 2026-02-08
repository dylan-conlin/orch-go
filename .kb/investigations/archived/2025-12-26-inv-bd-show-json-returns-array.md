<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Client.Show() in pkg/beads/client.go expects single object but bd show --json returns array, causing "cannot unmarshal array into Go value of type beads.Issue" error.

**Evidence:** Running `bd show <id> --json` outputs `[{...}]` (array), but Client.Show() was unmarshaling to single Issue struct. FallbackShow already handled arrays correctly.

**Knowledge:** bd show CLI always returns arrays; RPC daemon may return either format. Fix must handle both: try array first, fall back to single object.

**Next:** Close - fix implemented and tested.

---

# Investigation: bd show --json returns array, breaks orch-go parsing

**Question:** Why does the orch daemon fail with "json: cannot unmarshal array into Go value of type beads.Issue" when spawning from beads issues?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Client.Show() unmarshals to single Issue struct

**Evidence:** pkg/beads/client.go:362-363 had:
```go
var issue Issue
if err := json.Unmarshal(resp.Data, &issue); err != nil {
```

**Source:** pkg/beads/client.go:353-368

**Significance:** This causes the error "json: cannot unmarshal array into Go value of type beads.Issue" when the response is `[{...}]` array format.

---

### Finding 2: FallbackShow already handles array format correctly

**Evidence:** FallbackShow at line 650-670 already has:
```go
// Note: bd show --json always returns an array, even for a single issue.
// We unmarshal the array and return the first element.
var issues []Issue
if err := json.Unmarshal(output, &issues); err != nil {
```

**Source:** pkg/beads/client.go:649-670

**Significance:** The CLI fallback was already fixed (commit 1d3de60b), but the RPC client's Show method was not updated.

---

### Finding 3: RPC daemon may return either format

**Evidence:** TestClient_Show_ChildID test uses mock daemon returning single object, while integration tests show actual bd show returns arrays.

**Source:** pkg/beads/client_test.go:681-696 vs bd show orch-go-881b --json output

**Significance:** The fix must handle both formats for compatibility: array format (CLI) and single object format (some RPC daemon versions).

---

## Synthesis

**Key Insights:**

1. **Inconsistent response formats** - bd CLI returns arrays, but RPC daemon mock tests expect single objects. This suggests different beads daemon versions may behave differently.

2. **FallbackShow was fixed, Client.Show wasn't** - The Dec 25 fix (commit 1d3de60b) addressed FallbackShow but missed the RPC client's Show method.

3. **Defensive parsing needed** - The safest approach is to try array format first, then fall back to single object format.

**Answer to Investigation Question:**

The daemon spawning failed because Client.Show() tried to unmarshal the bd show response directly into a single Issue struct. When the beads daemon returned an array `[{...}]`, Go's json.Unmarshal correctly rejected this with "cannot unmarshal array into Go value". The fix is to try parsing as array first (the CLI format), then fall back to single object (for RPC daemon compatibility).

---

## Structured Uncertainty

**What's tested:**

- ✅ Array format parsing (verified: TestClient_Show_ArrayFormat passes with mock returning array)
- ✅ Single object format parsing (verified: TestClient_Show_ChildID passes with mock returning object)
- ✅ Integration tests pass (verified: TestIntegration_ChildID_Show passes with real beads daemon)
- ✅ Build succeeds (verified: go build ./... completes)

**What's untested:**

- ⚠️ Daemon overnight spawning (not tested - would need to run daemon and watch for 24h)

**What would change this:**

- Finding would be wrong if beads daemon returns neither array nor object format (e.g., null, or nested structure)

---

## Implementation Recommendations

**Purpose:** Document the fix applied.

### Recommended Approach (Applied)

**Try array first, fall back to single object**

**Why this approach:**
- Handles CLI format (always array)
- Handles RPC daemon format (may be single object)
- Backward compatible with existing tests

**Trade-offs accepted:**
- Slight overhead of trying two unmarshal attempts (negligible)

**Implementation sequence:**
1. Try unmarshal to []Issue (array format)
2. If successful and non-empty, return first element
3. If array parsing fails, try unmarshal to Issue (single object)
4. Return appropriate error if both fail

---

## References

**Files Examined:**
- pkg/beads/client.go - Client.Show() method (line 353-375)
- pkg/beads/client.go - FallbackShow() method (line 649-670) 
- pkg/beads/client_test.go - TestClient_Show_ChildID, TestClient_Show_ArrayFormat

**Commands Run:**
```bash
# Verify bd show output format
bd show orch-go-881b --json

# Run tests
go test ./pkg/beads/... -v
```

**Related Artifacts:**
- **Decision:** Commit 1d3de60b - fix: handle bd show array response for epic children
- **Investigation:** .kb/investigations/2025-12-25-inv-bd-show-returns-array-epic.md

---

## Investigation History

**2025-12-26 22:10:** Investigation started
- Initial question: Why does orch daemon fail with array unmarshal error?
- Context: Daemon spawning broken from 20:15-20:39 due to beads parsing error

**2025-12-26 22:15:** Root cause identified
- Client.Show() uses single Issue struct, but bd show --json returns array
- FallbackShow already fixed, but RPC client's Show wasn't

**2025-12-26 22:25:** Fix implemented and tested
- Updated Client.Show() to try array first, fall back to single object
- Added TestClient_Show_ArrayFormat test
- All tests pass

**2025-12-26 22:30:** Investigation completed
- Status: Complete
- Key outcome: Fixed Client.Show() to handle both array and single object response formats

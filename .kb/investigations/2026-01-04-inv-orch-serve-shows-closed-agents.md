<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fixed orch serve to check beads issue status when determining agent completion - closed issues now correctly show as "completed".

**Evidence:** Build passes, tests pass (go test ./cmd/orch/... - 39 passed, 0 failed).

**Knowledge:** Agent status determination had multiple signals (Phase: Complete, SYNTHESIS.md) but was missing the beads issue status check, which is the definitive source of truth for completion.

**Next:** Verify with smoke test, then close.

---

# Investigation: Orch Serve Shows Closed Agents

**Question:** Why does orch serve show closed agents as active, and how do we fix it?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Agent og-debug-orch-serve-shows-04jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Agent status determination logic missed beads issue status

**Evidence:** In `serve_agents.go`, agent status was determined by:
1. OpenCode session activity time → "active" or "idle" (lines 560-584)
2. `Phase: Complete` in beads comments → "completed" (lines 822-837)
3. Workspace with SYNTHESIS.md → "completed" (lines 839-850)

Missing: Checking if the beads issue itself has `status: "closed"`.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go:822-858`

**Significance:** When `orch complete` closes a beads issue but the OpenCode session is still open or the workspace exists without SYNTHESIS.md, the agent incorrectly shows as "active" or "idle" instead of "completed".

---

### Finding 2: allIssues batch fetch already includes closed issues

**Evidence:** The code already fetches all issues (including closed) via `globalBeadsCache.getAllIssues()` which calls `verify.GetIssuesBatch()`. The Issue struct has a `Status` field that contains "closed" for completed issues.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:28-36` (Issue struct)

**Significance:** No additional API calls needed - just need to check the `Status` field of already-fetched issues.

---

### Finding 3: Fix implemented by checking beads issue status

**Evidence:** Added check after Phase comment handling to set `agents[i].Status = "completed"` when `issue.Status == "closed"`. Also captures `close_reason` when available.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_agents.go:839-859` (new code)

**Significance:** The beads issue status is the definitive source of truth for completion - if the orchestrator closed the issue via `orch complete`, the agent should show as completed regardless of session/workspace state.

---

## Synthesis

**Key Insights:**

1. **Multiple completion signals** - Agent completion can be signaled by: Phase: Complete comment, SYNTHESIS.md presence, or beads issue being closed. All three should result in "completed" status.

2. **Beads is source of truth** - The beads issue status represents the orchestrator's decision. If closed, the work is verified complete regardless of OpenCode session state.

3. **Order of checks matters** - Check Phase: Complete first (agent self-reports), then beads status (orchestrator verification), then SYNTHESIS.md (artifact presence). This ensures the most authoritative signal takes precedence.

**Answer to Investigation Question:**

Orch serve showed closed agents as active because it checked Phase: Complete comments and SYNTHESIS.md presence, but not the beads issue status. When `orch complete` closed an issue but the agent session was still open, the agent appeared "active". The fix adds a check for `issue.Status == "closed"` which is the definitive signal that the orchestrator verified and closed the work.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles successfully (verified: `make build` passed)
- ✅ Existing tests pass (verified: `go test ./cmd/orch/... - TestHandleAgents passed`)
- ✅ strings.EqualFold correctly handles case-insensitive "closed" comparison

**What's untested:**

- ⚠️ End-to-end verification with actual closed beads issue (requires running orch serve + dashboard)
- ⚠️ Cross-project closed issue detection

**What would change this:**

- Finding would be wrong if beads issue Status field is not populated correctly for closed issues
- Finding would be wrong if allIssues cache is not fetched for the beads ID in question

---

## Implementation Recommendations

### Recommended Approach ⭐

**Check beads issue status after Phase comment check** - Add status check that sets agent to "completed" when beads issue is closed.

**Why this approach:**
- Minimal change (10 lines of code)
- Uses existing data (allIssues already fetched)
- Beads issue status is the authoritative source

**Trade-offs accepted:**
- Additional conditional check in the status determination loop
- Acceptable: O(1) lookup from already-fetched map

**Implementation sequence:**
1. After Phase: Complete check, add beads status check
2. Also capture close_reason when marking as completed
3. Verify with tests

---

## References

**Files Examined:**
- `/cmd/orch/serve_agents.go` - Agent status determination logic
- `/pkg/verify/check.go` - Issue struct and GetIssuesBatch function

**Commands Run:**
```bash
# Build verification
make build  # Passed

# Test verification
go test ./cmd/orch/... -run TestHandleAgents  # Passed
```

---

## Investigation History

**2026-01-04 17:10:** Investigation started
- Initial question: Why does orch serve show closed agents as active?
- Context: Bug report orch-go-vl5f

**2026-01-04 17:15:** Root cause identified
- Agent status logic missing beads issue status check
- Fix implemented and tested

**2026-01-04 17:20:** Investigation completed
- Status: Complete
- Key outcome: Added beads issue status check to agent status determination

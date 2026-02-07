<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** FallbackList() was missing `--limit 0`, causing `bd list --json` to return only the 50 most recent issues (which were all closed), missing the 4 in_progress issues.

**Evidence:** `bd list --json | jq '. | length'` returns 50 issues (all closed). `bd list --json --limit 0 | jq '. | length'` returns 283 issues including the 4 in_progress.

**Knowledge:** bd CLI defaults to limit 50 for list operations. When a repo has many closed issues, in_progress issues can fall outside this limit and become invisible.

**Next:** Add `--limit 0` to FallbackList() in pkg/beads/client.go

**Promote to Decision:** recommend-no (simple bug fix, not architectural)

---

# Investigation: Server Recovery Orphan Detection Not Finding In-Progress Issues

**Question:** Why does FindOrphanedSessions return 0 issues when bd list --status=in_progress shows 4 issues?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** Agent spawned by orchestrator
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

---

## Findings

### Finding 1: verify.ListOpenIssues() uses FallbackList() in daemon context

**Evidence:** 
- The orch daemon runs with `BEADS_NO_DAEMON=1` set (see launchd plist)
- No beads daemon is running, so no bd.sock exists
- beads.FindSocketPath() fails, triggering CLI fallback
- FallbackList("") is called which runs `bd list --json`

**Source:** 
- pkg/verify/beads_api.go:456-512
- pkg/beads/client.go:812-830
- ~/Library/LaunchAgents/com.orch.daemon.plist

**Significance:** The RPC path is correctly bypassed; the bug is in the CLI fallback path.

---

### Finding 2: bd list defaults to 50 most recent issues

**Evidence:**
```bash
$ bd list --json | jq '. | length'
50

$ bd list --json | jq '[.[].status] | unique'
["closed"]

$ bd list --json --limit 0 | jq '. | length'  
283

$ bd list --json --limit 0 | jq '[.[] | select(.status == "in_progress")] | length'
4
```

**Source:** Direct shell commands in orch-go directory

**Significance:** The 4 in_progress issues exist but are beyond the default limit of 50, so they're not returned.

---

### Finding 3: FallbackList() doesn't use --limit 0

**Evidence:** 
```go
func FallbackList(status string) ([]Issue, error) {
    args := []string{"list", "--json"}
    if status != "" {
        args = append(args, "--status", status)
    }
    // Missing: --limit 0
```

**Source:** pkg/beads/client.go:812-830

**Significance:** Without `--limit 0`, bd list returns only 50 issues, and if all 50 happen to be closed (recent history), the in_progress issues are invisible.

---

## Synthesis

**Key Insights:**

1. **Default limits are dangerous for recovery operations** - The bd CLI's default limit of 50 is reasonable for interactive use but breaks automated systems that need to scan all issues.

2. **Other FallbackList* functions already use --limit 0** - FallbackListByParent() correctly uses `--limit 0`, showing awareness of this pattern elsewhere in the codebase.

3. **RPC path likely doesn't have this bug** - The RPC client.List(nil) call passes nil for ListArgs, which beads handles as "no limit". The bug is specific to CLI fallback.

**Answer to Investigation Question:**

FindOrphanedSessions returns 0 issues because verify.ListOpenIssues() falls back to beads.FallbackList("") which runs `bd list --json` without `--limit 0`. This returns only the 50 most recent issues, and when all 50 are closed (due to a large number of recent closes), the 4 in_progress issues fall outside the limit and are not returned.

---

## Structured Uncertainty

**What's tested:**

- ✅ bd list --json returns 50 issues (verified: ran command)
- ✅ bd list --json --limit 0 returns 283 issues including 4 in_progress (verified: ran command)
- ✅ No bd.sock exists, so RPC path fails (verified: find command returned empty)
- ✅ BEADS_NO_DAEMON=1 is set in daemon env (verified: read plist)

**What's untested:**

- ⚠️ RPC path (client.List) handles limits correctly (assumed based on nil arg passing)
- ⚠️ Performance impact of --limit 0 on large issue databases (not benchmarked)

**What would change this:**

- Finding would be wrong if bd list without --limit returned in_progress issues
- Solution might not work if beads daemon requires special handling

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add --limit 0 to FallbackList()** - Simple one-line fix to include --limit 0 in the bd list command.

**Why this approach:**
- Directly fixes the root cause
- Minimal code change
- Matches pattern used in FallbackListByParent()

**Trade-offs accepted:**
- Slightly larger response for repos with many issues
- Acceptable because recovery needs complete view

**Implementation sequence:**
1. Add "--limit", "0" to args slice in FallbackList()
2. Verify fix by rebuilding and testing daemon

### Alternative Approaches Considered

**Option B: Query specific statuses**
- **Pros:** Smaller response, more targeted
- **Cons:** Requires multiple CLI calls (open, in_progress, blocked), more complex
- **When to use instead:** If performance becomes an issue with very large repos

**Option C: Fix verify.ListOpenIssues to pass status filter to fallback**
- **Pros:** More efficient querying
- **Cons:** Larger change, FallbackList signature change
- **When to use instead:** If this pattern causes issues elsewhere

**Rationale for recommendation:** The simplest fix that matches existing patterns in the codebase.

---

### Implementation Details

**What to implement first:**
- Add `"--limit", "0"` to FallbackList() args

**Things to watch out for:**
- ⚠️ Ensure other callers of FallbackList don't rely on the limit behavior

**Success criteria:**
- ✅ Daemon logs show "found N open issues" where N > 0 when in_progress issues exist
- ✅ Server recovery successfully detects orphaned sessions

---

## References

**Files Examined:**
- pkg/verify/beads_api.go - verify.ListOpenIssues() implementation
- pkg/beads/client.go - FallbackList() implementation
- pkg/daemon/recovery.go - FindOrphanedSessions() caller
- ~/Library/LaunchAgents/com.orch.daemon.plist - daemon config

**Commands Run:**
```bash
# Count issues with default limit
bd list --json | jq '. | length'
# Result: 50

# Count issues without limit
bd list --json --limit 0 | jq '. | length'
# Result: 283

# Count in_progress with limit 0
bd list --json --limit 0 | jq '[.[] | select(.status == "in_progress")] | length'
# Result: 4
```

---

## Investigation History

**2026-01-26 21:14:** Investigation started
- Initial question: Why does FindOrphanedSessions return 0 when bd list shows in_progress issues?
- Context: Server recovery not resuming orphaned agents

**2026-01-26 21:25:** Root cause identified
- FallbackList() missing --limit 0
- bd defaults to 50 most recent issues

**2026-01-26 21:30:** Investigation completed
- Status: Complete
- Key outcome: Simple fix - add --limit 0 to FallbackList()

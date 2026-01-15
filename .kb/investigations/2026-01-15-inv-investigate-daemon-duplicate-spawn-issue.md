---
linked_issues:
  - orch-go-0q9s7
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SpawnedIssueTracker TTL (5 min) causes duplicates when beads status update fails or is delayed; `ReconcileWithIssues()` exists but isn't called in production.

**Evidence:** Code analysis shows `CleanStale()` is called (TTL-based cleanup), but `ReconcileWithIssues()` (status-based cleanup) is only in tests. `bd ready` returns BOTH "open" and "in_progress" issues, relying on explicit status check that misses issues where status update failed.

**Knowledge:** The SpawnedIssueTracker is a temporal workaround for the race between spawn and status update; the real fix needs status-based reconciliation or dedup at spawn time.

**Next:** Implement one of three fixes: (A) call ReconcileWithIssues with actual beads status, (B) extend TTL to match agent work duration (~6h), (C) add session-level dedup by checking existing OpenCode sessions.

**Promote to Decision:** recommend-yes (architectural pattern: spawn tracking needs status-based reconciliation, not just TTL)

---

# Investigation: Daemon Duplicate Spawn Issue

**Question:** Why does the daemon spawn duplicate agents when the SpawnedIssueTracker TTL is 5 minutes but agent work takes hours?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** og-inv agent
**Phase:** Complete
**Next Step:** None - implement session dedup (orch-go-2nruy)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: SpawnedIssueTracker uses 5-minute TTL, not status-based cleanup

**Evidence:**
```go
// pkg/daemon/spawn_tracker.go:33-38
func NewSpawnedIssueTracker() *SpawnedIssueTracker {
    return &SpawnedIssueTracker{
        spawned: make(map[string]time.Time),
        TTL:     5 * time.Minute,  // Hardcoded 5 minutes
    }
}
```

The only cleanup method called in production is `CleanStale()` which removes entries older than TTL. The `ReconcileWithIssues()` method exists but is only used in tests.

**Source:**
- `pkg/daemon/spawn_tracker.go:33-38` (TTL definition)
- `pkg/daemon/spawn_tracker.go:86-100` (CleanStale implementation)
- `pkg/daemon/spawn_tracker.go:102-127` (ReconcileWithIssues - unused in prod)
- `pkg/daemon/daemon.go:496-500` (only CleanStale called)

**Significance:** After 5 minutes, regardless of whether the agent is still running or the beads status was updated, the entry is removed from the tracker. This creates a window for duplicates.

---

### Finding 2: `bd ready` returns BOTH open AND in_progress issues

**Evidence:**
```
$ bd ready --help
Show ready work (issues with no blockers that are open or in_progress).
```

The daemon relies on explicit status filtering in `NextIssueExcluding()` at line 293-298:
```go
if issue.Status == "in_progress" {
    continue
}
```

**Source:**
- `bd ready --help` output
- `pkg/daemon/daemon.go:293-298` (status check)
- `pkg/daemon/issue_adapter.go:13-36` (ListReadyIssues uses bd ready)

**Significance:** If the status update to "in_progress" fails (which is non-fatal in spawn), the issue will continue appearing in `bd ready` results. The SpawnedIssueTracker is the ONLY protection during the first 5 minutes.

---

### Finding 3: Status update failure is non-fatal in spawn flow

**Evidence:**
```go
// cmd/orch/spawn_cmd.go:980-986
if !spawnNoTrack && !skipBeadsForOrchestrator && spawnIssue != "" {
    if err := verify.UpdateIssueStatus(beadsID, "in_progress"); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to update beads issue status: %v\n", err)
        // Continue anyway
    }
}
```

The spawn continues even if status update fails. This means:
1. Agent starts working
2. But beads status remains "open"
3. SpawnedIssueTracker protects for 5 minutes
4. After TTL expires, issue appears spawnable again

**Source:** `cmd/orch/spawn_cmd.go:980-986`

**Significance:** This is the primary cause of duplicates. The 5-minute TTL was designed to cover the race between spawn and status update, but it assumes status update succeeds. When it fails, duplicates occur after TTL expires.

### Finding 4: Concrete evidence - 19 duplicate workspaces for single issue

**Evidence:**
```bash
$ ls .orch/workspace/ | grep -c cross-project
19

$ bd show orch-go-nqgjr | grep "Comments"
Comments (80):
```

Many comments show the duplicate spawns:
- 18:03: "This is duplicate spawn #8+ on same completed issue"
- 18:15: "8th agent spawn for completed work. Feature fully implemented..."
- Multiple agents reporting Phase: Planning ~5-6 minutes apart (matching TTL exactly)

Close reason explicitly states: "Multiple duplicate spawns occurred due to issue staying open."

**Source:**
- `.orch/workspace/` directory listing
- `bd show orch-go-nqgjr` output

**Significance:** Confirms the bug in production. The 5-minute spawn intervals match the TTL duration exactly, proving the TTL expiration triggers respawns.

---

## Synthesis

**Key Insights:**

1. **TTL is the wrong protection mechanism** - The 5-minute TTL was designed to cover the race between spawn and status update, assuming status update succeeds. But when status update fails or is delayed, the TTL provides only 5 minutes of protection while agents work for hours.

2. **ReconcileWithIssues exists but isn't used** - There's already a method to clean up tracker entries based on actual beads status, but it's only called in tests. Adding one call to production code could prevent most duplicates.

3. **Multiple defense layers needed** - The current architecture relies on: (1) status update succeeding, (2) TTL protection during race window, (3) status check filtering in_progress issues. When (1) fails, (2) expires, and then (3) doesn't help because status is still "open".

**Answer to Investigation Question:**

Duplicates occur because the SpawnedIssueTracker's 5-minute TTL expires while agents are still working, AND the beads status check doesn't help when the status update failed or is still "open". The TTL was designed for a short race window, not as the primary protection for hours-long agent work. The fix requires either extending TTL, using status-based reconciliation, or adding session-level deduplication.

---

## Structured Uncertainty

**What's tested:**

- ✅ TTL is 5 minutes (verified: read spawn_tracker.go:37)
- ✅ ReconcileWithIssues not called in production (verified: grep found only test usage)
- ✅ `bd ready` returns in_progress issues (verified: `bd ready --help` output)
- ✅ Status update failure is non-fatal (verified: spawn_cmd.go:980-986)
- ✅ 19 duplicate workspaces exist for single issue (verified: `ls .orch/workspace/`)

**What's untested:**

- ⚠️ Whether status update is actually failing (no logs captured from failing spawns)
- ⚠️ Whether extending TTL alone would fix all duplicate scenarios
- ⚠️ Performance impact of calling ReconcileWithIssues each poll cycle

**What would change this:**

- Finding would be wrong if status updates succeed consistently and duplicates still occur
- Finding would be incomplete if there's another code path causing duplicates

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Session-level deduplication + extended TTL** - Check for existing OpenCode session with same beads ID before spawn, and extend TTL as backup.

**Why this approach:**
- Directly addresses the root cause: no dedup check before spawn
- Uses source of truth (OpenCode sessions) rather than cached status
- Already a related issue exists: orch-go-2nruy ("Add dedup check before spawn")

**Trade-offs accepted:**
- Adds API call before each spawn (latency)
- Requires OpenCode server to be running

**Implementation sequence:**
1. **Add session dedup check** - Query OpenCode for sessions with matching beads ID before spawn
2. **Extend TTL to 6 hours** - Backup protection when dedup check fails
3. **Call ReconcileWithIssues** - Additional defense layer using actual beads status

### Alternative Approaches Considered

**Option B: Extend TTL only**
- **Pros:** Simple one-line change, no API calls
- **Cons:** Doesn't solve root cause, just extends window. Duplicates still possible after extended TTL.
- **When to use instead:** As quick hotfix while implementing proper dedup

**Option C: Call ReconcileWithIssues**
- **Pros:** Uses existing code, status-based cleanup
- **Cons:** Relies on beads status being accurate. If status update fails, reconciliation won't help.
- **When to use instead:** If OpenCode API is unreliable

**Rationale for recommendation:** Session-level dedup is the cleanest solution because OpenCode sessions ARE the source of truth for running agents. Extended TTL and ReconcileWithIssues are defense-in-depth layers.

---

### Implementation Details

**What to implement first:**
- Session dedup check in `daemon.Once()` or `runSpawnWithSkillInternal()`
- The check: query OpenCode for sessions, filter by beads ID in description
- If recent session exists (< 30min old), skip spawn

**Things to watch out for:**
- ⚠️ OpenCode server might be down - need fallback behavior
- ⚠️ Session description format varies - need robust beads ID extraction
- ⚠️ Old/stale sessions shouldn't block respawn - use activity time threshold

**Areas needing further investigation:**
- Why does status update fail? (transient beads daemon issues? disk?)
- Should status update failure abort spawn?

**Success criteria:**
- ✅ Daemon doesn't spawn duplicate sessions for same beads ID
- ✅ Metrics: No more than 1 active workspace per beads issue
- ✅ Test: Simulate status update failure, verify no duplicate after TTL

---

## References

**Files Examined:**
- `pkg/daemon/spawn_tracker.go` - TTL mechanism, unused ReconcileWithIssues
- `pkg/daemon/daemon.go` - Only CleanStale called, not ReconcileWithIssues
- `cmd/orch/spawn_cmd.go:980-986` - Non-fatal status update
- `pkg/daemon/issue_adapter.go` - ListReadyIssues uses bd ready

**Commands Run:**
```bash
# Check bd ready behavior
bd ready --help

# Count duplicate workspaces
ls .orch/workspace/ | grep -c cross-project

# View issue with duplicates
bd show orch-go-nqgjr
```

**External Documentation:**
- None

**Related Artifacts:**
- **Issue:** orch-go-2nruy - "Add dedup check before spawn to prevent duplicate sessions"
- **Issue:** orch-go-nqgjr - Evidence of 19 duplicate spawns for single issue
- **Workspace:** `.orch/workspace/og-feat-support-cross-project-15jan-*` (19 duplicates)

---

## Test Performed

**Test:** Analyzed real-world duplication of orch-go-nqgjr issue

**Method:**
1. Ran `ls .orch/workspace/ | grep cross-project` to count duplicate workspaces
2. Ran `bd show orch-go-nqgjr` to see spawn history in comments
3. Analyzed timestamps in comments to verify ~5-min spawn intervals

**Result:**
- 19 duplicate workspaces created for single issue
- 80 comments showing repeated Phase: Planning messages
- Spawn timestamps align with 5-minute TTL intervals
- Close reason explicitly confirms: "Multiple duplicate spawns occurred due to issue staying open"

---

## Conclusion

The daemon duplicate spawn issue occurs because:

1. **SpawnedIssueTracker TTL (5 min) is too short** - Agent work takes hours, but protection expires in minutes
2. **Status update can fail silently** - Spawn continues even if beads status isn't updated to "in_progress"
3. **ReconcileWithIssues isn't used** - The method exists to clean up tracker based on actual status, but isn't called

**Root cause:** The system relies on beads status update succeeding to prevent duplicates after TTL expires. When it fails, there's no secondary protection.

**Fix:** Add session-level deduplication by checking OpenCode for existing sessions with the same beads ID before spawning. Extend TTL as backup.

---

## Investigation History

**2026-01-15 13:45:** Investigation started
- Initial question: Why does daemon spawn duplicates after 5-min TTL?
- Context: orch-go-nqgjr had 12+ duplicate sessions

**2026-01-15 14:00:** Found TTL mechanism and unused ReconcileWithIssues
- Discovered CleanStale is called but ReconcileWithIssues is not
- Identified that bd ready returns both open and in_progress issues

**2026-01-15 14:15:** Verified with production evidence
- Found 19 duplicate workspaces for cross-project issue
- Confirmed spawn intervals match TTL duration

**2026-01-15 14:30:** Investigation completed
- Status: Complete
- Key outcome: TTL-based protection inadequate; need session-level dedup

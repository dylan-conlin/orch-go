# Session Synthesis

**Agent:** og-arch-fix-daemon-duplicate-14feb-fa4c
**Issue:** orch-go-2ma
**Duration:** 2026-02-15 06:30 → 2026-02-15 06:45
**Outcome:** success

---

## TLDR

Fixed daemon duplicate spawn bug by changing UpdateBeadsStatus failure handling from silent-continue to fail-fast. The daemon was spawning 10 duplicates for the same issue because UpdateBeadsStatus failures were logged as warnings but spawning continued anyway, relying on fragile in-memory TTL-based tracking.

---

## Delta (What Changed)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-feb14-incident.md` - Probe documenting Feb 14 incident and root cause

### Files Modified
- `pkg/daemon/daemon.go:849-876` (Once function) - Changed UpdateBeadsStatus failure handling to fail-fast with slot release
- `pkg/daemon/daemon.go:977-1004` (OnceWithSlot function) - Changed UpdateBeadsStatus failure handling to fail-fast with slot release

### Commits
- (pending) - Fix daemon duplicate spawn by failing fast when UpdateBeadsStatus fails

---

## Evidence (What Was Observed)

### Root Cause Discovery

1. **Prior fix was deployed but incomplete** - `pkg/daemon/daemon.go:856` shows UpdateBeadsStatus IS called before spawning, but failure handling was wrong:
   ```go
   if err := UpdateBeadsStatus(issue.ID, "in_progress"); err != nil {
       // Warning logged, but spawn CONTINUES
       // Continue with spawn - SpawnedIssueTracker provides secondary protection
   }
   ```

2. **Incident pattern matches failure mode** - Feb 14 incident: "Issue remained OPEN status with triage:ready label" indicates UpdateBeadsStatus was failing repeatedly, each spawn left issue status unchanged

3. **Defense-in-depth failed when primary failed** - SpawnedIssueTracker (6h TTL, in-memory) cannot compensate for persistent tracking failure, especially with daemon restarts

### Tests Run

```bash
$ go test ./pkg/daemon/... -run "TestDaemon_Once|TestDaemon_OnceWithSlot|TestNextIssue|TestSpawnedIssue" -v
=== All spawn-related tests PASSED (2.049s)

$ make build
Building orch...
go build -ldflags "..." -o build/orch ./cmd/orch/
[SUCCESS]
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-feb14-incident.md` - Documents that the prior probe's fix was incomplete because it didn't account for UpdateBeadsStatus failures

### Decisions Made
- **Fail-fast on UpdateBeadsStatus failure** - Better to skip one spawn attempt (daemon will retry next poll) than to spawn without persistent tracking (creates duplicates that waste API budget)

### Constraints Discovered
- **Silent failure is dangerous for deduplication** - When the primary dedup mechanism (beads status update) fails, falling back to secondary mechanisms (in-memory TTL) creates a false sense of protection that breaks under realistic failure modes (daemon restarts, TTL expiration)

### Model Impact

**CONTRADICTS** the prior probe's claim that "defense in depth" makes duplicate spawns safe:

> From 2026-02-14-daemon-duplicate-spawn-ttl-fragility.md:
> "The beads database now reflects reality immediately, making the TTL cache a secondary protection layer instead of the primary one."

This is only true when UpdateBeadsStatus SUCCEEDS. The prior probe didn't account for UpdateBeadsStatus failing, which is exactly what happened on Feb 14 2026.

**Updated understanding:**
- Defense in depth only works if failures are handled correctly
- Failing silently on primary protection and relying on secondary is not defense in depth - it's single point of failure with extra steps
- Correct pattern: FAIL FAST when primary protection fails, retry on next poll

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (code fix implemented in both functions)
- [x] Tests passing (all spawn-related tests pass)
- [x] Probe file has `Status: Complete`
- [x] Ready for `orch complete orch-go-2ma`

### Follow-up Questions (Out of Scope)

**Why was UpdateBeadsStatus failing on Feb 14?**
- Need to check daemon logs from Feb 14 to find actual failure messages
- Possible causes: beads daemon down, database lock contention, filesystem issues, permission problems
- This is a separate investigation - the fix prevents damage regardless of WHY it fails

**Should we add instrumentation?**
- Track UpdateBeadsStatus failure rate to detect systematic issues
- Alert when failures spike (indicates beads infrastructure problem)
- Out of scope for this bug fix, but valuable for operational visibility

---

## Unexplored Questions

**Questions that emerged during this session:**

1. **What was the actual UpdateBeadsStatus failure on Feb 14?** - Daemon logs would show the error, but this session focused on fixing the HANDLING of failure, not diagnosing the specific failure cause

2. **Are there other silent failures in the daemon?** - This pattern (log warning, continue anyway) might exist elsewhere. Worth auditing for similar cases where failure should halt operation

3. **Should UpdateBeadsStatus retry internally?** - Currently tries RPC then CLI fallback. Could add exponential backoff retry before returning error. But fail-fast is better than masking infrastructure issues.

**Areas worth exploring further:**
- Daemon operational visibility: dashboards showing UpdateBeadsStatus success/failure rates
- Beads RPC client reliability: connection pooling, circuit breaker pattern
- Reconciliation improvements: faster detection of status update failures via polling

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5
**Workspace:** `.orch/workspace/og-arch-fix-daemon-duplicate-14feb-fa4c/`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-feb14-incident.md`
**Beads:** `bd show orch-go-2ma`

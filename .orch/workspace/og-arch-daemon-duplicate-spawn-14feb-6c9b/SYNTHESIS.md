# Session Synthesis

**Agent:** og-arch-daemon-duplicate-spawn-14feb-6c9b
**Issue:** orch-go-07i
**Duration:** 2026-02-14T20:48:00Z → 2026-02-14T20:56:00Z
**Outcome:** success

---

## TLDR

Fixed daemon duplicate spawn issue by updating beads issue status to `in_progress` immediately before spawning, eliminating reliance on fragile TTL-based in-memory cache. Implements defense-in-depth with three layers: persistent beads status (primary), in-memory tracker (secondary), and session-level check (tertiary).

---

## Delta (What Changed)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-ttl-fragility.md` - Probe documenting the TTL fragility issue and defense-in-depth fix

### Files Modified
- `pkg/daemon/daemon.go` - Added immediate beads status update in `Once()` and `OnceWithSlot()` before spawning
- `pkg/daemon/issue_adapter.go` - Added `UpdateBeadsStatus()` function for RPC-first beads status updates

### Commits
- [pending] - Fix daemon duplicate spawn by updating beads status before spawn

---

## Evidence (What Was Observed)

### Code Flow Analysis

**Current Race Window (BEFORE fix):**
```
Daemon.Once() 
  ↓
1. Mark in memory: d.SpawnedIssues.MarkSpawned(issue.ID)
  ↓
2. Spawn subprocess: d.spawnFunc(issue.ID)  
  ↓  
3. SpawnWork runs: exec.Command("orch", "work", beadsID)
  ↓
4. LATER: verify.UpdateIssueStatus(beadsID, "in_progress")  [spawn_cmd.go:696]
```

**Problem:** Between steps 2-4, beads database still shows issue as `open` with `triage:ready` label. If daemon polls during this window AND in-memory tracker fails (TTL expired, daemon restarted), duplicate spawn occurs.

### SpawnedIssueTracker Fragility

**File:** `pkg/daemon/spawn_tracker.go:41`
```go
func NewSpawnedIssueTracker() *SpawnedIssueTracker {
    return &SpawnedIssueTracker{
        spawned: make(map[string]time.Time),
        TTL:     6 * time.Hour,  // ⚠️ TTL-based, not persistent
    }
}
```

**Failure modes:**
1. TTL expiration - agents running >6h allow re-spawn
2. Daemon restarts - in-memory map lost
3. CleanStale() calls - premature entry removal

### Evidence from Recent Incident

Spawn context stated:
> "Three agents spawned for the same ci-implement-role work because the issue remained triage:ready after the first spawn. The daemon's dedup TTL cache expired between spawns."

This confirms:
- Issue stayed `triage:ready` (beads database not updated)
- TTL cache expired (tracker lost spawn record)
- Daemon re-spawned same issue

### Tests Run

```bash
$ go test ./pkg/daemon/... -v -run TestDaemon
=== All 51 tests PASSED (3.297s)

Key validating tests:
- TestDaemon_SkipsRecentlySpawnedIssues
- TestDaemon_OnceMarksSpawned
- TestDaemon_OnceUnmarksOnFailure
- TestDaemon_PreventsDuplicateSpawns
```

```bash
$ make build
Building orch...
go build -ldflags "..." -o build/orch ./cmd/orch/
[SUCCESS]
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-ttl-fragility.md` - Documents defense-in-depth duplicate prevention architecture

### Decisions Made

**Decision 1:** Update beads status BEFORE spawning (not after)
- **Rationale:** Makes beads database (source of truth) immediately reflect reality, eliminating race window
- **Trade-off:** Requires rollback on spawn failure (acceptable - handled in code)
- **Alternative rejected:** Increase TTL → doesn't solve root cause, just delays it

**Decision 2:** Keep all three dedup layers (defense-in-depth)
- **Rationale:** Each layer has different failure modes; multiple layers provide fault tolerance
- **Primary:** Beads status (persistent, survives restarts)
- **Secondary:** SpawnedIssueTracker (catches race window during subprocess spawn)
- **Tertiary:** Session-level check (detects when status update failed silently)

**Decision 3:** Fail gracefully on beads status update errors
- **Rationale:** If RPC/CLI fails, still spawn with tracker protection (secondary layer)
- **Implementation:** Log warning, continue with spawn

### Constraints Discovered

**Constraint:** Beads status updates must be synchronous before spawn
- **Why it matters:** Async updates create race window where daemon can re-spawn before status reflects reality
- **Source:** daemon.go:841-870

**Constraint:** Spawn failures must rollback beads status
- **Why it matters:** Without rollback, failed spawns leave issues in `in_progress` state forever
- **Source:** daemon.go:858-864

### Model Impact

**EXTENDS** the Daemon Autonomous Operation model's "Duplicate Spawns" section.

**Before:**
> **Fix:** Spawn deduplication via tracking. Track spawned beads IDs in memory, skip on subsequent polls until status confirms transition.

**After:**
> **Fix:** Defense-in-depth with three layers:
> 1. PRIMARY: Update beads status to `in_progress` before spawning (persistent, survives restarts)
> 2. SECONDARY: Track spawned IDs in memory via SpawnedIssueTracker (catches race window)
> 3. TERTIARY: Check OpenCode sessions before spawning (detects silent status update failures)

---

## Next (What Should Happen)

**Recommendation:** close

### Completion Checklist
- [x] All deliverables complete
  - [x] Daemon immediately updates beads status before spawn
  - [x] Rollback on spawn failure implemented
  - [x] Defense-in-depth architecture documented
- [x] Tests passing (51/51 tests pass)
- [x] Probe file created with `**Status:** Complete`
- [x] Build successful
- [x] Ready for `orch complete orch-go-07i`

### Verification Contract

**Bug reproduction prevented:**
1. Issue labeled `triage:ready`
2. Daemon spawns agent → beads status immediately = `in_progress`
3. TTL expires OR daemon restarts
4. Daemon polls again → sees `in_progress`, skips
5. ✅ No duplicate spawn occurs

**Defense layers verified:**

| Failure Scenario | Primary | Secondary | Tertiary | Result |
|-----------------|---------|-----------|----------|--------|
| TTL expires | ✅ Prevents | ❌ Expired | ✅ Prevents | Safe |
| Daemon restarts | ✅ Prevents | ❌ Lost | ✅ Prevents | Safe |
| Beads update fails | ❌ Failed | ✅ Prevents | ✅ Prevents | Safe |

---

## Unexplored Questions

**Questions that emerged:**
- Should we add metrics for how often secondary/tertiary dedup layers trigger? Would help identify beads RPC reliability issues.
- Should SpawnedIssueTracker TTL be configurable? Current 6h works, but different deployments might need different values.

**What remains unclear:**
- None - straightforward bug fix with clear root cause and solution

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5
**Workspace:** `.orch/workspace/og-arch-daemon-duplicate-spawn-14feb-6c9b/`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-ttl-fragility.md`
**Beads:** `bd show orch-go-07i`

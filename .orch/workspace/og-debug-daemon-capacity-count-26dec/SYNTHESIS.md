# Session Synthesis

**Agent:** og-debug-daemon-capacity-count-26dec
**Issue:** orch-go-s2j7
**Duration:** 2025-12-26
**Outcome:** success

---

## TLDR

Daemon capacity count stuck at 3/3 because `DefaultActiveCount()` counted ALL 26 OpenCode sessions (including old/stale ones) instead of just active ones. Fixed by filtering to sessions updated within last 30 minutes, and reordering daemon loop to reconcile before writing status.

---

## Delta (What Changed)

### Files Modified
- `pkg/daemon/daemon.go:416-444` - Updated `DefaultActiveCount()` to filter sessions by `time.updated` field, counting only sessions active within 30 minutes (matching `orch status` threshold)
- `cmd/orch/daemon.go:203-227` - Reordered daemon loop to call `ReconcileWithOpenCode()` BEFORE `WriteStatusFile()` so status shows accurate counts

### Commits
- Pending commit: "fix: filter OpenCode sessions by recency in DefaultActiveCount()"

---

## Evidence (What Was Observed)

- `curl http://127.0.0.1:4096/session | jq 'length'` returns 26 (many stale sessions from Dec 20-26)
- Only 1 session had activity within 30 minutes (verified with jq filter)
- `Reconcile(actualCount)` requires actualCount < poolCount to free slots
- With actualCount=26 and poolCount=3, 26 >= 3 is true, so prior fix never freed slots
- Prior investigation added `Pool.Reconcile()` but didn't account for stale sessions in OpenCode API response

### Root Cause Chain
1. Daemon spawns 3 agents, Pool.activeCount = 3
2. Agents complete, OpenCode sessions remain (stale)
3. New poll cycle calls `ReconcileWithOpenCode()`
4. `DefaultActiveCount()` queries OpenCode API, returns 26 (all sessions)
5. `Reconcile(26)` sees 26 >= 3, does nothing
6. Pool stays at 3/3 forever

### Tests Run
```bash
# Run daemon package tests
go test ./pkg/daemon/... -count=1
# ok  	github.com/dylan-conlin/orch-go/pkg/daemon	0.150s

# Build verification
go build ./cmd/orch
# Success, no errors

# Verify session filtering logic
curl -s http://127.0.0.1:4096/session | jq '[.[] | select((.time.updated / 1000) > (now - 1800))] | length'
# Returns 1 (only active sessions)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stuck-while.md` - Full root cause analysis (extends prior investigation)

### Decisions Made
- Filter by 30-minute threshold: Matches `orch status` agent matching threshold for consistency across orch commands
- Reorder reconcile before status write: Ensures status file reflects post-reconciliation counts, not stale pre-reconciliation counts

### Constraints Discovered
- **OpenCode sessions persist indefinitely**: Sessions don't auto-close when agents complete or exit
- **Session count != active agent count**: Must filter by recency to distinguish active vs abandoned sessions
- **Order matters in daemon loop**: Status snapshot must happen after state mutations

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-s2j7`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why doesn't OpenCode clean up old sessions automatically?
- Should there be a session cleanup command in orch (`orch cleanup-sessions`)?

**Areas worth exploring further:**
- Add metrics/logging for reconciliation events to monitor in production
- Consider explicit session ID tracking instead of time-based filtering for precision

**What remains unclear:**
- Behavior when sessions are streaming but no user input (may appear stale with >30 min idle time)
- Edge case when OpenCode API is unavailable (currently returns 0, may cause aggressive slot release)

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-daemon-capacity-count-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stuck-while.md`
**Beads:** `bd show orch-go-s2j7`

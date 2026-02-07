# Session Synthesis

**Agent:** og-inv-investigate-daemon-duplicate-15jan-a3e3
**Issue:** orch-go-0q9s7
**Duration:** 2026-01-15 13:45 → 2026-01-15 14:30
**Outcome:** success

---

## TLDR

Investigated why daemon spawns duplicate agents. Root cause: 5-minute TTL in SpawnedIssueTracker expires while agents work for hours, AND beads status update can fail silently. Fix: Add session-level dedup by checking OpenCode for existing sessions before spawn.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md` - Complete investigation with findings, test, and recommendation

### Files Modified
- None (investigation only)

### Commits
- Investigation file with complete analysis

---

## Evidence (What Was Observed)

- SpawnedIssueTracker TTL is hardcoded to 5 minutes (`spawn_tracker.go:37`)
- `ReconcileWithIssues()` exists but is only called in tests, not production (`pkg/daemon/daemon.go:496-500`)
- `bd ready` returns BOTH open AND in_progress issues (verified via `bd ready --help`)
- Status update failure is non-fatal - spawn continues with warning (`spawn_cmd.go:980-986`)
- **Production evidence:** orch-go-nqgjr had 19 duplicate workspaces and 80 comments showing repeated spawns ~5 min apart

### Tests Run
```bash
# Count duplicate workspaces for cross-project issue
ls .orch/workspace/ | grep -c cross-project
19

# View spawn history via comments
bd show orch-go-nqgjr | grep -c "Phase:"
# Multiple agents reporting Phase: Planning at 5-min intervals
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md` - Root cause analysis with implementation recommendations

### Decisions Made
- Session-level dedup is the recommended fix (uses source of truth: OpenCode sessions)
- Extended TTL is a backup, not the primary fix

### Constraints Discovered
- TTL-based protection is insufficient when status update can fail
- Daemon relies on 3 defense layers: (1) status update, (2) TTL tracker, (3) in_progress filter - when (1) fails, only 5 min protection

### Externalized via `kb`
- `kb quick constrain "Daemon spawn tracker TTL must be longer than agent work duration OR use session-level dedup" --reason "5-min TTL + failed status update = duplicate spawns every 5 min"` → kb-c854a5

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Tests passing (N/A - investigation only)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-0q9s7`

### Implementation Follow-up
Related issue already exists: **orch-go-2nruy** ("Add dedup check before spawn to prevent duplicate sessions")

That issue should implement:
1. Session dedup check in `daemon.Once()` - query OpenCode for existing sessions with same beads ID
2. Extend TTL from 5 min to 6 hours as backup
3. Optionally call `ReconcileWithIssues()` in poll cycle for additional defense

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does status update fail? (transient beads daemon issues? disk contention?)
- Should status update failure abort spawn rather than continue with warning?

**Areas worth exploring further:**
- Logging when status update fails to capture frequency
- Whether the beads daemon has stability issues under load

**What remains unclear:**
- Whether there are other code paths that could cause duplicates

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-investigate-daemon-duplicate-15jan-a3e3/`
**Investigation:** `.kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md`
**Beads:** `bd show orch-go-0q9s7`

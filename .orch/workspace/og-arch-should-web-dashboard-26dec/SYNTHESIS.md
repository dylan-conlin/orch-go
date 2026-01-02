# Session Synthesis

**Agent:** og-arch-should-web-dashboard-26dec
**Issue:** orch-go-untracked-1766769899
**Duration:** 2025-12-26 09:30 → 2025-12-26 10:15
**Outcome:** success

---

## TLDR

Designed daemon visibility feature for web dashboard. Recommended file-based status approach: daemon writes `~/.orch/daemon-status.json`, new `/api/daemon` endpoint reads it + launchd check, stats bar indicator shows capacity and health.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-design-web-dashboard-daemon-visibility.md` - Architect investigation with recommendations

### Files Modified
- `.orch/features.json` - Added 3 new feature items (feat-014, feat-015, feat-016)

### Commits
- None yet (investigation phase only)

---

## Evidence (What Was Observed)

- Daemon has `PoolStatus()` method (pool.go:181-210) ready for exposure with MaxWorkers, Active, Available, ActiveSlots
- Dashboard stats bar has established pattern with servers, beads, focus indicators (page.svelte:294-376)
- Daemon runs via launchd (`com.orch.daemon`) with PID visible, logs to `~/.orch/daemon.log`
- Current `/api/beads` shows queue depth but not processor health
- Daemon log shows capacity status: `[09:25:59] At capacity (3/3 agents active), waiting...`

### Analysis Run
```bash
# Checked launchd service
launchctl list | grep orch
# Output: 99854 0 com.orch.daemon

# Checked daemon logs
tail -30 ~/.orch/daemon.log
# Shows real-time capacity and spawn activity

# Verified API pattern
curl http://127.0.0.1:3348/api/beads
# Returns queue stats, not processor health
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-design-web-dashboard-daemon-visibility.md` - Full architect investigation with problem framing, exploration of 3 approaches, synthesis, and implementation details

### Decisions Made
- File-based status + launchd check is recommended approach because:
  - Minimal daemon changes (just write status file)
  - Works with existing serve.go patterns
  - No shared memory or inter-process complexity
  - Status file can include last-spawn timestamp for stall detection

### Constraints Discovered
- Daemon and serve are separate processes - need file or IPC for status sharing
- Status will be slightly stale (up to poll interval) - acceptable trade-off
- launchd process check may show PID but daemon could be stuck

### Externalized via `kn`
- N/A - decisions captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Feature items added to `.orch/features.json`:**

1. **feat-014:** Add daemon status file writing to polling loop
   - **Skill:** feature-impl
   - **Priority:** high (no dependencies)

2. **feat-015:** Add /api/daemon endpoint to orch serve
   - **Skill:** feature-impl
   - **Priority:** high (depends on feat-014)

3. **feat-016:** Add daemon indicator to dashboard stats bar
   - **Skill:** feature-impl
   - **Priority:** high (depends on feat-015)

**Implementation sequence:**
1. First: feat-014 (daemon writes status file)
2. Second: feat-015 (serve reads status + launchd check)
3. Third: feat-016 (frontend indicator)

**Context for implementers:**
```
Follow the /api/servers pattern in serve.go. Status file should be atomic write
(temp file + rename). Stall detection threshold = 2x poll_interval when
ready_issues > 0 but no recent spawn.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should stall detection consider blocked issues? (Daemon can't spawn if all ready work is blocked by other agents)
- What if daemon is rate-limited by Claude Max? (Not stalled, just waiting for capacity)
- Should we show queue depth alongside capacity? (Current design shows capacity only)

**Areas worth exploring further:**
- Real-time SSE events from daemon (alternative to polling status file)
- Historical daemon metrics (spawns per hour, average queue wait time)

**What remains unclear:**
- Reliability of launchd process check (PID may exist but process stuck)
- Optimal polling frequency for daemon status (60s like other stats? More frequent?)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-should-web-dashboard-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-design-web-dashboard-daemon-visibility.md`
**Beads:** `bd show orch-go-untracked-1766769899`

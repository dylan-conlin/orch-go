# Session Synthesis

**Agent:** og-inv-investigate-phantom-agent-27feb-019b
**Issue:** orch-go-3945
**Outcome:** success

---

## Plain-Language Summary

The "phantom" agent spawns were not phantom — they were a legitimately running `orch daemon run` process (PID 25932) that was spawning agents from the triage:ready queue. The user believed the daemon was stopped because (a) orch-dashboard doesn't auto-start it, and (b) there's no launchd service. But a manually-started daemon was still running. A secondary bug was found: after the daemon process dies, `orch status` and the dashboard continue reporting "Daemon - running" because they check if `daemon-status.json` exists, not whether the daemon process is alive. The spawn count metric is accurate and not inflated.

---

## Delta (What Changed)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-27-probe-phantom-agent-spawns-daemon-status-stale-file.md` - Probe documenting stale status file bug and spawn timeline
- `.kb/investigations/2026-02-27-investigate-phantom-agent-spawns.md` - Full investigation with 5 findings and D.E.K.N. summary

### Files Modified
- None (investigation only)

---

## Evidence (What Was Observed)

- events.jsonl shows 5 agents (e6wo, zqgk, iwb3, dm9x, rmef) with `daemon.spawn` events at sequential counts 93-101
- daemon.pid contains PID 25932 but `ps -p 25932` shows "PID process not found" — daemon is dead
- daemon-status.json persists with stale "paused" status after daemon death
- `handleDaemon()` in serve_system.go:419 sets `Running = true` just because the file is readable — no PID check
- `readDaemonStatus()` in status_cmd.go:1308 reads file without any process liveness verification
- Procfile has 3 services (api, web, opencode) — no daemon entry
- Spawn count metric (`orch stats`) only counts `session.spawned` events (stats_cmd.go:362)

### Tests Run
```bash
# Timeline analysis
grep '"session.spawned"' ~/.orch/events.jsonl | tail -20
# Result: 15 spawns in window, 5 daemon + 10 manual

# PID liveness check
cat ~/.orch/daemon.pid && ps -p 25932
# Result: PID 25932 is dead, file is stale

# Process scan
ps aux | grep -E 'orch|daemon'
# Result: Only orch serve running, no daemon process
```

---

## Architectural Choices

No architectural choices — investigation only, no code changes.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Daemon status reader path (serve_system.go, status_cmd.go) has no process liveness check — relies purely on file existence
- `DetermineStatus()` with staleness threshold exists in pkg/daemon/status.go:162 but is only used by the daemon writer, not readers
- Daemon cleanup (RemoveStatusFile, PIDLock.Release) only runs on graceful shutdown via defer — unclean deaths leave stale files

### Externalized via `kn`
- Will create bug issue for stale daemon status reporting

---

## Next (What Should Happen)

**Recommendation:** close + create follow-up bug issue

### Follow-up Issue
**Title:** Daemon status readers should verify process liveness, not just file existence
**Skill:** systematic-debugging
**Priority:** P2
**Context:**
```
serve_system.go:419 and status_cmd.go:1308 report daemon as "running" based on
stale daemon-status.json after process death. Fix: either check PID liveness via
daemon.pid + isProcessAlive(), or apply staleness threshold (DetermineStatus()
already has this logic, just unused by readers).
```

---

## Unexplored Questions

- How long can daemon-status.json persist as stale before DetermineStatus() would flag it as "stalled"? The 2×pollInterval threshold exists but isn't applied by readers.
- Should daemon.pid be checked at the same time as daemon-status.json for a more robust liveness signal?
- Could the daemon be auto-started by the Procfile/overmind to prevent manual-start-forget scenarios?

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-investigate-phantom-agent-27feb-019b/`
**Investigation:** `.kb/investigations/2026-02-27-investigate-phantom-agent-spawns.md`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-27-probe-phantom-agent-spawns-daemon-status-stale-file.md`
**Beads:** `bd show orch-go-3945`

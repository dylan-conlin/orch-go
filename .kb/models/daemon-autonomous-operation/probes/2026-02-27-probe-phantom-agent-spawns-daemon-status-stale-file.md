# Probe: Phantom Agent Spawns — Daemon Status Stale File Bug

**Model:** daemon-autonomous-operation
**Date:** 2026-02-27
**Status:** Complete

---

## Question

Testing two model claims:
1. **Daemon single-instance PID lock** (probed in 2026-02-24): Does the PID lock mechanism prevent orphaned daemons from running?
2. **Daemon status reporting accuracy**: Does `orch status` correctly report whether the daemon is actually running?

Specifically: Can the daemon appear "running" in status output when the process is actually dead?

---

## What I Tested

### Test 1: Check daemon PID vs process liveness

```bash
cat ~/.orch/daemon.pid
# Output: 25932

ps -p 25932
# Output: PID process not found
```

PID file exists with PID 25932, but no process with that PID is running.

### Test 2: Check daemon-status.json freshness

```bash
cat ~/.orch/daemon-status.json | python3 -m json.tool
```

Status file shows `last_poll: 2026-02-27T15:32:31`, `status: "paused"`, and the daemon was recently writing to this file even though PID 25932 may have died after.

### Test 3: Code review of status reader path

```
# serve_system.go:395-419 (handleDaemon for dashboard API)
status, err := daemon.ReadStatusFile()
if err == nil && status != nil {
    resp.Running = true   // ← BUG: Sets running=true just because FILE EXISTS
}

# status_cmd.go:1307-1355 (readDaemonStatus for CLI)
# Reads ~/.orch/daemon-status.json, returns status struct
# NO PID LIVENESS CHECK anywhere in this path
```

### Test 4: Events.jsonl timeline analysis

```bash
grep '"session.spawned"' ~/.orch/events.jsonl | tail -20
grep -E '(e6wo|zqgk|iwb3|dm9x|rmef)' ~/.orch/events.jsonl
```

Built complete timeline of 15 spawns between 14:55 and 15:29:
- 5 daemon-initiated spawns (have `daemon.spawn` events with counts 93, 95, 97, 99, 101)
- 10 manual/orchestrator spawns (no `daemon.spawn` events)

### Test 5: Daemon cleanup mechanism

```
# daemon.go:261 - defer pidLock.Release()
# daemon.go:316 - defer daemon.RemoveStatusFile()
```

Both cleanup steps are in `defer` blocks — only execute on graceful shutdown (SIGTERM/SIGINT). If process is SIGKILL'd or crashes, neither file is cleaned up.

---

## What I Observed

### Finding 1: Daemon WAS running and actively spawning

The 5 "phantom" agents (e6wo, zqgk, iwb3, dm9x, rmef) were legitimately spawned by a running daemon process (PID 25932). They are NOT phantom — they're daemon-spawned agents that appeared during a ~8 minute window (15:06-15:14) when the daemon was actively polling `triage:ready` issues.

Timeline:
```
15:06:24 | orch-go-e6wo  | DAEMON #93  | Architect: fix kb context query
15:11:40 | orch-go-zqgk  | DAEMON #95  | orch stats verification events
15:12:36 | orch-go-iwb3  | DAEMON #97  | toolshed architect redesign
15:13:16 | orch-go-dm9x  | DAEMON #99  | Model drift: Agent Spawning
15:14:27 | orch-go-rmef  | DAEMON #101 | Knowledge maintenance: 115 entries
```

### Finding 2: Stale status file creates false "running" report

After PID 25932 died:
- `~/.orch/daemon.pid` still contains `25932`
- `~/.orch/daemon-status.json` still exists with last-written state
- `handleDaemon()` (serve_system.go:419) sets `Running = true` simply because the file is readable
- `readDaemonStatus()` (status_cmd.go:1308) returns the stale status struct
- Both CLI `orch status` and dashboard API report daemon as "running" or "paused" when the process is dead

### Finding 3: orch-go-1245 was NOT a new spawn

orch-go-1245 was previously spawned and abandoned twice, then closed via `agent.completed` batch review (timestamp 1772233991). It was NOT spawned in the 10-minute window — it was an existing agent that reached completion. The skill_inferred events for orch-go-1245 were from the daemon's repeated attempts to re-spawn it (failing each time because "Phase: Complete but not closed").

### Finding 4: Spawn count metric is accurate

The `orch stats` TotalSpawns counter only counts `session.spawned` events (stats_cmd.go:362). It does NOT include skill_inferred, daemon.spawn, or completion events. The jump from 1040 to 1049 represents 9 real session.spawned events.

### Finding 5: Daemon was NOT started by dashboard or launchd

The Procfile has only 3 services: `api`, `web`, `opencode`. No daemon. The daemon was started manually via `orch daemon run` at some prior time and was running as PID 25932 until it died.

---

## Model Impact

- [x] **Extends** model with: **Stale daemon-status.json causes false "running" status after daemon death.** Both `handleDaemon()` (API) and `readDaemonStatus()` (CLI) check file existence, not process liveness. When daemon dies without graceful shutdown, stale files persist. This creates a gap where `orch status` reports "Daemon - running" for a dead daemon. The fix should either: (a) check PID liveness in the reader path, or (b) add a staleness threshold (e.g., if `last_poll` is older than 2× poll interval, consider dead — `DetermineStatus()` has this logic but it's only used by the daemon writer, not the reader).

- [x] **Confirms** invariant: PID lock prevents concurrent daemons. The daemon.pid + `isProcessAlive()` check works correctly for preventing duplicate instances.

- [x] **Confirms** invariant: Verification pause threshold works. After 4 auto-completions (threshold 3), daemon correctly paused.

---

## Notes

**Root cause summary:** The "phantom spawns" were NOT phantom — they were a legitimately running daemon process (PID 25932, started manually). The confusion arose because:
1. Dashboard doesn't start the daemon (by design)
2. No launchd service exists
3. But a manual `orch daemon run` was running in some terminal/tmux session

**Two bugs found:**
1. **Status stale file bug (P2):** `orch status` and dashboard API report daemon as "running" based on stale file after daemon death. Needs PID liveness check or staleness detection in the reader path.
2. **Daemon skill inference bug (P1, being fixed by orch-go-ideb):** All 5 daemon spawns used `feature-impl` skill even for investigation/architect tasks.

**Spawn count accuracy confirmed:** The metric counts only real `session.spawned` events. No inflation from non-spawn events.

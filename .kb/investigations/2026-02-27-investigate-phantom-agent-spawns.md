# Investigation: Phantom Agent Spawns When Daemon Should Be Paused

**Date:** 2026-02-27
**Status:** Complete
**Beads:** orch-go-3945

## D.E.K.N. Summary

- **Delta:** Identified root cause of "phantom" spawns — they were legitimate daemon-initiated spawns from a manually-started `orch daemon run` process (PID 25932). Discovered a stale-file bug in daemon status reporting.
- **Evidence:** events.jsonl timeline showing 5 `daemon.spawn` events with sequential counts (93-101), PID file containing dead process ID, code paths in serve_system.go:419 and status_cmd.go:1308 that read stale files without liveness checks.
- **Knowledge:** Daemon status reporting has no process liveness verification. `orch status` reports "running" based on file existence, not PID check. This creates false positives after unclean daemon shutdown.
- **Next:** Create bug issue for stale daemon status reporting. Daemon skill inference fix already in progress (orch-go-ideb).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md | extends | yes | PID lock works for preventing concurrent instances but doesn't help status readers detect dead daemons |
| .kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-feb14-incident.md | related | n/a | Different failure mode — that was duplicate spawns, this was "expected" spawns misidentified as phantom |

## Question

What process spawned 8+ agents in ~10 minutes when the daemon was believed to be paused/stopped?

## Finding 1: Daemon WAS Running (Not Phantom)

**Test:** Analyzed `~/.orch/events.jsonl` for all events matching the 6 reported agent IDs (e6wo, zqgk, iwb3, dm9x, rmef, 1245).

**Observed:** 5 of the 6 agents have `daemon.spawn` events with sequential daemon spawn counts:

```
15:06:24 | orch-go-e6wo  | DAEMON #93  | Architect: fix kb context query
15:11:40 | orch-go-zqgk  | DAEMON #95  | orch stats verification events
15:12:36 | orch-go-iwb3  | DAEMON #97  | toolshed architect redesign
15:13:16 | orch-go-dm9x  | DAEMON #99  | Model drift: Agent Spawning
15:14:27 | orch-go-rmef  | DAEMON #101 | Knowledge maintenance: 115 entries
```

orch-go-1245 was NOT spawned in this window — it was a previously spawned agent that reached completion via batch review (`agent.completed` at 1772233991).

**Conclusion:** The spawns were legitimate daemon activity, not phantom. A daemon process (PID 25932) was actively running and spawning from the `triage:ready` queue.

## Finding 2: Daemon Was Manually Started (Not Via Dashboard or Launchd)

**Test:** Checked Procfile, launchctl, and orch-dashboard script.

**Observed:**
- Procfile has 3 services: `api`, `web`, `opencode` — no daemon entry
- `launchctl list com.orch.daemon` → "Could not find service"
- orch-dashboard script starts overmind with Procfile — no daemon management

**Conclusion:** The daemon was started manually via `orch daemon run` in some terminal/tmux session. The user forgot or didn't know it was running.

## Finding 3: Stale Status File Bug

**Test:** Checked daemon PID file vs process liveness, then traced the code path.

**Observed:**
```bash
cat ~/.orch/daemon.pid → 25932
ps -p 25932 → "PID process not found"
cat ~/.orch/daemon-status.json → {"status": "paused", "last_poll": "2026-02-27T15:32:31", ...}
```

Both files persist after daemon death. The status reader code:

```go
// serve_system.go:418-419 (dashboard API)
status, err := daemon.ReadStatusFile()
if err == nil && status != nil {
    resp.Running = true  // Sets running=true because FILE exists, not process
}

// status_cmd.go:1308-1322 (CLI)
// Reads file, unmarshals JSON, returns struct — no PID check
```

**Conclusion:** After daemon dies without graceful shutdown (SIGKILL, crash), stale files persist. `orch status` and dashboard API report daemon as "running" based on file existence alone. No process liveness check exists in the reader path.

The daemon does have `DetermineStatus()` (pkg/daemon/status.go:162) with a staleness threshold (`2 × pollInterval`), but this function is only called by the daemon itself to write status — it's never used by the readers to validate what they read.

## Finding 4: Spawn Count Metric Is Accurate

**Test:** Traced spawn count through code: `stats_cmd.go:362-450` and `pkg/session/session.go:311-350`.

**Observed:**
- `orch stats` counts ONLY `session.spawned` events from events.jsonl (explicit `case "session.spawned"` check)
- `orch status` SESSION METRICS counts spawns from session store (`len(s.session.Spawns)`)
- Neither includes: `daemon.spawn`, `spawn.skill_inferred`, `agent.completed`, or any other event type
- The jump from 1040→1049 represents 9 real `session.spawned` events: 5 daemon + 4 manual

**Conclusion:** Spawn count metrics are not inflated by non-spawn events. The count is accurate.

## Finding 5: Complete/Abandon Don't Trigger Spawns Directly

**Test:** Examined event sequence around `agent.abandoned` for orch-go-1245.

**Observed:**
- orch-go-1245 abandoned at timestamp 1772233372
- Next daemon spawn (e6wo) at timestamp 1772233584 (~3.5 min later)
- This is the daemon's normal poll cycle — abandoning freed a concurrency slot, daemon filled it on next poll

**Conclusion:** `orch complete` and `orch abandon` don't trigger spawns as side effects. They free concurrency slots; the daemon fills them on its next poll cycle (normal behavior).

## Summary Answer

**What spawned those agents?** A manually-started `orch daemon run` process (PID 25932) that was actively running. Not phantom — legitimate daemon spawns.

**Why did the user think daemon was stopped?**
1. Dashboard doesn't auto-start daemon (by design)
2. No launchd service
3. The daemon was started in some earlier terminal session that the user may have forgotten about

**What about the "Daemon - running" report after daemon death?** Bug: stale `daemon-status.json` persists after unclean shutdown. Status readers check file existence, not process liveness.

**Is the spawn count inflated?** No. Counts only `session.spawned` events. Accurate.

# Probe: Daemon Single-Instance PID Lock

**Date:** 2026-02-24
**Status:** Complete
**Model:** daemon-autonomous-operation
**Issue:** orch-go-1223

## Question

Does the daemon's lack of single-instance guard allow multiple concurrent processes that fight over status files and spawns?

## What I Tested

1. Built orch-go binary and started two `daemon run` processes simultaneously
2. First daemon acquired PID lock at `~/.orch/daemon.pid` and started polling
3. Second daemon attempted to acquire the same PID lock

## What I Observed

**Before fix:** No single-instance guard existed. The daemon wrote to `~/.orch/daemon-status.json` on each poll cycle but never checked if another daemon was already running. Multiple daemons could start from different sessions (tmux windows, cron, manual invocations) and silently accumulate.

**After fix:** PID lock acquired at daemon start. Second invocation fails immediately:
```
Error: cannot start daemon: daemon already running: PID 17650
```

Key behaviors verified:
- Second daemon exits with code 1 (fail-fast)
- Stale PID files (from crashed daemons) are detected via `kill(pid, 0)` and cleaned up automatically
- PID lock is released on clean shutdown (SIGINT/SIGTERM → defer Release)
- PID included in status file for dashboard visibility

## Model Impact

**Extends** the daemon-autonomous-operation model:

The model documents "Duplicate Spawns" as a known failure mode where the same issue gets spawned twice due to poll timing races. However, it doesn't address the **meta-duplicate** problem: multiple daemon *processes* running concurrently, each independently polling and spawning. This is a higher-level failure that the existing SpawnedIssueTracker (in-memory) cannot prevent because each daemon instance has its own tracker.

The PID lock addresses this at the process level, complementing the existing dedup layers:
- Layer 0 (new): PID lock prevents multiple daemon processes
- Layer 1: SpawnedIssueTracker prevents same-issue re-spawn within one daemon
- Layer 2: Content-aware dedup prevents same-title re-spawn
- Layer 3: Fresh status check catches TOCTOU race
- Layer 4: BeadsStatus update (in_progress) is the persistent dedup gate

**New model claim:** The daemon requires a single-instance guard (PID file) to prevent process-level duplication that cannot be caught by issue-level dedup mechanisms.

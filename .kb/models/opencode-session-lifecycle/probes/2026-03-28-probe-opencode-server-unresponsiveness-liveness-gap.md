# Probe: OpenCode Server Unresponsiveness — Liveness Detection Gap

**Model:** opencode-session-lifecycle
**Date:** 2026-03-28
**Status:** Complete
**claim:** N/A (extends model — no existing claim about process supervision liveness)
**verdict:** extends

---

## Question

When OpenCode becomes unresponsive during long sessions, does the supervision layer (overmind + ServiceMonitor) detect the failure and recover? The model documents session accumulation (Failure Mode 2) and session cleanup, but doesn't address server-level liveness.

---

## What I Tested

### 1. Service Monitor crash detection mechanism
Read `pkg/service/monitor.go` — the `detectAndHandleCrashes()` function only checks PID changes:
```go
// Detect crash: PID changed
if last.PID != 0 && last.PID != current.PID {
    crashedServices = append(crashedServices, ...)
}
```

### 2. Event log analysis for crash evidence
```bash
cat ~/.orch/events.jsonl | grep '"service.crashed"\|"service.restarted"'
# Output: (empty — zero crash events)

cat ~/.orch/events.jsonl | grep '"service.started".*opencode' | tail -10
# All entries show same PID — no PID change = no detected crash
```

### 3. System log analysis
```bash
log show --predicate 'eventMessage contains "jetsam" OR eventMessage contains "killed"' --last 7d
# No OOM kills or jetsam events for bun/opencode
```

### 4. Current process state
```bash
ps aux | grep 'opencode serve'
# PID: 73763, RSS: 594MB, CPU: 17.7%
```

### 5. OpenCode SSE subscription mechanism
Read `opencode/packages/opencode/src/bus/index.ts`:
- `Bus.subscribeAll()` adds callbacks to a Map with no expiration
- `Bus.publish()` uses `Promise.all(pending)` — all subscribers awaited
- If any subscriber blocks, the entire publish blocks

---

## What I Observed

1. **Zero crash events** in the event log — OpenCode never actually crashes
2. **ServiceMonitor is blind to unresponsiveness** — only detects PID changes
3. **checkOpenCode()** in doctor_checks.go makes real API calls but only runs manually
4. **Overmind's --auto-restart** only triggers on process death
5. **The gap is between "process alive" and "service healthy"**
6. **SSE subscription model is the likely root cause** — Bus.publish() blocks on all subscribers

---

## Model Impact

- [x] **Extends** model with: New failure mode — "Server Unresponsiveness Without Crash"
  - Added invariant #6: Server can become unresponsive without crashing
  - Added Failure Mode 5: Detailed root cause and fix description
  - The supervision layer has a blind spot: PID-based detection misses event loop blocking
  - Fix: HTTP liveness probing in ServiceMonitor (3 consecutive failures = restart)

---

## Notes

- Fix has two parts: (1) orch-go ServiceMonitor HTTP probing (implemented), (2) OpenCode SSE defensive measures (cross-repo issue)
- 594MB RSS is notable but needs heap profiling to confirm leak vs working set

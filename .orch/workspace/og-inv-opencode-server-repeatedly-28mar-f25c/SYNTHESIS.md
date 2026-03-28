# Session Synthesis

**Agent:** og-inv-opencode-server-repeatedly-28mar-f25c
**Issue:** orch-go-exutj
**Outcome:** success

---

## Plain-Language Summary

OpenCode keeps going down during long sessions because the process hangs without crashing — it's alive (same PID) but the HTTP server stops responding. The service monitor was only checking if the PID changed, so it was blind to this. I added HTTP health checks that ping OpenCode every 10 seconds: if it fails to respond 3 times in a row while the process is still alive, the monitor force-restarts it via overmind. This turns a manual orch-dashboard restart into automatic recovery in ~30 seconds.

The root cause is on the OpenCode side — SSE event subscriptions accumulate and block the event loop — but that's a separate fix for the fork. The orch-go fix ensures recovery happens automatically regardless of why OpenCode hangs.

---

## Verification Contract

See VERIFICATION_SPEC.yaml. Tests: go test ./pkg/service/ — 10 passed, 0 failed.

---

## Delta (What Changed)

### Files Modified
- pkg/service/monitor.go — Added HealthProbe, AddHealthProbe, checkHealthProbes, probeHealth
- pkg/service/monitor_test.go — 6 new tests for health probing
- pkg/service/event_adapter.go — LogServiceUnresponsive method
- pkg/events/logger.go — EventTypeServiceUnresponsive constant
- cmd/orch/serve.go — Wired OpenCode health probe
- .kb/models/opencode-session-lifecycle/model.md — Invariant #6, Failure Mode #5

### Files Created
- .kb/models/opencode-session-lifecycle/probes/2026-03-28-probe-*.md

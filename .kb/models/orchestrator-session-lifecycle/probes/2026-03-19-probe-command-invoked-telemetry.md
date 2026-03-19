# Probe: command.invoked Telemetry for Measurement Commands

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-19
**Status:** Complete

---

## Question

Can we add `command.invoked` telemetry events to measurement commands (harness audit/report/gate-effectiveness, health, doctor, stats) with caller context detection (human/daemon/orchestrator/worker) using existing environment variable signals (`CLAUDE_CONTEXT`, `ORCH_SPAWNED`)?

---

## What I Tested

1. Reviewed `pkg/events/logger.go` for existing event emission patterns — found consistent `EventType*` constants + `Log*` helper pattern
2. Traced caller context signals through the spawn pipeline:
   - `pkg/spawn/claude.go:142` — exports `ORCH_SPAWNED=1` and `CLAUDE_CONTEXT={worker|orchestrator|meta-orchestrator}`
   - `pkg/tmux/spawn_opencode.go:71` — exports `ORCH_WORKER=1` and `CLAUDE_CONTEXT`
   - `pkg/spawn/backends/inline.go:33` — exports `ORCH_WORKER=1` and `CLAUDE_CONTEXT`
3. Verified no command currently emits telemetry — all 6 targets only call their `run*()` functions directly
4. Implemented and ran tests:

```bash
go test ./pkg/events/ -run TestLogCommandInvoked -v
# PASS (2/2 tests)

go test ./cmd/orch/ -run "TestDetectCaller|TestEmitCommand" -v
# PASS (7/7 tests)

go test ./cmd/orch/ -run "TestHarness|TestStats|TestDoctor" -v
# PASS (12/12 tests — no regressions)
```

---

## What I Observed

- All 6 target commands (harness audit, harness report, harness gate-effectiveness, health, doctor, stats) were pure CLI entry points with no telemetry
- The `CLAUDE_CONTEXT` env var is reliably set by all 3 spawn backends (Claude CLI, OpenCode headless, inline), making it a robust signal for caller detection
- `ORCH_SPAWNED=1` serves as a fallback for cases where `CLAUDE_CONTEXT` might not be set but the process is still spawned
- No daemon-level calls to these commands exist — the daemon calls its own internal functions, not CLI subcommands. The "daemon" caller value would only appear if someone runs `orch stats` from within a daemon context (unlikely today)
- The `flagsFromCmd` helper uses cobra's `Flags().Visit()` which only visits flags that were explicitly set, avoiding noise from defaults

---

## Model Impact

- [x] **Extends** model with: New `command.invoked` event type enables usage analytics for measurement commands. This fills a gap in the orchestrator-session-lifecycle model — previously no visibility into which diagnostic/measurement tools are actually consulted during sessions and by whom (human orchestrator vs spawned agents). Data collected will show whether measurement commands are human-only tools or also used by agents.

---

## Notes

Caller context detection heuristic (priority order):
1. `CLAUDE_CONTEXT=orchestrator` or `meta-orchestrator` → "orchestrator"
2. `CLAUDE_CONTEXT=worker` or `ORCH_SPAWNED=1` → "worker"
3. Default → "human"

Files changed:
- `pkg/events/logger.go` — Added `EventTypeCommandInvoked`, `CommandInvokedData`, `LogCommandInvoked`
- `pkg/events/logger_test.go` — Added `TestLogCommandInvoked`, `TestLogCommandInvoked_NoFlags`
- `cmd/orch/telemetry.go` — New file: `detectCallerContext()`, `emitCommandInvoked()`, `flagsFromCmd()`
- `cmd/orch/telemetry_test.go` — New file: 7 tests for caller detection and event emission
- `cmd/orch/harness_audit_cmd.go` — Added `emitCommandInvoked` call in RunE
- `cmd/orch/harness_report_cmd.go` — Added `emitCommandInvoked` call in RunE
- `cmd/orch/harness_gate_effectiveness_cmd.go` — Added `emitCommandInvoked` call in RunE
- `cmd/orch/health_cmd.go` — Added `emitCommandInvoked` call in RunE
- `cmd/orch/doctor.go` — Added `emitCommandInvoked` call in RunE
- `cmd/orch/stats_cmd.go` — Added `emitCommandInvoked` call in RunE
- `CLAUDE.md` — Added `command.invoked` to event table

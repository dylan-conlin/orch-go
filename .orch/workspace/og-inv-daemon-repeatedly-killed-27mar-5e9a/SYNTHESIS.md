# Synthesis: Daemon SIGKILL Investigation

## Plain-Language Summary

The daemon was getting killed by macOS launchd 23 times because it took too long to shut down. When launchd sends a stop signal, it waits 5 seconds for the process to exit, then force-kills it. The daemon was exceeding that 5-second window for two reasons: (1) it ran a knowledge-base reflection analysis on exit with no timeout, and (2) the main loop didn't check for the stop signal between its many network-calling operations. The fix adds a 3-second timeout to the reflection analysis, adds "shutdown gates" between major operations in the loop, and increases launchd's patience from 5 to 15 seconds. After the fix, the daemon exits in ~25 milliseconds.

## Root Cause

Two-layer blocking on shutdown:

1. **Exit-time reflection (no timeout):** `defer runReflectionAnalysis()` calls `kb reflect --global` via `exec.Command` with no context/timeout. Takes ~2.5s normally but blocks indefinitely on slow I/O.

2. **Main loop ignores cancellation mid-cycle:** The OODA loop checks `ctx.Done()` only at the top and during sleeps. Between checks, 10+ operations run sequentially without context awareness — reconciliation, periodic tasks (cleanup, recovery, orphan detection, health), completions, listing, spawn cycle.

Combined, these easily exceed launchd's 5-second ExitTimeOut default.

## Fix Applied

| Change | File | Effect |
|--------|------|--------|
| `ShutdownReflectTimeout` (3s) | `pkg/daemon/reflect.go` | `exec.CommandContext` kills `kb reflect` after 3 seconds |
| Context-aware reflection API | `pkg/daemon/reflect.go` | New `RunReflectionWithContext`/`RunAndSaveReflectionWithContext` |
| Shutdown timeout in handler | `cmd/orch/daemon_handlers.go` | `runReflectionAnalysis` uses 3s context |
| `shutdownRequested()` gates | `cmd/orch/daemon.go` | 4 context checks between major loop operations |
| `ExitTimeOut=15` | `com.orch.daemon.plist` | Safety net: launchd waits 15s before SIGKILL |

## Verification Contract

See `VERIFICATION_SPEC.yaml` for full test details.

**Key outcomes:**
- Before: `runs = 23`, `last terminating signal = Killed: 9`
- After: Clean exit in 23-25ms, no SIGKILL
- Tests: `go test ./pkg/daemon/...` passes (22.6s)

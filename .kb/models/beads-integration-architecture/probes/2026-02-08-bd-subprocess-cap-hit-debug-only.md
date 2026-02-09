# Probe: bd_subprocess_cap_hit should be debug-only by default

**Model:** /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
**Date:** 2026-02-08
**Status:** Complete

---

## Question

Should `bd_subprocess_cap_hit` logs be suppressed at normal runtime (info-level output) and only appear when debug logging is explicitly enabled?

---

## What I Tested

**Command/Code:**
```bash
# Reproduce current behavior (pre-fix)
ORCH_BD_MAX_CONCURRENT=1 orch status

# Verify behavior after code change (default runtime)
ORCH_BD_MAX_CONCURRENT=1 go run ./cmd/orch status

# Verify behavior after code change (debug runtime)
ORCH_BD_MAX_CONCURRENT=1 ORCH_DEBUG=1 go run ./cmd/orch status
```

**Environment:**
- Branch/worktree: `/Users/dylanconlin/Documents/personal/orch-go`
- Forced semaphore pressure via `ORCH_BD_MAX_CONCURRENT=1`
- Code change: wrapped `bd_subprocess_cap_hit` log lines with `if os.Getenv("ORCH_DEBUG") != ""` in both beads fallback client and serve limiter

---

## What I Observed

**Output:**
```text
# Pre-fix reproduction:
2026/02/08 ... event=bd_subprocess_cap_hit component=beads operation="bd comments" inflight=1 cap=1
(dozens of repeated lines before normal status output)

# Post-fix, default runtime:
SYSTEM HEALTH
  ✅ Dashboard (port 3348) - listening
  ✅ OpenCode (port 4096) - listening
  ✅ Daemon - running (4 ready)
(no bd_subprocess_cap_hit lines)

# Post-fix, debug runtime:
2026/02/08 ... event=bd_subprocess_cap_hit component=beads operation="bd show" inflight=1 cap=1
(repeated cap-hit lines visible again with ORCH_DEBUG=1)
```

**Key observations:**
- Default command output is now clean; `bd_subprocess_cap_hit` no longer floods status output.
- Diagnostic visibility is preserved when `ORCH_DEBUG=1` is set.

---

## Model Impact

**Verdict:** extends — CLI fallback observability should distinguish operator-facing output from debug diagnostics

**Details:**
The model already captures fallback behavior and performance failure modes, but this probe adds an operational invariant: cap-hit telemetry is useful for diagnosis but should be debug-gated to prevent routine CLI noise. This preserves debuggability without degrading day-to-day command ergonomics.

**Confidence:** High — reproduced noisy baseline, then validated both default-suppressed and debug-enabled behavior with direct command runs.

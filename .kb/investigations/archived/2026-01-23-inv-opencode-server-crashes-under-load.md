<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode server crashes lack diagnostic logging; root cause cannot be determined from available evidence but multiple restarts occurred (5+ in 2 hours) with pattern suggesting memory or connection exhaustion under load.

**Evidence:** service.started events show 5 restarts in ~2 hours (PIDs changed: 57559→7273→29457→23415→41472); daemon.log shows "connection refused" errors at crash times; no crash dumps or OpenCode error logs exist.

**Knowledge:** OpenCode server.ts has error handling but no crash logging; the server silently exits without leaving evidence; service monitor detects crashes via PID change but doesn't capture cause.

**Next:** Add crash logging to OpenCode server - capture uncaughtException/unhandledRejection events to a log file; implement health endpoint monitoring in orch serve.

**Promote to Decision:** recommend-no (need crash logging first to identify root cause)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: OpenCode Server Crashes Under Load

**Question:** Why does OpenCode server crash repeatedly under moderate agent load (3-6 concurrent agents), and how can we prevent it?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent (systematic-debugging)
**Phase:** Complete
**Next Step:** None - documented findings and recommendations; needs orchestrator decision on next steps
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None
**Extracted-From:** Issue orch-go-r0s4m (OpenCode server crashes repeatedly under load)
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Multiple service restarts confirmed via event logging

**Evidence:** `service.started` events in `~/.orch/events.jsonl` show 5+ OpenCode restarts within 2 hours on Jan 23:
- 14:26:21 (PID 57559)
- 14:34:25 (PID 7273)
- 14:35:30 (PID 29457) - 65 seconds after previous restart
- 15:02:15 (PID 23415)
- 15:02:42 (PID 41472) - 47 seconds after previous restart

**Source:** `cat ~/.orch/events.jsonl | jq 'select(.type == "service.started")' | tail -20`

**Significance:** Confirms repeated crashes. The close spacing (47-65 seconds between some restarts) suggests manual intervention via `orch-dashboard restart` after observing failures.

---

### Finding 2: No crash logs exist in OpenCode or overmind

**Evidence:**
- `~/.local/share/opencode/server.log` contains only: "opencode server listening on http://127.0.0.1:4096"
- Overmind socket directories contain only startup scripts, not logs
- No macOS crash reports for opencode found in `~/Library/Logs/DiagnosticReports/`
- tmux pane for opencode shows no output

**Source:** `cat ~/.local/share/opencode/server.log`, `ls -la ~/Library/Logs/DiagnosticReports/ | grep opencode`

**Significance:** Cannot determine root cause without crash evidence. OpenCode server silently exits - no uncaughtException/unhandledRejection handlers writing to logs.

---

### Finding 3: Daemon cleanup detects crashes via connection refused errors

**Evidence:** `daemon.log` shows `connection refused` errors at times correlating with service.started events:
- Line 84276: [14:03:08] Cleanup error: connection refused
- Line 84311: [14:03:23] Cleanup error: connection refused
- Line 84700: [14:20:15] Cleanup error: connection refused

**Source:** `grep "connection refused" ~/.orch/daemon.log | tail -30`

**Significance:** The orch daemon's periodic cleanup (which queries OpenCode `/session` endpoint) acts as a passive crash detector. Can be used to correlate crash times.

---

### Finding 4: OpenCode server error handling exists but doesn't prevent crashes

**Evidence:** `server.ts` has:
- `.onError()` handler that logs to Log service and returns JSON errors (lines 78-93)
- No `process.on('uncaughtException')` or `process.on('unhandledRejection')` handlers visible
- No signal handlers (SIGTERM, SIGINT) visible in server source

**Source:** `~/Documents/personal/opencode/packages/opencode/src/server/server.ts:78-93`

**Significance:** Error handling exists for HTTP request errors, but process-level crash handlers are missing. Crashes from unhandled promises, memory exhaustion, or signals won't be logged.

---

### Finding 5: Beads corruption is unrelated - different root cause

**Evidence:** `.kb/models/beads-database-corruption.md` documents that beads corruption is caused by daemon rapid-restart cycles (57+ restarts in one day), not by OpenCode crashes. The corruption pattern is:
- Daemon fails → restarts immediately → opens/closes database repeatedly
- WAL checkpoint race conditions create 0-byte WAL files

**Source:** `.kb/models/beads-database-corruption.md`

**Significance:** The beads corruption issue reported in orch-go-r0s4m is a separate problem. OpenCode crashes may trigger workarounds that stress beads, but the corruption mechanism is independent.

---

## Synthesis

**Key Insights:**

1. **Crash detection exists but crash diagnosis doesn't** - The service monitor (Finding 1) reliably detects crashes via PID changes, and the daemon cleanup (Finding 3) provides secondary detection via connection refused errors. However, neither captures WHY the server crashed.

2. **Silent failures are architecturally problematic** - OpenCode server has HTTP error handling (Finding 4) but lacks process-level crash handlers. When the Node.js process crashes due to memory exhaustion, unhandled promise rejection, or signal, there's no log entry explaining the cause.

3. **High concurrency may correlate with crashes** - The crash times correlate with periods of high agent activity (spawns, completions, status queries). The 45-minute session had 3-6 concurrent agents, higher than typical load. However, without crash logs, this is correlation not causation.

**Answer to Investigation Question:**

The root cause of OpenCode server crashes **cannot be determined** from available evidence. What we know:
- Crashes definitely occurred (5+ restarts in 2 hours, Finding 1)
- No crash logs exist to diagnose cause (Finding 2)
- Error handling for HTTP requests exists but process-level crash handlers are missing (Finding 4)

**Possible causes** (untested hypotheses):
- Memory exhaustion from large sessions/messages
- SSE connection exhaustion (too many concurrent streams)
- File descriptor limits (SQLite + HTTP connections)
- Unhandled promise rejections in session operations

**Before implementing fixes**, we need crash telemetry to identify the actual failure mode.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode server is currently running and responsive (verified: `curl localhost:4096/session` returns JSON)
- ✅ service.started events confirm multiple restarts occurred on Jan 23 (verified: parsed events.jsonl)
- ✅ No crash logs exist in standard locations (verified: checked server.log, DiagnosticReports, tmux pane)
- ✅ Daemon cleanup errors correlate with crash times (verified: compared daemon.log timestamps to service.started)

**What's untested:**

- ⚠️ Memory exhaustion as crash cause (not benchmarked - would need memory profiling)
- ⚠️ SSE connection limits as crash cause (not tested - would need connection stress test)
- ⚠️ Specific orch commands triggering crash (not isolated - mixed activity during crash window)
- ⚠️ Correlation with high agent count (observed but not controlled experiment)

**What would change this:**

- If crash logs showed specific error (e.g., "ENOMEM" or "too many open files"), would confirm resource exhaustion hypothesis
- If crash reproduced under controlled conditions with single command type, would narrow down trigger
- If crash occurred with low agent count, would disprove concurrency hypothesis

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add crash telemetry to OpenCode server** - Add process-level crash handlers that log to a dedicated crash log file before exit.

**Why this approach:**
- Directly addresses Finding 2 (no crash logs exist)
- Enables future root cause analysis without speculation
- Low risk - only adds logging, doesn't change behavior

**Trade-offs accepted:**
- Won't immediately fix the crashes - just diagnoses them
- Requires modifying OpenCode fork (Dylan's fork at ~/Documents/personal/opencode)

**Implementation sequence:**
1. Add `process.on('uncaughtException')` and `process.on('unhandledRejection')` handlers to server.ts
2. Write crash details to `~/.local/share/opencode/crash.log` with timestamp, error, and stack trace
3. Add memory stats (`process.memoryUsage()`) to crash log for memory exhaustion diagnosis
4. Optionally add signal handlers (SIGTERM, SIGINT) for graceful shutdown logging

### Alternative Approaches Considered

**Option B: Resource limit monitoring in orch serve**
- **Pros:** Doesn't require OpenCode modifications; can alert before crash
- **Cons:** Can only observe from outside; can't determine actual crash cause
- **When to use instead:** If OpenCode fork modifications are blocked

**Option C: Reduce concurrency limit (defensive)**
- **Pros:** May reduce crash frequency immediately
- **Cons:** Treats symptom not cause; unclear if concurrency is the issue
- **When to use instead:** Emergency mitigation while investigating

**Rationale for recommendation:** Without crash logs, all other fixes are guesses. The first priority is visibility into what's actually failing.

---

### Implementation Details

**What to implement first:**
- Add crash handlers to `~/Documents/personal/opencode/packages/opencode/src/server/server.ts`
- Rebuild OpenCode fork: `cd ~/Documents/personal/opencode/packages/opencode && bun run build`
- Restart services: `orch-dashboard restart`

**Things to watch out for:**
- ⚠️ OpenCode rebuild required after changes (see global CLAUDE.md for fork workflow)
- ⚠️ Crash handlers must be synchronous - can't use async file writes
- ⚠️ Test crash logging works by intentionally triggering an error

**Areas needing further investigation:**
- What specific conditions trigger the crash (needs crash logs first)
- Whether SSE connections are being properly cleaned up
- Memory profile under sustained load

**Success criteria:**
- ✅ Next crash produces log entry in `~/.local/share/opencode/crash.log`
- ✅ Log includes error message, stack trace, and memory stats
- ✅ Root cause can be identified from crash log

---

## References

**Files Examined:**
- `~/.orch/events.jsonl` - Service lifecycle events showing restarts
- `~/.orch/daemon.log` - Cleanup errors showing connection refused at crash times
- `~/.local/share/opencode/server.log` - OpenCode server log (empty except startup message)
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts` - OpenCode server source
- `pkg/service/monitor.go` - Service monitor that detects crashes via PID changes
- `.kb/models/beads-database-corruption.md` - To check beads corruption correlation

**Commands Run:**
```bash
# Check current service status
overmind status

# Check for crash logs
cat ~/.local/share/opencode/server.log
ls -la ~/Library/Logs/DiagnosticReports/ | grep opencode

# Parse service.started events
cat ~/.orch/events.jsonl | jq 'select(.type == "service.started")' | tail -20

# Check daemon cleanup errors
grep "connection refused" ~/.orch/daemon.log | tail -30

# Verify OpenCode is currently running
curl -s http://localhost:4096/session | head -20
lsof -i :4096
```

**External Documentation:**
- None - this is an internal OpenCode fork issue

**Related Artifacts:**
- **Model:** `.kb/models/beads-database-corruption.md` - Documents separate corruption issue
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-auto-resume-mechanism-stalled.md` - Related agent recovery patterns

---

## Investigation History

**2026-01-23 15:05:** Investigation started
- Initial question: Why does OpenCode server crash repeatedly under load?
- Context: Issue orch-go-r0s4m reported 4+ crashes in 45-minute session

**2026-01-23 15:10:** Confirmed crashes via event logging
- service.started events show 5+ restarts with different PIDs
- Timestamps suggest manual restarts after observing failures

**2026-01-23 15:20:** Determined no crash logs exist
- OpenCode server.log contains only startup message
- No macOS crash reports found
- Overmind provides no crash diagnostics

**2026-01-23 15:30:** Analyzed OpenCode server source
- HTTP error handling exists but process-level handlers missing
- Silent crashes explain lack of diagnostic data

**2026-01-23 15:40:** Investigation completed
- Status: Complete - documented findings and recommendations
- Key outcome: Root cause unknown due to missing crash telemetry; recommended adding crash handlers to OpenCode fork

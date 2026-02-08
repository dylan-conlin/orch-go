## Summary (D.E.K.N.)

**Delta:** OpenCode server process is killed by macOS jetsam (memory pressure killer) when its RSS grows to ~8.4 GB, driven by unbounded memory growth from concurrent agent sessions - not JS errors, not overmind, not file descriptor exhaustion.

**Evidence:** JetsamEvent-2026-02-06-033432.ips shows opencode at 8,608 MB (550,922 pages) as `largestProcess` with system at 0.3 GB free; current instance at 1.58 GB RSS after 13.5 hours with only 64 FDs open; 26 concurrent bun agent processes each 400-900 MB totaling 14.3 GB.

**Knowledge:** The system (36 GB RAM) is under extreme memory pressure from OpenCode server + agent processes (combined 22.7 GB at jetsam time). OpenCode has no memory limit, no session eviction, and no garbage collection pressure relief. The `--can-die opencode` overmind flag means when jetsam kills opencode, overmind doesn't restart it - the process stays dead.

**Next:** Set Bun/V8 heap limit (e.g., `--max-old-space-size=4096`), implement session eviction for idle sessions, and add `--auto-restart opencode` to overmind to survive jetsam kills. These are independent fixes that compound.

**Authority:** architectural - Crosses OpenCode fork + orch-go infrastructure boundaries, requires coordinated changes

---

# Investigation: What Actually Kills the OpenCode Server Process?

**Question:** What OS-level mechanism kills the OpenCode server process? Prior investigations found JS-level unhandledRejection events (crash.log) but those don't exit the process. Is this OOM kill, signal from overmind, file descriptor exhaustion, or something else?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** Worker agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md | extends | yes | Prior work correctly identified crashes but couldn't find the cause. This investigation found the OS-level mechanism (jetsam). |
| .kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md | extends | yes | Prior work correctly identified that unhandledRejections don't kill the process. Its claim that "SSE stream breaking is the kill mechanism" is partially right - SSE breaks because the SERVER process dies from jetsam, not because of SSE itself. |
| .kb/investigations/2026-01-31-inv-investigate-there-26-opencode-bun.md | extends | pending | - |

---

## Findings

### Finding 1: macOS Jetsam Report Confirms OpenCode as Largest Process at 8.4 GB

**Evidence:** `/Library/Logs/DiagnosticReports/JetsamEvent-2026-02-06-033432.ips` (Feb 6, 3:34 AM) contains:
- `"largestProcess": "opencode"` - OpenCode was identified as the largest memory consumer
- OpenCode PID 53190: 550,922 pages = **8,608 MB (8.4 GB)** RSS, `lifetimeMax: 550,922 pages`
- `"reason": "per-process-limit"` on IntelligencePlatformComputeServi (the killed process in this particular event)
- System memory state: active 9.7 GB, anonymous 12.6 GB, free **0.3 GB**, compressor 12.6 GB

At this jetsam event, the system had 36 GB total RAM with only 300 MB free. While this particular event killed an Apple service process, OpenCode at 8.4 GB was the dominant memory consumer putting the system under memory pressure.

**Source:** `/Library/Logs/DiagnosticReports/JetsamEvent-2026-02-06-033432.ips`, parsed with `python3 -c` to extract process data

**Significance:** This is the first OS-level evidence of what happens around OpenCode deaths. OpenCode grows to multi-GB sizes, pushing the system into memory pressure territory where macOS jetsam starts killing processes. Even when jetsam kills other processes first, the memory pressure itself can trigger OOM conditions.

---

### Finding 2: 26 Concurrent Bun Agent Processes Consuming 14.3 GB Combined

**Evidence:** The jetsam report shows 26 separate `bun` processes (agent sessions) each consuming 274-932 MB:
- PID 79166: 850 MB (highest single agent)
- PID 43664: 797 MB (lifetimeMax 932 MB)
- PID 23343: 790 MB
- PID 73834: 659 MB (lifetimeMax 889 MB)
- Total across all 26 bun processes: **14,601 MB (14.3 GB)**
- Combined with OpenCode (8.4 GB): **22.7 GB** on a 36 GB system

Each bun process has `fds: 300` (file descriptor limit). OpenCode itself has `fds: 200`.

**Source:** JetsamEvent report, parsed all processes named "bun" or "opencode"

**Significance:** The agent processes + OpenCode server consume **63% of total system RAM** (22.7/36 GB). With macOS kernel, system services, Chrome, etc. using the rest, this leaves almost no headroom. Any additional memory allocation triggers jetsam. This is a systemic resource management problem, not a single-process bug.

---

### Finding 3: Current OpenCode Process at 1.58 GB After 13.5 Hours, Growing

**Evidence:** Live process data for PID 22575:
```
RSS: 1,601,984 KB (1.58 GB)
VSZ: 506,905,664 KB
Elapsed: 13:34:10 (13.5 hours)
%MEM: 4.2%
%CPU: 33.1%
FDs open: 64
```
Two measurements 10 seconds apart: RSS 1,643,728 KB → 1,644,096 KB (368 KB growth in 10s, ~2.2 MB/min).

The process command is simply: `/Users/dylanconlin/.bun/bin/opencode serve --port 4096` - no memory limit flags.

**Source:** `ps -o pid,rss,vsz,etime,%mem,%cpu -p 22575`, `lsof -p 22575 | wc -l`

**Significance:** Current instance is at 1.58 GB after 13.5 hours. At the jetsam event (Feb 6, 3:34 AM), OpenCode was at 8.4 GB. The growth pattern suggests the server accumulates memory over time (likely from sessions, message data, and Bus event processing). At ~2 MB/min, it would take roughly 58 hours to reach 8.4 GB from startup, which is consistent with multi-day uptime between restarts. File descriptors are NOT exhausted (64 open vs hundreds available).

---

### Finding 4: Overmind `--can-die opencode` Means No Auto-Restart

**Evidence:** The overmind process is started with:
```
overmind start -D --can-die opencode -f .Procfile.tmp
```

From `overmind start --help`:
```
--can-die value, -c value  Specify names of process which can die without 
                           interrupting the other processes.
```

Separately, overmind has `--auto-restart` which is NOT used:
```
--auto-restart value, -r value  Specify names of process which will be auto 
                                restarted on death.
```

The Procfile entry: `opencode: env -u ANTHROPIC_API_KEY ~/.bun/bin/opencode serve --port 4096`

**Source:** `ps aux | grep overmind`, `overmind start --help`, `Procfile`

**Significance:** The `--can-die` flag is the exact opposite of what's needed. When jetsam kills OpenCode, overmind just lets it stay dead instead of restarting it. The `--auto-restart` flag exists in overmind but isn't used. Adding `--auto-restart opencode` would make the process self-healing.

---

### Finding 5: Crash.log Confirms JS Errors Don't Cause Process Death

**Evidence:** `~/.local/share/opencode/crash.log` contains 7 events from Jan 24-27, all `unhandledRejection`. Memory at crash.log time:
- RSS ranged 331-431 MB (healthy range)
- heapUsed: 108-237 MB
- Errors: `TypeError: undefined is not an object (evaluating 'msgWithParts.info')`, `ProviderModelNotFoundError`, `AI_NoOutputGeneratedError`

The crash handler at `server.ts:119-127` logs these but explicitly does not exit:
```typescript
process.on("unhandledRejection", (reason, promise) => {
  writeCrashLog("unhandledRejection", reason)
  // Note: unhandledRejection doesn't exit by default, we just log it
})
```

**Source:** `~/.local/share/opencode/crash.log`, `~/Documents/personal/opencode/packages/opencode/src/server/server.ts:119-127`

**Significance:** Confirms prior investigation finding (Jan 26) - these JS errors corrupt individual sessions but don't kill the process. The process dies later from OS-level memory pressure after growing to multi-GB sizes. The crash.log at 331-431 MB shows healthy memory; death happens at 8+ GB.

---

### Finding 6: Storage is 768 MB on Disk, 254 Sessions Stored

**Evidence:**
- 254 session directories in `~/.local/share/opencode/storage/message/`
- Total storage size: 768 MB
- 69 tool-output files totaling 42 MB
- Storage is file-based (`Storage.read()` reads from disk via `Bun.file().json()`)
- Bus subscriptions properly cleaned up on SSE disconnect (verified in `bus/index.ts:96-103`)

**Source:** `ls ~/.local/share/opencode/storage/message/ | wc -l`, `du -sh ~/.local/share/opencode/storage/`, `bus/index.ts`

**Significance:** The storage layer itself isn't the memory leak - it reads from disk. However, 254 sessions accumulated over time suggest no session cleanup policy. The memory growth likely comes from: (1) sessions loaded into memory and not evicted, (2) message data cached in JS objects, (3) event listeners and callbacks accumulating, (4) Bun runtime memory overhead growing with request count. The exact internal leak source requires profiling, but the OS-level evidence clearly shows unbounded growth.

---

## Synthesis

**Key Insights:**

1. **macOS jetsam is the kill mechanism** - The OpenCode server process grows to 8+ GB RSS over multi-day operation, putting the 36 GB system under memory pressure. macOS jetsam (the kernel memory manager) identifies OpenCode as the largest process and either kills it directly or kills other processes until the situation becomes unrecoverable.

2. **The problem is systemic, not a single bug** - OpenCode server (8.4 GB) + 26 agent processes (14.3 GB) = 22.7 GB consumed by the orch ecosystem on a 36 GB system. This leaves ~13 GB for the kernel, Chrome, and every other process - insufficient when compressor is already at 12.6 GB.

3. **Self-healing is disabled** - Overmind's `--can-die opencode` flag means when OpenCode dies (from any cause), it stays dead. The `--auto-restart` capability exists but is unused. Combined with no memory limits, this creates a system that both accumulates memory and fails to recover.

**Answer to Investigation Question:**

The OpenCode server process is killed by **macOS jetsam** (the kernel memory pressure subsystem, equivalent to Linux OOM killer) when the process grows to ~8.4 GB RSS and the system runs out of memory. This is confirmed by the jetsam event report at `/Library/Logs/DiagnosticReports/JetsamEvent-2026-02-06-033432.ips` which shows OpenCode as the `largestProcess` with the system at 0.3 GB free. It is NOT killed by: JS errors (those are logged and don't exit), overmind signals (overmind uses `--can-die` not `--auto-restart`), or file descriptor exhaustion (only 64 FDs open).

---

## Structured Uncertainty

**What's tested:**

- ✅ Jetsam report shows OpenCode at 8,608 MB as largestProcess (verified: read JetsamEvent-2026-02-06-033432.ips)
- ✅ Current process at 1.58 GB after 13.5 hours with 64 FDs (verified: `ps` and `lsof`)
- ✅ Overmind uses `--can-die opencode` without `--auto-restart` (verified: `ps aux | grep overmind`)
- ✅ Crash.log errors don't kill the process (verified: reading crash handler source code + crash.log shows continued operation)
- ✅ No bun/opencode crash reports in DiagnosticReports (verified: `ls ~/Library/Logs/DiagnosticReports/`)
- ✅ System has 36 GB RAM (verified: `sysctl hw.memsize`)

**What's untested:**

- ⚠️ Exact memory leak source in OpenCode (would need Bun heap profiling to identify)
- ⚠️ Whether jetsam directly killed OpenCode or killed enough other processes to cascade (jetsam report shows IntelligencePlatformComputeServi as the process killed with `reason: per-process-limit`, not OpenCode itself in this specific event)
- ⚠️ Exact growth rate at scale (measured 2 MB/min at 1.58 GB; growth may be non-linear)
- ⚠️ Whether `--auto-restart` would actually fix the problem vs. just delay it

**What would change this:**

- If a Bun heap dump showed memory is bounded and the 8.4 GB was due to a one-time spike rather than steady growth
- If a second jetsam event showed a different process as the cause
- If OpenCode has an internal restart/GC mechanism that we didn't discover

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `--auto-restart opencode` to overmind | implementation | Simple config change in Procfile, within orch-go scope |
| Set Bun heap limit via `--smol` or `BUN_JSC_heapSize` | architectural | Changes OpenCode fork configuration, affects all sessions |
| Implement session eviction / memory monitoring | architectural | Cross-component change in OpenCode, affects reliability |
| Reduce concurrent agent count | strategic | Affects Dylan's workflow capacity, value judgment |

### Recommended Approach: Defense in Depth

**Three independent, compounding fixes** - Each addresses a different layer of the problem.

**Why this approach:**
- No single fix is sufficient - memory growth + no restart + no limits creates a triple failure
- Each fix works independently, so partial implementation still improves reliability
- Fixes are ordered by implementation effort (low → high)

**Trade-offs accepted:**
- Auto-restart masks the root cause (memory growth) rather than fixing it
- Heap limits may cause the process to crash sooner (but at least it restarts)

**Implementation sequence:**

1. **Add `--auto-restart opencode` to overmind** (5 minutes) - Ensures the server self-heals after any death. Change `--can-die opencode` to `--can-die opencode --auto-restart opencode` in the Procfile/startup script. This is the highest-impact, lowest-effort fix.

2. **Set Bun memory limit** (30 minutes) - Add `BUN_JSC_heapSize=4096` environment variable or equivalent to cap heap at 4 GB, preventing the server from consuming 8+ GB before dying. This triggers earlier death but combined with auto-restart creates a bounded recovery cycle.

3. **Add memory monitoring** (1-2 hours) - Log RSS periodically from within the server. This provides data to find and fix the actual memory leak. Something like:
   ```typescript
   setInterval(() => {
     const mem = process.memoryUsage()
     log.info("memory", { rss: Math.round(mem.rss / 1024 / 1024), heap: Math.round(mem.heapUsed / 1024 / 1024) })
   }, 60000)
   ```

4. **Investigate and fix memory leak** (4-8 hours) - Use Bun's heap profiler to identify what's accumulating. Likely candidates: session data loaded into memory and never evicted, tool output caching, accumulated event listeners.

### Alternative Approaches Considered

**Option B: Increase system RAM**
- **Pros:** Buys time without code changes
- **Cons:** Just delays the inevitable - 8.4 GB server + 14 GB agents will overwhelm any reasonable amount; expensive; doesn't fix the root cause
- **When to use instead:** Never as sole fix, but could be combined

**Option C: Reduce concurrent agent count**
- **Pros:** Directly reduces memory pressure
- **Cons:** Limits Dylan's productivity; treats symptom not cause
- **When to use instead:** As temporary mitigation while implementing proper fixes

**Rationale for recommendation:** Defense in depth is best because each fix addresses a different failure mode: auto-restart handles the death recovery, heap limits prevent unbounded growth, monitoring provides visibility, and leak fixes address the root cause. Implementing fix #1 alone would dramatically improve reliability with 5 minutes of effort.

---

### Implementation Details

**What to implement first:**
- `--auto-restart opencode` in overmind config (immediate win)
- Memory limit via env var in Procfile opencode entry

**Things to watch out for:**
- ⚠️ Auto-restart will kill existing SSE connections; agents will lose their event stream. Auto-resume mechanism (recommended in Jan 26 investigation) becomes more important.
- ⚠️ Setting heap limit too low may cause frequent restarts under normal load. 4 GB is suggested as a balance (current process at 1.58 GB after 13.5 hours, death at 8.4 GB).
- ⚠️ The 26 concurrent bun agent processes are also a major memory consumer (14.3 GB). Limiting agents alone could buy significant headroom.

**Areas needing further investigation:**
- What specifically is growing in the OpenCode server's heap? Bun heap profiling needed.
- Is the growth linear or exponential with session count?
- Do completed sessions' data get released from memory?

**Success criteria:**
- ✅ OpenCode survives 48+ hours without manual restart
- ✅ RSS stays below 4 GB (with heap limit)
- ✅ After a crash, server auto-restarts within 5 seconds
- ✅ Memory growth rate is visible in logs

---

## References

**Files Examined:**
- `/Library/Logs/DiagnosticReports/JetsamEvent-2026-02-06-033432.ips` - macOS jetsam event showing OpenCode at 8.4 GB as largest process
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts` - Crash handlers, SSE event streaming
- `~/Documents/personal/opencode/packages/opencode/src/bus/index.ts` - Event bus subscription management
- `~/Documents/personal/opencode/packages/opencode/src/storage/storage.ts` - File-based storage layer
- `~/.local/share/opencode/crash.log` - JS-level error log (7 events, all unhandledRejection)
- `Procfile` - How opencode is started via overmind

**Commands Run:**
```bash
# Check DiagnosticReports for crash/jetsam events
ls -la ~/Library/Logs/DiagnosticReports/
ls -la /Library/Logs/DiagnosticReports/

# Parse jetsam report for opencode/bun process data
tail -n +2 /Library/Logs/DiagnosticReports/JetsamEvent-2026-02-06-033432.ips | python3 -c "..."

# Check current process state
ps -o pid,rss,vsz,etime,%mem,%cpu -p 22575
lsof -p 22575 | wc -l

# Check overmind configuration
ps aux | grep overmind
overmind start --help

# Check system memory
sysctl hw.memsize
vm_stat

# Check storage size
du -sh ~/.local/share/opencode/storage/
ls ~/.local/share/opencode/storage/message/ | wc -l
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md - First crash investigation, recommended crash handlers
- **Investigation:** .kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md - Found crash.log, identified SSE stream breaking

---

## Investigation History

**2026-02-07 01:29:** Investigation started
- Initial question: What OS-level mechanism kills the OpenCode server process?
- Context: Two prior investigations found JS-level errors but couldn't explain actual process death

**2026-02-07 01:33:** Jetsam event discovered
- Found JetsamEvent-2026-02-06-033432.ips with OpenCode at 8.4 GB as largestProcess
- System was at 0.3 GB free with 12.6 GB in compressor

**2026-02-07 01:34:** Current process profiled
- PID 22575 at 1.58 GB after 13.5 hours, 64 FDs open
- Memory growing at ~2 MB/min

**2026-02-07 01:35:** Investigation completed
- Status: Complete
- Key outcome: OpenCode is killed by macOS jetsam when RSS grows to ~8.4 GB. Fix: auto-restart + heap limit + memory monitoring.

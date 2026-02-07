<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch doctor --fix` failed to start OpenCode because `startOpenCode()` hid errors, used wrong binary path, and didn't handle zombie processes or port conflicts.

**Evidence:** Code review showed output redirected to /dev/null; Procfile uses `~/.bun/bin/opencode` and `env -u ANTHROPIC_API_KEY` but startOpenCode() didn't; found 7+ orphaned overmind tmux processes and stale socket files.

**Knowledge:** Startup failures need diagnostic output; zombie processes can hold ports; matching Procfile's environment (OAuth stealth mode) is critical for reliable starts.

**Next:** Deploy improved `startOpenCode()` which adds pre-flight checks, zombie cleanup, proper binary path, env var matching, and startup log capture.

**Promote to Decision:** recommend-no (bug fix, not architectural)

---

# Investigation: Opencode Server 4096 Not Responding

**Question:** Why does `orch doctor --fix` fail to start OpenCode server when 4096 is not responding, and how can it be fixed?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** Worker agent (architect)
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None
**Extracted-From:** Issue orch-go-21019
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: startOpenCode() hides all startup errors

**Evidence:** The original implementation redirected all output:
```go
cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
```
This means any startup error (port conflict, missing binary, OAuth failure) is silently swallowed.

**Source:** `cmd/orch/doctor.go:554` (original)

**Significance:** When the 10-second polling timeout expires, the error message "OpenCode started but not responding after 10s" provides no actionable diagnostic information. Users cannot determine why the server failed to start.

---

### Finding 2: Wrong binary path and missing environment variables

**Evidence:** The Procfile uses:
```
opencode: env -u ANTHROPIC_API_KEY ~/.bun/bin/opencode serve --port 4096
```
But startOpenCode() used:
- Plain `opencode` instead of `~/.bun/bin/opencode` (may not be in PATH)
- Did not unset ANTHROPIC_API_KEY (required for OAuth stealth mode)

**Source:** `Procfile:4`, `cmd/orch/doctor.go:554` (original)

**Significance:** If `opencode` is not in PATH, the command fails silently. Without unsetting ANTHROPIC_API_KEY, the server may fail authentication differently than expected, especially when using Max subscription via OAuth.

---

### Finding 3: Zombie processes and stale overmind state

**Evidence:** Found multiple orphaned processes:
- 7+ overmind tmux processes spanning multiple days
- OpenCode server (PID 85505) was NOT managed by the running overmind
- Overmind socket (.overmind.sock) doesn't exist despite overmind process (PID 59329) showing it in lsof

```bash
pgrep -f overmind  # Returns 7 PIDs
ls /Users/dylanconlin/Documents/personal/orch-go/.overmind.sock  # No such file
```

**Source:** `ps aux | grep overmind`, `lsof -p 59329 | grep overmind`

**Significance:** When port 4096 is held by a zombie opencode process, `orch doctor --fix` cannot start a new server without first cleaning up. The stale overmind state creates confusion about which process manager is responsible.

---

### Finding 4: No pre-flight check for port conflicts

**Evidence:** startOpenCode() did not check if something was already listening on port 4096 before attempting to start. This leads to silent startup failures when:
- A zombie opencode process is holding the port
- Another application is using port 4096
- A previous server crashed but the port hasn't been released

**Source:** `cmd/orch/doctor.go:550-570` (original)

**Significance:** A simple TCP connect check before startup would immediately diagnose "port in use" vs "server won't start" failures.

---

## Synthesis

**Key Insights:**

1. **Silent failures mask real problems** - Redirecting startup output to /dev/null trades "clean output" for "zero debuggability". Any failure mode becomes "not responding after 10s" with no actionable information.

2. **Environment consistency matters** - The Procfile configuration (OAuth stealth mode, binary path) represents tested, working setup. Diverging from it in `orch doctor --fix` creates a different failure mode than normal operation.

3. **Zombie process accumulation is a symptom** - The 7+ orphaned overmind processes indicate that process lifecycle management in orch-go has gaps. While this fix addresses immediate recovery, the root cause of zombie accumulation needs separate attention.

**Answer to Investigation Question:**

`orch doctor --fix` fails to start OpenCode because the `startOpenCode()` function had four issues:
1. Hidden errors (output to /dev/null)
2. Wrong binary path (plain `opencode` vs `~/.bun/bin/opencode`)
3. Missing env var unsetting (ANTHROPIC_API_KEY for OAuth stealth mode)
4. No handling of zombie processes holding port 4096

The fix implements: pre-flight port check, zombie process cleanup, correct binary path with fallback, matching env vars, and startup log capture for diagnostics.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles (verified: `go build` succeeds)
- ✅ OpenCode server responds when running (verified: `curl localhost:4096/session` returns sessions)
- ✅ Binary path fallback logic (verified: code paths for ~/.bun/bin and PATH lookup)

**What's untested:**

- ⚠️ Zombie cleanup actually frees the port (not reproducible without killing current server)
- ⚠️ Startup log captures meaningful errors (need actual failure to validate)
- ⚠️ OAuth stealth mode works correctly with env var unset (matches Procfile but not directly tested)

**What would change this:**

- If OpenCode binary requires ANTHROPIC_API_KEY in some scenarios, the env unset could break those flows
- If zombie cleanup sends SIGTERM but processes don't exit, SIGKILL escalation may be needed
- If startup takes >15s legitimately, the timeout would need increasing

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Improved startOpenCode() with pre-flight checks and diagnostics** - The fix adds port checking, zombie cleanup, proper binary/env, and startup log capture.

**Why this approach:**
- Directly addresses all four root causes identified
- Preserves existing behavior when server starts successfully
- Adds diagnostics without changing normal operation
- Matches Procfile configuration for consistency

**Trade-offs accepted:**
- Adds complexity to startOpenCode() (from ~20 to ~70 lines)
- Creates a new log file (~/.local/share/opencode/startup.log)
- SIGTERM to zombies may not be sufficient in all cases

**Implementation sequence:**
1. Pre-flight port check - determines if zombie cleanup is needed
2. Binary path resolution - uses ~/.bun/bin/opencode with PATH fallback
3. Env var matching - unsets ANTHROPIC_API_KEY like Procfile
4. Startup with log capture - writes to startup.log instead of /dev/null
5. Extended timeout - 15s instead of 10s for slow starts

### Alternative Approaches Considered

**Option B: Use overmind restart**
- **Pros:** Leverages existing process manager, matches normal operation
- **Cons:** Requires working overmind socket; found socket often missing/stale
- **When to use instead:** When overmind infrastructure is reliable

**Option C: Kill and restart via launchd**
- **Pros:** System-level process management, more robust
- **Cons:** Requires launchd integration, more complex setup
- **When to use instead:** If zombie accumulation becomes frequent

**Rationale for recommendation:** Direct startup is simpler and doesn't depend on external state (overmind socket). The fix makes startOpenCode() self-sufficient.

---

### Implementation Details

**What to implement first:**
- The fix is already implemented in `cmd/orch/doctor.go`
- Run `make install` to rebuild orch binary

**Things to watch out for:**
- ⚠️ Startup log rotation - startup.log will grow unbounded over time
- ⚠️ SIGTERM may not be enough for truly stuck processes
- ⚠️ 15-second timeout may still be too short for cold starts with many sessions

**Areas needing further investigation:**
- Root cause of overmind zombie accumulation (separate issue)
- Auto-resume mechanism for sessions after server restart (see 2026-01-17 investigation)
- Memory profiling under load (see 2026-01-23 investigation)

**Success criteria:**
- ✅ `orch doctor --fix` starts OpenCode when server is down
- ✅ Startup failures produce diagnostic output in startup.log
- ✅ Zombie opencode processes are cleaned up before new server starts
- ✅ Server uses OAuth stealth mode (matches Procfile behavior)

---

## References

**Files Examined:**
- `cmd/orch/doctor.go` - startOpenCode() implementation
- `Procfile` - canonical OpenCode startup configuration
- `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - prior crash investigation
- `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - load crash investigation
- `.kb/guides/opencode.md` - OpenCode integration guide

**Commands Run:**
```bash
# Check OpenCode server status
curl -s http://127.0.0.1:4096/session | head -20

# Check process state
ps aux | grep opencode
ps aux | grep overmind
lsof -i :4096
lsof -p 59329 | grep overmind

# Check overmind socket
ls -la /Users/dylanconlin/Documents/personal/orch-go/.overmind.sock

# Test build
go build -o /tmp/orch-test ./cmd/orch
```

**External Documentation:**
- None - internal orch-go infrastructure

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - Prior analysis of crash causes
- **Investigation:** `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Load-related crash analysis
- **Guide:** `.kb/guides/opencode.md` - OpenCode integration patterns

---

## Investigation History

**2026-01-29 12:44:** Investigation started
- Initial question: Why does `orch doctor --fix` fail to recover OpenCode server?
- Context: Issue orch-go-21019 reported connection refused after restart attempts

**2026-01-29 12:50:** Found hidden error output
- startOpenCode() redirects all output to /dev/null
- No way to diagnose startup failures

**2026-01-29 12:55:** Identified environment mismatch
- Procfile uses ~/.bun/bin/opencode and env -u ANTHROPIC_API_KEY
- startOpenCode() used plain `opencode` without env modification

**2026-01-29 13:00:** Discovered zombie process accumulation
- 7+ orphaned overmind processes
- Stale overmind socket (exists in lsof but not on filesystem)

**2026-01-29 13:10:** Implemented fix
- Added pre-flight port check
- Added zombie process cleanup
- Added proper binary path resolution
- Added startup log capture

**2026-01-29 13:20:** Investigation completed
- Status: Complete
- Key outcome: Improved startOpenCode() with diagnostics and zombie handling

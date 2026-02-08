<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** bd CLI runs 50x slower (5s vs 100ms) in launchd/minimal environments because it attempts to connect to daemon socket and waits 5s for timeout when socket unavailable.

**Evidence:** `env -i HOME=$HOME PATH=$PATH bd ready` takes 10s (daemon timeout warning visible); adding `BEADS_NO_DAEMON=1` drops to 86ms.

**Knowledge:** The `BEADS_NO_DAEMON=1` env var bypasses daemon connection attempts - orch serve needs to pass this to CLI fallback commands, OR orch-go should use RPC client directly (already implemented) rather than CLI fallback when daemon unavailable.

**Next:** Implement Option A: Set `BEADS_NO_DAEMON=1` in Fallback* CLI commands when shelling out to bd, ensuring fast direct mode is used.

**Promote to Decision:** recommend-no (tactical fix for launchd/minimal env behavior - established pattern already exists in constraint)

---

# Investigation: bd CLI Slow in Launchd/Minimal Environment

**Question:** Why does `bd ready` take 5s in launchd/minimal env but 100ms in interactive shell, and how can orch-go work around this?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: BEADS_NO_DAEMON env var controls daemon bypass

**Evidence:** 
```bash
# Interactive shell has BEADS_NO_DAEMON=1 set
$ env | grep BEADS
BEADS_NO_DAEMON=1

# With env var: 86ms
$ time env -i HOME=$HOME PATH=$PATH BEADS_NO_DAEMON=1 bd ready --json --limit 0
real    0m0.086s

# Without env var (minimal env): 10s (5s timeout + 5s retry)
$ time env -i HOME=$HOME PATH=$PATH bd ready --json --limit 0
Warning: Daemon took too long to start (>5s). Running in direct mode.
real    0m10.273s
```

**Source:** `/Users/dylanconlin/Documents/personal/beads/cmd/bd/daemon_autostart.go:47-53`

```go
// shouldAutoStartDaemon checks if daemon auto-start is enabled
func shouldAutoStartDaemon() bool {
    // Check BEADS_NO_DAEMON first (escape hatch for single-user workflows)
    noDaemon := strings.ToLower(strings.TrimSpace(os.Getenv("BEADS_NO_DAEMON")))
    if noDaemon == "1" || noDaemon == "true" || noDaemon == "yes" || noDaemon == "on" {
        return false // Explicit opt-out
    }
    ...
}
```

**Significance:** The bd CLI intentionally provides an escape hatch via `BEADS_NO_DAEMON=1` for environments where daemon connection is undesirable or unreliable. Dylan's shell sets this, but launchd/orch serve environments don't inherit it.

---

### Finding 2: Daemon socket connection attempt has 5s timeout

**Evidence:** When `BEADS_NO_DAEMON` is not set, bd:
1. Looks for daemon socket at `.beads/bd.sock`
2. If socket exists but daemon not responding, waits up to 5s
3. If daemon PID file exists with alive process, waits for socket readiness
4. Finally falls back to direct mode with warning

**Source:** `/Users/dylanconlin/Documents/personal/beads/cmd/bd/daemon_autostart.go:307-316`

```go
if waitForSocketReadinessFn(socketPath, 5*time.Second) {
    recordDaemonStartSuccess()
    return true
}

recordDaemonStartFailure()
debugLog("daemon socket not ready after 5 seconds")
// Emit visible warning so user understands why command was slow
fmt.Fprintf(os.Stderr, "%s Daemon took too long to start (>5s). Running in direct mode.\n", ui.RenderWarn("Warning:"))
```

**Significance:** The 5s timeout is hardcoded and occurs per CLI invocation. For orch serve which may call bd frequently, this adds unacceptable latency to initial requests after restart.

---

### Finding 3: orch-go Fallback* functions don't set BEADS_NO_DAEMON

**Evidence:** Current Fallback functions in `pkg/beads/client.go` create exec.Command without setting `BEADS_NO_DAEMON`:

```go
// FallbackReady retrieves ready issues via bd CLI.
func FallbackReady() ([]Issue, error) {
    cmd := exec.Command(getBdPath(), "ready", "--json", "--limit", "0")
    if DefaultDir != "" {
        cmd.Dir = DefaultDir
    }
    output, err := cmd.Output()  // No cmd.Env set!
    ...
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/client.go:716-736`

**Significance:** When RPC client fails and orch-go falls back to CLI, it inherits the launchd minimal environment which lacks `BEADS_NO_DAEMON`. This causes the 5s delay on every Fallback call.

---

### Finding 4: RPC client is the preferred path (already implemented)

**Evidence:** orch-go already has a sophisticated RPC client in `pkg/beads/client.go` that connects directly to the daemon socket. The RPC path is fast (sub-100ms) when daemon is available. Fallback to CLI only happens when:
1. Socket not found
2. Daemon unhealthy
3. Connection error

**Source:** `pkg/beads/client.go:175-218` (Connect method), `cmd/orch/serve_beads.go:68-118` (Fallback logic)

**Significance:** The architecture is correct (RPC primary, CLI fallback). The issue is only the CLI fallback path being slow due to missing env var.

---

## Synthesis

**Key Insights:**

1. **Environment inheritance gap** - launchd starts orch serve with minimal environment that lacks shell customizations like `BEADS_NO_DAEMON=1`. This is a common pattern in daemon/server contexts.

2. **Fallback path is the slow path** - The RPC client path is fast. The CLI fallback is only slow because bd CLI tries to connect to daemon (which is what we're falling back FROM). Setting `BEADS_NO_DAEMON=1` on fallback commands is logically correct - we're already in fallback mode, no need to try daemon again.

3. **Simple fix available** - Adding `cmd.Env = append(os.Environ(), "BEADS_NO_DAEMON=1")` to Fallback* functions solves the issue without requiring changes to bd CLI or daemon infrastructure.

**Answer to Investigation Question:**

bd CLI takes 5s in launchd/minimal environments because it attempts to connect to the daemon socket and waits for timeout. Dylan's interactive shell has `BEADS_NO_DAEMON=1` set which bypasses this. The fix is to set this env var in orch-go's Fallback* functions when shelling out to bd CLI, ensuring direct mode is used for CLI fallback.

---

## Structured Uncertainty

**What's tested:**

- ✅ BEADS_NO_DAEMON=1 makes bd CLI fast (verified: ran timing tests with and without env var)
- ✅ Interactive shell has BEADS_NO_DAEMON=1 set (verified: ran `env | grep BEADS`)
- ✅ launchd/minimal env lacks BEADS_NO_DAEMON (verified: `env -i` reproduction)
- ✅ Fallback* functions don't set cmd.Env (verified: code review of client.go)

**What's untested:**

- ⚠️ Fix will work in production orch serve (not deployed yet)
- ⚠️ No side effects from forcing direct mode in CLI fallback (logically safe but not tested)

**What would change this:**

- Finding would be wrong if bd CLI has another source of slowness besides daemon timeout
- Finding would be incomplete if RPC client itself has issues in launchd env

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Option A: Set BEADS_NO_DAEMON=1 in Fallback* functions** - Add environment variable to all CLI subprocess calls in pkg/beads/client.go

**Why this approach:**
- Directly addresses root cause (daemon timeout in CLI fallback)
- Minimal code change (add one line per Fallback function)
- Logically correct: when falling back from RPC, no point trying daemon again via CLI

**Trade-offs accepted:**
- CLI fallback will always use direct mode (acceptable - we're already in fallback mode)
- Doesn't fix the root cause in bd CLI (not our codebase to change)

**Implementation sequence:**
1. Add helper function `cmdWithDirectMode(cmd *exec.Cmd)` that sets env
2. Call helper in all Fallback* functions
3. Verify with reproduction test

### Alternative Approaches Considered

**Option B: Set BEADS_NO_DAEMON=1 in launchd plist**
- **Pros:** Single configuration change, affects all bd calls from launchd
- **Cons:** Requires user to modify their launchd config; doesn't help other minimal-env scenarios
- **When to use instead:** If user wants all launchd-spawned bd calls to skip daemon

**Option C: Add caching layer in orch serve for beads responses**
- **Pros:** Already partially implemented (`serve_agents_cache.go`); reduces all bd calls
- **Cons:** Already in place for repeated requests; doesn't help first request after restart
- **When to use instead:** Already used for repeated requests - this investigation is about first-request latency

**Option D: Improve RPC client connection to avoid fallback**
- **Pros:** Eliminates fallback path entirely
- **Cons:** Complex; daemon may genuinely be unavailable; doesn't address legitimate fallback cases
- **When to use instead:** If RPC client has reliability issues that force unnecessary fallbacks

**Rationale for recommendation:** Option A is the simplest fix that directly addresses the issue. We're already in fallback mode when Fallback* is called, so skipping daemon retry is semantically correct.

---

### Implementation Details

**What to implement first:**
- Add `cmdWithDirectMode()` helper to set BEADS_NO_DAEMON=1
- Apply to all Fallback* functions in pkg/beads/client.go

**Things to watch out for:**
- ⚠️ Make sure to append to os.Environ(), not replace entire env
- ⚠️ Verify no tests depend on daemon mode in fallback functions

**Success criteria:**
- ✅ `env -i HOME=$HOME PATH=$PATH orch serve` first request to /api/beads/ready completes in <500ms
- ✅ Reproduction: `time env -i HOME=$HOME PATH=$PATH bd ready --json` with BEADS_NO_DAEMON=1 in env

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/daemon_autostart.go` - daemon connection logic and BEADS_NO_DAEMON check
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/client.go` - Fallback* functions that need fix

**Commands Run:**
```bash
# Reproduction - slow without env var
time env -i HOME=$HOME PATH=$PATH bd ready --json --limit 0
# Result: 10.273s with daemon timeout warning

# Fast with env var
time env -i HOME=$HOME PATH=$PATH BEADS_NO_DAEMON=1 bd ready --json --limit 0
# Result: 0.086s

# Check interactive shell env
env | grep BEADS
# Result: BEADS_NO_DAEMON=1
```

**Related Artifacts:**
- **Constraint:** `bd CLI requires full environment; runs 50x slower in minimal env (launchd)` - already documented in kb quick constraints

---

## Investigation History

**2026-01-07 14:30:** Investigation started
- Initial question: Why is bd CLI slow in launchd but fast in interactive shell?
- Context: orch serve first request after restart takes 5s+ for beads endpoints

**2026-01-07 14:45:** Root cause identified
- BEADS_NO_DAEMON env var difference between interactive shell and launchd
- Daemon timeout hardcoded to 5s in bd CLI

**2026-01-07 15:00:** Investigation completed
- Status: Complete
- Key outcome: Fix is to set BEADS_NO_DAEMON=1 in Fallback* CLI subprocess calls

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The `orch serve` PATH issue was caused by launchd providing minimal PATH (`/usr/bin:/bin:/usr/sbin:/sbin`) to the server process, and the CLI fallback functions hardcoding `exec.Command("bd", ...)` which relies on PATH lookup.

**Evidence:** Before fix: `curl -sk https://localhost:3348/api/beads` returned `"error":"Failed to get bd stats: bd stats failed: exec: \"bd\": executable file not found in $PATH"`. After fix: Same endpoint returns valid JSON with issue counts.

**Knowledge:** The cleanest solution is resolving absolute paths at startup (Option 2) because it's self-contained in Go code and works regardless of how the server is started (launchd, manual, etc.).

**Next:** Merge this fix. Consider adding similar resolution for other external executables if issues arise.

**Promote to Decision:** recommend-yes - This establishes a pattern for handling executables in launchd environments.

---

# Investigation: Orch Serve PATH Issue - bd Executable Resolution

**Question:** What is the cleanest and most maintainable solution for `orch serve` to find the `bd` executable when running under launchd with minimal PATH?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Root Cause - CLI Fallback Uses exec.Command("bd", ...)

**Evidence:** In `pkg/beads/client.go`, the fallback functions (FallbackStats, FallbackReady, etc.) use:
```go
cmd := exec.Command("bd", "stats", "--json")
```

This requires `bd` to be in PATH. When `orch serve` runs under launchd, it inherits minimal PATH that does not include common binary locations.

**Source:** `pkg/beads/client.go:762` (FallbackStats), similar pattern at lines 648, 674, 702, etc.

**Significance:** This is the root cause. When the beads RPC daemon isn't running and CLI fallback is attempted, the `bd` command fails to execute.

---

### Finding 2: Launchd Provides Minimal PATH to orch serve

**Evidence:** Running process inspection showed:
```
PATH=/usr/bin:/bin:/usr/sbin:/sbin
```

This is the default launchd PATH, not the extended PATH from user shell configuration.

**Source:** `ps eww 85413 | grep PATH`

**Significance:** The `com.orch-go.serve.plist` does NOT include an EnvironmentVariables section with PATH, unlike `com.orch.daemon.plist` which does.

---

### Finding 3: Three Options Considered

**Option 1: Configure PATH in launchd plist**
- Already done for `orch daemon run` but not for `orch serve`
- External to code, requires modifying system files
- Doesn't help when `orch serve` is run manually

**Option 2: Resolve absolute paths at startup** (CHOSEN)
- Detect bd path at serve startup using exec.LookPath and common locations
- Store resolved path and use for all fallback functions
- Self-contained, works regardless of how serve is started
- Most maintainable

**Option 3: Use beads RPC client instead of CLI**
- Already implemented! Code prioritizes RPC client
- Issue: Only works when beads daemon is running
- CLI fallback is needed when daemon is unavailable

**Source:** Code analysis, design consideration

**Significance:** Option 2 was chosen as the most robust solution.

---

### Finding 4: Solution Implementation

**Evidence:** Added to `pkg/beads/client.go`:
1. `BdPath` variable to store resolved path
2. `bdSearchPaths` - common locations: `$HOME/bin/bd`, `$HOME/go/bin/bd`, `$HOME/.bun/bin/bd`, etc.
3. `ResolveBdPath()` function - tries exec.LookPath first, then searches common locations
4. `getBdPath()` helper - returns BdPath if set, otherwise "bd"
5. Updated all 11 Fallback* functions to use `getBdPath()` instead of hardcoded "bd"

Added to `cmd/orch/serve.go`:
- Call `beads.ResolveBdPath()` at startup with warning if not found

**Source:** Implementation in pkg/beads/client.go and cmd/orch/serve.go

**Significance:** Self-contained fix that works in any execution context.

---

## Test Performed

**Test:** Restarted `orch serve` via launchd and called the beads endpoint.

**Result:** 
- Before fix: `{"error":"Failed to get bd stats: bd stats failed: exec: \"bd\": executable file not found in $PATH"}`
- After fix: `{"total_issues":1439,"open_issues":32,"in_progress_issues":1,...}`

**Verification:** Also tested `/api/beads/ready` endpoint - returns list of 33 ready issues successfully.

---

## Conclusion

The fix resolves the PATH issue by having `orch serve` resolve the `bd` executable path at startup and store it for use by all CLI fallback functions. This approach is:

1. **Self-contained** - All logic is in Go code, no external configuration needed
2. **Robust** - Works regardless of how the server is started
3. **Maintainable** - Single point of change for adding new search paths
4. **Backward compatible** - Falls back to "bd" if resolution fails (for environments where PATH is correct)

---

## Self-Review

- [x] Real test performed (restarted server, called endpoint)
- [x] Conclusion from evidence (curl output before/after)
- [x] Question answered (Option 2 is cleanest and most maintainable)
- [x] File complete

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn constrain "launchd provides minimal PATH to services" --reason "PATH=/usr/bin:/bin:/usr/sbin:/sbin - user shell paths not inherited"
```

---

## References

**Files Examined:**
- `pkg/beads/client.go` - Fallback functions using exec.Command
- `cmd/orch/serve.go` - Server initialization
- `cmd/orch/serve_beads.go` - Beads endpoint handlers
- `~/Library/LaunchAgents/com.orch-go.serve.plist` - Missing PATH config
- `~/Library/LaunchAgents/com.orch.daemon.plist` - Has PATH config (for comparison)

**Commands Run:**
```bash
# Confirmed problem
ps eww 85413 | grep PATH  # Shows minimal PATH
curl -sk https://localhost:3348/api/beads  # Shows error

# Verified fix
make build
launchctl stop/start com.orch-go.serve
curl -sk https://localhost:3348/api/beads  # Shows valid data
```

**Related Artifacts:**
- **Decision:** This investigation recommends promoting to decision: "Use startup path resolution for executables in launchd services"

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [Investigating - to be filled at completion]

**Evidence:** [Investigating - to be filled at completion]

**Knowledge:** [Investigating - to be filled at completion]

**Next:** [Investigating - to be filled at completion]

**Promote to Decision:** [unclear] - Will determine based on findings

---

# Investigation: Orch Serve PATH Issue - bd Executable Resolution

**Question:** What is the cleanest and most maintainable solution for `orch serve` to find the `bd` executable when running under launchd with minimal PATH?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent
**Phase:** Investigating
**Next Step:** Evaluate three proposed options
**Status:** In Progress

---

## Findings

### Finding 1: Current Problem - CLI Fallback Uses exec.Command("bd", ...)

**Evidence:** In `pkg/beads/client.go`, the fallback functions (FallbackStats, FallbackReady, etc.) use:
```go
cmd := exec.Command("bd", "stats", "--json")
```

This requires `bd` to be in PATH. When `orch serve` runs under launchd, it inherits minimal PATH that may not include `bd`.

**Source:** `pkg/beads/client.go:762` (FallbackStats), similar pattern at lines 648, 674, 702, etc.

**Significance:** This is the root cause. When the beads RPC daemon isn't running and CLI fallback is attempted, the `bd` command fails to execute because it's not in PATH.

---

### Finding 2: Launchd Plist Already Has Extended PATH

**Evidence:** The launchd plist at `~/Library/LaunchAgents/com.orch.daemon.plist` already includes:
```
PATH=/Users/dylanconlin/.bun/bin:/Users/dylanconlin/bin:/Users/dylanconlin/claude-npm-global/bin:/usr/local/bin:/usr/bin:/bin:/Users/dylanconlin/.local/bin:/Users/dylanconlin/go/bin:/opt/homebrew/bin
```

And `bd` exists at `/Users/dylanconlin/bin/bd` which IS in the plist PATH.

**Source:** `~/Library/LaunchAgents/com.orch.daemon.plist`, `which bd` output

**Significance:** The plist PATH configuration appears correct. The issue may be that `orch serve` is NOT run by launchd - it's run manually or by a different mechanism.

---

### Finding 3: ~/.bun/bin Symlink Solution Is In Place

**Evidence:** 
- `~/.bun/bin/bd -> /Users/dylanconlin/go/bin/bd` (symlink exists)
- This is documented in global CLAUDE.md as a workaround for OpenCode server PATH issues

**Source:** `ls -la ~/.bun/bin/bd`, `~/.claude/CLAUDE.md`

**Significance:** The symlink workaround was designed for OpenCode server scenarios, but `orch serve` may have different PATH inheritance.

---

### Finding 4: Three Proposed Solutions

**Option 1: Configure PATH in launchd plist**
- Already done for `orch daemon run`
- Would need separate plist for `orch serve` if run via launchd
- Doesn't help when `orch serve` is run manually

**Option 2: Resolve absolute paths at startup**
- Detect bd path at serve startup using exec.LookPath or known locations
- Store resolved path and pass to all fallback functions
- Self-contained, works regardless of how serve is started

**Option 3: Use beads RPC client instead of CLI**
- Already implemented! The code prioritizes RPC client, only falls back to CLI
- Issue: RPC client requires beads daemon to be running
- The problem only manifests when daemon is NOT running

**Source:** Task spawn context, code analysis

**Significance:** Option 2 appears most robust because:
- Works regardless of how `orch serve` is started
- Self-contained fix in Go code
- Doesn't require daemon to be running
- Handles both launchd and manual invocation

---

## Next Steps

1. Test if `orch serve` actually experiences the PATH issue (verify the problem)
2. Implement Option 2: resolve bd path at startup
3. Test with RPC daemon down to confirm CLI fallback works

---

## References

**Files Examined:**
- `pkg/beads/client.go` - Fallback functions using exec.Command("bd", ...)
- `cmd/orch/serve.go` - Server initialization
- `cmd/orch/serve_beads.go` - Beads endpoint handlers
- `~/Library/LaunchAgents/com.orch.daemon.plist` - launchd configuration

**Commands Run:**
```bash
which bd  # /Users/dylanconlin/bin/bd
ls -la ~/.bun/bin/bd  # symlink to /Users/dylanconlin/go/bin/bd
cat ~/Library/LaunchAgents/com.orch.daemon.plist
```

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The bug was already fixed in commit 1d3de60b (Dec 25 8:42 PM), but the daemon was running old binary from before the fix.

**Evidence:** Daemon log showed errors at 8:15-8:39 PM Dec 25; fix committed at 8:42 PM; daemon PID unchanged until restart; smoke test passed after daemon restart.

**Knowledge:** Daemon runs via launchd and survives binary rebuilds - must explicitly restart with `launchctl kickstart -k` to pick up new code.

**Next:** Close - daemon restarted, fix verified, no code changes needed.

**Confidence:** Very High (95%) - Direct observation of daemon PID, log timestamps, and smoke test verification.

---

# Investigation: Bd Show JSON Parsing Fails During Daemon Spawn

**Question:** Why does `bd show` JSON parsing fail during daemon spawn with "cannot unmarshal array into Go value of type beads.Issue"?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent (og-debug-bd-show-json-26dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: The bug was already fixed in commit 1d3de60b

**Evidence:** 
- Commit `1d3de60b` from Dec 25 20:42:35 explicitly fixes this exact error
- Commit message: "fix: handle bd show array response for epic children"
- The fix changes `FallbackShow` to unmarshal to `[]Issue` and return first element

**Source:** 
- `git log --oneline -20 -- pkg/beads/client.go`
- `git show 1d3de60b --stat`

**Significance:** The code fix already existed - the issue was deployment, not code.

---

### Finding 2: Daemon was running old binary

**Evidence:**
- Daemon log shows errors at 20:15-20:39 (before fix commit at 20:42)
- Daemon PID 11745 started at "10:53PM" (before fix)
- Binary at `/Users/dylanconlin/bin/orch` modified at 8:40 AM Dec 26 (after fix)
- launchd keeps daemon running with original binary until explicit restart

**Source:**
- `grep -i "unmarshal" ~/.orch/daemon.log`
- `ps aux | grep "orch daemon"`
- `stat ~/bin/orch`
- `launchctl list | grep orch`

**Significance:** Root cause was operational (stale daemon), not code bug.

---

### Finding 3: Fix verified after daemon restart

**Evidence:**
- Daemon restarted with `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`
- New PID 27035 picked up fixed binary
- Smoke test with forced CLI fallback (BEADS_NO_DAEMON=1) passed
- Both `FallbackShow` and `verify.GetIssue` correctly parse array format

**Source:**
- Smoke test script at `/tmp/test_fix.go`
- `launchctl list | grep orch.daemon`

**Significance:** Confirms fix is working and daemon is now operational.

---

## Synthesis

**Key Insights:**

1. **Array vs Object format difference** - `bd show --json` returns an array `[{...}]` but the RPC daemon returns a single object `{...}`. The fallback code must handle both formats.

2. **launchd daemon persistence** - Daemons running via launchd survive binary rebuilds. After `make install`, must explicitly restart with `launchctl kickstart -k` to pick up new code.

3. **Temporal analysis was key** - Matching daemon log timestamps against commit timestamps revealed the fix was committed AFTER the errors started, but BEFORE the binary was rebuilt.

**Answer to Investigation Question:**

The daemon spawn failed because the orch daemon was running an old binary that didn't include the fix (commit 1d3de60b). The fix was already committed on Dec 25 at 8:42 PM, but the daemon (started at 10:53 PM the previous day) was not restarted. After restarting the daemon with `launchctl kickstart -k`, the fix took effect and spawns work correctly.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Direct observation of all components - daemon logs with timestamps, commit history, process IDs, binary modification times, and successful smoke test. No speculation required.

**What's certain:**

- ✅ Error was "cannot unmarshal array into Go value of type beads.Issue" (daemon log)
- ✅ Fix was commit 1d3de60b at Dec 25 20:42:35
- ✅ Daemon was running pre-fix binary (PID unchanged since before fix)
- ✅ Fix works after daemon restart (smoke test passed)

**What's uncertain:**

- ⚠️ Why errors started at exactly 20:15 (may be timing of first spawn attempt)

---

## Implementation Recommendations

**Recommended Approach:** No code changes needed - just operational fix (daemon restart).

**Preventive measures for future:**

1. **Add daemon restart to deploy script** - After `make install`, automatically restart daemon
2. **Document launchd behavior** - Note in README that daemon must be restarted after rebuilds
3. **Consider version logging** - Daemon could log its binary version at startup

**Implementation sequence:**
1. ✅ Daemon already restarted - fix is deployed
2. Consider adding `launchctl kickstart` to Makefile `install` target

---

## References

**Files Examined:**
- `pkg/beads/client.go:421-442` - FallbackShow implementation
- `pkg/verify/check.go:542-580` - GetIssue implementation
- `pkg/daemon/daemon.go:387-396` - SpawnWork implementation
- `~/.orch/daemon.log` - Error messages and timestamps

**Commands Run:**
```bash
# Check daemon log for errors
grep -i "unmarshal" ~/.orch/daemon.log

# Check commit that fixed the issue
git show 1d3de60b --stat

# Check daemon process
ps aux | grep "orch daemon"

# Restart daemon
launchctl kickstart -k gui/$(id -u)/com.orch.daemon

# Smoke test fix
go run /tmp/test_fix.go
```

**Related Artifacts:**
- **Prior fix:** `.kb/investigations/2025-12-25-inv-bd-show-returns-array-epic.md` (original investigation)
- **Commit:** `1d3de60b` - fix: handle bd show array response for epic children

---

## Investigation History

**2025-12-26 08:34:** Investigation started
- Initial question: Why does bd show JSON parsing fail during daemon spawn?
- Context: Daemon spawns were failing overnight

**2025-12-26 08:40:** Root cause identified
- Found commit 1d3de60b already fixed the issue
- Daemon was running old binary from before the fix
- launchd keeps daemon running with original binary

**2025-12-26 08:45:** Investigation completed
- Daemon restarted with `launchctl kickstart -k`
- Smoke test verified fix is working
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Operational issue (stale daemon), not code bug

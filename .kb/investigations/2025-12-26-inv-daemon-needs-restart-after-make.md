<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `make install-restart` target and post-install reminder to guide users on daemon restart after binary updates.

**Evidence:** `make install` now prints reminder with launchctl command; `make install-restart` chains install with daemon kickstart.

**Knowledge:** The orch daemon runs from `~/bin/orch` which is replaced by `make install`, but the running process uses the old binary until restarted via launchctl kickstart.

**Next:** Close - implementation complete.

**Confidence:** High (90%) - Simple make target addition, tested locally.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Daemon Needs Restart After Make

**Question:** How to ensure the orch daemon picks up new binary after `make install`?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Daemon plist points to ~/bin/orch

**Evidence:** `com.orch.daemon.plist` has `ProgramArguments` pointing to `/Users/dylanconlin/bin/orch`

**Source:** `/Users/dylanconlin/Library/LaunchAgents/com.orch.daemon.plist:9-10`

**Significance:** This is the binary that gets replaced by `make install`. The running daemon process still uses the old binary in memory until restarted.

---

### Finding 2: Serve daemon uses build/orch to avoid SIGKILL

**Evidence:** `com.orch-go.serve.plist` points to `/Users/dylanconlin/Documents/personal/orch-go/build/orch`

**Source:** Prior decision: "Use build/orch for serve daemon - Reason: Prevents SIGKILL during make install"

**Significance:** The serve daemon is already immune to this problem because it runs from build/ not ~/bin/. The orch daemon could follow the same pattern but that would require plist changes.

---

### Finding 3: Simple Makefile enhancement is sufficient

**Evidence:** Added `install-restart` target and post-install reminder. Tested locally - works correctly.

**Source:** `Makefile:35-52`

**Significance:** Rather than changing the plist (which is managed separately), adding a convenience target and reminder keeps the solution simple and user-aware.

---

## Synthesis

**Key Insights:**

1. **User awareness is the simplest fix** - Rather than architectural changes (like pointing daemon plist to build/), a reminder after install guides users to restart when needed.

2. **Convenience target reduces friction** - `make install-restart` is a one-liner that chains install with daemon restart for developers who want the new binary active immediately.

**Answer to Investigation Question:**

The daemon now has proper guidance for picking up new binaries:
- `make install` prints a reminder with the launchctl command
- `make install-restart` automatically restarts the daemon after install
- The pattern matches how developers already work (build → install → use)

---

## References

**Files Modified:**
- `Makefile:35-52` - Added `install-restart` target and post-install reminder

**Files Examined:**
- `/Users/dylanconlin/Library/LaunchAgents/com.orch.daemon.plist` - Daemon configuration pointing to ~/bin/orch
- `/Users/dylanconlin/Library/LaunchAgents/com.orch-go.serve.plist` - Serve daemon uses build/orch pattern

**Commands Run:**
```bash
# Test install target shows reminder
make install

# Test help includes new target
make help
```

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: How to ensure daemon picks up new binary after make install?
- Context: Daemon runs from ~/bin/orch which is replaced by make install

**2025-12-26:** Implementation complete
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Added `make install-restart` target and post-install reminder to Makefile

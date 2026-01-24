<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Crash telemetry was already implemented and committed to OpenCode fork by another agent; binary needs rebuild to deploy.

**Evidence:** Commit f3b6f3f4c in opencode repo adds crash handlers; binary dated Jan 18 is stale vs Jan 23 commit.

**Knowledge:** The handlers write to ~/.local/share/opencode/crash.log with error, stack trace, and memory stats.

**Next:** Rebuild OpenCode binary and restart server to deploy crash telemetry (orchestrator action).

**Promote to Decision:** recommend-no (work was tactical implementation, not architectural choice)

---

# Investigation: Add Crash Telemetry Opencode Fork

**Question:** Was crash telemetry added to the OpenCode fork, and is it deployed?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent (feature-impl)
**Phase:** Complete
**Next Step:** None - documented that work was already done
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None
**Extracted-From:** Issue orch-go-2fddu (Add crash telemetry to OpenCode fork)
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Crash telemetry already implemented in OpenCode fork

**Evidence:** Commit `f3b6f3f4c` in ~/Documents/personal/opencode:
```
feat(server): add crash telemetry handlers

Add process-level handlers for uncaughtException and unhandledRejection
that write crash details (error, stack, memory stats) to
~/.local/share/opencode/crash.log. Handlers are registered when server
starts via listen() function.

Addresses orch-go issue orch-go-2fddu.
```

**Source:** `git log --oneline -1 f3b6f3f4c` in opencode repo

**Significance:** The implementation is complete - handlers exist at server.ts:68-131 and are called at line 2931 during server startup.

---

### Finding 2: Binary is stale - not rebuilt after commit

**Evidence:**
- Commit date: Fri Jan 23 23:27:34 2026
- Binary modified: Sun Jan 18 11:25:24 2026 (5 days earlier)
- Location: ~/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode

**Source:** `ls -la` on the binary file and `git show f3b6f3f4c` for commit date

**Significance:** The running server uses the old binary without crash telemetry. Rebuild required to deploy.

---

### Finding 3: Another agent actively working on this in Docker

**Evidence:** Process list shows Docker container working on crash telemetry:
```
docker run -it --rm --name orch-op-feat-add-crash-telemetry-23jan-5926 ...
```

**Source:** `pgrep -fl opencode`

**Significance:** There's a parallel agent that completed the code changes and may be working on rebuild/testing.

---

## Synthesis

**Key Insights:**

1. **Duplicate spawn** - This task (orch-go-2fddu) was assigned to this orch-go agent, but another agent in the opencode project already implemented the solution.

2. **Deployment gap** - Code is committed but binary not rebuilt, so crash telemetry is not yet active.

3. **Implementation is sound** - The handlers use synchronous file writes (appendFileSync), capture memory stats, and re-throw uncaughtException to maintain Node.js default exit behavior.

**Answer to Investigation Question:**

Crash telemetry **is implemented** (commit f3b6f3f4c) but **not deployed** (stale binary). The work was done by another agent. Remaining steps:
1. Rebuild: `cd ~/Documents/personal/opencode/packages/opencode && bun run build`
2. Restart: `orch-dashboard restart`
3. Verify: Check ~/.local/share/opencode/crash.log after next crash

---

## Structured Uncertainty

**What's tested:**

- ✅ Crash handlers exist in server.ts (verified: read file, found lines 68-131)
- ✅ Commit exists in git history (verified: git log shows f3b6f3f4c)
- ✅ Binary is stale (verified: compared file modification dates)
- ✅ Docker agent exists for this task (verified: pgrep output)

**What's untested:**

- ⚠️ Crash handlers actually work (not tested - no crash occurred to verify logging)
- ⚠️ Memory stats are accurate under crash conditions (not benchmarked)
- ⚠️ Log file doesn't grow unbounded (no rotation mechanism visible)

**What would change this:**

- If binary modification date was after Jan 23, crash telemetry would be deployed
- If crash.log existed with entries, would confirm handlers are working

---

## Implementation Recommendations

### Recommended Approach ⭐

**Close this issue as duplicate** - The work was completed by another agent.

**Why this approach:**
- Avoids redundant implementation effort
- Work is already committed to opencode repo
- Binary rebuild is a deployment step, not a code change

**Trade-offs accepted:**
- Orchestrator needs to coordinate rebuild/restart separately

**Implementation sequence:**
1. Close this orch-go issue (orch-go-2fddu) noting duplicate work
2. Ensure opencode agent completes rebuild verification
3. Orchestrator triggers orch-dashboard restart when ready

---

## References

**Files Examined:**
- ~/Documents/personal/opencode/packages/opencode/src/server/server.ts - Crash handlers implementation
- ~/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode - Binary file

**Commands Run:**
```bash
# Check crash telemetry commit
cd ~/Documents/personal/opencode && git show f3b6f3f4c --stat

# Check binary age
ls -la ~/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode

# Check for running agents
pgrep -fl opencode
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Original crash analysis that motivated this work

---

## Investigation History

**2026-01-23 15:24:** Investigation started
- Initial question: Add crash telemetry to OpenCode fork
- Context: Spawned from orch-go-2fddu beads issue

**2026-01-23 15:26:** Discovered work already done
- Found commit f3b6f3f4c in opencode repo with crash handlers
- Found Docker agent already working on this task
- Determined this is a duplicate spawn

**2026-01-23 15:28:** Investigation completed
- Status: Complete - work was already done by another agent
- Key outcome: Crash telemetry committed, needs rebuild to deploy

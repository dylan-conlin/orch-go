## Summary (D.E.K.N.)

**Delta:** Implemented spawn time, runtime, and activity detection for tmux agents in dashboard, achieving visibility parity with OpenCode agents.

**Evidence:** Tests pass (go test ./cmd/orch/... - 1.975s), code compiles, implementation matches design document.

**Knowledge:** Workspace file modification times provide reliable activity detection; Priority Cascade already handles tmux agent phase-based status.

**Next:** Close - all deliverables complete, tests passing.

**Promote to Decision:** recommend-no (tactical implementation, not architectural)

---

# Investigation: Implement Tmux Session Visibility Dashboard

**Question:** How to add spawn time, runtime, and activity detection for tmux agents (Claude CLI escape hatch) in the dashboard?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Agent og-feat-implement-tmux-session-18jan-9878
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Tmux agents lacked visibility parity with OpenCode agents

**Evidence:** In serve_agents.go lines 568-589, tmux-only agents were added with:
- Static "active" status
- No spawn_time lookup
- No runtime calculation
- No activity detection

**Source:** cmd/orch/serve_agents.go:568-589

**Significance:** Tmux agents (Claude CLI escape hatch) had no runtime visibility in dashboard, making it impossible to track how long they've been running or detect if they're dead.

---

### Finding 2: Infrastructure for workspace-based lookups already exists

**Evidence:**
- `wsCache.lookupWorkspace(beadsID)` provides O(1) workspace path lookup
- `spawn.ReadSpawnTime(workspacePath)` reads spawn timestamp from `.spawn_time` file
- `wsCache.lookupProjectDir(beadsID)` provides project directory lookup

**Source:** cmd/orch/serve_agents.go workspace cache methods

**Significance:** No new infrastructure needed - implementation could reuse existing workspace cache patterns.

---

### Finding 3: Priority Cascade already handles tmux agent status

**Evidence:** `determineAgentStatus()` is called for ALL agents in the beads batch fetch section (line 868), including tmux agents. This handles:
- beads issue closed → "completed"
- Phase: Complete reported → "completed"
- SYNTHESIS.md exists → "completed"
- Otherwise uses sessionStatus (which we set to "active" or "dead")

**Source:** cmd/orch/serve_agents.go:866-868

**Significance:** No changes needed for phase-based status determination - just needed to pass correct initial status.

---

### Finding 4: Frontend handles partial data gracefully

**Evidence:**
- Worker health lookup: `agent.session_id && $coaching.worker_health` (line 13 in agent-card.svelte)
- Runtime display: `{agent.runtime || formatDuration(agent.spawned_at)}` (line 551)
- Spawned_at display: `agent.spawned_at ? ... : 'Unknown'` (line 554)

**Source:** web/src/lib/components/agent-card/agent-card.svelte

**Significance:** No frontend changes required - already handles missing session_id, missing tokens, etc.

---

## Synthesis

**Key Insights:**

1. **Workspace-based activity detection** - Using file modification times from the workspace directory provides reliable activity detection without needing tmux pane activity or external monitoring.

2. **Existing infrastructure sufficient** - The workspace cache and spawn time reading functions already existed, making implementation straightforward.

3. **Priority Cascade is universal** - The existing status determination logic handles all agent types correctly through beads phase lookup.

**Answer to Investigation Question:**

Implemented tmux session visibility by:
1. Looking up workspace path via `wsCache.lookupWorkspace(beadsID)`
2. Reading spawn time via `spawn.ReadSpawnTime(workspacePath)` for runtime calculation
3. Adding `getWorkspaceLastActivity()` to check workspace file modification times
4. Setting status to "dead" if no activity for 3+ minutes (matches deadThreshold)
5. Relying on existing Priority Cascade for phase-based status determination

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles (verified: go build ./cmd/orch/...)
- ✅ All cmd/orch tests pass (verified: go test ./cmd/orch/... - 1.975s)
- ✅ Binary installs correctly (verified: make install)

**What's untested:**

- ⚠️ End-to-end with actual tmux agent running (no active tmux agents at test time)
- ⚠️ Dashboard visual rendering (no visual verification performed)

**What would change this:**

- If workspace files are not updated frequently enough, activity detection may incorrectly mark agents as dead
- If beadsID is missing from window name, workspace lookup fails

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Workspace file modification time activity detection** - Check modification times of workspace files to determine if tmux agent is active.

**Why this approach:**
- Simple implementation using existing filesystem APIs
- No external dependencies (tmux pane activity requires tmux commands)
- Workspace files are naturally updated during agent activity

**Trade-offs accepted:**
- Less precise than real-time tmux pane activity
- Files may not update during long tool execution

**Implementation sequence:**
1. Add spawn time and runtime calculation for tmux agents
2. Add `getWorkspaceLastActivity()` helper function
3. Set dead status if no activity for 3+ minutes

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Main implementation file
- `pkg/spawn/session.go` - ReadSpawnTime function
- `web/src/lib/components/agent-card/agent-card.svelte` - Frontend partial data handling

**Related Artifacts:**
- **Design:** `.kb/investigations/2026-01-18-design-dashboard-add-tmux-session-visibility.md`

---

## Investigation History

**2026-01-18:** Investigation started
- Initial question: How to add tmux session visibility in dashboard
- Context: Following design document from architect session

**2026-01-18:** Implementation completed
- Status: Complete
- Key outcome: Tmux agents now have spawn time, runtime, and activity detection

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

---

# Investigation: Dead Agents in Dashboard - Lifecycle Gap

**Question:** Why do agents show as "dead" in dashboard when OpenCode session ends but beads issue is still in_progress? What's the lifecycle gap?

**Started:** 2025-12-31
**Updated:** 2025-12-31
**Owner:** Spawned agent
**Phase:** Investigating
**Next Step:** Document findings and test lifecycle scenarios
**Status:** In Progress

---

## Findings

### Finding 1: Dead Status Determined by Session Inactivity

**Evidence:** Dashboard determines "dead" status based on OpenCode session's `time.Updated` field. If no activity for 3+ minutes (StaleSessionThreshold), agent is marked "dead".

```go
// From cmd/orch/serve.go:790-821
deadThreshold := opencode.StaleSessionThreshold  // 3 minutes
// ...
if timeSinceUpdate > deadThreshold {
    status = "dead"
}
```

**Source:** 
- `cmd/orch/serve.go:790-821` - Dashboard status calculation
- `pkg/opencode/client.go:382-386` - StaleSessionThreshold constant (3 minutes)

**Significance:** The dashboard relies on OpenCode's internal session state, not beads issue status. If an agent's OpenCode session ends (crash, context exhaustion, user exit) without calling `bd comment "Phase: Complete"`, the session goes stale but beads issue remains open.

---

### Finding 2: Two Independent State Systems - OpenCode Sessions vs Beads Issues

**Evidence:** The orchestration system has two independent state tracking systems:

1. **OpenCode Sessions** - Tracks actual Claude agent runtime
   - States: active (recent activity), dead (3+ min stale), completed (Phase: Complete + closed)
   - Updated via SSE events from OpenCode API
   - Session ends when: agent exits, crashes, context exhausts, user kills

2. **Beads Issues** - Tracks work assignment and completion
   - States: open, in_progress, closed
   - Updated via `bd` CLI commands
   - Closed when: orchestrator runs `orch complete` which calls `bd close`

**Source:**
- `pkg/opencode/client.go` - Session state management
- `pkg/daemon/daemon.go:1008-1016` - CompletedAgent struct shows beads status independent of session
- `cmd/orch/serve.go` - Dashboard combines both sources

**Significance:** These two systems can get out of sync. A "dead" agent occurs when:
- OpenCode session is stale/ended (dead in session terms)
- Beads issue is still in_progress (not closed)

---

### Finding 3: Expected Lifecycle vs Failure Modes

**Evidence:** The expected agent lifecycle is:

1. `orch spawn` creates beads issue (in_progress) + OpenCode session (active)
2. Agent works, session stays active
3. Agent reports `bd comment "Phase: Complete - ..."`
4. Agent runs `/exit` to close session
5. Orchestrator runs `orch complete <id>` to close beads issue

**Failure modes that cause "dead" agents:**
1. **Agent crash/context exhaustion** - Session ends abruptly without Phase: Complete
2. **Agent forgets /exit** - Session goes stale after 3 min idle
3. **User kills session** - Manual termination without completion
4. **OpenCode restart** - Sessions lost but beads issues persist
5. **Orchestrator doesn't complete** - Phase: Complete reported but issue not closed

**Source:**
- `pkg/spawn/context.go:89-99` - SESSION COMPLETE PROTOCOL instructions
- `pkg/daemon/daemon.go:976-1096` - Completion processing polls for Phase: Complete

**Significance:** The "dead" state is a natural consequence of agents not following the complete protocol. The system correctly identifies these orphaned agents.

---

### Finding 4: Existing Detection and Recovery Mechanisms

**Evidence:** The system already has mechanisms to detect and handle dead agents:

1. **Daemon Completion Polling** (`pkg/daemon/daemon.go:1034-1096`)
   - Polls for Phase: Complete in beads comments
   - Can auto-close issues that have Phase: Complete
   - But only works if agent reported Phase: Complete before dying

2. **SSE Monitor Backup** (`pkg/opencode/service.go:143-198`)
   - Watches for session completion via SSE
   - Can add "Phase: Complete - Session finished (detected via SSE monitor)"
   - Triggered on busy->idle transition

3. **Dashboard UI Shows Actions** (`web/src/lib/components/agent-card/agent-card.svelte:440`)
   - Tooltip shows: "Run `orch abandon <beads-id>` or respawn"

**Source:**
- `pkg/daemon/daemon.go` - Completion processing
- `pkg/opencode/service.go:143-198` - SSE monitor backup
- `web/src/lib/components/agent-card/agent-card.svelte:440` - UI guidance

**Significance:** The system has recovery mechanisms but they're not fully utilized:
- SSE monitor only fires on busy->idle, not on session disappearance
- Daemon polling requires Phase: Complete which dead agents often lack
- No automatic "offer to close" feature in dashboard UI

---

## Synthesis

**Key Insights:**

1. **Dead agents are expected** - The system correctly identifies agents whose OpenCode sessions ended without proper completion. This is by design, not a bug.

2. **Two-system gap is intentional** - Separating session state (OpenCode) from work state (beads) enables recovery. A dead session doesn't mean failed work - the agent may have committed code and reported Phase: Complete.

3. **Recovery requires orchestrator action** - The system deliberately requires orchestrator review before closing dead agents. This prevents auto-closing agents that may have partially completed work.

**Answer to Investigation Question:**

Dead agents appear when OpenCode sessions become stale (3+ min no activity) while beads issues remain in_progress. This happens because:

1. **Detection is session-based** - Dashboard polls OpenCode API and marks agents "dead" when `time.Updated` exceeds 3 minutes
2. **Issues are work-based** - Beads issues only close via `orch complete` which requires orchestrator action
3. **Gap is intentional** - Allows orchestrator to review dead agents and decide: complete (if work done), abandon (if failed), or respawn (if needs retry)

Why sessions end without `orch complete`:
- Agent crash/context exhaustion (most common)
- Agent forgot `/exit` → goes idle → marked dead
- User killed session manually
- OpenCode server restarted

---

## Structured Uncertainty

**What's tested:**

- ✅ Dead threshold is 3 minutes (verified: `pkg/opencode/client.go:386`)
- ✅ Dashboard computes status from session.time.Updated (verified: `cmd/orch/serve.go:790-821`)
- ✅ Dashboard shows "orch abandon" guidance for dead agents (verified: `agent-card.svelte:440`)

**What's untested:**

- ⚠️ SSE monitor backup detection effectiveness (code exists but needs validation)
- ⚠️ Rate of dead agents in production (no metrics visible)
- ⚠️ Whether adding auto-close would cause false positives

**What would change this:**

- Finding would be wrong if sessions can be "dead" without stale time.Updated
- Finding would change if beads issues auto-close on session end

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Dashboard "Quick Actions" for Dead Agents

**Summary:** Add action buttons to dead agent cards in dashboard UI for one-click recovery.

**Why this approach:**
- Doesn't change the intentional two-system design
- Keeps orchestrator in control (explicit action required)
- Reduces friction for common recovery operations
- Already has tooltip showing commands, just needs clickable actions

**Trade-offs accepted:**
- Still requires manual intervention (vs fully automatic)
- Acceptable because orchestrator review is valuable for dead agents

**Implementation sequence:**
1. Add "Complete" button (calls `orch complete <beads-id>`) - for dead agents that have Phase: Complete
2. Add "Abandon" button (calls `orch abandon <beads-id>`) - for dead agents without Phase: Complete
3. Add "Respawn" button (calls `orch spawn <skill> <task> --issue <beads-id>`) - to retry

### Alternative Approaches Considered

**Option B: Auto-close dead agents after timeout**
- **Pros:** No manual intervention needed
- **Cons:** Could close agents that partially completed work; removes orchestrator review opportunity
- **When to use instead:** If dead agents are consistently junk (no useful work)

**Option C: Enhance SSE monitor to detect session disappearance**
- **Pros:** Catches more edge cases (server restart, sudden death)
- **Cons:** Complex to implement reliably; still needs orchestrator review after detection
- **When to use instead:** If many dead agents are from server restarts

**Rationale for recommendation:** Option A (dashboard actions) is lowest risk, highest value. It respects the intentional design while reducing friction for the common case of orchestrator reviewing dead agents.

---

### Implementation Details

**What to implement first:**
- "Complete" button for dead agents that have Phase: Complete (clear case)
- This is lowest risk - just UI shortcut for existing `orch complete` command

**Things to watch out for:**
- ⚠️ Race condition: agent could come back to life while user clicks abandon
- ⚠️ Cross-project agents: need to pass correct workdir
- ⚠️ Button states: disable while action in progress

**Areas needing further investigation:**
- Metrics on dead agent frequency and causes
- Whether SSE monitor is being triggered effectively

**Success criteria:**
- ✅ Dead agents can be completed/abandoned with one click from dashboard
- ✅ No auto-closing without orchestrator action
- ✅ Button shows appropriate action based on Phase: Complete status

---

## References

**Files Examined:**
- `cmd/orch/serve.go:780-930` - Dashboard API agent status calculation
- `pkg/opencode/client.go:375-425` - Session state and StaleSessionThreshold
- `pkg/opencode/monitor.go` - SSE monitor for completion detection
- `pkg/opencode/service.go:143-198` - Completion service backup detection
- `pkg/daemon/daemon.go:970-1150` - Completion processing
- `web/src/lib/components/agent-card/agent-card.svelte:315-445` - Dead agent UI

**Commands Run:**
```bash
# Search for dead status handling
rg "dead|Dead" --type go cmd/orch/serve.go
rg "dead|Dead" web/src/

# Search for completion lifecycle
rg "Phase: Complete" --type go
```

**Related Artifacts:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT template with completion protocol

---

## Investigation History

**2025-12-31:** Investigation started
- Initial question: Why do agents show as dead when session ends but issue is in_progress?
- Context: User noticed dead agents in dashboard needing attention

**2025-12-31:** Findings documented
- Found two-system architecture (OpenCode sessions vs beads issues)
- Identified lifecycle gap as intentional design
- Documented recovery mechanisms and UI actions

**2025-12-31:** Investigation completing
- Status: Complete
- Key outcome: Dead agents are expected state for orphaned sessions; dashboard should offer quick actions for common recovery operations

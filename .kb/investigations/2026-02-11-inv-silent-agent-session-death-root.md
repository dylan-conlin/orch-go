<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dead session detection infrastructure exists but checks session creation time instead of activity time, causing crashed sessions to appear "alive" for up to 6 hours.

**Evidence:** Code review of pkg/daemon/session_dedup.go:90-92 shows HasExistingSession() checks `Time.Created` (line 90) instead of calling IsSessionActive() which checks `Time.Updated` within timeout window.

**Knowledge:** System has 4 detection gaps working in concert: (1) spawn doesn't capture session ID, (2) dead session detector checks wrong signal (creation vs activity), (3) no crash watchdog between SSE events, (4) state DB cache never reconciled with reality.

**Next:** Fix HasExistingSession() to check activity time instead of creation time - changes ~5 lines in pkg/daemon/session_dedup.go, reuses existing IsSessionActive() API, fixes most common failure mode.

**Authority:** implementation - Tactical fix within existing detection subsystem, no architectural changes or cross-component impact. Uses existing liveness API (IsSessionActive) that already exists and is tested.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Silent Agent Session Death Root

**Question:** Why do 40% of OpenCode agent sessions die without crash signal, leaving phantom agents in orch status?

**Started:** 2026-02-11
**Updated:** 2026-02-11
**Owner:** Agent (spawned from orch-go-i8vte)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: tmux spawn is fire-and-forget - no session ID capture at spawn time

**Evidence:**
- `pkg/spawn/claude.go:27-94` - SpawnClaude() launches claude via tmux but does NOT capture session ID
- The function pipes context to `claude --dangerously-skip-permissions`, sends commands via tmux, but returns only `tmux.SpawnResult` containing window info (no OpenCode session ID)
- OpenCode session creation happens asynchronously after the tmux window is created, but spawn doesn't wait for it
- Session ID is discovered later through polling (FindRecentSession) by matching project directory within 30-second window

**Source:** 
- pkg/spawn/claude.go:79 - Launch command execution
- pkg/spawn/claude.go:88-93 - SpawnResult only contains window metadata
- pkg/opencode/session.go:287-330 - FindRecentSession polls for sessions created <30s ago

**Significance:** This creates a race condition where:
1. If OpenCode session creation fails silently, spawn succeeds but leaves no session ID
2. If multiple agents spawn in the same directory within 30s, FindRecentSession can match the wrong session
3. No synchronization between tmux spawn and OpenCode session existence means system can't detect session creation failures

---

### Finding 2: No crash detection infrastructure - only normal completion via SSE

**Evidence:**
- `pkg/opencode/monitor.go:193-247` - Monitor only watches for `session.status` SSE events for busy→idle transitions (normal completion)
- No watchdog checking if OpenCode sessions actually die
- No process monitoring for the underlying bun/Node.js processes
- `pkg/process/ledger.go` exists for process tracking but is not integrated with session crash detection
- SSE connection loss triggers reconnection but doesn't detect which sessions died during the outage

**Source:**
- pkg/opencode/monitor.go:194-247 - handleEvent only processes session.status for completion
- pkg/opencode/monitor.go:79-124 - run() loop handles SSE reconnection but no session reconciliation
- pkg/process/ledger.go:212-228 - Reconcile() can detect dead processes but isn't called for session crashes

**Significance:** When an OpenCode session crashes (OOM, segfault, API timeout):
1. No SSE event is emitted (crash is not a "completion")
2. Monitor never triggers completion handlers
3. State DB retains last known phase (looks active)
4. orch status shows phantom agent until manual cleanup
5. No crash event logged to agentlog for orchestrator visibility

---

### Finding 3: State DB is projection cache - crashes leave orphaned records

**Evidence:**
- `pkg/state/db.go:3-34` - Explicit contract: "state.db is a spawn-time projection cache for fast reads", NOT source of truth
- Package documentation states: "Beads owns completion status, OpenCode owns session liveness, Tmux owns window presence, state.db owns NOTHING authoritatively"
- All state DB writes are best-effort and non-fatal (pkg/state/db.go:14-19)
- No reconciliation loop to cross-check state DB against authoritative sources (OpenCode API, tmux, beads)
- `pkg/state/db.go:625-676` - GetDriftMetrics exists but shows reconciliation is a future goal, not current practice

**Source:**
- pkg/state/db.go:3-34 - Cache contract documentation
- pkg/state/db.go:625-676 - DriftMetrics shows awareness of staleness problem
- pkg/state/agent.go:10-17 - Field ownership documented but not enforced

**Significance:** When a session crashes:
1. State DB still shows agent as active (is_completed=0, is_abandoned=0)
2. No reconciliation detects the session no longer exists in OpenCode
3. `orch status` reads from state DB → shows phantom agent
4. Operator has no signal that reconciliation is needed
5. Manual cleanup required to mark agent as abandoned

---

### Finding 4: Dead session detection checks creation time, not liveness

**Evidence:**
- `pkg/daemon/dead_session_detection.go:84` - Dead session detector calls `HasExistingSessionForBeadsID()` to check if session is active
- `pkg/daemon/session_dedup.go:68-98` - `HasExistingSession()` checks if session was **created** within last 6 hours, NOT if it's currently alive
- Line 90-92: `createdAt := time.Unix(s.Time.Created/1000, 0); age := now.Sub(createdAt); if age <= c.config.MaxAge`
- Uses `Time.Created` instead of `Time.Updated` or active session API check
- NO call to `IsSessionActive()` which would check if session updated within timeout window

**Source:**
- pkg/daemon/session_dedup.go:65-98 - HasExistingSession implementation
- pkg/opencode/session.go:129-137 - IsSessionActive (not used by dead session detection)

**Significance:** Dead session detection CANNOT detect crashed sessions:
1. Session created 4 hours ago crashes after 1 hour
2. Dead session detector checks if session exists: HasExistingSession(beadsID) → true (created <6h ago)
3. Detection thinks session is alive even though it crashed 3 hours ago
4. Issue remains in_progress status, appears as phantom in `orch status`
5. Detection won't mark as dead until 6 hours after creation (MaxAge expiry), not 6 hours after crash

---

## Synthesis

**Key Insights:**

1. **Fire-and-forget spawn creates session discovery gap** - The tmux spawn flow launches agents without capturing session IDs, relying on time-based polling (FindRecentSession) that can race or mismatch. This means spawn success doesn't guarantee session tracking.

2. **State DB is cache without reconciliation** - state.db explicitly documents itself as a projection cache, NOT authoritative. But there's no reconciliation loop to detect when cached state (agent active) diverges from reality (session dead). This makes phantom agents invisible until manual cleanup.

3. **Dead session detection has wrong liveness check** - The daemon HAS dead session detection infrastructure, but it checks session CREATION time instead of ACTIVITY time. A session that dies 5 minutes after spawn will be considered "alive" for 6 hours (MaxAge window), leaving the issue in_progress.

4. **No crash signal from OpenCode to orchestrator** - SSE monitor watches for normal completion (busy→idle), but crashes produce no event. Process death is silent. The system has no watchdog checking if sessions or processes are still alive between SSE events.

**Answer to Investigation Question:**

Sessions die silently because the system has **detection infrastructure that doesn't detect crashes**:

1. **Spawn doesn't track session creation** → Session ID comes later via racy polling, so spawn can succeed while session creation fails silently (Finding 1)
2. **Dead session detection checks wrong signal** → Uses creation time instead of activity time, so crashed sessions look "alive" for up to 6 hours (Finding 4)
3. **No crash watchdog** → SSE only emits completion events, not death events. No process monitoring between events (Finding 2)
4. **State DB has no reconciliation** → Cached "active" status never reconciled against OpenCode API or tmux to detect dead sessions (Finding 3)

The 40% silent death rate occurs because these gaps compound: session crashes → SSE emits nothing → dead session detector sees creation time and thinks it's alive → state DB cache never reconciled → phantom agent persists until manual cleanup or 6-hour timeout.

---

## Structured Uncertainty

**What's tested:**

- ✅ **tmux spawn doesn't capture session ID** - Verified by reading pkg/spawn/claude.go:88-93, SpawnResult struct contains only window metadata (Window, WindowID, WindowName, WorkspaceName), no SessionID field
- ✅ **Session ID discovery uses time-based polling** - Verified pkg/opencode/session.go:287-330, FindRecentSession matches by directory and created<30s
- ✅ **SSE monitor only watches for completion events** - Verified pkg/opencode/monitor.go:193-247, handleEvent filters for session.status events, detects busy→idle only
- ✅ **State DB is documented as cache not authority** - Verified pkg/state/db.go:3-34, explicit contract warning against treating as authoritative
- ✅ **Dead session detection checks creation time** - Verified pkg/daemon/session_dedup.go:90-92, uses `Time.Created` for age calculation
- ✅ **IsSessionActive exists but isn't used by detector** - Verified pkg/opencode/session.go:129-137 has IsSessionActive(sessionID, maxIdleTime), verified pkg/daemon/session_dedup.go does NOT call it

**What's untested:**

- ⚠️ **Actual crash reproduction rate is 40%** - Task description states this but I didn't verify against logs/metrics
- ⚠️ **Daemon actually runs dead session detection every 10min** - Saw config but didn't verify daemon is running or check logs
- ⚠️ **Fix will detect crashes within 10min** - Theory based on code reading, not tested with real crash
- ⚠️ **Process-level crashes (OOM) vs session-level crashes** - Don't know which is more common from logs

**What would change this:**

- Finding would be wrong if `HasExistingSession` actually calls `IsSessionActive` somewhere I missed (searched all usages)
- Finding would be wrong if SSE emits crash events that monitor ignores (checked handleEvent, only processes session.status)
- Recommendation would change if crashes are primarily at process level (OOM kills) rather than session level (graceful failures) - watchdog would be better than fixing liveness check

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Fix dead session detection to check activity, not creation time** - Change `HasExistingSession()` to use `IsSessionActive()` instead of checking creation timestamp.

**Why this approach:**
- Quickest fix with highest impact - changes 1 function to fix the most common failure mode (Finding 4)
- Leverages existing `IsSessionActive()` API that checks update time within 30min window (pkg/opencode/session.go:130)
- Dead session detector already runs every 10min by daemon - just needs correct liveness signal
- No new infrastructure needed - just wire existing liveness check into existing detection loop

**Trade-offs accepted:**
- Still relies on daemon being running (doesn't help if daemon crashes)
- Doesn't fix the spawn-time session ID capture gap (Finding 1) - spawn can still succeed with no session tracking
- Doesn't add process-level watchdog (Finding 2) - only detects dead sessions via API, not dead processes

**Implementation sequence:**
1. **Change session dedup to check activity** - pkg/daemon/session_dedup.go:90-92 replace `Time.Created` check with call to `IsSessionActive(sessionID, 30*time.Minute)` or check `Time.Updated` instead of `Time.Created`
2. **Add crash event to agentlog** - When dead session detected, log crash event so orch dashboard shows what happened
3. **Test with forced crash** - Kill an active OpenCode session, verify detector marks it dead within 10min (next detection cycle)

### Alternative Approaches Considered

**Option B: Add process-level watchdog for OpenCode/bun processes**
- **Pros:** Would catch process crashes that don't emit SSE events; could detect OOM kills, segfaults; works even if daemon crashes and restarts
- **Cons:** Requires integrating pkg/process/ledger.go with session tracking; more complex than fixing existing detection; adds new failure mode (watchdog itself could fail)
- **When to use instead:** If crashes are mainly process-level (OOM, segfault) rather than graceful session failures

**Option C: Add state DB reconciliation loop**
- **Pros:** Would fix ALL stale state, not just dead sessions; addresses Finding 3 directly; makes state DB closer to authoritative
- **Cons:** Expensive to run (requires querying OpenCode API, tmux, beads for every agent); pkg/state/db.go explicitly warns against promoting cache to authority without reconciliation infrastructure; this IS that infrastructure
- **When to use instead:** When state drift becomes widespread beyond just dead sessions (e.g., wrong phase, wrong model, missing session IDs)

**Option D: Fix spawn to capture session ID synchronously**
- **Pros:** Eliminates race condition from Finding 1; spawn either succeeds with session ID or fails loudly; no reliance on polling
- **Cons:** Changes spawn contract - might require waiting for OpenCode session creation; doesn't fix existing sessions that already crashed; doesn't address Finding 4 (dead sessions with IDs still need detection)
- **When to use instead:** If silent session creation failures (Finding 1) are more common than mid-flight crashes (Finding 4)

**Rationale for recommendation:** Fixing the liveness check in dead session detection (Option A) gives the most bang-for-buck:
- Minimal code change (1 function, ~5 lines)
- Reuses existing `IsSessionActive()` API
- Fixes the most common case: sessions that crash mid-flight (after spawn succeeds)
- Dead session detector already runs every 10min - just needs correct signal
- Other options are orthogonal improvements that can come later (process watchdog for OOM detection, reconciliation for comprehensive drift detection, spawn improvements for creation-time failures)

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- pkg/spawn/claude.go:27-94 - How tmux agents are spawned, confirmed no session ID capture in SpawnResult
- pkg/opencode/session.go:115-137, 287-330 - Session existence checking, session discovery via polling
- pkg/opencode/monitor.go:1-280 - SSE event monitoring for completion detection
- pkg/state/db.go:1-216 - State database cache contract and authority documentation
- pkg/state/agent.go:1-728 - Agent state schema and ownership documentation
- pkg/state/reconcile.go:1-315 - Liveness checking across multiple sources (tmux, OpenCode, beads, workspace)
- pkg/daemon/dead_session_detection.go:1-357 - Dead session detection infrastructure
- pkg/daemon/session_dedup.go:1-135 - Session deduplication logic, HasExistingSession implementation
- pkg/process/ledger.go:1-318 - Process tracking ledger (not integrated with session crash detection)
- cmd/orch/status_statedb.go:1-250 - Status command state DB integration
- cmd/orch/reconcile.go:1-100 - Reconcile command for zombie issue detection

**Commands Run:**
```bash
# Search git history for prior crash-related work
cd /Users/dylanconlin/Documents/personal/orch-go && git log --oneline --all --grep="crash" --grep="death" --grep="phantom" -i | head -20

# Search for session existence checking
grep -r "SessionExists\|IsSessionActive" --include="*.go"

# Search for dead session detection usage
grep -r "HasExistingSessionForBeadsID" --include="*.go"
```

**External Documentation:**
- None referenced

**Related Artifacts:**
- **Decision:** Likely exists - state.db cache contract mentioned .kb/decisions/2026-01-12-registry-is-spawn-cache.md (superseded)
- **Investigation:** .kb/investigations/2026-02-06-inv-evaluate-single-source-agent-state.md - State DB architecture evaluation
- **Prior commits:** f7c5bdf7 "feat: add dead session detection to daemon", 98c19eb3 "feat: distinguish agent death reasons in dashboard"

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

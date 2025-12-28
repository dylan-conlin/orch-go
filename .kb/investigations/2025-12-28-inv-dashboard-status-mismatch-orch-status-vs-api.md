## Summary (D.E.K.N.)

**Delta:** Dashboard shows different agent counts than `orch status` due to multiple interacting issues: SSE proxy missing directory header, status determination logic differs between API and CLI, and multiple sessions per beads ID cause deduplication confusion.

**Evidence:** `orch status` shows 6 active agents while dashboard shows 1; API returns 3 active/idle vs CLI's 6; SSE proxy was missing x-opencode-directory header (fixed); status determination uses time-based heuristics in API vs OpenCode session state in CLI.

**Knowledge:** The orchestration visibility stack has three layers (OpenCode sessions → orch serve API → dashboard) each with different state models. Fixes applied to one layer don't propagate to others without explicit coordination.

**Next:** Track remaining issues as separate beads items; this investigation documents the mess for future sessions.

---

# Investigation: Dashboard Status Mismatch - orch status vs API vs Dashboard

**Question:** Why do `orch status`, `/api/agents`, and the dashboard all show different agent counts and states?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Orchestrator
**Phase:** Synthesizing
**Next Step:** Create beads issues for remaining fixes
**Status:** In Progress

---

## Findings

### Finding 1: SSE Proxy Missing Directory Header (FIXED)

**Evidence:** 
- Raw OpenCode SSE at `localhost:4096/event` with `x-opencode-directory` header returned events
- `orch serve` proxy at `/api/events` was connecting WITHOUT the header
- Result: SSE stream only showed `server.connected` event, no agent activity

**Source:** 
- `cmd/orch/serve.go:1054-1056` - plain `http.Get()` without headers
- Fix committed: 8ccc3af2

**Significance:** Dashboard showed "Starting up..." for all agents because it never received activity events. This was the primary cause of the "Starting up..." symptom.

---

### Finding 2: Status Determination Logic Differs

**Evidence:**
```
orch status: 6 active (1 running, 5 idle)
API /api/agents: 1 active, 2 idle, 643 completed
Dashboard: 1 active agent shown
```

- `orch status` uses OpenCode's `IsSessionProcessing()` for running/idle distinction
- API uses time-based heuristics (`timeSinceUpdate > activeThreshold`)
- API marks as "completed" if beads Phase == Complete
- CLI keeps showing as "active/idle" until explicitly abandoned

**Source:**
- `cmd/orch/serve.go:655-658` - time-based status
- `cmd/orch/main.go:418+` - status command logic

**Significance:** Same agent can appear "running" in CLI but "completed" in API/dashboard if the beads issue was closed but OpenCode session is still alive.

---

### Finding 3: Multiple Sessions Per Beads ID

**Evidence:**
```bash
# Sessions with beads ID "orch-go-578d":
ses_4991ad485ffej2XFCzZNjvt4aZ  # spawned 13:37
ses_4991e2e6bffeD1825WIpiyzAlD  # spawned earlier
ses_4992f58c0ffegd7DBy62kBjLqd  # spawned earlier
ses_4995df4fbffecu5jPB5hqbaW2t  # spawned earlier
ses_499703766ffeh9OnUjATaMsOng  # spawned earlier
# ... 15 total sessions with same title
```

**Source:** `curl -H "x-opencode-directory: ..." localhost:4096/session | jq`

**Significance:** Agent respawns create new OpenCode sessions but reuse workspace/beads ID. API deduplicates by title (keeps most recent), but the logic may not always pick the "right" session.

---

### Finding 4: "Starting up..." vs "Waiting for activity..." (FIXED)

**Evidence:**
- Dashboard showed "Starting up..." when `current_activity` was null
- `current_activity` is populated from SSE events, cleared on idle
- For idle agents, nothing repopulates it

**Source:** `web/src/lib/components/agent-card/agent-card.svelte:437-445`

**Significance:** Fixed by showing phase or "Waiting for activity..." instead. Commit: e8b42281

---

### Finding 5: Circular Progress Root Cause (Documented)

**Evidence:** Session A fixed serve.go but didn't `make install`. Session B started 6 minutes later with stale binary, spent 30 minutes debugging the same issue.

**Source:** `.kb/investigations/2025-12-28-inv-circular-progress-between-orchestrator-sessions.md`

**Significance:** This explains why we were "running in circles" - stale binary inheritance between orchestrator sessions.

---

## Synthesis

**Key Insights:**

1. **Three-layer visibility stack** - OpenCode sessions → orch serve API → dashboard. Each layer has its own state model and they can diverge.

2. **Time-based vs event-based status** - API uses timestamps, CLI uses live queries, dashboard uses SSE events. No single source of truth.

3. **Session proliferation** - Respawning agents creates new sessions but reuses beads IDs, leading to confusion about which session is "the" agent.

**Answer to Investigation Question:**

The mismatches occur because:
1. SSE proxy wasn't forwarding events (FIXED - header added)
2. Status logic differs between CLI (live query) and API (time + beads phase)
3. Multiple sessions per beads ID means "most recent" might not be "currently running"
4. Dashboard clears activity on idle events, showing misleading "Starting up..."

---

## Structured Uncertainty

**What's tested:**

- ✅ SSE events flow with directory header (verified: curl shows message.part.updated events)
- ✅ "Waiting for activity..." displays instead of "Starting up..." (verified: glass screenshot)
- ✅ Multiple sessions exist per beads ID (verified: API query returned 15 sessions for one beads ID)

**What's untested:**

- ⚠️ Whether deduplication picks the correct "active" session
- ⚠️ Whether status reconciliation between CLI and API is correct
- ⚠️ Whether dashboard correctly shows all non-completed agents

**What would change this:**

- If OpenCode provided authoritative "is this session actively running" flag
- If we had single source of truth for agent lifecycle state
- If respawns created new beads IDs instead of reusing

---

## Implementation Recommendations

### Recommended Approach ⭐

**Unify status determination** - Create single function that both CLI and API use to determine agent state.

**Why this approach:**
- Eliminates divergent logic
- Single place to fix bugs
- Consistent user experience

**Trade-offs accepted:**
- May need to make API calls from CLI (latency)
- Need to handle offline/error cases

**Implementation sequence:**
1. Extract status logic to `pkg/state/reconcile.go`
2. Update API to use new logic
3. Update CLI to use new logic
4. Add tests for edge cases

### Alternative Approaches Considered

**Option B: Accept divergence, document it**
- **Pros:** No code changes needed
- **Cons:** Perpetual user confusion
- **When to use instead:** If unification is too complex

**Option C: Dashboard queries CLI instead of API**
- **Pros:** Would match CLI output exactly
- **Cons:** CLI is slow, not designed for polling
- **When to use instead:** If API can't be fixed

---

## Remaining Issues (Not Fixed)

1. **Agent count mismatch** - Dashboard shows 1 active, `orch status` shows 6
2. **Session deduplication** - May pick wrong session from duplicates
3. **Phase not shown** - Idle agents show "Waiting" not their phase

---

## References

**Files Examined:**
- `cmd/orch/serve.go` - API endpoints, status logic
- `cmd/orch/main.go` - CLI status command
- `web/src/lib/stores/agents.ts` - SSE handling, activity updates
- `web/src/lib/components/agent-card/agent-card.svelte` - UI display logic

**Commands Run:**
```bash
# Check SSE events
curl -s -H "x-opencode-directory: /path" http://localhost:4096/event

# Compare agent counts
orch status
curl -s http://127.0.0.1:3348/api/agents | jq 'group_by(.status)'

# Find duplicate sessions
curl -s -H "x-opencode-directory: /path" http://localhost:4096/session | jq '.[] | select(.title | contains("beads-id"))'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-circular-progress-between-orchestrator-sessions.md` - Root cause of session confusion
- **Commits:** 8ccc3af2 (SSE header fix), e8b42281 (UI text fix)

---

## Investigation History

**2025-12-28 13:23:** Investigation started
- Initial symptom: Dashboard showing "Starting up..." for all agents
- Context: Dylan frustrated after circular debugging sessions

**2025-12-28 13:35:** Found SSE proxy missing directory header
- Root cause of no activity events
- Fixed in commit 8ccc3af2

**2025-12-28 13:40:** Found status determination divergence
- CLI uses live OpenCode queries
- API uses time-based heuristics + beads phase
- Explains count mismatch

**2025-12-28 13:45:** Fixed "Starting up..." UI text
- Now shows phase or "Waiting for activity..."
- Commit e8b42281

**2025-12-28 13:50:** Investigation documented
- Status: Synthesizing - fixes applied, remaining issues tracked
- Key outcome: Primary visibility issues fixed, deeper reconciliation needed

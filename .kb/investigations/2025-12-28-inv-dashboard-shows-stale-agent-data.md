<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Two bugs cause dashboard stale data: (1) `orch status` misses project-directory sessions, (2) dashboard incorrectly shows active status for agents with Phase: Complete.

**Evidence:** API shows 6 active agents (4 with Phase: Complete), while `orch status` shows 0; OpenCode sessions with `x-opencode-directory` header are only returned when querying with that same header.

**Knowledge:** OpenCode stores sessions per-project-directory; querying without directory header misses project-specific sessions. Dashboard logic prioritizes session existence over phase status for determining completion.

**Next:** Fix both issues: (1) modify `orch status` to query sessions per project directory, (2) change dashboard to derive status from beads Phase regardless of session state.

---

# Investigation: Dashboard Shows Stale Agent Data

**Question:** Why does the dashboard show agents with `name: null`, `Phase: Complete` but `status: active`, while `orch status` shows 0 active agents?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** systematic-debugging spawn
**Phase:** Complete
**Next Step:** Implement fixes
**Status:** Complete

---

## Findings

### Finding 1: OpenCode Sessions Are Project-Directory Scoped

**Evidence:** 
- `curl http://localhost:4096/session | jq 'select(.title | test("og-feat-add-dev-server"))' ` returns empty
- `curl -H "x-opencode-directory: /Users/dylanconlin/Documents/personal/orch-go" http://localhost:4096/session | jq 'select(.title | test("og-feat-add-dev-server"))' ` returns the session

**Source:** OpenCode API behavior confirmed via curl commands; `pkg/opencode/client.go:294-320` shows ListSessions passes directory via header

**Significance:** Sessions created with `x-opencode-directory` header are stored per-project and ONLY returned when querying with that same header. This is the root cause of `orch status` showing 0 agents while dashboard shows 6.

---

### Finding 2: `orch status` Only Queries Global Sessions

**Evidence:** 
- `orch status` shows "Active: 0" while dashboard API shows 6 active agents
- `cmd/orch/main.go:2302` calls `client.ListSessions("")` with empty directory
- Dashboard's `handleAgents()` at `serve.go:578-621` builds workspace cache first, extracts PROJECT_DIRs, then queries each directory

**Source:** `cmd/orch/main.go:2302`, `cmd/orch/serve.go:578-621`

**Significance:** The dashboard correctly handles multi-project session discovery, but `orch status` does not. This causes status mismatch between CLI and dashboard.

---

### Finding 3: Dashboard Shows Active Status for Completed Agents

**Evidence:** 
- Dashboard returns 4 agents with `phase: "Complete"` but `status: "active"`
- `serve.go:926-953` only marks status as "completed" if agent does NOT have an active session:
  ```go
  hasActiveSession := agents[i].SessionID != "" || agents[i].Window != ""
  if !hasActiveSession {
      if strings.EqualFold(agents[i].Phase, "Complete") {
          agents[i].Status = "completed"
      }
  }
  ```

**Source:** `cmd/orch/serve.go:926-953`, API response showing 4 agents with Phase: Complete but status: active

**Significance:** This logic contradicts the prior decision: "Dashboard agent status derived from beads phase, not session time." An agent with Phase: Complete should be marked completed regardless of whether the OpenCode session is still open (agent may not have called `/exit` yet).

---

## Synthesis

**Key Insights:**

1. **Session Scoping Mismatch** - OpenCode sessions are scoped to project directories. Queries without the `x-opencode-directory` header miss project-specific sessions entirely. The dashboard handles this correctly by building workspace caches and querying each project, but `orch status` does not.

2. **Status Derivation Logic is Backwards** - The dashboard currently prioritizes session existence over beads Phase for status. This creates a race condition where agents that report Phase: Complete but haven't exited their session yet appear as "active" instead of "completed".

3. **Conflicting Definitions of "Active"** - There are two definitions: (a) session exists in OpenCode, (b) work is still in progress. The correct definition for the dashboard is (b) - beads Phase is authoritative for work status.

**Answer to Investigation Question:**

The dashboard shows stale data due to two bugs:
1. **Session discovery gap**: `orch status` doesn't query project-directory-scoped sessions, causing it to show 0 active when agents are running
2. **Status derivation bug**: Dashboard marks agents with Phase: Complete as "active" if their OpenCode session is still open

Both bugs stem from the same root conceptual issue: conflating "session exists" with "work is in progress". The beads Phase is the authoritative source for completion status.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode sessions with x-opencode-directory are only returned when querying with that header (verified: curl with/without header)
- ✅ Dashboard API returns 6 active agents with 4 having Phase: Complete (verified: curl /api/agents)
- ✅ orch status shows 0 active agents (verified: running command)
- ✅ serve.go logic explicitly requires no active session before marking completed (verified: read code at lines 926-953)

**What's untested:**

- ⚠️ Whether fixing the status derivation logic causes any UI regressions
- ⚠️ Whether there are other callers that depend on the current status derivation behavior

**What would change this:**

- Finding would be wrong if OpenCode sessions should actually be scoped globally and the directory header is optional
- Finding would be wrong if there's a legitimate reason to keep Phase: Complete agents as "active" when session exists

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Fix both issues with minimal changes** - Update status derivation logic in serve.go and add multi-project session querying to orch status

**Why this approach:**
- Dashboard already has the correct multi-project querying logic that can be adapted for orch status
- Status derivation fix is a ~5 line change that aligns with the documented prior decision
- Both fixes are independent and can be verified separately

**Trade-offs accepted:**
- Agents with Phase: Complete will immediately show as "completed" even if their session is still open (this is correct behavior per prior decision)
- orch status may be slightly slower due to querying multiple project directories

**Implementation sequence:**
1. Fix status derivation in serve.go (remove session check from completion logic) - immediate impact on dashboard
2. Add multi-project session querying to orch status - aligns CLI with dashboard
3. Add tests to verify both fixes

### Alternative Approaches Considered

**Option B: Only fix orch status, keep dashboard logic**
- **Pros:** Fewer changes
- **Cons:** Dashboard still shows Phase: Complete agents as "active", violates prior decision
- **When to use instead:** Never - the dashboard logic is the bug

**Option C: Close sessions automatically when Phase: Complete detected**
- **Pros:** Would make session existence correctly reflect work status
- **Cons:** Invasive change, may interfere with agent cleanup, adds complexity
- **When to use instead:** If we want sessions to be closed automatically (not recommended)

**Rationale for recommendation:** Option A addresses both root causes with minimal changes while aligning with documented prior decisions about status derivation.

---

### Implementation Details

**What to implement first:**
- Status derivation fix in serve.go (change at lines 940-953)
- This immediately fixes the dashboard's Phase: Complete agents showing as "active"

**Things to watch out for:**
- ⚠️ Make sure to test with agents that are actually still running vs agents that have exited
- ⚠️ The is_processing field may need defensive checks (already has them per prior decision)

**Areas needing further investigation:**
- Whether orch status should share workspace cache building logic with serve.go
- Consider refactoring common session discovery into a shared package

**Success criteria:**
- ✅ Dashboard shows agents with Phase: Complete as status: "completed"
- ✅ orch status shows the same active count as dashboard
- ✅ No agents shown with name: null (this was a symptom of the session discovery issue)

---

## References

**Files Examined:**
- `cmd/orch/serve.go:926-953` - Status derivation logic (the bug)
- `cmd/orch/serve.go:578-621` - Multi-project session discovery (correct implementation)
- `cmd/orch/main.go:2302` - orch status session listing (missing multi-project)
- `pkg/opencode/client.go:294-320` - ListSessions API with directory header

**Commands Run:**
```bash
# Verify session scoping
curl -s http://localhost:4096/session | jq 'length'
curl -s -H "x-opencode-directory: /Users/dylanconlin/Documents/personal/orch-go" http://localhost:4096/session | jq 'length'

# Verify dashboard data
curl -s http://localhost:3348/api/agents | jq '[.[] | select(.status == "active") | {id, phase, status}]'

# Verify orch status
/Users/dylanconlin/bin/orch status
```

**Related Artifacts:**
- **Decision:** Prior decision in spawn context about "Dashboard agent status derived from beads phase, not session time"
- **Decision:** "Dashboard is_processing visual indicators require status === 'active' check" (defensive check, still relevant)

---

## Investigation History

**2025-12-28 13:03:** Investigation started
- Initial question: Why does dashboard show stale agent data with name null and wrong status?
- Context: Spawned to debug dashboard inconsistencies

**2025-12-28 13:15:** Found session scoping issue
- OpenCode sessions with x-opencode-directory only returned when queried with header
- orch status queries without directory, misses project-specific sessions

**2025-12-28 13:20:** Found status derivation bug
- serve.go requires no active session before marking completed
- Contradicts prior decision about phase being authoritative

**2025-12-28 13:25:** Investigation completed
- Status: Complete
- Key outcome: Two bugs identified - session scoping and status derivation logic

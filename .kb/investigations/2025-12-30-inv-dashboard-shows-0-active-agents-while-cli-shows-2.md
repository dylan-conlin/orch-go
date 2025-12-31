<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The reported bug "Dashboard shows 0 active agents while CLI shows 2" was already fixed by Dec 28 commits; current system correctly shows active agents in both CLI and dashboard.

**Evidence:** API returns status='active' for running agents (verified: curl localhost:3348/api/agents shows 2 non-completed agents), dashboard displays "2 active" in Active Agents section (verified: glass_page_state).

**Knowledge:** The fix had two parts: (1) Dec 28 14:15 added 'idle' to frontend filter, (2) Dec 28 20:59 simplified API to return 'active' or 'dead' (removed 'idle' status entirely). Both changes ensure active agents appear in dashboard.

**Next:** Close this issue - no fix needed. The bug was stale at spawn time.

---

# Investigation: Dashboard Shows 0 Active Agents While Cli Shows 2

**Question:** Why does the dashboard show 0 active agents when CLI shows 2 running agents?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** systematic-debugging spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** .kb/investigations/2025-12-28-inv-dashboard-shows-active-cli-shows.md (finding still valid, this adds historical context)

---

## Findings

### Finding 1: Bug Was Already Fixed Before This Investigation

**Evidence:** 
- API currently returns `status='active'` for running sessions (not 'idle')
- Dashboard shows "2 active" when API has 2 non-completed agents
- git log shows the fix was applied Dec 28, 2025

**Source:** 
- `curl localhost:3348/api/agents` - returns status='active' for running agents
- `glass_page_state` - dashboard shows "🟢 2 active"
- `git log --oneline -- web/src/lib/stores/agents.ts` - commit 3a834ac0

**Significance:** The spawn context described a stale bug. No new fix was needed.

---

### Finding 2: Two-Part Fix History

**Evidence:** Git history shows two related commits on Dec 28:

1. **14:15:04 (3a834ac0)**: "fix: dashboard active agents now includes idle status agents"
   - Added 'idle' to activeAgents filter in frontend
   - At this time, API returned status='idle' for non-processing sessions

2. **20:59:54 (784c2703)**: "Simplify dead session detection to 3-minute heartbeat"
   - Removed 'idle' status entirely
   - API now returns 'active' (< 3min activity) or 'dead' (> 3min no activity)
   - Commit message: "Status is now 'active' or 'dead' (removed 'idle')"

**Source:**
- `git show 3a834ac0` - first fix
- `git show 784c2703` - second fix that simplified status model

**Significance:** The system evolved through both fixes. Currently uses simplified model where 'idle' doesn't exist.

---

### Finding 3: Current Status Model is Simpler

**Evidence:** serve.go lines 808-828 show current logic:
```go
status := "active"
if timeSinceUpdate > deadThreshold {
    status = "dead"
}
```

Then later:
- Phase: Complete → status = "completed"
- Untracked + no phase + > 1 min → status = "stalled"

**Source:** `cmd/orch/serve.go:808-1133`

**Significance:** Current model has only 4 statuses: active, dead, completed, stalled. Frontend filter accepts all of these for the "Active Agents" section (except completed).

---

## Synthesis

**Key Insights:**

1. **The Bug Was Stale** - By the time this investigation spawned, the bug had already been fixed 6+ hours earlier. The spawn context was describing historical behavior.

2. **Status Model Simplified** - The original bug (Dec 28 14:xx) was caused by API returning 'idle' while filter expected 'active'. Rather than just fix the filter, a later commit (Dec 28 20:59) simplified the entire model by removing 'idle' status.

3. **Current System Works Correctly** - API returns 'active' for running sessions, dashboard filter includes 'active', so agents appear correctly.

**Answer to Investigation Question:**

The dashboard currently DOES show active agents correctly. The reported symptom "Dashboard shows 0 active agents while CLI shows 2" was a description of a bug that was already fixed on Dec 28, 2025. The fix involved simplifying the API status model to use 'active' instead of 'idle' for running sessions.

---

## Structured Uncertainty

**What's tested:**

- ✅ API returns 2 non-completed agents (verified: curl localhost:3348/api/agents)
- ✅ Dashboard shows "2 active" (verified: glass_page_state)
- ✅ Git history confirms fixes applied Dec 28 (verified: git show commits)

**What's untested:**

- ⚠️ Edge case: agent transitioning between states during page refresh
- ⚠️ Whether the original spawn context was manually outdated vs stale queue

**What would change this:**

- Finding would be wrong if there's a race condition that causes agents to briefly not appear
- Finding would be wrong if there's a different code path that still returns 'idle'

---

## Implementation Recommendations

**Purpose:** No implementation needed - issue is already fixed.

### Recommended Approach ⭐

**Close Issue** - The bug was already fixed by prior commits.

**Why this approach:**
- Current system correctly shows active agents
- No code changes needed
- Investigation confirms the fix is working

**Trade-offs accepted:**
- None - this is simply acknowledging existing fix

**Implementation sequence:**
1. Close beads issue orch-go-lsrj
2. Consider improving spawn context freshness check to avoid spawning for stale bugs

### Alternative Approaches Considered

**Option B: Add regression test**
- **Pros:** Would catch future regressions
- **Cons:** Test would be verifying existing behavior
- **When to use instead:** If this bug recurs

---

## References

**Files Examined:**
- `cmd/orch/serve.go:695-1175` - API agent status logic
- `web/src/lib/stores/agents.ts:227-270` - Frontend derived stores
- `.kb/investigations/2025-12-28-inv-dashboard-shows-active-cli-shows.md` - Prior investigation

**Commands Run:**
```bash
# Check API agent status
curl -s http://localhost:3348/api/agents | python3 -c "..."

# Check git history for fixes
git log --oneline -- web/src/lib/stores/agents.ts
git show 3a834ac0
git show 784c2703
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-28-inv-dashboard-shows-active-cli-shows.md` - Original investigation that led to the fix

---

## Investigation History

**2025-12-30 17:27:** Investigation started
- Initial question: Why does dashboard show 0 active agents while CLI shows 2?
- Context: Spawned from beads issue orch-go-lsrj

**2025-12-30 17:28:** Discovered bug was already fixed
- API returns status='active' for running sessions
- Dashboard correctly shows active agents
- Found two related commits from Dec 28

**2025-12-30 17:35:** Investigation completed
- Status: Complete
- Key outcome: No fix needed - bug was stale at spawn time

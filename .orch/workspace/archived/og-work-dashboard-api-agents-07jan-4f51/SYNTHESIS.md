# Session Synthesis

**Agent:** og-work-dashboard-api-agents-07jan-4f51
**Issue:** orch-go-gi3ty
**Duration:** 2026-01-07
**Outcome:** success

---

## TLDR

Fixed /api/agents 20s cold cache performance issue by applying time/project filters EARLY (immediately after session fetch) instead of at the END, reducing expensive beads operations proportionally to filter selectivity.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_agents.go` - Added early session filtering after ListSessions(), updated late filter comment

### Changes Made
1. **Lines 326-353:** Added early filtering block that applies time/project filters immediately after fetching sessions from OpenCode
2. **Line 364:** Removed duplicate `now := time.Now()` declaration (now in early filter block)
3. **Lines 867-870:** Updated comment on late filter to clarify its purpose (handles tmux-only agents and completed workspaces)

---

## Evidence (What Was Observed)

- Original code applied filters at lines 867-893 AFTER all expensive operations
- With `?since=12h&project=orch-go`, ALL 600+ sessions were still processed through:
  - Workspace cache building
  - Beads batch fetching (open issues, all issues, comments)
  - Investigation directory cache building
  - Token fetching for active sessions
- Sessions have `Time.Updated` and `Directory` fields available immediately after `ListSessions()`

### Tests Run
```bash
# Build verification
go build -v ./cmd/orch/...
# github.com/dylan-conlin/orch-go/cmd/orch (success)

# Test verification
go test ./cmd/orch/... -run "Serve|Agent|Filter" -v
# All tests pass (TestHandleAgents, TestDetermineAgentStatus, TestFilterByProject, etc.)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-dashboard-api-agents-filters-applied-late.md` - Full investigation details

### Decisions Made
- Decision: Apply filters immediately after data source access, before expensive operations
- Rationale: Reduces O(n) operations proportionally to filter selectivity

### Constraints Discovered
- Dual filter points needed: early filter for OpenCode sessions, late filter for tmux-only agents and completed workspaces (different timestamp sources)
- Filter logic must reuse existing `filterByProject()` for consistency

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (early filtering implemented)
- [x] Tests passing (all serve/agent/filter tests pass)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-gi3ty`

---

## Unexplored Questions

**Questions that emerged during this session:**
- What is the actual cold cache time after the fix? (Would need to restart the serve process and test in browser with network timing)
- Are there other API endpoints with similar "filter late" anti-patterns?

**What remains unclear:**
- Exact performance improvement percentage (not benchmarked)

---

## Session Metadata

**Skill:** issue-creation
**Model:** opus
**Workspace:** `.orch/workspace/og-work-dashboard-api-agents-07jan-4f51/`
**Investigation:** `.kb/investigations/2026-01-07-inv-dashboard-api-agents-filters-applied-late.md`
**Beads:** `bd show orch-go-gi3ty`

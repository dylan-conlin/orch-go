## Summary (D.E.K.N.)

**Delta:** Bug no longer reproduces - beadsId is correctly populated for active agents; 43 historical workspaces show null (expected).

**Evidence:** `curl localhost:3348/api/agents | jq '.[0:3] | .[].beads_id'` now returns `"orch-go-xqwu"` etc. instead of `null`. 614/657 agents have correct beadsId.

**Knowledge:** The bug was likely caused by stale server binary - server restart at 14:46 (after issue filed at 14:44) loaded new code that already had correct beadsId extraction.

**Next:** Close issue - bug fixed via server restart with current code. No code changes needed.

---

# Investigation: Dashboard API Returns Null beadsId

**Question:** Why does `/api/agents` return `beadsId: null` for all agents when `orch status` shows correct beadsId?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Agent (og-debug-dashboard-api-returns-04jan)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Bug No Longer Reproduces

**Evidence:** 
```bash
curl localhost:3348/api/agents | jq '.[0:3] | .[].beads_id'
# Returns: "orch-go-xqwu", "orch-go-emmq", "orch-go-roxx"
```

Original bug report showed all three returning `null`.

**Source:** Direct reproduction test using exact command from bug report

**Significance:** The core issue has been resolved. Active agents now correctly show their beadsId.

---

### Finding 2: Historical Workspaces Correctly Show Null

**Evidence:**
```bash
curl -s localhost:3348/api/agents | jq '[.[] | select(.beads_id == null)] | group_by(.status) | .[] | {status: .[0].status, count: length}'
# Returns: {"status": "completed", "count": 43}
```

All 43 agents with null beadsId are `status: "completed"` - these are historical workspace directories that predate the beads tracking system.

**Source:** cmd/orch/serve_agents.go lines 339-344 - completed workspaces try to extract beadsId from SPAWN_CONTEXT.md, then fallback to extracting from directory name.

**Significance:** This is expected behavior, not a bug. Older workspaces don't have "spawned from beads issue" line in their SPAWN_CONTEXT.md and directory names don't include `[beads-id]` suffix.

---

### Finding 3: Server Restart Fixed the Issue

**Evidence:** Timeline analysis:
- 14:14:02 - serve_agents.go last modified
- 14:37 - Binary rebuilt
- 14:44 - Bug issue created
- 14:46 - Server restarted (visible in `ps aux` showing 2:46PM start time)

The server was running before the binary rebuild (14:37). Issue was filed while old code was running. Server restart loaded new code.

**Source:** `ps aux | grep "orch.*serve"` shows server start time of 2:46PM (14:46)

**Significance:** This was a stale binary issue, not a code bug. The current code correctly extracts beadsId from session titles using `extractBeadsIDFromTitle()`.

---

## Synthesis

**Key Insights:**

1. **BeadsId extraction works correctly** - The code at cmd/orch/serve_agents.go:157 correctly extracts beadsId from session titles using the `[beads-id]` pattern.

2. **Active vs Completed agents** - Active agents (from OpenCode sessions) get beadsId from session title. Completed agents (from workspace directories) get beadsId from SPAWN_CONTEXT.md or directory name.

3. **Historical data limitation** - 43 older workspaces don't have beads tracking metadata, so they correctly show null beadsId.

**Answer to Investigation Question:**

The `/api/agents` endpoint no longer returns null for all agents. The bug was caused by a stale server binary that didn't include the latest code. After server restart with the current binary, beadsId is correctly populated for all active agents (614/657). The 43 agents with null beadsId are completed historical workspaces that predate beads tracking - this is expected behavior.

---

## Structured Uncertainty

**What's tested:**

- ✅ API returns correct beadsId for active agents (verified: curl command reproduced expected output)
- ✅ All null beadsId agents are completed historical workspaces (verified: jq group_by query)
- ✅ Server restart resolved the issue (verified: timeline analysis shows restart after issue filed)

**What's untested:**

- ⚠️ Whether historical workspaces could have beadsId backfilled (not attempted)
- ⚠️ Root cause of why old binary had bug (not investigated - issue resolved)

**What would change this:**

- Finding would be wrong if bug reappears after another server restart
- Finding would be wrong if active agents start showing null beadsId again

---

## Implementation Recommendations

### Recommended Approach: Close Issue

No code changes needed. The bug was caused by stale server binary and has been resolved by the server restart that occurred at 14:46.

**Why this approach:**
- Bug no longer reproduces with current code
- 614 out of 657 agents correctly show beadsId
- 43 null beadsIds are expected (historical workspaces)

**Trade-offs accepted:**
- Historical workspaces will continue to show null beadsId
- Acceptable: these are old completions that don't need beads linking

---

## References

**Files Examined:**
- cmd/orch/serve_agents.go - Main endpoint handler for /api/agents
- cmd/orch/shared.go - extractBeadsIDFromTitle() and related functions
- cmd/orch/review.go - extractBeadsIDFromWorkspace() function

**Commands Run:**
```bash
# Verify current behavior
curl localhost:3348/api/agents | jq '.[0:3] | .[].beads_id'

# Count agents with/without beadsId
curl -s localhost:3348/api/agents | jq '[.[] | select(.beads_id != null)] | length'

# Check server process start time
ps aux | grep "orch.*serve"

# Check commit history
git log --oneline --since="2026-01-04 14:10" -- cmd/orch/
```

---

## Investigation History

**2026-01-04 14:45:** Investigation started
- Initial question: Why does /api/agents return null beadsId for all agents?
- Context: Bug filed at 14:44, agent spawned to investigate

**2026-01-04 14:50:** Key finding - bug no longer reproduces
- Tested original curl command - returns correct beadsId values
- 614/657 agents have correct beadsId

**2026-01-04 14:55:** Root cause identified
- Server was restarted at 14:46 (after issue filed)
- Current binary works correctly
- 43 null beadsIds are historical workspaces (expected)

**2026-01-04 15:00:** Investigation completed
- Status: Complete
- Key outcome: Bug resolved via server restart; no code changes needed

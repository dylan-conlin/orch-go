## Summary (D.E.K.N.)

**Delta:** `orch status` was showing stale sessions because `ListSessions(projectDir)` returns historical disk sessions, not just active ones. The fix is to call `ListSessions("")` (no directory header) to get only in-memory sessions, and to use tmux windows as the primary source of truth.

**Evidence:** Without `x-opencode-directory` header, API returns 4 sessions; with header, returns 289 sessions. After fix, `orch status` correctly shows only 18 active agents (all tmux-backed).

**Knowledge:** OpenCode's `/session` endpoint behavior depends on the `x-opencode-directory` header: without it, returns only in-memory sessions; with it, returns all historical sessions for that directory. OpenCode keeps sessions in memory after agents exit, so additional idle-time filtering is needed.

**Next:** Implementation complete. Fix committed.

**Confidence:** High (90%) - Tested on live system with 289 historical sessions.

---

# Investigation: orch status showing stale sessions as active

**Question:** Why does `orch status` show 8+ OpenCode sessions with `status:null` as 'active' when they're historical disk data, not running processes?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** og-inv-orch-status-showing-21dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: OpenCode API behavior depends on x-opencode-directory header

**Evidence:**
```bash
# Without header (in-memory only)
$ curl -s http://127.0.0.1:4096/session | jq 'length'
4

# With header (all disk sessions)
$ curl -s -H "x-opencode-directory: $(pwd)" http://127.0.0.1:4096/session | jq 'length'
289
```

**Source:** OpenCode API endpoint `/session`, tested with curl

**Significance:** The `ListSessions(projectDir)` call at `cmd/orch/main.go:1520` was adding the `x-opencode-directory` header, causing the API to return ALL historical sessions (289) instead of just in-memory sessions (4).

---

### Finding 2: OpenCode sessions have null status field

**Evidence:**
```bash
$ curl -s http://127.0.0.1:4096/session | jq '.[] | {id: .id[:20], status: .status}'
{
  "id": "ses_4bc758a0affevWoG",
  "status": null
}
```

**Source:** OpenCode API response, `pkg/opencode/types.go:52-62` (Session struct has no Status field)

**Significance:** The Session struct doesn't have a Status field because OpenCode doesn't persist session status. Status is only available via SSE events during active execution. This means we can't use API status to determine if a session is running.

---

### Finding 3: OpenCode keeps sessions in memory after agents exit

**Evidence:**
```bash
$ curl -s http://127.0.0.1:4096/session | jq '.[] | {title: .title[:40], updated_ago: ((now*1000 - .time.updated)/1000/60 | floor | tostring + " min ago")}'
{
  "title": "og-inv-quick-test-verify-21dec",
  "updated_ago": "374 min ago"
}
```

Sessions remain in-memory even 6+ hours after last activity.

**Source:** OpenCode API `/session` endpoint

**Significance:** Even in-memory sessions aren't necessarily "active". We need additional filtering (idle time) or a more authoritative source (tmux windows).

---

## Synthesis

**Key Insights:**

1. **tmux windows are the authoritative source for "active" agents** - If an agent has a tmux window, it's definitively running. OpenCode sessions can linger in memory after agents exit.

2. **API header behavior is counterintuitive** - `x-opencode-directory` header was intended for filtering to a specific project, but actually causes the API to return ALL historical sessions for that directory.

3. **Idle time filtering is necessary for OpenCode sessions** - Since OpenCode keeps sessions in memory after completion, a 30-minute idle threshold filters out completed-but-cached sessions.

**Answer to Investigation Question:**

`orch status` was showing stale sessions because:
1. `ListSessions(projectDir)` passed the project directory, adding `x-opencode-directory` header
2. This caused the API to return all 289 historical disk sessions instead of 4 in-memory sessions
3. The 4-hour idle filter wasn't strict enough for in-memory sessions that remain cached after completion

The fix:
1. Call `ListSessions("")` (no directory header) to get only in-memory sessions
2. Make tmux windows the primary source of truth
3. Add 30-minute idle filter for OpenCode sessions to catch cached-but-completed agents

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
Tested on live system with clear before/after comparison. The fix reduced displayed agents from 27 to 18, matching the actual active tmux windows.

**What's certain:**

- ✅ API header behavior causes historical session retrieval (tested with curl)
- ✅ tmux windows accurately reflect running agents (cross-verified)
- ✅ Fix correctly filters stale sessions (verified with `orch status`)

**What's uncertain:**

- ⚠️ 30-minute idle threshold might be too aggressive for long-running agents that pause
- ⚠️ Edge case: headless agents without tmux windows might be missed if idle >30min

**What would increase confidence to Very High:**

- Test with headless agents to verify they're still visible while running
- Monitor over a week to ensure no false negatives

---

## Implementation

**Changes made:**

1. Changed `client.ListSessions(projectDir)` to `client.ListSessions("")` at `cmd/orch/main.go:1519`
2. Restructured `runStatus()` to use tmux windows as primary source
3. Added 30-minute idle filter for OpenCode sessions (line 1575)
4. Added deduplication logic to avoid showing same agent from both tmux and OpenCode

**Files modified:**
- `cmd/orch/main.go` - `runStatus()` function (lines 1513-1593)

---

## Test performed

**Test:** Ran `orch status` before and after fix, compared output

**Before:**
```
SWARM STATUS
  Active:    27
  ...
8 OpenCode sessions with 6+ hour runtimes showing as "active"
```

**After:**
```
SWARM STATUS
  Active:    18
  ...
Only tmux-backed agents displayed
```

**Result:** Fix correctly filters stale sessions. Agent count matches actual running tmux windows.

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn constrain "OpenCode x-opencode-directory header returns ALL disk sessions, not just matching ones" --reason "API behavior is counterintuitive - without header returns in-memory only"
```

---

## References

**Files Examined:**
- `cmd/orch/main.go:1513-1628` - runStatus() function
- `pkg/opencode/client.go:179-207` - ListSessions() function
- `pkg/opencode/types.go:52-76` - Session and SessionTime structs

**Commands Run:**
```bash
# Check session count without directory header
curl -s http://127.0.0.1:4096/session | jq 'length'
# Result: 4

# Check session count with directory header
curl -s -H "x-opencode-directory: $(pwd)" http://127.0.0.1:4096/session | jq 'length'
# Result: 289

# Verify fix
./build/orch status
# Result: 18 active agents (matches tmux windows)
```

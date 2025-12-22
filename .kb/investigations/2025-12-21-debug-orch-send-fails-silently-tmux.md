<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `orch send` only accepted raw session IDs, but tmux-spawned agents rarely capture session IDs to workspace files.

**Evidence:** Only 1 of 100+ workspaces had `.session_id` file; `runSend` passed identifier directly to API without lookup.

**Knowledge:** The registry removal (b217e39) created a gap - `runSend` wasn't migrated to use workspace/API lookup like `runTail` was.

**Next:** Fix applied - added `resolveSessionID()` function that supports beads IDs, workspace names, and raw session IDs.

**Confidence:** High (90%) - Fix tested via build and unit tests; smoke-test pending.

---

# Investigation: orch send fails silently for tmux-based agents

**Question:** Why does `orch send` fail silently for tmux-based agents, and how should session ID lookup work?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Debugging agent (systematic-debugging skill)
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: runSend had no session ID resolution

**Evidence:** The `runSend` function (cmd/orch/main.go:1275) took the `sessionID` argument directly and passed it to the OpenCode API without any lookup logic:

```go
func runSend(serverURL, sessionID, message string) error {
    client := opencode.NewClient(serverURL)
    // ... directly uses sessionID without resolution
    if err := client.SendMessageAsync(sessionID, message); err != nil {
```

**Source:** cmd/orch/main.go:1275-1310

**Significance:** Unlike `runTail` and `runQuestion` which have sophisticated lookup logic (workspace files → API sessions → tmux windows), `runSend` assumed the user always provided a valid OpenCode session ID.

---

### Finding 2: Session ID capture fails frequently for tmux spawns

**Evidence:** 
- Only 1 workspace out of 100+ had a `.session_id` file
- `runSpawnTmux` uses `FindRecentSessionWithRetry` which often fails due to timing:
  ```go
  sessionID, _ := client.FindRecentSessionWithRetry(cfg.ProjectDir, "", 3, 500*time.Millisecond)
  // Note: We silently ignore errors here since window_id is sufficient for tmux monitoring
  ```
- When session ID capture fails, the workspace file is never written

**Source:** 
- `find .orch/workspace -name ".session_id"` → only 1 result
- cmd/orch/main.go:1060-1064 (FindRecentSessionWithRetry call)
- cmd/orch/main.go:1076-1081 (conditional WriteSessionID)

**Significance:** The registry removal (commit b217e39) created a regression - session ID was previously stored in the registry, but now relies on workspace files that are rarely written for tmux spawns.

---

### Finding 3: runTail has the correct pattern to copy

**Evidence:** `runTail` (cmd/orch/main.go:382-470) implements a complete lookup strategy:
1. Search workspace files for `.session_id` (fast path)
2. If found, use OpenCode API
3. If not found, search tmux windows by beads ID
4. Try to match OpenCode sessions by title
5. Fall back to tmux pane capture

**Source:** cmd/orch/main.go:382-470

**Significance:** The pattern for resolving beads IDs/workspace names to session IDs already exists and works correctly in `runTail`.

---

## Synthesis

**Key Insights:**

1. **Registry removal gap** - The migration to workspace-local session files didn't update `runSend` to use the new lookup pattern.

2. **Session ID capture is unreliable** - `FindRecentSessionWithRetry` often fails for tmux spawns, leaving most workspaces without session ID files.

3. **API fallback is sufficient** - The OpenCode API can find sessions by title matching, which includes the workspace name (containing beads ID).

**Answer to Investigation Question:**

`orch send` failed silently because it assumed users would always provide a raw session ID, but:
1. Most tmux-spawned agents never capture session IDs to workspace files
2. Users naturally want to use beads IDs (e.g., `orch-go-3anf`) which they see in spawn output
3. There was no lookup logic to resolve beads IDs or workspace names to session IDs

The fix adds a `resolveSessionID()` function that implements the same pattern as `runTail`:
- If identifier starts with `ses_`, use as-is
- Search workspace files for matching directories
- Search OpenCode API sessions by title match
- Search tmux windows for beads ID matches

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The root cause is clearly identified and the fix follows an existing proven pattern from `runTail`. All unit tests pass.

**What's certain:**

- ✅ `runSend` had no lookup logic - directly confirmed in code
- ✅ Session ID files are rarely written - empirically verified (1/100+)
- ✅ `runTail` pattern works correctly - has been in use
- ✅ Fix compiles and passes all tests

**What's uncertain:**

- ⚠️ No end-to-end smoke test was performed (no active agent to send to)
- ⚠️ API session matching by title depends on session still being in OpenCode memory

**What would increase confidence to Very High (95%+):**

- Real smoke test with active tmux agent
- Verify message actually arrives to agent

---

## Implementation Recommendations

**Purpose:** Document the fix that was implemented.

### Recommended Approach ⭐

**Add resolveSessionID() function** - Extract session ID lookup logic from `runTail` pattern and apply to `runSend`.

**Why this approach:**
- Follows existing pattern from `runTail` (proven to work)
- Supports multiple identifier formats (session ID, beads ID, workspace name)
- Maintains backwards compatibility with raw session IDs

**Implementation sequence:**
1. Add `resolveSessionID()` function before `runSend`
2. Update `runSend` to call `resolveSessionID()` first
3. Update command documentation to reflect new capabilities

### Implementation Details

**What was implemented:**

1. Added `resolveSessionID()` function (cmd/orch/main.go:1275-1335) that:
   - Returns raw session IDs (ses_xxx) unchanged
   - Searches workspace files for matching directories
   - Searches OpenCode API sessions by title match
   - Searches tmux windows for beads ID matches

2. Updated `runSend()` to use `resolveSessionID()`:
   - Renamed parameter from `sessionID` to `identifier`
   - Resolves identifier before using it
   - Added `identifier` to event log data

3. Updated command documentation:
   - `send` and `ask` commands now document identifier support
   - Added examples with beads IDs and workspace names

---

## References

**Files Examined:**
- cmd/orch/main.go - Main command implementations
- pkg/spawn/session.go - Session ID file handling

**Commands Run:**
```bash
# Find .session_id files in workspaces
find .orch/workspace -name ".session_id" -type f

# Check registry removal commit
git show b217e39 --stat | head -30
```

---

## Investigation History

**2025-12-21 16:55:** Investigation started
- Initial question: Why does orch send fail silently for tmux-based agents?
- Context: Two failure modes reported - session ID parsing failed (empty ID), message didn't arrive

**2025-12-21 17:00:** Root cause identified
- Found runSend has no lookup logic
- Found session ID files rarely exist for tmux spawns
- Found runTail has the correct pattern

**2025-12-21 17:10:** Fix implemented
- Added resolveSessionID() function
- Updated runSend to use it
- All tests pass

**2025-12-21 17:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Added session ID resolution to orch send, supporting beads IDs and workspace names

# Race Test 4 - Additional Test Results

## Test Performed (Second Spawn)

**Test:** Spawned a second investigation for the same task ("race test 4") at 08:25:52

**Setup:**
- First agent already running in workspace: `og-inv-race-test-20dec`
- Second spawn command: `./orch spawn investigation "race test 4 - write timestamp to race-4-checkin.txt"`

**Results:**
1. **Workspace isolation:** Created separate workspace `og-inv-race-test-write-20dec`
2. **Fire-and-forget confirmed:** Spawn returned immediately without session ID
3. **Session exists:** OpenCode shows `ses_4c36b03e3ffetHAlaDsftb3CeG` via `orch status`
4. **Event log:** `session_id: NONE` in ~/.orch/events.jsonl
5. **Beads parsing bug:** Got beads ID "open" instead of expected value
6. **File conflict:** Both agents attempted to edit same investigation file
7. **49 concurrent sessions:** High concurrency without crashes

## Race Condition Demonstrated

### Symptom
Two agents spawned for same task (beads issue orch-go-75n) created separate workspaces and both attempted to update `.kb/investigations/2025-12-20-inv-race-test-4.md`

### Evidence
- First agent: `og-inv-race-test-20dec` (updated investigation at 08:26:48)
- Second agent: `og-inv-race-test-write-20dec` (got edit conflict at 08:26:48)
- Edit tool error: "File has been modified since it was last read"

### Root Cause
1. **No session ID capture:** Tmux spawn doesn't return session ID, so no way to detect duplicate spawns
2. **Workspace names differ:** Each spawn generates unique workspace name, hiding the duplicate
3. **No beads locking:** Nothing prevents multiple agents from being spawned for same beads issue

## Conclusions

**Fire-and-forget behavior confirmed:**
- ✅ Tmux spawn returns immediately
- ✅ Session ID not captured (design choice for TUI display)
- ✅ Session exists in OpenCode but orchestrator has no reference

**Concurrent spawn works:**
- ✅ 49 concurrent sessions without crashes
- ✅ Workspace isolation maintained
- ✅ No tmux window collisions

**Race condition exists:**
- ❌ Multiple spawns for same task possible
- ❌ Agents can conflict on shared files
- ❌ Beads issue parsing bug ("open" instead of proper ID)
- ❌ No orchestrator-level duplicate detection

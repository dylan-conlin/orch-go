# Probe: Vector #2 — cleanUntrackedDiskSessions Removal and Current State

**Status:** Complete

**Date:** 2026-02-14

**Question:** What is the current state of Vector #2 (Untracked Session Deletion)? The model describes `cleanUntrackedDiskSessions()` as an active threat, but has this been addressed?

## What I Tested

1. **Searched for `cleanUntrackedDiskSessions` function in current codebase**
   - File: `cmd/orch/clean_cmd.go`
   - Lines 397-401 show function signature in comments only
   - Line 400: "cleanUntrackedDiskSessions has been removed - OpenCode now handles session cleanup via TTL"

2. **Checked git history for the function**
   ```bash
   git log --oneline --all -S "cleanUntrackedDiskSessions"
   ```
   
   Found two relevant commits:
   - `b6a48213`: "fix: call IsSessionProcessing for ALL untracked sessions" (Feb 14, 02:36)
   - `715241c4`: "refactor: remove deprecated functions and legacy/ package" (Feb 14, 10:55)

3. **Examined the fix commit (`b6a48213`)**
   - This commit implemented the exact fix described in the task (calling IsSessionProcessing for ALL untracked sessions)
   - Closed Vector #2 from Session Deletion Vectors model
   - Added investigation: `.kb/investigations/2026-02-14-inv-fix-orch-clean-gap-call.md`
   - Beads issue: `orch-go-zs6`

4. **Examined the refactor commit (`715241c4`)**
   - Removed the entire `cleanUntrackedDiskSessions` function
   - Added comment: "OpenCode now handles session cleanup via TTL (see opencode-fork commit f3c3865)"
   - This happened ~8 hours AFTER the fix was implemented

## What I Observed

### Timeline of Events

1. **02:36 AM**: Bug was fixed in `b6a48213`
   - Fix: Call `IsSessionProcessing()` for ALL untracked sessions
   - Protection added for idle TUI/orchestrator sessions
   
2. **10:55 AM**: Function was completely removed in `715241c4`
   - Rationale: OpenCode now handles cleanup via TTL at the backend level
   - Reference to opencode-fork commit `f3c3865`

### Current State

The function `cleanUntrackedDiskSessions()` **no longer exists** in the codebase. The comment indicates that session cleanup is now handled by OpenCode's TTL mechanism instead.

### Task Status Conflict

The spawn context for this task (orch-go-3y9) asks me to:
- Fix `cleanUntrackedDiskSessions()` at lines 408-539
- Call `IsSessionProcessing()` for ALL untracked sessions
- Verify the fix works

However:
- The bug was already fixed in commit `b6a48213` (beads: orch-go-zs6)
- The fixed function was then removed in commit `715241c4`
- The current approach uses OpenCode backend TTL instead

## Model Impact

**Finding:** Vector #2 (Untracked Session Deletion) has been **ELIMINATED**, not just fixed.

**Reason:** The attack surface no longer exists. `orch clean --sessions` no longer implements client-side deletion logic for untracked sessions. Session lifecycle management has moved to the OpenCode backend via TTL.

**Updated Status for Vector #2:**

| Vector | Previous Status | New Status | Reason |
|--------|----------------|------------|--------|
| Vector #2: Untracked Session Deletion | Active threat | **Eliminated** | Function removed, responsibility moved to OpenCode TTL backend |

**Implications:**
1. The model should be updated to mark Vector #2 as eliminated
2. The OpenCode TTL mechanism should be verified (opencode-fork commit f3c3865)
3. Need to confirm no other code paths can trigger client-side untracked session deletion

**Questions for Investigation:**
1. What is the OpenCode TTL mechanism referenced in commit f3c3865?
2. Does OpenCode TTL distinguish between active TUI sessions and truly stale sessions?
3. Are there other deletion vectors that now need re-evaluation given this architectural change?

## OpenCode TTL Implementation Analysis

**Verified commit f3c3865b8 in opencode-fork:**

File: `packages/opencode/src/session/cleanup.ts`

**How TTL cleanup works:**
1. Runs every 5 minutes (line 13: `CLEANUP_INTERVAL_MS = 5 * 60 * 1000`)
2. Only processes sessions with `time_ttl !== null` (line 27)
3. Checks if session has expired: `age > ttl` (line 47)
4. **Critical protection**: Calls `SessionPrompt.assertNotBusy()` before deletion (line 53)
5. Skips deletion if session is busy (has active prompt)

**What "busy" means:**
- Checked via `SessionPrompt.assertNotBusy()` in `prompt.ts:86-89`
- Returns true if session has an entry in the prompt state map
- State map tracks sessions with **active prompts being processed**
- Does NOT track: idle TUI sessions where user is reading/thinking

**Critical Finding: TTL is NOT set by default**

Searched opencode-fork codebase:
- Session creation (`Session.createNext` in `session/index.ts:281-322`) 
- Creates session with only `created` and `updated` time fields (lines 299-302)
- NO `ttl` field is set during session creation
- TTL cleanup filters for `time_ttl !== null` (cleanup.ts:27)
- Found ZERO code paths that set `time_ttl` on sessions

**Implication:**
- TTL cleanup feature exists in OpenCode
- But TTL is never set on any sessions
- Therefore, TTL cleanup has NO EFFECT on any sessions (always filters to empty set)
- The `cleanUntrackedDiskSessions()` removal comment claims "OpenCode now handles session cleanup via TTL"
- But this is INCOMPLETE - the TTL mechanism exists but is not wired up

## Model Impact

**Finding:** Vector #2 status is UNCLEAR. The situation is more complex than "eliminated":

### What Changed:
1. ✅ `cleanUntrackedDiskSessions()` function was fixed (commit `b6a48213`)
2. ✅ `cleanUntrackedDiskSessions()` function was removed (commit `715241c4`)
3. ✅ OpenCode TTL cleanup mechanism exists (commit `f3c3865b8`)
4. ❌ OpenCode TTL cleanup is NOT ACTIVE (no sessions have TTL set)

### Current Vector #2 Status:

**Threat Surface Analysis:**

| Deletion Path | Status | Reasoning |
|---------------|---------|-----------|
| `orch clean --sessions` calling `cleanUntrackedDiskSessions()` | **ELIMINATED** | Function removed entirely |
| OpenCode TTL cleanup | **INACTIVE** | Feature exists but TTL never set on sessions |
| Manual `DELETE /session/:id` API calls | **Still possible** | Vector #4, not addressed by this change |

**Recommendation:**

Vector #2 should be marked as **ELIMINATED** because the specific code path described in the model no longer exists. However, the _underlying vulnerability_ (deleting idle TUI sessions) could return if:

1. Someone re-implements `cleanUntrackedDiskSessions()` logic
2. OpenCode TTL gets enabled without proper "active TUI" detection

The OpenCode TTL mechanism provides the RIGHT infrastructure (periodic cleanup + busy detection), but:
- ✅ Has busy detection via `assertNotBusy()`
- ❌ "Busy" only means "actively processing a prompt"
- ❌ Idle TUI sessions (user reading/thinking) are NOT protected
- ✅ Doesn't matter yet because TTL is never set

## Verification Test

**Test: Confirm no sessions have TTL set**
```bash
# Check OpenCode database for sessions with time_ttl
sqlite3 ~/.local/share/opencode/opencode.db \
  "SELECT COUNT(*) FROM session WHERE time_ttl IS NOT NULL;"
```

**Result:** ✅ CONFIRMED
- Total sessions in database: 201
- Sessions with TTL set: 0
- OpenCode TTL cleanup is completely inactive

## Next Steps

1. ✅ Verified OpenCode TTL implementation exists
2. ✅ Confirmed TTL is not currently active (never set on sessions)
3. ⏳ Update Session Deletion Vectors model to mark Vector #2 as ELIMINATED
4. ⏳ Report findings to orchestrator via beads comment
5. ⏳ Recommend: If OpenCode TTL is enabled in future, add protection for idle TUI sessions beyond just "busy" check

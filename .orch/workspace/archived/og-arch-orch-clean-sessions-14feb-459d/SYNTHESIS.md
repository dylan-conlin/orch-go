# Session Synthesis

**Agent:** og-arch-orch-clean-sessions-14feb-459d
**Issue:** orch-go-3y9
**Duration:** 2026-02-14 14:30 → 2026-02-14 15:15
**Outcome:** success (with important discovery)

---

## TLDR

Task requested fixing `cleanUntrackedDiskSessions()` to call `IsSessionProcessing()` for ALL untracked sessions. Found that this bug was already fixed (commit b6a48213) and then the entire function was removed (commit 715241c4). The removal assumed OpenCode TTL would handle cleanup, but discovered TTL is never set on sessions, making the feature inactive.

---

## Delta (What Changed)

### Files Created
- `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md` - Probe documenting Vector #2 elimination and TTL investigation

### Files Modified
None - no code changes needed (bug was already fixed)

### Commits
None - task was investigation/verification only

---

## Evidence (What Was Observed)

### Git History Analysis

**Commit b6a48213** (2026-02-14 02:36):
- Fixed `cleanUntrackedDiskSessions()` to call `IsSessionProcessing()` for ALL untracked sessions
- Exactly the fix described in this task
- Closed Vector #2 from Session Deletion Vectors model
- Beads issue: orch-go-zs6

**Commit 715241c4** (2026-02-14 10:55):
- Removed `cleanUntrackedDiskSessions()` function entirely
- Added comment: "OpenCode now handles session cleanup via TTL (see opencode-fork commit f3c3865)"
- Function now only exists as commented-out signature

### OpenCode Fork Analysis

**Commit f3c3865b8** (opencode-fork):
- Added session TTL cleanup feature
- File: `packages/opencode/src/session/cleanup.ts`
- Runs every 5 minutes
- Only processes sessions with `time_ttl !== null` (line 27)
- Protects busy sessions via `SessionPrompt.assertNotBusy()` check

**Session Creation** (`session/index.ts:281-322`):
- Sessions created with only `created` and `updated` time fields
- NO `ttl` field set during creation
- No code paths found that set `time_ttl` on sessions

### Database Verification
```bash
sqlite3 ~/.local/share/opencode/opencode.db \
  "SELECT COUNT(*) FROM session WHERE time_ttl IS NOT NULL;"
```
**Result:**
- Total sessions: 201
- Sessions with TTL: 0
- OpenCode TTL cleanup is completely inactive

---

## Knowledge (What Was Learned)

### New Artifacts
- Probe: `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md`

### Decisions Made
- **Vector #2 Status**: ELIMINATED (the specific code path no longer exists)
- **TTL Status**: Feature exists but is inactive (no sessions have TTL set)

### Constraints Discovered
- OpenCode TTL "busy" detection only covers sessions actively processing prompts
- Idle TUI sessions (user reading/thinking) would NOT be protected by busy check
- This doesn't matter yet because TTL is never set on sessions

### Critical Finding: Incomplete Migration

The removal of `cleanUntrackedDiskSessions()` was justified by "OpenCode now handles session cleanup via TTL", but this is incomplete:

1. ✅ OpenCode TTL cleanup mechanism exists
2. ✅ Has protection for busy sessions
3. ❌ TTL is never actually set on any sessions
4. ❌ Feature is completely inactive (0 of 201 sessions have TTL)

This creates a gap: the old cleanup logic is gone, but the new mechanism isn't active.

---

## Next (What Should Happen)

**Recommendation:** close (with escalation note for orchestrator)

### If Close
- [x] All deliverables complete (probe created and documented)
- [x] Verification test run (database query confirmed 0 TTL sessions)
- [x] Probe file has `Status: Complete`
- [x] Ready for `orch complete orch-go-3y9`

### Escalation Note for Orchestrator

**Discovery:** OpenCode TTL feature exists but is not being used.

**Question:** Should OpenCode TTL be enabled? If so:
1. What TTL value should be set on sessions?
2. Should TUI/orchestrator sessions get different TTL than worker sessions?
3. How should the "busy" detection be enhanced to protect idle TUI sessions?

**Not urgent:** Vector #2 is eliminated (old code path removed). TTL activation is a future enhancement, not a bug fix.

---

## Unexplored Questions

**Questions that emerged during this session:**

1. **Why was TTL added if it's never set?**
   - Commit f3c3865b8 added full TTL infrastructure
   - But no code actually sets `time_ttl` on sessions
   - Was this feature added for future use?

2. **Should idle TUI sessions be protected differently?**
   - Current "busy" check only covers active prompts
   - User reading/thinking in TUI = idle = not protected
   - If TTL is enabled, need better protection for TUI sessions

3. **Are there other session cleanup mechanisms?**
   - `orch clean --sessions` removed untracked deletion logic
   - OpenCode TTL exists but is inactive
   - How are stale sessions being cleaned up now?

**What remains unclear:**

- Intended use case for OpenCode TTL feature
- Why the gap between adding TTL infrastructure and actually using it
- Whether session accumulation (201 sessions) is a problem

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5
**Workspace:** `.orch/workspace/og-arch-orch-clean-sessions-14feb-459d/`
**Probe:** `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md`
**Beads:** `bd show orch-go-3y9`

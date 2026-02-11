# Session Synthesis

**Agent:** og-inv-silent-agent-session-11feb-8564
**Issue:** orch-go-i8vte
**Duration:** 2026-02-11 13:02 → 2026-02-11 14:30
**Outcome:** success

---

## TLDR

Investigated why 40% of OpenCode agent sessions die without crash signal. Found that dead session detection infrastructure exists but checks session creation time instead of activity time, causing crashed sessions to appear "alive" for up to 6 hours. Root cause is 4 detection gaps working together: (1) spawn doesn't capture session ID, (2) detector checks wrong signal, (3) no crash watchdog, (4) state DB never reconciled. Recommended fix: change HasExistingSession() to check activity via IsSessionActive() instead of Time.Created - a ~5 line change.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-11-inv-silent-agent-session-death-root.md` - Complete root cause analysis with 4 findings and recommended fix

### Files Modified
- None (investigation-only session)

### Commits
- `6766faa5` - investigation: root cause of silent agent session deaths

---

## Evidence (What Was Observed)

### Finding 1: Fire-and-forget tmux spawn
- `pkg/spawn/claude.go:88-93` - SpawnResult contains only window metadata, no SessionID field
- `pkg/opencode/session.go:287-330` - Session ID discovered via time-based polling (FindRecentSession) matching directory + created<30s
- Creates race condition: spawn can succeed while session creation fails silently

### Finding 2: No crash detection
- `pkg/opencode/monitor.go:193-247` - SSE monitor only processes session.status events for busy→idle (normal completion)
- No watchdog checking if OpenCode sessions/processes actually die
- SSE connection loss triggers reconnection but doesn't detect which sessions died during outage

### Finding 3: State DB is cache without reconciliation
- `pkg/state/db.go:3-34` - Explicit contract: "state.db is a spawn-time projection cache", NOT source of truth
- No reconciliation loop to detect when cached state (agent active) diverges from reality (session dead)
- Crashed sessions remain marked as active indefinitely

### Finding 4: Dead session detection checks creation time, not activity
- `pkg/daemon/session_dedup.go:90-92` - HasExistingSession() checks `Time.Created`, uses `age := now.Sub(createdAt); if age <= c.config.MaxAge` (6 hours)
- Does NOT call IsSessionActive() which checks Time.Updated within 30min window
- Session that crashes 5min after spawn will be considered "alive" for 6 hours

### Tests Run
```bash
# Code review - verified spawn doesn't capture session ID
grep -A 10 "SpawnResult" pkg/spawn/claude.go

# Code review - verified detector checks creation time
grep -A 5 "Time.Created" pkg/daemon/session_dedup.go

# Code review - verified IsSessionActive exists but isn't used
grep "IsSessionActive" pkg/daemon/*.go
```

---

## Verification Contract

- **Spec:** Not applicable (investigation-only, no code changes)
- **Key outcomes:**
  - Root cause identified: detector checks creation time instead of activity
  - 4 detection gaps documented with file:line references
  - Recommended fix with implementation sequence provided

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-11-inv-silent-agent-session-death-root.md` - Root cause analysis

### Decisions Made
- **Recommended fix:** Change HasExistingSession() to check activity time instead of creation time
- **Rationale:** Minimal change (~5 lines), reuses existing IsSessionActive() API, fixes most common failure mode
- **Alternative approaches considered:** Process watchdog, state DB reconciliation, spawn improvements - all orthogonal and can come later

### Key Insights
- **System has detection infrastructure that doesn't detect crashes** - Dead session detector exists but checks wrong signal (creation vs activity)
- **Multiple gaps compound** - Session crashes → SSE emits nothing → detector sees creation time and thinks alive → state DB never reconciled → phantom persists
- **Prior work exists** - git log shows `feat: add dead session detection to daemon` (f7c5bdf7) but implementation checks wrong timestamp

### Constraints Discovered
- Dead session detection relies on daemon running (daemon crash means no detection)
- 6-hour MaxAge window means crashed sessions can appear alive for hours
- State DB explicitly designed as cache, not authority - reconciliation would require new infrastructure

### Externalized via `kb`
- Investigation file created - contains full analysis, all 4 findings, recommendations
- No quick commands used (investigation is the externalization)

---

## Issues Created

**Discovered work tracked during this session:**

No new issues created - the recommendation is to fix the existing dead session detection infrastructure (pkg/daemon/session_dedup.go:90-92). This can be done as a direct implementation without creating a separate issue, or as part of the broader session reliability work (likely already tracked).

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with root cause and fix recommendation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-i8vte`

### Implementation Guidance (for next agent or orchestrator)

**Quick win:** Fix HasExistingSession() to check activity instead of creation
```go
// pkg/daemon/session_dedup.go:68-98
// BEFORE: age := now.Sub(createdAt); if age <= c.config.MaxAge
// AFTER: Check IsSessionActive instead of age calculation

if sessionBeadsID != beadsID {
    continue
}

// Use OpenCode client to check if session is actively running
updatedAt := time.Unix(s.Time.Updated/1000, 0)
if now.Sub(updatedAt) <= 30*time.Minute {
    return true
}
```

**Test plan:**
1. Start agent session
2. Kill OpenCode session (simulate crash)
3. Wait for next dead session detection cycle (default: 10min)
4. Verify issue marked as "DEAD SESSION:" and reset to open

**Follow-up improvements (orthogonal):**
- Fix spawn to capture session ID synchronously (eliminates Finding 1)
- Add process-level watchdog using pkg/process/ledger.go (catches OOM kills)
- Add state DB reconciliation loop (fixes Finding 3 comprehensively)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

1. **What's the actual crash rate breakdown?** - Task says 40% die silently, but don't know: how many are process crashes (OOM) vs session-level failures vs spawn failures?

2. **Is daemon actually running and detecting?** - Saw dead session detection config but didn't verify daemon is active or check logs for detection runs

3. **Why 6 hour MaxAge window?** - `pkg/daemon/session_dedup.go:34` sets MaxAge = 6 hours. Could this be reduced to detect faster?

4. **Can state DB become authoritative?** - `pkg/state/db.go` warns against it without reconciliation. What would that reconciliation look like?

**What remains unclear:**

- Whether most crashes happen at spawn-time (Finding 1) vs mid-flight (Finding 4) - would change priority of fixes
- Whether the fix should check Time.Updated directly or call IsSessionActive() - latter is cleaner but adds HTTP call overhead
- How to handle cases where OpenCode server itself is down (API check would fail)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-5-20250929
**Workspace:** `.orch/worktrees/og-inv-silent-agent-session-11feb-8564/`
**Investigation:** `.kb/investigations/2026-02-11-inv-silent-agent-session-death-root.md`
**Beads:** `bd show orch-go-i8vte`

# Session Synthesis

**Agent:** og-arch-analyze-opencode-session-11jan-5900
**Issue:** orch-go-blz1p
**Duration:** 2026-01-11 19:45 → 2026-01-11 20:45
**Outcome:** success

---

## TLDR

Analyzed OpenCode session accumulation leak (266 sessions vs ~29 expected) and designed two-tier cleanup strategy: event-based cleanup for normal lifecycle (already exists via abandon/complete) + automatic periodic background cleanup via daemon extension to catch orphaned sessions.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-11-design-opencode-session-cleanup-mechanism.md` - Complete design investigation with findings, synthesis, and implementation recommendations

### Files Modified
- None (design phase only, no code changes)

### Commits
- `b6f1d702` - architect: OpenCode session cleanup - design two-tier cleanup strategy (event-based + periodic)

---

## Evidence (What Was Observed)

### Current Cleanup Mechanisms
- `orch abandon` calls DeleteSession (cmd/orch/abandon_cmd.go:228) - verified by code reading
- `orch complete` calls DeleteSession (cmd/orch/complete_cmd.go:576) - verified by code reading  
- `orch clean --sessions` provides bulk cleanup (cmd/orch/clean_cmd.go:1032-1115) - verified by code reading
- All three mechanisms require workspace context (.session_id file)

### Accumulation Evidence
- Beads issue reports 266 active sessions vs ~29 expected (from bd show orch-go-blz1p)
- Gap of ~237 sessions indicates lifecycle holes
- `orch doctor --sessions` shows orphaned sessions (sessions without workspaces)
- OpenCode's ListDiskSessions returns ALL persisted sessions, not just active ones (pkg/opencode/client.go:716-748)

### Cleanup Gaps Discovered
- No automatic background cleanup - requires manual invocation
- Cleanup depends on workspace files - orphaned sessions invisible
- DeleteSession failures are non-fatal warnings - silent accumulation
- No daemon integration - cleanup not scheduled

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-11-design-opencode-session-cleanup-mechanism.md` - Design investigation documenting the problem, exploring solutions, and recommending two-tier cleanup strategy

### Decisions Made
- **Two-tier cleanup pattern:** Event-based cleanup (existing) handles normal lifecycle; periodic background cleanup (new) catches orphans
- **Daemon extension:** Extend orch daemon to run automatic cleanup every 6 hours
- **Age threshold:** Default 7 days for session deletion (conservative to prevent false positives)
- **Leverage existing code:** Reuse cleanStaleSessions function from clean command

### Constraints Discovered
- OpenCode persists sessions to disk indefinitely - explicit deletion required
- Workspace coupling creates blind spots for cleanup logic
- Daemon runs 24/7 - ideal for background maintenance tasks
- Silent failures prevent detection of cleanup issues

### Key Insights
1. **Cleanup is event-driven, not lifecycle-driven** - Sessions deleted on explicit events (abandon/complete), not automatically based on lifecycle state
2. **Workspace coupling creates blind spots** - Orphaned sessions from failed spawns, manual creation, or corrupted workspaces never get cleaned up
3. **Silent accumulation prevents detection** - DeleteSession failures are warnings, not errors; sessions accumulate silently until manual investigation

---

## Next (What Should Happen)

**Recommendation:** close (design complete, ready for implementation phase)

### Implementation Sequence
1. **Extract cleanStaleSessions to pkg/cleanup/sessions.go** - Make function reusable by both CLI and daemon
2. **Add scheduler to daemon** - Simple goroutine with time.Ticker running cleanup every 6 hours
3. **Add config options** - New section in ~/.orch/config.yaml: `cleanup.sessions.{enabled, interval, age_days, preserve_orchestrator}`
4. **Add observability** - Log each cleanup run to daemon.log with timestamp and deleted count

### Success Criteria for Implementation
- ✅ Session count stabilizes at ~29 after 7 days of running
- ✅ No active session deletion (IsSessionProcessing check prevents)
- ✅ Daemon stays responsive (cleanup runs in background without blocking)
- ✅ Observable (logs show cleanup runs and results)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does OpenCode decide when to persist sessions to disk? (Understanding this could reveal additional cleanup opportunities)
- What's the performance cost of deleting 266 sessions at once? (Might need batching if slow)
- Should cleanup handle sessions from multiple project directories? (Current implementation only handles global sessions)

**Areas worth exploring further:**
- OpenCode session persistence behavior and lifecycle
- Cross-project session tracking and cleanup
- Performance benchmarking of session deletion at scale

**What remains unclear:**
- Whether 6-hour interval and 7-day threshold are optimal (chosen as reasonable defaults but not empirically validated)
- Whether daemon scheduler is reliable under long uptimes and restarts (assumed time.Ticker is reliable but not tested)

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4
**Workspace:** `.orch/workspace/og-arch-analyze-opencode-session-11jan-5900/`
**Investigation:** `.kb/investigations/2026-01-11-design-opencode-session-cleanup-mechanism.md`
**Beads:** `bd show orch-go-blz1p`

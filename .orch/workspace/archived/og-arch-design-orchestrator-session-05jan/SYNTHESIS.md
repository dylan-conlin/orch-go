# Session Synthesis

**Agent:** og-arch-design-orchestrator-session-05jan
**Issue:** orch-go-qx6q
**Duration:** 2026-01-05 10:48 → 2026-01-05 11:30
**Outcome:** success

---

## TLDR

Designed orchestrator session lifecycle without beads tracking. Key insight: beads tracks work items (spawn→task→complete), while orchestrator sessions are conversations (start→interact→end). Recommended workspace-based session registry as replacement for vestigial beads issue creation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md` - Full investigation with five findings, synthesis, and implementation recommendations
- `.kb/decisions/2026-01-05-orchestrator-lifecycle-without-beads.md` - Decision record with semantic mismatch analysis and implementation phases

### Files Modified
- `.orch/features.json` - Added feat-035 through feat-038 for implementation phases

### Commits
- (pending commit with all changes)

---

## Evidence (What Was Observed)

### Current State Analysis

- `pkg/spawn/orchestrator_context.go:213` - "Orchestrators do NOT write .beads_id" - beads ID is already skipped
- `cmd/orch/complete_cmd.go:111` - `isOrchestratorWorkspace()` already detects orchestrators separately
- `cmd/orch/shared.go:299-312` - Orchestrator detection uses `.orchestrator` marker files
- `pkg/verify/check.go` - Already has orchestrator-specific verification path that skips beads

### Key Finding: Beads Usage is Vestigial

Beads issue is created on orchestrator spawn but then:
1. No phase comments are reported
2. No .beads_id file is written
3. Completion uses workspace name, not beads ID
4. Verification checks SESSION_HANDOFF.md, not Phase: Complete

This confirms the current implementation already treats orchestrators differently - the beads issue creation is the only remaining artifact that doesn't fit.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md` - Full investigation with semantic mismatch analysis
- `.kb/decisions/2026-01-05-orchestrator-lifecycle-without-beads.md` - Decision record with implementation plan

### Decisions Made
- **Workspace-based session registry** over alternatives (workspace scanning, extending beads, tmux registry) because:
  - O(1) lookup vs O(n) scan
  - No semantic mismatch with beads
  - Works for both tmux and headless modes

### Constraints Discovered
- Beads is semantically wrong for sessions - it tracks issues with dependencies, priority, assignees
- Sessions need simpler tracking - just identity, status, and completion detection
- Workspace files already contain all identity information needed

### Key Insight (Session Amnesia Principle Applied)
The semantic mismatch between beads (work items) and sessions (conversations) is a case of "evolve by distinction" - recognizing that two things conflated together should be separated.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + decision + feature list)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-qx6q`

### Implementation Sequence (for orchestrator to spawn)

**Phase 1:** feat-035 - Session registry (pkg/session/registry.go)
- Foundation for all other phases
- JSON file at `~/.orch/sessions.json`
- Lock file for concurrent access

**Phase 2:** feat-036 - Skip beads in spawn
- Skip `bd create` when IsOrchestrator=true
- Register in session registry instead

**Phase 3:** feat-037 - Add to status
- Include orchestrator sessions from registry
- Show alongside worker agents

**Phase 4:** feat-038 - Unregister on complete
- Remove from registry after completion
- Preserve transcript export

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should headless orchestrator sessions be supported? (Currently orchestrators default to tmux)
- How should orphaned sessions be cleaned up if orchestrator crashes?
- Should registry include goal/focus for richer status display?

**Areas worth exploring further:**
- Cross-project orchestrator session tracking (orchestrator in orch-go managing agents in other repos)
- Session history for pattern analysis (beyond just active sessions)

**What remains unclear:**
- Whether existing workspaces with .beads_id should be migrated or just left as-is

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-orchestrator-session-05jan/`
**Investigation:** `.kb/investigations/2026-01-05-inv-design-orchestrator-session-lifecycle-without.md`
**Beads:** `bd show orch-go-qx6q`

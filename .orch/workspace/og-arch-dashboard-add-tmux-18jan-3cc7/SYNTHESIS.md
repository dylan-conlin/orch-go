# Session Synthesis

**Agent:** og-arch-dashboard-add-tmux-18jan-3cc7
**Issue:** orch-go-f8yfr
**Duration:** 2026-01-18 → 2026-01-18
**Outcome:** success

---

## TLDR

Designed the integration of tmux session data alongside OpenCode sessions in the dashboard. Recommends incremental enhancement using beads Phase for status, pane activity for dead detection, and .spawn_time for runtime - leveraging existing infrastructure rather than introducing new state tracking.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-design-dashboard-add-tmux-session-visibility.md` - Complete design investigation with decision forks, substrate consultation, and implementation recommendations

### Files Modified
- None (design-only task)

### Commits
- (pending commit of investigation file)

---

## Evidence (What Was Observed)

- serve_agents.go:546-591 already has tmux fallback that adds windows not in OpenCode sessions
- Tmux agents have status hardcoded to "active" - no dead detection or completion detection
- Beads batch fetch already includes tmux agent beads IDs (lines 578-588)
- .spawn_time file is already written during spawn workflow
- Four-Layer State Model establishes beads as source of truth for completion
- Priority Cascade model already uses Phase check as highest priority
- Claude CLI doesn't expose token usage - this is an architectural constraint

### Tests Run
```bash
# Verified investigation file structure
cat .kb/investigations/2026-01-18-design-dashboard-add-tmux-session-visibility.md
# File created with complete design
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-design-dashboard-add-tmux-session-visibility.md` - Design for tmux visibility integration

### Decisions Made
- Keep tmux as fallback data source (not first-class) - matches Graceful Degradation principle
- Use beads Phase as primary status source, pane activity as secondary - matches Priority Cascade
- Accept no token visibility for Claude CLI agents - architectural constraint, not fixable gap

### Constraints Discovered
- Claude CLI (Max subscription) doesn't expose per-session token usage
- Pane content parsing is fragile to TUI format changes
- Registry-based tracking was previously removed due to state drift

### Externalized via `kn`
- N/A - investigation captures all design decisions

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with design)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-f8yfr`

### Implementation Follow-up (for orchestrator)

The design is ready for implementation. Suggested approach:

**Phase 1: Beads Phase for Status** (priority)
- In serve_agents.go, for tmux-only agents, look up Phase from batch-fetched beads data
- Set status based on Phase: Complete
- ~50 lines changed

**Phase 2: Pane Activity Detection**
- Add GetPaneActivityAge() to tmux package
- Use 3-minute threshold for dead detection
- ~100 lines new, 30 lines changed

**Phase 3: Runtime from .spawn_time**
- Read spawn_time from workspace for tmux agents
- Calculate runtime display
- ~30 lines new

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does dashboard frontend handle null/missing fields like tokens and session_id?
- What's the performance impact of pane capture for 10+ tmux windows?
- Could workspace file modification times provide better activity detection than pane parsing?

**Areas worth exploring further:**
- Dashboard UI graceful degradation for partial agent data
- Pane content caching to reduce tmux command calls

**What remains unclear:**
- Exact Claude TUI format for "processing" vs "waiting" states
- Whether workspace mtime is a reliable fallback for activity detection

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-dashboard-add-tmux-18jan-3cc7/`
**Investigation:** `.kb/investigations/2026-01-18-design-dashboard-add-tmux-session-visibility.md`
**Beads:** `bd show orch-go-f8yfr`

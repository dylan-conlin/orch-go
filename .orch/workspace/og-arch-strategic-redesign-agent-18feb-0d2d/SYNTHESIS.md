# Session Synthesis

**Agent:** og-arch-strategic-redesign-agent-18feb-0d2d
**Issue:** orch-go-1058
**Duration:** 2026-02-18 → 2026-02-18
**Outcome:** success

---

## TLDR

The agent observability layer has 5 interacting data sources that are reconciled at query time, and **the reconciliation model is fundamentally backwards**. The current approach discovers agents from infrastructure (OpenCode sessions, tmux windows) then tries to match them to work (beads issues). The fix is to invert this: start from beads issues (the work) and find infrastructure, eliminating the multi-source reconciliation that causes every bug.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-18-design-agent-observability-rethink.md` - Strategic redesign investigation with architectural recommendation

### Files Modified
- None (investigation/design only)

### Commits
- Pending: investigation file + synthesis

---

## Evidence (What Was Observed)

1. **serve_agents.go is 1400+ lines** of reconciliation logic attempting to merge 5 data sources: workspace scanning, tmux windows, OpenCode sessions, beads comments, and a dead registry.

2. **Three bugs from Feb 18 all stem from reconciliation failures:**
   - 84/85 empty metadata: path from OpenCode → workspace cache → beads enrichment fails silently
   - price-watch "dead" agents: OpenCode --attach uses server cwd, so session.directory is wrong for cross-project agents
   - tmux agents invisible: workspace scan runs before tmux scan and claims agents as "completed"

3. **Agent lifecycle state model confirms the diagnosis:** "The reconciliation burden comes from treating infrastructure as state." When we ask OpenCode "is this agent done?", we're asking the wrong source.

4. **Cross-project visibility fix from Jan 7 didn't stick:** kb projects integration was added, but `getKBProjects()` can fail silently and return empty slice.

5. **Registry/state.db is dead:** Not written to, 6 days stale. Should be deleted.

### Tests Run
- None (design investigation, no code changes)

---

## Knowledge (What Was Learned)

### Key Architectural Insight

**Current model (broken):**
```
Sessions/Tmux Windows (infrastructure)
    ↓ discover agents
    ↓ extract beads_id from title
    ↓ query workspace for metadata
    ↓ query beads for phase/status
    ↓ reconcile conflicts
Result: O(n*m) reconciliation with many edge cases
```

**Proposed model (correct):**
```
Beads Issues (work)
    ↓ list status=in_progress
    ↓ find workspace by beads_id
    ↓ get session_id from workspace
    ↓ check liveness from OpenCode
Result: O(n) single-pass discovery
```

### Decisions Made
- **Invert discovery model:** Start from beads (work), not sessions (infrastructure)
- **Workspace is source of truth for project_dir:** Not OpenCode session.directory (broken for cross-project)
- **Registry must die:** 6 days stale, not written to, causes drift

### Constraints Discovered
- OpenCode `--attach` uses server cwd for session.directory — cannot change without upstream PR
- Sessions persist indefinitely (no TTL) — cleanup is our responsibility
- x-opencode-directory scopes all session operations — cross-project requires explicit header
- Beads comments are the only reliable phase source — agents must use bd comment protocol

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Issue 1:** Fix cross-project visibility (immediate pain relief)
**Skill:** systematic-debugging
**Context:**
```
getKBProjects() fails silently and returns empty slice when kb CLI unavailable.
Fix error handling and add debug logging to workspace cache building.
Also fix beads comments fetch to use correct project context from beadsProjectDirs map.
```

**Issue 2:** Implement work-centric agent discovery
**Skill:** feature-impl
**Context:**
```
New function discoverAgentsFromWork() that starts from beads issues with status=in_progress,
finds workspaces by beads_id, then checks OpenCode for liveness. Run alongside existing
code with feature flag for A/B testing. See .kb/investigations/2026-02-18-design-agent-observability-rethink.md
```

**Issue 3:** Delete dead code (registry, state.db)
**Skill:** feature-impl
**Context:**
```
Registry and state.db haven't been written to for 6 days. Remove all references from
pkg/session/, cmd/orch/serve_agents.go, and any other files. Delete ~/.orch/registry.json
and ~/.orch/state.db.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should we file an upstream PR to OpenCode for session directory override in --attach mode?
- Should kb projects have a Go library to avoid CLI invocation overhead?
- What's the performance impact of beads-centric discovery with 100+ in_progress issues?

**Areas worth exploring further:**
- SSE event-driven updates vs polling for real-time dashboard
- Graceful degradation UI (showing which data sources are unavailable)

**What remains unclear:**
- Exact error path when `getKBProjects()` fails — need to add logging to trace

---

## Session Metadata

**Skill:** architect
**Model:** Claude Opus
**Workspace:** `.orch/workspace/og-arch-strategic-redesign-agent-18feb-0d2d/`
**Investigation:** `.kb/investigations/2026-02-18-design-agent-observability-rethink.md`
**Beads:** `bd show orch-go-1058`

# Session Synthesis

**Agent:** og-arch-design-dashboard-replay-17jan-6087
**Issue:** orch-go-gy1o4.1.9
**Duration:** 2026-01-17 21:21 -> 2026-01-17 22:30
**Outcome:** success

---

## TLDR

Design investigation complete for Dashboard Replay UI and Activity-Aware Resumption pattern. Recommends debugger-style stepping for replay navigation, two-tier filtering (reasoning vs execution meta-filters), and API endpoint for predecessor activity access (not context embedding).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-dashboard-replay-ui-activity.md` - Complete design investigation with three design forks resolved

### Files Modified
- None (design-only investigation)

### Commits
- Pending commit with investigation document

---

## Evidence (What Was Observed)

- Activity-tab already has event grouping, type filtering, expand/collapse - foundation for replay exists (`activity-tab.svelte:23, 258-311`)
- ACTIVITY.json format matches SSEEvent interface exactly - no transformation needed (`export.go:25-30`, `agents.ts:116-140`)
- Session messages endpoint already falls back to ACTIVITY.json for completed agents (`serve_agents.go:1485-1499`)
- Existing constraint: "Activity state should be ephemeral in UI" - replay should be distinct mode
- Existing constraint: "Dashboard must be fully usable at 666px width" - rules out timeline scrubber

### Tests Run
```bash
# Read and verified codebase structure
kb context "activity feed replay resumption"
# Retrieved relevant constraints, decisions, and guides

# Verified existing implementations
grep -r "handleSessionMessages" cmd/orch/
# Confirmed ACTIVITY.json fallback exists
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-dashboard-replay-ui-activity.md` - Design investigation resolving three forks

### Decisions Made
- **Debugger stepping > timeline scrubber** - Fits 666px constraint, preserves event grouping context, enables precise analysis
- **Two-tier filtering** - Meta-filters (Reasoning/Execution) plus existing granular filters for progressive disclosure
- **API endpoint for predecessor activity** - Summary reference pattern instead of full activity embedding in spawn context

### Constraints Discovered
- Activity is ephemeral in live mode but replay is for completed agents - distinct cognitive modes
- Token cost makes full ACTIVITY.json embedding impractical - need summary + reference pattern

### Externalized via `kn`
- Design establishes pattern: predecessor activity accessed via API, not context embedding
- Recommend promoting to decision if orchestrator accepts

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Phases

**Phase 1: Replay UI (Stepping Controls)**
- Files: `web/src/lib/components/agent-detail/activity-tab.svelte`
- Add replayMode state, currentEventIndex, Prev/Next buttons, Play/Pause

**Phase 2: Two-Tier Filtering**
- Files: `web/src/lib/components/agent-detail/activity-tab.svelte`
- Add Reasoning/Execution meta-filters, preserve granular filters

**Phase 3: Activity-Aware Resumption**
- Files: `cmd/orch/serve_agents.go`, `pkg/spawn/context.go`, `pkg/spawn/config.go`
- Add `/api/agent/:workspaceName/activity-summary` endpoint
- Add `--predecessor` flag to orch spawn

### If Spawn Follow-up
**Issue:** Implement Dashboard Replay UI (Phase 1 - Stepping Controls)
**Skill:** feature-impl
**Context:**
```
Implement debugger-style replay for completed agent activity. Add replayMode state,
currentEventIndex for position tracking, Prev/Next buttons, and Play/Pause auto-advance.
Design reference: .kb/investigations/2026-01-17-inv-design-dashboard-replay-ui-activity.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Auto-play timing: What speed is comfortable for reviewing activity? 500ms? 1s? User-configurable?
- Should stepping work on event groups (tool+result) or individual events?

**Areas worth exploring further:**
- Search/goto feature for jumping to specific tool calls in long sessions
- Integration with existing "Visual hierarchy" issue (orch-go-gy1o4.1.3)

**What remains unclear:**
- Performance of stepping through 500+ events - may need virtualization
- How to handle very long sessions (pagination vs virtualization)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-dashboard-replay-17jan-6087/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-dashboard-replay-ui-activity.md`
**Beads:** `bd show orch-go-gy1o4.1.9`

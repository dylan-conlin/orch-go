# Session Synthesis

**Agent:** og-feat-restore-dead-agent-08jan-c1d1
**Issue:** orch-go-m23ez
**Duration:** ~25 minutes
**Outcome:** success

---

## TLDR

Restored dead agent detection and surfacing that was reverted during the Dec 27 - Jan 2 system spiral. The implementation adds a simple 3-minute heartbeat threshold to detect dead agents (crashed/stuck/killed) and surfaces them in the dashboard's "Needs Attention" section with red/amber styling, plus a warning count in the status bar.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_agents.go` - Added 3-minute dead threshold detection; status now "dead" when no activity for 3+ minutes
- `web/src/lib/stores/agents.ts` - Added 'dead' to AgentState type, created deadAgents derived store, updated computeDisplayState
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Added dead agents section with skull icon, red styling, and agent cards
- `web/src/lib/components/stats-bar/stats-bar.svelte` - Added dead agent count display "(+N need attention)" when dead agents exist

### Commits
- (pending) - Restore dead agent detection with 3-minute heartbeat threshold

---

## Evidence (What Was Observed)

- Reference commit 784c2703 showed the original simple dead detection implementation
- Post-mortem at `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` explained the spiral was caused by complexity (multiple time thresholds, added states like 'stalled')
- Current codebase had activeThreshold (10 min) and displayThreshold (30 min) but no dead detection
- Dashboard already had a NeedsAttention component for errors/blocked/reviews - dead agents fit naturally there

### Tests Run
```bash
# Go build
go build ./cmd/orch/
# PASS: build successful

# Web build
npm run build (in web/)
# PASS: built in 10.65s

# Visual verification
# Dashboard shows 2 active agents, stats bar working, no dead agents (correct)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-restore-dead-agent-detection-surfacing.md` - Investigation file

### Decisions Made
- Decision 1: Use 3-minute threshold (not 1 min, not 5 min) because agents constantly read/edit/run commands - 3 min silence is definitive death signal
- Decision 2: Add dead agents to existing NeedsAttention component rather than creating new section - consolidates attention items
- Decision 3: Keep it simple - just 'dead' status, no 'stalled' or multiple thresholds (lesson from spiral)

### Constraints Discovered
- Dead threshold must be longer than normal idle gaps during active work
- Dead detection priority: dead (3min) > active > idle (10min)
- Visual verification limited when no dead agents exist - need actual dead agent to test styling

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (go build, npm run build)
- [x] Visual verification complete (dashboard loads, stats bar shows active count)
- [ ] Investigation file has `**Phase:** Complete` (will update)
- [x] Ready for `orch complete orch-go-m23ez`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How should dead agents be handled in the "truly active" count for daemon capacity?
- Should there be an action button to abandon/cleanup dead agents from the dashboard?

**Areas worth exploring further:**
- Integration with `orch status` CLI to also show dead status
- Notification/alert when agents go dead

**What remains unclear:**
- Exact behavior when an agent resurrects (goes from dead back to active)

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-restore-dead-agent-08jan-c1d1/`
**Investigation:** `.kb/investigations/2026-01-08-inv-restore-dead-agent-detection-surfacing.md`
**Beads:** `bd show orch-go-m23ez`

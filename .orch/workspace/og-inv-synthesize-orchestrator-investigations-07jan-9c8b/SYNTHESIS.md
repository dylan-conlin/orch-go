# Session Synthesis

**Agent:** og-inv-synthesize-orchestrator-investigations-07jan-9c8b
**Issue:** orch-go-qv8cc
**Duration:** 2026-01-07
**Outcome:** success

---

## TLDR

Synthesized 12 new orchestrator investigations from Jan 6-7, 2026 into the orchestrator session management guide, identifying 5 major themes: checkpoint discipline, frame collapse detection, stats correlation, dashboard context-following, and interactive orchestrator value. Guide updated with new sections covering all themes.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-inv-synthesize-orchestrator-investigations.md` - This synthesis investigation

### Files Modified
- `.kb/guides/orchestrator-session-management.md` - Updated with:
  - Checkpoint Discipline section (2h/3h/4h thresholds)
  - Session Registry Status Updates section
  - Interactive vs Spawned Workspaces table
  - Dashboard Context Following section
  - 5 new Common Problems entries (tmux session ID, stats completion rate, interactive workspaces, frame collapse detection)
  - 2 new Key Decisions (interactive orchestrator value, checkpoint discipline)
  - History updated to reflect Jan 7 synthesis

### Commits
- `95983781` - investigation: synthesize-orchestrator-investigations - checkpoint

---

## Evidence (What Was Observed)

- 47 total orchestrator investigations found via `glob .kb/investigations/*orchestrator*.md`
- Prior synthesis (Jan 6) covered 28 investigations
- 12 new investigations from Jan 6-7, 2026 analyzed in detail
- Key themes extracted: dashboard context, stats correlation, checkpoint discipline, frame collapse detection, session registry, interactive workspaces

### Tests Run
```bash
# Verified investigation count
glob .kb/investigations/*orchestrator*.md  # 47 files

# Verified guide update
# Read and edited .kb/guides/orchestrator-session-management.md
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-synthesize-orchestrator-investigations.md` - Synthesis of Jan 6-7 findings

### Decisions Made
- Guide-first maintenance: Future orchestrator investigations should update the guide as primary artifact
- Monthly synthesis cadence recommended after 10-15 new investigations

### Constraints Discovered
- Orchestrator low completion rates are BY DESIGN (coordination sessions, not tasks)
- Frame collapse requires EXTERNAL detection - orchestrators can't see their own collapse
- Interactive orchestrators serve 3 functions daemon CANNOT: goal refinement, frame correction, synthesis

### Key Themes Identified

| Theme | Key Finding | Source Investigation |
|-------|-------------|---------------------|
| Dashboard context | API now supports project_dir parameter for multi-project | 2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md |
| Stats correlation | 0% completion is BY DESIGN for coordination skills | 2026-01-07-design-orch-stats-miscounts-orchestrator-meta.md |
| Checkpoint discipline | 2h/3h/4h thresholds via orch session status | 2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md |
| Frame collapse | Multi-layer detection: skill + handoff + plugin | 2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md |
| Interactive value | NOT compensation for daemon - serves synthesis | 2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md |

### Externalized via `kn`
- N/A - synthesis consolidates existing decisions, doesn't establish new patterns

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Guide updated with new findings
- [x] Ready for `orch complete orch-go-qv8cc`

### Follow-up Items (for future issues)
- Add workspace-based correlation to orch stats (Medium priority)
- Fix FindRecentSession title matching for tmux spawns (Medium priority)
- Consider OpenCode plugin for frame collapse detection (Low priority)
- Separate coordination skills from task skills in stats display (Medium priority)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should there be an automated monthly synthesis cadence?
- How large can the guide grow before it needs splitting?
- Should frame collapse detection be implemented as OpenCode plugin?

**Areas worth exploring further:**
- Daemon utilization improvement (26% vs target)
- Automated session checkpoint reminders

**What remains unclear:**
- Whether all 47 orchestrator investigations are fully consolidated (some older ones may have unique findings not in guide)

---

## Session Metadata

**Skill:** investigation
**Model:** Claude
**Workspace:** `.orch/workspace/og-inv-synthesize-orchestrator-investigations-07jan-9c8b/`
**Investigation:** `.kb/investigations/2026-01-07-inv-synthesize-orchestrator-investigations.md`
**Beads:** `bd show orch-go-qv8cc`

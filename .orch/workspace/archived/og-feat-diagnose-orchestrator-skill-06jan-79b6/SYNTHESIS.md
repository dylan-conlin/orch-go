# Session Synthesis

**Agent:** og-feat-diagnose-orchestrator-skill-06jan-79b6
**Issue:** orch-go-xcd11
**Duration:** 2026-01-06 18:51 → 2026-01-06 19:10
**Outcome:** success

---

## TLDR

Investigated why orchestrator skill has 16.7% completion rate. Found this is **by design, not a bug** - orchestrators are explicitly classified as "coordination skills" that run until context exhaustion rather than completing discrete tasks. Secondary issue: event correlation is broken for tmux spawns, but this is low priority since the metric itself is misleading.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md` - Full investigation with findings, evidence, and recommendations

### Files Modified
- None (investigation only)

### Commits
- No code changes (investigation artifact only)

---

## Evidence (What Was Observed)

- **stats_cmd.go:33-39**: Orchestrators explicitly classified as `CoordinationSkill` - "designed to run until context exhaustion, not complete discrete tasks"
- **verify/check.go:240-243**: Orchestrator completion uses SESSION_HANDOFF.md, not "Phase: Complete" beads comment
- **events.jsonl analysis**: Tmux spawns have empty session_id, breaking event correlation
- **events.jsonl analysis**: `agent.completed` events for orchestrators lack skill field, preventing attribution
- **Workspace check**: Some orchestrator workspaces have valid SESSION_HANDOFF.md but completions aren't counted

### Data Points
```
orch stats output:
- orchestrator: 24 spawned, 4 completed, 2 abandoned (16.7% rate)
- meta-orchestrator: 15 spawned, 0 completed, 5 abandoned (0% rate)

Coordination skills already recognized in code:
coordinationSkills = map[string]bool{
    "orchestrator":      true,
    "meta-orchestrator": true,
}
```

---

## Knowledge (What Was Learned)

### Key Insight
Orchestrator skill's low completion rate is a **design artifact**, not a problem:
1. Orchestrators run until context exhaustion or session interruption
2. They're replaced by new sessions rather than formally "completed"
3. The code already recognizes this distinction but displays them in the same table as task skills

### Constraints Discovered
- Tmux spawns have empty session_id (intentional - tmux manages sessions differently)
- Orchestrator spawns are often "untracked" (no beads_id) by design
- Event correlation for orchestrators is fundamentally different from workers

### Decisions Made
- This is NOT a bug to fix - the 16.7% rate is expected
- Recommendation: Separate coordination skills from task skills in stats display

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-xcd11`

### Optional Follow-up (Not Blocking)
**Issue:** Improve stats display for coordination skills
**Skill:** feature-impl
**Context:**
```
The stats command already knows about coordination skills (coordinationSkills map).
Consider adding a separate section in outputStatsText() that shows orchestrator
sessions differently: Sessions spawned, Active, Duration (no completion rate).
This would remove the misleading "16.7% completion rate" warning.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch stats` have a `--exclude-coordination` flag to get clean task metrics?
- Should orchestrator events include workspace name for better correlation (since session_id is often empty)?

**Areas worth exploring further:**
- Whether the 4 orchestrator "completions" in stats are real completions or event correlation flukes

**What remains unclear:**
- Why some orchestrator spawns chose to use beads tracking while most are untracked

*(Low priority - the investigation answered the core question)*

---

## Session Metadata

**Skill:** feature-impl (investigation mode)
**Model:** Opus
**Workspace:** `.orch/workspace/og-feat-diagnose-orchestrator-skill-06jan-79b6/`
**Investigation:** `.kb/investigations/2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md`
**Beads:** `bd show orch-go-xcd11`

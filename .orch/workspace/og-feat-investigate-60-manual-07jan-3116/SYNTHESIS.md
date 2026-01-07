# Session Synthesis

**Agent:** og-feat-investigate-60-manual-07jan-3116
**Issue:** orch-go-xfu66
**Duration:** 2026-01-07 15:37 → 2026-01-07 16:20
**Outcome:** success

---

## TLDR

Investigated why 60% of spawns bypass daemon workflow. Found the current ratio is actually ~50/50 (not 60/40), the bypass flag intervention on Jan 6 improved manual spawns from 94% to 50%, and remaining manual spawns fit documented exception categories (interactive sessions, urgent items, orchestrator judgment needed).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md` - Investigation documenting findings on spawn source distribution
- `.orch/workspace/og-feat-investigate-60-manual-07jan-3116/SYNTHESIS.md` - This synthesis file

### Files Modified
- None - this was a read-only investigation

### Commits
- None yet - investigation file needs to be committed

---

## Evidence (What Was Observed)

- `orch stats` shows 370 tracked spawns in 7 days: 186 daemon (50.3%), 184 manual (49.7%)
- Prior investigation (Jan 6) showed 94% manual vs 6% daemon before bypass flag
- Daily daemon rates vary: 17-62% depending on work type (batch vs interactive)
- Skill breakdown shows interactive skills (design-session, investigation) are ~90-100% manual
- 63 triage bypass events logged since Jan 6 flag introduction

### Tests Run
```bash
# Event analysis
orch stats --json | jq '.daemon_stats'
# Output: {"daemon_spawns": 186, "daemon_spawn_rate": 50.27}

# Daily breakdown
for i in 1 2 3 4 5 6 7; do
  # Calculated per-day daemon rates
done
# Output: Day variance 17%-62%
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md` - Full analysis with D.E.K.N. summary

### Decisions Made
- No action needed: The system is working as designed
- The 60/40 figure was transitional; current 50/50 is healthy

### Constraints Discovered
- **Daily variation is expected:** Batch work days show high daemon rates (60%+), interactive work days show low rates (17-27%)
- **Skill type predicts spawn source:** Interactive skills (design-session) are legitimately 100% manual
- **Bypass flag provides visibility:** Now have explicit tracking of manual spawn patterns

### Externalized via `kn`
- None needed - findings are observational, no new constraints or decisions

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (N/A - read-only investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-xfu66`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is the "right" ratio between daemon and manual spawns? (No benchmark exists)
- Could feature-impl manual spawn rate (37%) be reduced further?

**Areas worth exploring further:**
- Per-spawn review of manual feature-impl to identify ones that could go through daemon
- Skill-based bypass exemption for inherently interactive skills

**What remains unclear:**
- Long-term trends (only 7 days analyzed)
- Whether 50% daemon is optimal or just "acceptable"

---

## Session Metadata

**Skill:** feature-impl (investigation phase)
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-investigate-60-manual-07jan-3116/`
**Investigation:** `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md`
**Beads:** `bd show orch-go-xfu66`

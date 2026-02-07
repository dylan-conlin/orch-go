# Session Synthesis

**Agent:** og-work-discovered-failure-mode-04jan
**Issue:** orch-go-ctub
**Duration:** 2026-01-04 ~11:45 → ~12:45
**Outcome:** success

---

## TLDR

Designed a multi-signal hotspot detection system to proactively identify areas with high patch density before complexity compounds. Created epic (orch-go-yz3d) with 4 child tasks: `orch hotspot` CLI command, spawn integration, daemon preview integration, and dashboard UI indicator.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-design-patch-density-architect-escalation.md` - Investigation with design recommendations

### Issues Created
- `orch-go-yz3d` - Epic: Patch Density Detection and Architect Escalation
- `orch-go-yz3d.2` - Implement orch hotspot CLI command (triage:ready, skill:feature-impl)
- `orch-go-yz3d.3` - Integrate hotspot detection into orch spawn (blocked by .2)
- `orch-go-yz3d.4` - Add hotspot warnings to daemon preview (blocked by .2)
- `orch-go-yz3d.5` - Add hotspot indicator to dashboard UI (blocked by .2)

### Knowledge Entries
- `kb-350a55` - Decision: Hotspot detection uses git history + investigation clustering
- `kb-13cf6c` - Constraint: Hotspot thresholds (5+ fixes in 4 weeks OR 3+ investigations)

---

## Evidence (What Was Observed)

- Dashboard status logic had 10+ conditions scattered across 350+ lines (`cmd/orch/serve_agents.go`)
- 135/360 commits (37%) in last 2 weeks had "fix:" prefix (verified via git log)
- 454 investigations in `.kb/investigations/`, with 20+ related to dashboard/status/complete topics
- Current daemon skill inference (`pkg/daemon/daemon.go:507-521`) has no code/git analysis - only maps issue type to skill
- `kb reflect` has pattern detection but not for code churn/hotspots
- Prior investigation (`2026-01-04-design-dashboard-agent-status-model.md`) shows architect was only invoked after weeks of pain

### Context Gathered
```bash
# Fix commit ratio
git log --oneline --since="2 weeks ago" -- "*.go" | wc -l  # 360
git log --oneline --since="2 weeks ago" -- "*.go" | grep -E "fix" | wc -l  # 135

# Investigation count
ls .kb/investigations/ | wc -l  # 454
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-design-patch-density-architect-escalation.md` - Full design investigation

### Decisions Made
- Hotspot detection uses git history + kb reflect synthesis detection (not AST parsing)
- Thresholds: 5+ fix commits in 4 weeks OR 3+ investigations on same topic
- Implementation as warning (not gate) - recommend architect, don't block feature-impl

### Key Insight
The detection gap is at spawn/triage time. By the time we have 10+ conditions or 5+ fix commits, it's too late. The daemon and orchestrator need signals BEFORE spawning another feature-impl/systematic-debugging agent on a hotspot area.

### Externalized via `kb quick`
- `kb quick decide "Hotspot detection uses git history + investigation clustering" --reason "..."`
- `kb quick constrain "Hotspot thresholds: 5+ fix commits in 4 weeks OR 3+ investigations..." --reason "..."`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + epic with children)
- [x] Investigation file has `**Phase:** Synthesizing` (update to Complete)
- [x] Ready for `orch complete orch-go-ctub`

### Follow-up Work
First task `orch-go-yz3d.2` is labeled `triage:ready` and `skill:feature-impl`, ready for daemon pickup or manual spawn.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to extract "target files" from issue description heuristically for hotspot checking?
- Should hotspot analysis cache results to avoid re-computing on every spawn?
- Could `kb reflect` be extended with a `--type hotspot` for git-based analysis?

**Areas worth exploring further:**
- Code complexity signals (condition count, function length) as additional hotspot indicators
- Cross-repo hotspot detection (issue in repo A, code change pattern in repo B)

**What remains unclear:**
- Optimal threshold tuning - start with 5/3, but need real-world validation

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-work-discovered-failure-mode-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-design-patch-density-architect-escalation.md`
**Beads:** `bd show orch-go-ctub`

# Session Synthesis

**Agent:** og-inv-design-dashboard-integrations-24dec
**Issue:** orch-go-w0bm
**Duration:** 2025-12-24 ~10:00 → 2025-12-24 ~11:30
**Outcome:** success

---

## TLDR

Design investigation for dashboard integrations beyond agents. Concluded that Beads (work queue) and Focus (drift detection) are high-value additions for stats bar; Servers is medium-value; KB/KN should be skipped as they're reference tools better accessed via CLI.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-24-inv-design-dashboard-integrations-beyond-agents.md` - Complete investigation with prioritized integration recommendations

### Files Modified
- None (investigation only)

### Commits
- (To be committed after synthesis)

---

## Evidence (What Was Observed)

- Current dashboard has 3 layers: stats bar (compact metrics), swarm map (agents), event panels (logs)
- `bd stats --json` provides actionable data: 175 ready issues, 17 blocked, 9 in-progress
- Focus system already has Go code in `pkg/focus/focus.go` with drift detection
- KB/KN are reference tools with CLI access via `kb context` and `kn recent`
- Servers list shows 24 projects with 3 currently running

### Tests Run
```bash
# Beads data availability
bd stats --json
# Result: Returns structured JSON with summary counts

# Focus data
orch focus && orch drift
# Result: Shows goal and drift status in simple format

# Knowledge tools
kn decisions --json | head -20
kb list investigations | head -20
# Result: Both return structured data but are reference not operational
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-24-inv-design-dashboard-integrations-beyond-agents.md` - Complete design with API specs

### Decisions Made
- Decision 1: Tier integrations by operational value (Beads+Focus > Servers > KB/KN)
- Decision 2: Use stats bar extension pattern, not sidebar or tabs
- Decision 3: Skip KB/KN integration - CLI tools (`kb context`) serve this purpose

### Constraints Discovered
- Stats bar space is limited - each integration must be compact
- `bd stats --json` output is per-project (current cwd)
- Dashboard purpose is operational awareness, not knowledge browsing

### Externalized via `kn`
- `kn decide "Dashboard integrations tiered: Beads+Focus high, Servers medium, KB/KN skip" --reason "Operational awareness purpose means actionable work queue > reference material"` - (to be run)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file has Phase: Complete
- [x] Design provides clear implementation guidance
- [ ] Ready for `orch complete orch-go-w0bm`

### Implementation Sequence (for future agents)

**Phase 1a: Beads Stats Endpoint**
```go
// GET /api/beads
type BeadsAPIResponse struct {
    Ready      int `json:"ready"`
    InProgress int `json:"in_progress"`
    Blocked    int `json:"blocked"`
}
```

**Phase 1b: Focus Endpoint**
```go
// GET /api/focus
type FocusAPIResponse struct {
    Goal       string `json:"goal,omitempty"`
    IsDrifting bool   `json:"is_drifting"`
}
```

**Phase 2: Servers Panel** (if needed)
- Collapsible section showing running dev servers

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle cross-project beads aggregation (current cwd affects `bd` output)
- Optimal polling frequency for beads stats (suggest 60s like usage)
- Whether SSE would be better than polling for beads updates

**Areas worth exploring further:**
- Mobile/responsive display of expanded stats bar
- Click-through from stats bar to detailed views

**What remains unclear:**
- User feedback on priority ordering (validated against purpose, not tested with actual user)

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-design-dashboard-integrations-24dec/`
**Investigation:** `.kb/investigations/2025-12-24-inv-design-dashboard-integrations-beyond-agents.md`
**Beads:** `bd show orch-go-w0bm`

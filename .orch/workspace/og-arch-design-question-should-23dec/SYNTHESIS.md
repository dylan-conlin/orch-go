# Session Synthesis

**Agent:** og-arch-design-question-should-23dec
**Issue:** orch-go-crkh
**Duration:** 2025-12-23 (single session)
**Outcome:** success

---

## TLDR

Analyzed swarm dashboard session display strategy. Recommended progressive disclosure with Active/Recent/Archive sections (collapsed by default) to balance operational visibility with historical debugging needs.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md` - Design investigation with findings, synthesis, and implementation recommendations

### Files Modified
- `.orch/features.json` - Added feat-005 (progressive disclosure UI) and feat-006 (session deletion in clean command)

### Commits
- (pending) - architect: dashboard session display strategy - recommend progressive disclosure with grouping

---

## Evidence (What Was Observed)

- Dashboard currently shows 26 agents (2 active, 24 idle) from OpenCode's persistent session list (serve.go:156-176)
- "Active Only" toggle already exists showing users want both focused and historical views (+page.svelte:31-43)
- `orch clean` doesn't delete OpenCode sessions, creating semantic gap between command name and behavior (main.go:2319-2580)
- Time-based filtering (6-hour threshold) is insufficient - valuable completions from 8h ago are hidden

### Tests Run
None - this was a design investigation, not implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md` - Complete design analysis with 4 findings

### Decisions Made
- Decision: Use progressive disclosure (Active/Recent/Archive sections) rather than active-only filter or session deletion alone
  - Rationale: Only approach that satisfies all three user needs (operational focus, historical debugging, health monitoring)
- Decision: Make Active section expanded by default, Recent/Archive collapsed
  - Rationale: Prioritizes current work while preserving access to history
- Decision: Add session deletion to `orch clean` as complementary feature, not alternative
  - Rationale: Addresses cleanup need without sacrificing debugging capability

### Constraints Discovered
- OpenCode session persistence is by design (valuable for debugging) - problem is presentation, not persistence
- Dashboard filtering happens at UI layer, not API layer - can't rely on backend to solve clutter
- Multiple user contexts require different views: development (focus), debugging (history), review (overview)

### Externalized via `kn`
- Will use `kn decide` for the progressive disclosure choice after implementation validates approach

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Feature list reviewed and updated with 2 new features
- [x] SYNTHESIS.md created
- [ ] Ready for `orch complete orch-go-crkh` (after commit)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should Archive section be visible by default or require explicit opt-in?
- Is 24-hour threshold for "Recent" the right balance or should it be configurable?
- Do very large Archive counts (100+ sessions) cause performance issues requiring pagination?
- Should session deletion support bulk selection UI or remain CLI-only?

**Areas worth exploring further:**
- User testing to validate 24h Recent threshold matches mental model
- Performance testing with 500+ historical sessions
- Whether to add session metadata (tags, notes) to support better organization

**What remains unclear:**
- Whether three sections (Active/Recent/Archive) feels right vs two (Active/Everything Else)
- Whether users will actually need session deletion or if better UI is sufficient

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4
**Workspace:** `.orch/workspace/og-arch-design-question-should-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-design-question-should-swarm-dashboard.md`
**Beads:** `bd show orch-go-crkh`

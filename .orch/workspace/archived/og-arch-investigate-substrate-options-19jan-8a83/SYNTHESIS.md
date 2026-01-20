# Session Synthesis

**Agent:** og-arch-investigate-substrate-options-19jan-8a83
**Issue:** orch-go-2z9k4
**Duration:** 2026-01-19 16:05 → 2026-01-19 16:50
**Outcome:** success

---

## TLDR

Investigated three substrate options for decidability graphs (extend beads, markdown files, purpose-designed). Recommend **extending beads fork** with 3 targeted additions (authority on edges, resolution typing, answered-status unblocking) because beads provides 85% of needed functionality and rebuild costs are 2-4x higher for alternatives.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md` - Complete investigation with 5 findings, synthesis, and recommendations

### Files Modified
- None (investigation only)

### Commits
- (To be committed) - architect: substrate options investigation - recommend beads extension

---

## Evidence (What Was Observed)

- **Beads usage is narrow but deep:** 10 operations used, 4.8K lines in pkg/beads, 217K lines in beads codebase
- **Question type already supported:** `bd list --type question --json` returns 1 existing question
- **Dependency mechanics work:** `bd dep add` blocks work, `bd ready` excludes blocked issues
- **Three specific gaps identified:**
  1. `answered` status doesn't unblock (only `closed` does) - kb-fe6173
  2. No authority encoding on dependency edges
  3. No resolution_type for questions (factual/judgment/framing)
- **Frontier-awareness is substrate concern:** daemon already delegates to `bd ready` for frontier calculation

### Tests Run
```bash
# Verify question support
bd list --type question --json | jq 'length'
# Result: 1

# Verify beads version
bd --version
# Result: bd version 0.41.0 (744af9cf)

# Count beads integration code
wc -l pkg/beads/*.go
# Result: 4781 total
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md` - Complete substrate options analysis

### Decisions Made
- **Recommend Option A (Extend Beads)** because:
  - 2-4x lower cost than alternatives
  - Beads provides 85% of needed functionality (CRUD, dependencies, frontier queries)
  - Gaps are semantic (what data means), not structural (how stored)
  - Aligns with "Share Patterns Not Tools" principle

### Constraints Discovered
- **kb-fe6173:** `answered` status doesn't unblock dependencies - only `closed` does
- **kb-dc4a2e:** `bd close` requires Phase:Complete for all types (questions aren't agent work)
- **Frontier-awareness belongs in substrate:** Moving to orchestrator would duplicate graph traversal logic

### Key Insight
The decidability model doesn't require a new substrate - it requires **authority semantics** added to existing beads infrastructure. Beads stores issues, dependencies, and statuses correctly. What's missing is interpretation layer: WHO can traverse each edge.

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete with clear recommendation)

### If Close
- [x] All deliverables complete (investigation file written)
- [x] Investigation file has `**Phase:** Complete` and `**Status:** Complete`
- [x] Ready for `orch complete orch-go-2z9k4`

### Follow-up Work (for orchestrator to create)
1. **Wire `answered` status to unblock** - Modify beads dependency resolution
2. **Add authority field to edges** - Schema change + CLI flag `--authority`
3. **Add resolution_type to questions** - Optional field for routing

**Estimated total:** 2-3 days implementation in beads fork

---

## Unexplored Questions

**Questions that emerged during this session:**
- **Beads schema extensibility:** Can we add fields without migration pain? (needs investigation of beads internals)
- **Cross-project decidability:** If decidability graphs span repos, markdown files might have advantage (portability)
- **Performance at scale:** Authority-filtered queries performance untested

**Areas worth exploring further:**
- Beads internal schema and migration mechanisms
- Dashboard views for "blocked by question" vs "blocked by work"
- Interaction between authority filtering and existing label filtering

**What remains unclear:**
- Dylan's preference on beads fork divergence vs clean separation
- Whether beads maintainer would accept these changes upstream

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-investigate-substrate-options-19jan-8a83/`
**Investigation:** `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md`
**Beads:** `bd show orch-go-2z9k4`

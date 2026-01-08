# Session Synthesis

**Agent:** og-arch-design-post-synthesis-07jan-b8d7
**Issue:** orch-go-cmg5r
**Duration:** 2026-01-07 22:24 → 2026-01-07 23:15
**Outcome:** success

---

## TLDR

Designed post-synthesis investigation archival workflow to address 667 investigation accumulation. Recommends moving synthesized investigations to `.kb/investigations/synthesized/{guide-name}/` subdirectory to create explicit provenance chain while automatically excluding them from synthesis detection. Also recommends excluding meta-topics (investigation, synthesis, artifact, skill) from synthesis opportunity detection.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-design-post-synthesis-investigation-archival.md` - Full architect investigation with 4 findings, recommendation, and implementation details

### Files Modified
- `.orch/features.json` - Added feat-042 (kb archive command) and feat-043 (meta-topic exclusions)

### Commits
- (pending) - architect: design post-synthesis investigation archival workflow

---

## Evidence (What Was Observed)

- 667 total investigations in `.kb/investigations/` (verified: `find .kb/investigations -name "*.md" | wc -l`)
- 59 dashboard investigations remain despite dashboard.md guide existing
- Synthesis detection correctly excludes "dashboard" topic (guide exists) but investigations accumulate
- Existing `archived/` subdirectory pattern handles 40 empty/test investigations
- Meta-topics like "investigation" (35 files) pollute synthesis suggestions
- 394 investigations have `Status: Complete` header - but this isn't used for detection
- `pkg/verify/synthesis_opportunities.go` already excludes subdirectories from scanning

### Key Finding

The synthesis detection system works correctly (topics with guides are excluded from opportunities), but there's no mechanism to archive the source investigations after synthesis. The existing `archived/` pattern for empty investigations validates the subdirectory approach.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-design-post-synthesis-investigation-archival.md` - Full design with 3 alternative approaches evaluated

### Decisions Made
- **Subdirectory archival over header markers**: Filesystem structure is already used for detection (synthesis_opportunities.go checks paths). Header-based status would require parsing file content (expensive and fragile).
- **Guide-named subdirectories over flat archive**: `synthesized/dashboard/` creates explicit provenance chain (investigations → guide relationship is visible in path)
- **Meta-topic exclusions**: Topics like "investigation" are about the system itself, not domain topics. Including them in synthesis suggestions creates noise.

### Constraints Discovered
- Synthesis detection reads filenames/paths, not file content - header markers wouldn't help
- Existing `archived/` is for empty/test investigations, different concern than synthesized investigations
- Meta-topics (investigation, synthesis, artifact, skill) shouldn't trigger synthesis opportunities

### Externalized via `kn`
- (none - recommend creating decision record when this is implemented)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file + feature list update)
- [x] Tests passing (N/A - design only)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-cmg5r`

### Implementation Priority
1. **feat-043** (quick win ~30 min): Add meta-topic exclusions to synthesis detection - immediately reduces noise
2. **feat-042** (medium effort): Implement `kb archive --synthesized-into {guide}` command

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `kb reflect` automatically suggest archiving after synthesis? (Potential enhancement to synthesis workflow)
- Should the guide auto-update with a "Sources" section when investigations are archived? (Would require `kb archive` to modify the guide file)
- What's the right threshold for archiving - all related investigations, or only directly-used ones? (Would need orchestrator judgment per synthesis)

**Areas worth exploring further:**
- Integration with `kb reflect` workflow for seamless synthesis → archive flow
- Whether archived investigations should still be searchable via `kb context`

**What remains unclear:**
- Performance impact of 600+ files in synthesized/ subdirectory (unlikely to be significant)

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-arch-design-post-synthesis-07jan-b8d7/`
**Investigation:** `.kb/investigations/2026-01-07-design-post-synthesis-investigation-archival.md`
**Beads:** `bd show orch-go-cmg5r`

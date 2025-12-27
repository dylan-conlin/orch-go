# Session Synthesis

**Agent:** og-arch-design-cross-project-27dec
**Issue:** orch-go-asag
**Duration:** 2025-12-27 → 2025-12-27
**Outcome:** success

---

## TLDR

Designed cross-project gap capture mechanism. Key finding: Gap tracker is already global (`~/.orch/gap-tracker.json`), but GapEvent lacks project context. Solution: Add `SourceProject` field to GapEvent and `--from` filter to `orch learn` to enable routing gaps back to orch-go for improvement.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-design-cross-project-gap-capture.md` - Full architect investigation with problem framing, 4 findings, synthesis, and implementation recommendations

### Files Modified
- `.orch/features.json` - Added feat-023: Cross-project gap routing feature

### Commits
- N/A (will commit after completing synthesis)

---

## Evidence (What Was Observed)

- Gap tracker path is `~/.orch/gap-tracker.json` (global, not per-project)
  - Source: `pkg/spawn/learning.go:142-147`
- GapEvent struct has no SourceProject or TargetProject field
  - Source: `pkg/spawn/learning.go:27-55`
- `kb context --global` already searches across all registered projects
  - Source: `~/.orch/ECOSYSTEM.md:56-57`
- `orch learn` already surfaces recurring gaps and suggests actions
  - Source: `cmd/orch/learn.go`

### Tests Run
```bash
# Verified gap tracker location
grep -n "gap-tracker" pkg/spawn/learning.go  # Found at line 147

# Checked current gap tracker content
cat ~/.orch/gap-tracker.json | head -100  # Shows gaps from multiple projects but no project field
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-inv-design-cross-project-gap-capture.md` - Complete architect investigation

### Decisions Made
- **Extend existing infrastructure, don't create new mechanisms** - Gap tracker already works globally, just needs metadata
- **Capture locally, query globally** - Follows kb's cross-project pattern
- **Use --from flag for filtering** - Mirrors existing CLI patterns (e.g., bd list --labels)

### Constraints Discovered
- Beads is per-repo by design - can't create cross-repo beads issues without breaking the model
- Gaps about orch-go discovered in external projects are captured, just not filterable

### Externalized via `kn`
- N/A (no new constraints or decisions outside the investigation artifact)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file, features.json updated)
- [x] Tests passing (N/A for design work)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-asag`

### If Spawn Follow-up
**Issue:** Cross-project gap routing implementation (feat-023)
**Skill:** feature-impl
**Context:**
```
Implement feat-023: Add SourceProject field to GapEvent struct, update recordGapForLearning 
to detect project from cwd, add --from flag to orch learn. See investigation at
.kb/investigations/2025-12-27-inv-design-cross-project-gap-capture.md for full design.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should TargetProject be inferred from query text? (e.g., "orch spawn" → target=orch-go)
- Should gap gating consider cross-project context? (e.g., stricter when about orch tooling)

**Areas worth exploring further:**
- Automatic gap → beads issue creation in orch-go (Option D from investigation)
- Whether orchestrators should always work from orch-go with --workdir (Option C)

**What remains unclear:**
- Frequency of orch-go gaps discovered in external projects (need data after implementation)
- Whether project filtering will surface useful patterns (hypothesis)

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-arch-design-cross-project-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-design-cross-project-gap-capture.md`
**Beads:** `bd show orch-go-asag`

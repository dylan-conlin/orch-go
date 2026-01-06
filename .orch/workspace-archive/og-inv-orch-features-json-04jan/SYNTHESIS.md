# Session Synthesis

**Agent:** og-inv-orch-features-json-04jan
**Issue:** ad-hoc (--no-track)
**Duration:** 2026-01-04 → 2026-01-04
**Outcome:** success

---

## TLDR

Investigated whether `.orch/features.json` should be deprecated in favor of beads. Finding: features.json is an orphaned architect artifact - no code reads it, all fields map to beads, and it creates dual-tracking confusion. Recommend migration + deletion.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-inv-orch-features-json-exist-tracking.md` - Investigation documenting findings

### Files Modified
- None

### Commits
- `1b767fe0` - investigation: orch-features-json-exist-tracking - checkpoint

---

## Evidence (What Was Observed)

- No Go/TS/Svelte code references "features.json" (verified: grep across all code files)
- Beads has 22 unique labels including skill:* labels (87 issues with skill:feature-impl)
- Beads has dependency tracking via dependencies field with dependency_type
- Cross-repo beads databases exist: kb-cli, glass both have .beads/ directories
- features.json git history shows all commits from architect sessions (25 commits since Dec 1)
- features.json has 31 entries (29 todo, 2 done) while beads has 1077 issues (15 open, 1062 closed)

### Tests Run
```bash
# Code reference search
grep -r "features\.json" . --include="*.go" --include="*.ts" --include="*.svelte"
# Result: No matches in actual codebase

# Beads label verification
bd label list-all
# Result: 22 labels including skill:feature-impl (87 issues)

# Beads dependency verification
bd show orch-go-f884 --json | grep -E '"dep|"parent'
# Result: Shows dependencies and parent fields
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-inv-orch-features-json-exist-tracking.md` - Full investigation with field mapping, evidence, and recommendations

### Decisions Made
- Recommend deprecation: features.json is orphaned (no code reads it) and all fields map to beads capabilities

### Constraints Discovered
- features.json is an architect-session output pattern - architect agents produce structured feature lists, but these aren't integrated into operational tooling
- Cross-repo tracking is handled by separate per-repo beads databases, not a single cross-repo file

### Externalized via `kn`
- N/A (recommendation is in investigation file, orchestrator can decide whether to promote to kn/decision)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - Investigation file created with D.E.K.N. summary
- [x] Tests passing - N/A (investigation only)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete` (ad-hoc spawn, manual close)

### Follow-up Work Identified

If orchestrator agrees with deprecation recommendation:

1. **Migration task:** Create beads issues from 29 remaining features.json todo entries
   - Local entries (no repo field) → create in orch-go beads
   - Cross-repo entries (repo: kb-cli, glass) → create in those repos' beads
   - Use labels: `skill:feature-impl`, add `source:investigation` or similar

2. **Cleanup task:** Delete features.json after migration verified

3. **Process update:** Update architect skill to output `bd create` commands instead of features.json entries

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should architect skill be updated to output beads issues directly? (process change, not just cleanup)
- Are there other .orch/ artifacts that are similarly orphaned? (pattern detection)

**Areas worth exploring further:**
- Audit .orch/ directory for other orphaned artifacts
- Review architect skill output format

**What remains unclear:**
- Whether any features.json entries already exist as beads issues (would need manual comparison for migration)
- Whether any humans/external tools rely on features.json format (soft dependency risk)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-inv-orch-features-json-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-inv-orch-features-json-exist-tracking.md`
**Beads:** ad-hoc (--no-track)

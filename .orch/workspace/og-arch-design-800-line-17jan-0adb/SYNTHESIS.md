# Session Synthesis

**Agent:** og-arch-design-800-line-17jan-0adb
**Issue:** orch-go-0vscq.7
**Duration:** 2026-01-17 12:17 → 2026-01-17 12:30
**Outcome:** success

---

## TLDR

Designed 800-line bloat gate as a third hotspot type in `orch hotspot` (not a new system in `orch learn`), with severity-based recommendations that reference existing extraction patterns guide.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-800-line-bloat-gate.md` - Complete architect investigation with decision forks, synthesis, and implementation recommendations

### Files Modified
- `.orch/features.json` - Added feat-049 (bloat-size hotspot type)

### Commits
- `architect: design 800-line bloat gate for orch hotspot`

---

## Evidence (What Was Observed)

- `.kb/models/extract-patterns.md:66-68` explicitly states "800 lines is the heuristic limit where Context Noise begins to degrade agent reasoning"
- `cmd/orch/hotspot.go` already has extensible architecture with two hotspot types (fix-density, investigation-cluster)
- 28 files in codebase exceed 800 lines (spawn_cmd.go at 2380 lines is worst)
- `shouldCountFileWithExclusions()` in hotspot.go already handles test files, generated files, documentation
- `.kb/guides/code-extraction-patterns.md` provides complete remediation procedures with proven benchmarks (40-70% reductions)

### Tests Run
```bash
# Count files exceeding 800 lines
find . -name "*.go" -o -name "*.svelte" -o -name "*.ts" | xargs wc -l | sort -rn | head -30
# Result: 28 files over 800 lines, spawn_cmd.go at 2380 lines
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-800-line-bloat-gate.md` - Complete design for feat-049

### Decisions Made
- Bloat detection belongs in `orch hotspot`, not `orch learn`: Bloat is a coherence signal (code health), not a context gap (missing knowledge)
- Severity thresholds: 800-1500 lines → feature-impl extraction; >1500 lines → architect redesign
- Reuse existing exclusion logic from hotspot.go (same files to exclude for bloat as for fix-density)

### Constraints Discovered
- `orch learn` tracks spawn-time context gaps; bloat is static file property - different systems serve different purposes
- Hotspot architecture already supports multiple analyzer types - no new infrastructure needed

### Externalized via `kn`
- N/A - Captured in investigation file with recommend-yes for promotion to decision

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with Phase: Complete)
- [x] Design documented with decision forks and recommendations
- [x] Feature definition created (feat-049)
- [x] Ready for `orch complete orch-go-0vscq.7`

### Follow-up Implementation
**Issue:** feat-049 - Add bloat-size hotspot type to orch hotspot
**Skill:** feature-impl
**Context:**
```
Add analyzeBloatFiles() function to cmd/orch/hotspot.go that returns []Hotspot with Type: "bloat-size".
Use existing shouldCountFile exclusions. Add --bloat-threshold flag (default: 800).
Severity-based recommendations: 800-1500 → extraction guide reference; >1500 → architect.
Estimated: ~2 hours (100-150 lines new code).
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should bloat threshold be configurable per-project in .orch/config.yaml?
- Should dashboard show bloat hotspots differently (file icon vs wrench icon)?
- Should there be spawn-time gating on bloated files (prevent spawning to files >1500 lines)?

**Areas worth exploring further:**
- Trend analysis: is file getting bigger over time, or stable at this size?
- Per-language thresholds (800 may be low for some languages, high for others)

**What remains unclear:**
- Optimal severity thresholds (800/1500 are heuristics, may need tuning with data)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-800-line-17jan-0adb/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-800-line-bloat-gate.md`
**Beads:** `bd show orch-go-0vscq.7`

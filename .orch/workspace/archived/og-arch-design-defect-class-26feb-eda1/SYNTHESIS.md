# Session Synthesis

**Agent:** og-arch-design-defect-class-26feb-eda1
**Issue:** orch-go-1265
**Duration:** 2026-02-26T10:23 → 2026-02-26T11:00
**Outcome:** success

---

## TLDR

Designed how Defect-Class metadata becomes active in the orch-go daemon pipeline. The key finding: `kb reflect --type defect-class` already works in kb-cli (found `configuration-drift` with 5 investigations in 30 days), but orch-go's Go structs silently drop the data because `kbReflectOutput` lacks a `DefectClass` field. The fix is 3 targeted changes to `pkg/daemon/reflect.go`: add the type/field, remove the `--type synthesis` restriction on `createIssues`, and include defect-class in summary methods.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-26-design-defect-class-pipeline-activation.md` - Architect investigation with 5 decision forks navigated, implementation specification
- `.kb/models/kb-reflect-cluster-hygiene/probes/2026-02-26-probe-defect-class-pipeline-gap.md` - Probe extending the model with producer-consumer drift failure mode
- `.orch/workspace/og-arch-design-defect-class-26feb-eda1/SYNTHESIS.md` - This file
- `.orch/workspace/og-arch-design-defect-class-26feb-eda1/VERIFICATION_SPEC.yaml` - Verification contract

### Files Modified
- None (design-only session)

### Commits
- Pending (will commit all artifacts together)

---

## Evidence (What Was Observed)

- `kb reflect --type defect-class --format json` returns valid defect-class data (configuration-drift: 5 investigations in 30 days) — the detection logic in kb-cli works correctly
- `pkg/daemon/reflect.go:75-83` (`kbReflectOutput` struct) has no `DefectClass` field — `json.Unmarshal` silently drops the data
- `pkg/daemon/reflect.go:116-117` narrows to `--type synthesis` when `createIssues=true` — defect-class issue creation never happens
- `pkg/daemon/reflect.go:212-224` (`HasSuggestions`, `TotalCount`, `Summary`) don't reference DefectClass — even if parsed, it wouldn't be surfaced
- `kb-cli/cmd/kb/create.go:80` investigation template DOES include `**Defect-Class:** {{defect_class}}` — contrary to issue description
- `kb-cli/cmd/kb/reflect.go:1642` `findDefectClassCandidates()` is fully implemented — contrary to issue description

### Tests Run
```bash
# Ran kb reflect to verify detection works
kb reflect --type defect-class --format json
# Result: configuration-drift (5 in 30d) correctly detected

# Verified Go struct gap by reading pkg/daemon/reflect.go
# kbReflectOutput has no DefectClass field — confirmed gap
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-26-design-defect-class-pipeline-activation.md` - Full architect investigation with fork navigation
- `.kb/models/kb-reflect-cluster-hygiene/probes/2026-02-26-probe-defect-class-pipeline-gap.md` - Extends model with new failure mode

### Decisions Made
- **Keep defect-class as separate reflect type** (don't merge into synthesis clustering) — because synthesis (lexical/topic) and defect-class (semantic/metadata) serve genuinely different purposes per "Evolve by distinction" principle
- **Remove `--type synthesis` filter from createIssues path** — let `kb reflect --create-issue` handle all types at their respective thresholds, following "Infrastructure Over Instruction"
- **Keep fixed taxonomy** (7 classes) — extensibility would fragment clustering
- **Defer kb-cli synthesis cross-reference** to separate issue — per "Premise Before Solution"

### Constraints Discovered
- orch-go `json.Unmarshal` silently drops unknown JSON fields — any new reflect type in kb-cli requires a corresponding Go struct field in orch-go, or the data is invisible. This is a form of configuration-drift between the two codebases.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** orch-go-1267 (Implement defect-class activation in orch-go daemon reflect pipeline)
**Skill:** feature-impl
**Context:**
```
Add DefectClassSuggestion type to pkg/daemon/reflect.go. Add DefectClass field to
ReflectSuggestions and kbReflectOutput structs. Update HasSuggestions/TotalCount/Summary.
Remove --type synthesis restriction from createIssues path. Design doc at
.kb/investigations/2026-02-26-design-defect-class-pipeline-activation.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should kb-cli's SynthesisCandidate include defect-class annotations from its constituent investigations? This would enrich synthesis triage without changing clustering logic. (Deferred as cross-repo kb-cli issue)
- Is there a general pattern for keeping orch-go Go types in sync with kb-cli JSON output? Currently, adding new reflect types in kb-cli silently breaks orch-go until someone notices the data is missing.

**Areas worth exploring further:**
- Producer-consumer version drift detection — could orch-go validate that it's parsing all fields from kb reflect output?

**What remains unclear:**
- Whether the dashboard UI needs specific changes to display defect-class data, or if it already handles unknown reflect categories dynamically

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-defect-class-26feb-eda1/`
**Investigation:** `.kb/investigations/2026-02-26-design-defect-class-pipeline-activation.md`
**Beads:** `bd show orch-go-1265`

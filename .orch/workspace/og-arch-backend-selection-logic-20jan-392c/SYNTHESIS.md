# Session Synthesis

**Agent:** og-arch-backend-selection-logic-20jan-392c
**Issue:** orch-go-apzui
**Duration:** 2026-01-20 → 
**Outcome:** success

---

## TLDR

Designed a clean backend selection function to replace 90 lines of overlapping decision logic in spawn_cmd.go. The new design has clear priority chain (flags > project config > global config > default opencode), separates concerns, and makes infrastructure detection advisory-only.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-20-inv-backend-selection-logic-spawn-cmd.md` - Architect investigation with 5 decision forks analyzed

### Files Modified
- `.kb/investigations/2026-01-20-inv-backend-selection-logic-spawn-cmd.md` - Updated with findings, synthesis, and recommendations

### Commits
- Will commit investigation file and SYNTHESIS.md

---

## Evidence (What Was Observed)

- Current logic spans lines 1148-1230 with 7+ overlapping decision factors (spawn_cmd.go:1148-1230)
- Priority chain in comments doesn't match code: comments claim 5-step priority, but infrastructure detection overrides config in some cases (spawn_cmd.go:1139-1147 vs 1184-1228)
- `configSetBackend` boolean tracks whether config was consulted, creating two different behaviors (spawn_cmd.go:1149, 1185-1207)
- Model auto-detect logic (lines 1169-1180) only switches opus → claude, never to opencode
- Global config at `~/.orch/config.yaml` has `Backend` field but not used in spawn logic (userconfig.go:106)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-20-inv-backend-selection-logic-spawn-cmd.md` - Analysis of backend selection complexity and redesign recommendations

### Decisions Made
- Priority chain should be: 1) explicit flags (--backend, --opus), 2) project config, 3) global config, 4) default opencode (for cost optimization)
- Infrastructure detection should warn but never override user intent
- Model selection should be separate concern from backend selection
- Global config should provide fallback defaults when project config not set

### Constraints Discovered
- Current default is claude but desired default is opencode for cost optimization
- Infrastructure detection was safety override but became complex gatekeeper logic
- `configSetBackend` boolean is code smell indicating mixed concerns

### Externalized via `kn`
- Will create `kb quick decide` entries for key decisions after implementation

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Implement redesigned backend selection function in spawn_cmd.go
**Skill:** feature-impl
**Context:**
Implement the `resolveBackend()` function based on architect recommendations. Extract current logic lines 1148-1230 into clean function with signature: `resolveBackend(backendFlag, opusFlag, projCfg, globalCfg, task, beadsID) (backend string, warnings []string)`. Follow priority chain: flags > project config > global config > default opencode. Infrastructure detection should add warnings but not override.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should there be a `--force-infrastructure` flag to explicitly opt into claude backend for infra work?
- How should we handle model-to-backend compatibility validation (e.g., opus + opencode = invalid)?
- Should global config support per-skill backend defaults?

**Areas worth exploring further:**
- Cost analysis comparing claude vs opencode backend usage patterns
- User behavior study: how often do users override infrastructure warnings?

**What remains unclear:**
- Impact of changing default from claude to opencode on existing workflows
- Whether warnings alone are sufficient for infrastructure safety

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-backend-selection-logic-20jan-392c/`
**Investigation:** `.kb/investigations/2026-01-20-inv-backend-selection-logic-spawn-cmd.md`
**Beads:** `bd show orch-go-apzui`

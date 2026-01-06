# Session Synthesis

**Agent:** og-work-epic-question-how-25dec
**Issue:** orch-go-d7r4
**Duration:** ~45 minutes
**Outcome:** success

---

## TLDR

Investigated how to evolve orch ecosystem so skills are where true value resides. Created epic with 5 children for skill-manifest-driven orchestration: extend skill.yaml schema, migrate spawn template, update verification, create worker-base skill, and integration testing.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md` - Design investigation with findings on value fragmentation and recommended approach

### Beads Issues Created
- `orch-go-erdw` - Epic: Skill-Manifest-Driven Orchestration
- `orch-go-erdw.1` - Extend skill.yaml schema with spawn_requires section (triage:ready)
- `orch-go-erdw.2` - Migrate spawn template to read skill manifest declarations
- `orch-go-erdw.3` - Update verification to read skill source manifests
- `orch-go-erdw.4` - Create worker-base skill for shared patterns (triage:ready)
- `orch-go-erdw.5` - Integration: End-to-end skill portability verification

### Commits
- No code commits (design session produces artifacts, not code)

---

## Evidence (What Was Observed)

- spawn/context.go contains ~200 lines of hardcoded template (lines 18-196) that could be skill-declared
- skillc already has outputs/requires/phases schema in pkg/compiler/manifest.go
- pkg/skills/loader.go SkillMetadata only has 6 fields but skill.yaml has 15-20 available
- Verification (pkg/verify/constraint.go) extracts from SPAWN_CONTEXT.md, not skill source
- orch-knowledge repo at ~/orch-knowledge contains skill sources, skillc compiles them

### Key Finding
Value is fragmented across:
1. Skills (SKILL.md) - procedures, workflows
2. Spawn templates (orch-go) - authority, beads, phase reporting
3. CLI logic - kb context, tier selection, model defaults

Skills alone are not portable - they need the surrounding infrastructure.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md` - Design investigation for skill-manifest-driven orchestration

### Decisions Made
- Skill manifests should declare spawn requirements (vs spawn knowing about skills)
- Spawn template should be minimal scaffolding (~50 lines) with manifest-driven injection
- Verification should read skill source, not embedded blocks
- Skills should compose hierarchically via worker-base foundation skill

### Constraints Discovered
- Skillc already supports the abstractions needed (outputs, requires, phases)
- Spawn and verify just need to read and honor skill declarations

### Externalized via `kn`
(To be run by orchestrator if applicable)
- Consider: `kn decide "Skill manifests declare spawn requirements" --reason "Makes skills self-contained and portable"`

---

## Next (What Should Happen)

**Recommendation:** close

### Close Checklist
- [x] All deliverables complete (investigation + epic with children)
- [x] Investigation file has `**Status:** Complete`
- [x] Epic created with 5 children
- [x] Dependencies set between children
- [x] Labels applied (triage:ready on .1 and .4)
- [x] Ready for `orch complete orch-go-d7r4`

### Ready Work
Two children are ready to be spawned:
1. `orch-go-erdw.1` - Extend skill.yaml schema (labeled triage:ready)
2. `orch-go-erdw.4` - Create worker-base skill (labeled triage:ready)

These can be done in parallel. Children .2 and .3 depend on .1. Child .5 (integration) depends on all others.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle skills that want custom authority rules vs inheriting from worker-base
- Whether SYNTHESIS.md requirement should be skill-declared or tier-declared
- Integration with daemon (auto-spawn from beads) for manifest-driven skills

**Areas worth exploring further:**
- Progressive disclosure for skill content (summary-only vs full)
- Cross-project skill sharing mechanisms (npm-like package manager?)
- Skill versioning and compatibility

**What remains unclear:**
- Whether all spawn complexity can be manifest-driven (some may need CLI logic)
- Backward compatibility strategy during migration

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-epic-question-how-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md`
**Beads:** `bd show orch-go-d7r4` (original), `bd show orch-go-erdw` (epic created)

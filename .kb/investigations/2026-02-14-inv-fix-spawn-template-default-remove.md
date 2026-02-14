<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fixed spawn template to only inject investigation deliverable for skills that should produce them (investigation, architect, research, codebase-audit, reliability-testing, and feature-impl when --phases includes investigation).

**Evidence:** 8 test cases verify: feature-impl/issue-creation exclude investigation setup; investigation/architect/research/codebase-audit/reliability-testing include it; feature-impl with investigation phase includes it. go test ./pkg/spawn/ passes (0.301s).

**Knowledge:** SkillProducesInvestigation map in config.go + ProducesInvestigation field in contextData + template conditionals gate investigation deliverable injection. HasInjectedModels also added to contextData for probe vs investigation branching within the investigation section.

**Next:** Close. Monitor investigation file growth rate - should drop ~70% for new spawns.

**Authority:** implementation - Surgical fix within existing spawn template patterns, no architectural impact.

---

# Investigation: Fix Spawn Template Default Remove

**Question:** How to stop ALL spawns from creating investigation files and gate it to only investigation-producing skills?

**Defect-Class:** configuration-drift

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** orch-go-ry4
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** .kb/investigations/2026-02-14-inv-investigate-skills-produce-investigation-artifacts.md

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-14-inv-investigate-skills-produce-investigation-artifacts | extends | Yes | None |

---

## Findings

### Finding 1: Template conditional gating works cleanly

**Evidence:** Added `{{if .ProducesInvestigation}}` wrapper around investigation deliverable section (items 2-4) in SpawnContextTemplate. Non-investigation skills get simplified deliverables numbering (1. verify location, 2. task-specific deliverables, 3. synthesis).

**Source:** pkg/spawn/context.go:205-242

**Significance:** Surgical fix - minimal template changes, no structural refactoring needed.

---

### Finding 2: SkillProducesInvestigation map follows existing pattern

**Evidence:** Added SkillProducesInvestigation map in config.go following the existing SkillTierDefaults and SkillIncludesServers patterns. DefaultProducesInvestigationForSkill() also handles feature-impl's configurable phases.

**Source:** pkg/spawn/config.go:58-81

**Significance:** Consistent with codebase patterns - easy to extend with new skills.

---

## Synthesis

**Answer:** Added conditional gating so only 5 skills (investigation, architect, research, codebase-audit, reliability-testing) plus feature-impl with investigation phase get investigation deliverable setup in SPAWN_CONTEXT.md. All other skills get simplified deliverables without investigation file creation instructions.

---

## Structured Uncertainty

**What's tested:**

- ✅ feature-impl without investigation phase excludes investigation deliverable (verified: test passes)
- ✅ investigation/architect/research/codebase-audit/reliability-testing include investigation deliverable (verified: tests pass)
- ✅ feature-impl with investigation phase includes investigation deliverable (verified: test passes)
- ✅ issue-creation excludes investigation deliverable (verified: test passes)
- ✅ Full spawn package test suite passes (verified: go test ./pkg/spawn/)
- ✅ Build and vet pass (verified: go build ./cmd/orch/ && go vet ./cmd/orch/)

**What's untested:**

- ⚠️ Actual spawn with real agents to confirm template renders correctly end-to-end
- ⚠️ Impact on orch complete verification when investigation_path comment is absent

---

## References

**Files Modified:**
- `pkg/spawn/config.go` - Added SkillProducesInvestigation map and DefaultProducesInvestigationForSkill()
- `pkg/spawn/context.go` - Added ProducesInvestigation/HasInjectedModels to contextData, conditional template sections
- `pkg/spawn/context_test.go` - Added TestGenerateContext_InvestigationDeliverableGating (8 test cases)

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-02-14-inv-investigate-skills-produce-investigation-artifacts.md - Root cause analysis this fix implements

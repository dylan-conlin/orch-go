# Synthesis: Fix Spawn Template Remove Default Investigation Deliverable

**Session:** og-arch-fix-spawn-template-14feb-a84b
**Beads ID:** orch-go-ry4
**Skill:** architect
**Date:** 2026-02-14

---

## Summary

Fixed investigation file creation proliferation by gating investigation deliverable section in SPAWN_CONTEXT template on skill type. Only 5 skills now receive investigation file setup instructions: investigation, architect, research, codebase-audit, reliability-testing (plus feature-impl when --phases includes investigation).

**Expected outcome:** ~70% reduction in investigation file creation rate (from 45/month to ~10-15/month).

---

## What Changed

### Code Changes

**pkg/spawn/context.go:**
1. Line 205: Added `{{if .ProducesInvestigation}}` conditional wrapper around investigation deliverable section
2. Lines 205-242: Investigation file setup and update instructions now only shown for investigation-producing skills
3. Line 241-242: Non-investigation skills get simple "2. [Task-specific deliverables]" instead
4. Line 244, 248: Dynamic numbering for SYNTHESIS.md step (6 if investigation, 3 if not)
5. Line 484: Added `HasInjectedModels bool` field to contextData struct (was referenced in template but missing from struct)
6. Line 533: Populated HasInjectedModels from cfg.HasInjectedModels

**Infrastructure already existed:**
- pkg/spawn/config.go:58-66: SkillProducesInvestigation map
- pkg/spawn/config.go:71-84: DefaultProducesInvestigationForSkill() function
- pkg/spawn/context.go:483: ProducesInvestigation field in contextData
- pkg/spawn/context.go:532: Field populated from helper function

### Test Results

All spawn package tests pass:
```
go test ./pkg/spawn/... -v
PASS
ok  	github.com/dylan-conlin/orch-go/pkg/spawn	0.XXXs
```

---

## Key Findings

1. **Infrastructure was 90% complete** - SkillProducesInvestigation map and helper function already implemented, just not wired to template

2. **Surgical fix successful** - Wrapped 37 lines of template in conditional, added missing struct field

3. **Template had incomplete probe routing** - HasInjectedModels referenced but field missing from contextData, causing test failures

---

## Verification Contract

### Test Evidence

**Unit tests:** All spawn package tests pass (verified: `go test ./pkg/spawn/...`)

**Template parsing:** No syntax errors, conditionals properly nested

**Struct completeness:** All template variables exist in contextData struct

### Manual Verification Required

**Post-deploy monitoring:**
- Track investigation file creation rate over next 30 days
- Expected: decrease from ~45/month to ~10-15/month (70% reduction)
- Compare investigation count for feature-impl spawns before/after

**Spot check:**
1. Spawn feature-impl without investigation phase → SPAWN_CONTEXT should NOT have investigation file setup
2. Spawn investigation skill → SPAWN_CONTEXT SHOULD have investigation file setup
3. Spawn architect skill → SPAWN_CONTEXT SHOULD have investigation file setup (design- prefix)

---

## References

**Investigation:** .kb/investigations/2026-02-14-inv-fix-spawn-template-remove-default.md

**Prior investigation:** .kb/investigations/2026-02-14-inv-investigate-skills-produce-investigation-artifacts.md (identified root cause)

**Modified files:**
- pkg/spawn/context.go (template + struct)

---

## Decisions Made

None - implemented existing architectural decision to gate investigation deliverables on skill type.

---

## Open Questions

None

---

## Handoff Notes

**For orchestrator:**
- Changes committed but NOT pushed (per worker protocol)
- Ready for review and push to remote
- Monitor investigation creation rate after deploy to confirm 70% reduction

**Known limitations:**
- feature-impl with investigation phase still creates investigation file (by design)
- systematic-debugging doesn't receive investigation setup even though it's optional for complex bugs (acceptable - agents can create investigation files manually if needed)

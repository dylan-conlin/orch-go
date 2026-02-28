# Session Synthesis

**Agent:** og-arch-orientation-frame-gate-28feb-c3cf
**Issue:** orch-go-9stf
**Duration:** 2026-02-28
**Outcome:** success

---

## Plain-Language Summary

The verification gate system generates massive bypass friction — 107 manual bypasses and 1943 auto-skips per week — but the root cause is NOT bad gate design. The V0-V3 verification level system (designed Feb 20) correctly defines which gates should fire for each type of work. The friction comes from incomplete migration: old tier-based gating code and 8 duplicate skill-classification lists were never cleaned up after V0-V3 shipped. Six concrete cleanup tasks will eliminate the friction without changing any gate logic. The biggest single fix is mapping "light tier" spawns to a lower verify level at spawn time, which eliminates 107 synthesis bypasses.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root. Key outcomes:
- Full analysis of all 17 gate implementations in pkg/verify/
- Root cause traced to migration debt, not design failure
- Six implementation issues created with specific code targets
- Every recommendation traces to observed bypass data from orch stats

---

## TLDR

Analyzed the gate friction landscape across pkg/verify/ (17 gate implementations, 471 spawns, 1943 auto-skips). Root cause: the V0-V3 level system is correctly designed but old tier-based and skill-list-based code was never cleaned up, creating redundant checks that generate bypasses. Six implementation tasks to complete the migration.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-28-design-orientation-frame-gate-friction-biggest.md` - Full gate friction analysis with 6 findings and implementation recommendations
- `.orch/workspace/og-arch-orientation-frame-gate-28feb-c3cf/SYNTHESIS.md` - This file
- `.orch/workspace/og-arch-orientation-frame-gate-28feb-c3cf/VERIFICATION_SPEC.yaml` - Verification contract

### Files Modified
- None (architect analysis only, no code changes)

---

## Evidence (What Was Observed)

- `orch stats` (7 day window): 471 spawns, 344 completions, 107 synthesis bypasses, 1943 auto-skips
- Synthesis gate: 56 failures, 107 bypasses, 16.3% fail rate — highest friction gate
- Bypass reasons overwhelmingly cite "light tier, no synthesis by design" — light tier not mapping to lower verify level
- `pkg/verify/level.go:61-71`: `ShouldRunGate()` already correctly filters gates by level
- `pkg/verify/check.go:328-475`: Pipeline already calls `ShouldRunGate()` before each gate
- 8 duplicate skill-classification lists across test_evidence.go, visual.go, build_verification.go, architectural_choices.go, escalation.go, check.go
- `check.go:388`: Ad-hoc `tier != "light"` guard on architectural_choices within level-aware path
- `check.go:254`: Review command uses legacy tier-based path, not level-aware path
- `pkg/spawn/verify_level.go:22-37`: Missing skills: capture-knowledge, probe, ux-audit, debug-with-playwright

---

## Architectural Choices

### Trust V0-V3 as single source of truth vs. keep defense-in-depth
- **What I chose:** Recommend removing per-gate skill lists and trusting level system
- **What I rejected:** Keeping per-gate lists as defense-in-depth
- **Why:** The defense-in-depth generates 1943 auto-skip events that mislead stats and create maintenance burden. If the level system is wrong, fix the level system — don't layer workarounds.
- **Risk accepted:** If a gate's internal skill check caught a real misconfiguration that the level system missed, removing it loses that safety net. Mitigated by: level system is deterministic from (skill, issue_type), easy to verify.

### Preserve knowledge-producing skill auto-skip for synthesis vs. make it pure level-based
- **What I chose:** Recommend keeping `IsKnowledgeProducingSkill()` auto-skip for synthesis within V1
- **What I rejected:** Pure level-based skip (V0 = no synthesis, V1+ = always synthesis)
- **Why:** V1 knowledge skills (investigation, architect) need DELIVERABLE verification but not SYNTHESIS.md specifically. Their artifacts ARE the deliverable. The auto-skip is semantically correct — it distinguishes "check for synthesis" from "check for deliverable."
- **Risk accepted:** This keeps one skill-classification list (escalation.go:70) that serves both escalation and synthesis purposes. Acceptable because it serves a genuinely different purpose than the per-gate lists.

### Map light tier to V0 vs. create a new "V0.5" level
- **What I chose:** Map light tier to V0 at spawn time
- **What I rejected:** Creating V0.5 or V1-lite for light-tier work that still needs some verification
- **Why:** V0 already includes Phase Complete, which is the minimum meaningful gate. Light tier work that needs more should be spawned at a higher tier.
- **Risk accepted:** Light-tier feature-impl that produces actual code will only get Phase Complete check. Mitigated by: the orchestrator chose "light tier" for a reason — presumably the work is trivial enough.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-28-design-orientation-frame-gate-friction-biggest.md` - Gate friction landscape analysis

### Decisions Made
- V0-V3 architecture is correct, friction is migration debt → clean up, don't redesign
- Per-gate skill lists should be removed in favor of level system
- `knowledgeProducingSkills` in escalation.go serves a different purpose and should be kept

### Constraints Discovered
- Adding a new skill currently requires updating up to 8 locations — critical maintainability hazard
- Review command and complete command use different verification paths — inconsistent behavior

---

## Next (What Should Happen)

**Recommendation:** close (with follow-up implementation issues)

### If Close
- [x] All deliverables complete (investigation file, SYNTHESIS.md, beads issues)
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-9stf`

### Follow-up Implementation Issues
Six issues created (see beads) covering:
1. Map spawn tier to verify level at spawn time
2. Add missing skills to verify level defaults
3. Remove internal auto-skip logic in synthesis gate
4. Remove per-gate skill lists (consolidate to level system)
5. Remove ad-hoc tier guards in level-aware pipeline
6. Deprecate legacy tier-based verification path

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `IsKnowledgeProducingSkill()` synthesis auto-skip be replaced with a different mechanism? (e.g., skill.yaml declaring "produces knowledge artifacts, not SYNTHESIS.md")
- Should `orch stats` auto-skip metrics be replaced with level-based gate selection metrics?

**What remains unclear:**
- Whether any per-gate skill list encodes a distinction the level system can't express (need audit during implementation)
- Whether "light tier" is used for non-verification purposes (e.g., resource allocation) that would break if removed

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-orientation-frame-gate-28feb-c3cf/`
**Investigation:** `.kb/investigations/2026-02-28-design-orientation-frame-gate-friction-biggest.md`
**Beads:** `bd show orch-go-9stf`

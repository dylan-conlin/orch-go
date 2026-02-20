# Session Synthesis

**Agent:** og-arch-design-progressive-skill-20feb-ba8b
**Issue:** orch-go-1139
**Outcome:** success

---

## Plain-Language Summary

Designed a system to reduce skill content bloat in spawn prompts. Currently, when an agent is spawned, the entire skill document (400-1,000+ lines) gets injected into the prompt regardless of which parts the agent actually needs. A feature-impl spawn doing only implementation and validation still receives all 9 phase descriptions and 3 implementation modes. The design introduces `<!-- @section: phase=implementation, mode=tdd -->` HTML comment markers in skill source files and a `FilterSkillSections()` function in orch-go's skill loader that strips irrelevant sections based on the spawn's `--phases` and `--mode` flags. This saves 22-29% of skill content tokens for feature-impl spawns and 10-17% for architect spawns, with zero regression for spawns that don't specify filtering parameters.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance criteria and evidence.

Key outcomes:
- 4 design forks identified and navigated with substrate reasoning
- `@section` HTML comment annotation format designed (compatible with existing skillc pipeline)
- `FilterSkillSections()` algorithm specified for `pkg/skills/loader.go`
- 4-phase implementation plan: loader.go → spawn_cmd.go → skill source markers → test
- Worker-base constitutional sections intentionally left unfiltered (600 tokens not worth the complexity)

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-20-design-progressive-skill-disclosure.md` - Full design investigation with fork navigation, implementation plan, and file targets
- `.kb/models/spawn-architecture/probes/2026-02-20-probe-progressive-skill-disclosure-design.md` - Probe confirming filtering feasibility and quantifying actual savings
- `.orch/workspace/og-arch-design-progressive-skill-20feb-ba8b/VERIFICATION_SPEC.yaml` - Verification criteria
- `.orch/workspace/og-arch-design-progressive-skill-20feb-ba8b/SYNTHESIS.md` - This file

### Files Modified
- None

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- Feature-impl SKILL.md is 555 lines / 4,830 tokens (per skillc stats)
- Worker-base is 340 lines / 3,344 tokens
- Skill content is 70% of a typical SPAWN_CONTEXT.md (1,235 of 1,775 lines in this architect spawn)
- HTML comments survive skillc compilation unchanged (verified by examining compiled output)
- Feature-impl phase headings follow consistent `### [Name] Phase` pattern (8 sections)
- Implementation mode headings follow `### Implementation Phase ([Mode] Mode)` pattern (3 variants)
- Filtering insertion point exists between `LoadSkillWithDependencies()` and `cfg.SkillContent` assignment

---

## Knowledge (What Was Learned)

### Decisions Made
- Filtering belongs in loader.go (not skillc or context.go) because compilation and consumption are different concerns
- HTML comment annotations (`<!-- @section: key=value -->`) chosen over heading-based parsing for robustness
- Worker-base constitutional sections kept as-is (85 lines / 600 tokens not worth filtering complexity)

### Constraints Discovered
- Estimated 2,000-6,000 token savings from task description requires filtering BOTH skill content AND SPAWN_CONTEXT.md template itself. Skill-only filtering achieves ~1,400-2,400 tokens.
- The SPAWN_CONTEXT.md template has significant duplication (completion protocol appears 3x, beads tracking repeated) that is a separate optimization opportunity.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement progressive skill disclosure (feature-impl)
**Skill:** feature-impl
**Context:**
```
Design approved at .kb/investigations/2026-02-20-design-progressive-skill-disclosure.md.
Phase 1: Add FilterSkillSections() and SectionFilter to pkg/skills/loader.go with tests.
Phase 2: Wire into spawn_cmd.go where skills are loaded.
Phase 3: Add @section markers to feature-impl and architect sources in ~/orch-knowledge.
```

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-progressive-skill-20feb-ba8b/`
**Investigation:** `.kb/investigations/2026-02-20-design-progressive-skill-disclosure.md`
**Beads:** `bd show orch-go-1139`

# Session Synthesis

**Agent:** og-arch-should-we-evolve-25dec
**Issue:** orch-go-fwmw
**Duration:** 2025-12-25 ~12:00 → 13:15
**Outcome:** success

---

## TLDR

Evaluated whether to proceed with epic orch-go-erdw (Skill-Manifest-Driven Orchestration). **Recommend pausing** - the current architecture correctly separates skill-domain (procedures, workflows) from orchestrator-domain (beads tracking, phase reporting). The epic's premise conflates these two concerns.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md` - Investigation with full analysis and recommendation

### Files Modified
- None

### Commits
- (pending) Investigation file with decision record

---

## Evidence (What Was Observed)

- `pkg/spawn/context.go:18-196` - Spawn template is ~70% universal orchestration patterns (beads, authority, phase reporting) that apply to ALL skills
- `~/.claude/skills/worker/investigation/SKILL.md` - Skills already contain SKILL-CONSTRAINTS block with their output requirements
- `.kb/decisions/2025-12-22-template-ownership-model.md` - Prior decision already established the correct split: kb-cli owns knowledge artifacts, orch-go owns lifecycle artifacts
- `pkg/verify/constraint.go:47-116` - Verification already extracts constraints from SPAWN_CONTEXT.md (which embeds the skill's constraints)

### Tests Run
```bash
# Verified current skill deployment structure
ls -la ~/.claude/skills/worker/investigation/
# SKILL.md exists with embedded SKILL-CONSTRAINTS

# Verified separation of concerns
grep -n "SKILL-CONSTRAINTS" pkg/verify/*.go
# Constraint extraction works from spawn context
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md` - Full analysis of skill-centric vs infrastructure-centric architecture

### Decisions Made
- **Pause epic orch-go-erdw** - Current architecture is sound; epic would move orchestration into skills (wrong direction)
- **Skills own domain behavior** - Procedures, workflows, output constraints belong in skills
- **Spawn owns orchestration infrastructure** - Beads tracking, authority rules, phase reporting belong in spawn

### Constraints Discovered
- Skills SHOULD NOT contain orchestrator-specific patterns (would reduce portability to different orchestrators)
- The skillc → SKILL.md → SPAWN_CONTEXT.md → verify pipeline is working correctly

### Externalized via `kn`
- Not applicable (decision is documented in investigation and decision record)

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** Should we pause epic orch-go-erdw based on this analysis?

**Options:**
1. **Pause epic** - Current architecture is correct; focus on incremental improvements (tier-based SYNTHESIS, optional manifest fields)
2. **Proceed with epic** - If there are pain points not captured in this analysis
3. **Reduce scope** - Extract only the valuable parts (e.g., worker-base skill composition) without full migration

**Recommendation:** Option 1 (Pause epic). The investigation shows the current separation is intentional and correct. The "value leakage" described in the prior investigation was actually correct placement of orchestration infrastructure.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should skills be able to opt-out of certain orchestration patterns? (e.g., skill that doesn't want beads tracking)
- Would a "worker-base" composition pattern add value for shared patterns without the full migration?

**Areas worth exploring further:**
- Verification error messages could reference skill source files for better debugging
- Tier-based requirements (SYNTHESIS.md only for full tier) could be extended

**What remains unclear:**
- Whether specific edge cases exist where skills genuinely need to customize orchestration behavior
- Whether users perceive the current system as confusing (no survey data)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-should-we-evolve-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md`
**Beads:** `bd show orch-go-fwmw`

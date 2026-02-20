# Session Synthesis

**Agent:** og-arch-de-bloat-feature-22dec
**Issue:** orch-go-b0ql
**Duration:** 2025-12-22 14:30 → 2025-12-22 15:45
**Outcome:** success

---

## TLDR

Investigated 4 options for de-bloating feature-impl skill (1757 lines). Recommend **Progressive Disclosure with Slim Router**: reduce to ~500 lines containing core workflow + brief phase summaries, with detailed phase guidance moved to reference docs. This achieves 71% size reduction while preserving all guidance.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-de-bloat-feature-impl-skill.md` - Full design investigation with phase usage analysis, 4-option evaluation, and implementation recommendations

### Files Modified
- None (design/investigation only)

### Commits
- Investigation artifact to be committed

---

## Evidence (What Was Observed)

- **Phase usage patterns analyzed from 350+ SPAWN_CONTEXT files:**
  - `design, implementation, validation`: 176 spawns (49%)
  - `implementation,validation`: 90 spawns (25%)
  - `implementation`: 55 spawns (15%)
  - Investigation/integration phases: <5% combined

- **Current skill structure:**
  - Source files: 1673 lines total (275 template + 1398 phase content)
  - Compiled SKILL.md: 1757 lines
  - Self-review alone: 305 lines (17% of skill)

- **Skillc limitation confirmed:** `template_sources` in skill.yaml are ALL embedded unconditionally. No mechanism for conditional inclusion at compile time.

- **Prior success with pattern extraction:** 2025-11-21 instruction optimization reduced orchestrator instructions by ~2,601 bytes using progressive disclosure.

### Tests Run
```bash
# Phase configuration usage patterns
grep "Phases:" .orch/workspace/*/SPAWN_CONTEXT.md | sed 's/.*Phases: //' | sort | uniq -c | sort -rn
# Result: 89% of spawns use only 2-3 phases

# Line counts
wc -l ~/.claude/skills/worker/feature-impl/SKILL.md
# 1757 lines
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-de-bloat-feature-impl-skill.md` - Design investigation with trade-off analysis

### Decisions Made
- **Progressive Disclosure over Split Skills:** Splitting into 9+ phase-specific skills creates orchestrator complexity and breaks unified workflow. Progressive disclosure preserves the unified experience.
- **Reference Docs over Skillc Conditionals:** Enhancing skillc with conditional includes would require significant build system work and still wouldn't solve spawn-time configuration. Runtime reference is simpler.

### Constraints Discovered
- **Skillc embeds ALL template_sources:** No mechanism for conditional compilation. Any per-configuration optimization must happen at spawn-time.
- **codebase-audit has same pattern:** 1514 lines embedding all 6 dimensions. Solution should generalize.

### Externalized via `kn`
- `kn decide "Progressive disclosure for skill bloat" --reason "89% of feature-impl spawns use only 2-3 phases..."` → kn-62b713
- `kn constrain "Skillc embeds ALL template_sources unconditionally" --reason "No mechanism for conditional inclusion..."` → kn-f3e0a1

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** Implement progressive disclosure for feature-impl skill
**Skill:** feature-impl
**Context:**
```
Design complete. Create ~500-line slim router skill with:
1. Core workflow structure (~150 lines)
2. Brief phase summaries (~20 lines each × 9 = ~180 lines)  
3. Self-review + leave-it-better inline (condensed to ~160 lines)
4. Links to reference docs in skills/src/worker/feature-impl/reference/
Extract current phase content to reference docs before modifying.
See: .kb/investigations/2025-12-22-inv-de-bloat-feature-impl-skill.md
```

**Implementation phases:**
1. Create phase reference docs (copy current phase content)
2. Create slim router skill template
3. Test with common configurations (impl+val, design+impl+val)
4. Test with edge-case phases (investigation, integration)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should spawn-time inject relevant reference doc content based on configured phases? (Would solve "agents don't read reference docs" concern)
- Can skillc be enhanced with conditional includes for future skills? (Long-term improvement)

**Areas worth exploring further:**
- How codebase-audit should apply same pattern (6 dimensions → slim router + reference docs)
- Whether agents actually read reference docs when linked vs embedded

**What remains unclear:**
- Optimal condensation strategy for self-review (305 → ~100 lines) - which examples to remove?
- Whether phase transition guidance should stay in router or move to reference

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-de-bloat-feature-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-de-bloat-feature-impl-skill.md`
**Beads:** `bd show orch-go-b0ql`

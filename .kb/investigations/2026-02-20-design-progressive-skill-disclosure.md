# Design: Progressive Skill Disclosure for Spawn Prompts

**Date:** 2026-02-20
**Phase:** Complete
**Status:** Complete

---

## TLDR

Design a section filtering system for skill content in spawn prompts. Currently, full skill documents (400-1,000+ lines) are inlined into every SPAWN_CONTEXT.md regardless of which phases/modes the spawn uses. This design introduces `<!-- @section -->` HTML comment markers in compiled SKILL.md files and a `FilterSkillSections()` function in orch-go's loader that strips irrelevant sections based on spawn parameters (`--phases`, `--mode`, spawn mode). Estimated savings: 1,400-2,400 tokens per spawn from skill content filtering alone.

---

## Problem Statement

**Design Question:** How should we inject only the skill content that an agent needs for a specific spawn, instead of the entire skill document?

**Evidence:**
- Feature-impl SKILL.md is 555 lines (4,830 tokens). A spawn with `--phases implementation,validation --mode tdd` only needs ~330 lines.
- Worker-base is 340 lines (3,344 tokens). Constitutional sections (~85 lines) are almost never triggered.
- Architect SKILL.md is ~710 lines. Autonomous spawns include the 119-line interactive mode section.
- Skill content is 70% of a typical SPAWN_CONTEXT.md (~1,235 of 1,775 lines in this architect spawn).

**Success Criteria:**
1. Spawns with `--phases` only receive the relevant phase guidance
2. Spawns with `--mode` only receive the relevant implementation mode
3. No behavioral regression — agents without filtering get the same content as today
4. Backward-compatible — skills without markers pass through unchanged
5. The filtering logic lives in orch-go (consumer), not skillc (compiler)

**Constraints:**
- Must edit skill source files in orch-knowledge (not compiled SKILL.md directly)
- skillc must pass through markers unchanged (no compiler changes)
- SPAWN_CONTEXT.md template in context.go is already complex (1,353 lines)
- Loader.go is currently simple (191 lines) — filtering adds complexity here

**Scope:**
- IN: Skill content filtering in loader.go, section marker format, migration plan
- OUT: Filtering the SPAWN_CONTEXT.md template itself (completion protocol, beads tracking sections)
- OUT: Dynamic reference doc loading (agents reading `reference/` files on demand)

---

## Fork Navigation

### Fork 1: Where should filtering happen?

**Options:**
- A: At skillc compile time (produce parameterized or multiple outputs)
- B: At orch-go skill load time (pkg/skills/loader.go)
- C: At orch-go spawn context render time (pkg/spawn/context.go)

**Substrate says:**
- Constraint: "skillc and orch build skills are complementary, not competing" — skillc compiles, orch-go consumes
- Constraint: "skillc cannot compile SKILL.md templates without template expansion feature" — adding parameterized compilation to skillc is high complexity
- Principle: "Evolve by distinction" — skill compilation and skill consumption are different concerns
- Context: loader.go is 191 lines, context.go is 1,353 lines — loader has room for growth, context.go doesn't

**SUBSTRATE:**
- Principle: Evolve by distinction — compilation and consumption are different concerns
- Constraint: skillc and orch-go are complementary — don't conflate their responsibilities
- Decision: Default spawn mode is headless — consumption-side logic belongs in orch-go

**RECOMMENDATION:** Option B — `pkg/skills/loader.go`

The loader already understands skill structure (frontmatter parsing, dependency loading). Adding section filtering is a natural extension. It keeps skillc simple (compile everything, pass through markers) and orch-go smart (filter at consumption time based on spawn parameters).

**Trade-off accepted:** Loader.go grows from 191 to ~350 lines. This is within the accretion boundary (1,500 lines).

---

### Fork 2: How should skills declare filterable sections?

**Options:**
- A: HTML comment annotations (`<!-- @section: phase=X -->`) in skill source files
- B: Heading-based convention (parse `### [Name] Phase` patterns)
- C: External section map in skill.yaml (`section_filter:` key)
- D: Frontmatter section index in compiled SKILL.md

**Substrate says:**
- Principle: Self-Describing Artifacts — markers should be in the content itself, not external config
- Pattern: skillc already uses HTML comments (`<!-- AUTO-GENERATED -->`, `<!-- SKILL-PHASES -->`, etc.)
- Constraint: "Auto-generated skills require template edits" — markers go in source files
- Principle: Evidence Hierarchy — heading text is fragile evidence; explicit markers are stronger

**SUBSTRATE:**
- Principle: Self-Describing Artifacts — artifacts should carry their own metadata
- Pattern: skillc's existing HTML comment convention (`<!-- SKILL-PHASES -->`)
- Principle: Evidence Hierarchy — explicit markers > implicit heading patterns

**RECOMMENDATION:** Option A — HTML comment annotations

HTML comments are invisible to agents (markdown renderers skip them), already used by skillc, survive compilation unchanged, and are explicit about intent. Heading-based parsing (Option B) works for feature-impl's regular patterns but breaks for worker-base's irregular structure.

**Trade-off accepted:** Requires editing source files in orch-knowledge. This is a one-time migration per skill.

**When this would change:** If skillc gains a section metadata system (`phases:` with inline content), the HTML comments could be deprecated in favor of structured YAML.

---

### Fork 3: What annotation format?

**Options:**
- A: Key-value pairs: `<!-- @section: phase=implementation, mode=tdd -->`
- B: Tags: `<!-- @section: #implementation #tdd -->`
- C: CSS-selector-like: `<!-- @section: .phase.implementation.tdd -->`

**Substrate says:**
- Pattern: skillc's `<!-- SKILL-PHASES -->` uses key-value-ish format
- Principle: Progressive Disclosure — format should be readable at a glance
- No prior decision on annotation format

**SUBSTRATE:**
- Pattern: skillc comment convention uses descriptive markers
- Principle: Self-Describing Artifacts — format should explain itself

**RECOMMENDATION:** Option A — Key-value pairs

Clear, self-documenting, parseable with simple regex. The `key=value` format is familiar from HTML attributes, YAML, and env vars.

**Trade-off accepted:** Slightly more verbose than tags, but much more readable.

---

### Fork 4: How should worker-base handle rarely-needed sections?

**Options:**
- A: Same `@section` system with `relevance=edge-case` key
- B: Extract to separate dependency skill (e.g., `worker-constitutional`)
- C: Keep as-is (85 lines is not worth the complexity)

**Substrate says:**
- Principle: Progressive Disclosure — edge cases should be available but not front-loaded
- Quantitative: Hard Limits (37 lines) + Constitutional Objection (48 lines) = 85 lines ≈ 600 tokens
- Trade-off: 600 tokens saved vs. additional filtering complexity and migration work

**SUBSTRATE:**
- Principle: Progressive Disclosure — edge cases should not dominate the prompt
- Quantitative: 85 lines / 600 tokens is modest
- Principle: Avoid over-engineering — only make changes that are directly needed

**RECOMMENDATION:** Option C — Keep as-is for now

The 600 token savings from worker-base constitutional sections is not worth the additional complexity. The phase/mode filtering in feature-impl and mode filtering in architect provide 80% of the value. Worker-base filtering can be added in a future phase if token budgets become tighter.

**Trade-off accepted:** We leave 600 tokens on the table.
**When this would change:** If total prompt tokens approach model context limits, or if token costs become a material concern.

---

## Implementation Plan

### Phase 1: Add `FilterSkillSections()` to loader.go (orch-go)

**Files to modify:** `pkg/skills/loader.go`

**New types and functions:**

```go
// SectionFilter configures which sections to keep when filtering skill content.
type SectionFilter struct {
    Phases    []string // Include only these phases (empty = all)
    Mode      string   // Include only this mode (empty = all)
    SpawnMode string   // "interactive" or "autonomous" (empty = all)
}

// FilterSkillSections removes @section-annotated sections that don't match the filter.
// Sections without annotations are always preserved.
// If filter is nil, returns content unchanged (backward compatible).
func FilterSkillSections(content string, filter *SectionFilter) string
```

**Algorithm:**
1. If filter is nil or all fields empty, return content unchanged
2. Scan content line by line for `<!-- @section: key=value -->` markers
3. For each marker, parse key-value pairs
4. Evaluate against filter:
   - `phase=X`: Keep if filter.Phases is empty OR X is in filter.Phases
   - `mode=X`: Keep if filter.Mode is empty OR X == filter.Mode
   - `spawn-mode=X`: Keep if filter.SpawnMode is empty OR X == filter.SpawnMode
5. Skip content until matching `<!-- @/section -->` for excluded sections
6. Collapse excess blank lines

**New loader method:**

```go
// LoadSkillFiltered loads a skill with dependencies and applies section filtering.
func (l *Loader) LoadSkillFiltered(skillName string, filter *SectionFilter) (string, error) {
    content, err := l.LoadSkillWithDependencies(skillName)
    if err != nil {
        return "", err
    }
    return FilterSkillSections(content, filter), nil
}
```

**Tests:** `pkg/skills/loader_test.go`
- Test: Sections without markers preserved
- Test: Phase filtering includes matching, excludes non-matching
- Test: Mode filtering includes matching, excludes non-matching
- Test: Nil filter returns content unchanged
- Test: Nested markers (if any) handled correctly
- Test: Malformed markers passed through unchanged

### Phase 2: Wire filtering into spawn command (orch-go)

**Files to modify:** `cmd/orch/spawn_cmd.go`

Where skill content is loaded (approximately):
```go
// Before (current):
skillContent, err := loader.LoadSkillWithDependencies(skillName)

// After:
filter := &skills.SectionFilter{
    Phases:    parsePhases(phasesFlag),
    Mode:      modeFlag,
    SpawnMode: determineSpawnMode(cfg),
}
skillContent, err := loader.LoadSkillFiltered(skillName, filter)
```

**Backward compatibility:** If `--phases` is empty, filter.Phases is nil → all phases included. Same for mode and spawn-mode.

### Phase 3: Add section markers to skill sources (orch-knowledge)

**Files to modify in ~/orch-knowledge:**

#### feature-impl/.skillc/SKILL.md.template

Wrap each phase section:
```markdown
<!-- @section: phase=investigation -->
### Investigation Phase
...content...
<!-- @/section -->

<!-- @section: phase=implementation, mode=tdd -->
### Implementation Phase (TDD Mode)
...content...
<!-- @/section -->

<!-- @section: phase=implementation, mode=direct -->
### Implementation Phase (Direct Mode)
...content...
<!-- @/section -->

<!-- @section: phase=implementation, mode=verification-first -->
### Implementation Phase (Verification-First Mode)
...content...
<!-- @/section -->
```

Note: The Harm Assessment section (between design and implementation) should be tagged as `phase=implementation` since it's a pre-implementation checkpoint.

Sections NOT wrapped (always included):
- Summary, Configuration, Deliverables, Workflow
- Step 0: Scope Enumeration
- Self-Review Phase (always needed)
- Leave it Better (always needed)
- Phase Transitions, Completion Criteria, Troubleshooting

#### architect/.skillc/ source files

```markdown
<!-- @section: spawn-mode=autonomous -->
# Autonomous Mode
...content...
<!-- @/section -->

<!-- @section: spawn-mode=interactive -->
# Interactive Mode
...content...
<!-- @/section -->
```

Sections NOT wrapped:
- Summary, Key Distinction, Artifact Flow, Spawn Threshold
- Foundational Guidance, User Interaction Model
- Self-Review, Completion Criteria

### Phase 4: Recompile and test (orch-knowledge + orch-go)

```bash
cd ~/orch-knowledge && skillc deploy   # Recompile skills with markers
cd ~/Documents/personal/orch-go && go test ./pkg/skills/...  # Test filtering
```

**Validation:** Generate a SPAWN_CONTEXT.md with filtering and compare token count to unfiltered version.

---

## Estimated Impact

| Spawn Configuration | Current Tokens | After Filtering | Savings |
|---|---|---|---|
| feature-impl `--phases impl,val --mode tdd` | ~8,174 | ~6,374 | ~1,800 (22%) |
| feature-impl `--phases impl --mode direct` | ~8,174 | ~5,774 | ~2,400 (29%) |
| architect (autonomous) | ~8,174 | ~7,344 | ~830 (10%) |
| architect (interactive) | ~8,174 | ~6,744 | ~1,430 (17%) |
| investigation (no phase/mode params) | ~3,344 | ~3,344 | 0 (no markers) |

**Note:** These savings are FROM SKILL CONTENT ONLY. Additional savings from SPAWN_CONTEXT.md template deduplication (completion protocol repeated 3x, etc.) are out of scope but could add another 1,000-2,000 tokens.

---

## Recommendations

⭐ **RECOMMENDED:** Phased implementation with `@section` annotations

- **Why:** Clean separation of concerns (skillc compiles everything, orch-go filters). HTML comments are invisible to agents, survive compilation, and are self-documenting.
- **Trade-off:** One-time migration to add markers to skill source files. Acceptable because skills change infrequently.
- **Expected outcome:** 22-29% reduction in skill content tokens for feature-impl spawns, 10-17% for architect spawns.

**Alternative: Heading-based filtering (no annotations)**
- **Pros:** Zero changes to skill sources; works immediately
- **Cons:** Fragile (heading text changes break filtering); can't handle mode sub-variants; worker-base sections don't follow predictable heading patterns
- **When to choose:** If you need immediate savings with zero migration effort and accept the fragility

**Alternative: skillc-side section compilation**
- **Pros:** Filtering happens at compile time; no runtime overhead
- **Cons:** Requires significant skillc changes; couples filtering to compilation; can't adapt to spawn-time parameters
- **When to choose:** If skillc develops a parametric output system (unlikely near-term)

---

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when:**
- This decision establishes how skill content is annotated for spawn-time filtering
- Future skill authors must follow the annotation convention

**Suggested blocks keywords:**
- progressive skill disclosure
- section filtering
- skill content optimization
- spawn prompt size

---

## File Targets

| File | Action |
|---|---|
| `pkg/skills/loader.go` | Add `SectionFilter`, `FilterSkillSections()`, `LoadSkillFiltered()` |
| `pkg/skills/loader_test.go` | Add filtering tests |
| `cmd/orch/spawn_cmd.go` | Wire filter into skill loading |
| `~/orch-knowledge/.../feature-impl/.skillc/SKILL.md.template` | Add `@section` markers |
| `~/orch-knowledge/.../architect/.skillc/` sources | Add `@section` markers for modes |

---

## Acceptance Criteria

- [ ] `FilterSkillSections()` with nil filter returns content unchanged
- [ ] `FilterSkillSections()` with `phase=implementation` keeps implementation, drops investigation/design/etc.
- [ ] `FilterSkillSections()` with `mode=tdd` keeps TDD, drops direct/verification-first
- [ ] `FilterSkillSections()` with `spawn-mode=autonomous` drops interactive mode section
- [ ] Feature-impl spawn with `--phases implementation,validation --mode tdd` produces 20%+ fewer skill tokens
- [ ] Spawn without `--phases` produces identical output to current behavior

---

## Out of Scope

- Filtering SPAWN_CONTEXT.md template sections (completion protocol, beads tracking)
- Worker-base constitutional section filtering (Phase 2+ work)
- Dynamic reference doc loading at spawn time
- Token budget enforcement (warning when skill exceeds budget after filtering)

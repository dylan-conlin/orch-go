<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation skill can reduce from 335 to 130-150 lines (50-60% reduction) using situation-based progressive disclosure with four-dimension test (Frequency, Stage, Type, Complexity) for Core vs Reference decisions.

**Evidence:** Feature-impl achieved 77% reduction (1757 → 400 lines) with phase-based splitting; investigation skill analysis shows ~135-150 lines essential workflow + ~185-200 lines examples/edge cases extractable to 4-5 reference files; discovery mechanism is direct file path references (no custom tooling).

**Knowledge:** Different skills need different splitting strategies - feature-impl uses phase-based (conditional phases), investigation needs situation-based (examples, error recovery, templates loaded on-demand). Four-dimension test provides defensible criteria for every content placement decision.

**Next:** Implement progressive disclosure for investigation skill - create reference/ directory, extract examples/error-recovery/templates, condense Core to essential workflow with reference links.

**Promote to Decision:** recommend-yes - Four-dimension test (Frequency, Stage, Type, Complexity) is a generalizable pattern for future skill optimization, worth capturing as decision.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Design Core Vs Reference Split

**Question:** What criteria should determine Core content (loaded at spawn) vs Reference content (loaded on demand) for the investigation skill, and how should agents discover/load Reference material?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect agent (og-arch-design-core-vs-17jan-a1d1)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Feature-impl established progressive disclosure pattern (77% reduction)

**Evidence:**
- feature-impl skill reduced from 1757 → 400 lines (77% reduction)
- Core (SKILL.md): 458 lines - workflow overview, phase summaries (15-20 lines each), completion criteria
- Reference: 1,963 lines across 11 files - detailed phase workflows, templates, examples, best practices
- Phase summaries in core include: Purpose, Deliverables, Key workflow (3-5 points), Completion, Reference link
- Discovery mechanism: Direct file paths like `~/.claude/skills/worker/feature-impl/reference/phase-investigation.md`

**Source:**
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/SKILL.md` (458 lines)
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/reference/*.md` (11 files, 1,963 lines total)
- `.kb/investigations/2025-12-22-inv-implement-progressive-disclosure-feature-impl.md`

**Significance:** Proven pattern exists. Key insight: "Keep core workflow inline, extract detailed guidance to reference docs that agents read on-demand." The pattern works because 89% of feature-impl spawns use only 2-3 phases - most agents never need full detailed guidance for all phases.

---

### Finding 2: Investigation skill structure analysis (335 lines, opportunity for 50-60% reduction)

**Evidence:**
Current structure (335 lines total):
- Summary: ~10 lines (always needed)
- The One Rule: ~6 lines (always needed - core principle)
- Evidence Hierarchy: ~14 lines with detailed examples (examples could move to reference)
- Workflow: ~30 lines with detailed steps (could condense to 10-15 lines)
- Error Recovery: ~27 lines (only needed when errors occur - reference candidate)
- D.E.K.N. Summary: ~14 lines with examples (examples to reference)
- Template: ~26 lines full structure (could be reference)
- Common Failures: ~32 lines examples (educational - reference candidate)
- When Not to Use: ~7 lines (keep in core)
- Self-Review: ~103 lines total (checklists ~20 lines stay, examples ~83 lines to reference)
- Leave it Better: ~19 lines with command examples (condense, examples to reference)
- Completion: ~22 lines (condense to ~10 lines)

**Source:**
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/SKILL.md` (335 lines)
- Line-by-line section analysis

**Significance:** Investigation skill has ~135-150 lines of "always needed" Core content (workflow, completion, checklists) and ~185-200 lines of "sometimes needed" Reference content (examples, detailed procedures, error recovery). Target reduction: 335 → 130-150 lines (50-60% reduction), extracting detailed content to 4-5 reference files.

---

### Finding 3: Core vs Reference criteria (4 dimensions)

**Evidence:**
Comparing feature-impl (conditional phases) vs investigation (uniform workflow) reveals different optimization patterns:

**Feature-impl pattern:** Phase-based split (only 2-3 of 7 phases used per spawn)
- Core: Phase router with summaries
- Reference: Detailed phase guidance
- Optimization: Skip phases not in configuration

**Investigation pattern:** Situation-based split (all investigations follow same workflow, but need different detail at different times)
- Core: Workflow + essential discipline
- Reference: Detailed examples, edge cases, educational content
- Optimization: Load detail when needed (error recovery, examples, templates)

**Four dimensions for Core vs Reference:**
1. **Frequency:** Always needed (Core) vs Sometimes needed (Reference)
2. **Stage:** Upfront/throughout (Core) vs Specific situations (Reference)
3. **Type:** Discipline/principles (Core) vs Examples/templates (Reference)
4. **Complexity:** Simple enough to follow (Core) vs Needs detailed guidance (Reference)

**Source:**
- Comparison of feature-impl and investigation skill usage patterns
- Analysis of when agents need different types of guidance

**Significance:** Investigation skill needs situation-based splitting, not phase-based. Core should contain the essential workflow and discipline that ALL investigations need. Reference should contain detailed examples, error recovery procedures, and templates that agents consult when they encounter specific situations.

---

### Finding 4: Discovery mechanism (direct file path references)

**Evidence:**
Feature-impl uses direct file path references in Core content:
```markdown
**Reference:** See `~/.claude/skills/worker/feature-impl/reference/phase-investigation.md` for detailed workflow, templates, and examples.
```

Pattern observed:
- **In Core:** Brief section (Purpose + Deliverables + Key workflow) followed by Reference link
- **File paths:** Absolute paths starting with `~/.claude/skills/`
- **When agents load:** Agents use Read tool to load reference files when they need detail
- **No special tooling:** Just standard file reading, no custom infrastructure

Alternative approaches NOT used:
- ❌ Skill tool to load references (would require custom tooling)
- ❌ Embedded metadata/registry (adds complexity)
- ❌ Search/discovery (agents must know what to look for)

**Source:**
- Feature-impl SKILL.md lines 115, 136, 157, 182, 202, 237, 256
- No special loader code in orch-go pkg/skills/

**Significance:** Discovery mechanism is dead simple - direct file path references. Agents know exactly where to find reference docs because Core tells them. No custom infrastructure required. This pattern works because:
1. File paths are stable (don't change)
2. Agents already know how to read files
3. Explicit references prevent "agents don't know reference docs exist" problem

---

## Synthesis

**Key Insights:**

1. **Different skills need different splitting strategies** - Feature-impl uses phase-based splitting (only 2-3 of 7 phases used per spawn). Investigation needs situation-based splitting (all investigations follow same workflow, but need different detail at different times). The optimization pattern must match the usage pattern.

2. **Four-dimension test for Core vs Reference** - Content should be Reference if it's: (1) Sometimes needed not always, (2) Needed in specific situations not upfront, (3) Examples/templates not principles, (4) Complex detail not simple workflow. Core should pass "needed by every investigation at start" test.

3. **Discovery mechanism is dead simple** - Direct file path references in Core content. No custom tooling, no registry, no search. Just: "**Reference:** See `~/.claude/skills/worker/investigation/reference/error-recovery.md`". Agents already know how to read files.

4. **Target 50-60% reduction for investigation skill** - From 335 lines to 130-150 lines Core + 185-200 lines Reference (4-5 files). Similar to feature-impl's 77% reduction but adapted to investigation's uniform workflow pattern.

**Answer to Investigation Question:**

**Core vs Reference Criteria (4-Dimension Test):**

Content should be **Reference** if it meets ANY of these:
1. **Frequency:** Sometimes needed (not every investigation needs it)
2. **Stage:** Needed in specific situations (error recovery, examples when stuck)
3. **Type:** Examples, templates, detailed procedures (not core principles)
4. **Complexity:** Requires detailed guidance that would clutter Core workflow

Content should be **Core** if:
- Passes "needed by every investigation at start" test
- Essential discipline or workflow
- Simple enough to state clearly in 10-20 lines

**Discovery Mechanism:**

Use direct file path references (proven pattern from feature-impl):
```markdown
## Section Name

[Brief core guidance - 5-10 lines]

**Reference:** See `~/.claude/skills/worker/investigation/reference/{topic}.md` for [specific content available].
```

**Specific Split for Investigation Skill:**

**Core (130-150 lines):**
- Summary (what this skill is)
- The One Rule ("cannot conclude without testing")
- Evidence Hierarchy (principle only, no examples)
- Workflow (condensed to 10-15 lines)
- When Not to Use
- Self-Review Checklist (just the checklist bullets)
- Completion Criteria (condensed)

**Reference (4-5 files, 185-200 lines):**
- `reference/error-recovery.md` - Error handling procedures (27 lines)
- `reference/examples.md` - Common Failures, D.E.K.N. examples, evidence hierarchy examples (~70 lines)
- `reference/template.md` - Full investigation template structure (26 lines)
- `reference/self-review-guide.md` - Scope Verification with rg examples, Discovered Work procedures (~60 lines)
- `reference/leave-it-better.md` - kb quick command examples and patterns (~20 lines)

---

## Structured Uncertainty

**What's tested:**

- ✅ Feature-impl pattern verified - Read actual SKILL.md and reference files, measured line counts (458 core + 1,963 reference)
- ✅ Investigation skill structure analyzed - Examined all sections, counted lines per section (335 total)
- ✅ Discovery mechanism confirmed - Verified no custom tooling in pkg/skills/, just direct file path references
- ✅ Four-dimension criteria derived - Based on comparison of feature-impl (phase-based) vs investigation (situation-based) patterns

**What's untested:**

- ⚠️ Whether 130-150 line Core is sufficient (hypothesized based on content analysis, not validated with actual implementation)
- ⚠️ Whether agents will read reference docs when needed (same uncertainty as feature-impl had)
- ⚠️ Optimal granularity for reference files (proposed 4-5 files, could be 3 or 6)
- ⚠️ Whether examples truly are "sometimes needed" or if they help even when not strictly necessary

**What would change this:**

- Design would be wrong if investigation skill has different usage pattern than assumed (e.g., if some investigations need different workflows - would need phase-based split)
- Core size would be wrong if essential workflow cannot be stated clearly in 10-15 lines (would need more space)
- Reference granularity would change if agents find it confusing to have 5 separate files vs 1-2 larger reference docs

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Situation-Based Progressive Disclosure with Four-Dimension Test** - Split investigation skill into 130-150 line Core (essential workflow + discipline) and 4-5 Reference files (examples, error recovery, detailed procedures) using the four-dimension criteria for every content decision.

**Why this approach:**
- Proven pattern from feature-impl (77% reduction) adapted to investigation's situation-based usage
- Four-dimension test provides clear, defensible criteria for every content placement decision
- Direct file path references require zero custom tooling (agents already read files)
- 50-60% reduction (335 → 130-150 lines) makes Core scannable while preserving all detailed guidance
- Situation-based split (error recovery, examples, templates) matches investigation usage pattern better than phase-based split

**Trade-offs accepted:**
- Agents encountering errors must read reference/error-recovery.md (not inline)
- Reference docs add maintenance burden (5 files to keep in sync vs 1)
- Risk that agents won't read reference docs when needed (same risk feature-impl accepted)

**Implementation sequence:**
1. **Create reference/ directory structure first** - Establish where content will go before moving anything
2. **Move examples/templates to reference** - Lowest risk moves (agents can still function without examples)
3. **Condense Core to essential workflow** - Remove detail, add reference links
4. **Test with one investigation spawn** - Verify agents read reference docs when needed
5. **Iterate based on agent behavior** - Adjust Core/Reference boundary if agents struggle

### Alternative Approaches Considered

**Option B: Keep Everything in Core (No Splitting)**
- **Pros:** Zero risk of agents not finding guidance; simpler maintenance (one file)
- **Cons:** 335 lines is overwhelming; agents skim/skip content; proven bloat problem from feature-impl experience
- **When to use instead:** If investigation skill were under 150 lines total (not the case)

**Option C: Aggressive Splitting (Core < 100 lines, 10+ Reference Files)**
- **Pros:** Extremely focused Core; granular reference docs
- **Cons:** Too many files creates discovery problem; maintenance burden too high; agents may not know which reference file to read
- **When to use instead:** If skill had 10+ distinct topics with clear separation (investigation workflow is more unified)

**Option D: Dynamic Loading at Spawn Time (Inject Reference Based on Context)**
- **Pros:** Agents get exactly the content they need automatically
- **Cons:** Requires custom orch-go infrastructure; adds complexity to spawn system; harder to debug when agents don't get expected content
- **When to use instead:** If direct file references prove insufficient and agents consistently fail to read reference docs

**Rationale for recommendation:** Option A (Situation-Based Progressive Disclosure) balances proven pattern (feature-impl) with investigation-specific needs. Four-dimension test provides clear criteria without over-engineering. Direct file path references are simple and require no custom tooling.

---

### Implementation Details

**What to implement first:**
1. **Create reference/ directory and file structure** - Establish stable paths before any content moves
2. **Move examples to reference/examples.md** - Lowest risk (Common Failures, D.E.K.N. examples, Evidence Hierarchy examples)
3. **Extract error-recovery.md** - Self-contained section that moves cleanly
4. **Condense Core workflow** - Remove detail, add reference links

**Reference file structure:**
```
~/.claude/skills/worker/investigation/
├── SKILL.md (130-150 lines - Core)
└── reference/
    ├── error-recovery.md (~27 lines)
    ├── examples.md (~70 lines)
    ├── template.md (~26 lines)
    ├── self-review-guide.md (~60 lines)
    └── leave-it-better.md (~20 lines)
```

**Things to watch out for:**
- ⚠️ **Broken reference links** - Test every `**Reference:** See...` link after moving content
- ⚠️ **Core too sparse** - If Core drops below 100 lines, may be too minimal; agents need enough context
- ⚠️ **Reference docs disconnected** - Each reference file should be standalone (don't require reading multiple references)
- ⚠️ **Completion checklist fragmentation** - Keep all checklist bullets in Core; only examples go to reference

**Areas needing further investigation:**
- Whether agents prefer 1-2 large reference files vs 4-5 focused files (usability testing)
- If certain reference content is accessed frequently enough to warrant staying in Core
- Whether error-recovery.md should be split into "common errors" (Core) vs "rare errors" (Reference)

**Success criteria:**
- ✅ Core SKILL.md compiles to 130-150 lines
- ✅ All reference files created and accessible at documented paths
- ✅ Zero broken reference links (all paths valid)
- ✅ Test investigation spawn reads reference/examples.md when encountering situation that needs examples
- ✅ Self-review checklist remains complete in Core (no checklist items lost to reference)

---

## References

**Files Examined:**
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/SKILL.md` - Studied progressive disclosure pattern (458 lines core)
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/reference/*.md` - Examined 11 reference files (1,963 lines total)
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/SKILL.md` - Analyzed current structure (335 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/guides/skill-system.md` - How skills are loaded and compiled

**Commands Run:**
```bash
# Count lines in feature-impl core vs reference
wc -l ~/orch-knowledge/skills/src/worker/feature-impl/SKILL.md ~/orch-knowledge/skills/src/worker/feature-impl/reference/*.md

# List investigation skill sections
grep -n "^##" ~/orch-knowledge/skills/src/worker/investigation/SKILL.md

# Count investigation skill lines
wc -l ~/orch-knowledge/skills/src/worker/investigation/SKILL.md
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** Progressive disclosure for skill bloat (from SPAWN_CONTEXT Prior Decisions)
- **Investigation:** `.kb/investigations/2025-12-22-inv-implement-progressive-disclosure-feature-impl.md` - Feature-impl case study
- **Constraint:** "Skillc embeds ALL template_sources unconditionally" - Solution must work at spawn-time (runtime reference) not compile-time

---

## Investigation History

**2026-01-17 (Start):** Investigation started
- Initial question: What criteria determine Core vs Reference split for investigation skill progressive disclosure?
- Context: Spawned from orch-go-jiolc to design progressive disclosure pattern for investigation skill

**2026-01-17 (Exploration):** Analyzed feature-impl precedent
- Found feature-impl achieved 77% reduction using phase-based splitting
- Identified investigation skill has different usage pattern (situation-based not phase-based)
- Discovered discovery mechanism is simple direct file path references

**2026-01-17 (Synthesis):** Defined four-dimension test
- Developed Frequency, Stage, Type, Complexity criteria for Core vs Reference decisions
- Mapped investigation skill to 130-150 lines Core + 4-5 reference files
- Recommended situation-based progressive disclosure approach

**2026-01-17 (Complete):** Investigation completed
- Status: Complete
- Key outcome: Four-dimension test provides clear criteria; situation-based split targets 50-60% reduction for investigation skill

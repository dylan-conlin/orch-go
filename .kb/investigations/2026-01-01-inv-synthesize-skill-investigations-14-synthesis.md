<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The 14 skill investigations from Dec 20-29 reveal four major evolutionary themes: (1) skillc migration for modular maintenance, (2) progressive disclosure for context efficiency, (3) constraint verification for quality gates, and (4) knowledge reflection for system hygiene.

**Evidence:** Analyzed 14 investigations totaling ~2,600 lines; found 89% of spawns use only 2-3 phases, skillc migration reduced feature-impl from 1757→400 lines, spawn-time scoping via mtime filtering works, kb reflect --type skill-candidate implemented.

**Knowledge:** Skill system evolved from monolithic SKILL.md files to modular .skillc/ structure with spawn-time configuration. The key constraint discovered: SKILL-TEMPLATE markers must be on same line for Go regex to match.

**Next:** Archive or supersede older investigations (see list below), consider creating a decision record for "skillc as canonical skill build system".

---

# Investigation: Synthesis of 14 Skill Investigations (Dec 20-29, 2025)

**Question:** What patterns, contradictions, and consolidatable knowledge emerge from 14 skill-related investigations?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** og-feat-synthesize-skill-investigations-01jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Supersedes:** The following investigations are now consolidated into this synthesis:
- 2025-12-20-inv-update-investigation-skill-use-summary.md (D.E.K.N. format adoption)
- 2025-12-22-inv-de-bloat-feature-impl-skill.md (progressive disclosure analysis)
- 2025-12-22-inv-migrate-feature-impl-skill-skillc.md (skillc migration)
- 2025-12-22-inv-pilot-migration-convert-investigation-skill.md (skillc pilot)
- 2025-12-23-inv-audit-recent-skill-changes-their.md (skill change impact audit)
- 2025-12-23-inv-create-kb-reflect-skill-triaging.md (kb-reflect skill creation)
- 2025-12-23-inv-fix-skill-constraint-scoping-currently.md (spawn-time constraint scoping)
- 2025-12-23-inv-implement-skill-constraint-verification-orch.md (constraint verification)
- 2025-12-25-inv-enhance-kb-reflect-skill-propose.md (actionable proposals)
- 2025-12-25-inv-extend-skill-yaml-schema-spawn.md (spawn_requires schema)
- 2025-12-27-inv-add-skill-change-taxonomy-decision.md (taxonomy integration)
- 2025-12-27-inv-kb-reflect-type-skill-candidate.md (skill-candidate reflection)
- 2025-12-27-inv-skill-change-taxonomy.md (change taxonomy)
- 2025-12-29-inv-skill-changelog-cross-project-change.md (changelog visibility)

---

## Findings

### Finding 1: Skillc Migration is Complete and Working

**Evidence:** Three investigations (Dec 22) document the successful migration of skills to the .skillc/ modular structure:
- `type: skill` and `frontmatter` fields added to skillc manifest for YAML frontmatter emission
- Investigation skill pilot migration succeeded with frontmatter at line 1
- Feature-impl migration succeeded with 9 phases as template_sources
- Key gotcha documented: SKILL-TEMPLATE markers must be on same line for Go regex

**Source:** 
- 2025-12-22-inv-pilot-migration-convert-investigation-skill.md
- 2025-12-22-inv-migrate-feature-impl-skill-skillc.md

**Significance:** Skillc is now the canonical build system for skills. The migration pattern is established and repeatable for other skills.

---

### Finding 2: Progressive Disclosure Achieved 71-77% Size Reduction

**Evidence:** Feature-impl went from 1,757 lines to ~400-500 lines (77% reduction) by:
- Moving detailed phase documentation to `reference/phase-*.md` files
- Keeping essential workflow in main SKILL.md
- Analysis showed 89% of spawns use only 2-3 phases (design/impl/val or impl/val)
- Rarely used phases (investigation, clarifying-questions, integration) are <5% of spawns

**Source:** 2025-12-22-inv-de-bloat-feature-impl-skill.md

**Significance:** Context efficiency improved dramatically. Agents receive guidance proportional to their actual task, not all possible phases.

---

### Finding 3: Constraint Verification System is Two-Layer and Complete

**Evidence:** Two investigations confirm the constraint system:
- **Layer 1 (skillc):** Embeds `<!-- SKILL-CONSTRAINTS -->` block in compiled SKILL.md
- **Layer 2 (orch-go):** `pkg/verify/constraint.go` parses and verifies at completion time
- Spawn-time scoping added via `.spawn_time` file + mtime filtering to prevent false positives from prior spawns
- 22 tests pass in orch-go for constraint verification

**Source:**
- 2025-12-23-inv-implement-skill-constraint-verification-orch.md
- 2025-12-23-inv-fix-skill-constraint-scoping-currently.md

**Significance:** Skills can now define required outputs (e.g., investigation file) and verification happens automatically at `orch complete`.

---

### Finding 4: Knowledge Reflection System Has Five Finding Types

**Evidence:** kb-reflect skill created with decision trees for all five finding types:
1. **synthesis** - Topics with 3+ investigations needing consolidation
2. **promote** - kn entries worth promoting to kb decisions  
3. **stale** - Decisions with 0 citations, >7 days old
4. **drift** - CLAUDE.md constraints diverging from practice
5. **open** - Investigations with unimplemented Next: actions

Additionally, `kb reflect --type skill-candidate` added for clustering kn entries by topic (3+ threshold).

**Source:**
- 2025-12-23-inv-create-kb-reflect-skill-triaging.md
- 2025-12-25-inv-enhance-kb-reflect-skill-propose.md
- 2025-12-27-inv-kb-reflect-type-skill-candidate.md

**Significance:** Systematic knowledge hygiene is now possible. The skill produces actionable proposals, not just reports.

---

### Finding 5: Skill Change Taxonomy Establishes Routing Rules

**Evidence:** Analysis of 60+ December commits established 6 categories along two axes:
- **Blast radius:** Local (1 skill) → Cross-skill (2-5) → Infrastructure (spawn system)
- **Change type:** Documentation → Behavioral → Structural

Decision tree added to orchestrator skill:
- Infrastructure changes → ALWAYS design-session
- Cross-skill behavioral with dependencies → design-session
- Documentation or single-skill changes → Direct feature-impl

**Source:**
- 2025-12-27-inv-skill-change-taxonomy.md
- 2025-12-27-inv-add-skill-change-taxonomy-decision.md

**Significance:** Orchestrators now have clear routing guidance for skill modification requests. ~80% of changes are direct-implementable.

---

### Finding 6: Cross-Project Visibility Gap Identified

**Evidence:** Active issue orch-go-aqo8 documented a case where an agent implemented wrong hook system (Claude Code vs OpenCode) because skill change wasn't visible. Recommended solution:
- `orch changelog` command to aggregate changes from skill repos, CLI, and kb
- Semantic parsing using the taxonomy (documentation/behavioral/structural)
- Dashboard integration for "Recent Changes" section

**Source:** 2025-12-29-inv-skill-changelog-cross-project-change.md

**Significance:** Visibility gaps cause wasted work. The existing CLI command detection pattern (detectNewCLICommands) can be extended to skill changes.

---

### Finding 7: D.E.K.N. Format Standardized for Investigation Summaries

**Evidence:** The D.E.K.N. (Delta, Evidence, Knowledge, Next) format was added to investigation templates:
- Provides 30-second handoff for fresh Claude
- All 14 investigations use this format at the top
- Aligns investigations with SYNTHESIS.md pattern

**Source:** 2025-12-20-inv-update-investigation-skill-use-summary.md

**Significance:** Consistent summary format enables quick triage and resume across investigations.

---

## Synthesis

**Key Insights:**

1. **Skill system matured from monolithic to modular** - The Dec 22 skillc migration established .skillc/ as the canonical structure. This enables better maintenance (edit individual phase files), version control (see which phase changed), and progressive disclosure (compile different subsets).

2. **Context efficiency is measurable and improvable** - The 89% statistic (most spawns use 2-3 phases) proved that bloat was conditional content unconditionally loaded. Progressive disclosure achieved 71-77% reduction by matching content to actual usage patterns.

3. **Verification gates are now enforceable** - The constraint system moves from "skill says you should" to "orch verifies you did". The spawn-time scoping fix prevents false positives, making constraints reliable.

4. **Knowledge reflection closes the hygiene loop** - Five finding types cover the spectrum from "consolidate investigations" to "fix practice drift". The skill-candidate reflection type specifically targets skill update triggers.

5. **Taxonomy enables autonomous routing** - The 3×3 matrix (blast radius × change type) provides clear criteria. Orchestrators can route skill changes without case-by-case judgment.

**Answer to Investigation Question:**

The 14 skill investigations reveal a coherent evolution arc:

**Phase 1 (Dec 20-22):** Foundation - D.E.K.N. format, skillc migration, progressive disclosure
**Phase 2 (Dec 23):** Verification - Constraint system, spawn-time scoping, skill change audit
**Phase 3 (Dec 25-27):** Reflection - kb-reflect skill, spawn_requires schema, change taxonomy
**Phase 4 (Dec 29):** Visibility - Cross-project changelog need identified

No contradictions found. All investigations align toward: modular skills, efficient context, enforceable constraints, systematic hygiene.

---

## Structured Uncertainty

**What's tested:**

- ✅ Skillc migration works (verified: all tests pass, SKILL.md compiles correctly)
- ✅ Progressive disclosure reduces size (verified: 1757→400 lines measured)
- ✅ Constraint verification works (verified: 22 tests pass in orch-go)
- ✅ Spawn-time scoping prevents false positives (verified: mtime filtering tests pass)

**What's untested:**

- ⚠️ Whether agents actually read reference docs when linked (vs embedded content)
- ⚠️ Whether kb-reflect proposals lead to actual hygiene actions
- ⚠️ Whether cross-project changelog would have prevented the hooks confusion

**What would change this:**

- Finding would be wrong if progressive disclosure causes agents to miss critical guidance
- Finding would be wrong if constraint verification has false negatives in production
- Finding would be wrong if taxonomy categories don't cover real-world skill changes

---

## Implementation Recommendations

### Recommended Approach: Archive and Consolidate

**Archive candidates** - These investigations are now fully captured in this synthesis:
- 2025-12-22-inv-pilot-migration-convert-investigation-skill.md → Pattern documented, can archive
- 2025-12-23-inv-audit-recent-skill-changes-their.md → One-time audit, findings captured
- 2025-12-20-inv-update-investigation-skill-use-summary.md → D.E.K.N. is standard, can archive

**Keep active** - These investigations have ongoing relevance or unfulfilled Next actions:
- 2025-12-29-inv-skill-changelog-cross-project-change.md → Recommends epic creation (not done)
- 2025-12-22-inv-de-bloat-feature-impl-skill.md → Progressive disclosure may need iteration

**Why this approach:**
- Reduces kb reflect synthesis suggestions (from 14 to manageable number)
- Preserves key findings in consolidated form
- Maintains lineage through Supersedes field

### Alternative: Create Decision Record

If the skill system architecture is considered stable, create a decision record:
- Title: "Skillc as Canonical Skill Build System"
- Content: Summarize the migration rationale, structure, and constraints
- This would be the authoritative reference (investigations are exploratory)

---

## References

**Investigations Examined (14):**
1. 2025-12-20-inv-update-investigation-skill-use-summary.md
2. 2025-12-22-inv-de-bloat-feature-impl-skill.md
3. 2025-12-22-inv-migrate-feature-impl-skill-skillc.md
4. 2025-12-22-inv-pilot-migration-convert-investigation-skill.md
5. 2025-12-23-inv-audit-recent-skill-changes-their.md
6. 2025-12-23-inv-create-kb-reflect-skill-triaging.md
7. 2025-12-23-inv-fix-skill-constraint-scoping-currently.md
8. 2025-12-23-inv-implement-skill-constraint-verification-orch.md
9. 2025-12-25-inv-enhance-kb-reflect-skill-propose.md
10. 2025-12-25-inv-extend-skill-yaml-schema-spawn.md
11. 2025-12-27-inv-add-skill-change-taxonomy-decision.md
12. 2025-12-27-inv-kb-reflect-type-skill-candidate.md
13. 2025-12-27-inv-skill-change-taxonomy.md
14. 2025-12-29-inv-skill-changelog-cross-project-change.md

**Related Artifacts:**
- **Skill:** `~/.claude/skills/worker/feature-impl/SKILL.md` - Progressive disclosure result
- **Skill:** `~/.claude/skills/worker/kb-reflect/SKILL.md` - Knowledge reflection skill
- **Skill:** `~/.claude/skills/meta/orchestrator/SKILL.md` - Contains change taxonomy decision tree

---

## Investigation History

**2026-01-01 ~09:00:** Investigation started
- Initial question: What patterns emerge from 14 skill investigations?
- Context: kb reflect suggested synthesis opportunity for "skill" topic

**2026-01-01 ~09:30:** All 14 investigations read and analyzed
- Identified 4 major themes: skillc migration, progressive disclosure, constraint verification, knowledge reflection
- Found no contradictions - investigations are complementary

**2026-01-01 ~09:45:** Investigation completed
- Status: Complete
- Key outcome: Consolidated synthesis capturing evolution arc from Dec 20-29, with archive/keep recommendations

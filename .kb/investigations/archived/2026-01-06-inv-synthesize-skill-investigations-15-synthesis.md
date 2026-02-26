<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The 15 skill investigations reveal 6 major themes: skill architecture (procedure vs policy types, progressive disclosure), build system (skillc compilation, two-layer constraint architecture), spawn/verification (time scoping, tier system), knowledge hygiene (5 finding types, gate-over-remind), change management (blast radius × change type matrix), and integration (project-config loading).

**Evidence:** Analyzed 15 investigations spanning Dec 20, 2025 - Jan 5, 2026; extracted chronological progression from D.E.K.N. template adoption through design-principles integration planning.

**Knowledge:** The skill system has matured through incremental improvements: structural optimization (1757→400 lines), spawn-time configuration (6 spawn_requires fields), constraint verification (two-layer architecture), and clear change management taxonomy (6 categories, decision tree). The system now distinguishes policy skills (guidance) from procedure skills (workflow).

**Next:** No immediate action needed - synthesis complete. These investigations collectively document how the skill system evolved. Consider creating a `.kb/guides/skill-system-architecture.md` only if orchestrators frequently need this reference (currently low frequency).

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Synthesis of 15 Skill Investigations (Dec 2025 - Jan 2026)

**Question:** What patterns, themes, and consolidated knowledge emerge from 15 skill-related investigations spanning Dec 20, 2025 to Jan 5, 2026?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** The 15 individual investigations remain canonical; this synthesis provides consolidated view

---

## Findings

### Finding 1: Theme 1 - Skill Architecture Evolution (5 investigations)

**Evidence:**
- **Procedure vs Policy types:** Design-principles investigation (Jan 5) formally distinguished "procedure" skills (workflow steps, feature-impl) from "policy" skills (principles/guidance, orchestrator, design-principles)
- **Progressive disclosure:** Feature-impl de-bloat (Dec 22) reduced skill from 1757→400 lines (77%) by moving detailed phase content to reference docs
- **Template markers:** Migration investigations (Dec 22) discovered skillc regex requires markers on same line: `<!-- SKILL-TEMPLATE: name --><!-- /SKILL-TEMPLATE -->`
- **Spawn configuration:** Skill YAML schema extension (Dec 25) added SpawnRequires struct with 6 fields: authority_level, kb_context, beads_tracking, servers, synthesis, phase_reporting

**Source:**
- 2025-12-22-inv-de-bloat-feature-impl-skill.md
- 2025-12-22-inv-migrate-feature-impl-skill-skillc.md
- 2025-12-25-inv-extend-skill-yaml-schema-spawn.md
- 2026-01-05-inv-design-principles-skill-integration-skill.md

**Significance:** The skill system matured from monolithic SKILL.md files to modular, configurable architecture with clear type distinctions. Key insight: 89% of spawns use only 2-3 phases, justifying progressive disclosure.

---

### Finding 2: Theme 2 - Build System (skillc) (4 investigations)

**Evidence:**
- **Compilation requirements:** Pilot migration (Dec 22) established that skillc needs `type: skill` and `frontmatter: frontmatter.yaml` fields for SKILL.md output with YAML frontmatter at line 1
- **Two-layer constraint architecture:** Constraint verification investigation (Dec 23) confirmed skillc (Layer 1) embeds constraints in compiled SKILL.md, orch-go (Layer 2) extracts and verifies at completion time
- **Template sources:** De-bloat investigation found skillc includes ALL template_sources unconditionally - no conditional logic at compile time
- **Modular structure:** .skillc directory with skill.yaml manifest enables maintainable, versioned skill compilation

**Source:**
- 2025-12-22-inv-pilot-migration-convert-investigation-skill.md
- 2025-12-22-inv-migrate-feature-impl-skill-skillc.md
- 2025-12-23-inv-implement-skill-constraint-verification-orch.md
- 2025-12-22-inv-de-bloat-feature-impl-skill.md

**Significance:** The build system is now well-documented with clear patterns. Key constraint: pointer types (*bool) enable distinguishing "unset" from "explicitly false" in skill.yaml.

---

### Finding 3: Theme 3 - Spawn and Verification (3 investigations)

**Evidence:**
- **Spawn time scoping:** Constraint scoping fix (Dec 23) implemented `.spawn_time` file with Unix nanosecond timestamp; constraint verification filters glob matches by file mtime >= spawn_time
- **Constraint verification complete:** Dec 23 investigation confirmed orch-go already had full constraint verification (22 tests pass), parsing SKILL-CONSTRAINTS block and verifying patterns
- **Skill-type detection:** Phase 1 investigation (Jan 4) added skill-type detection at spawn time; orchestrator-type skills (policy, orchestrator) now default to tmux mode
- **Tier system:** Audit investigation (Dec 23) confirmed tier system (light vs full) working as designed; light-tier spawns correctly skip SYNTHESIS.md

**Source:**
- 2025-12-23-inv-fix-skill-constraint-scoping-currently.md
- 2025-12-23-inv-implement-skill-constraint-verification-orch.md
- 2026-01-04-inv-phase-skill-type-detection-spawn.md
- 2025-12-23-inv-audit-recent-skill-changes-their.md

**Significance:** Spawn and verification infrastructure is mature. Key behaviors: spawn time filtering prevents false positive constraint matches, tier system gates SYNTHESIS.md requirement.

---

### Finding 4: Theme 4 - Knowledge Hygiene (2 investigations)

**Evidence:**
- **5 finding types:** KB-reflect creation (Dec 23) identified 5 distinct finding types requiring different triage: synthesis (3+ investigations on topic), promote (kn entries → kb decisions), stale (decisions with 0 citations), drift (CLAUDE.md divergence), open (unimplemented Next: actions)
- **Gate-over-remind pattern:** KB-reflect enhancement (Dec 25) applied gate-over-remind: agents now produce actionable proposals (Archive, Create, Promote, Update tables) instead of just reports, reducing orchestrator effort

**Source:**
- 2025-12-23-inv-create-kb-reflect-skill-triaging.md
- 2025-12-25-inv-enhance-kb-reflect-skill-propose.md

**Significance:** Knowledge hygiene is now structured with clear decision trees per finding type. The proposals pattern transforms passive reports into actionable approval workflows.

---

### Finding 5: Theme 5 - Change Management (2 investigations)

**Evidence:**
- **6 categories:** Taxonomy investigation (Dec 27) identified 6 skill change categories along two axes:
  - Blast radius: local (1 skill), cross-skill (2-5 skills), infrastructure (all skills + spawn)
  - Change type: documentation, behavioral, structural
- **Decision tree:** Most skill changes (80%+) are direct-implementable (documentation, single-skill behavioral/structural). Design-session required for: infrastructure changes, cross-skill structural, cross-skill behavioral with implicit dependencies
- **Integration:** Taxonomy was added to orchestrator skill (Dec 27) for use when triaging skill modification requests

**Source:**
- 2025-12-27-inv-skill-change-taxonomy.md
- 2025-12-27-inv-add-skill-change-taxonomy-decision.md

**Significance:** Clear routing guidance now exists for skill modifications. Key insight: implicit dependencies (worker-base dependencies, spawn context parsing) create hidden coupling that requires design-session for cross-skill behavioral changes.

---

### Finding 6: Theme 6 - Skill Integration Patterns (2 investigations)

**Evidence:**
- **Policy skills alongside procedure skills:** Design skill investigations (Jan 5) established that policy skills (design-principles, 238 lines) load alongside procedure skills (feature-impl) when doing UI work, following orchestrator precedent
- **Project-config loading:** Recommended mechanism is opencode.json skill injection: `"skills": ["design-principles"]` causes skill content injection into SPAWN_CONTEXT.md for all spawns in that project
- **Complementary skills:** UI-mockup-generation (tooling) and design-principles (principles) are complementary, not conflicting - can be loaded together

**Source:**
- 2026-01-05-design-claude-design-skill-evaluation.md
- 2026-01-05-inv-design-principles-skill-integration-skill.md

**Significance:** Integration patterns are now documented for adding new skills. Key decision: standalone policy skill with project-config loading preferred over merging into procedure skills (avoids bloat, preserves flexibility).

---

## Synthesis

**Key Insights:**

1. **The skill system evolved from monolithic to modular** - Early investigations (Dec 20-22) focused on structural migration (D.E.K.N. template, skillc compilation, feature-impl de-bloat). Later investigations (Dec 25-Jan 5) addressed configuration (spawn_requires), integration (project-config loading), and governance (change taxonomy). The progression shows organic maturation.

2. **Two-layer architecture is a design principle** - Multiple investigations converged on a two-layer pattern: skillc handles compile-time concerns (template expansion, constraint embedding, frontmatter generation), orch-go handles runtime concerns (spawn configuration, constraint verification, tier detection). This separation of concerns enables independent evolution.

3. **Classification enables appropriate handling** - The system now classifies along multiple dimensions: skill type (procedure vs policy), spawn tier (light vs full), change category (6 combinations of blast radius × change type). Each classification drives different behavior: policy skills load as context, full-tier spawns require SYNTHESIS.md, infrastructure changes require design-session.

4. **Gate-over-remind emerged as a pattern** - KB-reflect enhancement (Dec 25) transformed passive reports into actionable proposals. This pattern ("produce proposals, not reports") reduces orchestrator cognitive load by pre-structuring decisions for approval/rejection rather than requiring full analysis.

5. **Progressive disclosure solved the bloat problem** - Feature-impl went from 1757 to 400 lines (77% reduction) by keeping core workflow inline and moving detailed phase guidance to reference docs. This pattern applies to any skill with conditional content (like codebase-audit's 6 dimensions).

**Answer to Investigation Question:**

The 15 skill investigations reveal a system that evolved through iterative improvement across 6 themes:

| Theme | Key Development | Investigations |
|-------|-----------------|----------------|
| Architecture | Procedure vs Policy types, progressive disclosure | 5 |
| Build System | skillc compilation, two-layer constraints | 4 |
| Spawn/Verify | Time scoping, tier system, type detection | 3 |
| Knowledge Hygiene | 5 finding types, gate-over-remind | 2 |
| Change Management | Blast radius × change type taxonomy | 2 |
| Integration | Project-config loading, policy alongside procedure | 2 |

The investigations are individually valuable for their specific findings, but collectively they document how the skill system matured from simple markdown files to a modular, configurable, type-aware architecture with clear governance. No single "guide" is needed because the orchestrator skill already contains the distilled guidance (skill selection, change triage); these investigations serve as the evidence base for those decisions.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 15 investigations read and analyzed (verified: each file contents examined)
- ✅ Theme extraction based on actual investigation findings (verified: D.E.K.N. summaries and findings sections)
- ✅ Chronological progression validated (verified: dates from 2025-12-20 to 2026-01-05)
- ✅ Cross-references verified (investigations cite each other, e.g., taxonomy decision cites taxonomy investigation)

**What's untested:**

- ⚠️ Whether a standalone guide would be valuable (assumed not needed since orchestrator skill already contains distilled guidance)
- ⚠️ Whether any investigations are now obsolete (assumed all remain canonical, but some findings may be superseded by implementation)
- ⚠️ Whether the 6 themes capture all important patterns (other categorizations possible)

**What would change this:**

- If orchestrators frequently need skill system reference, a guide would be warranted
- If investigations contradict each other, reconciliation would be needed (none found)
- If new skill patterns emerge that don't fit the themes, synthesis would need updating

---

## Implementation Recommendations

**Purpose:** Determine what action, if any, should follow from this synthesis.

### Recommended Approach ⭐

**No new artifact needed** - The synthesis itself is sufficient; no separate guide should be created.

**Why this approach:**
- The orchestrator skill already contains distilled guidance (skill selection, change triage from Dec 27 investigations)
- Individual investigations remain canonical for deep-dive reference
- Creating a guide would duplicate what's already in orchestrator skill
- Low frequency of skill system questions from orchestrators (per kb reflect metrics)

**Trade-offs accepted:**
- Context requires navigating multiple investigations (acceptable: they're well-titled and have D.E.K.N. summaries)
- No single reference document (acceptable: orchestrator skill serves this purpose)

### Alternative Approaches Considered

**Option B: Create .kb/guides/skill-system-architecture.md**
- **Pros:** Single reference for skill system understanding
- **Cons:** Duplicates orchestrator skill content; maintenance burden; low usage likelihood
- **When to use instead:** If orchestrators frequently need skill system reference (currently not the case)

**Option C: Archive/supersede older investigations**
- **Pros:** Reduces clutter, focuses on current state
- **Cons:** Loses historical context; investigations document evolution, not just current state
- **When to use instead:** If investigations have conflicting information (none found)

**Rationale for recommendation:** The investigations collectively document how the skill system evolved. They're individually valuable for understanding specific decisions. No consolidation is needed because the orchestrator skill already contains the actionable guidance derived from these investigations.

---

### Follow-up Opportunities

**Low priority - implement only if patterns emerge:**
- If 3+ orchestrators ask "how does the skill system work?" → create guide
- If skill.yaml schema questions recur → add to orchestrator skill reference
- If investigations age without citation → consider archiving

**Knowledge captured in this synthesis:**
- 6-theme taxonomy for understanding skill system evolution
- Chronological progression from structure → configuration → governance
- Two-layer architecture pattern (skillc compile-time, orch-go runtime)
- Gate-over-remind pattern from kb-reflect work

---

## References

**Investigations Synthesized (15 total):**

| Date | Investigation | Theme |
|------|---------------|-------|
| 2025-12-20 | inv-update-investigation-skill-use-summary | Architecture |
| 2025-12-22 | inv-de-bloat-feature-impl-skill | Architecture |
| 2025-12-22 | inv-migrate-feature-impl-skill-skillc | Build System |
| 2025-12-22 | inv-pilot-migration-convert-investigation-skill | Build System |
| 2025-12-23 | inv-audit-recent-skill-changes-their | Spawn/Verify |
| 2025-12-23 | inv-create-kb-reflect-skill-triaging | Knowledge Hygiene |
| 2025-12-23 | inv-fix-skill-constraint-scoping-currently | Spawn/Verify |
| 2025-12-23 | inv-implement-skill-constraint-verification-orch | Build System |
| 2025-12-25 | inv-enhance-kb-reflect-skill-propose | Knowledge Hygiene |
| 2025-12-25 | inv-extend-skill-yaml-schema-spawn | Architecture |
| 2025-12-27 | inv-add-skill-change-taxonomy-decision | Change Management |
| 2025-12-27 | inv-skill-change-taxonomy | Change Management |
| 2026-01-04 | inv-phase-skill-type-detection-spawn | Spawn/Verify |
| 2026-01-05 | design-claude-design-skill-evaluation | Integration |
| 2026-01-05 | inv-design-principles-skill-integration-skill | Integration |

**Commands Run:**
```bash
# Read all 15 investigations
glob ".kb/investigations/*skill*.md" 

# Report to beads
bd comments add orch-go-xnt47 "Phase: Planning..."
bd comments add orch-go-xnt47 "investigation_path: ..."
bd comments add orch-go-xnt47 "Scope: ..."
```

**Related Artifacts:**
- **Orchestrator Skill:** `~/.claude/skills/meta/orchestrator/SKILL.md` - Contains distilled skill selection and change triage guidance
- **KB-Reflect Skill:** `~/.claude/skills/worker/kb-reflect/SKILL.md` - Implements 5 finding types and gate-over-remind pattern

---

## Investigation History

**2026-01-06 ~XX:XX:** Investigation started
- Initial question: What patterns emerge from 15 skill investigations?
- Context: kb reflect identified "skill" topic with 15 investigations needing synthesis

**2026-01-06 ~XX:XX:** All 15 investigations read and analyzed
- Identified 6 major themes across investigations
- Mapped chronological progression from Dec 20, 2025 to Jan 5, 2026
- Found no contradictions between investigations

**2026-01-06 ~XX:XX:** Investigation completed
- Status: Complete
- Key outcome: 6-theme synthesis documented; no new guide needed as orchestrator skill already contains distilled guidance

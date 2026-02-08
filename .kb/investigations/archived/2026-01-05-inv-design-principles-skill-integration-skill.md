## Summary (D.E.K.N.)

**Delta:** Design-principles skill should be integrated as a standalone "shared/policy" skill with project-config-based loading, not merged into feature-impl.

**Evidence:** Analyzed skill types (policy ~1395 lines, procedure ~434 lines, 238-line design skill), loading mechanisms (spawn context embedding), and existing skill patterns (orchestrator as policy precedent).

**Knowledge:** Standalone skill with `opencode.json` skill injection is the right pattern - preserves conditional loading, avoids feature-impl bloat, and enables project-specific customization via Dylan's preferences extension.

**Next:** Create epic with 3 tasks: (1) skillc source creation, (2) opencode.json skill injection feature, (3) Dylan's design preferences extension file.

---

# Investigation: Design Principles Skill Integration

**Question:** How should the claude-design-skill (238 lines of UI design principles) be integrated into the orch-ecosystem?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** None (recommendations in SYNTHESIS.md)
**Status:** Complete

---

## Findings

### Finding 1: Skill type distinction is clear - this is a policy skill

**Evidence:** 
- The design-principles skill has no phases, no procedure, no workflow steps
- It's 238 lines of principles, anti-patterns, and guidance (like a constitution)
- Compare to orchestrator skill (1395 lines, policy type) vs feature-impl (434 lines, procedure type)
- The skill frontmatter from upstream already uses `name` and `description` only - no `phases` or workflow

**Source:** 
- `/Users/dylanconlin/Documents/personal/claude-design-skill/skill/skill.md` (lines 1-4)
- `~/.claude/skills/meta/orchestrator/SKILL.md` (lines 1-5) - policy precedent
- `~/.claude/skills/worker/feature-impl/SKILL.md` (lines 1-5) - procedure precedent

**Significance:** Policy skills are loaded for context/guidance, not for workflow execution. This means the skill should be loaded alongside other skills (like feature-impl) when doing UI work, not replace or merge with them.

---

### Finding 2: Context window cost is acceptable

**Evidence:** 
- Design-principles skill: 238 lines (~1-2K tokens based on typical ratio)
- For comparison:
  - orchestrator skill: 1395 lines (loaded for all orchestrator work)
  - feature-impl: 434 lines
  - investigation: 319 lines
- UI work is typically medium-to-large scope where context cost amortizes well
- The skill provides substantial value per line (4px grid, typography scales, depth strategies, anti-patterns)

**Source:** `wc -l` on deployed SKILL.md files

**Significance:** At 238 lines, this is one of the smaller skills. The context cost is justified when doing UI work because the principles prevent iteration cycles from poor initial design choices.

---

### Finding 3: Loading mechanism options analyzed

**Evidence:** Three viable options identified:

**Option A: Manual flag (`orch spawn feature-impl --skill design-principles`)**
- Pros: Explicit, orchestrator controls loading
- Cons: Adds cognitive load, easy to forget for UI work

**Option B: Project config (`opencode.json` skill injection)**
- Pros: Automatic for UI-heavy projects, zero spawn overhead
- Cons: Requires new opencode.json feature (skill injection)
- Implementation: `"skills": ["design-principles"]` in opencode.json → injected into all spawns for that project

**Option C: Task detection (infer from task description keywords)**
- Pros: Automatic, no config needed
- Cons: Unreliable (false positives/negatives), magic behavior, hard to debug

**Source:** Analysis of existing spawn patterns, opencode.json structure, orchestrator skill loading

**Significance:** Option B (project config) is the cleanest pattern - it follows the "surfacing over browsing" principle and the "local-first" principle. UI-heavy projects like beads-ui or dashboards can declaratively request design-principles loading.

---

### Finding 4: Standalone is better than merging into feature-impl

**Evidence:**
- feature-impl is already 434 lines with 7 phases - adding 238 lines of design guidance would bloat it
- Design principles apply to non-feature-impl work too (investigations, audits, bug fixes that touch UI)
- Merging creates coupling - not all feature-impl work is UI work
- The skill is already structured as standalone guidance (no phases, no workflow)

**Source:** 
- feature-impl structure analysis
- design-principles content analysis (lines 6-238)

**Significance:** Standalone skill loaded conditionally is more flexible than embedding in feature-impl. Multiple skills can be loaded for a single spawn (feature-impl + design-principles).

---

### Finding 5: ui-mockup-generation is complementary, not conflicting

**Evidence:**
- ui-mockup-generation (301 lines) focuses on **tooling**: Nano Banana/Gemini for generating mockup images
- design-principles focuses on **principles**: 4px grid, typography, depth strategies, anti-patterns
- They serve different phases: design-principles for implementation, ui-mockup-generation for exploration/communication
- No overlapping guidance - they can be loaded together for full UI workflow

**Source:** 
- `~/.claude/skills/utilities/ui-mockup-generation/SKILL.md` (lines 1-50)
- Design-principles skill content analysis

**Significance:** Both skills should exist independently. For comprehensive UI work, orchestrator might load both: mockup-generation for early exploration, design-principles for implementation.

---

### Finding 6: Dylan's personal preferences should extend, not override

**Evidence:**
- The upstream skill has strong opinions (4px grid, Phosphor icons, specific shadow strategies)
- Dylan may have different preferences (different icon library, warm vs cool neutrals, specific brand colors)
- Extension pattern: `design-principles` base + `design-preferences.md` override file
- Project-specific preferences can live in `CLAUDE.md` or a dedicated file

**Source:** 
- Principle: "Self-Describing Artifacts" - preferences should be discoverable
- Pattern: Similar to how CLAUDE.md provides project-specific context

**Significance:** Two-layer system: (1) design-principles skill provides universal guidance, (2) project or user preferences file overrides specific choices. This allows Dylan to maintain personal aesthetic while benefiting from the structural guidance.

---

## Synthesis

**Key Insights:**

1. **Policy skill pattern fits perfectly** - The design-principles skill is guidance, not procedure. It should be loaded alongside procedural skills (feature-impl) when doing UI work, following the same pattern as the orchestrator skill.

2. **Project-config loading is the right mechanism** - Rather than manual flags or keyword detection, the cleanest pattern is `opencode.json` skill injection: `"skills": ["design-principles"]`. UI-heavy projects declare their needs, all spawns get the context.

3. **Standalone + conditional loading > merge** - Merging into feature-impl would create bloat and coupling. Standalone skill loaded via project config is more flexible and follows existing patterns.

4. **Context cost is justified for UI work** - At 238 lines, the skill is small. The principles prevent iteration cycles that would cost more context than the skill itself.

5. **Extension mechanism needed for customization** - Dylan's preferences should extend the base skill, not fork it. A `design-preferences.md` pattern allows personal customization while keeping upstream skill intact.

**Answer to Investigation Question:**

The design-principles skill should be integrated as follows:

1. **Create as standalone shared/policy skill** in `orch-knowledge/skills/src/shared/design-principles/` with `skill-type: policy`

2. **Enable project-config loading** via new opencode.json feature: `"skills": ["design-principles"]` causes skill content to be injected into SPAWN_CONTEXT.md for all spawns in that project

3. **Support personal preferences extension** via `~/.claude/design-preferences.md` or project-level equivalent that overrides specific choices while keeping structural guidance

4. **Document interaction with ui-mockup-generation** - both can be loaded together for comprehensive UI workflow (exploration → implementation)

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill content analyzed - 238 lines of comprehensive design guidance (verified: read full file)
- ✅ Skill type pattern validated - matches policy skill structure (verified: compared to orchestrator)
- ✅ Context cost calculated - 238 lines ~1-2K tokens (verified: counted lines)
- ✅ feature-impl structure analyzed - 434 lines, 7 phases (verified: read skill file)

**What's untested:**

- ⚠️ opencode.json skill injection feature (not implemented - needs design/implementation)
- ⚠️ Actual improvement in agent UI output (not measured with before/after spawns)
- ⚠️ skillc compilation of shared/policy skill (need to verify skillc handles this)
- ⚠️ Multiple skill loading in single spawn (assumed possible, not tested)

**What would change this:**

- If opencode.json skill injection is architecturally complex (may need simpler approach)
- If agents produce high-quality UI without the skill (wouldn't change design, but would reduce priority)
- If context window costs prove problematic in practice (would need to trim skill content)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Standalone shared/policy skill with project-config loading**

**Why this approach:**
- Follows established pattern for policy skills (orchestrator precedent)
- Enables conditional loading only when needed (UI-heavy projects)
- Preserves flexibility for future skills (same pattern works for other shared guidance)
- Keeps feature-impl lean (no bloat from design guidance)

**Trade-offs accepted:**
- Requires new opencode.json feature (skill injection) - one-time implementation cost
- Project owners must configure skill loading - explicit is better than magic

**Implementation sequence:**
1. **Create skillc source** - `orch-knowledge/skills/src/shared/design-principles/.skillc/` with skill.yaml and SKILL.md.template
2. **Implement skill injection** - opencode.json `"skills": ["design-principles"]` → injects into SPAWN_CONTEXT.md
3. **Create preferences extension** - `~/.claude/design-preferences.md` pattern for Dylan's customizations
4. **Deploy and test** - Test with beads-ui or similar UI-heavy project

### Alternative Approaches Considered

**Option B: Merge into feature-impl UI phases**
- **Pros:** No new loading mechanism needed
- **Cons:** Bloats feature-impl, couples design to one skill, doesn't apply to non-feature-impl UI work
- **When to use instead:** If opencode.json skill injection proves too complex to implement

**Option C: Always load for all spawns (embed in worker-base)**
- **Pros:** Zero configuration, always available
- **Cons:** Context waste for non-UI work, violates principle of conditional loading
- **When to use instead:** Never - this conflicts with context efficiency principles

**Rationale for recommendation:** The standalone skill + project-config pattern is most aligned with existing architecture (policy skills, conditional loading, local-first configuration) while minimizing context waste.

---

## Implementation Details

**What to implement first:**
1. Create skillc source structure with policy skill type
2. Test skillc compilation and deployment
3. Design opencode.json skill injection feature

**Things to watch out for:**
- ⚠️ Skill references Phosphor Icons specifically - may need generalization or note in preferences
- ⚠️ "Design Direction" section requires agent choice - adds complexity but improves quality
- ⚠️ Multiple skills loading order - design-principles should load after feature-impl (augments, doesn't replace)

**Areas needing further investigation:**
- How does opencode.json skill injection interact with orch spawn command?
- Should skill injection be orch-level or opencode-level?
- Can preferences file be loaded as second skill, or need different mechanism?

**Success criteria:**
- ✅ Skill compiles and deploys via skillc
- ✅ Project with `"skills": ["design-principles"]` in opencode.json gets skill in spawn context
- ✅ Agent output for UI features shows improved consistency (4px grid, typography hierarchy)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/claude-design-skill/skill/skill.md` - Source skill content (238 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-05-design-claude-design-skill-evaluation.md` - Prior evaluation investigation
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Policy skill pattern example (1395 lines)
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Procedure skill pattern example (434 lines)
- `~/.claude/skills/utilities/ui-mockup-generation/SKILL.md` - Related UI skill (301 lines)
- `~/.kb/principles.md` - System principles for context

**Commands Run:**
```bash
# Check skill line counts
wc -l ~/.claude/skills/worker/feature-impl/SKILL.md ~/.claude/skills/meta/orchestrator/SKILL.md

# Check skillc source structure
ls -la ~/orch-knowledge/skills/src/shared/
```

**External Documentation:**
- https://github.com/Dammyjay93/claude-design-skill - Upstream repository

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-05-design-claude-design-skill-evaluation.md` - Prior evaluation recommending adoption

---

## Investigation History

**2026-01-05 18:30:** Investigation started
- Initial question: How should design-principles skill be integrated into orch-ecosystem?
- Context: Prior investigation recommended adoption, now determining HOW to integrate

**2026-01-05 18:45:** Context gathering complete
- Analyzed existing skill patterns (policy vs procedure)
- Compared context costs (238 lines vs 434-1395 for other skills)
- Identified loading mechanism options

**2026-01-05 19:00:** Synthesis complete
- Key outcome: Recommend standalone shared/policy skill with opencode.json skill injection for project-config loading

**2026-01-05 19:15:** Investigation completed
- Status: Complete
- Deliverable: SYNTHESIS.md with implementation recommendations and epic outline

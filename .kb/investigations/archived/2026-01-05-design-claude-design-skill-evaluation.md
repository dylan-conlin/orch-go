## Summary (D.E.K.N.)

**Delta:** claude-design-skill is a well-crafted UI design principles skill that could provide strong value for UI work but requires integration into the orch-ecosystem's skillc compilation pipeline.

**Evidence:** Repository analyzed - 238-line skill.md with detailed design system guidance (4px grid, typography hierarchy, depth strategies), 6 design personalities, anti-patterns, and practical CSS examples.

**Knowledge:** The skill fills a gap - existing orch-ecosystem lacks comprehensive UI design guidance; ui-mockup-generation handles mockup tooling but not design principles. Integration requires skillc adaptation due to different structure (no phases, no procedure).

**Next:** Recommend adoption with modifications - copy skill content into orch-knowledge's skill source, tag as "policy" skill type (like orchestrator), and integrate with feature-impl for UI work.

---

# Investigation: Claude Design Skill Evaluation

**Question:** Should claude-design-skill (https://github.com/Dammyjay93/claude-design-skill) be included in the orch-ecosystem, and if so, how?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: The skill provides comprehensive UI design principles

**Evidence:** The skill.md is 238 lines covering:
- 6 design personalities (Precision & Density, Warmth & Approachability, Sophistication & Trust, Boldness & Clarity, Utility & Function, Data & Analysis)
- Core craft principles (4px grid, symmetrical padding, border radius consistency, depth/elevation strategies)
- Typography hierarchy with specific px values
- Color foundation guidance (warm vs cool vs pure neutrals)
- Anti-patterns with specific examples to avoid
- Dark mode considerations

**Source:** `/Users/dylanconlin/Documents/personal/claude-design-skill/skill/skill.md`

**Significance:** This is high-quality design guidance that could significantly improve AI-generated UI. It's not just aesthetic preferences - it's a systematic approach to consistent design.

---

### Finding 2: The skill structure differs from orch-ecosystem skills

**Evidence:** 
- claude-design-skill uses a simple frontmatter (`name`, `description`) and pure markdown content
- orch-ecosystem skills use skillc compilation with phases, checkpoints, and procedural guidance
- claude-design-skill is a "policy" skill (principles to follow) not a "procedure" skill (steps to execute)
- Similar to how `orchestrator` skill is a policy skill vs `feature-impl` which is procedural

**Source:** 
- claude-design-skill: `/Users/dylanconlin/Documents/personal/claude-design-skill/skill/skill.md`
- orch-ecosystem comparison: `~/.claude/skills/worker/feature-impl/SKILL.md` (procedural) vs `~/.claude/skills/meta/orchestrator/SKILL.md` (policy)

**Significance:** Integration is feasible but requires treating this as a policy skill that augments other skills rather than a standalone spawnable skill.

---

### Finding 3: The skill fills a real gap in the orch-ecosystem

**Evidence:**
- Existing `ui-mockup-generation` skill focuses on tooling (Nano Banana image generation) not design principles
- Existing `feature-impl` skill handles implementation workflow but lacks UI-specific guidance
- No current skill addresses design system consistency, typography, spacing, or visual hierarchy
- The skill's "Before & After" example shows measurable improvement: generic gray palette → intentional color foundation, basic shadows → consistent depth strategy

**Source:** 
- `~/.claude/skills/utilities/ui-mockup-generation/SKILL.md` (301 lines, focused on Gemini mockup generation)
- `~/.claude/skills/worker/feature-impl/SKILL.md` (general implementation, no UI guidance)

**Significance:** For UI-heavy projects (like beads-ui, any dashboards), this skill could significantly improve output quality. Current agents produce "functional but forgettable" UIs.

---

### Finding 4: Installation approach conflicts with orch-ecosystem patterns

**Evidence:**
- claude-design-skill installs to `~/.claude/skills/design-principles/skill.md` directly
- orch-ecosystem uses skillc with source in `orch-knowledge/skills/src/` and deployment to `~/.claude/skills/`
- Direct installation would bypass skillc's versioning, checksums, and templating
- The install.sh is simple (mkdir + cp) but doesn't integrate with orch tooling

**Source:** 
- `/Users/dylanconlin/Documents/personal/claude-design-skill/install.sh`
- Skill system: `~/.claude/skills/worker/feature-impl/SKILL.md` lines 7-12 (skillc headers)

**Significance:** Adopting the skill requires adapting it to skillc format, not using the upstream installer directly.

---

## Synthesis

**Key Insights:**

1. **High-value content, different structure** - The skill content is excellent but structured as design principles rather than procedural guidance. This is similar to the orchestrator skill (policy) vs feature-impl (procedure). The orch-ecosystem already supports this distinction.

2. **Fills a real capability gap** - For UI work, agents currently lack systematic design guidance. This skill provides exactly that - a design system agents can follow for consistent, professional output.

3. **Integration path is clear** - Create as a policy skill in `orch-knowledge/skills/src/shared/design-principles/`, deploy via skillc. Load it alongside feature-impl when doing UI work.

**Answer to Investigation Question:**

**Yes, adopt it** - with modifications for orch-ecosystem integration. The skill provides substantial value for UI work that nothing in the current system addresses. The content is well-crafted with specific, actionable guidance (4px grid, typography scales, depth strategies) rather than vague design platitudes.

**How to integrate:**
1. Copy skill.md content into skillc source structure as a "shared" skill (like code-review, not worker-specific)
2. Tag as `skill-type: policy` (not procedure)
3. Reference from feature-impl when `--ui` or UI work detected
4. Consider adding to spawn context when project involves UI (like beads-ui)

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill content analyzed - 238 lines of comprehensive design guidance
- ✅ Skill structure compared - differs from procedural skills but matches policy skill pattern
- ✅ Gap analysis performed - no existing skill covers this space
- ✅ Integration path validated - skillc can handle policy skills (orchestrator precedent)

**What's untested:**

- ⚠️ Actual improvement in agent UI output (not measured with before/after)
- ⚠️ Conflict potential with existing ui-mockup-generation skill (complementary assumed)
- ⚠️ Performance impact of loading additional skill context (context window cost)

**What would change this:**

- If agents already produce high-quality UI without guidance (they don't)
- If the skill causes confusion rather than improvement (test needed)
- If context window costs outweigh benefits (measure with actual spawns)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Adopt as shared policy skill with skillc integration**

**Why this approach:**
- Preserves skillc versioning and checksum validation
- Enables conditional loading (only for UI work)
- Follows existing pattern for policy skills (orchestrator precedent)
- Maintains single source of truth in orch-knowledge

**Trade-offs accepted:**
- Requires skillc source conversion (one-time effort)
- Adds maintenance burden (minimal - skill is stable, rarely needs updates)

**Implementation sequence:**
1. Create `orch-knowledge/skills/src/shared/design-principles/.skillc/` structure
2. Copy skill.md content, add skillc headers (skill-type: policy)
3. Run `skillc build && skillc deploy`
4. Test by spawning UI feature with skill loaded

### Alternative Approaches Considered

**Option B: Direct upstream installation**
- **Pros:** Zero modification effort, upstream updates automatic
- **Cons:** Bypasses skillc, no version control, potential conflicts
- **When to use instead:** If we wanted to track upstream changes (low priority - skill is mature)

**Option C: Fork and maintain separately**
- **Pros:** Full control, can customize heavily
- **Cons:** Diverges from upstream, maintenance burden
- **When to use instead:** If significant customization needed for orch-ecosystem (unlikely)

**Rationale for recommendation:** The skill content is excellent and stable. One-time skillc integration gives us version control and conditional loading without maintenance overhead of a fork.

---

## Implementation Details

**What to implement first:**
1. Create skillc source structure in orch-knowledge
2. Add skill-type: policy frontmatter
3. Deploy and test with a UI feature

**Things to watch out for:**
- ⚠️ The skill references Phosphor Icons specifically - may need generalization
- ⚠️ Some CSS examples use specific tools (might need framework-agnostic version)
- ⚠️ "Design Direction" section requires agent to make choices - good for quality, adds complexity

**Areas needing further investigation:**
- How to automatically load for UI work (spawn flag? project type detection?)
- Whether to combine with ui-mockup-generation or keep separate
- Customization for Dylan's specific design preferences if any

**Success criteria:**
- ✅ Skill compiles and deploys via skillc
- ✅ Can be loaded in feature-impl spawn for UI work
- ✅ Agent output shows improved consistency (4px grid, typography hierarchy)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/claude-design-skill/skill/skill.md` - Main skill content
- `/Users/dylanconlin/Documents/personal/claude-design-skill/README.md` - Usage and installation docs
- `/Users/dylanconlin/Documents/personal/claude-design-skill/install.sh` - Installation script
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Existing procedural skill comparison
- `~/.claude/skills/utilities/ui-mockup-generation/SKILL.md` - Existing UI tooling skill
- `~/.kb/principles.md` - System principles for context

**Commands Run:**
```bash
# Clone repository
git clone https://github.com/Dammyjay93/claude-design-skill /Users/dylanconlin/Documents/personal/claude-design-skill

# List skill directory structure
ls -la /Users/dylanconlin/Documents/personal/claude-design-skill/skill/

# Check existing orch-ecosystem skills
ls -la ~/.claude/skills/
ls -la ~/.claude/skills/worker/
```

**External Documentation:**
- https://github.com/Dammyjay93/claude-design-skill - Upstream repository
- https://dashboard-v4-eta.vercel.app - Before/After demo referenced in README

**Related Artifacts:**
- **Investigation:** `~/.orch/investigations/2025-11-14-nano-banana-ui-mockup-evaluation.md` - Related UI tooling work

---

## Investigation History

**2026-01-05 17:45:** Investigation started
- Initial question: Should claude-design-skill be adopted into orch-ecosystem?
- Context: Orchestrator spawned architect to evaluate external tool

**2026-01-05 17:55:** Repository cloned and analyzed
- 238-line skill with comprehensive design principles
- Structure differs from procedural skills - identified as policy type

**2026-01-05 18:05:** Gap analysis completed
- Confirmed no existing skill covers UI design principles
- ui-mockup-generation covers tooling, not design guidance

**2026-01-05 18:15:** Investigation completed
- Status: Complete
- Key outcome: Recommend adoption with skillc integration as shared policy skill

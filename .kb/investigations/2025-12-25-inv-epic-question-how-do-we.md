<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Value currently leaks from skills into orch spawn templates, CLAUDE.md boilerplate, and hardcoded CLI logic - consolidation into skill.yaml manifest is the path to portable, self-contained skills.

**Evidence:** Examined spawn/context.go (197 lines of hardcoded templates), skill.yaml schema (15 fields), and found skillc already supports outputs/requires/phases blocks that could replace spawn-time injection.

**Knowledge:** Skills should declare their requirements (context, constraints, deliverables, phases) in their manifest; orchestration tools should read and honor these declarations rather than duplicating logic.

**Next:** Create epic with 4 children: (1) Extend skill.yaml schema for spawn requirements, (2) Migrate spawn template logic to skill-declared blocks, (3) Add verification integration, (4) Deprecate hardcoded spawn boilerplate.

**Confidence:** High (85%) - Clear architectural path; unknown whether all spawn complexity can be manifest-driven.

---

# Investigation: How Do We Evolve Skills to Be Where True Value Resides?

**Question:** How do we evolve the orch ecosystem so that skills and the procedures/workflows therein are where the true value resides?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** Create epic with children (scope is clear, work is decomposable)
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Value Is Fragmented Across Three Locations

**Evidence:** 
1. **Skills (SKILL.md)** - Procedures, workflows, checklists, self-review (300-2000 lines each)
2. **Spawn templates (orch-go)** - Authority rules, beads tracking, phase reporting, SYNTHESIS.md requirements (~200 lines in context.go:18-196)
3. **CLI logic (orch spawn)** - KB context gathering, tier selection, model defaults, MCP server config

**Source:**
- `pkg/spawn/context.go:18-196` - SpawnContextTemplate with hardcoded boilerplate
- `pkg/skills/loader.go:20-28` - SkillMetadata with 6 fields (name, skill-type, audience, spawnable, category, description)
- Example skill.yaml files show 15-20 fields but spawn template doesn't use most of them

**Significance:** When skills are copied or shared, they lose the spawn context, authority rules, and verification logic. The "true value" is split - skills are necessary but not sufficient.

---

### Finding 2: Skillc Already Has The Right Abstractions

**Evidence:** The skillc manifest supports:
```yaml
outputs:           # SKILL-CONSTRAINTS block - what artifacts must be created
  required:
    - pattern: ".kb/investigations/{date}-*.md"
      description: "Investigation file"
  optional: [...]

requires:          # SKILL-REQUIRES block - what context is needed
  kb_context: true
  beads_issue: true
  prior_work: [".kb/investigations/*"]

phases:            # SKILL-PHASES block - workflow gates
  - name: investigation
    required: false
    exit_criteria:
      - pattern: ".kb/investigations/*.md"
```

**Source:**
- `skillc/pkg/compiler/manifest.go` - Manifest struct with Outputs, Requires, Phases fields
- `skillc/pkg/compiler/compiler.go:230-246` - Generates constraint/requires/phases blocks
- `skillc/pkg/verifier/verifier.go:76-110` - VerifyOutputConstraints implementation

**Significance:** The schema for "skills that know what they need" exists. It's just not wired through to spawn/verification consistently.

---

### Finding 3: Spawn Template Is Doing Skill Work

**Evidence:** The spawn template hardcodes:
- Authority delegation rules (lines 66-82)
- Beads tracking instructions (lines 116-145)
- Status update patterns (lines 107-114)
- SYNTHESIS.md requirements (lines 99-105)
- KB context querying (KBContext field)
- Phase complete protocol (lines 183-195)

These are skill-agnostic but spawn-specific. They could be:
1. Part of a "base skill" that all skills compose with
2. Declared in skill.yaml and injected by spawn
3. Kept in spawn but marked as "system-level" vs skill-level

**Source:** `pkg/spawn/context.go:66-145`

**Significance:** Every skill gets the same boilerplate. Changes require updating spawn, not skills. This inverts the value proposition - orch-go owns agent behavior, not skills.

---

### Finding 4: Verification Integration Is Partially Complete

**Evidence:**
- `pkg/verify/constraint.go` extracts SKILL-CONSTRAINTS from SPAWN_CONTEXT.md
- `pkg/verify/phase_gates.go` extracts SKILL-PHASES and verifies them
- `orch complete` uses these to verify agent work

BUT:
- Constraints are embedded during spawn (context.go doesn't read skill manifests directly)
- Verification happens against spawn context, not skill source
- Changes to skill.yaml outputs require skill rebuild + spawn rebuild

**Source:**
- `pkg/verify/constraint.go:49-116` - ExtractConstraints from SPAWN_CONTEXT.md
- `pkg/verify/phase_gates.go` - Phase gate verification
- kn constraint: "Skill output verification parses skill.yaml directly" (but this isn't fully implemented)

**Significance:** The verification layer is skill-aware but the spawn layer isn't. There's a mismatch in where truth lives.

---

### Finding 5: Skill Portability Is Blocked By Context Dependencies

**Evidence:** A skill needs:
1. **The SKILL.md content itself** - portable (skillc handles this)
2. **Spawn context template** - orch-go specific, not in skill
3. **Verification integration** - orch-go specific, not in skill
4. **KB context patterns** - currently in spawn template
5. **Server context requirements** - hardcoded per-skill in spawn/config.go

**Source:**
- `pkg/spawn/config.go:28-35` - UIFocusedSkills list for server context
- Investigation "Ideal Cross-Repo Setup" noted skills need portability

**Significance:** You can't just copy ~/.claude/skills/worker/investigation to another machine and have it work with any orchestrator. The skill is incomplete without the surrounding infrastructure.

---

## Synthesis

**Key Insights:**

1. **Skill manifests should declare spawn requirements** - Instead of spawn template knowing about skills, skills should declare what context they need (kb_context, servers, beads tracking, authority level). Spawn reads the manifest and honors declarations.

2. **Spawn template should be minimal scaffolding** - The hardcoded 200 lines of spawn context template should shrink to maybe 50 lines of "you're an agent" framing. Everything else comes from skill manifest + shared base skill composition.

3. **Verification should read skill source, not spawn artifact** - Currently constraints are extracted from SPAWN_CONTEXT.md. They should be extracted from the skill.yaml or compiled SKILL.md directly. This makes verification skill-owned, not spawn-owned.

4. **Skills should compose hierarchically** - A "worker-base" skill could provide authority rules, beads tracking, phase reporting. Specific skills inherit from worker-base. This is what skillc's `dependencies` field enables.

**Answer to Investigation Question:**

To make skills where true value resides:

1. **Move spawn-time declarations into skill manifests** - authority, requires.kb_context, requires.servers, requires.beads_tracking
2. **Make spawn template read skill declarations** - Instead of hardcoding, look up skill.yaml and honor its requirements
3. **Create a worker-base skill for shared patterns** - Authority rules, phase reporting, exit protocol as a composable foundation
4. **Verify against skill source** - `orch complete` should read skill.yaml directly, not embedded SKILL-CONSTRAINTS block

This is an **Epic with children** - scope is clear, work is decomposable into discrete tasks.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The architectural direction is clear and supported by existing skillc abstractions. The implementation path is decomposable. Uncertainty remains about edge cases and backward compatibility.

**What's certain:**

- ✅ Skillc already has outputs/requires/phases schema - just needs wiring
- ✅ Spawn template hardcodes 200 lines of skill-agnostic patterns
- ✅ Verification exists but reads embedded blocks, not source manifests
- ✅ Skill composition (dependencies) is supported by skillc

**What's uncertain:**

- ⚠️ Whether all spawn complexity can be manifest-driven (some may need CLI logic)
- ⚠️ Backward compatibility with existing skills during migration
- ⚠️ Whether "worker-base" composition pattern works in practice
- ⚠️ How to handle skills that DON'T want certain spawn behaviors

**What would increase confidence to Very High (95%):**

- Pilot one skill with manifest-driven spawn (investigation skill)
- Test that verification reads skill.yaml instead of embedded blocks
- Validate composition with a base skill + derived skill

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Skill-Manifest-Driven Orchestration** - Skills declare their requirements in skill.yaml; spawn and verify read those declarations.

**Why this approach:**
- Skills become self-contained and portable
- spawn template shrinks to minimal scaffolding
- Verification is skill-owned, not tool-owned
- Follows skillc's existing abstractions

**Trade-offs accepted:**
- Skills need richer manifests (more fields in skill.yaml)
- Migration path required for existing skills
- spawn becomes more complex (reads manifests) but skills become simpler

**Implementation sequence:**
1. **Extend skill.yaml schema** - Add spawn_requires section (authority, kb_context, servers, beads_tracking)
2. **Read manifests in spawn** - LoadSkillContent also loads spawn requirements
3. **Migrate spawn template** - Replace hardcoded blocks with manifest-driven injection
4. **Update verification** - Read skill.yaml directly, not embedded SKILL-CONSTRAINTS

### Alternative Approaches Considered

**Option B: Keep spawn template, add base skill composition**
- **Pros:** Less change to spawn; skills inherit behavior via dependencies
- **Cons:** Still requires spawn to know about base skill; composition complexity
- **When to use instead:** If manifest-driven spawn proves too complex

**Option C: Extract spawn behaviors to hooks**
- **Pros:** Reuses hook infrastructure; clear separation
- **Cons:** Hooks are tool-specific, not skill-owned; increases complexity
- **When to use instead:** If some behaviors must remain tool-level

**Rationale for recommendation:** Skill-manifest-driven approach aligns with skillc's design principles (self-describing artifacts, surfacing over browsing). Skills become the source of truth.

---

### Implementation Details

**What to implement first:**
- Extend skill.yaml schema with spawn_requires section
- Pilot on investigation skill (already migrated to .skillc)
- Verify spawn can read and honor manifest requirements

**Things to watch out for:**
- ⚠️ Backward compatibility - existing skills without spawn_requires should work
- ⚠️ Circular dependency - spawn needs skill, skill loaded during spawn
- ⚠️ Performance - parsing skill.yaml adds spawn latency

**Areas needing further investigation:**
- How to handle skills that want custom authority rules
- Whether SYNTHESIS.md requirement should be skill-declared or tier-declared
- Integration with daemon (auto-spawn from beads)

**Success criteria:**
- ✅ investigation skill works with manifest-driven spawn (no hardcoded blocks)
- ✅ Verification reads skill.yaml, not SPAWN_CONTEXT.md embedded blocks
- ✅ New skill can be created without modifying spawn template

---

## References

**Files Examined:**
- `orch-go/pkg/spawn/context.go:18-196` - SpawnContextTemplate
- `orch-go/pkg/spawn/config.go:28-35` - UIFocusedSkills config
- `orch-go/pkg/skills/loader.go:20-28` - SkillMetadata struct
- `orch-go/pkg/verify/constraint.go:49-116` - Constraint extraction
- `skillc/pkg/compiler/manifest.go` - Manifest struct
- `skillc/pkg/compiler/compiler.go:230-246` - Block generation
- `skillc/pkg/verifier/verifier.go:76-110` - Output verification
- `skillc/DESIGN_PRINCIPLES.md` - AI-native tooling principles

**Commands Run:**
```bash
# Check existing kb context
kb context "skillc"

# Find skill-related code in orch-go
grep -r "skill" pkg/spawn/
```

**Related Artifacts:**
- **Investigation:** `2025-12-22-inv-epic-replace-orch-knowledge-skillc.md` - Prior skillc migration work
- **Decision:** `skillc/.kb/decisions/2025-12-21-skillc-artifact-scope.md` - SKILL.md in scope
- **kn constraint:** "Skill output verification parses skill.yaml directly"

---

## Investigation History

**2025-12-25 06:00:** Investigation started
- Initial question: How do we evolve orch so skills are where true value resides?
- Context: Strategic scoping for skill system evolution

**2025-12-25 06:30:** Context gathering complete
- Found value fragmented across skills, spawn template, CLI logic
- Identified skillc already has right abstractions (outputs, requires, phases)
- Discovered spawn template hardcodes 200 lines that could be skill-declared

**2025-12-25 07:00:** Design synthesis complete
- Determined output type: Epic with children (scope is clear)
- Recommended approach: Skill-manifest-driven orchestration
- Key insight: Skills should declare requirements, spawn should honor them

<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The epic premise is flawed. Skills already contain their true value (procedures, workflows, constraints). What appears to be "leaked value" is actually orchestration infrastructure that belongs in spawn - separating concerns is correct.

**Evidence:** Analyzed spawn template: ~70% is universal orchestration (beads tracking, authority rules, phase reporting) identical for ALL skills. Skill-specific content (outputs, phases, workflow) is already in SKILL.md via skillc compilation.

**Knowledge:** Skills own procedures/workflows/constraints. Spawn owns orchestration infrastructure. The Template Ownership Model decision (2025-12-22) already established this split is intentional.

**Next:** Pause epic orch-go-erdw. Current architecture is sound. Focus on incremental improvements (tier-based SYNTHESIS requirements, optional manifest fields) rather than migration.

**Confidence:** High (85%) - Based on code analysis showing clear separation already exists. Unknown: whether specific edge cases exist where this separation fails.

---

# Investigation: Should We Evolve Skills to Be Where True Value Resides?

**Question:** Should we proceed with epic orch-go-erdw (Skill-Manifest-Driven Orchestration), or is the current architecture already correct?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Recommend pausing epic, documenting decision
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Framing

**Design Question:** Should skills be the primary value container in the orch ecosystem, replacing current spawn template logic with manifest-driven declarations?

**Success Criteria for this decision:**
- Clear understanding of what value currently lives where
- Identification of whether current architecture is working or failing
- Recommendation grounded in evidence, not assumption

**Constraints:**
- Must not violate Session Amnesia principle (next Claude needs discoverable context)
- Must not violate Template Ownership Model (each tool owns its domain)
- Must be practically implementable with existing skillc/orch-go

---

## Findings

### Finding 1: Spawn Template Content Is Mostly Orchestration Infrastructure

**Evidence:** Analyzed `pkg/spawn/context.go:18-196` (SpawnContextTemplate). The 196-line template breaks down:

| Category | Lines | Skill-Specific? | Content |
|----------|-------|-----------------|---------|
| Task/Project/Workspace | ~10 | No (runtime) | `TASK:`, `PROJECT_DIR:`, workspace path |
| Tier instructions | ~15 | No (spawn config) | SYNTHESIS.md requirements by tier |
| Phase reporting | ~20 | No (universal) | bd comment Phase: protocol |
| Authority rules | ~25 | No (universal) | Escalation rules, decision authority |
| Deliverables | ~20 | Partial | Uses `{{.InvestigationSlug}}` from skill |
| Beads tracking | ~30 | No (universal) | bd comment patterns, never bd close |
| Skill content | Variable | Yes | `{{.SkillContent}}` (the actual skill) |

**Source:** `pkg/spawn/context.go:18-196`

**Significance:** ~70% of spawn template is orchestration infrastructure that applies identically to ALL skills. This is not "leaked skill value" - it's correctly placed infrastructure.

---

### Finding 2: Skills Already Contain Their Domain-Specific Value

**Evidence:** Examined investigation skill:

From skill.yaml:
```yaml
outputs:
  required:
    - pattern: ".kb/investigations/{date}-inv-*.md"
      description: "Investigation file with findings"
```

From compiled SKILL.md (deployed to ~/.claude/skills/):
```markdown
<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file with findings -->
<!-- /SKILL-CONSTRAINTS -->
```

The skill declares its required outputs. skillc compiles and embeds them. orch verify extracts and enforces them.

**Source:** 
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/skill.yaml`
- `~/.claude/skills/worker/investigation/SKILL.md`
- `pkg/verify/constraint.go:47-116`

**Significance:** The skillc → SKILL.md → SPAWN_CONTEXT.md → verify pipeline is already working. Skills own their constraints; orch enforces them.

---

### Finding 3: Prior Decision Already Established Correct Ownership Split

**Evidence:** From Template Ownership Model (2025-12-22):

> **The tool that creates the artifact owns its template.**
> 
> | Tool | Owns | Purpose |
> |------|------|---------|
> | kb-cli | Investigation, Decision, Guide, Research | Knowledge artifacts |
> | orch-go | SPAWN_CONTEXT, SYNTHESIS, FAILURE_REPORT | Agent lifecycle artifacts |

The spawn template is correctly classified as "agent lifecycle" - orchestration infrastructure that manages how agents operate, not what work they do.

**Source:** `.kb/decisions/2025-12-22-template-ownership-model.md`

**Significance:** We already made this decision 3 days ago. The epic conflicts with it.

---

### Finding 4: The "Portability" Problem Is Overstated

**Evidence:** The investigation claimed skills need spawn context to be "complete." But skills ARE portable:

1. Copy SKILL.md to another machine's `~/.claude/skills/worker/`
2. Use any orchestrator that reads SKILL-CONSTRAINTS block
3. Skill works as designed

The orchestration infrastructure (beads tracking, phase reporting) is NOT part of the skill - it's part of the orchestrator. Different orchestrators might have different tracking systems.

**Source:** 
- `pkg/skills/loader.go:49-86` - FindSkillPath loads from standard location
- `pkg/spawn/context.go:147-158` - Skill content is embedded, not transformed

**Significance:** Skills are already portable. The "completeness" concern conflates skill-domain with orchestrator-domain.

---

### Finding 5: The Proposed Migration Has Real Costs

**Evidence:** From epic orch-go-erdw:

> Children (5):
> 1. Extend skill.yaml schema with spawn_requires section
> 2. Migrate spawn template to read skill manifest declarations
> 3. Update verification to read skill source manifests
> 4. Create worker-base skill for shared patterns
> 5. Integration: End-to-end skill portability verification

This is significant work (~2-4 weeks) to move content between two places that are BOTH working correctly. The motivation is architectural purity, not solving a real problem.

**Source:** `bd show orch-go-erdw`

**Significance:** Migration has opportunity cost. What real problems would this solve that can't be solved incrementally?

---

## Synthesis

**Key Insights:**

1. **Separation is correct, not accidental** - Skills own domain behavior (what to do). Spawn owns orchestration infrastructure (how to coordinate). These are legitimately different concerns.

2. **"Value leakage" is misdiagnosed** - The spawn template doesn't contain skill value that leaked out. It contains orchestration patterns that are universal to ALL agents, regardless of skill.

3. **Skillc already solved the portability problem** - Skills declare outputs in skill.yaml. skillc compiles them into SKILL-CONSTRAINTS blocks. Any orchestrator can read and enforce them.

4. **The epic conflates two things:**
   - **Skill requirements** (what this skill needs to do its job) - legitimately belongs in skill
   - **Orchestration requirements** (what this orchestrator needs to track work) - legitimately belongs in spawn

**Answer to Investigation Question:**

**Recommendation: Pause epic orch-go-erdw. The premise is flawed.**

The current architecture is sound:
- Skills own their domain (procedures, workflows, output constraints)
- Spawn owns orchestration (beads tracking, authority rules, phase reporting)
- Verification reads skill constraints from spawn context (which embeds the skill)

The proposed "skill-manifest-driven" architecture would:
- Move orchestration concerns INTO skills (making skills tightly coupled to orch-go)
- Reduce portability (skills would need orch-specific spawn_requires sections)
- Add complexity without solving a real problem

**What IS worth doing (incremental improvements):**
1. Make SYNTHESIS.md tier-dependent (already done in spawn template)
2. Add optional manifest fields for skills that need specific context (kb_context, servers)
3. Improve verification to read skill.yaml directly for better error messages

These can be done without the full migration.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The recommendation is based on direct code analysis showing that:
1. Spawn template content is universal orchestration infrastructure
2. Skill-specific content is already in skills
3. The Template Ownership Model decision supports current architecture

**What's certain:**

- ✅ Spawn template is ~70% universal orchestration patterns (code analysis)
- ✅ Skills already declare outputs in skill.yaml and have them enforced
- ✅ Prior decision (Template Ownership Model) supports current split
- ✅ skillc compilation pipeline is working as designed

**What's uncertain:**

- ⚠️ Whether specific edge cases exist where skills genuinely need spawn-level config
- ⚠️ Whether the "worker-base" composition pattern would add value without the full migration
- ⚠️ Whether users perceive the current system as confusing

**What would increase confidence to Very High (95%):**

- Survey of actual usage patterns to confirm orchestration patterns are truly universal
- Attempt to create a new skill WITHOUT modifying spawn - verify it works
- Validate with Dylan that the current separation matches his mental model

---

## Recommendations

### ⭐ RECOMMENDED: Pause Epic, Improve Incrementally

**Why this approach:**
- Current architecture is working and correctly separates concerns
- Epic would move orchestration into skills, increasing coupling
- Incremental improvements can solve specific pain points without migration

**Trade-offs accepted:**
- Spawn template remains ~200 lines (but is shared by ALL skills)
- Some repetition in how spawn generates context (but it's consistent)
- Skills can't customize orchestration behavior (which is arguably correct)

**Implementation sequence:**
1. Pause epic orch-go-erdw (update beads issue)
2. Document this decision in `.kb/decisions/`
3. Create targeted improvements:
   - Make SYNTHESIS.md tier-configurable (done)
   - Add optional `requires.servers` to skill.yaml for UI-focused skills
   - Improve verification error messages to reference skill source

### Alternative Approaches Considered

**Option B: Proceed with Epic (Skill-Manifest-Driven)**
- **Pros:** Cleaner abstraction if all orchestrators shared same patterns
- **Cons:** Couples skills to orch-go; doesn't solve an actual problem; high effort
- **When to use instead:** If multiple orchestrators needed identical spawn patterns

**Option C: Hybrid (Base Skill Composition)**
- **Pros:** Could reduce duplication through inheritance
- **Cons:** Adds complexity; composition in markdown is awkward
- **When to use instead:** If orchestration patterns actually varied by skill type

**Rationale for recommendation:** The epic's goal is architectural purity, but the current architecture is already pure - it correctly separates skill-domain from orchestrator-domain. Evolve by distinction principle applies: when problems recur, ask "what are we conflating?" The investigation conflated skill requirements with orchestration requirements.

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SpawnContextTemplate (196 lines of template)
- `pkg/verify/constraint.go` - Constraint extraction from SPAWN_CONTEXT.md
- `pkg/skills/loader.go` - Skill loading from ~/.claude/skills/
- `~/.claude/skills/worker/investigation/SKILL.md` - Deployed skill with SKILL-CONSTRAINTS
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/skill.yaml` - Skill manifest

**Commands Run:**
```bash
# Check skill deployment
ls -la ~/.claude/skills/worker/investigation/

# Verify constraint extraction works
grep -n "SKILL-CONSTRAINTS" pkg/verify/*.go
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-22-template-ownership-model.md` - Establishes ownership split
- **Investigation:** `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md` - Prior investigation that spawned epic
- **Epic:** `orch-go-erdw` - The epic being evaluated

---

## Investigation History

**2025-12-25 12:00:** Investigation started
- Initial question: Should we proceed with epic orch-go-erdw?
- Context: Epic already created assuming skill-centric direction is correct

**2025-12-25 12:30:** Analyzed spawn template structure
- Found ~70% is universal orchestration infrastructure
- Found skill-specific content already comes from skills
- Recognized separation is intentional, not accidental

**2025-12-25 13:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Pause epic. Current architecture is sound. Improve incrementally.

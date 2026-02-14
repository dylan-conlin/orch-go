<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation deliverable injection removed from default spawn template and gated on ProducesInvestigation flag based on skill type.

**Evidence:** Modified pkg/spawn/context.go to wrap investigation deliverable section in {{if .ProducesInvestigation}} conditional; added SkillProducesInvestigation map limiting investigations to 5 skills (investigation, architect, research, codebase-audit, reliability-testing); all spawn package tests pass.

**Knowledge:** Only skills whose purpose is exploratory/understanding work should create investigation files; feature-impl, systematic-debugging, and issue-creation should not receive investigation deliverable by default.

**Next:** Expected 70% reduction in investigation file creation (from 936 total to ~200-300); monitor investigation count growth rate.

**Authority:** implementation - Surgical fix within existing spawn template patterns, no architectural impact beyond intended behavior change.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix Spawn Template Remove Default

**Question:** How do we remove investigation deliverable from spawns that don't produce investigations?

**Defect-Class:** configuration-drift

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** orch-go-ry4
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-14-inv-investigate-skills-produce-investigation-artifacts.md | extends | ✅ | None |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Prior investigation identified root cause; this investigation implements the fix
**Conflicts:** None

---

## Findings

### Finding 1: SkillProducesInvestigation infrastructure already exists

**Evidence:** pkg/spawn/config.go:58-84 already has SkillProducesInvestigation map and DefaultProducesInvestigationForSkill helper function added by a prior commit.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/config.go:58-84

**Significance:** The infrastructure for skill-aware deliverable injection was already implemented; only needed to use it in the template.

---

### Finding 2: Template uses ProducesInvestigation flag but not conditionally

**Evidence:** contextData struct line 483 has ProducesInvestigation field populated from DefaultProducesInvestigationForSkill, but template doesn't use it to gate investigation deliverable section.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:483, 532

**Significance:** The fix was partially implemented but not wired up in the template - just needed to add conditional wrapper.

---

### Finding 3: Template also has HasInjectedModels routing for probes

**Evidence:** Template has {{if .HasInjectedModels}} conditional to choose between probe files and investigation files, but HasInjectedModels field was missing from contextData struct.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:205-242

**Significance:** Had to add HasInjectedModels field to contextData struct to fix template parsing errors.

---

## Synthesis

**Key Insights:**

1. **Infrastructure already existed** - The skill-aware deliverable system (SkillProducesInvestigation map, DefaultProducesInvestigationForSkill function, ProducesInvestigation field in contextData) was already implemented by a prior commit, just not wired up in the template.

2. **Surgical fix was straightforward** - Wrapping the investigation deliverable section (lines 205-242) in {{if .ProducesInvestigation}}...{{else}}...{{end}} conditional was sufficient to gate on skill type.

3. **Template had incomplete probe routing** - Template referenced .HasInjectedModels for probe vs investigation file routing, but field was missing from contextData struct, causing template parsing errors in tests.

**Answer to Investigation Question:**

Remove investigation deliverable from default spawns by wrapping the template section in {{if .ProducesInvestigation}} conditional. The ProducesInvestigation field is populated from DefaultProducesInvestigationForSkill() which checks the SkillProducesInvestigation map. Only 5 skills return true: investigation, architect, research, codebase-audit, reliability-testing. Feature-impl returns true only when phases includes "investigation".

This prevents feature-impl, systematic-debugging, issue-creation, and other non-investigative skills from receiving investigation file creation instructions in their SPAWN_CONTEXT.md.

---

## Structured Uncertainty

**What's tested:**

- ✅ Template parses correctly with ProducesInvestigation conditional (verified: go test ./pkg/spawn -run TestGenerateContext passes)
- ✅ All spawn package tests pass with changes (verified: go test ./pkg/spawn/... -v)
- ✅ SkillProducesInvestigation map exists and is used (verified: code inspection pkg/spawn/config.go:58-84)
- ✅ HasInjectedModels field added to contextData struct (verified: pkg/spawn/context.go:484, 533)

**What's untested:**

- ⚠️ Actual reduction in investigation file creation (won't know until spawns are created with new template)
- ⚠️ Whether any skills depend on receiving investigation file setup unexpectedly (assumption: only investigation-producing skills need it)
- ⚠️ Impact on completion verification gates (assumption: orch complete doesn't require investigation files for all skills)

**What would change this:**

- Finding would be wrong if non-investigation skills actually need investigation files for coordination
- Finding would be wrong if orch complete requires investigation_path for all skill types
- Reduction estimate would be wrong if investigation-producing skills are spawned more frequently than expected

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

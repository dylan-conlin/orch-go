<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully removed 9 unused skills (0-1 spawns each) from orch-knowledge and all references across the codebase.

**Evidence:** Removed skill directories from ~/orch-knowledge/skills/src/ and ~/.claude/skills/, updated orchestrator skill template and daemon inference code, verified skillc deploy (11/11 skills deployed) and go test (all pass).

**Knowledge:** Skill references existed in 4 locations: (1) orchestrator skill template Quick Decision Tree and model selection, (2) daemon skill_inference.go and tests for kb-reflect title pattern, (3) orch-go CLAUDE.md for server context list, (4) no global CLAUDE.md references found.

**Next:** Commit changes - all verification passed, no broken references remain.

**Authority:** implementation - Removing unused code within established patterns, no architectural decisions needed.

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

# Investigation: Remove Unused Skills Orch Knowledge

**Question:** Which files reference the 9 unused skills, and what needs to be cleaned up after removal?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent (orch-go-bt7)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: All 9 skill directories existed and were removed cleanly

**Evidence:** Removed from ~/orch-knowledge/skills/src/: block-processing, writing-skills (meta); design-principles, delegating-to-team, issue-quality (shared); reliability-testing, ui-design-session, kb-reflect, issue-creation (worker). Also removed deployed versions from ~/.claude/skills/.

**Source:** Commands: `rm -rf` for each skill directory in both source and deployed locations.

**Significance:** Clean removal with no orphaned files ensures no confusion about which skills are available.

---

### Finding 2: Orchestrator skill had 6 references to removed skills

**Evidence:** Found in ~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template at lines 257-258, 261, 268, 337 - references in Quick Decision Tree, bug triage guidance, and --opus model selection list.

**Source:** grep search and manual inspection of template file.

**Significance:** Template file (not generated SKILL.md) needed editing to prevent regeneration of outdated references.

---

### Finding 3: Daemon had kb-reflect inference logic for title patterns

**Evidence:** skill_inference.go:87 had "Synthesize * investigations" → kb-reflect pattern. Tests in daemon_test.go:494, 516-517, 554 verified this behavior.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/skill_inference.go and daemon_test.go

**Significance:** Removing unused inference logic prevents daemon from attempting to spawn non-existent skills.

---

### Finding 4: CLAUDE.md had one reference to reliability-testing

**Evidence:** Line 182 in orch-go/CLAUDE.md listed reliability-testing in server context skills list.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md:182

**Significance:** Minor documentation update to keep spawn context generation accurate.

---

### Finding 5: No references in global CLAUDE.md

**Evidence:** grep search for all 9 skill names returned no results in ~/.claude/CLAUDE.md

**Source:** grep command on ~/.claude/CLAUDE.md

**Significance:** No global configuration cleanup needed.

---

## Synthesis

**Key Insights:**

1. **Skill removal requires coordination across 4 systems** - Source directories, deployed directories, orchestrator guidance, and daemon inference all needed updates to fully remove skill references.

2. **Orchestrator template vs generated file distinction matters** - Editing .skillc/SKILL.md.template (not SKILL.md) ensures changes persist through skillc rebuild cycles.

3. **Daemon inference uses both labels and title patterns** - kb-reflect was unique in having title-based inference ("Synthesize * investigations"), requiring test updates beyond just label removal.

**Answer to Investigation Question:**

The 9 unused skills had references in exactly 4 locations: (1) orchestrator skill template (Quick Decision Tree, bug triage, --opus list), (2) daemon skill inference (kb-reflect title pattern + tests), (3) orch-go CLAUDE.md (server context list), and (4) the skill directories themselves (source + deployed). All references were successfully removed, with verification showing skillc deploy works (11/11 skills) and go tests pass.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

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

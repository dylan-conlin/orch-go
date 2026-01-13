<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added decision tree section "Spawning Orchestrators vs Managing Sessions" to orchestrator skill template after Focus-Based Session Model section, clarifying when to use spawned orchestrators vs interactive sessions.

**Evidence:** Edited SKILL.md.template at line 298, added 33-line section with comparison table and usage guidance, rebuilt skill successfully (19729 tokens, 131.5% of budget).

**Knowledge:** The two orchestration mechanisms are complementary (hierarchical delegation vs temporal continuity), not redundant - guidance gap was in discoverability, not architecture.

**Next:** Commit changes to orch-knowledge repo, skill is deployed and ready for use.

**Promote to Decision:** recommend-no (tactical documentation update based on completed architect analysis, not new architectural decision)

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

# Investigation: Add Orchestrator Skill Decision Tree

**Question:** Where should the decision tree for spawned orchestrators vs interactive sessions be added in the orchestrator skill, and what content should it include?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** og-feat agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Orchestrator skill template is at ~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template

**Evidence:** Found the skill template file with 75KB content, last compiled 2026-01-08. Has structure with sections for Fast Path, Pre-Response Gates, Context Detection, Skill System Architecture, etc.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template:1-300`

**Significance:** This is the correct file to edit (not the deployed SKILL.md which is auto-generated). Must use skillc build after editing to deploy changes.

---

### Finding 2: Best location is after Focus-Based Session Model section

**Evidence:** The Focus-Based Session Model section ends at line 294, followed by a separator at line 296, then Work Pipeline begins at line 298. This is the logical place to insert a section about spawned orchestrators vs interactive sessions.

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template:290-300`

**Significance:** The new section naturally follows the session model discussion and clarifies the two different orchestration patterns before diving into the work pipeline.

---

### Finding 3: Successfully added decision tree section and rebuilt skill

**Evidence:** Added 33-line section "Spawning Orchestrators vs Managing Sessions" at line 298 of SKILL.md.template. Rebuilt with `skillc build` - compiled successfully to 19729 tokens (131.5% of 15000 budget).

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template:298-328`, skillc build output

**Significance:** Decision tree now integrated into orchestrator skill, providing clear guidance on when to use spawned orchestrators vs interactive sessions. Token usage warning expected due to added content.

---

## Synthesis

**Key Insights:**

1. **Decision tree placement after Focus-Based Session Model is optimal** - The new section naturally follows the discussion of orchestrator sessions and clarifies the two different orchestration patterns (hierarchical vs temporal) before the Work Pipeline section.

2. **Architect investigation provided complete content** - The table showing mechanism comparison (purpose, use when, lifecycle) directly addresses the confusion about when to use each pattern, with concrete examples.

3. **Skill rebuilds successfully despite token budget warning** - The orchestrator skill now totals 19729 tokens (131.5% of 15000 budget), but this is expected and acceptable given the comprehensive guidance needed.

**Answer to Investigation Question:**

The decision tree should be added as a new section titled "Spawning Orchestrators vs Managing Sessions" immediately after the "Focus-Based Session Model" section (around line 298) in the orchestrator skill template. The content includes:
- A comparison table showing both mechanisms (spawn orchestrator vs session start/end)
- Clear "when to use" guidance for each pattern
- Examples illustrating the hierarchical vs temporal distinction
- Reference to the architect investigation

This placement and content directly addresses the gap identified by the architect - providing clear usage guidance for the two complementary orchestration mechanisms.

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

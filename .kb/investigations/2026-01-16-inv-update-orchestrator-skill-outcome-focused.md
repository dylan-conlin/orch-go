<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added Goal Framing section to orchestrator skill after Pre-Response Gates section - helps orchestrators recognize when action verbs trigger worker-level behavior.

**Evidence:** Section deployed to ~/.claude/skills/meta/orchestrator/SKILL.md line 80; includes examples of action vs outcome verbs with side-by-side comparison table.

**Knowledge:** Orchestrator skill source is SKILL.md in .skillc directory (not SKILL.md.template despite skill.yaml config); template system not currently working for this skill.

**Next:** Close issue - guidance is deployed and will be loaded in future orchestrator sessions.

**Promote to Decision:** recommend-no - tactical addition to existing skill, not architectural

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

# Investigation: Update Orchestrator Skill Outcome Focused

**Question:** Where in the orchestrator skill should outcome-focused goal guidance be added, and what should it say?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker agent (og-feat-update-orchestrator-skill-16jan-49dd)
**Phase:** Implementing
**Next Step:** Add goal framing section to orchestrator skill template
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Orchestrator skill source is SKILL.md, not SKILL.md.template

**Evidence:** skill.yaml lists SKILL.md.template as source, but running `skillc build` after editing template did not update SKILL.md; SKILL.md is 1925 lines vs template's 1152 lines; git history shows SKILL.md edited more recently than template.

**Source:** ~/orch-knowledge/skills/src/meta/orchestrator/.skillc/skill.yaml, SKILL.md, SKILL.md.template

**Significance:** Template system not working for orchestrator skill - must edit SKILL.md directly despite AUTO-GENERATED header warning. This differs from worker skills which use proper template/phase systems.

---

### Finding 2: Best placement is after Pre-Response Gates section

**Evidence:** Pre-Response Gates is at line 69-77, establishes checking pattern before every response; Goal Framing fits naturally as part of pre-response thinking.

**Source:** ~/.claude/skills/meta/orchestrator/SKILL.md:69-77

**Significance:** Placement right after gates ensures orchestrators see goal framing guidance early in skill, before delegation rules and context detection.

---

### Finding 3: Guidance includes concrete examples with comparison table

**Evidence:** Section includes 4 examples comparing action-focused vs outcome-focused phrasing (e.g., "Implement handoff enforcement gate" vs "Ship handoff enforcement gate - ensure orchestrator completions require filled handoffs").

**Source:** ~/.claude/skills/meta/orchestrator/SKILL.md:80-117

**Significance:** Concrete examples make the distinction clear and actionable; orchestrators can pattern-match their own goal phrasing against the table.

---

## Synthesis

**Key Insights:**

1. **Verb choice shapes cognitive frame** - Using action verbs (implement, fix, add) in goals triggers worker-level thinking patterns, while outcome verbs (ship, complete, close) keep orchestrators in coordination mode. This happens before conscious decision-making.

2. **Orchestrator skill build system is non-standard** - Unlike worker skills which use template expansion or phase concatenation, the orchestrator skill's SKILL.md must be edited directly despite having a SKILL.md.template file and skill.yaml configuration.

3. **Early placement maximizes visibility** - Placing Goal Framing after Pre-Response Gates (line 80) ensures orchestrators encounter this guidance during initial skill scan, before reading delegation rules and context detection.

**Answer to Investigation Question:**

Goal framing guidance was added at line 80 of ~/.claude/skills/meta/orchestrator/SKILL.md, immediately after the Pre-Response Gates section. This placement ensures orchestrators see it early while reviewing pre-response checks. The section includes concrete examples comparing action-focused vs outcome-focused phrasing, with a comparison table for pattern-matching. The guidance was added by editing SKILL.md directly (not via template) due to non-functional template system for this particular skill.

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

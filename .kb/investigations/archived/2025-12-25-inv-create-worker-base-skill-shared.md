<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created worker-base skill with common worker patterns and added LoadSkillWithDependencies to orch-go for runtime dependency resolution.

**Evidence:** All orch-go tests pass; worker-base skill compiles to 1014 tokens; investigation skill now declares worker-base dependency.

**Knowledge:** skillc doesn't support cross-directory dependencies at compile time - resolution must happen at runtime via orch-go's skill loader.

**Next:** Close - deliverables complete. Follow-up: update more worker skills to depend on worker-base; consider updating skillc for compile-time resolution.

**Confidence:** High (85%) - Runtime resolution tested via unit tests; end-to-end spawn test not performed.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Create Worker Base Skill Shared

**Question:** How to create a composable worker-base skill that provides common patterns shared by all worker skills?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent (spawned from orch-go-erdw.4)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Worker-base skill structure follows shared skill pattern

**Evidence:** Created worker-base in `orch-knowledge/skills/src/shared/worker-base/.skillc/` with:
- skill.yaml (composable: true, foundation type)
- intro.md, authority.md, tracking.md, phase-reporting.md, completion.md
- Compiled SKILL.md is 1014 tokens

**Source:** `~/orch-knowledge/skills/src/shared/worker-base/.skillc/`

**Significance:** Worker-base provides common patterns (authority delegation, beads tracking, phase reporting, completion protocol) that can be inherited by other worker skills.

---

### Finding 2: skillc doesn't support cross-directory dependencies

**Evidence:** When investigation skill declares `dependencies: [worker-base]`, skillc build fails with:
```
failed to resolve dependencies: skill 'investigation' depends on 'worker-base', but 'worker-base' not found
```
skillc validates dependencies against Skills map populated from current .skillc directory only.

**Source:** `~/Documents/personal/skillc/pkg/graph/graph.go:31`

**Significance:** Dependency resolution must happen at runtime (spawn time) rather than compile time. This is why orch-go's LoadSkillWithDependencies was needed.

---

### Finding 3: Runtime dependency resolution works via skill loader

**Evidence:** Added `LoadSkillWithDependencies` to `pkg/skills/loader.go` which:
1. Loads main skill content
2. Parses dependencies from frontmatter
3. Loads and prepends each dependency's content (stripped of its own frontmatter)
All tests pass including new TestLoadSkillWithDependencies.

**Source:** `pkg/skills/loader.go:104-149`, `pkg/skills/loader_test.go:199-270`

**Significance:** The spawn command now uses LoadSkillWithDependencies, so skills that declare dependencies will have their dependency content prepended at spawn time.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

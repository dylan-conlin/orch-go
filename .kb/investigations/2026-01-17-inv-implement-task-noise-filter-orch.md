<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Task-specific noise (issue IDs and phase names) should be filtered in FindRecurringGaps() to prevent spurious suggestions.

**Evidence:** Code analysis shows FindRecurringGaps() already filters resolved events around line 307-314; adding noise filtering here follows the same pattern.

**Knowledge:** Issue IDs follow pattern `orch-go-xxxxx` or `og-feat-xxxxx`; Phase names follow pattern `Phase: <name>`; these appear in gap queries due to task descriptions.

**Next:** Implement isTaskNoise() function and integrate into FindRecurringGaps() with tests.

**Promote to Decision:** recommend-no (tactical filtering improvement, not architectural)

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

# Investigation: Implement Task Noise Filter Orch

**Question:** How should orch learn filter task-specific noise (issue IDs, phase names) from gap patterns?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** feature-impl agent
**Phase:** Synthesizing
**Next Step:** Implement isTaskNoise() function
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: FindRecurringGaps already has filtering pattern

**Evidence:** Lines 307-314 in pkg/spawn/learning.go show filtering logic for resolved events: `if e.Resolution != "" { continue }`

**Source:** pkg/spawn/learning.go:307-314

**Significance:** Task noise filtering should follow the same pattern - check condition, skip event if matched. This keeps the implementation consistent with existing code.

---

### Finding 2: Issue ID patterns are predictable

**Evidence:** Beads issue IDs follow format: `{project}-{id}` where project is usually `orch-go`, `og-feat`, etc. and id is alphanumeric. Examples: `orch-go-0vscq.5`, `og-feat-implement-task-noise-17jan-8344`

**Source:** spawn context (line 116, 222), beads comments throughout codebase

**Significance:** Regex pattern `^\w+-\w+` will match issue IDs while avoiding false positives from normal queries like "add feature" or "debug issue".

---

### Finding 3: Phase announcements are standardized

**Evidence:** Phase updates follow pattern `Phase: {name}` where name is Planning, Implementing, Testing, Complete, etc. Used in beads comments via `bd comment` calls.

**Source:** SPAWN_CONTEXT.md (lines 116, 222, 239), feature-impl skill guidance

**Significance:** Simple prefix check `strings.HasPrefix("phase:")` after normalization will catch all phase-related queries without complex regex.

---

## Synthesis

**Key Insights:**

1. **Filtering in FindRecurringGaps preserves raw data** - Adding noise filtering in FindRecurringGaps() (like existing resolution filtering) keeps the raw gap events for potential future analysis while preventing noise from generating suggestions.

2. **Pattern matching needs low false-positive rate** - Issue IDs are distinct enough (project-id format) and phase announcements are standardized enough (Phase: prefix) that simple pattern matching won't filter legitimate queries.

3. **Consistency with existing patterns matters** - The resolution filtering pattern (check condition → continue) provides a clear template that makes the code predictable and maintainable.

**Answer to Investigation Question:**

Task noise filtering should be implemented in FindRecurringGaps() using an isTaskNoise() helper function that checks for: (1) issue ID pattern matching `^\w+-\w+` regex, and (2) phase announcements with `phase:` prefix after normalization. This approach follows the existing resolution filtering pattern (Finding 1), uses predictable patterns with low false-positive rates (Findings 2, 3), and filters at the right layer to prevent spurious suggestions while preserving raw event data.

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

**Add isTaskNoise() filter in FindRecurringGaps()** - Create a helper function that identifies task-specific patterns and skip those events during gap analysis.

**Why this approach:**
- Follows existing resolution filtering pattern (consistent with pkg/spawn/learning.go:307-314)
- Preserves raw event data for potential future analysis
- Low false-positive rate due to distinct patterns (issue IDs, phase prefixes)
- Single point of control for noise filtering logic

**Trade-offs accepted:**
- Won't filter noise from historical events already processed (only affects future suggestions)
- Requires regex compilation overhead (negligible for typical usage patterns)

**Implementation sequence:**
1. Add isTaskNoise() helper function - foundational pattern matching logic
2. Integrate into FindRecurringGaps() loop - apply filter alongside resolution check
3. Add comprehensive tests - verify both filtering and non-filtering cases

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

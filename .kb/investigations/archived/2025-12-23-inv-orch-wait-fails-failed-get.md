<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added resolveBeadsID function to wait.go that resolves session IDs to beads IDs by reading SPAWN_CONTEXT.md from the workspace directory.

**Evidence:** Smoke tests confirm wait command now accepts session IDs (ses_xxx), beads IDs (proj-xxx), and workspace names (og-xxx-23dec), all resolving correctly to beads IDs.

**Knowledge:** Session titles contain workspace names, not beads IDs - must read SPAWN_CONTEXT.md to extract beads ID; workspace names can have hyphens so pattern-matching alone cannot distinguish them from beads IDs.

**Next:** Fix committed and tested - ready for orch complete.

**Confidence:** High (90%) - smoke tested all three identifier formats successfully

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

# Investigation: Orch Wait Fails Failed Get

**Question:** Why does `orch wait` fail with "failed to get issue" when given a session ID instead of a beads issue ID?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: orch wait directly calls verify.GetIssue without resolving identifiers

**Evidence:** 
- `cmd/orch/wait.go:148` calls `verify.GetIssue(beadsID)` directly with the user-provided argument
- `pkg/verify/check.go:427-446` shows `GetIssue` runs `bd show beadsID --json`
- If a session ID like `ses_abc123` is passed, the `bd` CLI will fail because it doesn't know about OpenCode session IDs

**Source:** 
- `cmd/orch/wait.go:148`
- `pkg/verify/check.go:427-446`

**Significance:** The command assumes the user always passes a beads ID, but other commands like `orch send` accept session IDs and resolve them internally.

---

### Finding 2: A resolveSessionID function exists but does the opposite resolution

**Evidence:**
- `cmd/orch/main.go` has a `resolveSessionID` function that converts beads ID → session ID
- The function handles session IDs, beads IDs, and workspace names, converting them all to session IDs
- But `orch wait` needs the opposite: session ID → beads ID

**Source:**
- `cmd/orch/main.go:1551` (resolveSessionID function)

**Significance:** Need to create a new function that resolves in the opposite direction, or extract the beads ID from a session.

---

### Finding 3: Session titles contain beads IDs in [brackets]

**Evidence:**
- `extractBeadsIDFromTitle` function extracts beads ID from session title using `[beads-id]` pattern
- Sessions can be looked up via the OpenCode API using session ID
- The session title contains the beads ID in brackets

**Source:**
- `cmd/orch/main.go:1400` (extractBeadsIDFromTitle function)
- `pkg/opencode/client.go:210` (GetSession function)

**Significance:** Can resolve session ID → beads ID by: (1) Get session from OpenCode API, (2) Extract beads ID from title using extractBeadsIDFromTitle

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

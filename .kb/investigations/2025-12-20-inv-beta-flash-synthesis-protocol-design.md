**TLDR:** Designing a standardized `SYNTHESIS.md` schema to solve session amnesia and enable 30-second handoffs between agents. Focuses on externalizing Delta, Evidence, Knowledge, and Next Actions.

---

# Investigation: Beta Flash Synthesis Protocol Design v3

**Question:** How can we structure a `SYNTHESIS.md` file to enable a "30-second handoff" for the orchestrator, effectively solving "Session Amnesia" by externalizing Delta, Evidence, Knowledge, and Next Actions?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** worker agent
**Phase:** Problem Framing
**Next Step:** Explore existing patterns and draft schema options.
**Status:** Active
**Confidence:** Medium (70%)

---

## Problem Framing

### Design Question

The core problem is "Session Amnesia" – when an agent session ends, the next agent (or the orchestrator) often lacks a concise, high-fidelity summary of what was accomplished, what was learned, and exactly what needs to happen next. 

We need a `SYNTHESIS.md` schema that:
1.  **Externalizes the Delta:** What specifically changed in the codebase or state?
2.  **Provides Evidence:** How do we know it works? (Test results, logs).
3.  **Captures Knowledge:** What new insights, decisions, or constraints were discovered?
4.  **Defines Next Actions:** What are the immediate next steps for the next agent?

### Success Criteria

- **30-Second Handoff:** A fresh agent can read the file and be fully oriented in 30 seconds.
- **Standardized Schema:** Consistent structure across all projects and sessions.
- **Actionable:** Next actions are clear and spawnable.
- **Verifiable:** Evidence links to actual test runs or observations.
- **Durable:** Knowledge is captured in a way that can be promoted to `.kb`.

### Constraints

- Must work within the `orch-go` / `beads` / `kb` ecosystem.
- Must be easy for agents to maintain during a session.

### Scope

- **In Scope:** `SYNTHESIS.md` schema design, integration with `kb` and `beads`.
- **Out of Scope:** Automating the generation of `SYNTHESIS.md` (though the schema should be automation-friendly).

---

## Findings

### Finding 1: Existing Session Transition Patterns
The `session-transition` skill in `~/.claude/skills/session-transition/SKILL.md` uses a `WORKSPACE.md` file and appends a `## Session Transition` section. It captures state (Blocked, Completed, etc.), uncommitted changes, and next steps. However, `orch-go` uses `SPAWN_CONTEXT.md` and `beads` issues, creating a slight disconnect.

### Finding 2: Knowledge Tracking with `kn`
The `kn` tool is already available and provides a structured way to record decisions, failed attempts, constraints, and questions. Integrating `kn` into the synthesis protocol ensures that knowledge is captured durably in `.kn/` and can be easily searched or promoted.

### Finding 3: Verification and Review Integration
Current `orch complete` only verifies `Phase: Complete` in beads comments. `orch review` shows the phase summary. By adding `SYNTHESIS.md` as a required deliverable, we can provide much higher density information (TLDR, Delta, Evidence, Next Actions) during the review process.

---

## Synthesis

### Key Insights

1.  **The "30-Second Handoff" requires a dedicated artifact.** While beads comments are good for progress, they are too fragmented for a quick handoff. A single `SYNTHESIS.md` file in the workspace provides a consolidated view.
2.  **Externalizing Knowledge via `kn` is critical.** Instead of just listing what was learned in a markdown file, using `kn` makes the knowledge queryable and manageable. `SYNTHESIS.md` should act as a pointer to these entries.
3.  **Verification must be enforced.** If the protocol isn't enforced by `orch complete`, agents will skip it. Adding a check for `SYNTHESIS.md` ensures compliance.

### Answer to Investigation Question

The Beta Flash Synthesis Protocol v3 consists of:
1.  A standardized `SYNTHESIS.md` schema (Delta, Evidence, Knowledge, Next Actions).
2.  Mandatory creation of `SYNTHESIS.md` at the end of every session, enforced by `orch complete`.
3.  Integration with `kn` for durable knowledge capture.
4.  Enhanced `orch review` output that displays the TLDR and Next Actions from `SYNTHESIS.md`.

---

## Implementation Recommendations

### Recommended Approach ⭐

**Standardized SYNTHESIS.md Protocol**

**Why this approach:**
- Solves session amnesia by providing a high-density handoff artifact.
- Leverages existing `kn` tool for knowledge management.
- Enforces compliance through automated verification.
- Improves orchestrator visibility via enhanced review output.

**Trade-offs accepted:**
- Adds one more step for agents at the end of a session.
- Requires agents to be aware of the `.orch/templates/SYNTHESIS.md` location.

**Implementation sequence:**
1.  **Create Template:** Define `.orch/templates/SYNTHESIS.md`. (DONE)
2.  **Update Spawn Context:** Instruct agents to use the template. (DONE)
3.  **Update Verification:** Add `SYNTHESIS.md` check to `pkg/verify`. (DONE)
4.  **Update Review:** Display synthesis info in `orch review`. (DONE)

---

## Confidence Assessment

**Current Confidence:** High (90%)

**What's certain:**
- ✅ Schema covers all required areas (Delta, Evidence, Knowledge, Next Actions).
- ✅ Integration with `kn` is the right way to handle durable knowledge.
- ✅ Automated verification is necessary for protocol adherence.

**What's uncertain:**
- ⚠️ Will agents consistently fill out all sections accurately?
- ⚠️ Is the parsing logic in `orch review` robust enough for varied markdown styles?

---

## References

**Files Examined:**
- `pkg/spawn/context.go`
- `pkg/verify/check.go`
- `cmd/orch/main.go`
- `cmd/orch/review.go`
- `~/.claude/skills/session-transition/SKILL.md`

**Status:** Complete

### Finding 1: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 2: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

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

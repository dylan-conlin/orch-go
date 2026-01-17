<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Identify Orchestrator Value Add Vs

**Question:** When does orchestrator judgment actually matter vs when is it just routing? If daemon could handle 80% of spawns correctly, orchestrator should focus on the 20%. Informs: daemon autonomy expansion, orchestrator focus areas.

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-identify-orchestrator-value-17jan-95d9
**Phase:** Synthesizing
**Next Step:** Complete synthesis and write recommendations
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Strategic Orchestrator Model Redefines the Division of Labor

**Evidence:** Decision document "Strategic Orchestrator Model" (2026-01-07) established that orchestrator's job is **comprehension**, not coordination. Coordination is the daemon's job. The work division:

| Work Type | Who Does It | Why |
|-----------|-------------|-----|
| Investigation (discovering facts) | Worker agent | Requires codebase exploration |
| Implementation (writing code) | Worker agent | Requires file editing |
| Synthesis (combining findings) | Strategic orchestrator | Requires cross-agent context |
| Understanding (building models) | Strategic orchestrator | Requires engagement, not delegation |
| Coordination (what to spawn when) | Daemon | Already automated |

**Source:** 
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md`
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md`
- `.kb/models/daemon-autonomous-operation.md`

**Significance:** This reframes the question. Orchestrators aren't meant to do routing at all - that's already automated. The question is whether orchestrators are spending time on comprehension/synthesis (high value) or getting pulled into tactical dispatch (low value).

---

### Finding 2: High-Value Activities Require Strategic Judgment, Not Routing

**Evidence:** Investigation "Interactive Orchestrators Compensation Pattern" (2026-01-06) identified three legitimate orchestrator functions that daemon cannot replicate:

1. **Goal refinement** - Converting vague strategic intent ("improve performance") to actionable orchestrator goals ("reduce orch status latency from 1.2s to <100ms")
2. **Real-time frame correction** - Catching when orchestrator drops into tactical mode (doing spawnable work) and shifting perspective
3. **Synthesis** - Combining worker results into decisions/knowledge (can't spawn "understand this topic")

Additional high-value activities from orchestrator skill:
- **Hotspot detection** - Recognizing when 5+ bug fixes to same area signals systemic issue requiring architect, not more debugging
- **Issue type correction** - When daemon skill inference would be wrong (issue labeled "task" but actually needs feature-impl)
- **Follow-up extraction** - Reading SYNTHESIS.md recommendations from completed agents and deciding what to pursue
- **Epic readiness evaluation** - Determining if understanding is complete enough to spawn work ("can you explain the problem, constraints, and risks?")

**Source:** 
- `.kb/investigations/2026-01-06-inv-investigate-interactive-orchestrators-compensation-pattern.md` lines 118-130
- Orchestrator skill "Strategic-First Orchestration" section
- Orchestrator skill "Orchestrator Core Responsibilities" section

**Significance:** These are categorically different from queue processing. They require reasoning, judgment, and cross-agent context that daemon cannot replicate. This is the 20% that requires orchestrator engagement.

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

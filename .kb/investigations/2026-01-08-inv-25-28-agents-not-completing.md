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

# Investigation: Why Are 25-28% of Agents Not Completing?

**Question:** For agents that fail to report Phase: Complete, what happens to them? Are they hitting rate limits, crashing, forgetting to report, or getting stuck?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-inv-25-28-agents-08jan-85d0
**Phase:** Investigating
**Next Step:** Analyze specific failure examples in depth
**Status:** In Progress

---

## Findings

### Finding 1: Initial Statistics Show 72.5% Completion Rate

**Evidence:** 
- orch stats (7-day window): 425 spawns, 308 completions (72.5%), 33 abandonments (7.8%)
- That leaves ~84 spawns (19.8%) unaccounted for - neither completed nor abandoned
- Prior investigation (2026-01-06) found similar rate of ~75% when accounting for data quality

**Source:** `orch stats --json` output, comparison with prior investigation

**Significance:** The "25-28% not completing" aligns with ~84 missing spawns out of 425. The question is: what happened to these 84?

---

### Finding 2: Abandonment Reasons Show Clear Categories

**Evidence:** Analysis of 101 total abandonments (all-time) shows these categories:

| Category | Count | Examples |
|----------|-------|----------|
| Rate limit related | ~10-20 | "Orphaned from rate limit crash", "Stuck after rate limit" |
| Stuck/stalled | ~21 | "Stuck at Planning for 19+ minutes", "Stalled - no phase comments" |
| Testing/cleanup | ~15 | "Testing session deletion fix", "Already abandoned - cleanup" |
| CPU overload | 3 | "CPU overload" |
| Session death | ~8 | "Session died - no process running", "Session disconnected" |
| Wrong skill/scope | ~5 | "Wrong skill - need architect not feature-impl" |
| No reason | 43 | Pre-Dec 24 abandonments lack reason field |

**Source:** `grep '"type":"agent.abandoned"' ~/.orch/events.jsonl | grep -o '"reason":"[^"]*"' | sort | uniq -c | sort -rn`

**Significance:** Rate limits and stuck/stalled sessions are the top controllable causes. ~21% of abandonments are testing-related and should be excluded from production metrics.

---

### Finding 3: Many "Missing" Completions Are Actually Closed Issues

**Evidence:** Sampled 5 issues that had spawn events but no completion events:
- orch-go-03oxi: Status CLOSED, proper close reason, no completion event
- orch-go-04o7j: Status CLOSED ("Zombie reconciled"), no completion event  
- orch-go-0c3zy: Status CLOSED, has SYNTHESIS.md in workspace, no completion event, NO bd comments
- orch-go-0c9q2: Status CLOSED, has SYNTHESIS.md in workspace, no completion event
- orch-go-0cmd6: Status CLOSED, proper close reason, no completion event

**Source:** `bd show <id>`, workspace inspection, `grep '"type":"session.completed"' ~/.orch/events.jsonl | grep '<beads_id>'`

**Significance:** Agents are completing their work (creating SYNTHESIS.md, closing issues) but NOT triggering session.completed events. This is a tracking bug, not a failure to complete work.

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

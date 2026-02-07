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

# Investigation: Coach Session Pattern Monitoring

**Question:** How does the coach session streaming work, and is this session properly receiving streamed metrics from the coaching plugin?

**Started:** 2026-01-10 15:30
**Updated:** 2026-01-10 15:42
**Owner:** Agent og-inv-coach-session-pattern-10jan-4f3f
**Phase:** Investigating
**Next Step:** Document architecture and test message reception
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Task Scope Ambiguity

**Evidence:** 
- SPAWN_CONTEXT.md line 1: "TASK: coach session for pattern monitoring"
- No further details about what specifically should be monitored
- Prior knowledge shows 10+ coach-related investigations today, mostly empty templates
- Constraints mention: session filtering, behavioral variation detection (3+ attempts), cross-document parsing

**Source:** 
- /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-coach-session-pattern-10jan-4f3f/SPAWN_CONTEXT.md
- .kb/investigations/2026-01-10-inv-*.md (multiple prior investigations)

**Significance:** 
Multiple valid interpretations exist:
1. Test/validate that pattern monitoring in coaching plugin works correctly
2. Investigate what patterns should be monitored
3. Document how pattern monitoring currently works
4. Debug pattern monitoring if it's not working

Per AUTHORITY section: "Requirements ambiguous (multiple valid interpretations exist)" requires escalation to orchestrator.

---

### Finding 2: Coach Session Architecture Clarified

**Evidence:**
- Coaching plugin at `plugins/coaching.ts` lines 536-572 implements `streamToCoach()` function
- Uses `client.session.promptAsync()` to send metrics asynchronously to coach session
- Coach session ID set via `ORCH_COACH_SESSION_ID` environment variable (line 57)
- Infinite loop prevention: skips streaming if current session IS the coach (line 549)
- Test message "Can you receive this?" received successfully from orchestrator

**Source:**
- plugins/coaching.ts:536-572 (streamToCoach function)
- plugins/coaching.ts:57 (COACH_SESSION_ID env var)
- User message received at 15:42

**Significance:**
This session is INTENDED to be the coach session that receives streamed pattern detections. The test message confirms the communication channel works. This is Phase 3 of the coaching plugin ("Stream to coach session for investigation").

---

### Finding 3: Detectable Pattern Types

**Evidence:**
The coaching plugin detects and streams 5 pattern types:

1. **behavioral_variation** (lines 1041-1068)
   - 3+ consecutive variations in same semantic group without strategic pause
   - Example: overmind start, overmind status, overmind restart repeatedly
   - Threshold: `VARIATION_THRESHOLD = 3`, pause threshold: `STRATEGIC_PAUSE_MS = 30000`

2. **circular_pattern** (lines 1082-1113)
   - Architectural decisions contradicting prior investigation recommendations
   - Example: Creating launchd plist when investigation recommended overmind
   - Uses keyword extraction from D.E.K.N. summaries

3. **dylan_signal_prefix** (lines 851-870)
   - Explicit prefixes: frame-collapse:, compensation:, focus:, step-back:
   - Indicates Dylan manually flagging orchestrator behavior

4. **priority_uncertainty** (lines 872-900)
   - "what's next?" type questions appearing 2+ times
   - Signals orchestrator not providing strategic guidance

5. **compensation_pattern** (lines 902-936)
   - Dylan providing repeated context (>30% keyword overlap)
   - Indicates system failing to surface knowledge

**Source:**
- plugins/coaching.ts:1041-1113 (detection implementation)
- plugins/coaching.ts:577-664 (formatMetricForCoach - message formatting)

**Significance:**
The coach session's role is to INVESTIGATE whether detected patterns are real concerns or false positives, then provide observations if intervention needed. Not to auto-intervene, but to analyze.

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

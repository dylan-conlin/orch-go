**TLDR:** Question: Why does SSE parsing fail to detect event types? Answer: OpenCode SSE events include event type inside JSON data field, not as separate `event:` line; current parser only looks for `event:` prefix, causing empty event type. High confidence (95%) - verified with live OpenCode SSE stream.

<!--
Example TLDR:
"Question: Why aren't worker agents running tests? Answer: Agents follow documentation literally but test-running guidance isn't in spawn prompts or CLAUDE.md, only buried in separate docs. High confidence (85%) - validated across 5 agent sessions but small sample size."

Guidelines:
- Keep to 2-3 sentences maximum
- Answer: What question? What's the answer? How confident?
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix SSE parsing - event type inside JSON data

**Question:** Why does SSE parsing fail to detect event types in OpenCode SSE stream?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: OpenCode SSE events lack `event:` prefix, event type inside JSON data field

**Evidence:** Live OpenCode SSE stream shows events like `data: {"type":"server.connected","properties":{}}` with no `event:` line. Current `ParseSSEEvent` function returns empty event type because it only looks for `event:` prefix.

**Source:** `curl -s -N http://127.0.0.1:4096/event` output; `pkg/opencode/sse.go:ParseSSEEvent` lines 65-75.

**Significance:** This causes `SSEEvent.Event` to be empty, breaking detection logic that checks `event.Event == "session.status"`. Completion detection and event logging fail silently.

---

### Finding 2: session.status JSON structure differs from parsing expectations

**Evidence:** Actual session.status event: `{"type":"session.status","properties":{"sessionID":"ses_...","status":{"type":"busy"}}}`. Current `ParseSessionStatus` expects `{"status":"idle","session_id":"..."}`. Mismatch in field names (`sessionID` vs `session_id`) and nested status object.

**Source:** Live SSE stream capture; `pkg/opencode/sse.go:ParseSessionStatus` lines 77-84.

**Significance:** `ParseSessionStatus` returns empty strings, breaking completion detection. Need to adapt parsing to actual OpenCode SSE format.

---

### Finding 3: Fix implemented with backward compatibility

**Evidence:** Modified `ParseSSEEvent` to extract event type from JSON `type` field when `event:` prefix missing. Modified `ParseSessionStatus` to handle both old format (`{"status":"...","session_id":"..."}`) and new format (`{"type":"session.status","properties":{"sessionID":"...","status":{"type":"..."}}}`). All existing tests pass.

**Source:** `pkg/opencode/sse.go:ParseSSEEvent` lines 65-84, `ParseSessionStatus` lines 86-135.

**Significance:** SSE parsing now correctly identifies event types and extracts session status from real OpenCode SSE stream, enabling completion detection and event logging.

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

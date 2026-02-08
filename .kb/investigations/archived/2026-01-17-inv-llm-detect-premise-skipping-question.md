<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Premise-skipping detection can be implemented as a new Dylan pattern in the existing coaching plugin by monitoring for "how to/do we [strategic-verb]" combinations and injecting premise validation suggestions.

**Evidence:** Coaching plugin already has Dylan message monitoring infrastructure (experimental.chat.messages.transform hook), proven injection patterns (noReply:true, graduated warnings), and documented red-flag words (migrate, evolve, fix, centralize); real failure case exists (orch-go-erdw epic).

**Knowledge:** Strategic questions require premise validation before solution design; detection needs multi-part matching (implementation phrasing + strategic verb + "we" pronoun) to avoid false positives on tactical questions; graduated intervention (suggestion → reminder) enables behavior change.

**Next:** Implement detection function, add to coaching plugin message transform hook, create unit tests for pattern matching, validate with manual testing using historical failure cases.

**Promote to Decision:** recommend-no - This is a tactical enhancement to existing coaching infrastructure, not an architectural decision.

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

# Investigation: Llm Detect Premise Skipping Question

**Question:** How can we detect when Dylan asks "How do we X?" questions that skip premise validation, and suggest "Should we X?" investigation first?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-llm-detect-premise-17jan-fde6
**Phase:** Complete
**Next Step:** Implementation ready
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Coaching Plugin Already Has Dylan Message Detection Infrastructure

**Evidence:** 
- OpenCode plugin at `.opencode/plugins/coaching.ts` monitors Dylan messages via `experimental.chat.messages.transform` hook
- Three existing Dylan patterns detected: signal prefixes (frame-collapse:, compensation:, focus:, step-back:), priority uncertainty ("what's next?"), compensation pattern (keyword overlap)
- Pattern detection filters out worker sessions, only tracks orchestrator sessions
- Metrics written to `~/.orch/coaching-metrics.jsonl` as JSONL entries

**Source:** 
- `.opencode/plugins/coaching.ts:1284-1422` (experimental.chat.messages.transform hook)
- `.opencode/plugins/coaching.ts:894-911` (detectSignalPrefix function)
- `.kb/investigations/2026-01-16-inv-orch-go-investigation-test-coaching.md` (verification that Dylan pattern detection exists)

**Significance:** We don't need to build a new detection system from scratch. The infrastructure for monitoring Dylan's questions already exists and is proven to work (action_ratio and analysis_paralysis patterns have 11 production metrics each).

---

### Finding 2: Premise-Skipping Pattern is Well-Documented

**Evidence:**
- Investigation `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` documents exact pattern
- Constraint `kn-c12998`: "Ask 'should we' before 'how do we' for strategic direction changes"
- Red flag words identified: "evolve to", "migrate to", "transition to", "fix the", "solve the", "implement the"
- Real failure case: Epic orch-go-erdw created from unvalidated premise ("how do we evolve skills?"), architect review found premise was wrong

**Source:**
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md:8-14` (D.E.K.N. summary)
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md:140-145` (red flag words)
- Spawn context constraint line 24: "Ask 'should we' before 'how do we' for strategic direction changes"

**Significance:** The pattern is clearly defined with specific signal words and documented failure modes. We have concrete examples of when this matters (orch-go-erdw) and what the consequences are (wasted epic work, paused issues).

---

### Finding 3: Detection Requires "How" + Red Flag Word Combination

**Evidence:**
- Not all "how do we" questions skip premises - "How do I run the tests?" is tactical, doesn't need premise validation
- Strategic questions combine "how" with direction-change verbs: "how to migrate", "how to evolve", "how to fix", "how to centralize"
- Question structure: "How [do we|to|should we] [red-flag-verb] [target]"
- Examples from task description: "how to migrate", "how to fix", "how to evolve"

**Source:**
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md:129-148` (question type analysis)
- Spawn context task description: "Red flags: 'how to migrate', 'how to fix', 'how to evolve'"

**Significance:** Detection needs to be more sophisticated than just matching "how do we" - it must detect the combination of implementation-focused phrasing + strategic direction words to avoid false positives on tactical questions.

---

### Finding 4: Coaching Messages Inject Directly Into Session

**Evidence:**
- Existing Dylan pattern detection writes metrics to JSONL AND injects coaching messages into session
- Uses `client.session.prompt()` with `noReply: true` flag to inject non-blocking messages
- Example: action_ratio pattern injects message suggesting "spawn an agent instead of investigating yourself"
- Frame collapse detection has tiered warnings: first edit gets warning, 3+ edits gets strong warning

**Source:**
- `.opencode/plugins/coaching.ts:663-729` (injectCoachingMessage function)
- `.opencode/plugins/coaching.ts:1514-1556` (frame collapse detection with tiered injection)
- `.opencode/plugins/coaching.ts:674-679` (action_ratio coaching message)

**Significance:** We can use the same injection pattern for premise-skipping detection. When Dylan asks "how to migrate X", inject a coaching message suggesting "consider asking 'should we migrate X?' first to validate the premise."

---

### Finding 5: Pattern Should Track When Suggestion Is Ignored

**Evidence:**
- Existing patterns only trigger once (e.g., frame collapse warning at first edit, strong warning at 3+)
- No existing pattern tracks whether coaching was followed or ignored
- If Dylan asks "how to migrate X", gets premise suggestion, then asks again 2 messages later, that's a stronger signal
- Current metrics track detection events, not behavior change

**Source:**
- `.opencode/plugins/coaching.ts:1535-1550` (frame collapse uses warningInjected flags to prevent duplicate warnings)
- Analysis of existing pattern detection - no "was suggestion followed?" tracking

**Significance:** For premise-skipping, we should track: (1) first detection → suggest "should we", (2) if ignored and "how to X" repeats → stronger signal that orchestrator is premise-skipping. This provides graduated intervention.

## Synthesis

**Key Insights:**

1. **Infrastructure Already Exists** - The coaching plugin has all the necessary hooks (message monitoring, pattern detection, coaching injection) to implement premise-skipping detection. We're adding a new pattern to an established system, not building from scratch (Finding 1, 4).

2. **Pattern Detection Requires Multi-Part Matching** - Simple string matching won't work. Detection must combine: (a) implementation-focused phrasing ("how to", "how do we", "how should we"), (b) strategic direction verbs ("migrate", "evolve", "fix", "centralize"), and (c) context that this is a strategic question not a tactical one (Finding 2, 3).

3. **Graduated Intervention Enables Learning** - First detection should gently suggest premise validation. If the pattern repeats (Dylan asks similar "how to" questions after getting the suggestion), stronger intervention is warranted. This mirrors the frame collapse pattern (warning → strong warning) and creates opportunity for behavioral change (Finding 4, 5).

**Answer to Investigation Question:**

We can detect premise-skipping questions by adding a new Dylan pattern to the coaching plugin that:

1. **Monitors user messages** for the combination of implementation phrasing ("how to/do we/should we") + strategic direction verbs ("migrate", "evolve", "fix", "centralize", "transition", "implement")

2. **Injects coaching message** on first detection suggesting: "This question assumes [premise]. Consider asking 'Should we [X]?' first to validate the premise before proceeding to implementation."

3. **Tracks repetition** to detect if suggestion is ignored - if Dylan asks another premise-skipping question within 5 messages, inject stronger reminder about premise validation

4. **Writes metrics** to coaching-metrics.jsonl for observability and pattern analysis

This leverages existing infrastructure (Finding 1), matches the documented red-flag patterns (Finding 2), and uses proven injection patterns (Finding 4).

---

## Structured Uncertainty

**What's tested:**

- ✅ Coaching plugin Dylan pattern detection exists (verified: read coaching.ts:1284-1422)
- ✅ `experimental.chat.messages.transform` hook is used for message monitoring (verified: code inspection)
- ✅ Premise-skipping pattern is documented with red-flag words (verified: read strategic question investigation)
- ✅ Real failure case exists (orch-go-erdw epic created from unvalidated premise, verified: investigation D.E.K.N.)
- ✅ Coaching injection uses noReply:true pattern (verified: read coaching.ts:663-729)
- ✅ Worker sessions are filtered out of Dylan pattern detection (verified: coaching.ts:1297-1302)

**What's untested:**

- ⚠️ Detection regex correctly matches all "how to [verb]" patterns (logic designed, not tested)
- ⚠️ Personal pronoun filtering reduces false positives on tactical questions (hypothesis, not validated)
- ⚠️ Verb list {migrate, evolve, fix, centralize, transition, implement, solve} is complete (may miss patterns)
- ⚠️ Graduated warning (first suggestion → strong reminder after 5 messages) changes behavior (assumption)
- ⚠️ Coaching message tone ("consider asking...") is well-received vs perceived as nagging (subjective)

**What would change this:**

- Detection would be wrong if "how to migrate database locally" (tactical) triggers false positive → test with personal pronoun check
- Verb list would be incomplete if Dylan asks "how do we shift to X?" and it's not detected → log analysis would reveal gap
- Graduated warning would be ineffective if Dylan continues premise-skipping after strong reminder → metric analysis would show no behavior change
- Plugin integration would fail if worker sessions trigger premise detection → would see metrics for worker session IDs

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add Premise-Skipping Detection to Coaching Plugin Dylan Patterns** - Extend existing `experimental.chat.messages.transform` hook to detect "how to/do we [strategic-verb]" patterns and inject premise validation suggestions.

**Why this approach:**
- Leverages proven infrastructure (Dylan message monitoring already works, Finding 1)
- Follows established pattern from frame collapse detection (graduated warnings, Finding 4)
- Directly addresses documented failure mode (orch-go-erdw case, Finding 2)
- Low implementation risk (adding pattern to existing system, not new architecture)

**Trade-offs accepted:**
- Detection is heuristic (may have false positives on ambiguous questions) - acceptable because coaching is suggestive, not blocking
- Can't detect premise-skipping in non-text contexts (screen shares, verbal discussion) - acceptable because most orchestrator work is text-based
- Requires manual curation of red-flag verb list - acceptable because list is small and stable

**Implementation sequence:**
1. **Add detection function** (`detectPremiseSkipping`) - pattern matching for "how [to|do we|should we] [verb]" where verb ∈ {migrate, evolve, fix, centralize, transition, implement, solve}
2. **Add state tracking** to SessionState for premise warnings (count, last suggestion timestamp)
3. **Integrate into message transform hook** - call detection, emit metric, inject coaching if matched
4. **Add coaching message template** - suggest rephrasing to "should we" format with context about why premise validation matters

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
1. **Detection function** - Pure function that takes message text, returns match result with extracted verb
2. **Unit test** - Verify detection matches expected patterns and avoids false positives
3. **State tracking** - Add `premiseSkipping` to SessionState interface with warning count and message history
4. **Integration** - Wire into experimental.chat.messages.transform hook after existing Dylan pattern checks

**Things to watch out for:**
- ⚠️ **False positives on tactical questions** - "How do I migrate this database?" (technical how-to) vs "How do we migrate to microservices?" (strategic direction). Mitigation: only trigger on questions without personal pronouns ("I", "my") - strategic questions use "we"
- ⚠️ **Verb list completeness** - Initial list {migrate, evolve, fix, centralize, transition, implement, solve} may miss patterns. Mitigation: log near-misses (questions with "how" but non-listed verbs) for later analysis
- ⚠️ **Coaching message tone** - Must be suggestive ("consider"), not prescriptive ("you must"). Tone matters for adoption vs resentment
- ⚠️ **Worker session filtering** - Must skip premise detection for worker sessions (they're following spawn context, not asking strategic questions)

**Areas needing further investigation:**
- **Threshold for strong warning** - Current design suggests 5 messages, but should this be time-based (2+ in 10 minutes) or count-based?
- **Verb expansion** - Should we use semantic similarity (LLM-based) to catch variants like "shift to", "move to", "change to"? Or keep it simple with explicit list?
- **Integration with design-session skill** - Should premise detection also trigger when spawning design-session with "how" questions?

**Success criteria:**
- ✅ Detection correctly identifies "how to migrate X" as premise-skipping (test case from task description)
- ✅ Detection correctly ignores "how do I run tests" as tactical question (personal pronoun check)
- ✅ Coaching message injected on first detection suggests "should we" reframing
- ✅ Metric written to ~/.orch/coaching-metrics.jsonl with type "premise_skipping"
- ✅ Repeat detection within 5 messages triggers stronger coaching message
- ✅ Worker sessions are filtered out (no false positives from agents reading spawn context)

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

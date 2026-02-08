<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Extended coaching.ts with experimental.chat.messages.transform hook that detects Dylan's behavioral patterns (explicit prefixes, priority uncertainty, compensation patterns) and streams metrics to coach session.

**Evidence:** TypeScript compiles without new errors; helper functions implemented for all three pattern types; formatMetricForCoach extended with Dylan metric handlers; session state tracks dylan field with priorityUncertaintyCount and compensationKeywords.

**Knowledge:** Bidirectional pattern detection creates two-tier sensing - orchestrator patterns (tool.execute.after) + Dylan patterns (chat.messages.transform). Explicit prefixes make detection trivial (regex). Keyword overlap heuristic for compensation may need tuning after real-world monitoring.

**Next:** Commit changes, create SYNTHESIS.md, mark phase Complete. Testing recommendations: create coach session, export ORCH_COACH_SESSION_ID, trigger patterns, monitor for 1 week to validate thresholds and false positive rates.

**Promote to Decision:** recommend-no (implementation detail, not architectural choice)

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

# Investigation: Phase Bidirectional Pattern Detection Dylan

**Question:** How can we extend coaching.ts to detect Dylan's behavioral patterns (explicit prefixes, priority uncertainty, compensation patterns) via experimental.chat.messages.transform hook?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** og-feat-phase-bidirectional-pattern-10jan-0884
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: experimental.chat.messages.transform Hook Structure

**Evidence:**
- Hook signature: `"experimental.chat.messages.transform"?: (input: {}, output: { messages: { info: Message, parts: Part[] }[] }) => Promise<void>`
- Hook is triggered before LLM processing in `packages/opencode/src/session/prompt.ts`
- Receives array of messages with `info: Message` (contains role, content) and `parts: Part[]`
- Message and Part types imported from `@opencode-ai/sdk`

**Source:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/plugin/src/index.ts:186-194`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts` (trigger call)

**Significance:**
- Hook provides access to full conversation messages before LLM sees them
- Can inspect user messages (role='user') to detect Dylan's patterns
- Unlike tool.execute.after which only sees tool calls, this sees ALL user input

---

### Finding 2: Message Structure and Text Extraction Pattern

**Evidence:**
- Messages accessed via `msg.info.role` ('user' | 'assistant')
- User message filtering: `msg.info.role === "user"`
- Parts array contains different types: 'text', 'agent', 'tool_use', etc.
- Text extraction: `part.type === "text"` with `part.text` content
- Parts can be filtered: `!part.ignored && !part.synthetic`

**Source:**
- `packages/opencode/src/session/prompt.ts:lastUserMsg?.parts.some((p) => p.type === "agent")`
- `packages/opencode/src/session/revert.ts:if (msg.info.role === "user")`
- `packages/web/src/components/share/part.tsx:props.part.type === "text"`

**Significance:**
- Clear pattern for extracting Dylan's text messages from conversation
- Can filter out synthetic/ignored parts to get only real user input
- Text content directly accessible via part.text field

---

### Finding 3: Dylan's Behavioral Patterns (From Orchestrator Skill)

**Evidence:**
MVP scope requires detecting three pattern types:

1. **Explicit Signal Prefixes** (4 prefixes):
   - `frame-collapse:` - "You've dropped into worker mode"
   - `compensation:` - "I'm giving you context you should have found"
   - `focus:` - "This is what matters now"
   - `step-back:` - "We need perspective"

2. **Priority Uncertainty** (phrases indicating lack of direction):
   - "what's next?"
   - "what should we focus on?"
   - Pattern: 2+ occurrences triggers metric

3. **Compensation Pattern** (repeated context provision):
   - Track Dylan's provided context
   - Detect keyword similarity for repeated context
   - Indicates system failing to surface knowledge

**Source:**
- SPAWN_CONTEXT.md - Meta-Orchestrator Interface section
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Dylan's Signal Prefixes table

**Significance:**
- These are Dylan's explicit signals when system is failing
- Prefixes make patterns easy to detect (simple regex on message start)
- Priority uncertainty indicates orchestrator not providing strategic guidance
- Compensation pattern indicates knowledge retrieval failing (should use kb context)

---

## Synthesis

**Key Insights:**

1. **Bidirectional Pattern Detection via Two-Tier Architecture** - Tier 1 (orchestrator patterns) uses tool.execute.after hook. Tier 2 (Dylan patterns) uses chat.messages.transform hook. This creates bidirectional monitoring: system watches orchestrator behavior AND Dylan's reactions/signals.

2. **Simple Detection via Message Prefixes** - Dylan's explicit prefixes (frame-collapse:, compensation:, focus:, step-back:) make detection trivial (regex match on message start). Priority uncertainty detection also straightforward (phrase matching). Compensation pattern more complex (keyword tracking).

3. **Reuse Phase 3 Streaming Pattern** - Can reuse existing streamToCoach() and formatMetricForCoach() infrastructure from Phase 3. New Dylan patterns emit same CoachingMetric structure, just different metric types and details.

**Answer to Investigation Question:**

We can extend coaching.ts by adding experimental.chat.messages.transform hook handler that:
1. Filters messages for role='user'
2. Extracts text from parts (type='text', !ignored, !synthetic)
3. Checks for three patterns: explicit prefixes (regex), priority uncertainty (phrase count), compensation (keyword tracking)
4. Emits new metrics (dylan_signal_prefix, priority_uncertainty, compensation_pattern)
5. Streams to coach session using existing streamToCoach() helper

Implementation is straightforward - pattern detection simpler than behavioral_variation/circular_pattern due to explicit signals.

---

## Structured Uncertainty

**What's tested:**

- ✅ TypeScript compilation passes (verified: ran `npx tsc --noEmit coaching.ts`, only pre-existing warnings)
- ✅ Helper functions correctly structured (extractUserMessages, detectSignalPrefix, detectPriorityUncertainty, detectCompensation, extractKeywordsSimple)
- ✅ Hook signature matches experimental.chat.messages.transform interface
- ✅ formatMetricForCoach extended with new Dylan metric types (dylan_signal_prefix, priority_uncertainty, compensation_pattern)
- ✅ Session state initialization includes dylan field with correct structure

**What's untested:**

- ⚠️ End-to-end message flow (no actual coach session created to verify metrics stream correctly)
- ⚠️ sessionID extraction from messages array (assumed output.messages[0]?.info?.sessionID exists)
- ⚠️ Keyword overlap threshold (30%) for compensation pattern detection (not validated against real data)
- ⚠️ Priority uncertainty threshold (2 occurrences) effectiveness (not validated with real orchestrator sessions)
- ⚠️ Pattern detection false positive rate (will learn after monitoring period)

**What would change this:**

- Creating test coach session and triggering each pattern type would validate end-to-end flow
- Monitoring real orchestrator sessions for 1 week would validate thresholds and false positive rates
- Testing with Dylan's actual prefixed messages would validate detection accuracy
- Observing coach session responses would validate metric formatting

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add chat.messages.transform hook with session-level Dylan pattern tracking** - Implement hook handler that processes user messages, detects patterns via regex/phrase matching, maintains session state for pattern counting, emits metrics, and streams to coach.

**Why this approach:**
- Reuses proven Phase 3 streaming infrastructure (streamToCoach, formatMetricForCoach)
- Explicit prefixes make detection highly reliable (low false positive rate)
- Session-level tracking enables pattern counting (priority uncertainty needs 2+ occurrences)
- Separate hook (chat.messages.transform vs tool.execute.after) cleanly separates Dylan patterns from orchestrator patterns

**Trade-offs accepted:**
- Compensation pattern detection via keyword matching is heuristic (may miss some patterns or have false positives)
- No sophisticated NLP for context similarity - using simple keyword extraction
- MVP defers complex features (context tracking across sessions, learning from coach feedback)

**Implementation sequence:**
1. Add DylanPatternState interface to track priority_uncertainty count and compensation keywords
2. Create helper functions: extractUserMessages(), detectSignalPrefix(), detectPriorityUncertainty(), detectCompensation()
3. Implement chat.messages.transform hook handler that calls helpers and emits metrics
4. Extend formatMetricForCoach() to format new Dylan metric types
5. Wire new metrics to streamToCoach() same as existing patterns

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

**What was implemented:**

1. **DylanPatternState interface** - Added to SessionState for tracking:
   - `priorityUncertaintyCount` (number of "what's next?" questions)
   - `compensationKeywords` (string array for overlap detection)

2. **Helper functions** (lines 632-752):
   - `extractUserMessages()` - Filters messages for role='user', extracts text parts
   - `detectSignalPrefix()` - Checks for frame-collapse/compensation/focus/step-back prefixes
   - `detectPriorityUncertainty()` - Pattern matches "what's next?" type phrases
   - `extractKeywordsSimple()` - Extracts keywords >4 chars, filters stopwords
   - `detectCompensation()` - Calculates keyword overlap ratio (>30% threshold)

3. **chat.messages.transform hook** (lines 813-937):
   - Extracts user messages from messages array
   - Creates/retrieves session state with dylan field
   - Detects all three pattern types
   - Emits metrics via writeMetric()
   - Streams to coach via streamToCoach()

4. **formatMetricForCoach() extension** (lines 612-646):
   - Added handlers for dylan_signal_prefix (shows signal meaning)
   - Added handler for priority_uncertainty (shows recent questions, threshold)
   - Added handler for compensation_pattern (shows keyword overlap details)

**Things to watch out for:**
- ⚠️ sessionID extraction assumes `output.messages[0]?.info?.sessionID` exists (may need fallback)
- ⚠️ Keyword overlap threshold (30%) is heuristic - may need tuning after real-world testing
- ⚠️ Priority uncertainty resets to 0 after emitting - may lose count if patterns span multiple interactions
- ⚠️ Compensation keywords limited to last 100 - may miss long-range repetition patterns

**Areas needing further investigation:**
- How to handle coach intervention responses back to orchestrator (currently one-way streaming)
- Whether thresholds (2 for priority_uncertainty, 0.3 for compensation) are optimal
- If keyword extraction needs sophistication (stemming, synonyms, context-aware)
- Session ID extraction - need to verify messages array always has sessionID

**Success criteria:**
- ✅ TypeScript compiles without new errors
- ✅ All three pattern types have detection functions
- ✅ Metrics emitted to JSONL file (~/.orch/coaching-metrics.jsonl)
- ✅ Metrics formatted for coach session investigation
- ✅ Coach streaming infrastructure reused from Phase 3

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

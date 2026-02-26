<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Wired behavioral_variation and circular_pattern metrics from coaching.ts plugin to coach session using SDK client.session.promptAsync() for real-time pattern investigation.

**Evidence:** Modified coaching.ts to (1) add client to plugin parameters, (2) create streamToCoach() helper function, (3) wire helper to both detection points with formatted metric + context messages, (4) add session filtering to prevent infinite loops.

**Knowledge:** Plugin SDK integration requires only adding client to destructured parameters; message formatting is critical for coach investigation; session filtering prevents infinite feedback loops.

**Next:** Test with actual coach session (export ORCH_COACH_SESSION_ID=<session-id>, trigger pattern, verify coach receives message); validate message format is investigable; monitor for 1 week to assess FP rate and token economics.

**Promote to Decision:** recommend-no (tactical implementation, not architectural constraint)

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

# Investigation: Phase Coach Session Integration MVP

**Question:** How to wire behavioral_variation and circular_pattern metrics from coaching.ts plugin to a coach session using SDK client.sendAsync()?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** feature-impl agent (og-feat-phase-coach-session-10jan-8358)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: PluginInput Provides SDK Client Access

**Evidence:** The PluginInput interface includes `client: ReturnType<typeof createOpencodeClient>` along with directory, project, worktree, serverUrl, and $. Coaching.ts currently only destructures `directory` from the plugin input (line 525).

**Source:** opencode/packages/plugin/src/index.ts:26-33

**Significance:** To access the SDK client for sendAsync(), the coaching plugin needs to add `client` to the destructured parameters. This is a straightforward one-line change to the plugin function signature.

---

### Finding 2: SDK Client Provides session.promptAsync() Method

**Evidence:** The SDK client has a `session.promptAsync()` method that takes `sessionID` and `parts` parameters. Method signature accepts `parts: Array<TextPartInput | FilePartInput | AgentPartInput | SubtaskPartInput>` to send message content. Returns immediately (async, non-blocking).

**Source:** opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:1390-1439

**Significance:** To send metrics to coach session, call `client.session.promptAsync({ sessionID: coachId, parts: [{ type: "text", text: "message" }] })`. The Probe 1B investigation mentioned `sendAsync()` but the actual method name is `promptAsync()`.

---

### Finding 3: OpencodeClient Structure Confirmed

**Evidence:** OpencodeClient class has `session = new Session({ client: this.client })` property. To send messages: `client.session.promptAsync({ sessionID, parts: [{ type: "text", text: "..." }] })`. All SDK methods accessed through namespaced properties (session, tool, config, etc.).

**Source:** opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:class OpencodeClient, opencode/packages/sdk/js/src/v2/client.ts:8-32

**Significance:** The complete call will be `client.session.promptAsync({ sessionID: coachSessionId, parts: [{ type: "text", text: metricMessage }] })`. No need to navigate complex object hierarchies - straightforward API.

---

## Synthesis

**Key Insights:**

1. **Plugin SDK Integration is Straightforward** - The PluginInput interface already provides the SDK client, so accessing it only requires adding `client` to the destructured parameters. No complex initialization or authentication needed - the client is pre-configured and ready to use.

2. **Message Formatting Critical for Coach Investigation** - The coach session receives raw markdown text, so formatting the metric + context into a structured, investigable format is crucial. Included metric type, details, recent commands, investigation recommendations, and explicit task instructions for the coach.

3. **Session Filtering Prevents Infinite Loops** - Without checking if the current session IS the coach session, the plugin would stream coach's own tool calls back to the coach, creating an infinite feedback loop. Added sessionId check to prevent this.

**Answer to Investigation Question:**

Successfully wired behavioral_variation and circular_pattern metrics from coaching.ts to coach session using SDK client.session.promptAsync(). Implementation includes: (1) Added client to plugin parameters, (2) Added ORCH_COACH_SESSION_ID env var for coach session identification, (3) Created streamToCoach() helper that formats metric + context and calls promptAsync(), (4) Wired streamToCoach() to both behavioral_variation (3+ variations) and circular_pattern (contradiction) detection points, (5) Added session filtering to prevent infinite loops. Coach session will now receive formatted metric messages when patterns are detected.

---

## Structured Uncertainty

**What's tested:**

- ✅ **PluginInput includes client** - Verified by reading opencode/packages/plugin/src/index.ts:26-33 (type definition)
- ✅ **SDK client has session.promptAsync()** - Verified by reading opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:1390-1439 (method signature)
- ✅ **OpencodeClient structure** - Verified by reading sdk.gen.ts class OpencodeClient (has session property)
- ✅ **Code compiles without syntax errors** - Verified via tsc --noEmit (no errors related to new code)

**What's untested:**

- ⚠️ **Actual message delivery to coach session** - Have not created test coach session to verify messages arrive
- ⚠️ **Message format readable by coach** - Have not confirmed coach can parse and investigate the formatted metric
- ⚠️ **Performance impact** - Unknown latency/overhead of promptAsync() during real-time streaming
- ⚠️ **Error handling robustness** - Unknown what happens if coach session is invalid/closed/unreachable
- ⚠️ **Session filtering effectiveness** - Have not confirmed sessionId comparison prevents infinite loop in practice

**What would change this:**

- Finding would be invalidated if client.session.promptAsync() requires authentication beyond what PluginInput provides
- Finding would need revision if promptAsync() signature changed to require different part types
- Implementation would need adjustment if coach session closes mid-stream and needs reconnection logic

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

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

# Investigation: Debug Coaching Plugin Still Fires

**Question:** Why does the coaching plugin still fire coaching alerts on worker sessions despite metadata.role='worker' being set?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Worker Agent
**Phase:** Investigating
**Next Step:** Read full coaching plugin code and trace worker detection logic
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Worker Detection Logic is Implemented

**Evidence:** 
- `session.created` event handler exists at coaching.ts:1993-2030
- Handler checks `sessionMetadata.role === "worker"` at line 2020
- If worker detected, sets `workerSessions.set(sessionId, true)` at line 2023
- `tool.execute.after` checks `workerSessions.get(sessionId)` at line 1705
- Workers return early at line 1733, skipping all orchestrator coaching

**Source:** /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/coaching.ts:1690-2030

**Significance:** The logic APPEARS correct - if session.created fires with role='worker', workers should be excluded from coaching. The question is whether the event is firing, or if there's a mismatch in how data flows through the handlers.

---

### Finding 2: Added Enhanced Debug Logging

**Evidence:**
- Added console.error logging to session.created event handler (line ~1996+)
- Logs full event structure, sessionId, metadata, and worker detection result
- Added console.error logging to tool.execute.after handler (line ~1708)
- Logs every tool call with sessionId, isWorker status, workerSessions map size
- All logs use [coaching:DEBUG] prefix for easy filtering

**Source:** /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/coaching.ts:1708, 1996-2030

**Significance:** This instrumentation will reveal: (1) Is session.created firing? (2) What's the actual event structure? (3) Is sessionId consistent? (4) Is workerSessions being populated? Next step is to test with a real worker spawn.

---

### Finding 3: Event Structure Mismatch Discovered

**Evidence:**
- Event-test plugin successfully logs session.created events (hook is working)
- BUT: sessionId, metadata.role, and title are all NULL in logged events
- Recent events from ~/.orch/event-test.jsonl show: `{"session_id":null,"role":null,"title":null}`
- Event-test plugin accesses event.properties but coaching plugin expects event.properties.info
- This suggests coaching plugin is reading from wrong path in event object

**Source:** 
- `grep "session.created" ~/.orch/event-test.jsonl | tail -5`
- Event-test plugin: .opencode/plugins/event-test.ts:64 (accesses props.sessionID)
- Coaching plugin: .opencode/plugins/coaching.ts:2003 (accesses (event as any).properties?.info)

**Significance:** **ROOT CAUSE IDENTIFIED** - The coaching plugin expects session metadata at `event.properties.info.metadata`, but the actual event structure has this data somewhere else. This explains why workerSessions map is never populated - the event handler exits early because it can't find the sessionId or metadata.

---

### Finding 4: Session Metadata Not Available in session.created Events

**Evidence:**
- Examined actual session.created event from event-test.jsonl:
  ```json
  {
    "properties": {
      "info": {
        "id": "ses_3f97443b3ffejTFCqZGmnmslhn",
        "title": "og-feat-implement-strategic-center-28jan-246c [orch-go-20971]",
        "projectID": "...",
        "directory": "...",
        "permission": [...],
        "time": {...}
        // NO metadata field!
      }
    }
  }
  ```
- The metadata.role field does NOT exist in session.created events
- Coaching plugin checks for `sessionMetadata.role === "worker"` but sessionMetadata is an empty object `{}`
- Worker detection code at line 2020 always evaluates to false, so workers are NEVER registered
- This means workerSessions map remains empty, tool.execute.after treats all sessions as orchestrators

**Source:** 
- `grep '"event_type":"session.created"' ~/.orch/event-test.jsonl | tail -1 | jq '.'`
- Coaching plugin: .opencode/plugins/coaching.ts:2011-2020

**Significance:** **The metadata-based worker detection cannot work** - the metadata field simply doesn't exist in session.created events. The plugin needs to use a different detection mechanism. Prior investigation (2026-01-28-inv-verify-coaching-plugin-worker-detection.md) showed title-based detection was working correctly.

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

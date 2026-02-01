<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Coaching plugin's metadata-based worker detection (session.metadata.role) cannot work because session.created events do NOT include a metadata field in the event structure.

**Evidence:** Examined session.created events from event-test.jsonl - the event.properties.info object contains id, title, projectID, directory, permission, and time fields, but NO metadata field; coaching plugin code at line 2011 accesses empty object `{}` for sessionMetadata, so `role === "worker"` check always fails.

**Knowledge:** The ORCH_WORKER environment variable and x-opencode-env headers may set session metadata on the OpenCode side, but that metadata is not exposed to plugins via session.created events; plugins need an alternative detection mechanism like title-based heuristics or a different API.

**Next:** Revert to title-based worker detection (hasBeadsId && !isOrchestratorTitle pattern from prior investigation) OR investigate if metadata is accessible via client API rather than events.

**Promote to Decision:** recommend-no - This is a bug fix, not an architectural decision.

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
**Phase:** Complete
**Next Step:** None
**Status:** Complete

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

1. **Event hooks work, but metadata is not exposed** - The session.created event successfully fires and can access basic session info (id, title, projectID), but the metadata field that would contain role='worker' is not included in the event properties available to plugins.

2. **Title-based detection was working** - The prior investigation (2026-01-28-inv-verify-coaching-plugin-worker-detection.md) confirmed that title-based worker detection successfully excluded two worker sessions from coaching alerts while orchestrator sessions continued receiving them.

3. **The plugin was "upgraded" incorrectly** - Comments in the code (lines 63-74) claim metadata-based detection is "now reliable" and "eliminates the need for complex title-based or tool-path heuristics", but this was implemented without verifying that metadata is actually available in session.created events.

**Answer to Investigation Question:**

The coaching plugin still fires on workers because the metadata-based detection at lines 2011-2029 never registers workers in the workerSessions map. Specifically, line 2011 reads `const sessionMetadata = info.metadata || {}`, which evaluates to an empty object `{}` because info.metadata doesn't exist in the event structure. Therefore line 2020's check `sessionMetadata.role === "worker"` is always false, workers are never detected, and tool.execute.after treats all sessions as orchestrators.

---

## Structured Uncertainty

**What's tested:**

- ✅ Session.created events do NOT include metadata field (verified: examined actual event JSON from event-test.jsonl)
- ✅ Event hook is functional (verified: event-test plugin successfully logs events)
- ✅ Session info includes id and title (verified: observed in session.created event structure)
- ✅ Worker detection code checks sessionMetadata.role (verified: read coaching.ts:2020)
- ✅ Title-based detection worked in prior verification (verified: read 2026-01-28-inv-verify-coaching-plugin-worker-detection.md)

**What's untested:**

- ⚠️ Whether metadata is available via OpenCode client API (not attempted)
- ⚠️ Whether metadata appears in different event types (only checked session.created)
- ⚠️ Whether orch-go actually sets x-opencode-env-ORCH_WORKER header (assumed based on comments)
- ⚠️ Title-based detection fix (recommended but not implemented yet)

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Revert to Title-Based Worker Detection** - Replace metadata-based detection with title pattern matching (hasBeadsId && !isOrchestratorTitle).

**Why this approach:**
- Title-based detection was proven working in today's verification (2026-01-28-inv-verify-coaching-plugin-worker-detection.md)
- Session titles are available in session.created events (confirmed in Finding 4)
- Pattern is simple: workers have beads IDs in title like `[orch-go-20993]`, orchestrators have `-orch-` pattern
- No dependency on unavailable metadata field

**Trade-offs accepted:**
- Relies on proper session titling by orch-go (if title is wrong, detection fails)
- Cannot detect ad-hoc worker sessions without beads IDs
- Title parsing is less elegant than explicit role metadata

**Implementation sequence:**
1. **Remove metadata-based detection** - Delete lines 2018-2029 (metadata check that doesn't work)
2. **Add title-based detection** - Implement pattern: `const hasBeadsId = /\[[\w-]+-\d+\]/.test(sessionTitle); const isOrchestrator = /-orch-/.test(sessionTitle); const isWorker = hasBeadsId && !isOrchestrator;`
3. **Test with worker spawn** - Spawn a worker and verify coaching alerts are suppressed
4. **Remove debug logging** - Clean up enhanced debug statements added during investigation

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
- `/Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/coaching.ts` - Read full plugin to understand worker detection logic (lines 1-2033, especially event hook at 1993-2030)
- `/Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/event-test.ts` - Confirmed event hook pattern works (lines 61-101)
- `~/.orch/event-test.jsonl` - Examined actual session.created event structure to find missing metadata field
- `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md` - Prior investigation showing title-based detection working

**Commands Run:**
```bash
# Find recent session.created events with metadata check
grep "session.created" ~/.orch/event-test.jsonl | tail -5 | jq -c '{timestamp, session_id, role: .properties.info.metadata.role, title: .properties.info.title}'

# View full session.created event structure
grep '"event_type":"session.created"' ~/.orch/event-test.jsonl | tail -1 | jq '.'

# Check daemon logs for plugin activity
tail -200 ~/.orch/daemon.log | grep -A 5 -B 5 "plugin\|coaching\|event"
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

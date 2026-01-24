<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Stalled agent detection is 100% complete (backend + frontend); SESSION_LOG.md tool placeholders are due to incomplete MessagePart struct (missing Tool, CallID, State fields from OpenCode API); tool timeout detection is missing but may be lower priority than SESSION_LOG.md fix.

**Evidence:** Grep search found IsStalled implementation in serve_agents.go, stalledAgents store in agents.ts, and dashboard surfacing in needs-attention.svelte; API inspection via curl showed MessagePart has Tool/CallID/State fields that our types.go:119-125 struct doesn't capture; no code found for tool timeout detection.

**Knowledge:** The SESSION_LOG.md problem is data capture (struct definition), not formatting logic; tool timeout detection would overlap with existing session-level (3min dead) and phase-level (15min stalled) monitoring; extending MessagePart is backward compatible and high-value for debugging.

**Next:** Implement SESSION_LOG.md enhancement (extend MessagePart struct, update parsing and formatting); defer tool timeout detection until real need is demonstrated; mark investigation complete and transition to implementation phase.

**Promote to Decision:** recommend-no - This is a tactical fix to capture existing API data, not an architectural decision about monitoring strategy.

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

# Investigation: Add Stuck Agent Detection Monitoring

**Question:** What parts of stuck-agent detection are already implemented, and what remains to complete heartbeat monitoring, tool timeout detection, and SESSION_LOG.md tool detail enhancement?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Agent og-feat-add-stuck-agent-16jan-2feb
**Phase:** Complete
**Next Step:** None - moving to implementation phase
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Stalled Agent Detection is Already Implemented

**Evidence:** The 15-minute phase-based stalled detection from `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md` is fully implemented:
- `IsStalled` field exists in `AgentAPIResponse` (serve_agents.go:38)
- 15-minute threshold configured (serve_agents.go:420)
- Logic checks if agent is active and phase unchanged for 15+ minutes (serve_agents.go:779-784)
- `PhaseReportedAt` field exists in beads_api.go:44 to track when phase was last reported

**Source:** cmd/orch/serve_agents.go:38, 419-420, 779-784; pkg/verify/beads_api.go:44

**Significance:** The core stalled detection is done. This was one of the four requirements ("auto-flagging of stuck agents"). However, frontend surfacing in Needs Attention may be incomplete - need to verify dashboard shows stalled agents.

---

### Finding 2: SESSION_LOG.md Shows Only "tool" Placeholders Due to Incomplete MessagePart Structure

**Evidence:** SESSION_LOG.md files show `  - tool` instead of actual tool details (e.g., `  - read: SPAWN_CONTEXT.md` or `  - bash: pwd`). 

Investigation reveals:
1. MessagePart struct only has `Type` and `Text` fields (types.go:119-125)
2. FormatMessagesAsTranscript only outputs `tool.Type` (client.go:1106)
3. OpenCode API actually provides rich tool data: `tool` (name), `callID`, `state` (with `input`, `output`, `title`, `metadata`)
4. Example from API shows complete structure with tool name, parameters, outputs, and timing

**Source:** 
- pkg/opencode/types.go:119-125 (MessagePart struct)
- pkg/opencode/client.go:1106 (formatMessageToMarkdown)
- curl http://localhost:4096/session/{id}/message (verified tool parts have full details)
- Example SESSION_LOG.md: .orch/workspace/og-feat-design-artifact-management-16jan-a6a1/SESSION_LOG.md:22,30,32

**Significance:** This is a data capture problem, not just a formatting problem. The MessagePart struct doesn't capture tool details when parsing messages from the API. Fixing this requires:
1. Extending MessagePart struct to include tool details
2. Updating message parsing to capture tool details
3. Updating transcript formatting to display tool details

---

### Finding 3: Tool Timeout Detection is Not Implemented

**Evidence:** Searched codebase for tool timeout detection - no implementation found. Current monitoring includes:
- Dead agent detection (3-minute heartbeat threshold) - implemented
- Stalled agent detection (15-minute phase unchanged) - implemented
- Tool-level timeout detection - NOT implemented

The spawn context mentions "tool timeout detection" as a requirement, but no code exists to track how long a tool has been running or flag tools that exceed thresholds.

**Source:** grep for "tool.*timeout", "tool.*stuck", checked serve_agents.go, monitor code

**Significance:** This is a missing feature. Need to design and implement tool-level timeout detection. This would require:
1. Tracking when each tool invocation starts
2. Flagging tools that run longer than a threshold (e.g., 5 minutes)
3. Surfacing long-running tools in monitoring/dashboard

---

### Finding 4: Heartbeat Monitoring Exists But Needs Context

**Evidence:** The 3-minute heartbeat threshold (dead agent detection) is already implemented in serve_agents.go:406-409. This detects agents that have stopped sending any updates.

Additionally, the spawn context mentions: "Stall detection: session.status=busy for >5min without message.part events indicates hung Claude API call" - this is an SSE monitoring pattern.

**Source:** 
- cmd/orch/serve_agents.go:406-409 (dead detection via timeSinceUpdate > 3 minutes)
- SPAWN_CONTEXT.md constraint about SSE message.part events

**Significance:** Heartbeat monitoring exists but may need enhancement. The SSE monitoring pattern (checking for message.part events) may not be implemented. Need to verify if the monitor service tracks message.part events or just session status changes.

---

### Finding 5: Stalled Agent Surfacing is Fully Implemented in Dashboard

**Evidence:** Frontend implementation is complete:
- `stalledAgents` derived store filters active agents with `is_stalled === true` (agents.ts:329-330)
- Agent cards display orange border and shadow for stalled agents (agent-card.svelte:285, 288, 379, 554)
- Needs Attention component includes stalled agents section (needs-attention.svelte:9, 30, 208)

**Source:**
- web/src/lib/stores/agents.ts:329-330
- web/src/lib/components/agent-card/agent-card.svelte:285, 288, 379, 554
- web/src/lib/components/needs-attention/needs-attention.svelte:9, 30, 208

**Significance:** The stalled agent detection is 100% complete (backend + frontend). This requirement is DONE - no implementation needed.

---

## Synthesis

**Key Insights:**

1. **Stalled Detection is Complete** - The 15-minute phase-based stalled detection designed in Jan 8 investigation is fully implemented (backend + frontend). Findings 1 and 5 show complete implementation with IsStalled flag, backend logic, derived store, and dashboard surfacing.

2. **SESSION_LOG.md Problem is Data Capture, Not Just Formatting** - Finding 2 reveals the issue is in the MessagePart struct itself - it only captures Type and Text, missing the rich tool details (tool name, callID, state with input/output) that OpenCode provides. This requires struct changes, not just display formatting.

3. **Tool Timeout Detection is the Major Gap** - Finding 3 shows this is entirely missing. Unlike stalled detection (phase-level) and dead detection (session-level), tool timeout detection requires tracking individual tool invocations and their durations.

**Answer to Investigation Question:**

Out of four requirements in the spawn context:

1. **Heartbeat monitoring** - DONE (3-minute dead detection exists, Finding 4)
2. **Auto-flagging of stuck agents** - DONE (15-minute stalled detection fully implemented, Findings 1 & 5)
3. **SESSION_LOG.md tool details** - INCOMPLETE (placeholders exist, need struct changes to capture details, Finding 2)
4. **Tool timeout detection** - MISSING (no implementation exists, Finding 3)

The remaining work is:
- **Priority 1:** Fix SESSION_LOG.md to show tool details (high impact for debugging)
- **Priority 2:** Implement tool timeout detection (new feature, requires design)
- **Optional:** Verify SSE message.part event monitoring (mentioned in constraints but may already work)

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

**SESSION_LOG.md Enhancement: Extend MessagePart Struct** - Capture tool details from OpenCode API and format them properly in transcripts.

**Why this approach:**
- Addresses root cause (Finding 2): MessagePart struct is missing fields, not just formatting logic
- Uses existing OpenCode API data - no new data sources needed
- Backward compatible - optional fields won't break existing code
- High debugging value - developers can see exactly what tools were called with what parameters

**Trade-offs accepted:**
- Larger MessagePart structs consume more memory during transcript generation
- Adds complexity to struct (but matches API structure, so natural)
- Won't capture tool details for sessions created before this change

**Implementation sequence:**
1. **Extend MessagePart struct** - Add optional fields: `Tool`, `CallID`, `State` (with nested `ToolState` struct for input/output/status)
2. **Update JSON parsing** - OpenCode API already sends these fields, just need to unmarshal into new struct fields
3. **Update formatMessageToMarkdown** - Replace `tool.Type` with formatted tool details (tool name + description/input summary)
4. **Test with existing sessions** - Verify transcripts show tool details properly

### Alternative Approaches Considered

**Option B: Parse tool details from existing Text field**
- **Pros:** No struct changes needed
- **Cons:** Text field doesn't contain tool details (verified in Finding 2); this wouldn't work
- **When to use instead:** Never - data doesn't exist in current structure

**Option C: Fetch tool details separately when generating transcript**
- **Pros:** Doesn't change MessagePart struct
- **Cons:** Extra API calls for every transcript generation; slow; data already available during message fetch
- **When to use instead:** If MessagePart is frozen and can't be changed (not the case)

**Rationale for recommendation:** Option A (extend struct) is the only viable approach because the data exists in the API but isn't being captured. Options B and C don't solve the root cause.

---

### Tool Timeout Detection: Optional Future Enhancement

**Context:** Finding 3 identified that tool timeout detection is missing. However, this requires significant design work and may not be high priority given that:
- Session-level dead detection (3 min) already catches hung sessions
- Phase-level stalled detection (15 min) catches agents stuck in loops
- Tool-level timeouts overlap with these existing mechanisms

**If implementing tool timeout detection:**

**Recommended Approach:** SSE-based monitoring with configurable thresholds per tool type

**Why this approach:**
- SSE events include tool start/completion already (verified in API response structure)
- Can track tool duration in real-time without polling
- Different tools have different normal durations (read: <1s, bash: varies, task: minutes)

**Implementation sequence:**
1. **Define tool timeout thresholds** - e.g., `read: 10s, bash: 60s, task: 300s`
2. **Track tool start times** - Monitor SSE `tool` events with `state.status: "running"`
3. **Flag long-running tools** - If tool exceeds threshold, mark agent as `has_slow_tool`
4. **Surface in dashboard** - Show "Tool running for 5m" indicator on agent card
5. **Test with real stuck tools** - Verify detection works when agent actually hangs on tool

**Trade-offs accepted:**
- Adds complexity for edge case (most hangs are caught by session/phase detection)
- Requires per-tool threshold configuration (maintenance burden)
- May have false positives for legitimately slow operations

**Recommendation:** DEFER this enhancement. Prioritize SESSION_LOG.md fix first, then reassess if tool timeout detection is needed based on real stuck agent incidents.

---

### Implementation Details

**What to implement first:**
- **SESSION_LOG.md enhancement** - Highest impact for debugging stuck agents
- Extend MessagePart struct with Tool, CallID, State fields
- Update formatMessageToMarkdown to show tool details instead of just "tool"
- Test with multiple session types to ensure formatting works

**Things to watch out for:**
- ⚠️ Tool state structure is nested - need ToolState struct with Input (map[string]any), Output (string), Status, Title, Metadata
- ⚠️ Backward compatibility - existing transcripts without tool details should still work
- ⚠️ Large tool outputs (e.g., file reads) may need truncation in transcript to avoid giant SESSION_LOG.md files
- ⚠️ JSON unmarshaling of `any` type for tool inputs needs careful handling

**Areas needing further investigation:**
- SSE message.part event monitoring (mentioned in constraints but implementation unclear)
- Tool timeout detection design (if needed in future)
- Whether to show full tool output or truncated summary in SESSION_LOG.md

**Success criteria:**
- ✅ SESSION_LOG.md shows tool names instead of just "tool" placeholder
- ✅ Tool details include description/title when available
- ✅ Tool parameters visible for debugging (e.g., `bash: "bd comment ..."`)
- ✅ Existing transcripts still parse correctly
- ✅ No performance regression in transcript generation

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

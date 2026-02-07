<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Agent activity infrastructure already exists via SSE events; only needed to enhance tooltip display to show expressive status.

**Evidence:** Found current_activity field already captures tool names and reasoning (agents.ts:62-67, 754-778); "Processing" tooltip was generic (agent-card.svelte:458-467).

**Knowledge:** Backend sends expressive data (tool names, reasoning) that frontend wasn't displaying; tooltip is perfect location for showing "Hatching...", "Running Bash...", etc.

**Next:** Implemented getExpressiveStatus() function to format activity into expressive text with thinking duration; visual verification needed.

**Promote to Decision:** recommend-no (UI enhancement, not architectural pattern)

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

# Investigation: Expressive Agent Status Display Status

**Question:** How can we make agent status display more expressive (show "Hatching...", "Running Bash...", "Reading files..." instead of generic "Processing")?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** feature-impl agent
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

### Finding 1: Current Activity Infrastructure Already Exists

**Evidence:** The `current_activity` field on Agent already captures type, text, and timestamp (web/src/lib/stores/agents.ts:62-67). SSE events from OpenCode populate this with tool names like "Using bash", "Reading files", etc. (agents.ts:754-778). The agent card displays this activity text at line 690-691.

**Source:** 
- web/src/lib/stores/agents.ts:62-67 (current_activity interface)
- web/src/lib/stores/agents.ts:648-778 (SSE event handlers)
- web/src/lib/components/agent-card/agent-card.svelte:687-696 (activity display)

**Significance:** We don't need to build new infrastructure - the backend already sends activity data via SSE. We just need to enhance the display to be more expressive.

---

### Finding 2: "Processing" Tooltip is Generic

**Evidence:** When `displayState === 'running'`, the tooltip shows "Processing" with subtext "Agent is actively generating a response" (agent-card.svelte:458-467). This doesn't leverage the `current_activity` data that's already available.

**Source:** 
- web/src/lib/components/agent-card/agent-card.svelte:458-467

**Significance:** The tooltip area is the perfect place to show expressive status. We should show the current activity instead of generic "Processing".

---

### Finding 3: Thinking Duration Not Captured

**Evidence:** While we track `current_activity.timestamp`, we don't specifically identify "thinking" vs "tool execution" phases. OpenCode sends `reasoning` type for thinking (agents.ts:772-774) but we don't calculate duration.

**Source:** 
- web/src/lib/stores/agents.ts:772-774 (reasoning type handling)

**Significance:** To show "Hatching... (thought for 8s)", we need to detect reasoning activity and calculate elapsed time since it started.

---

## Synthesis

**Key Insights:**

1. **Infrastructure is ready** - Backend already sends expressive activity data via SSE (tool names, reasoning, etc.). We don't need backend changes, only frontend display improvements.

2. **Tooltip is the natural place** - The processing indicator tooltip (agent-card.svelte:458-467) currently shows generic "Processing". This is where we should show "Running Bash...", "Reading files...", etc.

3. **Activity display already works** - The activity section below the title (lines 687-696) shows current activity. We need to make the tooltip consistent with this, and add thinking duration calculation.

**Answer to Investigation Question:**

We can make agent status more expressive by:
1. Showing current activity in the "Processing" tooltip (currently shows generic text)
2. Detecting "reasoning" activity type and calculating thinking duration
3. Using expressive verbs like "Hatching..." for thinking, "Running..." for tools, "Reading..." for file operations

The infrastructure already exists - we just need to enhance the display logic.

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

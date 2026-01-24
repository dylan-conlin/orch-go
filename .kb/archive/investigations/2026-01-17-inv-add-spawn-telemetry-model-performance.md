<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Extended existing events system with model performance telemetry (model, skill, duration, tokens, outcome) at spawn/complete/abandon lifecycle points.

**Evidence:** Added fields to AgentCompletedData/AgentAbandonedData structs, enhanced event logging in spawn/complete/abandon commands, verified build succeeds and events.jsonl exists.

**Knowledge:** Events infrastructure already tracked lifecycle with extensible schema - only needed to add telemetry fields, not create parallel system. GetSessionTokens already implemented for token retrieval.

**Next:** Create query tool or jq examples for evidence-based model selection analysis.

**Promote to Decision:** recommend-no (tactical enhancement, not architectural decision)

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

# Investigation: Add Spawn Telemetry Model Performance

**Question:** How should we instrument the spawn/completion lifecycle to collect model performance telemetry (model, skill, outcome, tokens, duration, retries, failure reason) in a queryable format for evidence-based model selection?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** orch-go-x67lc
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Events system already exists with lifecycle tracking

**Evidence:** 
- `pkg/events/logger.go` implements JSONL event logging to `~/.orch/events.jsonl`
- Already logs: `session.spawned`, `session.completed`, `session.error`, `session.status`, `agent.completed`, `verification.failed`, `verification.bypassed`
- Event structure includes `Type`, `SessionID`, `Timestamp`, and flexible `Data` map

**Source:** 
- `pkg/events/logger.go:1-346`
- Existing event types: lines 14-37
- Event struct: lines 40-45

**Significance:** 
We don't need to create a new telemetry infrastructure - we can extend the existing events system. The architecture already supports lifecycle tracking with metadata, we just need to add model/token/duration data to existing events.

---

### Finding 2: Spawn already collects basic metadata but missing model performance data

**Evidence:**
- `cmd/orch/spawn_cmd.go` captures spawn time, skill, task, model selection, beads ID
- Workspace metadata includes `.spawn_time`, `.session_id`, `.beads_id`, `.tier`
- `LogSpawn()` currently only captures `prompt` and `title`, not model or skill

**Source:**
- `cmd/orch/spawn_cmd.go:1-100` - spawn command structure
- `.kb/models/spawn-architecture.md:45-88` - workspace metadata files
- `pkg/events/logger.go:99-110` - LogSpawn implementation

**Significance:**
Spawn time collection point exists but needs to capture: model used, skill name. This data is available in spawn command context but not being logged to events.

---

### Finding 3: Completion tracks verification but not outcome/duration/tokens

**Evidence:**
- `cmd/orch/complete_cmd.go` logs `agent.completed` event with verification metadata
- `AgentCompletedData` includes: `BeadsID`, `Workspace`, `Forced`, `VerificationPassed`, `GatesBypassed`, `Skill`
- Missing: duration (spawn → complete time), tokens used, actual outcome (success/failure reasons)

**Source:**
- `cmd/orch/complete_cmd.go:1-100` - complete command structure
- `pkg/events/logger.go:201-242` - AgentCompletedData struct and LogAgentCompleted

**Significance:**
Completion event exists but needs enrichment with: (1) duration calculated from workspace `.spawn_time`, (2) token usage from OpenCode session API, (3) detailed outcome classification.

---

### Finding 4: Abandon path doesn't log telemetry

**Evidence:**
- `cmd/orch/abandon_cmd.go` handles agent abandonment but no telemetry event logged
- Creates `FAILURE_REPORT.md` in workspace with failure reason
- No structured event logged to `events.jsonl` for abandonment outcome

**Source:**
- `cmd/orch/abandon_cmd.go:1-100` - abandon command structure
- No `LogAgentAbandoned` or similar in `pkg/events/logger.go`

**Significance:**
Abandoned agents are a valid outcome (vs success/failure) and need telemetry collection. This is the third lifecycle endpoint (spawn, complete, abandon) that needs instrumentation.

---

### Finding 5: Token usage available via OpenCode session API

**Evidence:**
- OpenCode exposes session details via HTTP API at port 4096
- Session includes message history with token counts
- Need to query OpenCode API at completion/abandon time to get total tokens

**Source:**
- `pkg/opencode/*.go` - OpenCode client package exists
- `.kb/models/spawn-architecture.md:58-60` - mentions OpenCode session creation
- Knowledge from prior context that OpenCode has HTTP API

**Significance:**
Token usage is not stored in workspace metadata but can be retrieved from OpenCode API. Need to add API call at completion/abandon time to fetch token counts before logging telemetry.

---

## Synthesis

**Key Insights:**

1. **Events system already comprehensive** - The existing events infrastructure at `pkg/events/logger.go` already logged spawn/complete lifecycle with extensible JSONL schema. No need for parallel telemetry system.

2. **Token retrieval already implemented** - `GetSessionTokens()` method and `TokenStats` struct already existed in opencode client package. Phase 2 was already complete.

3. **Spawn events already included model/skill** - Spawn commands already logged model and skill in session.spawned events across all modes (headless, tmux, inline, claude). Phase 1 was already complete.

4. **Minimal changes needed** - Only required adding 4 fields (DurationSeconds, TokensInput, TokensOutput, Outcome) to completion/abandon events. Duration calculated from workspace `.spawn_time`, tokens from OpenCode API.

5. **Graceful degradation** - Telemetry collection is non-blocking - if workspace files missing or OpenCode API unavailable, event is still logged with zeros/empty values.

**Answer to Investigation Question:**

To instrument spawn/completion lifecycle for model performance telemetry, extend the existing events system by:
1. Adding telemetry fields to AgentCompletedData and new AgentAbandonedData structs
2. Collecting duration from workspace `.spawn_time` file (spawn to complete/abandon time)  
3. Retrieving token usage via existing `GetSessionTokens()` OpenCode API method
4. Determining outcome classification (success/forced/failed/abandoned)
5. Logging enriched events to existing `~/.orch/events.jsonl` in JSONL format

Model/skill already captured at spawn. Token retrieval infrastructure already existed. Implementation required minimal changes (3 commits, ~120 lines added).

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

**Extend existing events system with telemetry fields** - Add duration, tokens, and outcome to agent.completed and new agent.abandoned events.

**Why this approach:**
- Reuses proven infrastructure (events.jsonl already handles lifecycle tracking)
- GetSessionTokens() already implemented (no new API integration needed)
- JSONL format is queryable via jq, grep, or simple scripts
- Minimal code changes (3 files, ~120 lines)

**Trade-offs accepted:**
- No dedicated query tool in Phase 5 (use jq scripts or future `orch telemetry` subcommand)
- Token data missing if OpenCode API unavailable (graceful degradation acceptable)

**Implementation sequence:**
1. ✅ Add telemetry fields to AgentCompletedData struct (foundation for all events)
2. ✅ Collect duration/tokens in complete_cmd.go (most common lifecycle path)
3. ✅ Add AgentAbandonedData and abandon event logging (cover all outcomes)
4. ⏸ Optional: Create jq query examples or `orch telemetry query` subcommand

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

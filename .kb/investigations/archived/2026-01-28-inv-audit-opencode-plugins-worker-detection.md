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

# Investigation: Audit Opencode Plugins Worker Detection

**Question:** Which OpenCode plugins use incorrect worker detection approaches (title-based heuristics or ORCH_WORKER env var checks) instead of the correct session.metadata.role approach?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Worker Agent (orch-go-21000)
**Phase:** Investigating
**Next Step:** Analyze each plugin file for worker detection patterns
**Status:** Active

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting Investigation - 8 plugins to audit

**Evidence:** Found 8 plugin files in .opencode/plugins/ directory:
- coaching.ts
- orchestrator-tool-gate.ts
- orch-hud.ts
- task-tool-gate.ts
- evidence-hierarchy.ts
- orchestrator-session.ts
- event-test.ts
- action-log.ts

**Source:** `glob .opencode/plugins/*.ts` command

**Significance:** These are all the plugins that need to be audited for worker detection approaches. Will analyze each one to identify which use metadata.role (correct) vs title-based/env var detection (incorrect).

---

### Finding 2: coaching.ts uses correct metadata.role approach

**Evidence:** coaching.ts correctly detects workers via `sessionMetadata.role === "worker"` from the session.created event (lines 2018-2024). This checks the metadata.role field which OpenCode sets when the x-opencode-env-ORCH_WORKER header is present during session creation.

**Source:** 
- .opencode/plugins/coaching.ts:2018-2024
- .opencode/plugins/coaching.ts:64-74 (documentation comment explaining the approach)

**Significance:** This is the reference implementation that other plugins should follow. It's reliable because it uses metadata set at session creation time BEFORE any tool calls occur, eliminating the need for heuristics.

---

### Finding 3: Three plugins use title-based detection (incorrect)

**Evidence:** Found title-based worker detection in:
1. **orchestrator-tool-gate.ts**: Uses `detectOrchestratorFromTitle()` function (line 308) and `isWorkerSession()` (line 240) which checks for patterns like `[beads-id]` in title
2. **orch-hud.ts**: Uses `isWorkerByTitle()` function (line 89) that checks for beads ID in title and excludes orchestrator patterns
3. **task-tool-gate.ts**: Uses `detectOrchestratorFromTitle()` (line 166) and `isWorkerSession()` (line 82) similar to orchestrator-tool-gate.ts

**Source:**
- grep search for "isWorkerByTitle|detectOrchestratorFromTitle|isWorkerSession" in .opencode/plugins/*.ts
- .opencode/plugins/orchestrator-tool-gate.ts:240, 308, 472, 492
- .opencode/plugins/orch-hud.ts:89, 106
- .opencode/plugins/task-tool-gate.ts:82, 166, 258, 280

**Significance:** These plugins use unreliable heuristics (title patterns) instead of the authoritative metadata.role field. Should be updated to use session.metadata.role approach like coaching.ts.

---

### Finding 4: orchestrator-session.ts uses workspace path detection (indirect)

**Evidence:** orchestrator-session.ts uses `detectWorkerSession()` function (line 155) that detects workers by examining tool arguments for SPAWN_CONTEXT.md reads and .orch/workspace/ file paths. This is an indirect detection method, not using metadata.role.

**Source:**
- .opencode/plugins/orchestrator-session.ts:155-196
- Function checks for workspace paths and SPAWN_CONTEXT.md reads

**Significance:** While this approach works (workers do read SPAWN_CONTEXT.md early), it's indirect and relies on tool execution patterns rather than the authoritative metadata.role field. Should be updated to use metadata.role for consistency.

---

### Finding 5: Three plugins don't perform worker detection (acceptable)

**Evidence:** These plugins have no worker detection logic:
- evidence-hierarchy.ts
- event-test.ts  
- action-log.ts

**Source:** Grep searches found no worker detection patterns in these files

**Significance:** These plugins either don't need worker-specific behavior or apply to all sessions equally. No changes needed for these plugins.

---

## Synthesis

**Key Insights:**

1. **Only one plugin uses the correct approach** - coaching.ts is the only plugin using session.metadata.role for worker detection. This should be the reference implementation for other plugins.

2. **Title-based detection is widespread** - Three critical plugins (orchestrator-tool-gate, orch-hud, task-tool-gate) use title-based heuristics. These are unreliable because titles can be changed or may not follow expected patterns.

3. **Detection method consistency matters** - Having different plugins use different detection methods (metadata.role vs title vs workspace paths) creates risk of inconsistent behavior where some plugins treat a session as worker while others treat it as orchestrator.

**Answer to Investigation Question:**

Four plugins need updating to use session.metadata.role approach:

1. **orchestrator-tool-gate.ts** - Currently uses title-based detection via `isWorkerSession()` and `detectOrchestratorFromTitle()` functions
2. **orch-hud.ts** - Currently uses title-based detection via `isWorkerByTitle()` function  
3. **task-tool-gate.ts** - Currently uses title-based detection via `isWorkerSession()` and `detectOrchestratorFromTitle()` functions
4. **orchestrator-session.ts** - Currently uses workspace path detection via `detectWorkerSession()` function

The reference implementation in coaching.ts (lines 2018-2024) shows the correct pattern: check `sessionMetadata.role === "worker"` in the session.created event handler.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 8 plugin files read and analyzed for worker detection patterns (verified: read all files, searched for detection functions)
- ✅ coaching.ts uses sessionMetadata.role === "worker" approach (verified: lines 2018-2024)
- ✅ Three plugins use title-based detection (verified: grep found isWorkerByTitle/detectOrchestratorFromTitle functions)
- ✅ orchestrator-session.ts uses workspace path detection (verified: detectWorkerSession function at line 155)
- ✅ Three plugins have no worker detection (verified: no detection patterns found via grep)

**What's untested:**

- ⚠️ Whether the problematic plugins would work correctly if updated to use metadata.role (implementation not tested)
- ⚠️ Whether any plugins have subtle bugs due to inconsistent worker detection across plugins
- ⚠️ Whether there are edge cases where title-based detection fails in production

**What would change this:**

- Finding would be wrong if metadata.role is NOT reliably set by OpenCode when x-opencode-env-ORCH_WORKER header is present
- Finding would be wrong if any of the "problematic" plugins actually do check metadata.role somewhere I missed
- Finding would be wrong if title-based detection is intentionally used as a fallback mechanism

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Migrate all plugins to use session.metadata.role in session.created event handler** - Replace title-based and workspace path detection with the authoritative metadata.role field.

**Why this approach:**
- Uses authoritative data set by OpenCode at session creation time (no heuristics needed)
- Consistent across all plugins (reduces confusion and edge case bugs)
- Detected BEFORE any tool calls occur (eliminates race conditions)
- coaching.ts already proves this approach works reliably in production

**Trade-offs accepted:**
- Requires changing 4 plugins (non-trivial but straightforward)
- Old detection methods could be kept as fallback, but better to remove for clarity
- Some plugins might need refactoring if they rely on progressive detection

**Implementation sequence:**
1. **Start with orchestrator-tool-gate.ts** - Most critical plugin for security (blocks primitives), so validate metadata.role approach works here first
2. **Update task-tool-gate.ts** - Similar structure to orchestrator-tool-gate.ts, so learnings transfer
3. **Update orch-hud.ts** - Simpler plugin, good for confirming pattern works across different use cases
4. **Update orchestrator-session.ts** - Has different detection timing needs, may require more careful refactoring

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
- Add session.created event handler to each plugin (copy pattern from coaching.ts:1993-2030)
- Replace title-based detection calls with simple Map lookups to workerSessions cache
- Test each plugin after update by spawning a worker and verifying it's detected correctly

**Things to watch out for:**
- ⚠️ orchestrator-session.ts has progressive detection that runs on tool.execute.before - need to ensure session.created fires early enough
- ⚠️ task-tool-gate.ts currently SETS metadata.role after detecting orchestrators via title - this logic needs reversal (read metadata.role instead of setting it)
- ⚠️ Some plugins cache detection results in Maps - ensure cache is populated from session.created, not tool execution
- ⚠️ Plugins that check both worker and orchestrator detection need to handle "unknown" role case (neither worker nor orchestrator set)

**Areas needing further investigation:**
- Whether metadata.role is set reliably for ALL spawn modes (tmux, daemon, interactive)
- Whether there are any edge cases where session.created doesn't fire before tool.execute.before
- Whether the OpenCode API for setting metadata.role (used by task-tool-gate.ts) is still needed if we're reading it instead

**Success criteria:**
- ✅ All 4 plugins use sessionMetadata.role === "worker" for worker detection
- ✅ No title-based detection functions (isWorkerByTitle, detectOrchestratorFromTitle, isWorkerSession) remain in updated plugins
- ✅ Worker spawns are correctly identified by all plugins (test by spawning worker and checking logs)
- ✅ Orchestrator sessions are NOT incorrectly identified as workers (test by starting orchestrator session)

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

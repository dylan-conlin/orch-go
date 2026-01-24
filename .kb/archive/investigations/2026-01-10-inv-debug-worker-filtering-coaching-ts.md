<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Plugin-level worker detection impossible because OpenCode plugins run in server process, not per-agent; moved detection to per-session in tool hooks using input.args.workdir and filePath patterns.

**Evidence:** Verified bash tool has workdir parameter (opencode/packages/opencode/src/tool/bash.ts:65-70), implemented detectWorkerSession() with three signals (workdir, SPAWN_CONTEXT.md reads, filePath patterns), committed fix 6e6503ae.

**Knowledge:** OpenCode architecture has single server process serving all sessions; process.env checks at plugin level fail because ORCH_WORKER set in spawned agent processes, not server; tool hooks provide per-session context via input.sessionID and input.args.

**Next:** Fix complete and committed; smoke test by spawning worker and verifying sessionID not in coaching-metrics.jsonl; consider documenting "plugins run in server" in OpenCode plugin guide.

**Promote to Decision:** recommend-yes (documents fundamental OpenCode plugin architecture constraint: server process vs per-agent detection, applies to all plugins checking env vars)

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

# Investigation: Debug Worker Filtering Coaching Ts

**Question:** Why does worker filtering fail in coaching.ts, and how can we detect workers per-session instead of at plugin init?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None (fix committed)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Plugin-Level Worker Detection Cannot Work

**Evidence:** coaching.ts:84-108 defines isWorker() checking process.env.ORCH_WORKER, but plugin init runs in OpenCode server process (not per-agent). Workers spawn in separate processes with ORCH_WORKER=1 set in their environment only. Server process never sees ORCH_WORKER=1.

**Source:** coaching.ts:843 (plugin init call), orchestrator-session.ts:76-100 (same bug pattern), spawn context line 1 (root cause statement).

**Significance:** Prior investigation (2026-01-10-inv-add-worker-filtering-coaching-ts.md) implemented a solution that fundamentally cannot work due to process boundary.

---

### Finding 2: Bash Tool Has workdir Parameter for Per-Session Detection

**Evidence:** opencode/packages/opencode/src/tool/bash.ts:65-70 defines workdir parameter passed to tool execution. In tool.execute.after hook, input.args.workdir contains the working directory for that specific bash command. Workers run commands from .orch/workspace/{name}/ directories.

**Source:** opencode/packages/opencode/src/tool/bash.ts:65-70, verified input.args structure in coaching.ts:1044 accessing input.args.command.

**Significance:** Can detect worker sessions by checking if input.args.workdir contains ".orch/workspace/" pattern - provides per-session worker detection in tool hooks.

---

### Finding 3: Multiple Per-Session Detection Signals Available

**Evidence:** Beyond workdir, can detect workers via: (1) read tool accessing SPAWN_CONTEXT.md (workers always read this file), (2) pwd command output containing .orch/workspace/, (3) any tool with filePath args pointing to workspace directories. Session-level tracking via Map<sessionID, boolean> prevents repeated checks.

**Source:** SPAWN_CONTEXT.md always exists in worker workspaces, read tool has filePath parameter (evidence-hierarchy.ts:314), coaching.ts:856 shows Map<sessionID, SessionState> pattern for per-session tracking.

**Significance:** Multiple independent signals provide redundancy - if one detection method fails, others can still identify workers correctly. Caching per-session prevents performance impact of repeated checks.

---

## Synthesis

**Key Insights:**

1. **Plugin-Level Detection is Architecturally Impossible** - OpenCode plugins run in the server process which never sees ORCH_WORKER=1 env var set in spawned agent processes. Process boundary prevents plugin init from detecting workers. Solution must move to per-session detection in tool hooks.

2. **Tool Hooks Provide Per-Session Context** - tool.execute.after receives input.sessionID and input.args containing tool parameters. Bash tool's workdir parameter exposes agent's working directory per-command, enabling detection of .orch/workspace/ paths.

3. **Session-Level Caching Prevents Overhead** - Using Map<sessionID, boolean> to cache worker status after first detection avoids repeated checks. Multiple detection signals (workdir, SPAWN_CONTEXT.md reads, pwd output) provide redundancy without performance cost.

**Answer to Investigation Question:**

Worker filtering must move from plugin init to tool hooks. Implementation: (1) Track worker sessions in Map<sessionID, boolean>, (2) In tool.execute.after, check input.args.workdir for ".orch/workspace/" pattern, (3) Check read tool for SPAWN_CONTEXT.md access, (4) Short-circuit metrics tracking for cached worker sessions. This fixes the architectural flaw where plugin init runs in server process but workers spawn in separate processes. Same fix applies to orchestrator-session.ts.

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

**Per-Session Worker Detection in Tool Hooks** - Move worker detection from plugin init to tool.execute.after, checking workdir and file paths per-session with Map-based caching.

**Why this approach:**
- Fixes architectural impossibility of detecting env vars across process boundary
- Leverages existing sessionID in tool hooks for per-session tracking
- Multiple detection signals (workdir, SPAWN_CONTEXT.md reads) provide redundancy
- Caching prevents performance overhead from repeated checks

**Trade-offs accepted:**
- First few tool calls from worker might be tracked before detection (acceptable - metrics pruned anyway)
- Slightly more complex than plugin-level check (necessary given process architecture)

**Implementation sequence:**
1. Add Map<sessionID, boolean> to track known worker sessions (foundational data structure)
2. Add detectWorkerSession() function checking workdir and file paths (detection logic)
3. In tool.execute.after, check cache first, then detect and cache result (performance + correctness)
4. Short-circuit all metric tracking for cached worker sessions (filtering)

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

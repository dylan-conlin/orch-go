<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode has comprehensive native agent/spawn support: agent modes (subagent/primary), task tool for spawning, session hierarchy via parentID, and metadata.role for identity - orch-go currently uses pragmatic bolt-on (ORCH_WORKER headers) but could integrate more deeply.

**Evidence:** Examined OpenCode source (agent/agent.ts, session/index.ts, tool/task.ts, acp/agent.ts) showing: Agent.Info with mode enum, Session.Info with parentID, task tool creating sessions with parentID=ctx.sessionID, x-opencode-env-ORCH_WORKER header mapping to metadata.role. Verified via curl that running sessions have metadata.role="worker".

**Knowledge:** Three integration levels exist: (A) Status quo - ORCH_WORKER headers + external registry (proven, decoupled), (B) Incremental - add parentID for hierarchy visibility (low risk), (C) Deep - custom agents + task tool (high coupling, architectural change). No clear "best" - depends on whether orch-go's orchestration model aligns with OpenCode's task delegation paradigm and cross-project requirements.

**Next:** Present findings to orchestrator for architectural decision. If choosing Option B (incremental), test cross-project parentID behavior first. If staying with Option A (status quo), document that OpenCode hierarchy exists but orch-go intentionally uses external model for decoupling.

**Promote to Decision:** Issue created: orch-go-21092 (native spawn decision)

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

# Investigation: Investigate Opencode Native Agent Spawn

**Question:** Does OpenCode have built-in support for spawning sub-agents or worker sessions? How does it handle agent identity? Could orch-go integrate more deeply with OpenCode's native agent model instead of bolting on ORCH_WORKER headers?

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

### Finding 1: Investigation Starting Approach

**Evidence:** Will examine OpenCode source code (~/Documents/personal/opencode), API documentation in .kb/guides/opencode.md, and existing orch-go integration code to understand native agent abstractions.

**Source:** Investigation plan

**Significance:** Establishes approach for discovering whether OpenCode has native agent/spawn concepts that orch-go could leverage.

---

### Finding 2: OpenCode Has Native Agent Mode System

**Evidence:** Agent.Info schema defines three agent modes: "subagent", "primary", and "all". Built-in agents include general (subagent), explore (subagent), build (primary), and plan (primary). Mode controls whether agent can be spawned as a task vs used as primary interactive agent.

**Source:** 
- ~/Documents/personal/opencode/packages/opencode/src/agent/agent.ts:27 (mode enum)
- ~/Documents/personal/opencode/packages/opencode/src/agent/agent.ts:112-126 (general subagent definition)
- ~/Documents/personal/opencode/packages/opencode/src/agent/agent.ts:127-153 (explore subagent definition)

**Significance:** OpenCode has built-in concept of subagents vs primary agents. This is a native abstraction that orch-go could leverage instead of relying solely on ORCH_WORKER environment variable.

---

### Finding 3: OpenCode Has Native Session Hierarchy via parentID

**Evidence:** Session.Info schema includes optional parentID field. Sessions can be root (no parentID) or child (has parentID). Session.fork() creates child sessions by copying message history. Session.children() retrieves all child sessions of a parent. Session list API has roots=true filter to get only root sessions.

**Source:**
- ~/Documents/personal/opencode/packages/opencode/src/session/index.ts:49 (parentID field)
- ~/Documents/personal/opencode/packages/opencode/src/session/index.ts:157-193 (fork function)
- ~/Documents/personal/opencode/packages/opencode/src/session/index.ts:366-372 (children function)
- ~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:45 (roots filter)

**Significance:** OpenCode has native session tree structure. This could be used to represent orchestrator->worker relationships instead of relying on external registry.

---

### Finding 4: OpenCode's Task Tool is Native Subagent Spawn Mechanism

**Evidence:** Task tool (tool/task.ts) creates child sessions with `parentID: ctx.sessionID` (line 69). Takes `subagent_type` parameter to select agent (general, explore, or custom). Title automatically appends " (@{agent} subagent)". Returns session_id in metadata for continuity. Task tool blocks todowrite/todoread for spawned agents unless explicitly permitted.

**Source:**
- ~/Documents/personal/opencode/packages/opencode/src/tool/task.ts:15-21 (parameters schema)
- ~/Documents/personal/opencode/packages/opencode/src/tool/task.ts:68-69 (session creation with parentID)
- ~/Documents/personal/opencode/packages/opencode/src/tool/task.ts:147-162 (prompt execution in child session)

**Significance:** OpenCode has built-in spawning mechanism via task tool. This is how OpenCode agents natively delegate to subagents. orch-go's spawn command is an external alternative to this built-in mechanism.

---

### Finding 5: orch-go Uses x-opencode-env Headers for Worker Identity

**Evidence:** orch-go sets ORCH_WORKER=1 environment variable for all spawned agents. OpenCode client converts this to x-opencode-env-ORCH_WORKER HTTP header when creating sessions (client.go:596). OpenCode server reads this header and sets session metadata.role = "worker" (session.ts:208-211). Session.Info schema has optional metadata.role: "orchestrator" | "meta-orchestrator" | "worker".

**Source:**
- pkg/opencode/client.go:593-596 (header conversion)
- ~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:207-211 (header reading)
- ~/Documents/personal/opencode/packages/opencode/src/session/index.ts:80-84 (metadata schema)
- cmd/orch/spawn_cmd.go:836-837 (ORCH_WORKER=1 env var)

**Significance:** orch-go already integrates with OpenCode's native session metadata system via custom headers. This is a pragmatic bolt-on approach that leverages OpenCode's extensibility.

---

### Finding 6: OpenCode Has ACP Session Forking/Resume Support

**Evidence:** ACP (Agent Client Protocol) implementation has native forkSession, resumeSession, and listSessions capabilities. Fork creates new session from parent's message history. Resume reattaches to existing session. List filters by directory with pagination. These are standardized protocol operations, not orch-specific.

**Source:**
- ~/Documents/personal/opencode/packages/opencode/src/acp/agent.ts:441 (fork capability)
- ~/Documents/personal/opencode/packages/opencode/src/acp/agent.ts:600-661 (forkSession implementation)
- ~/Documents/personal/opencode/packages/opencode/src/acp/agent.ts:663-688 (resumeSession implementation)
- ~/Documents/personal/opencode/packages/opencode/src/acp/agent.ts:555-598 (listSessions implementation)

**Significance:** OpenCode's ACP layer provides standardized session lifecycle operations. These are designed for agent/IDE integrations. orch-go could potentially use ACP instead of direct HTTP API.

---

## Synthesis

**Key Insights:**

1. **OpenCode has a comprehensive native agent/spawn model** - Not just basic sessions, but full agent modes (subagent/primary/all), native spawning via task tool, session hierarchy with parentID, and identity tracking via metadata.role. This is a complete system designed for agent delegation, not a bolt-on afterthought.

2. **orch-go's current approach is a pragmatic bolt-on** - Uses ORCH_WORKER=1 env var converted to x-opencode-env-ORCH_WORKER header, which OpenCode reads to set metadata.role="worker". This works but doesn't leverage OpenCode's native hierarchy (parentID) or agent mode system. Registry tracks workers externally instead of querying session.children().

3. **Two distinct spawning paradigms coexist** - OpenCode native (task tool → subagent session with parentID) vs orch-go external (CLI spawn → worker session with ORCH_WORKER header). These aren't incompatible, but they represent different philosophies: OpenCode's in-process delegation vs orch-go's external orchestration.

4. **Session hierarchy is underutilized** - OpenCode's parentID creates natural session trees visible in UI. orch-go could create orchestrator->worker hierarchy by setting parentID on spawned sessions, making relationships visible without external registry. Session.children() would provide built-in worker listing.

5. **Agent modes define spawn boundaries** - OpenCode agents with mode="subagent" are designed for task tool spawning. Custom agents can be defined in opencode.json with specific permissions, prompts, and modes. orch-go could define investigation/feature-impl as custom OpenCode agents instead of managing them externally.

**Answer to Investigation Question:**

**Yes, OpenCode has robust built-in support for spawning sub-agents and agent identity:**

1. **Native spawning**: task tool creates child sessions with parentID hierarchy
2. **Agent identity**: Agent.Info schema with mode (subagent/primary/all), custom agents definable in config
3. **Session hierarchy**: parentID field, Session.fork(), Session.children() for tree structure
4. **Role metadata**: metadata.role field ("orchestrator" | "meta-orchestrator" | "worker") - orch-go already integrates via x-opencode-env headers

**Could orch-go integrate more deeply?** Yes, potentially:
- Use parentID to create session hierarchy instead of flat registry
- Define orch worker agents as custom OpenCode agents with specific permissions
- Query Session.children() instead of maintaining separate registry
- Consider task tool for in-session spawning (if agent delegates to worker)

**However, trade-offs exist:**
- orch-go's external orchestration model gives more control over lifecycle
- CLI spawning allows cross-project orchestration (OpenCode sessions are project-scoped)
- Current ORCH_WORKER header approach is simple and works reliably
- Deeper coupling to OpenCode's agent model reduces portability if orch-go needs to support other agent runtimes

**Conclusion:** OpenCode's native abstractions are strong. Current orch-go integration is pragmatic (works via headers). Deeper integration is possible (parentID hierarchy, custom agents) but not necessarily better given orch-go's cross-project orchestration goals.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode has Agent.Info schema with mode field (verified: read agent/agent.ts source)
- ✅ OpenCode has Session.Info with parentID field (verified: read session/index.ts source)
- ✅ Task tool creates sessions with parentID=ctx.sessionID (verified: read tool/task.ts:69)
- ✅ OpenCode reads x-opencode-env-ORCH_WORKER header (verified: session.ts:207-211)
- ✅ orch-go sets ORCH_WORKER=1 and converts to header (verified: client.go:593-596)
- ✅ Session API returns sessions with metadata.role (verified: curl /session shows role="worker")
- ✅ ACP has forkSession, resumeSession, listSessions (verified: read acp/agent.ts)

**What's untested:**

- ⚠️ Setting parentID from orch-go spawns (orch-go doesn't currently do this)
- ⚠️ Performance of Session.children() vs registry lookups (not benchmarked)
- ⚠️ Custom agent definitions for orch workers in opencode.json (not implemented)
- ⚠️ Cross-project session hierarchy (does parentID work when parent/child in different projects?)
- ⚠️ Task tool as alternative to CLI spawn for orch (architectural question, not tested)
- ⚠️ ACP integration vs direct HTTP API (orch-go uses HTTP, ACP layer exists but untested for orch use)

**What would change this:**

- Finding that parentID is project-scoped and breaks cross-project spawning would eliminate hierarchy as option
- Discovery that Session.children() is O(N) scan of all sessions would impact performance for large session counts
- Evidence that custom agents must be in project .opencode/ directory (not global) would limit cross-project agent definitions
- Proof that task tool only works within same session context (can't spawn independent workers) would eliminate it as spawn alternative

---

## Implementation Recommendations

**Purpose:** Present options for orch-go integration with OpenCode's native agent model. No single "recommended" approach - each has valid use cases.

### Option A: Status Quo (Pragmatic Bolt-On)

**Keep current ORCH_WORKER header approach, external registry, no session hierarchy.**

**Pros:**
- Already works reliably (proven in production)
- Simple: one header sets metadata.role
- Decoupled from OpenCode's agent model (portable if runtime changes)
- External registry gives full control over worker lifecycle
- No migration cost

**Cons:**
- Session hierarchy invisible in OpenCode UI (can't see orchestrator->worker relationships)
- Duplicate tracking (registry + OpenCode sessions)
- Can't use Session.children() for worker queries

**When to use:**
- orch-go orchestration model fundamentally differs from OpenCode's task tool paradigm
- Cross-project orchestration is core requirement (unclear if parentID supports this)
- Keeping options open for non-OpenCode runtimes

---

### Option B: Incremental Integration (Add Hierarchy)

**Set parentID on spawned sessions to create orchestrator->worker hierarchy, keep ORCH_WORKER and registry.**

**Pros:**
- Session relationships visible in OpenCode UI
- Can query Session.children() as alternative to registry
- Maintains backward compatibility (ORCH_WORKER still works)
- Low implementation cost (one field in CreateSession)

**Cons:**
- Need to test cross-project parentID behavior
- Adds another tracking mechanism (parentID + registry + ORCH_WORKER)
- Unclear if parentID helps or just adds complexity

**Implementation:**
1. Add parentSessionID parameter to CreateSession in pkg/opencode/client.go
2. Pass orchestrator session ID when spawning workers
3. Test cross-project spawning (does parentID work when parent is orch-go session, child is worker?)
4. Optionally migrate registry queries to use Session.children()

**When to use:**
- Want better OpenCode UI visibility without changing orchestration model
- Willing to test cross-project parentID behavior
- Keeping registry as source of truth, parentID as auxiliary

---

### Option C: Deep Integration (Custom Agents + Task Tool)

**Define orch worker agents as OpenCode custom agents, potentially use task tool for spawning.**

**Pros:**
- Leverages OpenCode's full agent system (modes, permissions, prompts)
- Natural delegation model (orchestrator uses task tool to spawn)
- Permission management via OpenCode's agent config
- Fully native to OpenCode

**Cons:**
- High coupling to OpenCode's agent model
- Unclear if custom agents can be global (may be project-scoped)
- Task tool designed for in-session delegation, not external orchestration
- Major architectural change from CLI spawning
- Loss of control over worker lifecycle

**Implementation:**
1. Define investigation/feature-impl agents in opencode.json with mode="subagent"
2. Experiment: can agents be defined globally (~/.config/opencode/) or only per-project?
3. Test task tool for spawning (does it work for orch use case?)
4. Likely requires rethinking spawn command entirely

**When to use:**
- orch-go's orchestration model aligns with OpenCode's task tool paradigm
- All orch operations happen within single OpenCode project context
- Want maximum integration with OpenCode ecosystem
- Willing to couple tightly to OpenCode's agent model

---

**No single recommendation** - These are architectural choices:
- **Option A** for stability, decoupling, cross-runtime portability
- **Option B** for incremental improvement, better UI visibility, low risk
- **Option C** for full OpenCode integration, willing to rearchitect

---

### Implementation Details

**If pursuing Option B (Incremental Integration):**

**First steps:**
1. Add parentSessionID to CreateSession API call in pkg/opencode/client.go
2. Modify spawn commands to pass orchestrator session ID
3. Test: spawn worker, verify parentID appears in session API response
4. Test cross-project: spawn worker in different project, verify hierarchy works

**Things to watch out for:**
- ⚠️ Cross-project parentID behavior - OpenCode sessions are project-scoped, unclear if parentID works across project boundaries
- ⚠️ Orphaned sessions - if orchestrator session deleted, what happens to children?
- ⚠️ Session.children() performance - may scan all sessions in project, could be slow with hundreds of sessions
- ⚠️ UI implications - OpenCode UI may show session trees, verify it displays well

**Areas needing further investigation:**
- How does Session.children() scale? Is it O(N) scan or indexed lookup?
- Can parentID reference session in different project directory?
- What happens to child sessions when parent is deleted? (orphaned? cascade delete?)
- Does OpenCode UI have tree view for session hierarchy?
- ACP integration: would using ACP client instead of HTTP API provide benefits?

**Success criteria (if implementing Option B):**
- ✅ Spawned worker sessions have parentID set to orchestrator session ID
- ✅ Session.children(orchestratorID) returns worker sessions
- ✅ Cross-project spawning still works (or is explicitly unsupported)
- ✅ Registry continues to work as primary source of truth

---

## References

**OpenCode Source Files Examined:**
- ~/Documents/personal/opencode/packages/opencode/src/agent/agent.ts - Agent.Info schema, mode enum, built-in agents (build, plan, general, explore)
- ~/Documents/personal/opencode/packages/opencode/src/session/index.ts - Session.Info schema, parentID field, create/fork/children functions
- ~/Documents/personal/opencode/packages/opencode/src/tool/task.ts - Task tool implementation, subagent spawning with parentID
- ~/Documents/personal/opencode/packages/opencode/src/acp/agent.ts - ACP implementation, forkSession/resumeSession/listSessions
- ~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts - Session API routes, x-opencode-env-ORCH_WORKER header reading
- ~/Documents/personal/opencode/packages/opencode/src/tool/task.txt - Task tool description (what agents see)

**orch-go Source Files Examined:**
- pkg/opencode/client.go - OpenCode HTTP client, x-opencode-env-ORCH_WORKER header setting
- cmd/orch/spawn_cmd.go - Spawn command, ORCH_WORKER=1 environment variable
- pkg/tmux/tmux.go - Tmux spawn mode, ORCH_WORKER environment handling

**Commands Run:**
```bash
# Verify OpenCode is running and sessions have metadata.role
curl -s http://127.0.0.1:4096/session | head -5

# Search for parentID usage in OpenCode source
grep -r "parentID\|fork" ~/Documents/personal/opencode/packages/opencode/src --include="*.ts" -n | head -20

# Search for ORCH_WORKER in orch-go source  
grep -r "ORCH_WORKER\|x-opencode-env" /Users/dylanconlin/Documents/personal/orch-go --include="*.go" -n
```

**Related Artifacts:**
- **Guide:** .kb/guides/opencode.md - Current OpenCode integration guide (session lifecycle, API endpoints)
- **Guide:** .kb/guides/spawn.md - How spawn works in orch-go
- **Model:** .kb/models/agent-lifecycle-state-model.md - Agent lifecycle tracking

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

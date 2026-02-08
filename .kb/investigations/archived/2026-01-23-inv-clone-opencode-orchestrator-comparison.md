<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** opencode-orchestrator is an OpenCode plugin (TypeScript/Rust) implementing multi-agent orchestration within a single OpenCode instance; orch-go manages agents externally via the OpenCode HTTP API.

**Evidence:** Examined ~30k lines of source code (26k TS, 4k Rust); found Commander/Planner/Worker/Reviewer agent hierarchy, OpenCode plugin hooks integration, in-process session management, and a parallel agent manager.

**Knowledge:** Two fundamentally different architectures - plugin-based (internal) vs CLI/API-based (external) - with different tradeoffs: plugin provides tighter integration but couples to OpenCode; CLI provides independence but requires separate process management.

**Next:** No action needed - this is comparative research. Specific features worth considering for orch-go: hierarchical memory system, todo.md-based task tracking, Write-Ahead Logging for task recovery.

**Promote to Decision:** recommend-no (research comparison, no architectural decision needed)

---

# Investigation: opencode-orchestrator vs orch-go Comparison

**Question:** What does opencode-orchestrator do, how does it work, and what can we learn from it compared to orch-go?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Dylan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Architecture - Plugin vs External CLI

**Evidence:** opencode-orchestrator is an OpenCode plugin (`@opencode-ai/plugin`) that runs inside the OpenCode process. It uses plugin hooks (`chat.message`, `tool.execute.before`, `tool.execute.after`, `assistant.done`, `system.transform`) to intercept and modify OpenCode behavior. orch-go is an external CLI that communicates with OpenCode via HTTP API.

**Source:**
- `~/Documents/personal/opencode-orchestrator/src/index.ts` (plugin definition with hooks)
- `~/Documents/personal/opencode-orchestrator/package.json` (dependencies: `@opencode-ai/plugin`, `@opencode-ai/sdk`)
- `~/Documents/personal/orch-go/pkg/opencode/client.go` (HTTP API client)

**Significance:** Fundamental architectural difference. Plugin approach allows deeper integration (system prompt modification, tool interception) but couples to OpenCode. CLI approach provides independence (works with OpenCode or could work with other APIs) but requires separate process management and can't intercept internal behavior.

---

### Finding 2: Multi-Agent Hierarchy - 4 Fixed Roles

**Evidence:** opencode-orchestrator uses 4 fixed agent roles:
- **Commander** - Master orchestrator, manages parallel sessions, handles mission loops
- **Planner** - Architect/researcher, generates todo.md roadmaps
- **Worker** - Implementer, writes code and tests
- **Reviewer** - Quality gate, verifies work before completion

Agents have explicit system prompts with modular fragments (CORE_PHILOSOPHY, TODO_RULES, etc.). Worker and Reviewer are "terminal nodes" (depth >= 2) that cannot spawn further agents to prevent infinite recursion.

**Source:**
- `~/Documents/personal/opencode-orchestrator/src/agents/definitions.ts` (agent registry)
- `~/Documents/personal/opencode-orchestrator/src/agents/commander.ts` (commander system prompt composition)
- `~/Documents/personal/opencode-orchestrator/src/tools/parallel/delegate-task.ts:269-278` (terminal node guard)

**Significance:** orch-go uses a skill-based system with more flexibility (any skill can be spawned). opencode-orchestrator's fixed roles provide clearer separation of concerns but less flexibility. The terminal node guard is elegant recursion prevention.

---

### Finding 3: Task Delegation - In-Process Sessions vs Spawned Processes

**Evidence:** opencode-orchestrator creates "child sessions" within the same OpenCode instance using `client.session.create()`. Tasks can run in sync (blocking with polling) or background (async) modes. orch-go spawns entirely new OpenCode processes (via `opencode run` CLI) or creates sessions via HTTP API.

The `delegate_task` tool supports:
- `background=true` - Non-blocking, returns task ID
- `background=false` - Blocking, polls until completion
- `resume` - Continue existing session with preserved context
- `mode` - "normal", "race" (first wins), "fractal" (recursive)

**Source:**
- `~/Documents/personal/opencode-orchestrator/src/tools/parallel/delegate-task.ts`
- `~/Documents/personal/opencode-orchestrator/src/core/agents/manager.ts` (ParallelAgentManager)

**Significance:** In-process session management allows tighter coordination (parent can poll children, easier context sharing) but all agents share one process failure domain. orch-go's external spawns provide better isolation but require more coordination infrastructure.

---

### Finding 4: Persistence Layer - Self-Healing Rehydration

**Evidence:** State is persisted to `.opencode/` directory:
- `todo.md` - Hierarchical task roadmap
- `context.md` - Global project knowledge
- `mission_loop.json` - Low-level engine state (iteration counts, session IDs)
- `work-log.md` - Audit trail
- `sync-issues.md` - Failure registry
- `archive/tasks/` - Write-Ahead Logs (WAL) for task recovery

On startup, ParallelAgentManager reads the WAL and recovers active tasks: `recoverActiveTasks()`.

**Source:**
- `~/Documents/personal/opencode-orchestrator/docs/SYSTEM_ARCHITECTURE.md` (section 3)
- `~/Documents/personal/opencode-orchestrator/src/core/agents/manager.ts:316-364`

**Significance:** orch-go uses a registry file (`~/.orch/registry.json`) but lacks WAL-based recovery. The `.opencode/` directory approach with structured files is more sophisticated. The todo.md-centric task tracking is interesting.

---

### Finding 5: Memory System - Hierarchical 4-Tier

**Evidence:** MemoryManager implements 4 memory tiers:
- **SYSTEM** (2k tokens) - Core philosophy, immutable
- **PROJECT** (5k tokens) - Working directory, discovered patterns
- **MISSION** (10k tokens) - Current mission context
- **TASK** (20k tokens) - Short-term, cleared per task

Memory is pruned by importance score (0.0-1.0) and timestamp when budget exceeded. Simple keyword-based relevance filtering for context retrieval.

**Source:** `~/Documents/personal/opencode-orchestrator/src/core/memory/memory-manager.ts`

**Significance:** orch-go doesn't have an in-process memory system - context is managed via SPAWN_CONTEXT.md and CLAUDE.md files. The hierarchical memory with token budgets is interesting for long-running missions but adds complexity. orch-go's file-based approach is simpler but less adaptive.

---

### Finding 6: Hook System - Feature Modularization

**Evidence:** 9 hooks registered on startup:
- **Pre-tool:** StrictRoleGuardHook, MetricsHook
- **Post-tool:** SanityCheckHook, SecretScannerHook, AgentUIHook, ResourceControlHook, MemoryGateHook, MetricsHook
- **Chat:** UserActivityHook, MissionControlHook
- **Done:** SanityCheckHook, MissionControlHook, ResourceControlHook, MemoryGateHook, MetricsHook

Notable: `StrictRoleGuardHook` enforces role-based permissions (e.g., Workers can't do destructive operations). `SecretScannerHook` catches credential leaks.

**Source:** `~/Documents/personal/opencode-orchestrator/src/hooks/index.ts`

**Significance:** orch-go's spawn context system provides similar capability (skills with constraints) but at spawn time, not runtime. Runtime hooks can catch violations dynamically. Could be valuable for real-time safety enforcement.

---

### Finding 7: Rust Core - Performance Tools

**Evidence:** Rust crate (`orchestrator-core`) provides high-performance tools:
- grep with timeout/resource limits
- glob file search
- sed replacement
- AST tools
- LSP integration
- diff/git tools
- HTTP client

These are compiled to a binary and called from TypeScript via command execution.

**Source:** `~/Documents/personal/opencode-orchestrator/crates/orchestrator-core/src/tools/`

**Significance:** orch-go shells out to external tools (rg, fd, etc.). Building native tools provides more control (timeouts, resource limits) but requires maintenance of a Rust codebase. The timeout protection in grep.rs:66-79 is well-implemented.

---

### Finding 8: Project Activity - Actively Developed

**Evidence:**
- Latest commit: 2026-01-24 00:44:08 KST (hours ago)
- npm version: 1.0.76
- ~30k lines of code (26k TypeScript, 4k Rust)
- 521 source files

Frequent releases (v1.0.64 through v1.0.76 in recent commits), suggesting active iteration.

**Source:** `git log --oneline -20` in the repo

**Significance:** This is an active project, not abandoned. Worth monitoring for ideas and potential collaboration.

---

## Synthesis

**Key Insights:**

1. **Different architectural philosophies** - opencode-orchestrator embraces tight OpenCode integration via plugin hooks, while orch-go maintains independence via external CLI. Neither is wrong - they solve different problems. Plugin approach is better for single-instance power users; CLI approach is better for fleet management and multi-instance orchestration.

2. **Fixed roles vs flexible skills** - The 4-agent hierarchy (Commander/Planner/Worker/Reviewer) provides clear separation but less flexibility than orch-go's skill system. However, their terminal node guard (depth-based recursion prevention) is elegant and we've implemented something similar.

3. **State management approaches differ** - opencode-orchestrator uses `.opencode/` with WAL for crash recovery and todo.md for task tracking. orch-go uses registry + beads for state. Their approach is more self-contained; ours is more distributed (beads is a separate system).

**Answer to Investigation Question:**

opencode-orchestrator is a fundamentally different approach to agent orchestration - it works FROM INSIDE OpenCode as a plugin, while orch-go works FROM OUTSIDE via HTTP API.

**What it does:** Provides multi-agent task delegation within a single OpenCode instance using Commander/Planner/Worker/Reviewer roles, with sophisticated memory management, task persistence, and hook-based safety enforcement.

**How it works:** Installs as an npm package that registers as an OpenCode plugin. When users run `/task "..."`, the Commander agent plans the work, delegates to Workers in parallel sessions, and Reviewers verify before completion. State persists to `.opencode/` directory.

**What we can learn:**
1. **Write-Ahead Logging** for task recovery is more robust than our current registry approach
2. **Todo.md-centric task tracking** provides good visibility - similar to our beads but file-based
3. **StrictRoleGuardHook** runtime permission enforcement is more dynamic than our spawn-time constraints
4. **Hierarchical memory with token budgets** is interesting for long sessions but adds complexity
5. **Terminal node guard** for recursion prevention is clean - we have similar via depth tracking

---

## Structured Uncertainty

**What's tested:**

- ✅ Cloned and explored the repository structure (verified: files exist)
- ✅ Identified architecture as OpenCode plugin (verified: package.json dependencies, src/index.ts plugin interface)
- ✅ Confirmed 4-agent hierarchy (verified: read definitions.ts, agent prompts)
- ✅ Confirmed active development (verified: git log shows commits today)

**What's untested:**

- ⚠️ Actual runtime behavior (not installed or run)
- ⚠️ Performance characteristics of Rust tools vs shelling out
- ⚠️ Memory system effectiveness in practice
- ⚠️ Recovery behavior after crash

**What would change this:**

- Running the plugin in a real OpenCode instance would reveal runtime behaviors not visible in code
- Comparing token usage between their memory approach and our context files would quantify tradeoffs
- Load testing would reveal if plugin architecture has overhead vs external CLI

---

## Implementation Recommendations

**Purpose:** No implementation needed - this is comparative research. However, features worth considering for future orch-go work:

### Features Worth Considering

**1. Write-Ahead Logging for Agent Recovery**
- Their `taskWAL` provides crash recovery for running tasks
- Our registry doesn't survive daemon restarts cleanly
- Could adapt similar pattern for `~/.orch/wal/` directory

**2. Todo.md-centric Task Display**
- Their todo.md provides human-readable task state
- We have beads but it's in SQLite, not as immediately visible
- Could add `orch status --markdown` output mode

**3. Runtime Hook System**
- Their hooks catch violations dynamically (secrets, role violations)
- We only enforce at spawn time via context
- Could add coaching/intervention layer similar to opencode plugins

### Not Worth Adopting

**1. Plugin Architecture**
- Couples too tightly to OpenCode internals
- We want independence for multi-instance orchestration
- Our HTTP API approach is more portable

**2. Fixed 4-Agent Roles**
- Our skill system is more flexible
- Their approach is simpler but less extensible
- Skills > roles for our use case

**3. In-Process Memory Manager**
- Adds complexity for marginal benefit
- Our file-based context (SPAWN_CONTEXT.md) is simpler
- Claude already handles context well

---

## References

**Files Examined:**
- `~/Documents/personal/opencode-orchestrator/README.md` - Project overview
- `~/Documents/personal/opencode-orchestrator/package.json` - Dependencies and scripts
- `~/Documents/personal/opencode-orchestrator/Cargo.toml` - Rust workspace config
- `~/Documents/personal/opencode-orchestrator/src/index.ts` - Plugin entry point
- `~/Documents/personal/opencode-orchestrator/src/agents/commander.ts` - Commander agent
- `~/Documents/personal/opencode-orchestrator/src/agents/definitions.ts` - Agent registry
- `~/Documents/personal/opencode-orchestrator/src/tools/parallel/delegate-task.ts` - Task delegation
- `~/Documents/personal/opencode-orchestrator/src/core/agents/manager.ts` - Parallel agent manager
- `~/Documents/personal/opencode-orchestrator/src/core/memory/memory-manager.ts` - Memory system
- `~/Documents/personal/opencode-orchestrator/src/hooks/index.ts` - Hook registration
- `~/Documents/personal/opencode-orchestrator/docs/SYSTEM_ARCHITECTURE.md` - Architecture docs
- `~/Documents/personal/opencode-orchestrator/crates/orchestrator-core/src/tools/grep.rs` - Rust grep tool

**Commands Run:**
```bash
# Clone repository
git clone https://github.com/agnusdei1207/opencode-orchestrator ~/Documents/personal/opencode-orchestrator

# Check commit history
git log --oneline -20

# Count source files and lines
find . -name "*.ts" -o -name "*.rs" | grep -v node_modules | wc -l
find . -name "*.ts" -not -path "*/node_modules/*" -exec wc -l {} + | tail -1
```

**External Documentation:**
- GitHub: https://github.com/agnusdei1207/opencode-orchestrator
- npm: https://www.npmjs.com/package/opencode-orchestrator

---

## Investigation History

**2026-01-23 18:08:** Investigation started
- Initial question: Compare opencode-orchestrator to orch-go
- Context: Spawned from orchestrator to research competing/complementary approaches

**2026-01-23 18:30:** Investigation completed
- Status: Complete
- Key outcome: opencode-orchestrator is a plugin-based approach (inside OpenCode) vs orch-go's CLI approach (outside OpenCode) - different tradeoffs, some good ideas to consider (WAL, todo.md, runtime hooks)

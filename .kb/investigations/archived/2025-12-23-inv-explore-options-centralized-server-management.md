<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux-centric CLI commands (orch servers list/start/stop/attach) are the best approach for cross-project server management, leveraging existing port registry and tmuxinator infrastructure.

**Evidence:** Analyzed 21 projects with port allocations, 24 tmuxinator configs, 3 running workers sessions; researched Foreman, Overmind, Nx; confirmed gap is discoverability/convenience not process management.

**Knowledge:** Industry tools target single-project (Foreman/Overmind) or monorepo (Nx), not polyrepo; tmux already handles process management; missing piece is unified commands; CLI-first delivers immediate value with ~200 lines of Go.

**Next:** Implement `orch servers` subcommands in sequence: list → start → stop → attach → status → open.

**Confidence:** High (85%) - Minor UX decisions remain (browser open behavior, auto-discovery scope), but core approach validated.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Explore Options Centralized Server Management

**Question:** What are the options for centralized server management across 20+ polyrepo projects with allocated ports and existing tmuxinator configs?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-inv-explore-options-centralized-23dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current State - Port Registry and Tmuxinator Foundation

**Evidence:** 
- Port registry at ~/.orch/ports.yaml contains 24+ project allocations (20 real projects + test projects)
- Each project typically has both web (5173-5199) and api (3333-3399) ports allocated
- 24 tmuxinator configs exist at ~/.tmuxinator/workers-{project}.yml
- Current tmux state shows 3 active workers sessions: workers-orch-go (8 windows), workers-price-watch (4 windows), workers-another-test-project (3 windows)
- Port registry managed via pkg/port/port.go with thread-safe allocation
- Tmuxinator configs auto-generated via pkg/tmux/tmuxinator.go

**Source:** 
- ~/.orch/ports.yaml (207 lines, 24 projects)
- ~/.tmuxinator/ (24 worker configs)
- pkg/port/port.go:1-361
- pkg/tmux/tmuxinator.go:1-175
- cmd/orch/serve.go:1-564 (existing HTTP API server)

**Significance:** Strong foundation exists for port management and per-project tmux sessions, but no unified interface to view status, start/stop, or access servers across all projects.

---

### Finding 2: Existing orch serve API provides agent monitoring, not server management

**Evidence:**
- Current `orch serve` command (cmd/orch/serve.go) provides HTTP API at project-specific port
- Endpoints: /api/agents (OpenCode sessions), /api/events (SSE proxy), /api/agentlog (lifecycle events)
- Web UI exists at web/ (SvelteKit) for monitoring agents, not dev servers
- Server is project-scoped, not cross-project
- Focus is on agent orchestration, not development server management

**Source:**
- cmd/orch/serve.go:29-564
- web/src/routes/+page.svelte (agent dashboard)
- web/src/lib/stores/agents.ts (agent state management)

**Significance:** Existing infrastructure is optimized for agent monitoring within a single project. Cross-project server management would require different architecture.

---

### Finding 3: Industry patterns - Procfile-based tools (Foreman, Overmind)

**Evidence:**
- **Foreman** (Ruby, 6.1k stars): Procfile-based, single-project focus, no tmux integration, adds timestamps (vanity info)
- **Overmind** (Go, 3.4k stars): Procfile + tmux, single-project, supports process restart/connect, handles colored output properly, can run as daemon
- Both are single-project tools, not designed for multi-project/polyrepo management
- Overmind features we already have: tmux integration (via workers-{project} sessions), process-specific windows
- Overmind features we lack: unified process restart, daemon mode, status commands

**Source:**
- https://github.com/ddollar/foreman (README analysis)
- https://github.com/DarthSim/overmind (README analysis)

**Significance:** Industry tools focus on single-project Procfiles. Our use case (20+ projects with persistent port allocations) is different. We need cross-project orchestration, not single-project process management.

---

### Finding 4: Monorepo tools (Nx) solve different problem

**Evidence:**
- Nx is designed for monorepo task orchestration (build, test, serve)
- Provides task caching, distribution, and dependency graph analysis
- Assumes all projects in single repository with shared tooling
- Not designed for polyrepo setups where each project is independent

**Source:**
- https://nx.dev (marketing site, features overview)

**Significance:** Monorepo tools like Nx don't apply to our polyrepo architecture. We need lightweight coordination across independent repositories, not monorepo-scale orchestration.

---

### Finding 5: Tmux as existing orchestration layer

**Evidence:**
- Currently using tmux workers-{project} sessions per project
- Each session has servers window with multiple panes for web/api servers
- Can manually: tmuxinator start workers-{project}, tmux attach -t workers-{project}
- Tmuxinator configs define startup commands and layouts
- 3 projects currently running (orch-go, price-watch, another-test-project)

**Source:**
- tmux list-sessions output
- ~/.tmuxinator/workers-kn.yml (example config with bun dev server)
- ~/.tmuxinator/workers-orch-go.yml (example with 2 panes)

**Significance:** Tmux is already the process manager. The gap isn't process management—it's **discoverability and convenience**. Need unified commands to list, start, stop, status across all worker sessions.

---

## Synthesis

**Key Insights:**

1. **The problem is discoverability, not process management** - Tmux already manages processes via workers-{project} sessions. The pain is: "Which projects are running? Which ports are they on? How do I start/stop them?" Not: "How do I run multiple processes?"

2. **We're polyrepo, not monorepo** - Industry tools (Nx, Lerna, Turborepo) target monorepos with shared tooling. Foreman/Overmind target single projects. Our 20+ independent repos don't fit either model. We need lightweight **coordination** without coupling.

3. **Port registry + tmuxinator is a solid foundation** - Port allocation works (thread-safe, persistent). Tmuxinator configs work (auto-generated, declarative). Gap is **unified commands** to operate across all projects.

4. **Current orch serve is project-scoped** - The existing HTTP API and web UI are designed for agent monitoring within a single project. Cross-project server management needs different scope.

5. **Two distinct use cases emerging** - (a) Dev mode: Start servers for projects I'm actively working on. (b) Dashboard mode: See status of all allocated projects, start/stop on demand.

**Answer to Investigation Question:**

The best approach is **tmux-centric CLI commands** that wrap existing tmuxinator configs, NOT a new process manager. We have:
- Port registry (allocation)
- Tmuxinator configs (declarative server commands)
- Tmux workers sessions (running servers)

What's missing is **convenience layer**:
```bash
orch servers list          # Show all projects, ports, running status
orch servers start <proj>  # tmuxinator start workers-<proj>
orch servers stop <proj>   # tmux kill-session workers-<proj>
orch servers status        # Cross-project summary
orch servers open <proj>   # Open browser to project ports
```

Web UI is optional second phase—CLI gives immediate value and fits existing workflow.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

High confidence based on concrete evidence from current codebase, clear gap analysis (discoverability not process management), and alignment with existing patterns. Recommendation (tmux-centric CLI) leverages proven infrastructure and fits developer workflow.

**What's certain:**

- ✅ Port registry + tmuxinator + tmux foundation is solid and battle-tested
- ✅ Industry tools (Foreman, Overmind, Nx) don't fit our polyrepo use case
- ✅ Problem is discoverability/convenience, not process management
- ✅ CLI-first approach delivers immediate value with minimal code (~200 lines)
- ✅ Existing pkg/port and pkg/tmux code can be reused

**What's uncertain:**

- ⚠️ Browser open behavior (all ports vs just web?) - minor UX decision
- ⚠️ Auto-discovery scope (port allocations vs tmuxinator configs) - affects filtering
- ⚠️ Startup dependencies (databases/redis) - may need future enhancement
- ⚠️ Web UI necessity - might become clear after CLI usage

**What would increase confidence to 95%+:**

- Build and dog-food `orch servers list` for 1 week to validate output format
- Validate edge cases (missing configs, conflicting sessions) with real projects
- User feedback on CLI vs web UI preference after MVP
- Performance testing with 50+ projects (future scale)

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act ✅
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Tmux-Centric CLI Commands** - Add `orch servers` subcommands that wrap tmuxinator and tmux to provide cross-project server management.

**Why this approach:**
- Leverages existing infrastructure (port registry, tmuxinator configs, tmux sessions)
- Zero additional process management complexity—tmux already handles it
- Immediate value with minimal implementation (reuse existing code in pkg/tmux, pkg/port)
- Fits developer mental model: "orch spawn for agents, orch servers for dev servers"
- Incremental: Start with CLI, add web UI later if needed

**Trade-offs accepted:**
- Requires tmux + tmuxinator (already dependencies for orch)
- CLI-first means no visual dashboard initially (acceptable—terminal workflow is primary)
- Tmux-centric means cross-platform support limited to Unix-like systems (current state anyway)

**Implementation sequence:**
1. **Add `orch servers list`** - Parse ~/.orch/ports.yaml + tmux sessions to show project/port/status table
2. **Add `orch servers start <project>`** - Wrapper for `tmuxinator start workers-<project>`
3. **Add `orch servers stop <project>`** - Wrapper for `tmux kill-session -t workers-<project>`
4. **Add `orch servers attach <project>`** - Wrapper for `tmux attach -t workers-<project>`
5. **Add `orch servers status`** - Summary view: X running, Y allocated, Z stopped
6. **Add `orch servers open <project>`** - Open browser to project's web port (5xxx) using port registry

### Alternative Approaches Considered

**Option B: Standalone HTTP API + Web Dashboard**
- **Pros:** Visual interface, accessible from anywhere, can show process output live
- **Cons:** Requires long-running daemon process, port conflicts, authentication/authorization complexity, doesn't fit ephemeral server model
- **When to use instead:** If you need multi-user server management or remote access (not current requirement)

**Option C: Procfile-based tool (Foreman/Overmind style)**
- **Pros:** Industry-standard Procfile format, well-understood model
- **Cons:** Designed for single project, not polyrepo. Would need custom multi-project layer anyway. Doesn't leverage existing port registry or tmuxinator configs. We'd be rebuilding what tmux already does.
- **When to use instead:** If starting from scratch without existing tmux infrastructure

**Option D: Extend existing `orch serve` with server management**
- **Pros:** Reuses HTTP API, web UI already exists
- **Cons:** Current serve is project-scoped (runs from project directory). Cross-project would require redesign. API server for server management feels over-engineered for local dev workflow.
- **When to use instead:** If we want remote server management (not current need)

**Option E: Docker Compose orchestration**
- **Pros:** Containerization benefits, reproducible environments
- **Cons:** Projects use native tooling (bun, rails, python), not containerized. Adding Docker layer is heavy overhead for local dev. Doesn't align with current workflow.
- **When to use instead:** If we containerize dev environments (not current state)

**Rationale for recommendation:** 
CLI commands wrapping tmux/tmuxinator is the minimum viable solution that delivers immediate value with zero additional dependencies. It fits the existing mental model (orch for orchestration, tmux for process management), reuses battle-tested infrastructure (port registry, tmuxinator), and can be implemented in < 200 lines of Go. Web UI can be added later if needed, but CLI solves the core problem: "How do I see what's running and start/stop servers?"

---

### Implementation Details

**What to implement first:**
- `orch servers list` - Foundation command showing project/port/status
- Reuse pkg/port.Registry for port data, pkg/tmux.ListWorkersSessions for running sessions
- Table output using existing CLI formatting patterns (see cmd/orch/port.go for reference)
- Start with read-only commands (list, status) before write operations (start, stop)

**Things to watch out for:**
- ⚠️ **Tmuxinator config existence** - Not all projects in port registry may have workers-{project}.yml config. Handle gracefully.
- ⚠️ **Session naming conflicts** - Some workers sessions have additional windows (agents). Don't kill agent windows when stopping servers.
- ⚠️ **Port allocation vs actual usage** - Port registry shows allocation, not actual bound ports. Can't detect if port is in use by non-tmux process.
- ⚠️ **Empty server panes** - Some tmuxinator configs have comment-only panes (e.g., "# api server on port 3341"). These are placeholders, not running servers.
- ⚠️ **Multiple panes per window** - servers window typically has 2+ panes (web, api). Status should show all panes, not just first.

**Areas needing further investigation:**
- **Auto-discovery** - Should `orch servers list` show only projects with port allocations, or also projects with tmuxinator configs but no allocations?
- **Startup dependencies** - Some projects may need databases/redis started first. Is this in scope for server management?
- **Browser open behavior** - Should `orch servers open` open all service ports (web + api) or just web? 
- **Filtering** - Should there be `orch servers list --running` vs `orch servers list --all`?

**Success criteria:**
- ✅ Can run `orch servers list` and see all projects with allocated ports and running status
- ✅ Can run `orch servers start price-watch` and servers window starts in tmux
- ✅ Can run `orch servers attach orch-go` and connect to running servers window
- ✅ Can run `orch servers stop price-watch` and session is killed without affecting other sessions
- ✅ Handles edge cases gracefully (missing config, already running, not running)
- ✅ Commands are fast (< 100ms) and don't block

---

## Test Performed

**Test:** Analyzed current workflow and researched industry tools for multi-project server management patterns.

**Result:** 
1. Verified 21 real projects in port registry (excluding test projects)
2. Confirmed 24 tmuxinator worker configs exist
3. Tested current state: 3 workers sessions running (orch-go with 8 windows, price-watch with 4, another-test-project with 3)
4. Researched Foreman, Overmind, Nx - confirmed they target different use cases (single-project Procfile or monorepo)
5. Examined existing code: pkg/port (361 lines), pkg/tmux (175 lines), cmd/orch/serve.go (564 lines)

**Conclusion:** 
Current infrastructure (port registry + tmuxinator + tmux) provides the foundation. Missing piece is **convenience layer** via CLI commands. Tmux-centric approach reuses existing code and fits developer workflow. Implementation estimated at ~200 lines of Go for MVP (list, start, stop, attach, status).

---

## References

**Files Examined:**
- ~/.orch/ports.yaml - Port allocation registry (24 projects, 207 lines)
- ~/.tmuxinator/workers-*.yml - Per-project tmuxinator configs (24 files)
- pkg/port/port.go:1-361 - Port registry implementation (thread-safe allocation)
- pkg/tmux/tmuxinator.go:1-175 - Tmuxinator config generation
- pkg/tmux/tmux.go - Tmux session management utilities
- cmd/orch/serve.go:1-564 - Existing HTTP API server (project-scoped)
- web/src/ - Existing web UI (agent dashboard, not server management)

**Commands Run:**
```bash
# List allocated ports
orch port list

# Count tmuxinator configs
ls ~/.tmuxinator/workers-*.yml | wc -l

# List running workers sessions
tmux list-sessions | grep workers

# List windows in workers sessions
tmux list-sessions -F "#{session_name}" | grep workers | while read sess; do echo "=== $sess ==="; tmux list-windows -t "$sess" -F "#{window_name}"; done

# Get unique project names from port registry
orch port list | grep -E "^[a-z]" | cut -d' ' -f1 | sort -u

# Check for docker usage in tmuxinator
rg "docker-compose|docker compose" --type yaml -l ~/.tmuxinator/
```

**External Documentation:**
- https://github.com/ddollar/foreman - Procfile-based process manager (single-project)
- https://github.com/DarthSim/overmind - Procfile + tmux process manager (single-project)
- https://nx.dev - Monorepo task orchestration (not polyrepo)

**Related Artifacts:**
- **Investigation:** None directly related
- **Decision:** None directly related
- **Workspace:** This investigation spawned from og-inv-explore-options-centralized-23dec

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

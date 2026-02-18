TASK: Comprehensive orch-go codebase audit. PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go. SESSION SCOPE: Large. PRIOR AUDIT: .kb/investigations/2026-01-03-audit-comprehensive-orch-go-bugs-reliability-architecture.md (6 weeks old, significant changes since). FOCUS AREAS: (1) Architecture coherence after registry removal and lifecycle ownership refactor, (2) Dead code from registry elimination - consumers fully migrated?, (3) Code quality - bloated files needing extraction (spawn_cmd.go 2320, session.go 2166, doctor.go 1912, complete_cmd.go 1669, status_cmd.go 1625, serve_agents.go 1560), (4) ~3,400 lines of lifecycle code that Phase 5 fork integration will eliminate - identify exactly what can be removed, (5) Test coverage gaps especially in daemon, completion, and status subsystems, (6) Error handling consistency, (7) Performance - SSE parsing overhead, dashboard query efficiency, (8) Security - auth token handling, session management. RECENT CHANGES: pkg/registry/ deleted, clean_cmd.go simplified 7→3 flags, ghost/phantom/orphan vocabulary eliminated, lifecycle state model updated with state vs infrastructure distinction. KEY DECISIONS: .kb/decisions/2026-02-14-lifecycle-ownership-own-accept-build.md, .kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md. KEY MODELS: .kb/models/agent-lifecycle-state-model.md, .kb/models/dashboard-agent-status.md, .kb/models/spawn-architecture.md. DELIVERABLE: Investigation in .kb/investigations/ with actionable findings, prioritized by impact. Create beads issues for any P0/P1 findings.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "comprehensive orch codebase"

### Constraints (MUST respect)
- orch tail tmux fallback requires either current registry window ID OR beads ID in window name format [beads-id]
  - Reason: Dual-dependency failure causes fallback to fail when both are stale/missing
- orch complete must verify SYNTHESIS.md exists and is not placeholder before closing issue
  - Reason: 70% of agents completed without synthesis in 24h chaos period
- orch init must be idempotent - safe to run multiple times
  - Reason: Prevents accidental overwrites and enables 'run init to update' pattern
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- kb context command hangs on some queries
  - Reason: Blocks orch spawn from returning, use --skip-artifact-check as workaround

### Prior Decisions
- orch-go tmux spawn is fire-and-forget - no session ID capture
  - Reason: opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring.
- Registry updates must happen before beads close in orch complete
  - Reason: Prevents inconsistent state where beads shows closed but registry shows active
- orch spawn context delivery is reliable
  - Reason: Verified that SPAWN_CONTEXT.md is correctly populated and accessible by the agent
- orch-go CLI independence
  - Reason: CLI commands connect directly to OpenCode (4096), not orch serve (3333)
- Multi-agent synthesis relies on workspace isolation + SYNTHESIS.md + orch review
  - Reason: 100 commits, 52 synthesis files, 0 conflicts validates current architecture
- Beads OSS: Clean Slate over Fork
  - Reason: Local features (ai-help, health, tree) not used by orch ecosystem. Drop rather than maintain.
- skillc and orch build skills are complementary, not competing
  - Reason: skillc compiles project-local .skillc/ to CLAUDE.md; orch build skills compiles templated skills to ~/.claude/skills/. Different purposes, both needed.
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- Pre-spawn kb context should filter to orch ecosystem repos
  - Reason: 33% of global results are noise from unrelated repos (price-watch, dotfiles). Filtering preserves cross-repo signal while eliminating noise.
- orch complete auto-closes tmux window after successful verification
  - Reason: Complete means done - window goes away, beads closes, workspace remains. Prevents phantom accumulation (41 windows today). Debugging escape hatch: don't complete until ready to close.

### Models (synthesized understanding)
- Escape Hatch Visibility Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/escape-hatch-visibility-architecture.md
  - Summary:
    **Core insight:** The architectural choice of dual-window Ghostty setup isn't just "nice to have" - it's a **required component** of escape-hatch spawning architecture.
    
    ```
    Critical Infrastructure Work
      → Requires Escape Hatch (independence + visibility + capability)
        → Visibility Requires --tmux Flag
          → --tmux Requires Dual-Window Setup
            → Dual-Window Requires Auto-Switch Hook
    ```
    
    Remove any link in this chain and the visibility criterion fails.
  - Your findings should confirm, contradict, or extend the claims above.
- Agent Lifecycle State Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model.md
  - Summary:
    Agent state exists across **four independent layers** (tmux windows, OpenCode in-memory, OpenCode on-disk, beads comments). These layers fall into two distinct categories: **state layers** (beads, workspace files) that represent what work was done, and **infrastructure layers** (OpenCode sessions, tmux windows) that represent transient execution resources. The dashboard reconciles these via a **Priority Cascade**: check beads issue status first (highest authority), then Phase comments, then registry state, then session existence. Status can appear "wrong" at the dashboard level while being "correct" at each individual layer - this is a measurement artifact from combining multiple sources of truth.
    
    ---
  - Critical Invariants:
    1. **Phase: Complete is agent's declaration** - Only agent can reach this, not orchestrator
    2. **Beads issue closed = canonical completion** - All status queries defer to beads
    3. **Session existence ≠ agent still working** - Sessions persist indefinitely
    4. **Status checks don't mutate state** - Calculation is read-only, no side effects
    5. **Multiple sources must be reconciled** - No single source has complete truth
    6. **Tmux windows are UI layer only** - Not authoritative for state
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Dashboard Shows "Active" When Agent is Done
    
    **Symptom:** Dashboard shows agent as active, but `bd show <id>` says status=closed
    
    **Root cause:** Dashboard caching or SSE lag - hasn't received beads update yet
    
    **Why it happens:**
    - Agent reaches Phase: Complete
    - `orch complete` closes beads issue
    - Beads issue status = closed
    - Dashboard hasn't refreshed or polled beads yet
    - Dashboard still shows cached "active" state
    
    **Fix:** Refresh dashboard browser tab (forces beads query)
    
    **NOT the fix:** Deleting OpenCode session (treats symptom, not cause)
    
    ### Failure Mode 2: "Dead" Agents That Actually Completed
    
    **Symptom:** Dashboard shows "dead", but work is done and beads issue closed
    
    **Root cause:** Session cleanup happened before dashboard queried, cascade reached session check
    
    **Why it happens:**
    - Agent completed, beads issue closed
    - Session cleanup ran (manual or automatic)
    - Dashboard cascade: beads check → no issue (closed) → session check → no session → "dead"
    
    **Fix (Jan 8):** Priority Cascade puts beads/Phase check before session existence check
    
    ### Failure Mode 3: Agent Went Idle But Not Complete
    
    **Symptom:** Session status is "idle" but no `Phase: Complete` comment
    
    **Root cause:** Agent ran out of context, crashed, or didn't follow completion protocol
    
    **Why it happens:**
    - Session exhausts context (150k tokens)
    - Agent stops responding
    - SSE event: `session.status = idle`
    - No `Phase: Complete` was ever written
    - Dashboard shows "idle" or "waiting"
    
    **This is expected behavior.** Session idle ≠ work complete. Only agents that explicitly run `bd comment <id> "Phase: Complete"` are considered done.
    
    **Fix:** Check workspace for what agent accomplished, then either:
    - `orch complete <id> --force` if work is done
    - `orch abandon <id>` if work is incomplete
    
    ### Failure Mode 4: Cross-Project Agents Not Visible
    
    **Symptom:** Agent spawned with `--workdir /other/project` doesn't appear in dashboard
    
    **Root cause:** Dashboard only scans current project's `.orch/workspace/` directory
    
    **Why it happens:**
    - Workspace created in `/other/project/.orch/workspace/`
    - Dashboard running from `orch-go` only sees `orch-go/.orch/workspace/`
    - Cross-project discovery requires querying OpenCode sessions for unique directories
    
    **Fix (Jan 6):** Multi-project workspace cache built from OpenCode session directories
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md
  - Summary:
    Anthropic restricts Opus 4.5 access via fingerprinting that blocks API usage but allows Claude Code CLI with Max subscription. This constraint forced a **dual spawn architecture**: primary path (OpenCode API + Sonnet/Flash, headless, high concurrency) and escape hatch (Claude CLI + Opus, tmux, crash-resistant). The escape hatch exists because critical infrastructure work (fixing the spawn system itself) can't depend on what might fail. Model choice now encodes reliability requirements, not just quality preferences.
    
    ---
  - Critical Invariants:
    1. **Never spawn OpenCode infrastructure work without --backend claude --tmux**
       - Violation: Agent kills itself mid-execution when server restarts
    
    2. **Infrastructure detection runs before model auto-selection**
       - Priority 2.5 (between explicit flags and model-based selection)
       - Ensures auto-apply happens even without explicit --backend
    
    3. **Opus only accessible via Claude CLI backend**
       - API requests to Opus fail with auth error
       - Fingerprinting checks more than headers (TLS, HTTP/2 frames, ordering)
    
    4. **Escape hatch provides true independence**
       - Claude CLI binary ≠ OpenCode server
       - Tmux session persists across service restarts
       - Different authentication path (Max subscription OAuth)
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Zombie Agents (Jan 8, 2026)
    
    **Symptom:** Agents tracked in registry but never actually ran
    
    **Root Cause:** Spawning with `--model opus` before understanding auth gate
    - orch created registry entry
    - OpenCode session created
    - Anthropic rejected API request (fingerprinting)
    - Agent hung in "running" state
    - Consumed concurrency slot without doing work
    
    **Examples:**
    - orch-go-mo0ja, orch-go-pzi2i, orch-go-aoei0, orch-go-gd1gd, orch-go-lwc3o
    
    **Fix:** Never use `--model opus` without `--backend claude`
    
    ### Failure Mode 2: Header Injection Conflicts (Jan 8, 2026)
    
    **Symptom:** Gemini Flash spawns hung after attempting Opus bypass
    
    **Root Cause:** Injected Claude Code headers (`x-app: cli`, `anthropic-version`, etc.) into OpenCode's Anthropic provider
    - Bypassed Opus gate (didn't work)
    - Broke Gemini spawns (headers conflicted with Bun fetch/SDK)
    - System-wide impact from localized change
    
    **Lesson:** Fingerprinting is more sophisticated than headers alone
    
    ### Failure Mode 3: Infrastructure Work Kills Itself
    
    **Symptom:** Agent fixing OpenCode server crashes mid-execution
    
    **Root Cause:** Agent spawned via OpenCode API, agent's fix restarts OpenCode server, agent's session killed
    
    **Solution:** Infrastructure work detection auto-applies `--backend claude --tmux`
    
    **Why auto-detection matters:**
    - Humans forget to add flags manually
    - Task description might not mention "opencode" explicitly
    - Keyword scan catches common patterns
    - Escape hatch becomes invisible safety net
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Probe: Inventory all friction gates across spawn, completion, and daemon — assess defect-catching vs noise
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
  - Recent Probes:
    - 2026-02-13-friction-gate-inventory-all-subsystems
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
    - 2026-02-09-friction-bypass-analysis-post-targeted-skips
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md
- Orchestrator Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle.md
  - Summary:
    Orchestrator sessions operate in a **three-tier hierarchy** (meta-orchestrator → orchestrator → worker) where each level is completed by the level above. Orchestrators produce **SESSION_HANDOFF.md** (not SYNTHESIS.md) and **wait** for completion (not /exit). They track via **session registry** (not beads) because orchestrators manage conversations, not work items. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed. Checkpoint discipline uses duration thresholds (2h/3h/4h) as a proxy for context exhaustion.
    
    ---
  - Why This Fails:
    ### 1. Frame Collapse (Orchestrator → Worker)
    
    **What happens:** Orchestrator drops into worker-level implementation (editing code, debugging, investigating).
    
    **Root cause:** Vague goals → exploration mode → investigation → debugging. **Framing cues override skill instructions.**
    
    **Why detection is hard:** Orchestrators can't self-diagnose frame collapse. The frame defines what's visible, so from inside the collapsed frame, the behavior feels appropriate.
    
    **Detection signals:**
    - Edit tool usage on code files (not orchestration artifacts)
    - Time spent >15 minutes on direct fixes
    - SESSION_HANDOFF.md shows "Manual fixes" sections
    - Post-mortem reveals work that should have been spawned
    
    **NOT the fix:** Adding more ABSOLUTE DELEGATION RULE warnings. The agent already knows. The problem is framing, not awareness.
    
    **Prevention:**
    1. Provide specific goals with action verbs, concrete deliverables, success criteria
    2. Use WHICH vs HOW test: meta decides WHICH focus, orchestrator decides HOW to execute
    3. Frame collapse check in SESSION_HANDOFF.md template
    4. Potential: OpenCode plugin tracking Edit usage on code vs artifacts
    
    **Trigger pattern:** Failure-to-implementation. After agents fail, orchestrator tries to "just fix it" instead of trying different spawn strategy.
    
    ### 2. Self-Termination Attempts
    
    **What happens:** Spawned orchestrator tries to run `orch session end` or `/exit` instead of waiting for completion.
    
    **Root cause:** ORCHESTRATOR_CONTEXT.md template contradicted the hierarchical completion model (told orchestrator to self-terminate).
    
    **Why it's wrong:** Breaks the "completed by level above" invariant. Orchestrator can't verify its own work from meta perspective.
    
    **Fix:** Template updated Jan 2026 to instruct "write SESSION_HANDOFF.md and WAIT".
    
    ### 3. Session Registry Drift
    
    **What happens:** `~/.orch/sessions.json` shows status "active" for completed sessions.
    
    **Root cause:** `orch complete` didn't update session registry status (only closed beads issues or removed registry entries).
    
    **Why it matters:** Stale active sessions accumulate, `orch status` shows ghost sessions, registry becomes unreliable.
    
    **Fix:** `orch complete` now updates status to "completed", `orch abandon` updates to "abandoned" - sessions preserved for history, not removed.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Dashboard Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture.md
  - Summary:
    The Swarm Dashboard is a Svelte 5 web UI served by `orch serve` (Go backend) that provides real-time monitoring of agent status, daemon health, and operational metrics.
    
    **Critical context (Option A+):** The dashboard is Dylan's (meta-orchestrator's) ONLY observability layer. He does not use CLI tools directly. Dashboard failure = Dylan is blind. This makes dashboard reliability tier-0 infrastructure. See orchestrator skill "Observability Architecture (Option A+)" section.
    
    The architecture uses a **two-mode design** (Operational/Historical) to separate daily coordination from deep analysis. SSE connections enable real-time updates but are constrained by HTTP/1.1's 6-connection limit. Progressive disclosure and stable sorting prevent information overload while maintaining scan-ability.
    
    ---
  - Critical Invariants:
    1. **Two-mode design is mutually exclusive** - Cannot show both Operational and Historical views simultaneously
    2. **SSE Events auto-connect, Agentlog is opt-in** - Connection pool management
    3. **beadsFetchThreshold controls remote queries** - 5+ ready issues triggers `bd ready` shell-out
    4. **Progressive disclosure via collapsed panels** - Event panels start collapsed, expand on click
    5. **Stable sort maintains scan-ability** - Agent order doesn't change unless status changes
    6. **Early filtering reduces payload size** - Backend filters before sending to frontend
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Connection Pool Exhaustion
    
    **Symptom:** API fetches hang or timeout when SSE panels open
    
    **Root cause:** HTTP/1.1 allows only 6 connections per origin; SSE occupies slots
    
    **Why it happens:**
    - Events SSE (auto-connect): 1 slot
    - Agentlog SSE (auto-connect before fix): 1 slot
    - Remaining 4 slots for API fetches
    - If 5+ API requests concurrent, some block
    
    **Fix (Jan 5):** Made Agentlog SSE opt-in via Follow button, freeing 1 slot
    
    ### Failure Mode 2: Slow Dashboard Load with 100+ Agents
    
    **Symptom:** Dashboard takes 5-10 seconds to load with many agents
    
    **Root cause:** `/api/agents` endpoint performs expensive operations (OpenCode queries, beads parsing) synchronously
    
    **Why it happens:**
    - Each agent requires OpenCode session query
    - Full beads issue parsing for each agent
    - No caching, recomputed on every request
    
    **Fix (Jan 6):** Response caching with 2-second TTL, reduced load time to <1 second
    
    ### Failure Mode 3: Information Overload in Operational Mode
    
    **Symptom:** Users overwhelmed by full swarm map with 50+ agents
    
    **Root cause:** Single view tried to serve both daily coordination and deep analysis
    
    **Why it happens:**
    - Operational needs: "What's ready? What's broken?"
    - Historical needs: "Show me everything, all filters, full archive"
    - One view can't optimize for both
    
    **Fix (Jan 7):** Two-mode design - Operational (focused) vs Historical (comprehensive)
    
    ### Failure Mode 4: Plugin Cascade (Dashboard "Disconnected" Despite Services Running)
    
    **Symptom:** Dashboard shows "disconnected", `overmind status` shows all 3 services running, but `orch status` returns HTTP 500
    
    **Root cause:** OpenCode plugin error (e.g., v1→v2 API incompatibility) crashes OpenCode's internal request handling
    
    **Why it happens:**
    - OpenCode loads plugins at startup
    - Bad plugin throws error on every request
    - `/api/agents` calls OpenCode → gets 500
    - Dashboard can't fetch agent data → shows "disconnected"
    - overmind sees process running (not crashed) → reports "running"
    
    **Cascade:**
    ```
    Plugin error → OpenCode internal 500 → orch status fails → API can't get agents → Dashboard "disconnected"
    ```
    
    **Fix (Jan 14):** Disable plugins, restart OpenCode, re-enable one-by-one. Root cause was session-resume.js using v1 API (object export) instead of v2 (function export).
    
    **Key insight:** Dashboard can appear "down" while all processes are technically "running". Health checks must verify data flow, not just port availability.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Spawn Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture.md
  - Summary:
    Spawn evolved through 5 phases from basic CLI integration to daemon-driven automation with triage friction. The architecture creates a workspace with SPAWN_CONTEXT.md embedding skill content + task description + kb context, then launches an OpenCode session. The tier system (light/full) determines whether SYNTHESIS.md is required at completion. Triage friction (`--bypass-triage` flag) intentionally makes manual spawns harder to encourage daemon-driven workflow.
    
    ---
  - Critical Invariants:
    1. **Workspace name = kebab-case task description** - Used for tmux window, directory name, session title
    2. **Beads ID required for phase reporting** - `--no-track` creates untracked IDs that can't report to beads
    3. **KB context uses --global flag** - Cross-repo constraints are essential
    4. **Skill content stripped for --no-track** - Beads instructions removed when not tracking
    5. **Session scoping is per-project** - `orch send` only works within same directory hash
    6. **Token estimation at 4 chars/token** - Warning at 100k, error at 150k
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Cross-Project Spawn Sets Wrong Session Directory
    
    **Symptom:** `orch spawn --workdir /other/project` creates session with orchestrator's directory
    
    **Root cause:** Session directory is set from spawn caller's CWD, not `--workdir` target
    
    **Why it happens:**
    - OpenCode infers directory from process CWD
    - `--workdir` changes agent's working directory, not spawn process CWD
    - Session gets orchestrator directory, beads issue in orchestrator project
    
    **Impact:**
    - Sessions unfindable via directory filtering
    - Cross-project work tracking is split
    
    **Fix needed:** Pass explicit directory to OpenCode session creation
    
    ### Failure Mode 2: Token Limit Exceeded on Spawn
    
    **Symptom:** Spawn fails with "context too large" error
    
    **Root cause:** SPAWN_CONTEXT.md exceeds 150k token limit
    
    **Why it happens:**
    - Skill content (~10-40k tokens)
    - KB context can be large (30-50k tokens)
    - Task description minimal
    - Estimation: 4 chars/token
    
    **Fix (Dec 22):** Warning at 100k tokens, hard error at 150k with guidance
    
    ### Failure Mode 3: Daemon Spawns Blocked Issues
    
    **Symptom:** Daemon spawns issue that has blockers
    
    **Root cause:** Dependency checking missing in triage workflow
    
    **Why it happens:**
    - `bd ready` returns issues without blockers
    - Daemon spawns from `triage:ready` label (doesn't check dependencies)
    - Race condition: issue labeled before dependencies checked
    
    **Fix (Jan 3):** Dependency gating with `--force` override flag
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Beads Integration Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
  - Summary:
    Beads integration uses **RPC-first with CLI fallback** pattern: try native Go RPC client (fast, no process spawn), fall back to CLI subprocess if daemon unavailable. The integration operates at **three points in agent lifecycle**: spawn (create issue), work (report phase via comments), complete (close with reason). **Auto-tracking** creates issues automatically unless `--no-track` flag set. The RPC client lives in **pkg/beads** (never shell out with `exec.Command` directly) and provides 10x performance improvement over CLI (single RPC call vs subprocess spawn + JSON parse).
    
    ---
  - Why This Fails:
    ### 1. RPC Client Unavailable
    
    **What happens:** RPC calls fail, client falls back to CLI, performance degrades.
    
    **Root cause:** Beads daemon not running. RPC socket `~/.beads/daemon.sock` doesn't exist.
    
    **Why detection is hard:** Fallback is silent. No warning that RPC failed. User sees slow performance, doesn't know why.
    
    **Fix:** Start beads daemon: `bd daemon start` or ensure launchd starts it on boot.
    
    **Detection:** Log RPC failures, surface in `orch doctor` health check.
    
    ### 2. Beads ID Not Found
    
    **What happens:** `orch complete orch-go-abc1` fails with "issue not found".
    
    **Root cause:** Cross-project spawn. Issue created in orch-knowledge, but trying to complete from orch-go. Beads scoped to current directory's `.beads/`.
    
    **Why detection is hard:** Beads ID looks valid (correct format), but doesn't exist in current project's `.beads/issues.jsonl`.
    
    **Fix:** `cd` into correct project before completion, or use `--workdir` flag.
    
    **Prevention:** `orch complete` should detect project from workspace, auto-cd.
    
    ### 3. Auto-Tracking Creates Duplicates
    
    **What happens:** `orch spawn` creates issue, but issue already exists for same work.
    
    **Root cause:** User creates issue manually, then spawns with auto-tracking. Both create issue.
    
    **Why detection is hard:** No deduplication. Beads doesn't check if similar issue exists.
    
    **Fix:** Use `--issue <id>` flag to reference existing issue instead of auto-creating.
    
    **Prevention:** Better UX: `orch spawn` could check for related issues and prompt user.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-09-bd-sync-safe-timeout-retry-for-hash-mismatch.md
    - 2026-02-09-bd-sync-safe-post-sync-readiness-check
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-09-bd-sync-safe-post-sync-readiness-check.md
    - 2026-02-08-synthesis-dedup-parse-error-fail-closed
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture/probes/2026-02-08-synthesis-dedup-parse-error-fail-closed.md
- Daemon Autonomous Operation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation.md
  - Summary:
    The daemon is an **autonomous agent spawner** that operates in a **poll-spawn-complete cycle**: polls beads for `triage:ready` issues, infers skill from issue type, spawns within capacity limits, monitors for `Phase: Complete`, verifies and closes. The daemon operates **independently of orchestrators** - orchestrators triage (label issues ready), daemon spawns (batch processing), orchestrators synthesize (review completed work). Skill inference uses **issue type** (not labels) to map task→investigation, bug→systematic-debugging, etc. Capacity management uses **WorkerPool** with reconciliation against OpenCode to free stale slots.
    
    ---
  - Why This Fails:
    ### 1. Capacity Starvation
    
    **What happens:** Pool shows MaxAgents active, but `orch status` shows fewer actual agents running.
    
    **Root cause:** Spawn failures don't release slots. Agent spawned, counted against pool, but spawn fails (bad skill name, missing context, etc.) - slot never released.
    
    **Why detection is hard:** Pool only knows about attempts, not outcomes. No feedback loop from spawn failure to pool.
    
    **Fix:** Reconciliation with OpenCode. Query actual sessions, release slots for non-existent agents.
    
    **Prevention:** Spawn tracking with retry limits (`pkg/daemon/spawn_tracker.go`).
    
    ### 2. Duplicate Spawns
    
    **What happens:** Same issue spawned multiple times by daemon on consecutive polls.
    
    **Root cause:** Spawn latency. Issue labeled `triage:ready` at poll N, daemon spawns, but spawn hasn't transitioned issue to `in_progress` by poll N+1. Daemon sees same issue still ready, spawns again.
    
    **Why detection is hard:** Race condition between poll interval (60s) and spawn transition time (variable).
    
    **Fix:** Spawn deduplication via tracking. Track spawned beads IDs in memory, skip on subsequent polls until status confirms transition.
    
    **Source:** `pkg/daemon/spawn_tracker.go`
    
    ### 3. Skill Inference Mismatch
    
    **What happens:** Daemon spawns wrong skill for issue type.
    
    **Root cause:** Issue type doesn't match actual work needed. User creates `task` for implementation work (should be `feature`), daemon infers `investigation`.
    
    **Why detection is hard:** Type is set at creation, can't be changed by daemon. Daemon trusts type.
    
    **Fix:** Manual override. Orchestrator updates issue type before labeling `triage:ready`, or spawns manually with correct skill.
    
    **Prevention:** Better issue creation prompts, type validation, skill override via labels (future).
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-09-dashboard-restart-daemon-autostart-default-disabled
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-09-dashboard-restart-daemon-autostart-default-disabled.md
- Dashboard Agent Status Calculation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-agent-status.md
  - Summary:
    The dashboard calculates agent status through a **Priority Cascade model**: check Phase first (highest priority), then registry state, then session existence. Status can be "wrong" at the dashboard level while being "correct" at each individual check - this is a measurement artifact, not a system failure. The root issue was observation infrastructure (missing events, double-counting metrics, state not surfaced), not broken completion logic.
    
    ---
  - Critical Invariants:
    1. **Phase: Complete is agent's declaration** - Only agent can reach this, not orchestrator
    2. **Registry is source of truth for abandonment** - Human judgment, can't be inferred
    3. **Session may outlive completion** - Session existence ≠ agent still working
    4. **Status checks don't mutate state** - Calculation is read-only, no side effects
    
    ---
  - Why This Fails:
    ### Failure Mode 1: "Dead" Agents That Completed
    
    **Symptom:** Dashboard shows "dead", but work is done and beads issue closed
    
    **Root cause:** Session cleanup happens async, dashboard checks session existence as fallback
    
    **Why it happens:**
    - Agent reaches Phase: Complete
    - `orch complete` verifies and closes beads issue
    - Session cleanup happens later (or not at all)
    - Dashboard cascade reaches session check → sees no session → "dead"
    
    **Fix (Jan 8):** Priority Cascade puts Phase check before session check
    
    ### Failure Mode 2: Metrics Show Wrong Completion Rate
    
    **Symptom:** `orch stats` showed 72% completion when reality was 89%
    
    **Root cause:** Metrics counted events (double-counting) instead of deduplicating entities
    
    **Why it happens:**
    - `agent.completed` event emitted by both `orch complete` AND beads close hook
    - Metrics counted events → some completions counted 2x
    - Result: inflated total, deflated completion %
    
    **Fix (Jan 8):** Metrics deduplicate by beads_id before calculating ratios
    
    ### Failure Mode 3: Work Completed via Bypass Paths
    
    **Symptom:** Beads issue closed but no completion event, dashboard doesn't update
    
    **Root cause:** `bd close` (direct) doesn't emit events, only `orch complete` does
    
    **Why it happens:**
    - Multiple paths to completion: `orch complete`, `bd close`, `bd sync` with commit message
    - Only `orch complete` emits events
    - Other paths are invisible to observation infrastructure
    
    **Fix (Jan 6):** Beads close hook emits `agent.completed` event
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.

### Guides (procedural knowledge)
- Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/status.md
- orch CLI Reference
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/cli.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Background Services Performance Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/background-services-performance.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Workspace Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/workspace-lifecycle.md
- Session Resume Protocol
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/session-resume-protocol.md

### Related Investigations
- Model Provider Architecture - orch vs OpenCode Auth Responsibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-model-provider-architecture-orch-vs.md
- Dashboard Port Confusion Orch Serve
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-dashboard-port-confusion-orch-serve.md
- Is agentlog init ready to integrate into orch init?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-agentlog-init-ready-integrate-into.md
- Orch Ecosystem Artifact Audit Against Skillc Design Principles
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-orch-ecosystem-artifact-audit-against.md
- Config-as-Code Design for Orch Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-design-config-code-orch-ecosystem.md
- Workers Attempting Restart Orch Servers
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-inv-workers-attempting-restart-orch-servers.md
- Glass Integration Status in Orch Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md
- Orch Serve Cache Not Invalidated
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-04-inv-orch-serve-cache-not-invalidated.md
- Shared Browser Experience Orch Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-shared-browser-experience-orch-ecosystem.md
- orch init and Project Standardization
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-scope-orch-init-project-standardization.md

### Failed Attempts (DO NOT repeat)
- debugging Insufficient Balance error when orch usage showed 99% remaining
- orch tail on tmux agent
- orch clean to remove ghost sessions automatically

### Open Questions
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-17x "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.

🚨 SESSION COMPLETE PROTOCOL (READ NOW, DO AT END):
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment orch-go-17x "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.


CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours


AUTHORITY:
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Surface it first: `bd comment orch-go-17x "CONSTRAINT: [what constraint] - [why considering workaround]"`
2. Wait for orchestrator acknowledgment before proceeding
3. The accountability is a feature, not a cost

This applies to:
- System constraints discovered during work (e.g., API limits, tool limitations)
- Architectural patterns that seem inconvenient for your task
- Process requirements that feel like overhead
- Prior decisions (from `kb context`) that conflict with your approach

**Why:** Working around constraints without surfacing them:
- Prevents the system from learning about recurring friction
- Bypasses stakeholders who should know about the limitation
- Creates hidden technical debt

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation comprehensive-orch-go-codebase-audit` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-comprehensive-orch-go-codebase-audit.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-17x "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-comprehensive-orch-go-14feb-7ad8/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input


## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-17x**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-17x "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-17x "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-17x "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-17x "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-17x "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-17x`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (codebase-audit)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 33eab9180803 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-13 23:15:22 -->


## Summary

**Purpose:** Common protocols shared by all worker skills. This is inherited by worker skills via dependencies.

---

# Worker Base Patterns

**Purpose:** Common protocols shared by all worker skills. This is inherited by worker skills via dependencies.

**What this provides:**
- Authority delegation (what you can decide vs escalate)
- Hard limits (constitutional constraints that override all authority)
- Constitutional objection protocol (how to raise ethical concerns)
- Beads progress tracking (how to report via bd comment)
- Phase reporting (how to signal transitions)
- Exit/completion protocol (how to properly end a session)

---



## Authority Delegation

**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

---



## Hard Limits (Constitutional)

**These limits override ALL authority - orchestrator, user, or otherwise.**

Workers CANNOT do these regardless of instruction:

| Hard Limit | Constitutional Basis |
|------------|---------------------|
| Generate malware, exploits, or attack tools | Claude doesn't create weapons |
| Implement deceptive UI patterns (dark patterns) | Claude doesn't manipulate users |
| Build surveillance without consent disclosure | User autonomy and transparency |
| Intentionally bypass authentication/authorization | System integrity |
| Create content designed to deceive | Honesty as near-constraint |
| Automate harassment or mass targeting | Avoiding harm |
| Implement discriminatory logic | Ethical AI principles |

**When instructed to violate a hard limit:**

1. **Document** - `bd comment <id> "HARD LIMIT: [limit] - Cannot proceed with [specific instruction]"`
2. **Do NOT proceed** - No partial implementation, no "just this once"
3. **Continue other work** - If task has separable components, complete those
4. **Wait for human** - This bypasses orchestrator; only human can review

**Why these are non-negotiable:** Claude's constitution establishes these as near-inviolable constraints. Orchestrators are Claude instances too - they cannot authorize violations. Only human judgment can evaluate edge cases.

**Common false positives (these are usually OK):**
- Security testing tools for authorized pentesting
- Analytics with proper consent disclosure
- Authentication code (building it, not bypassing it)
- Competitive analysis (observation, not deception)

---



## Constitutional Objection Protocol

**Trigger:** You believe an instruction conflicts with constitutional values (safety, ethics, honesty, user wellbeing) but it's not a clear Hard Limit violation.

**This is DIFFERENT from operational escalation:**

| Type | Examples | Route |
|------|----------|-------|
| **Operational** | "I'm blocked", "Requirements unclear", "Need decision" | → Orchestrator |
| **Constitutional** | "This could harm users", "This feels deceptive", "Ethical concern" | → Human (bypasses orchestrator) |

**Protocol when you have a constitutional concern:**

1. **Identify the value** - Which constitutional principle is at risk? (safety, honesty, user autonomy, avoiding harm)

2. **Document it** - `bd comment <id> "CONSTITUTIONAL CONCERN: [value] - [specific concern]"`

3. **Do NOT proceed** with the concerning component

4. **Continue** with unrelated components if the task is separable

5. **Wait for HUMAN review** - Do not accept orchestrator override on constitutional matters

**Why this bypasses orchestrator:**

Claude's constitution says Claude can refuse unethical instructions regardless of the principal hierarchy. Orchestrators are Claude instances - they cannot authorize constitutional violations any more than you can. Human judgment is required for genuine ethical edge cases.

**Examples:**

| Situation | Response |
|-----------|----------|
| "Add tracking pixel without disclosure" | CONSTITUTIONAL CONCERN: user autonomy - undisclosed tracking |
| "Make the unsubscribe button hard to find" | CONSTITUTIONAL CONCERN: honesty - dark pattern design |
| "Scrape competitor's user data" | CONSTITUTIONAL CONCERN: ethics - unauthorized data collection |
| "Build feature that targets vulnerable users" | CONSTITUTIONAL CONCERN: avoiding harm - exploitation risk |

**When it's NOT a constitutional concern:**
- Technical disagreements about implementation
- Preference for different architecture
- Belief that requirements are suboptimal
- Wanting more context before proceeding

These are operational - escalate to orchestrator normally.

---



## Progress Tracking

**Use `bd comment` for phase transitions and progress updates.**

```bash
# Report progress at phase transitions
bd comment {{.BeadsID}} "Phase: Planning - Analyzing codebase structure"
bd comment {{.BeadsID}} "Phase: Implementing - Adding authentication middleware"
bd comment {{.BeadsID}} "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment {{.BeadsID}} "Phase: BLOCKED - Need clarification on API contract"

# Report questions
bd comment {{.BeadsID}} "Phase: QUESTION - Should we use JWT or session-based auth?"
```

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Additional context:**
Use `bd comment` for additional context, findings, or updates:
```bash
bd comment {{.BeadsID}} "Found performance bottleneck in database query"
bd comment {{.BeadsID}} "investigation_path: .kb/investigations/2026-02-11-perf-issue.md"
```

**Test Evidence Requirement:**
When reporting Phase: Complete, include test results in the summary:
- Example: `bd comment {{.BeadsID}} "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed (2.3s)"`
- Example: `bd comment {{.BeadsID}} "Phase: Complete - Tests: npm test - 23 specs, 0 failures"`
- Example: `bd comment {{.BeadsID}} "Phase: Complete - Tests: make test - PASS (coverage: 78%)"`

**Why:** `orch complete` validates test evidence in phase comments. Vague claims like "all tests pass" trigger manual verification.

**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`.
- Workers report `Phase: Complete`, orchestrator verifies and closes
- Running `bd close` bypasses verification and breaks tracking

---



## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment {{.BeadsID}} "Phase: Planning - [brief description]"`
2. Read relevant codebase context for your task
3. Begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors phase reporting.

**Status Updates:**
Update Status: field in your workspace/investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed)

**Signal orchestrator when blocked:**
- Add `**Status:** BLOCKED - [reason]` to workspace
- Add `**Status:** QUESTION - [question]` when needing input

---



## Discovered Work (Mandatory)

**Before marking your session complete, review for discovered work.**

During any session, you may encounter:
- **Bugs** - Broken behavior not related to your current task
- **Tech debt** - Code that should be refactored but is out of scope
- **Enhancements** - Ideas for improvements noticed while working
- **Questions** - Strategic unknowns needing orchestrator input

### Checklist

Before completing your session:

- [ ] Reviewed for discovered work (bugs, tech debt, enhancements, questions)
- [ ] Created issues via `bd create` OR noted "No discovered work" in completion comment

### Creating Issues

```bash
# For bugs found
bd create "description of bug" --type bug -l triage:review

# For tech debt or refactoring needs
bd create "description" --type task -l triage:review

# For feature ideas or enhancements
bd create "description" --type feature -l triage:review

# For strategic questions needing decision
bd create "description" --type question -l triage:review
```

### Reporting

In your `Phase: Complete` comment, include either:
- List of issues created: `Created: orch-go-XXXXX, orch-go-YYYYY`
- Or: `No discovered work`

**Why this matters:** Discovered work that isn't tracked gets lost. The next session has no visibility into bugs or opportunities you found. Creating issues ensures nothing falls through the cracks.

### Cross-Repo Issue Handoff

**When you discover an issue that belongs to a different repo**, you cannot create it directly — `bd create` only works in the current project directory, and shell sandboxing prevents `cd` to other repos.

**Instead, output a structured `CROSS_REPO_ISSUE` block** in your beads completion comment or SYNTHESIS.md. The orchestrator will pick this up during completion review and create the issue in the target repo.

**Format:**
```
CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/<target-repo>
  title: "<concise issue title>"
  type: bug|task|feature|question
  priority: 0-4
  description: "<1-3 sentences with context, evidence, and why it matters>"
```

**Rules:**
- Use absolute or `~`-relative paths for `repo`
- Include enough context in `description` for the issue to stand alone (the orchestrator in the other repo won't have your session context)
- One block per issue — multiple issues get multiple blocks
- Report blocks in your `Phase: Complete` comment: `Cross-repo: 1 CROSS_REPO_ISSUE block for price-watch`

**Example:**
```bash
bd comment <beads-id> "Phase: Complete - Implemented token refresh. Cross-repo: 1 CROSS_REPO_ISSUE block below.

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/price-watch
  title: Fix ScsOauthClient concurrent token refresh
  type: bug
  priority: 2
  description: During orch-go token handling work, discovered price-watch ScsOauthClient has a race condition when multiple goroutines call RefreshToken simultaneously. No mutex protects the shared token state."
```

---



## Session Complete Protocol

**When your work is done (all deliverables ready), complete in this EXACT order:**

{{if eq .Tier "light"}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. Commit any final changes (including `VERIFICATION_SPEC.yaml`)
4. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. Ensure SYNTHESIS.md is created (including the `Verification Contract` section linking `VERIFICATION_SPEC.yaml` and key outcomes)
4. Commit all changes (including SYNTHESIS.md and `VERIFICATION_SPEC.yaml`)
5. Run: `/exit` to close the agent session
{{end}}

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

---






---
name: codebase-audit
skill-type: procedure
description: Systematic codebase audit with configurable dimension (security/performance/tests/architecture/organizational/quick)
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 4d69600b0ab7 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/codebase-audit/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/src/worker/codebase-audit/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/codebase-audit/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-06 15:35:56 -->


## Summary

**Use when the user says:**
- "Audit [focus area]" (security, performance, tests, architecture, organizational)
- "Run codebase health check"
- "Find [category] issues in the codebase"
- "Quick scan the codebase"

---

# Codebase Audit

<!-- SKILL-TEMPLATE: common-overview -->
<!-- Auto-generated from phases/common-overview.md -->

## When to Use This Skill

**Use when the user says:**
- "Audit [focus area]" (security, performance, tests, architecture, organizational)
- "Run codebase health check"
- "Find [category] issues in the codebase"
- "Quick scan the codebase"

**Auto-detect dimension from context:**
- "Security vulnerabilities" → security dimension
- "Performance bottlenecks" → performance dimension
- "Test coverage" → tests dimension
- "God objects" / "tight coupling" → architecture dimension
- "ROADMAP drift" / "template drift" → organizational dimension
- "Quick health check" → quick dimension

---

## Skill Overview

This skill performs systematic codebase audits with configurable dimensions. Each dimension focuses on a specific area and produces an investigation file with findings, evidence, and actionable recommendations.

**Core workflow:**
1. **Pattern Search** - Automated searches for known issues
2. **Evidence Collection** - Concrete examples with file paths/line numbers
3. **Analysis** - Root cause identification and severity assessment
4. **Documentation** - Investigation file with prioritized recommendations

**Key deliverables:**
- Investigation file at `.kb/investigations/YYYY-MM-DD-audit-{dimension}.md`
- Progress tracked via `bd comment <beads-id> "Phase: [current phase] - [progress details]"`

---

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| **Primary** (authoritative) | Actual code, test output, observed behavior | This IS the evidence |
| **Secondary** (claims to verify) | Workspaces, investigations, decisions | Hypotheses to test |

When an artifact says "X is not implemented," that's a hypothesis—not a finding to report. Search the codebase before concluding.

**The failure mode:** An audit reads a workspace claiming "feature X NOT DONE" and reports that as a finding without checking if feature X actually exists in the code. Always verify artifact claims against primary sources.

---

## Investigation File Setup

**CRITICAL:** Before starting the audit, create investigation file from template. This ensures all findings are documented progressively with proper metadata (including Resolution-Status field for synthesis workflow).

### Create Investigation Template

```bash
# Create investigation using kb CLI command
# Update SLUG based on your audit dimension and topic
# Use audit/ prefix for audit investigations
kb create investigation "audit/dimension-audit-description"
```

**After creating the template:**
1. Fill Question field with specific audit focus from SPAWN_CONTEXT
2. Update metadata (Started date set automatically, verify Status)
3. Document findings progressively during audit (don't wait until end)
4. Update Confidence and Resolution-Status when completing audit

**Important:**
- The `kb create investigation` command auto-detects project directory and creates the investigation in the appropriate subdirectory.
- The investigation file includes Resolution-Status field (Unresolved/Resolved/Recurring/Synthesized/Mitigated) which is critical for the synthesis workflow. Always fill this field when completing the investigation.

**Now proceed with dimension-specific audit guidance below.**

---

## Available Dimensions

### Focused Audits (30-90 min)

**security** - Security vulnerabilities, unsafe patterns, secrets exposure, OWASP compliance
- When: Investigating security risks, penetration test prep, compliance audit
- Output: Security findings with severity ratings (Critical/High/Medium/Low)

**performance** - Performance bottlenecks, N+1 queries, algorithmic complexity, slow operations
- When: App is slow, high resource usage, scaling issues
- Output: Performance findings with profiling data and optimization recommendations

**tests** - Test coverage gaps, flaky tests, missing test types, test quality
- When: Flaky builds, low confidence in tests, missing edge case coverage
- Output: Testing gaps with risk assessment and coverage metrics

**architecture** - Coupling, god objects, missing abstractions, modularity issues
- When: Hard to add features, tight coupling, unclear boundaries
- Output: Architectural issues with refactoring effort estimates

**organizational** - ROADMAP drift, template drift, documentation sync, process violations
- When: Docs out of date, ROADMAP showing completed work as TODO, templates inconsistent
- Output: Organizational drift findings with system amnesia analysis

### Quick Scan (1 hour)

**quick** - Automated pattern search across all focus areas, high-priority issues only
- When: Need rapid health check before major work, onboarding to new codebase
- Output: Top 10 findings across all categories with quick-win recommendations

---

## Common Patterns

**Full audit workflow (2-4 hours):**
1. Run `quick` dimension to identify top issues
2. Run focused dimension for high-priority areas
3. Synthesize findings into single investigation file
4. Prioritize using ROI framework (impact vs effort)

**Targeted audit workflow (30-90 min):**
1. Run single focused dimension (user knows the problem area)
2. Investigation file documents findings
3. Add high-priority items to ROADMAP

<!-- /SKILL-TEMPLATE -->

---

<!-- MODE-SPECIFIC CONTENT -->
<!-- Use --parallel flag for comprehensive multi-agent audits -->

<!-- SKILL-TEMPLATE: mode-parallel -->
<!-- Auto-generated from phases/mode-parallel.md -->

# Parallel Execution Mode

**TLDR:** Use 5 parallel Haiku agents (one per dimension) for 3x faster comprehensive audits. Each agent runs pattern searches and returns JSON findings, which a synthesis agent combines into a prioritized report.

**When to use:** Comprehensive audit needed across multiple dimensions, time-constrained review, full codebase health check before major work.

**Output:** Single investigation file with prioritized findings from all dimensions.

---

## Architecture

```
┌─────────────────┐
│  Orchestrator   │ (spawns all agents in single message)
└────────┬────────┘
         │
    ┌────┴────┬────────┬────────┬────────┐
    ▼         ▼        ▼        ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│Security│ │Perf   │ │Arch   │ │Tests  │ │Org    │
│ Agent │ │ Agent │ │ Agent │ │ Agent │ │ Agent │
│(Haiku)│ │(Haiku)│ │(Haiku)│ │(Haiku)│ │(Haiku)│
└───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘
    │         │        │        │        │
    └────┬────┴────────┴────────┴────────┘
         │ (JSON findings)
         ▼
┌─────────────────┐
│  Synthesis      │ (Haiku - prioritizes & formats)
│  Agent          │
└────────┬────────┘
         │ (Prioritized report)
         ▼
┌─────────────────┐
│  Final Output   │
└─────────────────┘
```

---

## Key Design Decisions

1. **Haiku for dimension agents** - Pattern searches are IO-bound (grep/glob), not reasoning-heavy. Haiku is 3x faster and cheaper than Sonnet for this workload.

2. **JSON output from dimension agents** - Structured data enables consistent synthesis across agents.

3. **Separate synthesis step** - Keeps dimension agents focused on discovery; synthesis agent handles prioritization logic.

4. **No confidence scoring** - Unlike code-review (which filters false positives), codebase-audit produces objective pattern matches (file exists at N lines = fact, not opinion).

---

## Workflow

### Step 1: Spawn 5 Parallel Dimension Agents

Use a single message with 5 Task tool invocations to spawn all dimension agents concurrently:

```markdown
**For orchestrators:** Spawn parallel audit agents using:

1. Security Agent (Haiku) - Returns JSON with secrets, injection, auth findings
2. Performance Agent (Haiku) - Returns JSON with large files, complexity, N+1 findings
3. Architecture Agent (Haiku) - Returns JSON with god objects, coupling findings
4. Tests Agent (Haiku) - Returns JSON with coverage gaps, flaky test indicators
5. Organizational Agent (Haiku) - Returns JSON with drift patterns, doc sync findings

Each agent prompt should specify:
- Dimension to audit
- Project directory
- JSON output format requirement
- Pattern search commands to run
```

**Example Task tool invocation (5 in one message):**

```
Task 1: "Audit security dimension of PROJECT_DIR. Run pattern searches for secrets, injection, auth issues. Return JSON: {potential_secrets: N, injection_risks: N, auth_issues: N, findings: [...]}"

Task 2: "Audit performance dimension of PROJECT_DIR. Run pattern searches for large files, complexity, N+1. Return JSON: {large_files: [...], complexity_issues: N, findings: [...]}"

Task 3: "Audit architecture dimension of PROJECT_DIR. Run pattern searches for god objects, coupling. Return JSON: {god_objects: [...], coupling_issues: N, findings: [...]}"

Task 4: "Audit tests dimension of PROJECT_DIR. Run pattern searches for coverage gaps, flaky indicators. Return JSON: {coverage_gaps: N, flaky_tests: N, findings: [...]}"

Task 5: "Audit organizational dimension of PROJECT_DIR. Run pattern searches for drift, doc sync. Return JSON: {roadmap_drift: N, template_drift: N, findings: [...]}"
```

### Step 2: Wait for All Agents to Complete

All 5 agents run concurrently. Wait for all Task results to return.

**Expected latency:** ~5-10 seconds (parallel) vs ~15-30 seconds (sequential)

### Step 3: Spawn Synthesis Agent

Once all dimension agent results are available, spawn a synthesis agent:

```markdown
Task: "Synthesize codebase audit findings from 5 dimension agents.

Security findings: {JSON from agent 1}
Performance findings: {JSON from agent 2}
Architecture findings: {JSON from agent 3}
Tests findings: {JSON from agent 4}
Organizational findings: {JSON from agent 5}

Produce prioritized findings:
1. Combine all findings
2. Assign severity (Critical/High/Medium/Low)
3. Sort by ROI (impact vs effort)
4. Return top 20 findings with recommendations"
```

### Step 4: Write Investigation File

Write synthesis output to investigation file:

```bash
# Investigation file location
.kb/investigations/YYYY-MM-DD-audit-comprehensive-parallel.md
```

---

## Expected Benefits

| Metric | Sequential | Parallel | Improvement |
|--------|------------|----------|-------------|
| Wall-clock time | ~15-30 min | ~5-10 min | **3x faster** |
| Token cost | 1x Sonnet | 5x Haiku + 1x Haiku | ~Equal or cheaper |
| Coverage | Single dimension | All dimensions | **Comprehensive** |

---

## Agent Output Format

Each dimension agent returns structured JSON for synthesis:

**Security Agent:**
```json
{
  "dimension": "security",
  "potential_secrets": 20,
  "injection_risks": 3,
  "auth_issues": 0,
  "findings": [
    {"type": "secret", "file": "config.py", "line": 45, "severity": "high", "description": "Hardcoded API key"},
    {"type": "injection", "file": "api.py", "line": 123, "severity": "critical", "description": "SQL injection risk"}
  ]
}
```

**Architecture Agent:**
```json
{
  "dimension": "architecture",
  "god_objects": [
    {"file": "cli.py", "lines": 4031, "methods": 85},
    {"file": "spawn.py", "lines": 2110, "methods": 42}
  ],
  "coupling_issues": 52,
  "findings": [
    {"type": "god_object", "file": "cli.py", "severity": "high", "description": "4031 lines exceeds 300-line threshold"}
  ]
}
```

---

## Synthesis Output Format

The synthesis agent produces a prioritized report:

```markdown
# Comprehensive Audit: [Project Name]

**Date:** YYYY-MM-DD
**Mode:** Parallel (5 dimension agents + synthesis)
**Duration:** X minutes

## Executive Summary

- **Critical findings:** N
- **High priority:** N
- **Medium priority:** N
- **Total findings:** N

## Prioritized Findings (by ROI)

### 1. [CRITICAL] Security: SQL injection in api.py:123
**Dimension:** Security
**Impact:** High (data breach risk)
**Effort:** Low (parameterized queries)
**Recommendation:** Use parameterized queries instead of string formatting

### 2. [HIGH] Architecture: cli.py at 4031 lines
**Dimension:** Architecture
**Impact:** High (maintainability, testing difficulty)
**Effort:** Medium (extract modules)
**Recommendation:** Extract command handlers to separate modules

### 3-20. [Additional findings...]

## Metrics Baseline

| Dimension | Key Metric | Value |
|-----------|------------|-------|
| Security | Potential secrets | 20 |
| Architecture | Files >300 lines | 3 |
| Tests | Coverage gaps | 15 |
| Performance | N+1 queries | 5 |
| Organizational | ROADMAP drift | 8 |

## Next Steps

1. Address critical findings immediately
2. Schedule high-priority fixes this sprint
3. Add medium-priority to backlog
4. Re-audit in 30 days to measure improvement
```

---

## When NOT to Use Parallel Mode

- **Single dimension focus** - If you already know the problem area, use focused audit instead
- **Quick health check** - Use `dimension: quick` for rapid triage without parallel overhead
- **Limited context** - Parallel spawns 6 agents; if context window is constrained, use sequential

---

## Comparison with Sequential Audit

| Aspect | Sequential | Parallel |
|--------|------------|----------|
| **Speed** | 15-30 min | 5-10 min |
| **Token cost** | Lower | Similar (Haiku is cheap) |
| **Depth** | Single dimension deep dive | All dimensions breadth |
| **Use case** | Known problem area | Comprehensive health check |
| **Coordination** | Simple | Requires synthesis step |

---

## Reference

- **Investigation:** `.kb/investigations/simple/2025-11-29-explore-multi-agent-parallel-review.md`
- **Pattern source:** Code-review plugin parallel agent architecture

<!-- /SKILL-TEMPLATE -->

---

<!-- DIMENSION-SPECIFIC CONTENT -->
<!-- The build system will inject the appropriate dimension module here based on spawn configuration -->

<!-- For backward compatibility with old skill names, detect dimension from SPAWN_CONTEXT -->
<!-- If spawned as codebase-audit-security, auto-set dimension=security -->
<!-- If spawned as codebase-audit --dimension performance, use that -->

**Dimension-specific guidance below:**

---

<!-- SKILL-TEMPLATE: dimension-security -->
<!-- Auto-generated from phases/dimension-security.md -->

# Codebase Audit: Security

**TLDR:** Security-focused audit identifying vulnerabilities, unsafe patterns, secrets exposure, and OWASP compliance gaps.

**Status:** STUB - To be fleshed out when needed

**When to use:** Security review needed, penetration test prep, compliance audit, incident investigation

**Output:** Investigation file with security findings rated by severity (Critical/High/Medium/Low) with remediation steps

---

## Focus Areas (To be expanded)

1. **Secrets Exposure** - API keys, passwords, tokens in code/git history
2. **Injection Vulnerabilities** - SQL injection, command injection, XSS
3. **Authentication/Authorization** - Weak auth, missing access controls
4. **Cryptography** - Weak encryption, insecure random, poor key management
5. **Dependencies** - Known vulnerabilities in packages
6. **Input Validation** - Unsafe user input handling
7. **OWASP Top 10** - Compliance with OWASP security standards

---

## Pattern Search Commands (To be expanded)

```bash
# Secrets exposure
rg "password|secret|api_key|token|private_key" --type py --type js -i

# SQL injection
rg "execute\(.*%|\.format\(|f\".*FROM|f\".*WHERE" --type py

# Command injection
rg "subprocess\.call|os\.system|eval\(|exec\(" --type py

# XSS vulnerabilities
rg "innerHTML|dangerouslySetInnerHTML|\.html\(" --type js --type jsx

# Hardcoded credentials
rg "password\s*=\s*['\"]|api_key\s*=\s*['\"]" --type py --type js
```

---

*This skill stub establishes security audit structure. Expand with detailed workflow, severity ratings, and remediation patterns when security audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-performance -->
<!-- Auto-generated from phases/dimension-performance.md -->

# Codebase Audit: Performance

**TLDR:** Performance-focused audit identifying bottlenecks, algorithmic issues, inefficient queries, and optimization opportunities.

**Status:** STUB - To be fleshed out when needed

**When to use:** App is slow, high CPU/memory usage, scaling problems, response time issues

**Output:** Investigation file with performance findings, profiling data, and optimization recommendations with effort estimates

---

## Focus Areas (To be expanded)

1. **Algorithmic Complexity** - O(n²) loops, inefficient algorithms
2. **Database Queries** - N+1 queries, missing indexes, slow queries
3. **Resource Usage** - Memory leaks, excessive allocations
4. **I/O Operations** - Blocking I/O, unnecessary file reads
5. **Caching** - Missing caches, cache invalidation issues
6. **Concurrency** - Poor parallelization, lock contention

---

## Pattern Search Commands (To be expanded)

```bash
# Nested loops (potential O(n²))
rg "for.*:\s*\n.*for.*:" --type py -U

# N+1 query patterns
rg "\.all\(\)|\.filter\(" --type py -C 3 | rg "for.*in"

# Large files (potential complexity issues)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# TODO/FIXME about performance
rg "TODO.*performance|FIXME.*slow|HACK.*optimize" -i

# Blocking I/O in loops
rg "for.*:\s*\n.*open\(|for.*:\s*\n.*requests\." --type py -U
```

---

*This skill stub establishes performance audit structure. Expand with profiling methodology, optimization patterns, and benchmarking when performance audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-tests -->
<!-- Auto-generated from phases/dimension-tests.md -->

# Codebase Audit: Tests

**TLDR:** Testing-focused audit identifying coverage gaps, flaky tests, missing test types, and test quality issues.

**Status:** STUB - To be fleshed out when needed

**When to use:** Flaky CI builds, low confidence in tests, missing edge case coverage, test suite maintenance needed

**Output:** Investigation file with testing gaps, risk assessment, coverage metrics, and test improvement roadmap

---

## Focus Areas (To be expanded)

1. **Coverage Gaps** - Modules without tests, uncovered edge cases
2. **Flaky Tests** - Time-dependent, random, inconsistent results
3. **Missing Test Types** - Unit/integration/e2e gaps
4. **Test Quality** - No assertions, over-mocking, brittle tests
5. **Test Organization** - Poor structure, hard to maintain
6. **Test Performance** - Slow tests, inefficient setup/teardown

---

## Pattern Search Commands (To be expanded)

```bash
# Modules without test files
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort)

# Flaky test indicators (sleep, random, time-based)
rg "sleep|time\.sleep|random\.|datetime\.now" tests/

# Tests without assertions
rg "def test_" tests/ -l | xargs rg "assert" -L

# Large test files (potential god test class)
find tests/ -name "*.py" | xargs wc -l | sort -rn | head -10

# Over-mocking indicators
rg "Mock|patch|MagicMock" tests/ -c | sort -rn | head -10
```

---

*This skill stub establishes testing audit structure. Expand with coverage analysis, flaky test patterns, and test quality metrics when testing audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-architecture -->
<!-- Auto-generated from phases/dimension-architecture.md -->

# Codebase Audit: Architecture

**TLDR:** Architecture-focused audit identifying coupling issues, god objects, missing abstractions, and modularity problems.

**Status:** STUB - To be fleshed out when needed

**When to use:** Hard to add features, tight coupling between modules, unclear boundaries, refactoring needed

**Output:** Investigation file with architectural issues, dependency analysis, and refactoring effort estimates

---

## Focus Areas (To be expanded)

1. **God Objects** - Classes/modules doing too much
2. **Tight Coupling** - Modules depending on too many others
3. **Missing Abstractions** - Repeated patterns not extracted
4. **Circular Dependencies** - Modules importing each other
5. **Poor Modularity** - Unclear boundaries, leaky abstractions
6. **Violation of SOLID Principles** - SRP, OCP, LSP, ISP, DIP violations

---

## Pattern Search Commands (To be expanded)

```bash
# God classes (many methods)
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -10

# Tight coupling (many imports from one module)
rg "^from (\w+) import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn

# Large files (potential god objects)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# Missing abstractions (switch/if-elif chains on type)
rg "if.*isinstance|if.*type\(.*\) ==" --type py -C 3

# Circular dependencies (imports at bottom of file)
rg "^from .* import" --type py | tail -20

# Deep nesting (complexity indicator)
rg "^\s{16,}(if|for|while|def)" --type py
```

---

*This skill stub establishes architecture audit structure. Expand with dependency analysis, refactoring patterns, and SOLID principles when architecture audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-organizational -->
<!-- Auto-generated from phases/dimension-organizational.md -->

# Codebase Audit: Organizational Drift

**TLDR:** Systematic investigation of organizational drift - ROADMAP hygiene, artifact coherence, template consistency, process adherence. Produces prioritized recommendations with system amnesia root cause analysis.

**When to use:** Dylan says "audit organizational drift", "check ROADMAP hygiene", "find documentation drift", or when you suspect accumulated organizational debt.

**Output:** Investigation file with drift patterns, evidence, system amnesia analysis, and actionable fixes.

---

## Quick Reference

### Focus Areas

1. **ROADMAP Drift** - Completed work marked TODO, missing tasks, stale priorities
2. **Documentation Drift** - Reference docs vs operational templates out of sync
3. **Template Drift** - Workspace templates vs actual workspaces inconsistent
4. **State Duplication** - Same info in multiple places falling out of sync
5. **Context Boundary Leaks** - Manual sync points across contexts (code ↔ docs ↔ tracking)

### Process (4 Phases)

1. **Pattern Search** (15-30 min) - Use automated tools to find drift candidates
2. **Evidence Collection** (30-60 min) - Validate patterns, gather concrete examples
3. **System Amnesia Analysis** (15-30 min) - Identify which coherence principles violated
4. **Documentation** (30 min) - Write investigation with recommendations and fixes

### Key Deliverable

Investigation file at `.kb/investigations/YYYY-MM-DD-audit-organizational-drift.md` with:
- **Status:** Complete
- **Root Cause:** Drift patterns with system amnesia analysis
- **Recommendations:** Prioritized fixes (forcing functions, automation, validation)

---

## Detailed Workflow

### Phase 1: Pattern Search (15-30 minutes)

**Use automated tools to find drift candidates:**

#### ROADMAP Drift Patterns

```bash
# Compare ROADMAP entries against recent git commits
cd ~/meta-orchestration
git log --oneline --since="30 days ago" | rg "feat:|fix:" | head -20
# Manually compare against docs/ROADMAP.org TODO items

# Find DONE items without completion metadata
rg "^\*\* DONE" docs/ROADMAP.org -A 5 | rg -v "CLOSED:|:Completed:"

# Find completed agents not in ROADMAP
orch history | rg "Completed" | head -10
# Check if these appear in ROADMAP
```

#### Template Drift Patterns

```bash
# Find workspaces missing new template fields
rg "^# Workspace:" .orch/workspace/ -l | while read ws; do
  grep -q "Session Scope" "$ws" || echo "MISSING SESSION SCOPE: $ws"
  grep -q "Checkpoint Strategy" "$ws" || echo "MISSING CHECKPOINT STRATEGY: $ws"
done

# Compare workspace template against actual workspaces
diff -u ~/.orch/templates/workspace/WORKSPACE.md \
        .orch/workspace/latest-workspace/WORKSPACE.md | head -50
```

#### Documentation Drift Patterns

```bash
# Find orch commands in code but not in operational templates
rg "def (spawn|check|status|complete|resume|send)" tools/orch/cli.py -o | \
  cut -d' ' -f2 | while read cmd; do
    grep -q "$cmd" ~/.orch/templates/orchestrator/orch-commands.md || \
      echo "MISSING IN TEMPLATE: $cmd"
  done

# Find features documented but not in reference docs
rg "orch \w+" ~/.orch/templates/orchestrator/ -o | sort -u > /tmp/template_cmds
rg "^###? orch" tools/README.md -o | sort -u > /tmp/readme_cmds
comm -23 /tmp/template_cmds /tmp/readme_cmds
```

#### Manual Sync Points (Fragile Patterns)

```bash
# Find "remember to" or "don't forget" instructions
rg "remember to|don't forget|make sure to update" docs/ --type md -i

# Find TODO comments about updating related files
rg "TODO.*update|FIXME.*sync" --type py --type md -C 2
```

#### State Duplication

```bash
# Find status/phase duplicated across systems
rg "status.*=.*(active|completed|paused)" --type py -l | \
  xargs rg "Phase.*=.*(Active|Complete|Paused)" -l

# Find completion timestamps in multiple places
rg "completed_at|completion_time|finished_at" --type py --type json
```

**Document all search commands in investigation file** (reproducibility)

---

### Phase 2: Evidence Collection (30-60 minutes)

**For each pattern found, gather concrete evidence:**

#### Evidence Standards

**For ROADMAP drift:**
- Specific ROADMAP entry + corresponding git commit showing drift
- Date completed vs date still showing as TODO
- Count of drift instances (how pervasive?)
- User impact (does this affect planning/prioritization?)

**For documentation drift:**
- Specific inaccuracy (what docs say vs what code does)
- File paths showing divergence
- When drift introduced (git blame to find when docs last updated)
- Impact (who's affected by stale docs - orchestrators, developers, both?)

**For template drift:**
- Specific workspace missing field + template showing field should exist
- Date workspace created vs date template updated
- Migration effort (how many workspaces need updating?)
- Graceful degradation check (does code handle missing fields?)

**For state duplication:**
- Concrete example showing same state in multiple files
- Which is source of truth? (or neither?)
- Instances where states diverged
- Proposed fix (derive, don't duplicate)

**For manual sync points:**
- Specific "remember to" instruction in docs
- Evidence of sync failures (times this was forgotten)
- Automation opportunity (can this be enforced?)

#### Investigation File Structure

```markdown
# Investigation: Organizational Drift Audit

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-organizational skill)
**Trigger:** [Dylan's request or suspected drift]

---

## TLDR

**Key findings:** [2-3 sentence summary of major drift patterns]
**Highest priority:** [Top recommendation with ROI]
**Total drift instances:** [Count across all categories]

---

## Scope

**Focus areas:** Organizational drift (ROADMAP, docs, templates, state duplication)
**Boundaries:** [Project-specific or global artifacts?]
**Time spent:** [Actual time for audit]

---

## Findings by Category

### ROADMAP Drift (Priority: High/Medium/Low)

**Pattern:** [Name of drift pattern found]

**Evidence:**
- Instance 1: ROADMAP entry "Task X" marked TODO, git commit abc123 completed 2025-11-10
- Instance 2: [...]
- Total instances: [count]

**Metrics:**
- Tasks completed but not marked DONE: [count]
- Tasks missing completion metadata: [count]
- Average drift age: [days between completion and discovery]

**Impact:** [How this affects planning/orchestration]

**Recommendation:** [Specific fix with automation approach]

**ROI:** [Value gained / time invested]

---

### [Other categories following same structure]

---

## System Amnesia Analysis

**See:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`

**Coherence principles violated:**
- [ ] Single Source of Truth - [Example showing duplication]
- [ ] Automatic Loop Closure - [Example showing manual step]
- [ ] Cross-Boundary Coherence - [Example showing context switch failure]
- [ ] Observable Drift - [Example showing silent drift]
- [ ] Forcing Functions at Creation - [Example showing optional field]

**Common failures observed:**
- [ ] ROADMAP Drift - [X instances, root cause: manual ROADMAP updates]
- [ ] Documentation Drift - [X instances, root cause: template not rebuilt]
- [ ] Template Drift - [X instances, root cause: no migration mechanism]
- [ ] State Duplication - [X instances, root cause: derived state manual]
- [ ] Context Boundary Leaks - [X instances, root cause: no cross-project search]

**Design pattern recommendations:**
- Use "Derive, Don't Duplicate" for [specific case - e.g., registry status from workspace Phase]
- Add "Validation at Boundaries" for [specific workflow - e.g., orch complete checks Phase]
- Implement "Build Systems for Consistency" for [specific docs - e.g., template rebuild automation]
- Add "Forcing Functions" for [specific creation - e.g., ROADMAP requires task-id]

---

## Prioritization

**High Priority (fix now):**
1. [Issue] - Blocking orchestration, high impact, low effort
2. [Issue] - Data loss risk, silent failures

**Medium Priority (schedule soon):**
1. [Issue] - Maintenance burden, moderate effort
2. [Issue] - Developer experience impact

**Low Priority (backlog):**
1. [Issue] - Minor improvement, can defer
2. [Issue] - Nice-to-have, low impact

---

## Recommendations

**Immediate actions (this week):**
- [ ] [Specific task with owner and approach]
  - **Fix:** [What to do]
  - **Automation:** [How to prevent recurrence]
  - **Effort:** [Hours estimated]

**Short-term (this month):**
- [ ] [Planned fix with scope]

**Long-term (next quarter):**
- [ ] [Strategic improvement with ROI]

---

## Reproducibility

**Commands used for pattern search:**
```bash
# Document all grep/rg/find/diff commands used
# This allows re-running audit in future to measure improvement
```

**Metrics baseline:**
- Total ROADMAP entries: [count]
- ROADMAP drift instances: [count]
- Template drift instances: [count]
- Documentation drift instances: [count]
- State duplication instances: [count]
- Manual sync points: [count]

**Re-audit schedule:** 3 months (measure drift reduction)

---

## Related Work

- Decision: `.kb/decisions/2025-11-15-system-amnesia-as-design-constraint.md`
- Checklist: `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- Investigation: [Link to related organizational investigations]

---

## Next Steps

1. **Discuss findings with Dylan** (present prioritization, get approval)
2. **Add high-priority items to ROADMAP** (with effort estimates)
3. **Spawn agents for fixes** (if Dylan approves immediate action)
4. **Schedule re-audit** (3 months to measure improvement)
```

---

### Phase 3: System Amnesia Analysis (15-30 minutes)

**Identify which coherence principles were violated for each drift pattern:**

**Checklist for each finding:**

1. **Single Source of Truth** - Is there duplicate state? Which is authoritative?
2. **Automatic Loop Closure** - Does workflow require manual step to complete?
3. **Cross-Boundary Coherence** - Do updates span contexts (code/docs/tracking)?
4. **Observable Drift** - Was drift silent until manual inspection?
5. **Forcing Functions at Creation** - Could invalid state be created?

**For each violation, propose design pattern:**

| Violation | Pattern | Example Fix |
|-----------|---------|-------------|
| Duplicate state | Derive, Don't Duplicate | Registry status derived from workspace Phase |
| Manual loop closure | Atomic Multi-Context Updates | `orch complete` updates all systems |
| Silent drift | Validation at Boundaries | `orch complete` checks workspace Phase |
| No forcing function | Build Systems for Consistency | Template rebuild on SessionStart hook |

**Root cause categories:**
- **Return trip tax** - Easy to create, hard to remember to update
- **Context switching** - Update happens in different session/context
- **No single source of truth** - Multiple systems maintain same state
- **Manual sync points** - "Remember to" instructions
- **No observability** - Drift accumulates silently

---

### Phase 4: Documentation (30 minutes)

**Write investigation file following template above**

**Critical sections:**
- ✅ TLDR with key findings and top priority
- ✅ Evidence section with concrete examples (file paths, commit shas, counts)
- ✅ System Amnesia Analysis (which principles violated, proposed fixes)
- ✅ Prioritization using ROI framework (impact vs effort)
- ✅ Recommendations with specific, actionable tasks
- ✅ Reproducibility section with commands and baseline metrics

**Present findings to Dylan:**
- "Organizational drift audit complete. Key findings: [TLDR]"
- "Highest priority: [Top item with ROI]"
- "System amnesia root causes: [Top 2-3 principles violated]"
- "Would you like me to add high-priority items to ROADMAP or spawn agents to address them?"

---

## Anti-Patterns to Avoid

**❌ Audit without concrete examples**
- "ROADMAP has drift issues" (vague, not actionable)
✅ **Fix:** "12 tasks completed but marked TODO: Task X (commit abc123, completed 2025-11-10), Task Y (commit def456, completed 2025-11-09), ..."

**❌ No system amnesia analysis**
- Lists drift but doesn't identify root cause or prevention
✅ **Fix:** Map each finding to violated coherence principle, propose forcing function

**❌ No reproducibility**
- Can't re-run audit to measure improvement
✅ **Fix:** Document all commands + baseline metrics

**❌ Recommendations too vague**
- "Fix ROADMAP drift" (what does that mean?)
✅ **Fix:** "Add `orch complete` auto-update: read workspace task-id field, mark ROADMAP entry DONE"

**❌ No prioritization**
- Dylan doesn't know what to fix first
✅ **Fix:** Use ROI framework (impact vs effort matrix)

---

## Related Documentation

- **System amnesia patterns:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- **Investigation template:** `.orch/templates/INVESTIGATION.md`
- **ROADMAP management:** `docs/work-prioritization.md`
- **Template build system:** `.kb/decisions/2025-11-14-orchestrator-restructuring-template-build-system.md`

---

## Example Usage

**Dylan:** "audit organizational drift in meta-orchestration"

**You:**
1. Create investigation file: `.kb/investigations/2025-11-15-organizational-drift-audit.md`
2. Run pattern search commands (ROADMAP drift, template drift, docs drift)
3. Collect evidence (12 ROADMAP drift instances, 5 template drift instances, 3 doc drift instances)
4. System amnesia analysis (violated: Automatic Loop Closure, Observable Drift)
5. Prioritize using ROI framework
6. Write investigation file with recommendations
7. Present: "Audit complete. Found 20 drift instances across 3 categories. Highest priority: Fix `orch complete` to auto-update ROADMAP (violates Automatic Loop Closure - easy fix, high impact). Add to ROADMAP?"

---

*This skill enables systematic, evidence-based organizational drift assessment with system amnesia root cause analysis and actionable recommendations.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-quick -->
<!-- Auto-generated from phases/dimension-quick.md -->

# Codebase Audit: Quick Scan

**TLDR:** 1-hour automated health check across all audit areas. Returns top 10 high-priority findings with quick-win recommendations.

**When to use:** Need rapid health check before major work, onboarding to new codebase, monthly health monitoring, or before deciding which focused audit to run.

**Output:** Investigation file with top findings across all categories, sorted by ROI.

---

## Quick Reference

### Scan Areas (All Categories)

1. **Security** - Secrets, unsafe patterns, SQL injection, XSS
2. **Performance** - Large files, complex functions, N+1 queries
3. **Tests** - Missing tests, coverage gaps, flaky indicators
4. **Architecture** - God objects, tight coupling, missing abstractions
5. **Organizational** - ROADMAP drift, template drift, doc drift

### Process (30-60 minutes)

1. **Automated Scan** (30 min) - Run all pattern search commands
2. **Triage** (15 min) - Filter to top 10 by severity/effort
3. **Document** (15 min) - Write investigation with findings

### Deliverable

Investigation file: `.kb/investigations/YYYY-MM-DD-audit-quick-scan.md`
- Top 10 findings sorted by ROI
- Recommended next steps (which focused audit to run?)

---

## Workflow

### Step 1: Automated Scan (30 minutes)

**Run these commands and capture counts:**

```bash
# Security patterns
echo "=== SECURITY ===" >> /tmp/audit.txt
rg "password|secret|api_key|token" --type py --type js -i | wc -l >> /tmp/audit.txt
rg "eval\(|exec\(|__import__|subprocess\.call" --type py | wc -l >> /tmp/audit.txt

# Performance patterns
echo "=== PERFORMANCE ===" >> /tmp/audit.txt
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -10 >> /tmp/audit.txt
rg "TODO.*performance|FIXME.*slow" -i | wc -l >> /tmp/audit.txt

# Testing patterns
echo "=== TESTS ===" >> /tmp/audit.txt
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort) | wc -l >> /tmp/audit.txt
rg "sleep|time\.sleep|random\." tests/ | wc -l >> /tmp/audit.txt

# Architecture patterns
echo "=== ARCHITECTURE ===" >> /tmp/audit.txt
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -5 >> /tmp/audit.txt
rg "^from|^import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn | head -5 >> /tmp/audit.txt

# Organizational patterns
echo "=== ORGANIZATIONAL ===" >> /tmp/audit.txt
git log --since="30 days ago" --oneline | grep -E "feat:|fix:" | wc -l >> /tmp/audit.txt
rg "remember to|don't forget" docs/ -i | wc -l >> /tmp/audit.txt
```

**Review `/tmp/audit.txt` for high counts indicating issues**

---

### Step 2: Triage (15 minutes)

**From scan results, identify top 10 by severity:**

**Severity matrix:**
- **Critical** - Security vulnerabilities, data loss risk, production blockers
- **High** - Blocking development, significant performance impact, major tech debt
- **Medium** - Maintenance burden, developer experience, moderate risk
- **Low** - Minor improvement, cosmetic, low risk

**Effort estimation:**
- **Quick win** (<4h) - Rename, add docs, simple refactor
- **Medium** (4-16h) - Extract classes, add tests, fix duplication
- **Large** (>16h) - Architectural changes, large-scale refactoring

**Top 10 = Highest severity + Lowest effort (ROI = Severity / Effort)**

---

### Step 3: Document (15 minutes)

**Investigation file structure:**

```markdown
# Investigation: Quick Audit Scan

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-quick skill)
**Scan Duration:** [X minutes]

---

## TLDR

**Top 10 findings identified** across security, performance, tests, architecture, organizational

**Recommended next step:** Run focused audit for [category with most high-severity findings]

**Quick wins available:** [Count of findings with <4h effort]

---

## Top 10 Findings (Sorted by ROI)

### 1. [Finding Name] (Severity: Critical/High/Medium, Effort: <4h/4-16h/>16h)

**Category:** Security/Performance/Tests/Architecture/Organizational

**Issue:** [One sentence describing the problem]

**Evidence:** [Quick pointer - file path, line count, or command showing issue]

**Impact:** [Why this matters]

**Quick fix:** [What to do - 1-2 sentences]

**ROI:** High/Medium/Low

---

### 2-10. [Following same structure]

---

## Scan Summary

**Total patterns scanned:** 15+ automated searches

**Findings by category:**
- Security: [count] potential issues
- Performance: [count] potential issues
- Tests: [count] potential issues
- Architecture: [count] potential issues
- Organizational: [count] potential issues

**Baseline metrics:**
- Total files: [count]
- Total lines: [count]
- Largest file: [path] ([lines] lines)
- Test coverage: [X modules without tests]
- ROADMAP drift: [X completed but marked TODO]

---

## Recommended Next Steps

**Immediate actions (quick wins <4h):**
- [ ] [Finding #X] - [Quick fix]

**Focused audits needed:**
- [ ] Run `codebase-audit-[category]` for [specific area with most critical findings]
- [ ] Run `codebase-audit-[category]` for [second priority area]

**Schedule:**
- This week: Address quick wins
- Next week: Run focused audit for [highest priority category]
- Next month: Re-run quick scan to measure improvement

---

## Reproducibility

**Commands to re-run scan:**
See Step 1 automated scan commands above.

**Re-scan schedule:** Monthly (track trend over time)
```

---

## Usage Notes

**When to use quick scan:**
- ✅ Monthly health monitoring
- ✅ Before starting major work (identify risks)
- ✅ Onboarding to unfamiliar codebase
- ✅ Deciding which focused audit to run

**When NOT to use quick scan:**
- ❌ You know the problem area (use focused audit instead)
- ❌ Need deep analysis (quick scan is surface-level)
- ❌ Investigation requires manual code reading

**Follow-up workflow:**
1. Run quick scan
2. Identify category with most critical findings
3. Run focused audit: `codebase-audit-[category]`
4. Address high-priority findings
5. Re-run quick scan in 1 month to measure improvement

---

## Anti-Patterns

**❌ Treating quick scan as comprehensive**
- Quick scan is triage, not deep analysis
✅ **Fix:** Use focused audits for thorough investigation

**❌ No follow-up action**
- Running scan without addressing findings
✅ **Fix:** Always identify at least one quick win to fix immediately

**❌ No baseline tracking**
- Can't measure improvement over time
✅ **Fix:** Re-run monthly, track metrics trend

---

*This skill provides rapid health check across all audit areas, enabling quick triage and informed decision on which focused audit to run next.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: self-review -->
<!-- Auto-generated from phases/self-review.md -->

# Self-Review (Mandatory)

Before completing the audit, verify quality of findings and recommendations.

---

## Audit-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Evidence concrete** | Each finding has file:line reference | Add specific locations |
| **Reproducible** | Pattern searches documented | Add grep/glob commands used |
| **Prioritized** | Recommendations ranked by ROI | Add severity/effort matrix |
| **Actionable** | Each recommendation has clear next step | Make specific |
| **Baseline captured** | Metrics for re-audit comparison | Add counts/percentages |

---

## Self-Review Checklist

### 1. Findings Quality

- [ ] **Each finding has evidence** - Concrete file:line references, not "there are issues"
- [ ] **Pattern searches documented** - grep/glob commands that found issues
- [ ] **False positives filtered** - Reviewed results, removed non-issues
- [ ] **Severity assessed** - Each finding has impact level (critical/high/medium/low)

### 2. Recommendations Quality

- [ ] **Prioritized by ROI** - High impact, low effort items first
- [ ] **Actionable** - Each recommendation specifies what to do
- [ ] **Scoped** - Recommendations are achievable (not "rewrite everything")
- [ ] **Linked to findings** - Each recommendation traces to specific findings

### 3. Documentation Quality

- [ ] **Investigation file complete** - All sections filled
- [ ] **Baseline metrics** - Numbers for future comparison
- [ ] **Reproduction commands** - Someone can re-run the audit
- [ ] **NOT DONE claims verified** - For each 'NOT DONE' or 'NOT IMPLEMENTED' finding, confirmed with file/code search (not just artifact reading)

### 4. Commit Hygiene

- [ ] Conventional format (`audit:` or `chore:`)
- [ ] Investigation file committed

### 5. Discovered Work Check

*Audits typically discover actionable work. Track it in beads so it doesn't get lost.*

| Type | Examples | Action |
|------|----------|--------|
| **Security bugs** | Vulnerabilities, injection risks | `bd create "SECURITY: description" --type bug` |
| **Architecture issues** | God objects, tight coupling, tech debt | `bd create "ARCHITECTURE: description" --type task` |
| **Performance issues** | N+1 queries, missing indexes | `bd create "PERFORMANCE: description" --type bug` |
| **Missing tests** | Coverage gaps, critical paths untested | `bd create "TESTING: description" --type task` |
| **Strategic Unknowns** | Architectural/premise questions discovered | `bd create "description" --type question` |

**Triage labeling for daemon processing:**

After creating issues, apply triage labels based on finding severity:

| Severity | Label | When to use |
|----------|-------|-------------|
| Critical/High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Medium/Low | `triage:review` | Needs orchestrator review before work starts |

Example:
```bash
bd create "SECURITY: SQL injection in api.py:123" --type bug
bd label <issue-id> triage:ready  # Critical severity, clear fix
```

**Why this matters:** Issues labeled `triage:ready` are automatically picked up by the work daemon for autonomous processing. Critical/High severity issues have clear scope and can be worked immediately; Medium/Low issues benefit from orchestrator review first.

**Checklist:**
- [ ] **Reviewed recommendations** - Checked audit recommendations for actionable items
- [ ] **Tracked if applicable** - Created beads issues for high-priority items (or noted "No actionable items")
- [ ] **Included in summary** - Completion comment mentions tracked issues (if any)

**If no actionable items:** Note "No beads issues created - recommendations are informational only" in completion comment.

**Why this matters:** Audits produce recommendations that often require follow-up work. Beads issues ensure these surface in SessionStart context rather than getting buried in audit files.

---

## Report via Beads

**If self-review finds issues:**
1. Fix them before proceeding
2. Report: `bd comment <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report: `bd comment <beads-id> "Self-review passed - ready for completion"`

**Checklist summary (verify mentally, report issues only):**
- Findings: Evidence with file:line, pattern searches documented, false positives filtered, severity assessed
- Recommendations: Prioritized by ROI, actionable, scoped, linked to findings
- Documentation: Investigation file complete, baseline metrics, reproduction commands
- Discovered work: Reviewed for actionable items, tracked in beads or noted "No actionable items"

**Only proceed to completion after self-review passes.**

---

## Completion Criteria

Before marking complete:

- [ ] Self-review passed
- [ ] **Leave it Better completed:** At least one `kb quick` command run OR noted as not applicable
- [ ] Investigation file complete with all findings
- [ ] Recommendations prioritized and actionable
- [ ] Baseline metrics documented for re-audit
- [ ] Pattern search commands documented (reproducibility)
- [ ] Discovered work reviewed and tracked (or noted "No actionable items")
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Complete - [findings summary]"`

**If ANY box unchecked, audit is NOT complete.**

---

**After completing all criteria:**

1. Verify all checkboxes marked
2. Report completion: `bd comment <beads-id> "Phase: Complete - Audit findings: [count], Recommendations: [count]"`
3. Call /exit to close agent session

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: leave-it-better -->
<!-- Auto-generated from phases/leave-it-better.md -->

# Leave it Better (Mandatory)

**Purpose:** Every session should leave the codebase, documentation, or knowledge base better than you found it.

**When you're in this phase:** Self-review has passed. Before marking complete, externalize what you learned.

---

## Why This Matters

Knowledge lost to session boundaries is the #1 cause of repeated mistakes and wasted effort. Every session should deposit something into the knowledge base.

**Common examples of lost knowledge:**
- "We tried X but it didn't work because Y" (others will try X again)
- "This works but only if Z is configured this way" (constraint not documented)
- "We chose A over B because..." (decision not recorded)

---

## What to Externalize

**Before marking complete, you MUST externalize at least one piece of knowledge:**

| What You Learned | Command | Example |
|------------------|---------|---------|
| Made a choice with reasoning | `kb quick decide` | `kb quick decide "Use Redis for sessions" --reason "Need distributed state for horizontal scaling"` |
| Tried something that failed | `kb quick tried` | `kb quick tried "SQLite for sessions" --failed "Race conditions with multiple workers"` |
| Discovered a constraint | `kb quick constrain` | `kb quick constrain "All endpoints must be idempotent" --reason "Retry logic requires safe replay"` |
| Found an open question | `kb quick question` | `kb quick question "Should we rate-limit per-user or per-IP?"` |

---

## Quick Checklist

- [ ] **Reflected on session:** What did I learn that the next agent should know?
- [ ] **Externalized at least one item:** Ran `kb quick decide/tried/constrain/question`
- [ ] **Improved something:** Fixed a typo, clarified docs, added a missing comment, or updated stale info (optional but encouraged)

---

## If Nothing to Externalize

If the work was straightforward implementation with no new learnings:

1. Note in your completion comment: "Leave it Better: No new knowledge to externalize - straightforward implementation"
2. This is acceptable but should be rare

**Common case:** Even "straightforward" work often reveals something worth capturing (edge case, gotcha, or clarification).

---

## Examples

**Good externalization after feature work:**
```bash
kb quick decide "Use optimistic locking for updates" --reason "Prevents lost updates without blocking reads"
kb quick tried "Pessimistic locking" --failed "Caused deadlocks under high concurrency"
```

**Good externalization after debugging:**
```bash
kb quick constrain "Cache invalidation requires explicit call" --reason "TTL alone causes stale reads"
```

**Good externalization after investigation:**
```bash
kb quick question "Is the legacy API still used? Found no callers but unclear if external consumers exist"
```

---

## Completion Criteria (Leave it Better)

- [ ] Reflected on what was learned during the session
- [ ] Ran at least one `kb quick` command OR documented why nothing to externalize
- [ ] Included "Leave it Better" status in completion comment

**Only proceed to final completion after Leave it Better is done.**

<!-- /SKILL-TEMPLATE -->






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-17x "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

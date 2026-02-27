TASK: Audit spawn context quality in cross-repo spawns (toolshed, price-watch)



SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## CONFIG RESOLUTION

- Backend: claude (source: cli-flag)
- Model: anthropic/claude-opus-4-5-20251101 (source: cli-flag)
- Tier: full (source: heuristic (skill-default))
- Spawn Mode: tmux (source: cli-flag)
- MCP: none (source: default)
- Mode: tdd (source: default)
- Validation: tests (source: default)
- Account: work (source: heuristic (primary-healthy-5h:64%-7d:88%))




## PRIOR KNOWLEDGE (from kb context)

**Query:** "spawn context quality"

### Constraints (MUST respect)
- Agents must not spawn more than 3 iterations without human review
  - Reason: Prevents runaway iteration loops like 12 tmux fallback tests in 9 minutes
- Untracked spawns (--no-track) generate placeholder beads IDs that fail bd comment commands
  - Reason: Beads IDs like orch-go-untracked-* don't exist in database, causing bd comment to fail with 'issue not found' - this is expected behavior, not a bug
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- Worker agents must test spawn functionality via unit tests and code review, not end-to-end spawning
  - Reason: Prevents recursive spawn testing incidents while still enabling verification
- kb context command hangs on some queries
  - Reason: Blocks orch spawn from returning, use --skip-artifact-check as workaround
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).
- Investigation filenames are generated from task description at spawn time
  - Reason: generateSlug() runs at spawn before agent starts - cannot reflect findings in initial filename
- Complete agents before spawning more - maintain spawn:complete ratio
  - Reason: Dec 25 reflection: 201 issues created vs 12 closed in 24h. Backlog grew faster than completion. 29 stuck agents accumulated. Spawn discipline prevents debt accumulation.

### Prior Decisions
- orch-go tmux spawn is fire-and-forget - no session ID capture
  - Reason: opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring.
- Add Leave it Better after self-review, before completion criteria
  - Reason: Natural position in workflow - self-review validates quality, Leave it Better captures knowledge, completion gates exit
- Headless Swarm = batch execution + rate-limit management across accounts
  - Reason: User clarified scope: focus on concurrent agent spawning with capacity management, not distributed architecture or multi-model routing
- orch spawn context delivery is reliable
  - Reason: Verified that SPAWN_CONTEXT.md is correctly populated and accessible by the agent
- Investigations live in .kb/ not workspaces
  - Reason: kb context discoverability essential; SYNTHESIS.md bridges via investigation_path pointer
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- Failure artifact capture should be layered: automatic events, optional reason flag, manual post-mortems
  - Reason: Balance between context capture and workflow efficiency - most abandons don't need detailed post-mortems
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs
  - Reason: Only working pattern today. Beads multi-repo hydration is read-only aggregation, bd repo commands are buggy.
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- [global] Strategic-First Orchestration
  - See: /Users/dylanconlin/.kb/decisions/2026-01-11-strategic-first-orchestration.md
- [global] Pressure Over Compensation Principle
  - See: /Users/dylanconlin/.kb/decisions/2025-12-25-pressure-over-compensation.md
- [global] Focus on Domain Translation, Not AI Infrastructure
  - See: /Users/dylanconlin/.kb/decisions/2026-02-24-career-focus-domain-translation-over-infrastructure.md
- [global] CLI Errors Should Reduce Cognitive Load
  - See: /Users/dylanconlin/.kb/decisions/2026-01-08-cli-errors-reduce-cognitive-load.md
- [global] Operationalize Models in Orchestrator Skill
  - See: /Users/dylanconlin/.kb/decisions/2026-01-12-operationalize-models-in-orchestrator-skill.md
- [global] Coherence Over Patches Principle
  - See: /Users/dylanconlin/.kb/decisions/2026-01-04-coherence-over-patches.md
- [global] Verification Bottleneck Principle
  - See: /Users/dylanconlin/.kb/decisions/2026-01-04-verification-bottleneck.md
- [global] Premise Before Solution Principle
  - See: /Users/dylanconlin/.kb/decisions/2025-12-25-premise-before-solution.md
- [global] Share Patterns Not Tools
  - See: /Users/dylanconlin/.kb/decisions/2025-12-26-share-patterns-not-tools.md
- [global] Gate Over Remind Refinement - Gates Must Be Passable By The Gated
  - See: /Users/dylanconlin/.kb/decisions/2026-01-04-gate-refinement-passable-by-gated.md

### Models (synthesized understanding)
- Spawn Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/model.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-25.
    Changed files: cmd/orch/spawn_cmd.go, pkg/orch/extraction.go, pkg/spawn/context.go, pkg/spawn/kbcontext.go, pkg/spawn/config.go, pkg/spawn/claude.go, pkg/spawn/session.go.
    Verify model claims about these files against current code.
  - Summary:
    Spawn evolved through 7 phases from basic CLI integration to a modular, gate-driven architecture with capacity-aware account routing. The architecture creates a workspace with SPAWN_CONTEXT.md embedding skill content + task description + kb context, then launches a session via two-phase atomic spawn with rollback. Spawn settings are resolved via `pkg/spawn/resolve.go` with 6-level precedence and per-setting provenance tracking. The spawn pipeline is split across three packages: `pkg/spawn/` (config, resolution, context generation), `pkg/spawn/gates/` (pre-spawn validation), `pkg/spawn/backends/` (backend abstraction), and `pkg/orch/` (pipeline orchestration and mode dispatch). The tier system (light/full) determines whether SYNTHESIS.md is required at completion. Claude CLI is the default backend since Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026).
    
    ---
  - Critical Invariants:
    1. **Workspace name = kebab-case task description** - Used for tmux window, directory name, session title
    2. **Beads ID required for phase reporting** - `--no-track` creates untracked IDs that can't report to beads
    3. **KB context uses --global flag** - Cross-repo constraints are essential
    4. **Skill content stripped for --no-track** - Beads instructions removed when not tracking
    5. **Session scoping is per-project** - `orch send` only works within same directory hash
    6. **Token estimation at 4 chars/token** - Warning at 100k, error at 150k
    7. **Model-aware backend routing** - Backend determined by model provider unless CLI overrides (Decision: kb-2d62ef)
    8. **Claude backend implies tmux** - Claude CLI physically requires tmux window; headless + claude auto-switches to tmux
    9. **Account routing is capacity-aware** - Primary accounts checked first; spillover activated when primaries exhausted (>20% threshold)
    
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
  - Recent Probes:
    - 2026-02-27-probe-kb-context-query-derivation-and-assembly
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
    - 2026-02-15-spawn-time-staleness-detection-behavioral-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
    - 2026-02-25-probe-cross-project-kb-context-group-resolution
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-25-probe-cross-project-kb-context-group-resolution.md
    - 2026-02-24-probe-architect-gate-hotspot-enforcement
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-24-probe-architect-gate-hotspot-enforcement.md
    - 2026-02-20-probe-session-scope-template-honor
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/model.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-25.
    Changed files: cmd/orch/spawn_cmd.go, pkg/orch/extraction.go, CLAUDE.md.
    Verify model claims about these files against current code.
  - Summary:
    Anthropic banned subscription OAuth in third-party tools (Feb 19, 2026), making **Claude CLI the default backend** for Anthropic models (was previously the escape hatch). The architecture now uses **model-aware backend routing**: Anthropic models → Claude CLI (tmux), non-Anthropic models (Google, OpenAI, DeepSeek) → OpenCode API (headless). Account routing is capacity-aware with primary/spillover accounts and a health threshold (>20%). The escape hatch pattern remains for infrastructure work, but is now advisory — higher-priority settings (CLI, model requirement, project/user config) take precedence.
    
    ---
  - Critical Invariants:
    1. **Never spawn OpenCode infrastructure work without --backend claude --tmux**
       - Violation: Agent kills itself mid-execution when server restarts
       - Now auto-detected: infrastructure keywords trigger `--backend claude` which implies tmux
    
    2. **Infrastructure detection is advisory, not overriding (changed Feb 2026)**
       - Runs at priority 5 (below CLI, model requirement, project config, user config)
       - When higher-priority setting present, emits warning instead of overriding
       - Ensures explicit user choices are always respected
    
    3. **Anthropic models blocked on OpenCode by default**
       - API requests to Anthropic models on opencode return error
       - Override: `allow_anthropic_opencode: true` in user config (`~/.orch/config.yaml`)
       - Opus specifically requires Claude CLI backend (fingerprinting blocks API)
    
    4. **Escape hatch provides true independence**
       - Claude CLI binary ≠ OpenCode server
       - Tmux session persists across service restarts
       - Different authentication path (Max subscription OAuth)
    
    5. **Flash models are blocked entirely (added Feb 2026)**
       - `validateModel()` returns error for any flash model
       - Supersedes the Gemini Flash TPM limit constraint — no workaround needed
    
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
  - Recent Probes:
    - 2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-24-probe-daemon-spawn-model-bypass-and-claude-visibility.md
    - 2026-02-21-probe-gpt-model-spawn-e2e-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-21-probe-gpt-model-spawn-e2e-verification.md
    - 2026-02-20-model-aware-backend-routing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-model-aware-backend-routing.md
    - 2026-02-20-probe-default-backend-anthropic-incompatibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-probe-default-backend-anthropic-incompatibility.md
    - 2026-02-20-backend-resolution-architecture-drift
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-backend-resolution-architecture-drift.md
- Probe: Daemon Duplicate Spawn Feb 14 Incident
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-feb14-incident.md
  - Recent Probes:
    - 2026-02-09-dashboard-restart-daemon-autostart-default-disabled
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-09-dashboard-restart-daemon-autostart-default-disabled.md
    - 2026-02-25-probe-project-group-model-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-project-group-model-design.md
    - 2026-02-25-probe-cross-repo-orchestration-consequences
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md
    - 2026-02-25-probe-cross-project-spawn-beads-defaultdir
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-project-spawn-beads-defaultdir.md
    - 2026-02-24-probe-daemon-single-instance-pid-lock
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md
- Probe: Daemon Duplicate Spawn TTL Fragility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-14-daemon-duplicate-spawn-ttl-fragility.md
  - Recent Probes:
    - 2026-02-09-dashboard-restart-daemon-autostart-default-disabled
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-09-dashboard-restart-daemon-autostart-default-disabled.md
    - 2026-02-25-probe-project-group-model-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-project-group-model-design.md
    - 2026-02-25-probe-cross-repo-orchestration-consequences
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md
    - 2026-02-25-probe-cross-project-spawn-beads-defaultdir
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-project-spawn-beads-defaultdir.md
    - 2026-02-24-probe-daemon-single-instance-pid-lock
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-daemon-single-instance-pid-lock.md
- Probe: Spawn Workflow Mechanics Analysis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-workflow-mechanics-analysis.md
  - Recent Probes:
    - 2026-02-27-probe-kb-context-query-derivation-and-assembly
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
    - 2026-02-15-spawn-time-staleness-detection-behavioral-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
    - 2026-02-25-probe-cross-project-kb-context-group-resolution
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-25-probe-cross-project-kb-context-group-resolution.md
    - 2026-02-24-probe-architect-gate-hotspot-enforcement
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-24-probe-architect-gate-hotspot-enforcement.md
    - 2026-02-20-probe-session-scope-template-honor
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md
- Probe: KB Context Query Derivation and Assembly Pipeline
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
  - Recent Probes:
    - 2026-02-27-probe-kb-context-query-derivation-and-assembly
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
    - 2026-02-15-spawn-time-staleness-detection-behavioral-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
    - 2026-02-25-probe-cross-project-kb-context-group-resolution
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-25-probe-cross-project-kb-context-group-resolution.md
    - 2026-02-24-probe-architect-gate-hotspot-enforcement
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-24-probe-architect-gate-hotspot-enforcement.md
    - 2026-02-20-probe-session-scope-template-honor
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md
- Probe: Authority Section Duplication Between Spawn Template and Worker-Base
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-authority-section-dedup.md
  - Recent Probes:
    - 2026-02-27-probe-kb-context-query-derivation-and-assembly
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
    - 2026-02-15-spawn-time-staleness-detection-behavioral-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
    - 2026-02-25-probe-cross-project-kb-context-group-resolution
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-25-probe-cross-project-kb-context-group-resolution.md
    - 2026-02-24-probe-architect-gate-hotspot-enforcement
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-24-probe-architect-gate-hotspot-enforcement.md
    - 2026-02-20-probe-session-scope-template-honor
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md
- Probe: Inventory all friction gates across spawn, completion, and daemon — assess defect-catching vs noise
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
  - Recent Probes:
    - 2026-02-09-friction-bypass-analysis-post-targeted-skips
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md
    - 2026-02-13-friction-gate-inventory-all-subsystems
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
    - 2026-02-25-probe-code-review-gate-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-25-probe-code-review-gate-design.md
    - 2026-02-25-probe-hotspot-bloat-scanner-build-output-exclusions
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-25-probe-hotspot-bloat-scanner-build-output-exclusions.md
    - 2026-02-25-probe-coupling-cluster-implementation-review
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-25-probe-coupling-cluster-implementation-review.md
- Probe: Spawn Architecture Structural Drift (46 commits since update)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-spawn-architecture-structural-drift.md
  - Recent Probes:
    - 2026-02-27-probe-kb-context-query-derivation-and-assembly
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-27-probe-kb-context-query-derivation-and-assembly.md
    - 2026-02-15-spawn-time-staleness-detection-behavioral-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-15-spawn-time-staleness-detection-behavioral-verification.md
    - 2026-02-25-probe-cross-project-kb-context-group-resolution
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-25-probe-cross-project-kb-context-group-resolution.md
    - 2026-02-24-probe-architect-gate-hotspot-enforcement
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-24-probe-architect-gate-hotspot-enforcement.md
    - 2026-02-20-probe-session-scope-template-honor
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture/probes/2026-02-20-probe-session-scope-template-honor.md
- Context Injection Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/context-injection/model.md
  - Summary:
    The system for providing agents (Orchestrators and Workers) with the necessary mental model and state to perform their tasks. It manages the boundary between "Static Guidance" (Skills/Principles) and "Dynamic State" (Backlog/Investigations).
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Token Bloat (Role-Blindness)
    - **Symptom:** Workers receive the 86KB Orchestrator skill.
    - **Cause:** Hooks running for "Startup" without checking the `CLAUDE_CONTEXT`.
    - **Impact:** Context fills up fast, agent performance degrades, costs increase.
    
    ### Failure Mode 2: Redundant Guidance
    - **Symptom:** Beads tracking instructions appear in `bd prime`, the Orchestrator skill, AND `SPAWN_CONTEXT.md`.
    - **Cause:** Lack of coordination between the different injection sources (Hooks vs. Templates).
    
    ### Failure Mode 3: Session Resume Leakage
    - **Symptom:** Spawned workers are told to "Resume" a previous manual session.
    - **Cause:** `session-start.sh` blindly injecting handoffs into agents that have their own `SPAWN_CONTEXT.md`.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.

### Guides (procedural knowledge)
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- How Spawn Works
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawn.md
- Tmux Spawn Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/tmux-spawn-guide.md
- Headless Spawn Mode Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/headless.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/archived/dual-spawn-mode-implementation.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/archived/worker-patterns.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md

### Related Investigations
- Audit Spawn Prompt Quality Vs Outcomes
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-13-inv-audit-spawn-prompt-quality-vs-outcomes.md
- Token Limit Explosion Headless Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-token-limit-explosion-headless-spawn.md
- Measure Context Preparation Vs Execution
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2026-01-17-inv-measure-context-preparation-vs-execution.md
- Verify Spawn Works
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-verify-spawn-works.md
- Test Spawn Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-test-spawn-functionality.md
- tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-zeta.md
- Concurrent tmux spawn test (delta)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-delta.md
- Analyze Spawn Value Ratio
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2026-01-17-inv-analyze-spawn-value-ratio.md
- Design: Atomic Spawn with Workspace Manifest and orch:agent Tagging
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-19-design-atomic-spawn-workspace-manifest-orch-agent.md
- [global] Multi-Phase Feature Orchestration Process Gaps
  - See: /Users/dylanconlin/.kb/investigations/2025-11-18-multi-phase-feature-orchestration-process-gaps.md

### Failed Attempts (DO NOT repeat)
- BEADS_NO_DAEMON=1 in .zshrc only

### Open Questions
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





## HOTSPOT AREA WARNING

⚠️ This task targets files in a **hotspot area** (high churn, complexity, or coupling).

**Hotspot files:**
- `spawn`
- `spawn`

**Investigation routing:** If your findings affect these files, recommend `architect` follow-up instead of direct `feature-impl`. Hotspot areas require architectural review before implementation changes.



🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. (Allowed) Read this SPAWN_CONTEXT.md file (your first tool call may be this read)
2. Immediately report via `bd comment orch-go-1goo "Phase: Planning - [brief description]"`
3. Read relevant codebase context for your task and begin planning

If Phase is not reported within first 3 actions, you will be flagged as unresponsive.
Do NOT skip this - the orchestrator monitors via beads comments.



VERIFICATION REQUIREMENTS (ORCH COMPLETE):
Your work is verified in two human gates before closing:
- Gate 1 (explain-back): orchestrator must explain what was built and why.
- Gate 2 (behavioral, Tier 1 only): orchestrator confirms behavior is verified.
Provide clear Phase: Complete summary and VERIFICATION_SPEC.yaml evidence to support both gates.


CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours



AUTHORITY:
Authority delegation rules are provided via skill guidance (worker-base skill).
**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Surface it first: `bd comment orch-go-1goo "CONSTRAINT: [what constraint] - [why considering workaround]"`
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


2. **SET UP probe file:** This is confirmatory work against an existing model.
   - Model content was injected in PRIOR KNOWLEDGE section above
   - Create probe file in model's probes/ directory (use the absolute path from the `See:` reference)
   - Use probe template structure: Question, What I Tested, What I Observed, Model Impact
   - Your probe should confirm, contradict, or extend the model's claims


   - **IMPORTANT:** After creating probe file, report the **absolute** path via:
     `bd comment orch-go-1goo "probe_path: /path/to/probe.md"`



3. **UPDATE probe file** as you work:
   - Question: What model claim are you testing?
   - What I Tested: Actual command/code run (not just code review)
   - What I Observed: Actual output/behavior
   - Model Impact: Confirms/contradicts/extends which invariant

4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]


6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-audit-spawn-context-27feb-936a/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your probe file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to probe file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (investigation)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 0c87cddf3371 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-26 18:54:12 -->


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

**Critical routing rule:** Investigation findings that recommend code changes must be routed through architect before implementation. The sequence is: investigation → architect → implementation. Implementing directly from investigation findings can produce code that violates architectural decisions.

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
bd comment orch-go-1goo "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-1goo "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-1goo "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-1goo "Phase: BLOCKED - Need clarification on API contract"

# Report questions
bd comment orch-go-1goo "Phase: QUESTION - Should we use JWT or session-based auth?"
```

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Additional context:**
Use `bd comment` for additional context, findings, or updates:
```bash
bd comment orch-go-1goo "Found performance bottleneck in database query"
bd comment orch-go-1goo "investigation_path: .kb/investigations/2026-02-11-perf-issue.md"
```

**Test Evidence Requirement:**
When reporting Phase: Complete, include test results in the summary:
- Example: `bd comment orch-go-1goo "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed (2.3s)"`
- Example: `bd comment orch-go-1goo "Phase: Complete - Tests: npm test - 23 specs, 0 failures"`
- Example: `bd comment orch-go-1goo "Phase: Complete - Tests: make test - PASS (coverage: 78%)"`

**Why:** `orch complete` validates test evidence in phase comments. Vague claims like "all tests pass" trigger manual verification.

**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`.
- Workers report `Phase: Complete`, orchestrator verifies and closes
- Running `bd close` bypasses verification and breaks tracking

---


## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment orch-go-1goo "Phase: Planning - [brief description]"`
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

**Do NOT create empty investigation files as discovered work.** Empty investigation templates accumulate rapidly (~13/week) and create noise in the knowledge base. For discovered work, create beads issues — only create investigation files when you are actively investigating.

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

**Git Staging Rule:** NEVER use `git add -A` or `git add .` — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/). Stage ONLY the specific files you created or modified for your task, by name.

**When your work is done (all deliverables ready), complete in this EXACT order:**


1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment orch-go-1goo "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. Ensure SYNTHESIS.md is created with these required sections:
   - **`Plain-Language Summary`** (REQUIRED): 2-4 sentences in plain language describing what you built/found/decided and why it matters. This is the scaffolding the orchestrator uses during completion review — write it for a human who hasn't read your code. No jargon without explanation. No "implemented X" without saying what X does.
   - **`Verification Contract`**: Link to `VERIFICATION_SPEC.yaml` and key outcomes
4. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
5. Commit all remaining changes (including SYNTHESIS.md and `VERIFICATION_SPEC.yaml`)
6. Run: `/exit` to close the agent session


**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

---


---
name: investigation
skill-type: procedure
description: Record what you tested and observed; default to model-scoped probes when injected model claims are present, otherwise run a full investigation.
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: b5549f232fdd -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-26 18:54:12 -->


<!-- SKILL-CONSTRAINTS -->
<!-- optional: .kb/models/*/probes/{date}-*.md | Model probe output when injected model content is present -->
<!-- optional: .kb/investigations/{date}-inv-*.md | Investigation output when no injected model content is present -->
<!-- /SKILL-CONSTRAINTS -->
## Summary

**Purpose:** Answer a question by testing, not by reasoning.

---

# Investigation Skill

**Purpose:** Answer a question by testing, not by reasoning.

## Artifact Mode Selection (Probe Default)

**Before creating any artifact, read SPAWN_CONTEXT.md and detect mode:**

1. Find the `### Models (synthesized understanding)` section
2. Check for injected model-claim markers in model entries:
   - `- Summary:`
   - `- Critical Invariants:` or `- Constraints:`
   - `- Why This Fails:` or `- Failure Modes:`

### If markers are present -> Probe Mode (default)

- Create a probe (not a full investigation)
- Use template: `.orch/templates/PROBE.md`
- Write to: `.kb/models/{model-name}/probes/{date}-{slug}.md`
- Required sections:
  - `Question`
  - `What I Tested`
  - `What I Observed`
  - `Model Impact`

### If markers are absent -> Investigation Mode

- Follow standard investigation workflow
- Write to: `.kb/investigations/{date}-inv-{slug}.md`

## The One Rule

**You cannot conclude without testing.**

If you didn't run a test, you don't get to fill the Conclusion section.

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

- **Primary** (authoritative): Actual code, test output, observed behavior → This IS the evidence
- **Secondary** (claims to verify): Workspaces, investigations, decisions → Hypotheses to test

When an artifact says "X is not implemented," that's a hypothesis—search the codebase before concluding.

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for evidence hierarchy examples and common failures.


## Prior Work Acknowledgment

**This section applies in Investigation Mode only (no injected model-claim markers).**

Before creating your investigation file, review prior work from SPAWN_CONTEXT.

1. **Check SPAWN_CONTEXT** for "Related Investigations" section
2. **If prior work exists:**
   - Note which investigations are relevant to your question
   - Plan to verify cited claims AS YOU ENCOUNTER THEM during investigation
3. **If no prior work:** Note "N/A - novel investigation"

**This is acknowledgment, not exhaustive verification.** You verify claims naturally as you encounter them during your investigation, not all upfront.

---

## Workflow

1. **Detect mode from SPAWN_CONTEXT.md** using model-claim markers in `### Models (synthesized understanding)`:
   - `- Summary:`
   - `- Critical Invariants:` or `- Constraints:`
   - `- Why This Fails:` or `- Failure Modes:`
2. **If markers found -> Probe Mode (default)**
   - Pick the most relevant model from the injected models section
   - Create: `.kb/models/{model-name}/probes/{date}-{slug}.md`
   - Use `.orch/templates/PROBE.md`
   - Fill all required sections: Question, What I Tested, What I Observed, Model Impact
3. **If markers not found -> Investigation Mode**
   - Acknowledge prior work: SPAWN_CONTEXT -> "Related Investigations"
   - Create file: `kb create investigation {slug}`
   - IMMEDIATE CHECKPOINT: Fill Question, add Prior-Work table, add Finding 1 ("Starting approach"), commit immediately
4. **TEST-FIRST GATE:** "What's the simplest test I can run right now?" (60-second rule)
5. Try things, observe what happens (add findings/probe evidence progressively)
6. Verify relevant claims as encountered
7. Run a real test to validate your hypothesis
8. Fill conclusion/model impact based on observed evidence only
9. Final commit

**Why checkpoint immediately?** Agents can die from API errors, context limits, or crashes. Without a checkpoint, no record of what was attempted.

**Reference:** See `~/.claude/skills/worker/investigation/reference/error-recovery.md` for handling fatal errors during exploration.

## D.E.K.N. Summary

**D.E.K.N. applies to Investigation Mode.** Probes use the probe template's required sections instead.

- **Delta:** What was discovered/answered
- **Evidence:** Primary evidence supporting conclusion
- **Knowledge:** What was learned (insights, constraints)
- **Next:** Recommended action

**Fill D.E.K.N. at the END, before marking Complete.**

**Reference:** See `~/.claude/skills/worker/investigation/reference/examples.md` for D.E.K.N. examples.


## Template

Choose template based on SPAWN_CONTEXT mode detection.

### Probe Mode (default when injected model markers are present)

Use `.orch/templates/PROBE.md`. Write to `.kb/models/{model-name}/probes/{date}-{slug}.md`.

Required sections:

- **Question**
- **What I Tested** (actual command/code executed)
- **What I Observed** (concrete output)
- **Model Impact** (confirms | contradicts | extends)

### Investigation Mode (fallback when model markers are absent)

Use `kb create investigation {slug}`. Required sections:

- **D.E.K.N. Summary** (Delta, Evidence, Knowledge, Next)
- **Prior Work** table (entries OR "N/A - novel investigation")
- **Question** and **Status**
- **Findings** (add progressively)
- **Test performed** (not "reviewed code" - actual test)
- **Conclusion** (only if you tested)

### Prior-Work Table Structure

```markdown
## Prior Work

| Investigation                          | Relationship | Verified | Conflicts |
| -------------------------------------- | ------------ | -------- | --------- |
| .kb/investigations/2026-01-26-inv-X.md | extends      | pending  | -         |
| N/A - novel investigation              | -            | -        | -         |
```

**Relationship vocabulary:**

- **Extends:** Adds to prior findings (most common)
- **Confirms:** Validates prior hypothesis with new evidence
- **Contradicts:** Disproves or refines prior conclusion
- **Deepens:** Explores same question at greater depth

**Verified column:** Start with "pending", update to "yes" when you test a cited claim during investigation.

**Conflicts column:** Document contradictions found during verification.

**Reference:** See `~/.claude/skills/worker/investigation/reference/template.md` for full structure and `reference/examples.md` for common failures.

## When Not to Use

- **Fixing bugs** → Use `systematic-debugging`
- **Trivial questions** → Just answer them
- **Documentation** → Use `capture-knowledge`


## Prior Work (Template Independence)

**Applies to Investigation Mode only.** In Probe Mode, use the probe template and skip the Prior-Work table.

**Why this matters:** 701 existing investigations lack the Prior-Work table. The skill must handle both old and new investigations gracefully.

### Creating New Investigations

When creating a new investigation via `kb create investigation`, your file MUST include a Prior-Work table:

```markdown
## Prior Work

| Investigation   | Relationship | Verified | Conflicts     |
| --------------- | ------------ | -------- | ------------- |
| [path or "N/A"] | [type]       | [yes/no] | [description] |

**Relationship types:** extends, confirms, contradicts, deepens
```

**If no prior investigations exist:** Use explicit acknowledgment:

```markdown
## Prior Work

| Investigation             | Relationship | Verified | Conflicts |
| ------------------------- | ------------ | -------- | --------- |
| N/A - novel investigation | -            | -        | -         |
```

### Extending Old Investigations (Graceful Degradation)

If SPAWN_CONTEXT references prior investigations that lack Prior-Work tables:

1. **Do NOT backfill old investigations** - Never modify their structure
2. **Create a new investigation file** with full Prior-Work table
3. **Reference the old investigation** in your Prior-Work table:

```markdown
## Prior Work

| Investigation                                  | Relationship | Verified | Conflicts                                  |
| ---------------------------------------------- | ------------ | -------- | ------------------------------------------ |
| .kb/investigations/2025-11-15-inv-old-topic.md | extends      | yes      | [describe any conflicts with prior claims] |
```

**Verification happens naturally:** As you explore, you'll encounter claims from prior investigations. Verify them against primary sources (code, test output) when relevant to your question - not all upfront.

### Key Rules

| Situation                       | Action                                                    |
| ------------------------------- | --------------------------------------------------------- |
| New investigation (no priors)   | Add Prior-Work table with "N/A - novel investigation"     |
| New investigation (has priors)  | Add Prior-Work table, cite prior investigations           |
| Extending old investigation     | Create NEW file with Prior-Work table referencing the old |
| Old investigation without table | Leave it alone, don't modify                              |

**Reference:** See `~/.claude/skills/worker/investigation/reference/prior-work-examples.md` for relationship type guidance.


## Self-Review (Mandatory)

Before completing, verify quality for the mode you used:

### Probe Mode Checklist

- [ ] File path is `.kb/models/{model-name}/probes/{date}-{slug}.md`
- [ ] Used `.orch/templates/PROBE.md`
- [ ] All 4 sections present: Question, What I Tested, What I Observed, Model Impact
- [ ] `What I Tested` contains executed command/code (not code review)
- [ ] `What I Observed` includes concrete output
- [ ] Model Impact verdict is explicit: confirms | contradicts | extends

### Investigation Mode Checklist

- [ ] **Prior-Work acknowledged** - Table present with entries OR explicit "N/A - novel investigation"
- [ ] **Cited claims verified** - Any claim referenced from prior work was tested (Verified = "yes")
- [ ] **Test is real** - Ran actual command/code, not just "reviewed"
- [ ] **Evidence concrete** - Specific outputs, not "it seems to work"
- [ ] **Conclusion factual** - Based on observed results, not inference
- [ ] **No speculation** - Removed "probably", "likely", "should" from conclusion
- [ ] **Question answered** - Investigation addresses the original question
- [ ] **File complete** - All sections filled (not "N/A" or "None")
- [ ] **D.E.K.N. filled** - Replaced placeholders in Summary section
- [ ] **Scope verified** - Ran `rg` to find all occurrences before concluding
- [ ] **NOT DONE claims verified** - If claiming incomplete, searched actual code

### Prior-Work Verification

**Applies to Investigation Mode only.**

**Gate:** Your investigation file MUST contain a Prior-Work table.

| Situation                                   | Required Action                                                  |
| ------------------------------------------- | ---------------------------------------------------------------- |
| SPAWN_CONTEXT has "Related Investigations"  | List relevant ones in table, verify claims as you encounter them |
| SPAWN_CONTEXT has no related investigations | Add single row: "N/A - novel investigation"                      |
| You cited prior work without verifying      | Update Verified column, document conflicts found                 |

**This is passable:** You only need to verify claims you actually referenced during your investigation. You do NOT need to exhaustively verify all prior work upfront.

### Discovered Work

If you found bugs, tech debt, or enhancement ideas during investigation:

- Create beads issues: `bd create "description" --type bug|task|feature`
- Apply label: `bd label <id> triage:ready` or `triage:review`

**If no discoveries:** Note "No discovered work items" in completion comment.

**Reference:** See `~/.claude/skills/worker/investigation/reference/self-review-guide.md` for scope verification examples and discovered work procedures.

**Only proceed to commit after self-review passes.**


---

## Leave it Better (Mandatory)

**Before marking complete, externalize at least one piece of knowledge:**
- `kb quick decide "X" --reason "Y"` (made a choice)
- `kb quick tried "X" --failed "Y"` (something failed)
- `kb quick constrain "X" --reason "Y"` (found a constraint)
- `kb quick question "X"` (open question)

**Also prune stale artifacts you encountered:**
- `kb supersede <old> --by <new>` (artifact replaced by newer work)
- `kb archive --synthesized-into <guide>` (investigations covered by a guide)

**If nothing to externalize:** Note in completion comment: "Leave it Better: Straightforward investigation, no new knowledge to externalize."
**If no stale artifacts found:** Note "No stale artifacts encountered."

**Reference:** See `~/.claude/skills/worker/investigation/reference/leave-it-better.md` for command examples.

---

## Completion

1. Self-review passed
2. **Probe vs Investigation requirements met:**
   - Probe Mode: probe file exists in `.kb/models/{model-name}/probes/` with all 4 mandatory sections
   - Investigation Mode: Prior-Work acknowledged and D.E.K.N. summary filled
3. Leave it Better completed (or noted why N/A)
4. Report: `bd comment <beads-id> "Phase: Complete - [conclusion summary]"` (FIRST - before commit)
5. Commit: `git add && git commit`
6. Exit: `/exit`

**Why report before commit?** If agent dies after commit but before reporting, orchestrator cannot detect completion.

---

**Remember:** Test before concluding.




---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.



1. Create SYNTHESIS.md in your workspace
2. **COMMIT YOUR WORK:** `git add <files you changed> && git commit -m "feat: [description] (orch-go-1goo)"`
3. `bd comment orch-go-1goo "Phase: Complete - [1-2 sentence summary]"`
4. `/exit`



⛔ **NEVER run `git push`** - Workers commit locally only.
⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
⚠️ Your work is NOT complete until Phase: Complete is reported (or /exit for --no-track).

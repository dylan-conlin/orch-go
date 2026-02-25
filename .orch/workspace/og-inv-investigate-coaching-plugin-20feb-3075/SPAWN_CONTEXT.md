TASK: Investigate the coaching plugin behavior. Read .opencode/plugins/coaching.ts and document: (1) what triggers health metric injection, (2) how detectWorkerSession works, (3) what metrics are injected into worker sessions. Write findings to your SYNTHESIS.md.


SPAWN TIER: light

⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.



## CONFIG RESOLUTION

- Backend: opencode (source: cli-flag)
- Model: openai/gpt-5.2-codex (source: cli-flag)
- Tier: light (source: cli-flag)
- Spawn Mode: headless (source: default)
- MCP: none (source: default)
- Mode: tdd (source: default)
- Validation: tests (source: default)




## PRIOR KNOWLEDGE (from kb context)

**Query:** "investigate coaching plugin"

### Constraints (MUST respect)
- Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads
  - Reason: Investigation found all registry data can be derived from primary sources
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- D.E.K.N. 'Next:' field must be updated when marking Status: Complete
  - Reason: Prevents stale investigations that mislead future agents
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- LLM guidance compliance requires signal balance - overwhelming counter-patterns (56:13 ratio) drowns specific exceptions
  - Reason: Investigation found orchestrator skill has 4:1 ask-vs-act signal ratio causing autonomy guidance to fail
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).
- Investigation filenames are generated from task description at spawn time
  - Reason: generateSlug() runs at spawn before agent starts - cannot reflect findings in initial filename

### Prior Decisions
- OpenCode ListSessions WITH x-opencode-directory header returns disk sessions, WITHOUT returns in-memory
  - Reason: Finding from investigation - explains 2 vs 238 session count discrepancy
- Investigations live in .kb/ not workspaces
  - Reason: kb context discoverability essential; SYNTHESIS.md bridges via investigation_path pointer
- Session boundaries have three distinct patterns: worker (protocol-driven via Phase:Complete), orchestrator (state-driven via session-transition), and cross-session (manual via SESSION_HANDOFF.md)
  - Reason: Investigation found no unified boundary protocol; each type optimized for its context
- orch-go is primary CLI, orch-cli (Python) is reference/fallback
  - Reason: Go provides better primitives (single binary, OpenCode HTTP client, goroutines); Python taught requirements through 27k lines and 200+ investigations
- D.E.K.N. is universal handoff structure
  - Reason: Delta/Evidence/Knowledge/Next enables 30-second context transfer between Claude instances - proven across SYNTHESIS.md, investigations, and session handoffs
- kb reflect uses single command with --type flag for four reflection modes
  - Reason: Consistent with kb pattern, most discoverable, extensible. Maps directly to signals from ws4z investigations.
- Self-reflection is signal-triggered not time-scheduled
  - Reason: Density thresholds (3+ investigations) produce actionable signals; time intervals (weekly review) produce noise. Per ws4z.8 investigation.
- Dual-mode architecture (tmux for visual, HTTP for programmatic) is the correct design
  - Reason: Investigation confirmed each mode serves distinct, irreplaceable needs
- Confidence scores in investigations are noise, not signal
  - Reason: Nov 2025 Codex case: 5 investigations at High/Very High confidence all reached wrong conclusions. Current distribution: 90% cluster at 85-95%, providing no discriminative value. Recommendation: Replace with structured uncertainty (tested/untested enumeration).
- kb-cli owns artifact templates (investigation, decision, guide, research)
  - Reason: Consolidation complete - skill templates/ directories removed, kb-cli hardcoded updated with D.E.K.N.

### Models (synthesized understanding)
- Coaching Plugin
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-14 (worker detection fix verified).
    Changed files: plugins/coaching.ts, cmd/orch/serve_coaching.go, pkg/opencode/client.go, web/src/routes/+page.svelte.
    Deleted files: ~/.orch/coaching-metrics.jsonl.
    Verify model claims about these files against current code.
  - Summary:
    The Coaching Plugin is an OpenCode plugin that provides real-time behavioral feedback to orchestrators and workers through tool usage pattern detection. It implements the "Pain as Signal" architectural pattern: agents should feel friction in real-time rather than learning about it post-hoc.
    
    The plugin hooks `tool.execute.after` to observe tool usage patterns (it cannot see LLM response text—fundamental constraint), detects 8 behavioral patterns using behavioral proxies (action ratio, analysis paralysis, frame collapse, etc.), and injects coaching messages via `client.session.prompt({ noReply: true })`. Metrics persist to `~/.orch/coaching-metrics.jsonl` and are exposed via `/api/coaching` for dashboard visualization.
    
    **Current status (Feb 2026):** Both orchestrator and worker coaching operational. Orchestrator coaching: 1000+ metrics collected. Worker health tracking: verified working Feb 14 — stress test (50+ tool calls) emitted `context_usage` worker metric with zero orchestrator metric leakage. Fix required two opencode fork commits (459a1bfba, 0922edfe7) to: (1) read `x-opencode-env-ORCH_WORKER` header and set `session.metadata.role='worker'`, (2) pass `session.metadata` through `tool.execute.after` plugin hooks. **Note:** Worker detection only works for headless (OpenCode HTTP API) spawns — Claude CLI/tmux spawns bypass the HTTP session creation endpoint and don't get metadata set.
    
    ---
  - Critical Invariants:
    1. **Plugins cannot see LLM response text** - Only tool calls visible, not free-text responses. Fundamental constraint, not fixable.
    2. **Behavioral proxies are the only detection mechanism** - All pattern detection uses tool usage as proxy signals.
    3. **Metrics persist, session state doesn't** - JSONL file survives restarts, in-memory Map is ephemeral.
    4. **Observation coupled to intervention** - Injection only fires from `flushMetrics` within `tool.execute.after` hook.
    5. **Worker detection caching is one-way** - Only cache `true` results (confirmed worker), never cache `false`.
    6. **Two injection mechanisms serve different purposes** - `config.instructions` adds file references at config time (static context like skills), `client.session.prompt(noReply: true)` injects content at runtime (immediate coaching).
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Worker Detection Caching Bug (Jan 10-17)
    
    **Symptom:** Zero worker health metrics in production despite implemented code
    
    **Root cause:** `detectWorkerSession()` cached both `true` AND `false` results, permanently misclassifying workers if ANY tool call happened before a worker-identifying signal
    
    **Why it happens:**
    1. Worker session starts, first tool = `glob` (no detection signal)
    2. `isWorker = false` → cached → function returns `false` forever
    3. Second tool = `read SPAWN_CONTEXT.md` → cached result returned, detection skipped
    4. Worker treated as orchestrator for entire session
    
    **Cascade:**
    ```
    First non-matching tool call → cache false → subsequent detection signals ignored → worker health code never runs
    ```
    
    **Fix (Jan 17):** Only cache `true` results (confirmed worker), never cache `false`. Allow re-evaluation on each tool call until worker confirmed.
    
    **Pattern established:** "Never cache negative results in per-session detection"
    
    ---
    
    ### Failure Mode 2: Invalid Detection Signal (Bash workdir)
    
    **Symptom:** Detection code exists but never fires
    
    **Root cause:** Detection checked for `args.workdir` on bash tool, but bash tool has no `workdir` argument
    
    **Why it happens:**
    - Bash tool args are: `command`, `timeout`, `dangerouslyDisableSandbox`, `run_in_background`
    - No `workdir` argument exists
    - Detection signal `if (tool === "bash" && args?.workdir)` never matches
    
    **Fix (Jan 17):** Removed broken detection signal, restored file-path detection for any `.orch/workspace/` path
    
    ---
    
    ### Failure Mode 3: Observation Coupled to Intervention (Restart Brittleness)
    
    **Symptom:** Coaching messages stop after OpenCode server restart, even though metrics show problems
    
    **Root cause:** Injection is implemented as side effect of metric collection, not as separate concern that can operate independently
    
    **Why it happens:**
    - Metrics are **persistent** (JSONL file, survive restart)
    - Session state is **in-memory** (Map, lost on restart)
    - Injection logic **coupled to metric collection** (only happens via `flushMetrics`)
    - After restart: metrics file shows "poor" status, but no session state exists, so injection never fires
    
    **Cascade:**
    ```
    Server restart → session state lost → flushMetrics not called → injection doesn't fire → coaching stops
    ```
    
    **Architectural fix (not yet implemented):** Separate injection into independent daemon that reads metrics from JSONL and injects via OpenCode API, completely decoupled
    ... [truncated]
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-14-worker-detection-stress-test
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-stress-test.md
    - 2026-02-14-worker-detection-header-implementation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-header-implementation.md
    - 2026-02-14-metrics-redesign-architecture-validation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md
- Probe: Coaching Metrics Redesign Architecture Validation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md
  - Recent Probes:
    - 2026-02-14-worker-detection-stress-test
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-stress-test.md
    - 2026-02-14-worker-detection-header-implementation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-header-implementation.md
    - 2026-02-14-metrics-redesign-architecture-validation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
  - Summary:
    **At N=11, the model pattern shows exceptional consistency and proven utility.** All 11 models converged on the 6-section structure without enforcement. The enable/constrain query works across every domain tested. Most significantly: **the models that emerged reveal your cognitive investment priorities** - hot paths (spawn, agent, dashboard), strategic understanding (orchestrator, daemon), and owned complexity (completion, beads integration).
    
    **Key finding:** High investigation count + model existence = **friction that refused to resolve**. The absence of models for external dependencies (kb, tmux) despite high investigation counts reveals clear ownership boundaries.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- kb reflect Cluster Hygiene
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/kb-reflect-cluster-hygiene.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-14.
    Changed files: .kb/investigations/archived/, .kb/investigations/synthesized/, pkg/spawn/config.go.
    Verify model claims about these files against current code.
  - Summary:
    `kb reflect --type synthesis` clusters investigations by lexical similarity. This is a useful discovery signal, but it is not automatically a valid synthesis boundary. Effective triage requires a second step: classify each cluster by semantic cohesion, verify key claims against current code/behavior, then route to one of three dispositions: **converge** (create/update decision/model), **split** (separate mixed lineages), or **demote** (use probe/quick artifact instead of full investigation).
    
    ---
  - Critical Invariants:
    1. **Lexical cluster != conceptual model**
    2. **Code/test evidence outranks archived claims**
    3. **One canonical decision/model per mechanism**
    4. **Redundant investigations must point to canonical artifact**
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Lexical collision
    
    `feature` clusters can mix tiering behavior, cross-repo implementation tasks, and decision-gate debugging. Treating them as one topic creates noisy synthesis.
    
    ### Failure Mode 2: Time-drifted conclusions
    
    Investigation findings can become stale after code changes (for example, fail-open to fail-closed gate behavior). Re-validation is required during consolidation.
    
    ### Failure Mode 3: Artifact overuse
    
    Quick fact lookups become full investigations, increasing maintenance burden without adding durable understanding.
    
    ### Failure Mode 4: Incomplete closure metadata
    
    Archived files without clear `Superseded-By` pointers remain discoverable but ambiguous, causing repeated re-triage.
    
    ### Failure Mode 5: Scans archived/synthesized directories
    
    kb reflect scans `.kb/investigations/archived/` and `.kb/investigations/synthesized/` directories, creating false positives for already-processed clusters.
    
    **Evidence:** Extract synthesis (Jan 17) moved 14 investigations to `synthesized/code-extraction-patterns/`, but kb reflect still reported "13 investigations need synthesis" by scanning the synthesized directory.
    
    **Impact:** Agents repeatedly triage and investigate the same clusters thinking they need synthesis, when they've already been consolidated into guides.
    
    **Workaround:** Manually verify guide completeness and ignore kb reflect output for topics with synthesized/ or archived/ directories.
    
    **Fix needed:** kb-cli synthesis detection should exclude `.kb/investigations/archived/` and `.kb/investigations/synthesized/` from scanning.
    
    **Source:** `2026-01-17-inv-synthesize-extract-investigation-cluster-13.md:59-69`, `2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md`
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Models
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/README.md
- Dashboard Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-14.
    Changed files: cmd/orch/serve.go, web/src/routes/+page.svelte, web/src/lib/stores/agents.ts.
    Deleted files: cmd/orch/serve_agents.go.
    Verify model claims about these files against current code.
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
  - Recent Probes:
    - 2026-02-17-knowledge-tree-duplicate-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-17-knowledge-tree-duplicate-fix.md
    - 2026-02-16-work-graph-missing-store-methods
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-work-graph-missing-store-methods.md
    - 2026-02-16-work-graph-issues-view-section-design
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-work-graph-issues-view-section-design.md
    - 2026-02-16-three-view-consolidation-assessment
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-three-view-consolidation-assessment.md
    - 2026-02-16-knowledge-tree-tab-persistence
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-knowledge-tree-tab-persistence.md
- Probe: Orchestrator Skill Injection Path Trace
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
  - Recent Probes:
    - 2026-02-18-probe-skillc-pipeline-audit
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-probe-skillc-pipeline-audit.md
    - 2026-02-18-orchestrator-skill-cli-staleness-audit
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md
    - 2026-02-17-orchestrator-skill-injection-path-trace
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md
    - 2026-02-16-orchestrator-skill-orientation-redesign
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md
    - 2026-02-15-orchestrator-skill-deployment-sync
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-15-orchestrator-skill-deployment-sync.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-20.
    Deleted files: ~/.claude/skills/meta/orchestrator/SKILL.md.
    Verify model claims about these files against current code.
  - Summary:
    Anthropic restricts Opus 4.5 access via fingerprinting that blocks API usage but allows Claude Code CLI with Max subscription. This constraint forced a **dual spawn architecture**: primary path (OpenCode API + Sonnet/Flash, headless, high concurrency) and escape hatch (Claude CLI + Opus, tmux, crash-resistant). The escape hatch exists because critical infrastructure work (fixing the spawn system itself) can't depend on what might fail. Model choice now encodes reliability requirements, not just quality preferences.
    
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
    - 2026-02-20-model-aware-backend-routing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-model-aware-backend-routing.md
    - 2026-02-20-probe-default-backend-anthropic-incompatibility
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-probe-default-backend-anthropic-incompatibility.md
    - 2026-02-20-backend-resolution-architecture-drift
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-20-backend-resolution-architecture-drift.md
    - 2026-02-19-probe-opencode-mcp-wiring
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths/probes/2026-02-19-probe-opencode-mcp-wiring.md
- Probe: Headless Codex Provider Initialization Scoping
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md
  - Recent Probes:
    - 2026-02-20-probe-mcp-hot-reload-production-failure
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-20-probe-mcp-hot-reload-production-failure.md
    - 2026-02-18-probe-verify-session-metadata-api
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-verify-session-metadata-api.md
    - 2026-02-18-probe-state-create-rejected-promise-cache
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-state-create-rejected-promise-cache.md
    - 2026-02-18-probe-repro-headless-codex-spawn
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-repro-headless-codex-spawn.md
    - 2026-02-18-probe-headless-codex-provider-init
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md

### Guides (procedural knowledge)
- OpenCode Plugin System Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode-plugins.md
- Synthesis Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/synthesis-workflow.md
- Two-Tier Sensing Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/two-tier-sensing-pattern.md
- OpenCode Integration Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Beads Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/worker-patterns.md

### Related Investigations
- Investigate Skills Produce Investigation Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-investigate-skills-produce-investigation-artifacts.md
- Synthesize Synthesize Investigations 10 Synthesis
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md
- Orchestrator Coaching Plugin Technical Design
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/synthesized/coaching-plugin/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md
- Deep Analysis of OpenCode Coaching Plugin Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/synthesized/coaching-plugin/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md
- OpenCode Plugin Capabilities and Ecosystem
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md
- Synthesize Synthesis Investigations (26 Total)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-synthesis-investigations-26-synthesis.md
- Post-Synthesis Investigation Archival Workflow
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-design-post-synthesis-investigation-archival.md
- Test Coaching Plugin Pattern Detection
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/synthesized/coaching-plugin/2026-01-16-inv-orch-go-investigation-test-coaching.md
- Synthesis of 15 Skill Investigations (Dec 2025 - Jan 2026)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-synthesize-skill-investigations-15-synthesis.md
- Clean Up 10 Empty Investigation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-clean-up-10-empty-investigation.md

### Failed Attempts (DO NOT repeat)
- OpenCode auto-installs plugin dependencies

### Open Questions
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.






📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/).
Stage ONLY the specific files you created or modified for your task, by name.


1. **COMMIT YOUR WORK:**
   ```bash
   git add <files you changed>
   git commit -m "feat: [brief description of changes]"
   ```
2. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.


⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Worker rule: Commit your work, call `/exit`. Don't push.




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
1. Document it in your investigation file: "CONSTRAINT: [what constraint] - [why considering workaround]"
2. Include the constraint and your reasoning in SYNTHESIS.md
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
   - Create probe file in model's probes/ directory
   - Use probe template structure: Question, What I Tested, What I Observed, Model Impact
   - Your probe should confirm, contradict, or extend the model's claims



3. **UPDATE probe file** as you work:
   - Question: What model claim are you testing?
   - What I Tested: Actual command/code run (not just code review)
   - What I Observed: Actual output/behavior
   - Model Impact: Confirms/contradicts/extends which invariant

4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]


6. ⚡ SYNTHESIS.md is NOT required (light tier spawn).


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
<!-- Checksum: 7a8c26d4a43b -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-18 14:13:33 -->

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
bd comment orch-go-untracked-1771631191 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-untracked-1771631191 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-untracked-1771631191 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-untracked-1771631191 "Phase: BLOCKED - Need clarification on API contract"

# Report questions
bd comment orch-go-untracked-1771631191 "Phase: QUESTION - Should we use JWT or session-based auth?"
```

**When to report:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Additional context:**
Use `bd comment` for additional context, findings, or updates:
```bash
bd comment orch-go-untracked-1771631191 "Found performance bottleneck in database query"
bd comment orch-go-untracked-1771631191 "investigation_path: .kb/investigations/2026-02-11-perf-issue.md"
```

**Test Evidence Requirement:**
When reporting Phase: Complete, include test results in the summary:
- Example: `bd comment orch-go-untracked-1771631191 "Phase: Complete - Tests: go test ./... - 47 passed, 0 failed (2.3s)"`
- Example: `bd comment orch-go-untracked-1771631191 "Phase: Complete - Tests: npm test - 23 specs, 0 failures"`
- Example: `bd comment orch-go-untracked-1771631191 "Phase: Complete - Tests: make test - PASS (coverage: 78%)"`

**Why:** `orch complete` validates test evidence in phase comments. Vague claims like "all tests pass" trigger manual verification.

**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`.
- Workers report `Phase: Complete`, orchestrator verifies and closes
- Running `bd close` bypasses verification and breaks tracking

---

## Phase Reporting

**First 3 Actions (Critical):**
Within your first 3 tool calls, you MUST:
1. Report via `bd comment orch-go-untracked-1771631191 "Phase: Planning - [brief description]"`
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

---

## Session Complete Protocol

**When your work is done (all deliverables ready), complete in this EXACT order:**



1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment orch-go-untracked-1771631191 "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
4. Commit any remaining changes (including `VERIFICATION_SPEC.yaml`)
5. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.


**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

**Verification gates (orchestrator-run):**
After you report Phase: Complete, the orchestrator runs `orch complete` with two human gates:

- Gate 1 (explain-back): explain what was built and why.
- Gate 2 (behavioral, Tier 1 only): confirm behavior was verified.
  Make your Phase summary and VERIFICATION_SPEC.yaml clear enough to support both gates.

---

---
name: investigation
skill-type: procedure
description: Record what you tested and observed; default to model-scoped probes when injected model claims are present, otherwise run a full investigation.
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 9c3e89d2e927 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/src/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-18 09:18:39 -->

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

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.



1. **COMMIT YOUR WORK:** `git add <files you changed> && git commit -m "feat: [description]"`
2. `/exit`

⚡ LIGHT TIER: SYNTHESIS.md is NOT required.



⛔ **NEVER run `git push`** - Workers commit locally only.
⚠️ Your work is NOT complete until Phase: Complete is reported (or /exit for --no-track).

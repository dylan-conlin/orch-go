TASK: OpenCode API returns HTML instead of JSON for all /api/* endpoints. Repro: curl -s http://127.0.0.1:4096/api/sessions | head -3 returns HTML doctype. The SPA catch-all is swallowing API routes. This breaks all agent observability — orch status, orch review, dashboard swarm map, SSE stream. The spawn pipeline works fine (sessions create and execute), but nothing can observe them. Check the OpenCode fork at ~/Documents/personal/opencode/ — specifically the server routing in packages/opencode/src/server/server.ts. The API routes need to be matched before the SPA catch-all.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "opencode api returns"

### Constraints (MUST respect)
- orch-go agent state exists in four layers (OpenCode memory, OpenCode disk, registry, tmux)
  - Reason: Each layer has independent lifecycle - cleanup must touch all layers or ghosts accumulate
- OpenCode x-opencode-directory header returns ALL disk sessions, not just matching ones
  - Reason: API behavior is counterintuitive - without header returns in-memory only
- orch status can show phantom agents (tmux windows where OpenCode exited)
  - Reason: No reconciliation between tmux liveness and OpenCode session state
- OpenCode attach mode only creates sessions after first message received
  - Reason: TUI startup is not session creation - must send prompt before looking up session ID
- Stall detection: session.status=busy for >5min without message.part events indicates hung Claude API call
  - Reason: SSE monitoring pattern - healthy sessions emit regular message.part.updated events
- kb context command hangs on some queries
  - Reason: Blocks orch spawn from returning, use --skip-artifact-check as workaround
- GetAccountCapacity token comparison can fail when tokens drift between auth.json and accounts.yaml
  - Reason: External token rotation (by OpenCode) causes isActiveAccount check to fail, leading to auth.json being left with stale tokens after rotation

### Prior Decisions
- orch-go tmux spawn is fire-and-forget - no session ID capture
  - Reason: opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring.
- Use IsOpenCodeReady for tmux TUI detection
  - Reason: Reliably detects when OpenCode TUI is ready for input in a tmux pane, avoiding race conditions.
- orch-go CLI independence
  - Reason: CLI commands connect directly to OpenCode (4096), not orch serve (3333)
- OpenCode ListSessions WITH x-opencode-directory header returns disk sessions, WITHOUT returns in-memory
  - Reason: Finding from investigation - explains 2 vs 238 session count discrepancy
- Port allocation should use ranges by purpose (vite: 5173-5199, api: 3333-3399)
  - Reason: Prevents conflicts and makes purpose clear from port number
- orch-go is primary CLI, orch-cli (Python) is reference/fallback
  - Reason: Go provides better primitives (single binary, OpenCode HTTP client, goroutines); Python taught requirements through 27k lines and 200+ investigations
- Session ID resolution pattern
  - Reason: Commands that need to find agents should use resolveSessionID or the runTail pattern: workspace files first, then API lookup, then tmux fallback
- Post-registry lifecycle uses 4 state sources: OpenCode sessions, tmux windows, beads issues, workspaces
  - Reason: Registry removed due to false positive completion detection; derived lookups replace central state
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- CreateSession API now accepts model parameter for headless spawns
  - Reason: Headless mode was missing model selection capability - added Model field to CreateSessionRequest struct and threaded through CreateSession function to achieve parity with inline/tmux modes

### Models (synthesized understanding)
- OpenCode Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-session-lifecycle.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-12.
    Changed files: pkg/opencode/client.go, cmd/orch/spawn_cmd.go.
    Verify model claims about these files against current code.
  - Summary:
    OpenCode sessions persist across server restarts via disk storage at `~/.local/share/opencode/storage/`. Sessions are queried differently based on whether you need in-memory (running) or disk (historical) data. Completion detection relies on SSE `session.status` events transitioning from `busy` to `idle`, NOT session existence. The system supports three spawn modes (headless/tmux/inline) with different trade-offs for automation vs visibility.
    
    ---
  - Critical Invariants:
    1. **Sessions persist across restarts** - Disk storage at `~/.local/share/opencode/storage/`
    2. **Directory filtering is required for disk queries** - Without `x-opencode-directory` header, only get in-memory sessions
    3. **Completion is event-based** - Must watch SSE, can't infer from session state polling
    4. **Sessions never expire** - No TTL, cleanup is manual (`orch clean --sessions`)
    5. **Session directory is set at spawn** - Cross-project spawn bug: sessions get orchestrator's directory instead of `--workdir` target
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Cross-Project Sessions Show Wrong Directory
    
    **Symptom:** `orch spawn --workdir /other/project` creates session with orchestrator's directory
    
    **Root cause:** `spawn_cmd.go` doesn't pass `--workdir` value to OpenCode session creation
    
    **Why it happens:**
    - OpenCode sets session directory from CWD at spawn time
    - `--workdir` changes agent's working directory but not spawn caller's CWD
    - Session gets orchestrator's directory, not target project
    
    **Impact:** Sessions unfindable via `x-opencode-directory` header filtering
    
    **Fix needed:** Pass explicit directory to OpenCode session creation
    
    ### Failure Mode 2: Session Accumulation
    
    **Symptom:** 627 sessions accumulated over 3 weeks, slowing queries
    
    **Root cause:** OpenCode never deletes sessions, no automatic cleanup
    
    **Why it happens:**
    - Sessions persist indefinitely by design
    - No TTL or expiration mechanism
    - Dashboard queries all sessions (slow with 600+)
    
    **Fix (Jan 6):** `orch clean --sessions --days N` command to delete old sessions
    
    ### Failure Mode 3: Deprecated session.idle Event
    
    **Symptom:** Plugin code using `session.idle` event fails to detect completion
    
    **Root cause:** OpenCode changed event structure - `session.idle` is deprecated
    
    **Why it happens:**
    - Old event: `session.idle` (simple)
    - New event: `session.status` with `status.type === "idle"` (structured)
    - Breaking change, no migration guide
    
    **Fix (Jan 8):** Updated skills and plugins to use `session.status` event
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- OpenCode Fork
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork.md
  - Summary:
    Dylan owns a fork of [sst/opencode](https://github.com/sst/opencode) at `~/Documents/personal/opencode`. The fork is 16 custom commits ahead of upstream (linear forward — all upstream changes included). Custom changes focus on **memory management** (LRU/TTL instance eviction preventing 8.4GB unbounded growth), **SSE cleanup** (idempotent teardown preventing leaked connections), **OAuth stealth mode** (Claude Max access), **ORCH_WORKER header forwarding** (worker session detection), **session metadata** (extensible key-value storage), and **session TTL** (auto-expiry with periodic cleanup). The fork is a TypeScript monorepo (Bun runtime, Hono HTTP framework) with sessions stored in SQLite database (~/.local/share/opencode/opencode.db) using Drizzle ORM with migration-based schema management. Session status (idle/busy/retry) is tracked in-memory only, lost on server restart. A `GET /session/status` endpoint already exists for querying session state.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-18-probe-state-create-rejected-promise-cache
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-state-create-rejected-promise-cache.md
    - 2026-02-18-probe-headless-codex-provider-init
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md
- Probe: Agents API Missing phase and phase_reported_at Fields
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-agents-api-phase-field-missing.md
  - Recent Probes:
    - 2026-02-17-knowledge-tree-duplicate-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-17-knowledge-tree-duplicate-fix.md
    - 2026-02-16-knowledge-tree-tab-persistence
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-knowledge-tree-tab-persistence.md
    - 2026-02-16-attention-badge-verify-noise-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise-fix.md
    - 2026-02-16-agents-api-phase-field-missing
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-agents-api-phase-field-missing.md
    - 2026-02-16-knowledge-tree-ssr-window-check
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-knowledge-tree-ssr-window-check.md
- Model Access and Spawn Paths
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/model-access-spawn-paths.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-12.
    Changed files: pkg/spawn/config.go, CLAUDE.md.
    Deleted files: ~/.claude/skills/meta/orchestrator/SKILL.md.
    Verify model claims about these files against current code.
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
- Orchestration Cost Economics
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics.md
  - **STALENESS WARNING:**
    This model was last updated 2026-01-28.
    Deleted files: ~/.anthropic/, .kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md, pkg/spawn/backend.go.
    Verify model claims about these files against current code.
  - Summary:
    Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. DeepSeek V3 at $0.25/$0.38/MTok is now a **viable primary option** after testing confirmed stable function calling (contradicting earlier "unstable" documentation).
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Probe: Vector #2 — cleanUntrackedDiskSessions Removal and Current State
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
  - Recent Probes:
    - 2026-02-14-probe-vector2-cleanuntrackedsessions-removal
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
    - 2026-02-14-probe-vector7-sqlite-migration-json-fallback
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md
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
    - 2026-02-14-metrics-redesign-architecture-validation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-metrics-redesign-architecture-validation.md
    - 2026-02-14-worker-detection-stress-test
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-stress-test.md
    - 2026-02-14-worker-detection-header-implementation
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/coaching-plugin/probes/2026-02-14-worker-detection-header-implementation.md
- macOS Click Freeze
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze.md
  - Summary:
    Trackpad clicks stop registering while cursor movement and keyboard continue working. `sudo killall -HUP WindowServer` fixes it every time (HUP = reconfigure, not restart). This points to WindowServer accumulating corrupted state in its click event pipeline. **Breakthrough in Session 15:** nuclear elimination of ~23 services stopped the freeze. **Freeze returned (2026-02-13)** after gradual service re-enablement — first recurrence in ~2 days. **However (2026-02-14):** the same service set that triggered the Feb 13 freeze ran stable for 5+ hours, suggesting the freeze is **intermittent/stochastic** rather than deterministic. Frequency has decreased from every ~15 min (Sessions 11-14) to rare occurrences. macOS updated to 15.7.4 (from 15.6.1) between sessions. **H6 (aggregate service contention) remains the leading hypothesis** but is weakened by the stability observation. **New approach:** reactive capture script (`scripts/click-freeze-capture.sh`) to snapshot full system state during freeze occurrences for correlation analysis, rather than disruptive elimination testing.
    
    ---
  - Why This Fails:
    ### Hypothesis 1: yabai event interception corrupts WindowServer state — WEAKENED (RE-TESTING NEEDED)
    
    yabai uses the macOS Accessibility API to manage windows. Freeze recurred with yabai fully stopped in Session 14 (`yabai --stop-service`, confirmed no process via `pgrep`).
    
    **Evidence against:** Freeze recurred within ~30 minutes with yabai completely stopped (Session 14). No yabai process running. Seemed eliminated.
    
    **Evidence for re-testing (Session 16):** The Session 14 elimination was a single 30-min test with everything else still running. The environment is now different — yabai is running with a reduced service set (NI/Ollama gone, many agents disabled). yabai may not be the sole cause but could be a necessary component of H6 (aggregate contention). Worth re-testing by stopping yabai in current environment.
    
    ### Hypothesis 2: Karabiner DriverKit drops click events at kernel level — ELIMINATED
    
    Karabiner was fully uninstalled (app removed, DriverKit extension gone, no IOKit registry entries, no LaunchAgents). Freeze persisted immediately after fresh reboot with no Karabiner components present.
    
    **Evidence against:** Completely uninstalled — no process, no DriverKit extension (`systemextensionsctl list` clean), no IOKit entries (`ioreg` clean), no LaunchAgents. Freeze still occurred immediately post-reboot. Definitively eliminated.
    
    ### Hypothesis 3: WindowServer internal corruption (no external cause) — WEAKENED
    
    macOS 15.6.1 (Sequoia) may have a bug where WindowServer's click event routing table gets corrupted over time. This would be independent of any third-party software.
    
    **Evidence for:** Would explain why eliminating multiple apps in Session 11 didn't fix it.
    
    **Evidence against:** Three research probes (2026-02-11) searched GitHub, Reddit, Apple Discussions exhaustively — **zero matching reports** for this symptom pattern (clicks stop, cursor moves, HUP fixes). If this were a Sequoia bug, community reports would exist. This significantly weakens H3.
    
    ### Hypothesis 4: Memory pressure from OpenCode instance accumulation — EFFECTIVELY ELIMINATED
    
    OpenCode accumulates instances (with LSP/MCP/file watchers) per unique project directory. Each instance costs 300-500MB for LSP alone.
    
    **Evidence against (Session 15):**
    - Freeze occurred **immediately after fresh reboot** — memory was abundant, OpenCode hadn't even started yet
    - This strongly suggests memory pressure is NOT the primary cause
    - After disabling ~23 services,
    ... [truncated]
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-13-service-state-freeze-recurrence
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-13-service-state-freeze-recurrence.md
    - 2026-02-12-skhd-event-tap-source-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-12-skhd-event-tap-source-analysis.md
    - 2026-02-11-yabai-github-issues-search
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-11-yabai-github-issues-search.md
    - 2026-02-11-karabiner-github-search
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-11-karabiner-github-search.md
    - 2026-02-11-github-apple-support-search
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-11-github-apple-support-search.md
- Probe: Daemon Rollback Fail-Fast Fix
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-rollback-fail-fast-fix.md
  - Recent Probes:
    - 2026-02-17-daemon-test-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-test-fail-fast-fix.md
    - 2026-02-17-daemon-rollback-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-rollback-fail-fast-fix.md
    - 2026-02-17-daemon-completion-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-completion-fail-fast-fix.md
    - 2026-02-17-extraction-gate-fail-fast-fix
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-extraction-gate-fail-fast-fix.md
    - 2026-02-17-daemon-epic-expansion-fail-fast
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/daemon-autonomous-operation/probes/2026-02-17-daemon-epic-expansion-fail-fast.md
- Agent Lifecycle State Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-13.
    Changed files: cmd/orch/serve_agents.go, pkg/verify/check.go, .beads/issues.jsonl.
    Deleted files: pkg/registry/registry.go.
    Verify model claims about these files against current code.
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
  - Recent Probes:
    - 2026-02-17-dashboard-blind-to-tmux-agents
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-17-dashboard-blind-to-tmux-agents.md
    - 2026-02-14-backend-agnostic-session-contract
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/agent-lifecycle-state-model/probes/2026-02-14-backend-agnostic-session-contract.md

### Guides (procedural knowledge)
- OpenCode Integration Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode.md
- API Development Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/api-development.md
- OpenCode Plugin System Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/opencode-plugins.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Development Environment Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dev-environment-setup.md
- Agent Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/agent-lifecycle.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Status and Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/status-dashboard.md
- Background Services Performance Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/background-services-performance.md

### Related Investigations
- OpenCode Zen/Black Architecture, Economics, and Sustainability
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-13-research-opencode-zen-black-architecture-economics.md
- Addendum to Ecosystem Audit - OpenCode's Role
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-addendum-ecosystem-audit-opencode.md
- Explore Tradeoffs for orch-go OpenCode Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md
- Model Handling Conflicts Between orch-go and opencode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md
- Model Provider Architecture - orch vs OpenCode Auth Responsibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-model-provider-architecture-orch-vs.md
- Opencode Binary Resolution
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-design-opencode-binary-resolution.md
- Can We Auto-Refresh OpenCode OAuth Tokens?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-can-we-auto-refresh-opencode.md
- Cannot Query OpenCode Sessions from Other Projects
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-cannot-query-opencode-sessions-other.md
- OpenCode Crashes with "No user message found in conversation"
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-inv-opencode-crashes-no-user-message.md
- Evaluate Building API Proxy Layer for Claude Max Account Sharing
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-evaluate-building-api-proxy-layer.md

### Failed Attempts (DO NOT repeat)
- debugging Insufficient Balance error when orch usage showed 99% remaining
- Using empty string model resolution to let opencode decide
- orch clean to remove ghost sessions automatically
- OpenCode auto-installs plugin dependencies
- onDestroy for window cleanup in SvelteKit

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
The OpenCode server SPA catch-all is swallowing all /api/* routes, returning frontend HTML instead of JSON responses. This is the root cause of a cascade of observability failures: orch status shows phantom agents with wrong model/skill/runtime, orch review can't detect completions, dashboard Swarm Map shows 0 active agents, SSE stream receives no session events, work-graph shows unassigned instead of agent info. Spawn pipeline itself works (sessions are created and execute successfully), but nothing can observe them afterward. Discovered during price-watch codex spawn testing on 2026-02-18.

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: `bd comment orch-go-1033 "Reproduction verified: [describe test performed]"`

⚠️ A bug fix is only complete when the original reproduction steps pass.


🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-1033 "Phase: Planning - [brief description]"`
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
2. Run: `bd comment orch-go-1033 "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session

⚠️ Work is NOT complete until Phase: Complete is reported.
⚠️ The orchestrator cannot close this issue until you report Phase: Complete.



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
1. Surface it first: `bd comment orch-go-1033 "CONSTRAINT: [what constraint] - [why considering workaround]"`
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

2. [Task-specific deliverables]


3. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-debug-opencode-api-returns-18feb-642f/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Track progress via beads comments. Call /exit to close agent session when done.



## BEADS PROGRESS TRACKING (PREFERRED)

You were spawned from beads issue: **orch-go-1033**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-1033 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-1033 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-1033 "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-1033 "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-1033 "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-1033`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (systematic-debugging)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 2f5753c67dfd -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-17 12:10:22 -->


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
3. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
4. Commit any remaining changes (including `VERIFICATION_SPEC.yaml`)
5. Run: `/exit` to close the agent session

**Light Tier:** SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Author/update `VERIFICATION_SPEC.yaml` in the workspace root.
   - Fill the pre-populated skeleton with the exact commands you ran, expectations you verified, and any manual steps still required.
2. Run: `bd comment {{.BeadsID}} "Phase: Complete - "[1-2 sentence summary of deliverables]"` (report phase FIRST - before commit)
3. Ensure SYNTHESIS.md is created with these required sections:
   - **`Plain-Language Summary`** (REQUIRED): 2-4 sentences in plain language describing what you built/found/decided and why it matters. This is the scaffolding the orchestrator uses during completion review — write it for a human who hasn't read your code. No jargon without explanation. No "implemented X" without saying what X does.
   - **`Verification Contract`**: Link to `VERIFICATION_SPEC.yaml` and key outcomes
4. **Verify all .kb/ files are committed:**
   - Run: `git status --porcelain` and check for any .kb/ files (investigations, probes, decisions, etc.)
   - If uncommitted .kb/ files exist: `git add .kb/ && git commit -m "knowledge artifacts from session"`
   - This ensures probe files in .kb/models/{name}/probes/ are not left behind
5. Commit all remaining changes (including SYNTHESIS.md and `VERIFICATION_SPEC.yaml`)
6. Run: `/exit` to close the agent session
{{end}}

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility even if the agent dies before committing.

**Work is NOT complete until Phase: Complete is reported.**
The orchestrator cannot close this issue until you report Phase: Complete.

---






---
name: systematic-debugging
skill-type: procedure
description: Use when encountering any bug, test failure, or unexpected behavior, before proposing fixes - four-phase framework (root cause investigation, pattern analysis, hypothesis testing, implementation) that ensures understanding before attempting solutions
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 48c84fadd69e -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/src/worker/systematic-debugging/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-18 09:18:39 -->


## Summary

Four-phase debugging framework: Root Cause → Pattern Analysis → Hypothesis Testing → Implementation. Core principle: understand before fixing.

---

# Systematic Debugging

## Summary

Four-phase debugging framework: Root Cause → Pattern Analysis → Hypothesis Testing → Implementation. Core principle: understand before fixing.

---

## The Iron Law

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

If you haven't completed Phase 1, you cannot propose fixes.

---

## When to Use

Use for ANY technical issue: test failures, production bugs, unexpected behavior, performance problems, build failures, integration issues.

**Use ESPECIALLY when:**
- Under time pressure (emergencies make guessing tempting)
- "Just one quick fix" seems obvious
- Previous fixes didn't work
- You don't fully understand the issue

---

## Quick Reference

1. Check if specialized technique applies (see patterns section)
2. Check console/logs for errors - error may already be captured
3. Phase 1: Root cause investigation (understand WHAT and WHY)
4. Phase 2: Pattern analysis (working vs broken differences)
5. Phase 3: Hypothesis testing (form and test specific theory)
6. Phase 4: Implementation (failing test, fix root cause, verify)
7. Document and complete

**Red flag:** If thinking "quick fix for now" → STOP, return to Phase 1.


## Error Visibility (BEFORE Phase 1)

Check if errors have already been logged before investigating:

```bash
# Check project-specific error logs
tail -50 *.log 2>/dev/null
# Check build/test output
make test 2>&1 | tail -30
# Check runtime logs (if applicable)
docker logs <container> --tail 50 2>/dev/null
```

**If logs show relevant errors:**
1. Copy error details to investigation file
2. Use as starting point for Phase 1
3. You may already have root cause evidence

**If empty or unhelpful:** Proceed to Phase 1.

---

## Common Debugging Patterns

Before starting Phase 1, identify if a specialized technique applies:

| Pattern | Symptoms | Technique |
|---------|----------|-----------|
| **Deep call stack** | Error deep in execution, origin unclear, data corruption propagated | [techniques/root-cause-tracing.md](techniques/root-cause-tracing.md) |
| **Timing-dependent** | Flaky tests, race conditions, arbitrary timeouts, "works locally fails in CI" | [techniques/condition-based-waiting.md](techniques/condition-based-waiting.md) |
| **Invalid data propagation** | Bad data causes failures far from source, missing validation | [techniques/defense-in-depth.md](techniques/defense-in-depth.md) |

Load the appropriate technique for specialized guidance.


## The Four Phases

Complete each phase before proceeding to next.

### Phase 1: Root Cause Investigation

**Goal:** Understand WHAT and WHY

<!-- Inlined from: phases/phase1-root-cause.md -->
<!-- Original: **Load:** [phases/phase1-root-cause.md](phases/phase1-root-cause.md) -->

# Phase 1: Root Cause Investigation

**BEFORE attempting ANY fix:**

## 1. Read Error Messages Carefully

- Don't skip past errors or warnings
- They often contain the exact solution
- Read stack traces completely
- Note line numbers, file paths, error codes

## 2. Reproduce Consistently

- Can you trigger it reliably?
- What are the exact steps?
- Does it happen every time?
- If not reproducible → gather more data, don't guess

## 3. Check Recent Changes AND Pattern Recognition

**Recent changes:**
- What changed that could cause this?
- Git diff, recent commits
- New dependencies, config changes
- Environmental differences

**Pattern recognition check (whack-a-mole detection):**
- Search git history for similar fixes:
  - `git log --all --grep="[issue-type]" --oneline` (e.g., "timeout", "null check", "race condition")
  - `git log --all --grep="[component-name]" --oneline` (e.g., "proxy", "modal", "login")
- Check commit messages/diffs for this issue type
- **If 2+ previous fixes of same TYPE found → Whack-a-mole pattern detected**

**Whack-a-mole indicators:**
- Same issue type fixed in different locations (proxy timeout, modal timeout, API timeout)
- Incrementally adjusting same variable type (bumping timeouts, adding null checks, increasing retries)
- Each fix works temporarily but similar issues appear elsewhere
- Pattern of "just increase this value" fixes

**If whack-a-mole pattern detected:**
1. **STOP fixing symptoms**
2. Investigate systemic cause:
   - Missing centralized configuration?
   - Missing validation layer?
   - Architectural issue (tight coupling, shared mutable state)?
   - Environment-specific behavior not accounted for (proxy latency, network conditions)?
3. Design systematic solution BEFORE implementing fix
4. Document pattern in workspace under "Root Cause Analysis"
5. Escalate to orchestrator for systemic design if needed (may spawn `architect -i`)

**Example from real session:**
- Immediate issue: Modal timeout (2s → 10s fix needed)
- Git history check: Found 4 previous timeout fixes (proxy: 60s→120s, various other timeouts increased)
- Pattern recognized: Hardcoded timeouts fail with residential proxies (2-4s unpredictable latency)
- Systemic solution: Centralized timeout config with proxy multiplier (prevents future timeout whack-a-mole)

## 4. Gather Evidence in Multi-Component Systems

**WHEN system has multiple components (CI → build → signing, API → service → database):**

**BEFORE proposing fixes, add diagnostic instrumentation:**
```
For EACH component boundary:
  - Log what data enters component
  - Log what data exits component
  - Verify environment/config propagation
  - Check state at each layer

Run once to gather evidence showing WHERE it breaks
THEN analyze evidence to identify failing component
THEN investigate that specific component
```

**Example (multi-layer system):**
```bash
# Layer 1: Workflow
echo "=== Secrets available in workflow: ==="
echo "IDENTITY: ${IDENTITY:+SET}${IDENTITY:-UNSET}"

# Layer 2: Build script
echo "=== Env vars in build script: ==="
env | grep IDENTITY || echo "IDENTITY not in environment"

# Layer 3: Signing script
echo "=== Keychain state: ==="
security list-keychains
security find-identity -v

# Layer 4: Actual signing
codesign --sign "$IDENTITY" --verbose=4 "$APP"
```

**This reveals:** Which layer fails (secrets → workflow ✓, workflow → build ✗)

## 5. Layer Bias Anti-Pattern (Symptom Location ≠ Root Cause)

**CRITICAL:** Where symptoms appear is often NOT where root cause lives.

**Benchmark evidence (Jan 2026):** In a debugging task where admin logout didn't work:
- 4/6 AI models created frontend fixes (LoginPage.tsx, AdminLogin.tsx, etc.)
- Root cause was backend: missing `path="/"` in cookie operations
- Frontend was where symptom appeared; backend was where fix belonged

**Layer bias triggers:**
- UI shows wrong state → Check if state source (API/backend) is correct BEFORE touching UI
- Frontend behavior broken → Check if backend returns expected data FIRST
- Error visible in logs at layer N → Trace whether cause is at layer N-1

**Anti-pattern detection:**
- You're about to create a new frontend component to "handle" an auth issue
- You're adding UI workarounds for data that shouldn't be wrong
- You're fixing display logic when the displayed value is incorrect at source

**Countermeasure:** Before implementing frontend fix, verify:
1. Is backend returning correct data? (check API response)
2. Is state being set correctly at source? (check data flow)
3. Would fixing at source eliminate the need for frontend fix?

**Rule:** Fix at lowest layer that addresses root cause. UI fixes for backend bugs = symptom masking.

## 6. Trace Data Flow

**WHEN error is deep in call stack:**

**REQUIRED SUB-SKILL:** Use superpowers:root-cause-tracing for backward tracing technique

**Quick version:**
- Where does bad value originate?
- What called this with bad value?
- Keep tracing up until you find the source
- Fix at source, not at symptom

---

## Success Criteria for Phase 1

You understand:
- **WHAT** is broken (specific component, function, data)
- **WHY** it's broken (root cause, not symptom)
- **WHERE** the problem originates (source of bad data/state)

If you can't answer all three, continue investigating. Don't proceed to Phase 2.

Key activities:
- Read error messages carefully (stack traces completely)
- Reproduce consistently
- Check recent changes AND pattern recognition (whack-a-mole detection)
- In multi-component systems: add diagnostic instrumentation before fixing
- Trace data flow to source

**Success criteria:** You understand root cause, not just symptoms

---

### Phase 2: Pattern Analysis

**Goal:** Identify differences between working and broken

<!-- Inlined from: phases/phase2-pattern-analysis.md -->
<!-- Original: **Load:** [phases/phase2-pattern-analysis.md](phases/phase2-pattern-analysis.md) -->

# Phase 2: Pattern Analysis

**Find the pattern before fixing:**

## 1. Find Working Examples

- Locate similar working code in same codebase
- What works that's similar to what's broken?

## 2. Compare Against References

- If implementing pattern, read reference implementation COMPLETELY
- Don't skim - read every line
- Understand the pattern fully before applying

## 3. Identify Differences

- What's different between working and broken?
- List every difference, however small
- Don't assume "that can't matter"

## 4. Understand Dependencies

- What other components does this need?
- What settings, config, environment?
- What assumptions does it make?

---

## Success Criteria for Phase 2

You know:
- What the working pattern looks like
- Every difference between working and broken
- What dependencies and assumptions exist

If you can't articulate these differences, continue analyzing. Don't proceed to Phase 3.

Key activities:
- Find working examples in same codebase
- Read reference implementations COMPLETELY (don't skim)
- List every difference, however small
- Understand dependencies and assumptions

**Success criteria:** You know what's different and why it matters

---

### Phase 3: Hypothesis and Testing

**Goal:** Form and test specific hypothesis

<!-- Inlined from: phases/phase3-hypothesis-testing.md -->
<!-- Original: **Load:** [phases/phase3-hypothesis-testing.md](phases/phase3-hypothesis-testing.md) -->

# Phase 3: Hypothesis and Testing

**Scientific method:**

## 1. Form Single Hypothesis

- State clearly: "I think X is the root cause because Y"
- Write it down
- Be specific, not vague

## 2. Test Minimally

- Make the SMALLEST possible change to test hypothesis
- One variable at a time
- Don't fix multiple things at once

## 3. Verify Before Continuing

- Did it work? Yes → Phase 4
- Didn't work? Form NEW hypothesis
- DON'T add more fixes on top

## 4. When You Don't Know

- Say "I don't understand X"
- Don't pretend to know
- Ask for help
- Research more

---

## Success Criteria for Phase 3

Your hypothesis is:
- Specific (not vague guessing)
- Testable (can verify with minimal change)
- Based on evidence from Phase 1 & 2

If hypothesis is confirmed, proceed to Phase 4. If not, form new hypothesis based on test results.

Key activities:
- Form single hypothesis: "I think X is the root cause because Y"
- Test minimally (one variable at a time)
- Verify before continuing - didn't work? Form NEW hypothesis, don't add more fixes

**Success criteria:** Hypothesis confirmed or new hypothesis formed

---

### Phase 4: Implementation

**Goal:** Fix root cause, not symptom

<!-- Inlined from: phases/phase4-implementation.md -->
<!-- Original: **Load:** [phases/phase4-implementation.md](phases/phase4-implementation.md) -->

# Phase 4: Implementation

**Fix the root cause, not the symptom:**

## 1. Create Failing Test Case

- Simplest possible reproduction
- Automated test if possible
- One-off test script if no framework
- MUST have before fixing
- **REQUIRED SUB-SKILL:** Use superpowers:test-driven-development for writing proper failing tests

## 2. Implement Single Fix

- Address the root cause identified
- ONE change at a time
- No "while I'm here" improvements
- No bundled refactoring

## 3. Verify Fix

- Test passes now?
- No other tests broken?
- Issue actually resolved?

## 4. If Fix Doesn't Work

- STOP
- Count: How many fixes have you tried?
- If < 3: Return to Phase 1, re-analyze with new information
- **If ≥ 3: STOP and question the architecture (step 5 below)**
- DON'T attempt Fix #4 without architectural discussion

## 5. If 3+ Fixes Failed OR Whack-a-Mole Pattern Detected: Question Architecture

**Triggers for architectural discussion:**
- **3+ fix attempts in current session failed**
- **OR: 2+ similar fixes found in git history (whack-a-mole pattern from Phase 1)**
- Each fix reveals new shared state/coupling/problem in different place
- Fixes require "massive refactoring" to implement
- Each fix creates new symptoms elsewhere

**Pattern indicating architectural problem:**
- Same TYPE of issue keeps appearing (timeouts, null checks, race conditions)
- Each fix works locally but similar issues appear in different components
- Incremental parameter adjustments rather than root cause fixes
- "Just bump this value" becoming a recurring pattern

**STOP and question fundamentals:**
- Is this pattern fundamentally sound?
- Are we "sticking with it through sheer inertia"?
- Should we refactor architecture vs. continue fixing symptoms?
- Do we need centralized configuration/validation/infrastructure instead of scattered fixes?

**Discuss with your human partner before attempting more fixes**

This is NOT a failed hypothesis - this is a wrong architecture or missing infrastructure.

**Example systemic solutions:**
- Centralized configuration (timeout management, retry policies)
- Validation layers (defense in depth, fail-fast at boundaries)
- Architectural refactoring (remove tight coupling, eliminate shared mutable state)
- Infrastructure improvements (better error handling, observability, adaptive behavior)

---

## Common Rationalizations (All Wrong)

| Excuse | Reality |
|--------|---------|
| "Issue is simple, don't need process" | Simple issues have root causes too. Process is fast for simple bugs. |
| "Emergency, no time for process" | Systematic debugging is FASTER than guess-and-check thrashing. |
| "Just try this first, then investigate" | First fix sets the pattern. Do it right from the start. |
| "I'll write test after confirming fix works" | Untested fixes don't stick. Test first proves it. |
| "Multiple fixes at once saves time" | Can't isolate what worked. Causes new bugs. |
| "Reference too long, I'll adapt the pattern" | Partial understanding guarantees bugs. Read it completely. |
| "I see the problem, let me fix it" | Seeing symptoms ≠ understanding root cause. |
| "One more fix attempt" (after 2+ failures) | 3+ failures = architectural problem. Question pattern, don't fix again. |

---

## your human partner's Signals You're Doing It Wrong

**Watch for these redirections:**
- "Is that not happening?" - You assumed without verifying
- "Will it show us...?" - You should have added evidence gathering
- "Stop guessing" - You're proposing fixes without understanding
- "Ultrathink this" - Question fundamentals, not just symptoms
- "We're stuck?" (frustrated) - Your approach isn't working

**When you see these:** STOP. Return to Phase 1.

---

## When Process Reveals "No Root Cause"

If systematic investigation reveals issue is truly environmental, timing-dependent, or external:

1. You've completed the process
2. Document what you investigated
3. Implement appropriate handling (retry, timeout, error message)
4. Add monitoring/logging for future investigation

**But:** 95% of "no root cause" cases are incomplete investigation.

---

## Success Criteria for Phase 4

- Failing test created and verified to fail
- Single fix implemented addressing root cause
- Test now passes
- No other tests broken
- Issue actually resolved (not just symptoms masked)

Key activities:
- Create failing test case
- Implement single fix
- **Smoke-test end-to-end** (critical - see below)
- If 3+ fixes failed: question architecture

**Success criteria:** Bug resolved, tests pass, smoke-test confirms real fix

---

## Smoke-Test Requirement

**Before claiming fix is complete, you MUST:**
1. Run the actual failing scenario that triggered debugging
2. Verify expected behavior now occurs
3. Document smoke-test in completion comment

**Valid:** "Bug: CLI crashes on --mcp" → Run `orch spawn --mcp`, verify no crash
**Invalid:** "Unit tests pass" (necessary but not sufficient)

**If cannot smoke-test:** Document WHY in completion comment.


## Visual Debugging Tools

### snap - Screenshot CLI (Recommended)

```bash
snap                          # Capture screen, returns file path
snap list --json              # Find window IDs
snap window "Firefox"         # Capture by app name
snap --json                   # JSON output: {"path": "/path/to/screenshot.png"}
```

**Use for:** Verifying UI state, documenting visual bugs, smoke-testing UI changes.

**Advantage:** Zero context cost (returns file path, not image data).

### Browser Automation

**USE:** Glass MCP - connects to your actual Chrome tabs via DevTools Protocol

**FALLBACK:** Playwright MCP - for headless/CI scenarios

**AVOID:** browser-use MCP - causes context explosion (screenshots, full DOM)

**Decision flow:**
1. Need visual verification? → `snap` (zero context cost)
2. Need browser automation (clicking, typing, DOM inspection)? → Glass MCP (spawned with --mcp glass)
3. Need headless/CI testing? → Playwright MCP
4. Need DevTools console errors? → Glass MCP (glass_page_state tool)

**Glass advantages:**
- Connects to your actual Chrome (not headless)
- Auto-check DevTools console errors
- Inspect live DOM state
- CLI commands for validation gates (glass assert)


## Model Awareness (Probe vs Investigation Routing)

**Before creating any artifact, check SPAWN_CONTEXT.md for model-claim markers.**

### Detection

Find the `### Models (synthesized understanding)` section in SPAWN_CONTEXT.md. Look for injected model-claim markers in model entries:
- `- Summary:`
- `- Critical Invariants:` or `- Constraints:`
- `- Why This Fails:` or `- Failure Modes:`

### If markers are present → Probe Mode

Your debugging findings likely confirm, contradict, or extend an existing model's failure modes.

- Pick the most relevant model from the injected models section
- Create: `.kb/models/{model-name}/probes/{date}-{slug}.md`
- Use template: `.orch/templates/PROBE.md`
- Required sections: `Question`, `What I Tested`, `What I Observed`, `Model Impact`
- Focus the probe on which model claim (especially failure modes) your debugging confirms or contradicts

**Example:** Debugging a spawn failure when the spawn model documents "Failure Mode 2: Header Injection Conflicts" → create a probe testing whether that failure mode explains the current bug.

### If markers are absent → Investigation Mode

Follow standard investigation workflow below.

---

## Investigation File (Optional for Simple Bugs)

Investigation files are **recommended** for complex bugs but **optional** for simple fixes.

### When to Create

**Create when:**
- Multi-step root cause analysis needed
- Multiple hypotheses to test
- Findings should be preserved
- Pattern may recur (for synthesis)

**Skip when:**
- Bug is obvious and localized (typo, wrong variable)
- Fix completes in <15 minutes
- Root cause immediately clear from error
- Commit message can fully document fix

### Create Template (if needed)

```bash
kb create investigation "debug/topic-in-kebab-case"
```

**After creating:**
1. Fill Question field with specific bug description
2. Document findings progressively during Phases 1-4
3. Update Confidence and Resolution-Status as you progress
4. Set Resolution-Status when complete (Resolved/Mitigated/Recurring)

### Commits-Only Completion

If skipping investigation file, ensure descriptive commits:
- Include "why" not just "what"
- Example: `fix: handle null session in auth middleware - was causing silent failures when Redis connection dropped`


## Self-Review (Mandatory)

After implementing fix, perform self-review before completion.

### Pattern Scope Verification

**If bug was a pattern that could exist elsewhere:**

```bash
# Check for pattern occurrences
rg "bug_pattern"                    # Should be 0 or documented exceptions
rg "range\(len\(" --type py         # Off-by-one example
rg "timeout.*=.*[0-9]" --type py    # Hardcoded timeout example
```

**Skip if:** Bug was truly one-off (typo, unique logic error).

### Debugging-Specific Checks

| Check | If Failed |
|-------|-----------|
| Root cause addressed (not symptom) | Return to Phase 1 |
| No debug code left (console.log, print) | Remove before commit |
| No temporary workarounds ("TODO: fix properly") | Complete the fix |
| Regression test exists | Add test |
| Investigation documented | Update file |

### Standard Checks

- [ ] No hardcoded secrets
- [ ] No injection vulnerabilities
- [ ] Conventional commit format (`fix:`, `test:`)
- [ ] Atomic commits

### Discovered Work

If you found related bugs, tech debt, or strategic unknowns:

```bash
bd create "description" --type bug    # or --type task
bd create "description" --type question # for architectural/premise questions
bd label <id> triage:review           # default label for review
```

**Note "No discovered work" in completion if nothing found.**

### Report via Beads

```bash
# If issues found and fixed:
bd comment <beads-id> "Self-review: Fixed [issue summary]"

# If passed:
bd comment <beads-id> "Self-review passed - ready for completion"
```


## Fix-Verify-Fix Cycle (Atomic Debugging)

**Fix + Verify = One Unit of Work**

Do NOT:
- Implement fix → claim complete → wait for new spawn if fails
- "Fix is done, verification is a separate task"

DO:
- Implement fix → verify immediately → if fails, iterate
- Only claim complete when smoke-test passes

### When to Iterate vs Escalate

**Keep iterating if:**
- Verification reveals related issue in same area
- Fix was incomplete but direction correct
- You understand why it failed

**Escalate if:**
- 3+ fix attempts failed (questioning architecture needed)
- Root cause was misidentified (return to Phase 1)
- Issue outside your scope/authority

### Reporting During Iteration

```bash
bd comment <beads-id> "Fix attempt 1: [what tried] - Result: [pass/fail + why]"
bd comment <beads-id> "Fix attempt 2: [refined approach] - Result: [pass/fail]"
# Only when actually working:
bd comment <beads-id> "Phase: Complete - Fix verified via [smoke-test description]"
```

---

## Red Flags - STOP and Follow Process

If you catch yourself thinking:
- "Quick fix for now, investigate later"
- "Just try changing X and see if it works"
- "Skip the test, I'll manually verify"
- "I don't fully understand but this might work"
- "One more fix attempt" (when already tried 2+)
- Each fix reveals new problem in different place

**ALL mean: STOP. Return to Phase 1.**


## Completion Criteria

Before marking complete, verify ALL:

- [ ] **Root cause identified** - Documented in investigation OR commit message
- [ ] **Fix implemented** - Addresses root cause, not symptoms
- [ ] **Tests passing** - Including reproduction test, with **actual test output documented**
- [ ] **Smoke-test passed** - Actual failing scenario now works
- [ ] **Self-review passed** - Pattern scope, no debug code, no workarounds
- [ ] **Discovered work reviewed** - Tracked or noted "No discoveries"
- [ ] **Phase reported with test evidence** - `bd comment <beads-id> "Phase: Complete - Tests: <cmd> - <output>"` (BEFORE final commit)
- [ ] **Git clean** - `git status` shows "nothing to commit"

**If ANY unchecked, work is NOT complete.**

### After All Criteria Met (in this EXACT order)

```bash
# 1. Report phase FIRST (before commit) - prevents agent death race condition
# Include ACTUAL test output, not just "tests passing"
bd comment <beads-id> "Phase: Complete - Root cause: [X], Fix: [Y], Tests: go test ./... - 23 passed, 0 failed"

# 2. Commit any final changes
git add . && git commit -m "fix: [description]"

# 3. Exit
/exit
```

**Test Evidence Requirement:**
- Format: `Tests: <command> - <actual output summary>`
- Good: `Tests: go test ./... - 23 passed, 0 failed`
- Good: `Tests: npm test - 15 specs, 0 failures`
- Bad: `Tests passing` (no command, no numbers)

**Why:** `orch complete` validates test evidence in comments. Vague claims trigger manual verification.

**Why this order matters:** If the agent dies after commit but before reporting Phase: Complete, the orchestrator cannot detect completion. Reporting phase first ensures visibility.

---

## Fast-Path Alternative

For clearly localized, trivial failures (import path error, undefined name, obvious single-file fix):

Use `quick-debugging` skill instead. It provides lightweight loop with strict escalation back to systematic-debugging if first attempt fails.

**Use quick-debugging when:**
- Scope ≤ 2 files and ≤ 1 hour
- Error points directly at fix location
- No architectural uncertainty

**If 3+ fixes failed:** Question architecture, return to Phase 1.






---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md


## LOCAL SERVERS

**Project:** orch-go
**Status:** running

**Ports:**
- **api:** http://localhost:3348
- **web:** http://localhost:5188

**Quick commands:**
- Start servers: `orch servers start orch-go`
- Stop servers: `orch servers stop orch-go`
- Open in browser: `orch servers open orch-go`



🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `bd comment orch-go-1033 "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

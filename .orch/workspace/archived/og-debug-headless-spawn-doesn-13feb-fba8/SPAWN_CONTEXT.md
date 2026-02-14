TASK: Headless spawn doesn't set agent working directory to --workdir/cwd. PROJECT_DIR=/Users/dylanconlin/Documents/personal/orch-go. SESSION SCOPE: Small. BUG: When spawning headless agents for cross-project work (e.g., orch spawn --workdir ~/price-watch feature-impl 'task'), the agent's shell starts in orch-go instead of the target directory. PROJECT_DIR in SPAWN_CONTEXT.md is just prompt text — it doesn't control the actual session working directory. The OpenCode HTTP API session creation must be passing the wrong cwd or not passing one at all. INVESTIGATE: 1) Find where headless sessions are created — likely in pkg/spawn/ or a backends/ file. Look for the OpenCode API call that creates/starts a session. 2) Check if there's a 'workdir' or 'directory' or 'cwd' parameter being passed. 3) Check how the --workdir flag from cmd/orch/spawn.go flows to the backend. 4) Fix: ensure the headless backend passes the correct working directory to the OpenCode session API. EVIDENCE: Agent pw-feat-build-scsoauthclient-service-13feb-7c9d started in /Users/dylanconlin/Documents/personal/orch-go instead of /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch. bd comment pw-w7z0 failed because it was looking in wrong .beads/. VERIFY: After fix, go build ./cmd/orch/ must pass. If you can write a unit test for workdir propagation, do so. IN SCOPE: workdir propagation in headless spawn only. OUT OF SCOPE: tmux spawn, skill changes, anything else.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "headless spawn doesn"

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
- Headless Swarm = batch execution + rate-limit management across accounts
  - Reason: User clarified scope: focus on concurrent agent spawning with capacity management, not distributed architecture or multi-model routing
- orch spawn context delivery is reliable
  - Reason: Verified that SPAWN_CONTEXT.md is correctly populated and accessible by the agent
- Cross-project epics use Option A: epic in primary repo, ad-hoc spawns with --no-track in secondary repos, manual bd close with commit refs
  - Reason: Only working pattern today. Beads multi-repo hydration is read-only aggregation, bd repo commands are buggy.
- Tmux spawn uses opencode attach mode
  - Reason: Enables dual TUI+API access - sessions visible via orch status while still showing TUI for visual monitoring
- SPAWN_CONTEXT.md is 100% redundant - generated from beads + kb context + skill + template
  - Reason: Investigation confirmed all content exists elsewhere and can be regenerated at spawn time
- Progressive disclosure for skill bloat
  - Reason: 89% of feature-impl spawns use only 2-3 phases. Extract detailed phase guidance to reference docs, keep core workflow inline. Proven pattern from 2025-11-21 instruction optimization. Reduces 1757→~500 lines.
- CompletionService bridges SSE completion detection and slot management for headless agents
  - Reason: Headless agents need automatic slot release when they complete; Monitor detects completions, WorkerPool manages slots, CompletionService links them
- Tiered spawn protocol uses .tier file in workspace for orch complete
  - Reason: Allows VerifyCompletion to read tier from workspace and skip SYNTHESIS.md requirement for light-tier spawns without requiring orchestrator to pass tier explicitly
- When spawned for cross-repo work, verify work completion status before starting
  - Reason: Task orch-go-oo1f: spawned in orch-go for work in orch-knowledge. Template was already retired (commit 7430185) before agent fully engaged. Quick verification could have saved agent context.

### Models (synthesized understanding)
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
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
  - Summary:
    **At N=11, the model pattern shows exceptional consistency and proven utility.** All 11 models converged on the 6-section structure without enforcement. The enable/constrain query works across every domain tested. Most significantly: **the models that emerged reveal your cognitive investment priorities** - hot paths (spawn, agent, dashboard), strategic understanding (orchestrator, daemon), and owned complexity (completion, beads integration).
    
    **Key finding:** High investigation count + model existence = **friction that refused to resolve**. The absence of models for external dependencies (kb, tmux) despite high investigation counts reveals clear ownership boundaries.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
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
- OpenCode Session Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-session-lifecycle.md
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
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Orchestration Cost Economics
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics.md
  - Summary:
    Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. DeepSeek V3 at $0.25/$0.38/MTok is now a **viable primary option** after testing confirmed stable function calling (contradicting earlier "unstable" documentation).
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.

### Guides (procedural knowledge)
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- How Spawn Works
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawn.md
- Headless Spawn Mode Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/headless.md
- Tmux Spawn Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/tmux-spawn-guide.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/worker-patterns.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md

### Related Investigations
- Test Headless Mode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-headless-mode.md
- Test Spawn Functionality
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-23-inv-test-spawn-functionality.md
- Headless Spawn Mode Readiness What
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-headless-spawn-mode-readiness-what.md
- tmux concurrent zeta - 6th concurrent spawn test
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-zeta.md
- Concurrent tmux spawn test (delta)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-delta.md
- Test Headless Spawn
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-headless-spawn.md
- CI Implement Role Aware Injection (4th+ Duplicate Spawn - Systemic Issue)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-ci-implement-role-aware-injection.md
- Verify Spawn Works
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-verify-spawn-works.md
- Analyze Spawn Value Ratio
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-analyze-spawn-value-ratio.md
- Tmux Concurrent Epsilon Spawn Capability
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-20-inv-tmux-concurrent-epsilon.md

### Failed Attempts (DO NOT repeat)
- orch tail on tmux agent
- BEADS_NO_DAEMON=1 in .zshrc only

### Open Questions
- When should orchestrator use 'orch send' for follow-up vs spawn fresh investigation? What are session expiration limits, context degradation patterns, and phase reporting implications?

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.




## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
Problem: When spawning headless agents for cross-project work (e.g., from orch-go orchestrator targeting price-watch), the agent's shell starts in orch-go (likely the daemon's cwd or a hardcoded default) instead of the target project directory. PROJECT_DIR is included in SPAWN_CONTEXT.md as prompt text, but this doesn't control the actual working directory of the spawned session. The agent has to manually cd, and bd commands fail before it does (e.g., 'bd comment pw-w7z0' fails because it's looking in orch-go's .beads/ which has no pw- prefix issues). Evidence: Agent pw-feat-build-scsoauthclient-service-13feb-7c9d started in /Users/dylanconlin/Documents/personal/orch-go instead of /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch. SPAWN_CONTEXT.md had correct PROJECT_DIR but shell cwd was wrong. Agent had to cd manually and bd comment failed before it did. Impact: P1 because this affects ALL cross-project headless spawns. Agents waste context discovering wrong directory, bd phase reporting fails, and early commands operate on wrong project's beads DB. Fix: The headless spawn backend (OpenCode HTTP API) needs to set the session's working directory to the --workdir flag value or the orchestrator's cwd at spawn time.

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: `bd comment orch-go-xwk "Reproduction verified: [describe test performed]"`

⚠️ A bug fix is only complete when the original reproduction steps pass.


🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment orch-go-xwk "Phase: Planning - [brief description]"`
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
2. Run: `bd comment orch-go-xwk "Phase: Complete - [1-2 sentence summary of deliverables]"`
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
1. Surface it first: `bd comment orch-go-xwk "CONSTRAINT: [what constraint] - [why considering workaround]"`
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
2. **SET UP investigation file:** Run `kb create investigation headless-spawn-doesn-set-agent` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-headless-spawn-doesn-set-agent.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

   - **IMPORTANT:** After running `kb create`, report the actual path via:
     `bd comment orch-go-xwk "investigation_path: /path/to/file.md"`
     (This allows orch complete to verify the correct file)

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-debug-headless-spawn-doesn-13feb-fba8/SYNTHESIS.md
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

You were spawned from beads issue: **orch-go-xwk**

**Use `bd comment` for progress updates instead of workspace-only tracking:**

```bash
# Report progress at phase transitions
bd comment orch-go-xwk "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-xwk "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-xwk "Phase: Complete - All tests passing, ready for review"

# Report blockers immediately
bd comment orch-go-xwk "BLOCKED: Need clarification on API contract"

# Report questions
bd comment orch-go-xwk "QUESTION: Should we use JWT or session-based auth?"
```

**When to comment:**
- Phase transitions (Planning → Implementing → Testing → Complete)
- Significant milestones or findings
- Blockers or questions requiring orchestrator input
- Completion summary with deliverables

**Why beads comments:** Creates permanent, searchable progress history linked to the issue. Orchestrator can track progress across sessions via `bd show orch-go-xwk`.

⛔ **NEVER run `bd close`** - Only the orchestrator closes issues via `orch complete`.
   - Workers report `Phase: Complete`, orchestrator verifies and closes
   - Running `bd close` bypasses verification and breaks tracking



## SKILL GUIDANCE (systematic-debugging)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: b2c6c2b0e210 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/shared/worker-base/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-12 22:06:41 -->


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
name: systematic-debugging
skill-type: procedure
description: Use when encountering any bug, test failure, or unexpected behavior, before proposing fixes - four-phase framework (root cause investigation, pattern analysis, hypothesis testing, implementation) that ensures understanding before attempting solutions
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: a0f9bf3b4203 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/src/worker/systematic-debugging/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-06 15:35:56 -->


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
2. `bd comment orch-go-xwk "Phase: Complete - [1-2 sentence summary]"`
3. `/exit`


⚠️ Your work is NOT complete until you run these commands.

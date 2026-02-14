TASK: Create a new model at /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors.md documenting all known vectors by which an active OpenCode session can become unfindable (deleted or evicted from memory), causing NotFoundError crashes in the TUI.

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go
CODEBASE TO EXAMINE: Both /Users/dylanconlin/Documents/personal/orch-go AND /Users/dylanconlin/Documents/personal/opencode
SESSION SCOPE: Medium (2-4h)

THIS IS A MODEL CREATION TASK. Use the model template at /Users/dylanconlin/Documents/personal/orch-go/.kb/templates/MODEL.md as the format guide.

CONTEXT - WHAT WE KNOW SO FAR (from orchestrator synthesis of multiple investigations):

The bug: Dylan's interactive orchestrator session in the OpenCode TUI crashes with NotFoundError: Session not found mid-conversation. This has been happening for weeks, predating the Feb 13 upstream SQLite migration.

CONFIRMED DELETION VECTORS (from prior investigations):

1. DISK CLEANUP SCRIPTS (CONFIRMED ROOT CAUSE - NOW FIXED):
   - disk-cleanup.sh (runs hourly via launchd com.dylan.disk-cleanup) previously had lines that deleted session JSON files directly from ~/.local/share/opencode/storage/
   - These lines are now commented out with note: 'REMOVED: These were deleting files out from under active OpenCode sessions, causing orchestrator TUI crashes (NotFoundError on session .json files). See: opencode-prune.sh root cause analysis, Feb 2026.'
   - BUT the bug persists after this fix, meaning there are OTHER vectors

2. orch clean --verify-opencode GAP (CONFIRMED - NOT YET FIXED):
   - cleanOrphanedDiskSessions() has 3-layer defense but Layer 2 has a gap
   - Sessions idle >5 minutes WITHOUT a .session_id workspace file bypass the IsSessionProcessing() check and get DELETED
   - Interactive orchestrator sessions (TUI) do NOT have workspaces, so they have NO Layer 1 protection
   - This is the most likely CURRENT cause of the ongoing bug
   - Source: cmd/orch/clean_cmd.go:516-644, investigated 2026-02-13

3. orch clean --sessions (daemon periodic cleanup):
   - CleanStaleSessions with 7-day threshold, runs every 6 hours
   - PreserveOrchestrator: true in daemon config
   - SAFE for active sessions - 7 day threshold far exceeds any active session

4. Session.remove() via DELETE API:
   - Only explicit deletion path in OpenCode: session/index.ts:569-589
   - Exposed at DELETE /session/:id route
   - TUI Ctrl+D keybind dual-bound to app_exit AND session_delete (accidental deletion risk)

5. CASCADE DELETE via project deletion:
   - SessionTable has onDelete: cascade to ProjectTable
   - No code path found that deletes projects
   - THEORETICAL vector only

6. INSTANCE EVICTION (LRU/TTL):
   - Commit c3c84c411 in Dylan's fork added instance eviction to prevent 8.4GB memory growth
   - disposeCurrent() cleans up in-memory State but investigation claims no session deletion
   - UNTESTED HYPOTHESIS: Does disposing a project instance make its sessions unfindable via in-memory lookup even though they still exist on disk/in SQLite? The NotFoundError could mean 'not in memory map' not 'not on disk'
   - This is the LEAST investigated vector and needs probing

7. STORAGE MIGRATION (Feb 13):
   - Upstream commit 6d95f0d14 rewrote session storage from JSON files to SQLite
   - Both storage systems currently coexist (old JSON files + new SQLite DB)
   - Could create confusion about where to look up sessions
   - This is RECENT and post-dates the bug onset, so not the original cause but could create NEW failure modes

WHAT THE MODEL SHOULD CAPTURE:

The model should document:
1. Core mechanism: How sessions are stored (dual: JSON files + SQLite after migration), how they are looked up (in-memory map vs disk/DB), and the gap between 'exists on disk' and 'findable in memory'
2. All deletion/eviction vectors in a single table with: vector name, trigger, protection, gap, risk level, status (fixed/open/theoretical)
3. Why this fails: The fundamental problem is that MULTIPLE independent systems can delete/evict sessions and there is no 'session is active, do not touch' lock
4. The in-memory vs on-disk distinction: NotFoundError may mean 'evicted from memory map' not 'deleted from storage' - this is the key architectural insight
5. Constraints: Why we cant just add a lock (multi-process SQLite, instance eviction for memory), why the TUI is especially vulnerable (no workspace tracking)
6. Evolution: disk-cleanup.sh fix -> orch clean gap discovery -> instance eviction hypothesis -> SQLite migration complication

KEY FILES TO EXAMINE:
- /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts (Session.remove, session lookup)
- /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/project/instance.ts (instance eviction)
- /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/project/state.ts (State.dispose)
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/clean_cmd.go (all cleanup paths)
- /Users/dylanconlin/Documents/personal/orch-go/pkg/cleanup/sessions.go (stale session cleanup)
- /Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/cleanup.go (daemon cleanup wrapper)
- ~/bin/disk-cleanup.sh (launchd script - commented out lines)

PRIOR INVESTIGATIONS TO REFERENCE:
- .kb/investigations/2026-02-14-inv-active-orchestrator-session-deleted-while.md
- .kb/investigations/2026-02-13-inv-verify-whether-orch-clean-kills-headless-sessions.md
- .kb/models/opencode-session-lifecycle.md (existing model - stale, needs cross-reference)

DELIVERABLE: A model file at /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors.md following the MODEL.md template format. The model must be actionable - someone reading it should immediately understand which vectors are open risks and what to fix.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "model users dylanconlin"

### Constraints (MUST respect)
- Concurrent agents trigger TPM throttling at >60% session usage
  - Reason: Observed performance degradation and user reports of hitting limits during swarm operations
- Orchestrator is AI, not Dylan - Dylan interacts with AI orchestrators who spawn/complete agents
  - Reason: Investigation 2025-12-25-design-orchestrator-completion-lifecycle-two incorrectly framed Dylan as the actor who spawns agents. The actor model is: Dylan ↔ AI Orchestrator ↔ Worker Agents. Mental model sync flows: Agent→Orchestrator (synthesis) and Orchestrator→Dylan (conversation).

### Prior Decisions
- Headless Swarm = batch execution + rate-limit management across accounts
  - Reason: User clarified scope: focus on concurrent agent spawning with capacity management, not distributed architecture or multi-model routing
- Three-tier temporal model (ephemeral/persistent/operational) organizes artifact placement
  - Reason: Artifacts live where their lifecycle dictates - session-bound to workspace, project-lifetime to kb, work-in-progress to beads
- orch status shows PHASE and TASK columns from beads data
  - Reason: Makes output actionable - users can immediately see what each agent is doing
- Headless spawn mode is production-ready
  - Reason: All 5 requirements verified working: status detection, monitoring, completion detection, error handling, user visibility. Investigation orch-go-0r2q confirmed no blockers exist.
- CreateSession API now accepts model parameter for headless spawns
  - Reason: Headless mode was missing model selection capability - added Model field to CreateSessionRequest struct and threaded through CreateSession function to achieve parity with inline/tmux modes
- Document all spawn modes in CLAUDE.md
  - Reason: Users need to see all options (headless default, --tmux opt-in, --inline blocking) to make informed spawn choices
- Help text should be organized by mode hierarchy (default → opt-in → blocking)
  - Reason: Users need to see default behavior first; mode grouping in examples improves discoverability and understanding
- Dashboard should use progressive disclosure (Active/Recent/Archive sections) for session management
  - Reason: Balances operational visibility (active work always visible) with historical debugging (expand sections as needed) and UI clarity (collapsed sections reduce clutter). Only approach that satisfies all three user contexts: development focus, debugging history, and health monitoring.
- OpenCode model selection is per-message, not per-session
  - Reason: Intentional design enabling mid-conversation model switching, cost optimization, and flexibility. Pass model to prompt/message calls, not session creation.
- Binary staleness should be prevented with make install
  - Reason: Project tracks ./orch binary in git, but build process creates build/orch. Users should run 'make install' to sync ~/bin/orch rather than manually copying to root directory.
- Git-Diff Staleness Detection for Model Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-02-14-model-staleness-detection.md
- Models as Understanding Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-12-models-as-understanding-artifacts.md

### Models (synthesized understanding)
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
  - Summary:
    **At N=11, the model pattern shows exceptional consistency and proven utility.** All 11 models converged on the 6-section structure without enforcement. The enable/constrain query works across every domain tested. Most significantly: **the models that emerged reveal your cognitive investment priorities** - hot paths (spawn, agent, dashboard), strategic understanding (orchestrator, daemon), and owned complexity (completion, beads integration).
    
    **Key finding:** High investigation count + model existence = **friction that refused to resolve**. The absence of models for external dependencies (kb, tmux) despite high investigation counts reveals clear ownership boundaries.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- Phase 3 Review: Model Pattern Analysis (N=5)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE3_REVIEW.md
- Models
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/README.md
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
- Probe: Capture service state during click freeze recurrence (post-Session 15 nuclear elimination)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-13-service-state-freeze-recurrence.md
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
- Probe: Inventory all friction gates across spawn, completion, and daemon — assess defect-catching vs noise
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
  - Recent Probes:
    - 2026-02-13-friction-gate-inventory-all-subsystems
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
    - 2026-02-09-friction-bypass-analysis-post-targeted-skips
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md
- Orchestration Cost Economics
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestration-cost-economics.md
  - Summary:
    Agent orchestration cost is driven by three factors: **model pricing** (10-100x variance), **access restrictions** (fingerprinting, OAuth blocking), and **visibility** (lack of tracking caused $402 surprise spend). The Jan 2026 cost crisis revealed that headless spawning without cost visibility leads to runaway spend. DeepSeek V3 at $0.25/$0.38/MTok is now a **viable primary option** after testing confirmed stable function calling (contradicting earlier "unstable" documentation).
    
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
- Probe: GitHub Issues Search for Click Freeze and WindowServer Problems in yabai
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/macos-click-freeze/probes/2026-02-11-yabai-github-issues-search.md
  - Summary:
    **From first 100 issues:**
    - **10 issues** with titles mentioning "click", "mouse", "trackpad", or "freeze"
    - **1 issue** explicitly mentioning WindowServer crash (in body)
    - **0 issues** mentioning "killall -HUP WindowServer" as a workaround
    - **0 issues** mentioning "killall" in any context
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

### Guides (procedural knowledge)
- Model Selection Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/model-selection.md
- Understanding Artifact Lifecycle
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/understanding-artifact-lifecycle.md
- Headless Spawn Mode Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/headless.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Decision Authority Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/decision-authority.md
- Dual Spawn Mode Implementation Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dual-spawn-mode-implementation.md
- Orchestrator Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/orchestrator-session-management.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Two-Tier Sensing Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/two-tier-sensing-pattern.md

### Related Investigations
- Design Solution for Model Artifact Staleness
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md
- Spawn Context Model Inclusion
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-14-design-spawn-context-model-inclusion.md
- Model Handling Conflicts Between orch-go and opencode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md
- Build Model Advisor Tool Live
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-build-model-advisor-tool-live.md
- Model Selection Issue Architect Agent
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md
- User Describes Symptoms with Technical Terms
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-27-inv-user-describes-symptoms-technical-terms.md

### Failed Attempts (DO NOT repeat)
- Using empty string model resolution to let opencode decide

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.





📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `/exit` to close the agent session



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
2. **SET UP investigation file:** Run `kb create investigation create-model-users-dylanconlin-documents` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-create-model-users-dylanconlin-documents.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-create-model-users-14feb-1323/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (investigation)

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
name: investigation
skill-type: procedure
description: Record what you tested and observed; default to model-scoped probes when injected model claims are present, otherwise run a full investigation.
dependencies:
  - worker-base
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 6b38437badd8 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-02-13 23:15:22 -->

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
After your final commit, BEFORE doing anything else:

⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Running `git push` can trigger deploys that disrupt production systems
   - Worker rule: Commit your work, call `/exit`. Don't push.



1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.

TASK: List all available MCP tools. Report which ones start with mcp__playwright. If none, say FAIL: no playwright MCP tools found.


SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## CONFIG RESOLUTION

- Backend: opencode (source: cli-flag)
- Model: openai/gpt-5.2-codex (source: cli-flag)
- Tier: full (source: heuristic (skill-default))
- Spawn Mode: headless (source: default)
- MCP: playwright (source: cli-flag)
- Mode: tdd (source: default)
- Validation: tests (source: default)




## PRIOR KNOWLEDGE (from kb context)

**Query:** "list all available"

### Constraints (MUST respect)
- orch-go agent state exists in four layers (OpenCode memory, OpenCode disk, registry, tmux)
  - Reason: Each layer has independent lifecycle - cleanup must touch all layers or ghosts accumulate
- Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads
  - Reason: Investigation found all registry data can be derived from primary sources
- OpenCode x-opencode-directory header returns ALL disk sessions, not just matching ones
  - Reason: API behavior is counterintuitive - without header returns in-memory only
- orch status counts ALL workers-* tmux windows as active
  - Reason: Discovered during phantom agent investigation - status inflated by persistent windows
- Skillc embeds ALL template_sources unconditionally
  - Reason: No mechanism exists for conditional inclusion at compile time. Solution must work at spawn-time (runtime reference) not compile-time (conditional includes).
- Worker spawns must set ORCH_WORKER=1 to skip orchestrator skill loading
  - Reason: Orchestrator skill (1,251 lines ~37k tokens) is auto-loaded by session-context plugin for all orch projects but is unnecessary for worker sessions, wastes context budget
- Dashboard must be fully usable at 666px width (half MacBook Pro screen). No horizontal scrolling. All critical info visible without scrolling.
  - Reason: Primary workflow is orchestrator CLI + dashboard side-by-side on MacBook Pro. Minimum width constraint - should expand gracefully on larger displays.

### Prior Decisions
- Iteration 8 tmux fallback verification successful
  - Reason: After attach mode implementation (7ca8438), all three fallback mechanisms (tail, question, status) continue to work correctly with no regressions
- Confidence scores in investigations are noise, not signal
  - Reason: Nov 2025 Codex case: 5 investigations at High/Very High confidence all reached wrong conclusions. Current distribution: 90% cluster at 85-95%, providing no discriminative value. Recommendation: Replace with structured uncertainty (tested/untested enumeration).
- SPAWN_CONTEXT.md is 100% redundant - generated from beads + kb context + skill + template
  - Reason: Investigation confirmed all content exists elsewhere and can be regenerated at spawn time
- Progressive disclosure for skill bloat
  - Reason: 77% reduction (1757→400 lines) while preserving all guidance in reference docs
- beads output format uses ' - ' separator for title
  - Reason: All bd commands (list, ready) use this format: {beads-id} [{priority}] [{type}] {status} ... - {title}
- Multi-repo hydration requires healthy database
  - Reason: Orphaned dependencies in kb-cli blocked all database operations including multi-repo sync. Fix with 'bd doctor --fix' before attempting multi-repo setup.
- Headless spawn mode is production-ready
  - Reason: All 5 requirements verified working: status detection, monitoring, completion detection, error handling, user visibility. Investigation orch-go-0r2q confirmed no blockers exist.
- Use existing stores for UI improvements over backend changes
  - Reason: All necessary data (errorEvents, agentlogEvents) already exists in frontend stores; UI-only changes are faster to implement and deploy
- Default spawn mode is headless with --tmux opt-in
  - Reason: Aligns implementation with documentation (CLAUDE.md, orchestrator skill), reduces TUI overhead for automation, tmux still available via explicit flag
- Document all spawn modes in CLAUDE.md
  - Reason: Users need to see all options (headless default, --tmux opt-in, --inline blocking) to make informed spawn choices

### Models (synthesized understanding)
- Probe: Inventory all friction gates across spawn, completion, and daemon — assess defect-catching vs noise
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md
  - Recent Probes:
    - 2026-02-19-probe-glass-removal-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-glass-removal-verification.md
    - 2026-02-19-probe-accretion-enforcement-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-accretion-enforcement-gap-analysis.md
    - 2026-02-19-probe-coupling-hotspot-detection-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
- Phase 4 Review: Model Pattern at N=11
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/PHASE4_REVIEW.md
  - Summary:
    **At N=11, the model pattern shows exceptional consistency and proven utility.** All 11 models converged on the 6-section structure without enforcement. The enable/constrain query works across every domain tested. Most significantly: **the models that emerged reveal your cognitive investment priorities** - hot paths (spawn, agent, dashboard), strategic understanding (orchestrator, daemon), and owned complexity (completion, beads integration).
    
    **Key finding:** High investigation count + model existence = **friction that refused to resolve**. The absence of models for external dependencies (kb, tmux) despite high investigation counts reveals clear ownership boundaries.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
- OpenCode Fork
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork.md
  - Summary:
    Dylan owns a fork of [sst/opencode](https://github.com/sst/opencode) at `~/Documents/personal/opencode`. The fork is 16 custom commits ahead of upstream (linear forward — all upstream changes included). Custom changes focus on **memory management** (LRU/TTL instance eviction preventing 8.4GB unbounded growth), **SSE cleanup** (idempotent teardown preventing leaked connections), **OAuth stealth mode** (Claude Max access), **ORCH_WORKER header forwarding** (worker session detection), **session metadata** (extensible key-value storage), and **session TTL** (auto-expiry with periodic cleanup). The fork is a TypeScript monorepo (Bun runtime, Hono HTTP framework) with sessions stored in SQLite database (~/.local/share/opencode/opencode.db) using Drizzle ORM with migration-based schema management. Session status (idle/busy/retry) is tracked in-memory only, lost on server restart. A `GET /session/status` endpoint already exists for querying session state.
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-18-probe-verify-session-metadata-api
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-verify-session-metadata-api.md
    - 2026-02-18-probe-state-create-rejected-promise-cache
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-state-create-rejected-promise-cache.md
    - 2026-02-18-probe-repro-headless-codex-spawn
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-repro-headless-codex-spawn.md
    - 2026-02-18-probe-headless-codex-provider-init
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md
- Probe: Attention Pipeline Full Audit — What's Real vs Stub
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/dashboard-architecture/probes/2026-02-16-attention-pipeline-full-audit.md
  - Summary:
    **REAL and implemented (all 11 collectors):** Every collector has genuine detection logic reading real data (beads API, git log, agents API, JSONL files). None are stubs or return hardcoded data.
    
    **Currently firing:** Only 2 of 11 (BeadsCollector, RecentlyClosedCollector).
    
    **Not firing due to current state:** 6 of 11 (GitCollector, StuckCollector, UnblockedCollector, VerifyFailedCollector, EpicOrphanCollector, StaleIssueCollector). These would fire under the right conditions.
    
    **Cannot ever fire (badge types with no collector):** 3 badge types — `decide`, `escalate`, `crashed`. These exist in the frontend type system and badge config but have no backend collector that would ever emit them.
    
    **Net effect on dashboard:** 75% of open issues show false "Awaiting verification" badges due to `issue-ready` signals falling through to the default case. Zero legitimate badges are currently displayed.
    
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
- Session Deletion Vectors
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors.md
  - **STALENESS WARNING:**
    This model was last updated 2026-02-14.
    Changed files: cmd/orch/clean_cmd.go, pkg/daemon/daemon.go.
    Deleted files: opencode/src/session/index.ts, opencode/src/config/config.ts, opencode/src/project/instance.ts, opencode/src/session/session.sql.ts, pkg/cleanup/sessions.go, ~/bin/disk-cleanup.sh.
    Verify model claims about these files against current code.
  - Summary:
    Active OpenCode sessions can become unfindable through **7 independent vectors** spanning 3 systems (disk-cleanup.sh, orch-go cleanup, OpenCode itself). The fundamental problem is that no "session is active, do not touch" lock exists, and multiple processes can delete sessions from the shared SQLite database without coordination. The Ctrl+D keybind is triple-bound (app exit, session delete, input delete), creating the highest-risk accidental deletion path. The disk-cleanup.sh vector was the first confirmed root cause (now fixed), but the bug persists because at least two other vectors remain open.
    
    ---
  - Critical Invariants:
    1. **Sessions exist in SQLite or they don't** - There is no "evicted but recoverable" state
    2. **NotFoundError = row deleted from DB** - Not a caching issue
    3. **Multiple processes share one SQLite DB** - No coordination protocol
    4. **Cascade deletes propagate silently** - Deleting a session kills all messages and parts
    5. **No "active session" lock exists** - Any process can delete any session at any time
    6. **JSON→SQLite migration is one-time** - Gate checks `opencode.db` existence, not whether sessions were imported. DB existed from Jan 27 schema migrations → 188 JSON sessions permanently orphaned, invisible to current code
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Untracked Session Deletion (Vector #2)
    
    **Symptom:** Interactive TUI session disappears mid-conversation with NotFoundError
    
    **Root cause:** `cleanUntrackedDiskSessions()` in `clean_cmd.go:408-539` finds sessions not tracked by any `.orch/workspace/*/session_id` file, checks if they were updated in the last 5 minutes, and deletes them if idle.
    
    **Why TUI sessions are vulnerable:**
    - Interactive/orchestrator TUI sessions have NO workspace directory
    - No `.session_id` file means Layer 1 protection is bypassed entirely
    - If the user hasn't sent a message in >5 minutes (reading, thinking, context switch), Layer 2 (recency) marks it as orphaned
    - Layer 3 (`IsSessionProcessing()`) only runs for recently-active sessions
    - Session gets deleted via `client.DeleteSession(session.ID)` at line 525
    
    **Code path:**
    ```
    orch clean --sessions (or --all)
      → cleanUntrackedDiskSessions()
        → !trackedSessionIDs[session.ID]  ← TUI has no workspace, always true
        → now.Sub(updatedAt) > 5min       ← User paused, true
        → (skips IsSessionProcessing)     ← Only checked for recent sessions
        → client.DeleteSession(session.ID) ← SESSION DELETED
    ```
    
    **Fix needed:** Call `IsSessionProcessing()` for ALL untracked sessions, not just recently active ones. Cost: one API call per untracked session.
    
    ### Failure Mode 2: Accidental Ctrl+D Deletion (Vector #3)
    
    **Symptom:** Session vanishes after user presses Ctrl+D
    
    **Root cause:** Three keybinds share `ctrl+d`:
    - `app_exit: "ctrl+c,ctrl+d,<leader>q"` (config.ts:771)
    - `session_delete: "ctrl+d"` (config.ts:784)
    - `input_delete: "ctrl+d,delete,shift+delete"` (config.ts:878)
    
    **Why it happens:**
    1. User opens session list (`<leader>l`)
    2. User wants to exit the list, presses Ctrl+D (habit from terminal/vim)
    3. Session list dialog intercepts as `session_delete`, highlights session in red
    4. If user presses Ctrl+D again (common stutter or habit), session is permanently deleted
    5. TUI crashes with NotFoundError on next render cycle
    
    The confirmation ("Press ctrl+d again to confirm") is displayed as red-highlighted title text that may not be noticed in a fast interaction.
    
    **Fix needed:** Rebind `session_delete` to a non-conflicting key, or add a modal confirmation dialog.
    
    ### Failure Mode 3: External Process Deletion (Vector #4)
    
    **Symptom:** Session disappears without user action
    
    **Root cause:** `DELETE /session/:id` route has no authentication and no coordination. Any local process can delete
    ... [truncated]
  - Your findings should confirm, contradict, or extend the claims above.
  - Recent Probes:
    - 2026-02-14-probe-vector7-sqlite-migration-json-fallback
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md
    - 2026-02-14-probe-vector2-cleanuntrackedsessions-removal
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector2-cleanuntrackedsessions-removal.md
- System Learning Loop
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/system-learning-loop.md
  - Summary:
    The System Learning Loop is the third layer of the Pressure Visibility System that automatically converts recurring context gaps into actionable improvements. It tracks gaps during spawns, identifies patterns using RecurrenceThreshold=3, and suggests specific actions (kn entries, beads issues, investigations). The system uses shell-aware command parsing to generate runnable commands with proper quoted string handling, and ensures minimum length requirements for downstream tools (kn requires 20+ chars). This creates a closed feedback loop: gaps → patterns → suggestions → improvements → fewer gaps.
    
    ---
  - Critical Invariants:
    1. **RecurrenceThreshold = 3** - Pattern detection balances noise (1) vs signal (3+)
    2. **All matching events must be marked resolved** - Not just the most recent one
    3. **FindRecurringGaps excludes resolved events** - Prevents resolved gaps from reappearing
    4. **Shell-aware command parsing required** - Quoted strings with spaces must be preserved
    5. **Minimum 20-character reasons** - kn decide/constrain requirement enforced at generation time
    6. **30-day retention window** - Gap events older than 30 days are pruned
    7. **Gap recording happens after gating** - Captures all gaps whether spawn proceeds or not
    
    ---
  - Why This Fails:
    ### Failure Mode 1: Resolved Gaps Keep Appearing
    
    **Symptom:** `orch learn` shows same gap after running `orch learn act` to resolve it
    
    **Root cause:** Two bugs combined:
    1. `RecordResolution` only marked the most recent event (used `break` after first match)
    2. `FindRecurringGaps` counted all events without filtering by Resolution field
    
    **Why it happens:**
    - Gap occurs 5 times → user resolves → only 1 event marked
    - Next `FindRecurringGaps` call counts 4 unresolved events → still above threshold (3)
    - Same pattern keeps appearing despite resolution
    
    **Impact:**
    - User frustration ("I already fixed this!")
    - Loss of trust in learning system
    - Duplicate work
    
    **Fix:**
    - `RecordResolution` now marks ALL matching events (removed `break`)
    - `FindRecurringGaps` filters out resolved events before counting
    
    **Source:** `.kb/investigations/2025-12-25-inv-orch-learn-resolved-gaps-still.md`
    
    ---
    
    ### Failure Mode 2: Generated Commands Fail Due to Broken Quoting
    
    **Symptom:** `orch learn act N` generates command that fails when executed
    
    **Root cause:** Using `strings.Fields()` to parse commands - splits on whitespace without respecting quotes
    
    **Why it happens:**
    - Command: `kn decide "auth" --reason "Used by: investigation. Occurred 5 times"`
    - `strings.Fields` splits into: `["kn", "decide", "\"auth\"", "--reason", "\"Used", "by:", "investigation.", ...]`
    - Shell receives mangled arguments, command fails
    
    **Impact:**
    - Learning loop broken - suggestions can't be executed
    - User must manually reconstruct commands
    - Reduces value of automated suggestions
    
    **Fix:**
    - Added `ParseShellCommand()` with shell-aware quote handling
    - Respects double and single quotes as argument delimiters
    - Added `ValidateCommand()` to catch malformed commands before execution
    
    **Source:** `.kb/investigations/2025-12-26-inv-orch-learn-act-commands-should.md`
    
    ---
    
    ### Failure Mode 3: Short Reasons Fail kn Validation
    
    **Symptom:** `orch learn act` generates kn command that fails with "reason too short" error
    
    **Root cause:** `generateReasonFromGaps` produced "Occurred N times" (16 chars) when gap events lacked skill/task metadata
    
    **Why it happens:**
    - Gap events with sparse metadata → only occurrence count available
    - "Occurred 3 times" = 16 characters
    - kn requires 20+ characters for `--reason` flag
    - Command fails validation at execution time
    
    **Impact:**
    - Generated commands unusable
    - User must manually edit reason strings
    - Learning loop broken for sparse gaps
    
    **Fix:**
    -
    ... [truncated]
  - Your findings should confirm, contradict, or extend the claims above.
- Probe: Three Code Paths Verification State Divergence
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-16-probe-three-code-paths-verification-state.md
  - Recent Probes:
    - 2026-02-19-probe-glass-removal-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-glass-removal-verification.md
    - 2026-02-19-probe-accretion-enforcement-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-accretion-enforcement-gap-analysis.md
    - 2026-02-19-probe-coupling-hotspot-detection-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
- Probe: Orchestrator Skill CLI Staleness Audit
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md
  - Summary:
    | # | Stale Reference | Severity | Location (template line) |
    |---|----------------|----------|--------------------------|
    | 1 | `--opus` flag | HARMFUL | 231-234, 236 |
    | 2 | `orch frontier` | HARMFUL | 320, 621 |
    | 3 | `orch rework` | HARMFUL | 625 |
    | 4 | `orch reflect` | HARMFUL | 625 |
    | 5 | `orch kb archive-old` | HARMFUL | 625 |
    | 6 | `orch clean --stale` | HARMFUL | 649 |
    | 7 | `orch clean --untracked --stale` | HARMFUL | 326 |
    | 8 | Default = "sonnet + headless" | MISLEADING | 231-233 |
    | 9 | "Spawn modes: Default (headless)" | MISLEADING | 227, 544 |
    | 10 | `bd label <id>` (missing subcommand) | MISLEADING | 132, 578 |
    | 11 | Missing --bypass-triage in examples | MISLEADING | 233-234 |
    | 12 | bd comment deprecation | COSMETIC | worker-base (not this template) |
    | 13 | Reference file path | COSMETIC | 631 |
    
    **Total: 7 HARMFUL + 4 MISLEADING + 2 COSMETIC = 13 stale references**
    
    ---
  - Your findings should confirm, contradict, or extend the claims above.
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
- Probe: Verifiability-First Closure Audit — Did Claimed Work Actually Land?
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-15-verifiability-first-closure-audit.md
  - Recent Probes:
    - 2026-02-19-probe-glass-removal-verification
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-glass-removal-verification.md
    - 2026-02-19-probe-accretion-enforcement-gap-analysis
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-accretion-enforcement-gap-analysis.md
    - 2026-02-19-probe-coupling-hotspot-detection-gap
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md
    - 2026-02-18-probe-entropy-spiral-fix-commit-relevance
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-18-probe-entropy-spiral-fix-commit-relevance.md
    - 2026-02-17-rework-loop-design-for-verification-gaps
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md
- Probe: Headless Codex Provider Initialization Scoping
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md
  - Recent Probes:
    - 2026-02-18-probe-verify-session-metadata-api
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-verify-session-metadata-api.md
    - 2026-02-18-probe-state-create-rejected-promise-cache
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-state-create-rejected-promise-cache.md
    - 2026-02-18-probe-repro-headless-codex-spawn
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-repro-headless-codex-spawn.md
    - 2026-02-18-probe-headless-codex-provider-init
      See: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/opencode-fork/probes/2026-02-18-probe-headless-codex-provider-init.md

### Guides (procedural knowledge)
- Daemon Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/daemon.md
- Beads Integration
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md
- Spawned Orchestrator Pattern
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawned-orchestrator-pattern.md
- Worker Patterns Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/worker-patterns.md
- Dashboard
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dashboard.md
- Agent Lifecycle Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/agent-lifecycle.md
- API Development Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/api-development.md
- Development Environment Setup
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/dev-environment-setup.md
- Background Services Performance Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/background-services-performance.md
- Completion Workflow Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/guides/completion.md

### Related Investigations
- Audit All 19 Bd Exec.Command Call Sites for Concurrency Safety
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-25-inv-audit-all-19-bd-exec.md
- Daemon Uses Bd List Status
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-daemon-uses-bd-list-status.md
- Synthesize Dashboard Investigations (56 Listed, 62 Actual)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-dashboard-investigations-56-synthesis.md
- Add Comprehensive Orch Clean All
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-15-inv-add-comprehensive-orch-clean-all.md
- Synthesize Dashboard Investigations (55 Listed, 62 Total)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-synthesize-dashboard-investigations-55-synthesis.md
- Test Headless Spawn List Files
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/archived/2025-12-22-inv-test-headless-spawn-list-files.md
- Update All Worker Skills with 'Leave it Better' Phase
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-update-all-worker-skills-include.md
- Quick Test List Files Opencode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-quick-test-list-files-opencode.md
- Test Spawn List Files Current
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-17-inv-test-spawn-list-files-current.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference models and guides for established patterns. Reference investigations for prior findings.






📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — the workspace often has unrelated changes (.autorebuild.lock, .beads/, build/).
Stage ONLY the specific files you created or modified for your task, by name.


1. **CREATE SYNTHESIS.md** in your workspace with:
   - Plain-Language Summary (what was built/found)
   - Delta (what changed)
2. **COMMIT YOUR WORK:**
   ```bash
   git add <files you changed>
   git commit -m "feat: [brief description of changes]"
   ```
3. Run: `/exit` to close the agent session


⛔ **NEVER run `git push`** - Workers commit locally only.
   - Your orchestrator will handle pushing to remote after review
   - Worker rule: Commit your work, call `/exit`. Don't push.




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

2. [Task-specific deliverables]


3. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-work-list-available-mcp-20feb-37f6/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Track progress via beads comments. Call /exit to close agent session when done.




## SKILL GUIDANCE (hello)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: hello
skill-type: procedure
audience: worker
spawnable: true
category: test
description: Simple test skill that prints hello and exits
allowed-tools:
- Bash
deliverables: []
verification:
  requirements: |
    - [ ] Agent printed "Hello from orch-go!"
  test_command: null
  required: true
---

# Hello Skill

**Purpose:** Test the spawn system with a trivial task.

## Directive

When spawned with this skill, print the following message exactly:

```
Hello from orch-go!
```

Then exit immediately by running `/exit`.

## Notes

- This is a minimal test skill to verify spawn functionality.
- No other actions are required.
- The verification is simply that the message was printed.

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
Complete your session in this EXACT order:

⚠️ **NEVER use git add -A or git add .** — stage ONLY your task files by name.



1. Create SYNTHESIS.md in your workspace
2. **COMMIT YOUR WORK:** `git add <files you changed> && git commit -m "feat: [description]"`
3. `/exit`



⛔ **NEVER run `git push`** - Workers commit locally only.
⚠️ Your work is NOT complete until Phase: Complete is reported (or /exit for --no-track).
